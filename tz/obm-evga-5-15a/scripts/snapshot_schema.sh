#!/usr/bin/env bash
# Снапшот схемы боевой obm_evga (фаза 0). Только чтение: pg_dump --schema-only,
# SELECT-ы по каталогу, COPY TO для сэмплов. Ничего не пишет в источник.
#
# Использование:
#   export EVGA_PG_DSN='postgres://<ro_user>:<pass>@10.58.57.39:5432/obm_evga?sslmode=prefer'
#   ./snapshot_schema.sh
#
# Требуется Docker (psql/pg_dump берутся из образа postgres:18-alpine).
# Результат — в ../schema/. В git коммитится только структура (см. ../schema/.gitignore):
# сэмплы данных содержат ПДн (ИИН, ФИО, счета) — их не коммитить и не пересылать.
set -euo pipefail

: "${EVGA_PG_DSN:?задай EVGA_PG_DSN (read-only учётка obm_evga)}"

OUT="$(cd "$(dirname "$0")/../schema" 2>/dev/null && pwd || true)"
[ -z "$OUT" ] && { mkdir -p "$(dirname "$0")/../schema"; OUT="$(cd "$(dirname "$0")/../schema" && pwd)"; }
IMG=postgres:18-alpine
# Страховка: сессии psql принудительно read-only (pg_dump и так работает в RO-транзакции).
PSQL=(docker run --rm -i -e PGOPTIONS='-c default_transaction_read_only=on' "$IMG" psql "$EVGA_PG_DSN" -v ON_ERROR_STOP=1 -X -q)
PGDUMP=(docker run --rm -i "$IMG" pg_dump "$EVGA_PG_DSN")

echo "==> 1/5 Полный DDL схемы public -> schema.sql"
"${PGDUMP[@]}" --schema-only --no-owner --no-privileges -n public > "$OUT/schema.sql"

echo "==> 2/5 Размеры таблиц -> rowcounts.txt"
"${PSQL[@]}" -At <<'SQL' > "$OUT/rowcounts.txt"
select relname || E'\t' || n_live_tup || E'\t' || pg_size_pretty(pg_total_relation_size(relid))
from pg_stat_user_tables order by n_live_tup desc;
SQL

echo "==> 3/5 Карта внешних ключей -> fk_map.txt"
"${PSQL[@]}" -At <<'SQL' > "$OUT/fk_map.txt"
select con.conrelid::regclass || '.' || att.attname || ' -> ' ||
       con.confrelid::regclass || '.' || fatt.attname || '  [' || con.conname || ']'
from pg_constraint con
join unnest(con.conkey)  with ordinality k(attnum, ord) on true
join unnest(con.confkey) with ordinality fk(attnum, ord) on fk.ord = k.ord
join pg_attribute att  on att.attrelid  = con.conrelid  and att.attnum  = k.attnum
join pg_attribute fatt on fatt.attrelid = con.confrelid and fatt.attnum = fk.attnum
where con.contype = 'f'
order by 1;
SQL

echo "==> 4/5 Данные справочников -> refdata.sql (без больших таблиц)"
"${PGDUMP[@]}" --data-only --no-owner --inserts \
  -t public.its_risk_status -t public.its_regions -t public.its_departments \
  -t public.its_risk_profile > "$OUT/refdata.sql" || echo "  (часть справочников не снялась — см. вывод)"

echo "==> 5/5 Сэмплы рабочих таблиц (ПДн! не коммитить) -> sample_*.csv"
# files намеренно исключён: возможны большие blob-поля; его структура есть в schema.sql
for t in its_tb_5_15a its_risk_notice its_risk_notice_5_15a its_risk_notice_recepient its_cli its_out its_req its_activity users; do
  "${PSQL[@]}" -c "\\copy (select * from public.\"$t\" limit 300) to stdout with csv header" \
    > "$OUT/sample_$t.csv" 2>/dev/null \
    && echo "  $t: ok" || { echo "  $t: нет таблицы или нет прав"; rm -f "$OUT/sample_$t.csv"; }
done

echo "Готово: $OUT"
echo "Дальше: ./load_dev_replica.sh — залить схему и справочники в dev-Postgres."
