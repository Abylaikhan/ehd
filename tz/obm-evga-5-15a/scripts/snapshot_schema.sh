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

echo "==> 5/5 СВЯЗНЫЙ срез уведомлений (FK-консистентно; ПДн! не коммитить) -> sample_*.csv"
# Якорь A — последние 300 действующих уведомлений. Один и тот же предикат во всех выборках,
# поэтому внешние ключи между таблицами совпадают (иначе список уведомлений пуст: req_id не
# находит its_req). Плюс 150 записей витрины в статусе 1 для теста формирования и общий срез
# its_cli/users для подсказок адресата/участников. files исключён (blob-поля).
A='(select * from its_risk_notice where "in$trash" is null order by id desc limit 300)'

sample() { # $1 имя таблицы(файла), $2 SELECT
  if "${PSQL[@]}" -c "\\copy ($2) to stdout with csv header" > "$OUT/sample_$1.csv" 2>/dev/null; then
    echo "  $1: $(( $(wc -l < "$OUT/sample_$1.csv") - 1 )) строк"
  else
    echo "  $1: ошибка/нет доступа"; rm -f "$OUT/sample_$1.csv"
  fi
}

sample its_risk_notice            "select * from $A n"
sample its_req                    "select * from its_req where id in (select req_id from $A a where req_id is not null)"
sample its_out                    "select * from its_out where id in (select its_out_id from $A a where its_out_id is not null)"
sample its_risk_notice_recepient  "select * from its_risk_notice_recepient where its_risk_notice_id in (select id from $A a)"
sample its_risk_notice_5_15a      "select * from its_risk_notice_5_15a where its_risk_notice_id in (select id from $A a)"
sample its_tb_5_15a               "select * from its_tb_5_15a where id in (select tb_5_15a_id from its_risk_notice_5_15a where its_risk_notice_id in (select id from $A a)) union select * from (select * from its_tb_5_15a where \"in\$trash\" is null and its_risk_status_id = 1 limit 150) s1"
sample its_cli                    "select * from its_cli where id in (select its_cli_id from its_risk_notice_recepient where its_risk_notice_id in (select id from $A a) union select sender_cli_id from its_out where id in (select its_out_id from $A a where its_out_id is not null)) union select * from (select * from its_cli where \"in\$trash\" is null limit 300) c1"
sample its_activity               "select * from its_activity where \"in\$trash\" is null limit 300"
sample users                      "select * from users where id in (select signer_id from $A a where signer_id is not null union select created_by from $A a where created_by is not null union select created_by from its_out where id in (select its_out_id from $A a where its_out_id is not null)) union select * from (select * from users where \"in\$trash\" is null limit 300) u1"

echo "Готово: $OUT"
echo "Дальше: ./load_dev_replica.sh — залить схему и справочники в dev-Postgres."
