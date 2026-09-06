#!/usr/bin/env bash
# Заливает снятый снапшот (schema.sql + refdata.sql + sample_*.csv) в dev-Postgres
# docker-compose-стека ehd2 как отдельную базу obm_evga — «песочница» для разработки.
# Боевую базу не трогает.
#
# Использование: ./load_dev_replica.sh   (стек должен быть поднят: make -C ../../.. … / docker compose up)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"   # корень ehd2
SCHEMA_DIR="$(cd "$(dirname "$0")/../schema" && pwd)"
COMPOSE=(docker compose -f "$ROOT/docker-compose.dev.yml")
DB_USER="${POSTGRES_USER:-ehd}"

[ -f "$SCHEMA_DIR/schema.sql" ] || { echo "Нет $SCHEMA_DIR/schema.sql — сначала ./snapshot_schema.sh"; exit 1; }

echo "==> Пересоздаю базу obm_evga в dev-Postgres"
"${COMPOSE[@]}" exec -T postgres psql -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 \
  -c "drop database if exists obm_evga;" -c "create database obm_evga;"

echo "==> Расширения и схема"
# дамп содержит CREATE SCHEMA public — убираем предсозданную схему
"${COMPOSE[@]}" exec -T postgres psql -U "$DB_USER" -d obm_evga -v ON_ERROR_STOP=1 \
  -c 'drop schema if exists public cascade;' >/dev/null
"${COMPOSE[@]}" exec -T postgres psql -U "$DB_USER" -d obm_evga -v ON_ERROR_STOP=1 \
  -c 'create schema if not exists public;' -c 'create extension if not exists "uuid-ossp" schema public;' >/dev/null
# CREATE SCHEMA public из дампа убираем — схема уже создана выше
sed '/^CREATE SCHEMA public;$/d' "$SCHEMA_DIR/schema.sql" | \
  "${COMPOSE[@]}" exec -T postgres psql -U "$DB_USER" -d obm_evga -v ON_ERROR_STOP=1 >/dev/null

if [ -f "$SCHEMA_DIR/refdata.sql" ]; then
  echo "==> Справочники"
  # replica-режим отключает FK: created_by и пр. ссылаются на неполные users
  { echo "set session_replication_role = replica;"; cat "$SCHEMA_DIR/refdata.sql"; } | \
    "${COMPOSE[@]}" exec -T postgres psql -U "$DB_USER" -d obm_evga -v ON_ERROR_STOP=1 >/dev/null
fi

shopt -s nullglob
for f in "$SCHEMA_DIR"/sample_*.csv; do
  t="$(basename "$f" .csv)"; t="${t#sample_}"
  echo "==> Сэмпл $t"
  # session_replication_role=replica отключает FK-триггеры: сэмплы неполны и ссылки могут висеть
  "${COMPOSE[@]}" exec -T postgres psql -U "$DB_USER" -d obm_evga -v ON_ERROR_STOP=1 \
    -c "set session_replication_role = replica;" \
    -c "\\copy public.\"$t\" from stdin with csv header" < "$f" \
    || echo "  $t: пропущен (структура/дубликаты)"
done

echo "Готово. DSN для локальной разработки:"
echo "  postgres://$DB_USER:<пароль ehd>@localhost:5433/obm_evga?sslmode=disable   # хост-порт см. docker-compose"
echo "  (изнутри compose: postgres:5432/obm_evga)"
