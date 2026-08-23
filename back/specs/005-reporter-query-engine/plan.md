# План реализации: Reporter — Query Engine

- **Спека**: [spec.md](./spec.md) · **Дата**: 2026-08-24

## Архитектура
`transport(http) → application(query_service) → querybuilder(pure) + Connector(exec)`.
Переиспользуем: snapshot из slice 004, `auth/contract.Identity` (роли + профиль регион/подразделение), `Connector`/`SourceConn` из slice 003 (расширим методом выполнения запроса).

## Слои и файлы
**domain**
- `query.go` — `QuerySpec`, `Filter`, канонические операторы, `AllowedOperators(displayType)`, `QueryResult`.
- расширить `data_view.go`/`snapshot.go` — поля `KeysetColumn`, `KeysetDir`, `RowScopeRegionColumn`, `RowScopeDepartmentColumn`.
- `rowscope.go` — `RowScope` (режим + профиль + маппинг колонок) и правила применимости.

**querybuilder** (`reporter/querybuilder/`) — ЧИСТЫЙ, максимально покрыт тестами
- `builder.go` — `Plan` (database, table, selectCols, filters, search, rowscope, keyset, limit) → `Build()` (SELECT) и `BuildCount()`; backtick-идентификаторы (regex-проверка), `{name:Type}` named-параметры, keyset-предикат, RLS-предикат.
- `operators.go` — маппинг оператора → SQL-фрагмент + тип параметра; валидация значения по display_type.
- `ident.go` — `safeIdent` (regex) + backtick-quote.

**repository**
- `models.go` — добавить 4 поля в `DataViewModel` (nullable строки).
- `data_view_repo.go` — `GetViewBySlug(slug)`; учесть новые поля в конвертере/Update.

**application**
- `query_service.go` — `QueryService`: резолв по slug (+snapshot), RBAC по ролям, проверка активного источника, валидация QuerySpec по snapshot (whitelist/операторы/типы), сборка `RowScope` из `Identity`, вызов builder, выполнение через `Connector`, формирование результата + next_cursor; `Count`; `PreviewDraft` (админ, без RLS, из текущей конфигурации).
- расширить `SourceConn`: `Query(ctx, sql, args...) ([]map[string]any, error)`, `ScalarUint64(...)`.
- `view_service.go` — при create задавать `KeysetColumn` из первичного ключа интроспекции; включать новые поля в snapshot при publish.

**chsource**
- реализовать `Query`/`ScalarUint64` с ClickHouse named-параметрами и динамическим сканированием строк; query settings (timeout/max rows/bytes/memory, query_id).

**transport/http**
- `query_dto.go`, `query_handlers.go` — user endpoints (`/reporter/views`, `/{slug}`, `/{slug}/query`, `/{slug}/count`) + admin preview; user-роуты под `requireAuth` (не только admin).
- `guard.go` — добавить `RequireAuth` (сессия без требования админа) для пользовательских маршрутов.
- `router.go` — смонтировать пользовательскую группу.

**wiring** — собрать `QueryService`, зарегистрировать пользовательские маршруты.

## Ключевые решения (безопасность — Принцип 3)
- **Named-параметры ClickHouse** для всех значений и для `database`/`table` (`{x:Identifier}`); колонки — backtick + regex-whitelist. Эталон — SQL из ТЗ (строка 624).
- **Keyset** по одному стабильному ключу (`keyset_column`, обычно первичный ключ): `key > {cursor}` (ASC) / `key < {cursor}` (DESC), `ORDER BY key`, `LIMIT pageSize+1` для определения следующей страницы; cursor = значение ключа последней строки.
- **RLS** применяется только при `by_profile` и `!IsAdmin`; предикаты по назначенным измерениям с `IS NOT NULL`; пустой профиль → без предиката (fail-open, security-тест).
- **total_count** — тот же WHERE (фильтры+поиск+RLS), без ORDER/LIMIT/keyset.
- Query settings на подключении: `max_execution_time`, `max_result_rows`, `max_memory_usage`.

## Тестирование (Принцип 7 — негативные тесты обязательны)
- Unit querybuilder: SELECT только whitelisted (нет `*`); значение → параметр (инъекция не ломает SQL); оператор вне таблицы → ошибка; keyset-предикат ASC/DESC; RLS-варианты (оба измерения / только регион / только подразделение / пусто → нет предиката); скрытая колонка не в SELECT; safeIdent отклоняет `a; DROP`.
- Unit application: RBAC (роль не из snapshot → 403); нормализация page_size; резолв slug (disabled → 404).
- E2E на стенде (demo-transactions): метаданные; страница+cursor; total_count; RLS для юзера-с-регионом vs админа; NULL-строка исключена; инъекционные попытки; предпросмотр.

## Риски
- Динамическое сканирование строк clickhouse-go: сперва `*any`-scan; если тип не поддержан — типизированный scan по `ColumnTypes().ScanType()`.
- Тип cursor зависит от типа ключа: в MVP ключ — целочисленный первичный (UInt64); иные типы — по source_type ключа, при неудаче помечаем в «вне объёма».
