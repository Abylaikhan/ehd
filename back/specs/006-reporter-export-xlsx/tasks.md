# Задачи: Reporter — Export XLSX

- **Спека**: [spec.md](./spec.md) · **План**: [plan.md](./plan.md) · **Дата**: 2026-08-24

| # | Задача | Требование | Файлы | Статус |
|---|---|---|---|---|
| T1 | Dep: excelize/v2 (мини-ADR в plan) | FR-8 | `go.mod` | ☑ |
| T2 | Domain: ErrExportBusy, ErrExportTooLarge | FR-6,FR-7 | `reporter/domain/errors.go` | ☑ |
| T3 | export-пакет: WriteXLSX через StreamWriter | FR-1,FR-8 | `reporter/export/xlsx.go` | ☑ |
| T4 | Рефактор: выделить validateFilters/buildSearch/buildRowScope | FR-4,FR-5 | `reporter/application/query_service.go` | ☑ |
| T5 | QueryService.Export (guard, exportable, лимит, матрица значений) | FR-2..7,FR-10 | `reporter/application/query_service.go` | ☑ |
| T6 | Transport: exportView handler + заголовки + маршрут + mapErr | FR-1,FR-9 | `reporter/transport/http/{query_handlers,router,handlers}.go` | ☑ |
| T7 | Unit: export headers=label/валидный XLSX; лимит→too_large; busy; exportable-only | AC-2,6,7 | `reporter/export/xlsx_test.go`, `reporter/application/export_test.go` | ☑ |
| T8 | E2E: analyst экспорт (166 строк, RU-заголовки, RLS, без department_code); заголовки ответа; 401/403 | AC-1..5 | — | ☑ |
| T9 | DoD: build/vet/gofmt/test; graphify; спека implemented; залив в main | Принцип 7 | — | ☑ |

## Definition of Done
- `go build/vet/gofmt` чисто; `go test ./...` зелёно (лимит/занятость/exportable-only).
- E2E: реальный XLSX analyst с RLS открывается; заголовки ответа корректны; `/readyz` зелёный.
- Трассировка REP-BR-008/009; graphify обновлён; спека `implemented`; коммит в `main`.
