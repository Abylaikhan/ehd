# План реализации: Reporter — Data Source

- **Спека**: [spec.md](./spec.md)
- **Дата**: 2026-08-24

## Архитектура (Принцип 1)
`transport(http) → application(service) → repository(GORM) + clickhouse(introspect)`; `domain` — чистый.
Reporter получает личность пользователя только через `auth/contract.Provider` (без сети). Пароль источника шифруется через `pkg/crypto.Cipher` (переиспользуем существующий, AES-256-GCM).

## Слои и файлы
**domain** (`internal/modules/reporter/domain/`)
- `data_source.go` — сущность `DataSource`, константы `Protocol*`, `SourceStatus*`; чистые правила (валидация протокола/статуса). Без инфраструктуры.
- `introspection.go` — типы `Database`, `Table` (+ вид объекта), `Column`.
- `errors.go` — доменные ошибки: `ErrSourceAlreadyExists`, `ErrSourceNotFound`, `ErrConnectionFailed`, `ErrValidation`.

**repository** (`internal/modules/reporter/repository/`)
- `models.go` — добавить `DataSourceModel` (+ TableName `data_sources`), включить в `Migrate`.
- `data_source_repo.go` — CRUD: `Create`, `Get`, `GetActive`, `List`, `Update`, `SetStatus`, `Count`. Хранит `password_enc []byte`.

**clickhouse-connector** (`pkg/clickhouse/`)
- `connector.go` — `Connect(ConnParams) (driver.Conn, error)` — dial по параметрам источника (host/port/protocol/tls/user/pass), таймауты, read-only settings. Не завязан на env-конфиг.
- `introspect.go` — `Ping(ctx)` (SELECT 1), `Databases(ctx, deny)`, `Tables(ctx, db)`, `Columns(ctx, db, table)` через `system.databases/tables/columns`. Только параметризованные запросы.

**application** (`internal/modules/reporter/application/`)
- `service.go` — `Service` с методами `CreateSource`, `GetSource`, `ListSources`, `UpdateSource`, `TestSource(id)`, `TestParams(params)`, `Activate/Deactivate`, `Databases`, `Tables`, `Columns`. Шифрование/дешифрование пароля здесь; наружу пароль не отдаётся.
- зависимости: `SourceRepo`, `*crypto.Cipher`, connector-функция (инъекция для тестируемости), `Config{ SystemDBDenylist []string }`.

**transport/http** (`internal/modules/reporter/transport/http/`)
- `guard.go` — middleware `RequireAdmin` на базе `auth/contract.Provider` (cookie `ehd_session`/Bearer → `CurrentUser` → проверка `IsAdmin`). Локальный `identityKey` (модули не импортируют transport друг друга).
- `dto.go` — request/response DTO; пароль только на вход, в ответах отсутствует.
- `handlers.go` — обработчики source + introspection; маппинг доменных ошибок в `httpserver.NewError`.
- `router.go` — заменить `ping`-заглушку на реальную регистрацию под `/reporter` с admin-guard.

**wiring** (`internal/app/app.go`)
- Собрать reporter `Service` (repo + cipher + connector + denylist из конфига) и передать `authService` (как `contract.Provider`) в `reporterhttp.Register`.

**config** (`config/config.go`)
- Добавить `Reporter.SystemDBDenylist` (env `REPORTER_SYSTEM_DB_DENYLIST`, default `system,INFORMATION_SCHEMA,information_schema`).

## Ключевые решения
- **Переиспользуем `cfg.Auth.EncKey`-cipher** для шифра пароля источника (тот же AES-GCM). Отдельный ключ для Reporter — необязательное усиление, вне слайса.
- **Connector инъектируется** в сервис функцией `func(ConnParams)(Conn,error)` → в unit-тестах подменяется фейком, реальный ClickHouse не нужен.
- **Единственность источника** проверяется в сервисе (`Count()>0 → ErrSourceAlreadyExists`).
- Интроспекция — строго через `system.*`/DESCRIBE; идентификаторы БД/таблицы валидируются и передаются параметрами ClickHouse-запроса.

## Тестирование (Принцип 7)
- Unit (без внешнего CH): валидация протокола/полей; маскирование пароля в DTO; denylist системных баз; ошибки маппятся в коды; единственность источника (409).
- Integration (на запущенном стеке): create → test `SELECT 1` → databases/tables/columns на demo-таблице; проверка отсутствия пароля в ответе и логах.
- Негативные: тест с неверным паролем → `SOURCE_CONNECTION_FAILED`; не-админ → 403; без сессии → 401.

## Риски
- Реальный per-source dial отличается от boot-подключения `cfg.CH` — покрываем integration-тестом на sandbox ClickHouse (`clickhouse:9000`, demo-таблица).
- Формат `system.columns` (ключевые признаки) — берём доступные поля, отсутствующие помечаем как неизвестные, не падаем.
