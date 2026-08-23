# План реализации: Reporter — Export XLSX

- **Спека**: [spec.md](./spec.md) · **Дата**: 2026-08-24

## Мини-ADR: XLSX-библиотека
Добавляется `github.com/xuri/excelize/v2` — де-факто стандарт генерации XLSX в Go, поддерживает `StreamWriter` (низкая память). Это аддитивная зависимость для новой возможности (экспорт), а не замена фиксированного стека (Fiber/GORM/zap/viper/clickhouse-go). Одобрено запросом владельца «Export XLSX».

## Архитектура
Переиспользуем slice 005: `QueryService.resolve` (RBAC + активный источник), валидацию QuerySpec, `querybuilder.Plan`, `Connector` для выполнения. Отличия экспорта: SELECT только `exportable` колонок, LIMIT = `export_row_limit+1` (детект превышения), без cursor, потоковая запись в XLSX.

## Слои и файлы
**domain**
- `errors.go` — `ErrExportBusy`, `ErrExportTooLarge`.

**export** (`reporter/export/`)
- `xlsx.go` — `WriteXLSX(w io.Writer, sheet string, headers []string, rows [][]any) error` через excelize `StreamWriter`.

**application** (`reporter/application/query_service.go`)
- Рефактор: выделить `validateFilters`, `buildSearch`, `buildRowScope` из `buildPlan` (переиспользование в экспорте).
- `QueryService.exportSem chan struct{}` (cap 1) — process-глобальный guard одновременности; инициализация в `NewQueryService`.
- `Export(ctx, req, slug, spec) (ExportResult, error)`:
  1. tryAcquire semaphore (иначе `ErrExportBusy`); defer release.
  2. `resolve` (RBAC, источник, snapshot).
  3. exportable-колонки из snapshot; если пусто — `ErrQueryValidation`.
  4. валидация фильтров/поиска; RowScope из профиля (by_profile, не-админ).
  5. `Plan` (SelectCols=exportable, Keyset без cursor, Limit=export_row_limit+1).
  6. выполнить; если строк > лимит → `ErrExportTooLarge`.
  7. собрать `ExportResult{Filename, Headers(label), Rows([][]any простых значений)}` (NULL→пусто, деструктуризация указателей, Stringer/дата).

**transport/http** (`reporter/transport/http/`)
- `query_handlers.go` — `exportView`: `Export()` → set headers (`Content-Type`, `Content-Disposition`, `Cache-Control: no-store`) → `export.WriteXLSX` в тело ответа.
- `router.go` — `user.Post("/views/:slug/export", h.exportView)`.
- `handlers.go` — `mapErr`: `EXPORT_BUSY` (429), `EXPORT_TOO_LARGE` (413).

## Ключевые решения
- **Тот же валидатор/builder**, что и `/query` — безопасность наследуется (whitelist, параметры, RLS).
- **Лимит через `LIMIT n+1`**: если вернулось > n строк — отказ, без полного `COUNT(*)`.
- **Один экспорт**: неблокирующий `select` по каналу-семафору; занято → `EXPORT_BUSY`.
- Порядок — по keyset-ключу (детерминизм), cursor не нужен.
- Приведение значений ячеек: `nil`/nil-указатель → пусто; `time.Time` как есть; `fmt.Stringer` (decimal) → строка; иначе — значение.

## Тестирование
- Unit export-пакет: заголовки=label; корректный XLSX (открывается excelize-ом, число строк/колонок).
- Unit application: exportable-фильтрация колонок; превышение лимита → `ErrExportTooLarge`; занятость → `ErrExportBusy`; RLS в плане.
- E2E: экспорт analyst → XLSX с 166 строками (RLS ALA∩D01), заголовки RU-label, `department_code` отсутствует; заголовки ответа; 401/403; лимит.

## Риски
- Материализация до `export_row_limit` строк в памяти (компромисс): для ≤100k приемлемо; истинный построчный стрим из курсора — отдельная оптимизация.
- Типы значений из clickhouse-go (decimal/nullable-указатели) — приведение в helper'е; проверяется E2E на demo-данных.
