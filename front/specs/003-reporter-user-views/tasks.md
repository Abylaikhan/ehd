# Задачи: Reporter UI — Каталог и таблица

- **Спека**: [spec.md](./spec.md) · **План**: [plan.md](./plan.md) · **Дата**: 2026-08-24

| # | Задача | Требование | Файлы | Статус |
|---|---|---|---|---|
| T1 | API-типы Reporter (ViewMeta/ColumnMeta/QuerySpec/QueryResult/Count/ListItem) | FR-2,3 | `shared/api/types.ts` | ☑ |
| T2 | base: EmptyState, ErrorState (переиспользуемые) | FR-10 | `layers/base/components/{EmptyState,ErrorState}.vue` | ☑ |
| T3 | reporter: utils format (formatCell, filenameFromDisposition) + тесты | FR-8, AC-7 | `layers/reporter/utils/format.ts`, `format.test.ts` | ☑ |
| T4 | reporter: composable useReporterViews (list/meta/query/count/export) | FR-1..6 | `layers/reporter/composables/useReporterViews.ts` | ☑ |
| T5 | Каталог `/reporter` (список витрин, Empty/Error, auth) | FR-1,10,11 | `layers/reporter/pages/reporter/index.vue` | ☑ |
| T6 | Таблица `/reporter/[slug]` (meta→колонки, query+count, load-more, поиск, экспорт, состояния, URL-state) | FR-2..11 | `layers/reporter/pages/reporter/[slug].vue` | ☑ |
| T7 | Unit-тесты утилит зелёные | AC-7 | `layers/reporter/utils/format.test.ts` | ☑ |
| T8 | Проверка: pnpm build; визуально на стенде (analyst каталог→таблица→поиск→ещё→экспорт; 403/404/Empty) | AC-1..7 | — | ☑ |
| T9 | DoD: build/test; graphify update; спека implemented; залив в main | Принцип 7 | — | ☑ |

## Definition of Done
- `pnpm build` без ошибок; `pnpm test` (vitest) зелёно.
- Визуально на стенде: analyst видит каталог и таблицу `demo-transactions` (166 строк RLS), поиск/«Показать ещё»/экспорт работают; состояния Loading/Empty/Error/403/404 отображаются.
- Трассировка REP-FR-050..055; graphify обновлён; спека `implemented`; коммит в `main`.
