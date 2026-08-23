# Задачи: Reporter — Data Source

- **Спека**: [spec.md](./spec.md) · **План**: [plan.md](./plan.md)
- **Дата**: 2026-08-24

Каждая задача трассируется к требованию и к файлам. Порядок — снизу вверх по слоям.

| # | Задача | Требование | Файлы | Статус |
|---|---|---|---|---|
| T1 | Domain: `DataSource`, протоколы/статусы, валидация | FR-1, FR-3 | `reporter/domain/data_source.go` | ☑ |
| T2 | Domain: типы интроспекции `Database/Table/Column` | FR-6..8 | `reporter/domain/introspection.go` | ☑ |
| T3 | Domain: доменные ошибки | FR-1,FR-2 | `reporter/domain/errors.go` | ☑ |
| T4 | Repository: `DataSourceModel` + AutoMigrate | FR-1,FR-4 | `reporter/repository/models.go` | ☑ |
| T5 | Repository: CRUD `data_source_repo.go` (Create/Get/GetActive/List/Update/SetStatus/Count) | FR-1,FR-3 | `reporter/repository/data_source_repo.go` | ☑ |
| T6 | Platform: ClickHouse connector по параметрам источника | FR-2,FR-5 | `pkg/clickhouse/connector.go` | ☑ |
| T7 | Platform: интроспекция (Ping/Databases/Tables/Columns) через `system.*` | FR-6..8 | `pkg/clickhouse/introspect.go` | ☑ |
| T8 | Config: `Reporter.SystemDBDenylist` | FR-6 | `config/config.go` | ☑ |
| T9 | Application: `Service` (create/test/activate/introspect) + шифр пароля | FR-1..8 | `reporter/application/service.go` | ☑ |
| T10 | Transport: admin-guard на `auth/contract.Provider` | FR-9 | `reporter/transport/http/guard.go` | ☑ |
| T11 | Transport: DTO (пароль только на вход) | FR-4 | `reporter/transport/http/dto.go` | ☑ |
| T12 | Transport: handlers + маппинг ошибок | FR-1..9 | `reporter/transport/http/handlers.go` | ☑ |
| T13 | Transport: router (заменить ping-заглушку) | FR-1..9 | `reporter/transport/http/router.go` | ☑ |
| T14 | Wiring: собрать Service, передать Provider в Register | все | `internal/app/app.go` | ☑ |
| T15 | Unit-тесты: валидация, маскирование пароля, denylist, единственность, маппинг ошибок | AC-1,4,7 | `reporter/application/service_test.go`, `reporter/domain/*_test.go` | ☑ |
| T16 | Проверка на стеке: create→test→databases/tables/columns на demo; пароль не в ответе/логах; 401/403 | AC-1..6 | — | ☑ |
| T17 | DoD: `go build/vet/gofmt/test`; обновить graphify; статус спеки → implemented | Принцип 7, DoD | — | ☑ |

## Definition of Done
- `go build ./...`, `go vet ./...`, `gofmt -l .` — чисто; `go test ./...` — зелёно (вкл. негативные).
- Реальный сценарий на запущенном стеке; `/readyz` зелёный; пароль не в ответах/логах.
- Каждое REP-FR трассировано к тесту; graphify-граф обновлён; спека переведена в `implemented`.
