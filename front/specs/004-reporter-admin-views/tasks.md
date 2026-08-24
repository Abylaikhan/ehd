# Задачи: Reporter UI — Админ-конструктор витрин

- **Спека**: [spec.md](./spec.md) · **План**: [plan.md](./plan.md) · **Дата**: 2026-08-24

| # | Задача | Требование | Файлы | Статус |
|---|---|---|---|---|
| T1 | Admin API-типы (DataSourceSummary, Introspect*, ViewSummary, ViewDetail, payloads) | FR-2,3,4 | `shared/api/types.ts` | ☑ |
| T2 | Composable useReporterAdmin (sources/introspect/views/roles) | FR-1..8 | `layers/reporter/composables/useReporterAdmin.ts` | ☑ |
| T3 | utils viewForm (columnsToPayload, statusMeta) + тесты | FR-4, AC-6 | `layers/reporter/utils/viewForm.ts`, `viewForm.test.ts` | ☑ |
| T4 | Список `/reporter/admin/views` (DataTable, действия, Создать) | FR-1,8,10 | `layers/reporter/pages/reporter/admin/views/index.vue` | ☑ |
| T5 | Создание `/reporter/admin/views/create` (каскад источник→база→таблица) | FR-2,10 | `layers/reporter/pages/reporter/admin/views/create.vue` | ☑ |
| T6 | Карточка `/reporter/admin/views/[id]` (колонки/права/row-scope/preview/publish) | FR-3..9,11 | `layers/reporter/pages/reporter/admin/views/[id].vue` | ☑ |
| T7 | Пункт меню «Витрины (админ)» в AppShell (для админа) | FR-1 | `layers/base/components/AppShell.vue` | ☑ |
| T8 | Unit-тесты утилит зелёные | AC-6 | `layers/reporter/utils/viewForm.test.ts` | ☑ |
| T9 | Проверка: pnpm build; визуально (admin: создать→настроить→preview→publish; отключить/удалить) | AC-1..6 | — | ☑ |
| T10 | DoD: build/test; graphify; спека implemented; залив в main | Принцип 7 | — | ☑ |

## Definition of Done
- `pnpm build` без ошибок; `pnpm test` зелёно.
- Визуально: admin создаёт витрину на `demo_transactions`, настраивает колонки/права/row-scope, делает preview и публикует; список отражает статусы.
- Трассировка REP-FR-020..044; graphify обновлён; спека `implemented`; коммит в `main`.
