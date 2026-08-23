# Задачи: Reporter — Data View

- **Спека**: [spec.md](./spec.md) · **План**: [plan.md](./plan.md)
- **Дата**: 2026-08-24

| # | Задача | Требование | Файлы | Статус |
|---|---|---|---|---|
| T1 | Domain: расширить `DataView` (параметры), константы RowScope/SortDir | FR-5,FR-10 | `reporter/domain/data_view.go` | ☑ |
| T2 | Domain: `ViewColumn`, `ViewPermission`, `PublishedSnapshot` | FR-3,FR-6,FR-8 | `reporter/domain/view_column.go`, `view_permission.go`, `snapshot.go` | ☑ |
| T3 | Domain: `DisplayTypeFor` (таблица типов ТЗ, разворот Nullable) | FR-4 | `reporter/domain/display_type.go` | ☑ |
| T4 | Domain: новые ошибки представления | FR-1,FR-7 | `reporter/domain/errors.go` | ☑ |
| T5 | Repository: модели views/columns/permissions + AutoMigrate | FR-1..8 | `reporter/repository/models.go` | ☑ |
| T6 | Repository: CRUD + транзакция публикации + slug-uniqueness | FR-1..10 | `reporter/repository/data_view_repo.go` | ☑ |
| T7 | Application: `ViewService` (create/get/list/update/columns/permissions/publish/disable/delete) | FR-1..12 | `reporter/application/view_service.go` | ☑ |
| T8 | Transport: DTO представления/колонок/прав | FR-1..8 | `reporter/transport/http/view_dto.go` | ☑ |
| T9 | Transport: handlers + расширить mapErr | FR-1..12 | `reporter/transport/http/view_handlers.go`, `handlers.go` | ☑ |
| T10 | Transport: маршруты `/views` под admin-guard | FR-12 | `reporter/transport/http/router.go` | ☑ |
| T11 | Wiring: собрать ViewService, зарегистрировать | все | `internal/app/app.go` | ☑ |
| T12 | Unit-тесты: DisplayTypeFor, slug, publish-валидация, snapshot только visible, edit→draft | AC-3..7 | `reporter/domain/display_type_test.go`, `reporter/application/view_service_test.go` | ☑ |
| T13 | E2E на стенде: create→columns→permissions→publish→snapshot; disable; 401/403 | AC-1..9 | — | ☑ |
| T14 | DoD: build/vet/gofmt/test; graphify update; статус спеки → implemented; залив в main | Принцип 7, DoD | — | ☑ |

## Definition of Done
- `go build/vet/gofmt` чисто; `go test ./...` зелёно (вкл. негативные publish-тесты).
- E2E на стенде: публикация demo-представления, snapshot только с видимыми колонками; `/readyz` зелёный.
- Каждое REP-FR трассировано к тесту; graphify обновлён; спека `implemented`; коммит в `main`.
