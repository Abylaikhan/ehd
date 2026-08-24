# План: Reporter UI — Админ-конструктор витрин

- **Спека**: [spec.md](./spec.md) · **Дата**: 2026-08-24

## Файлы
**shared/api/types.ts** — admin-типы:
- `DataSourceSummary` (id, code, name, status, ...); `IntrospectDatabase{name}`, `IntrospectTable{name,engine,kind}`, `IntrospectColumn{name,type,position,nullable,...}`.
- `ViewSummary` (list) и `ViewDetail` (+ `columns: ViewColumnDetail[]`, `role_codes: string[]`, keyset/row-scope поля).
- payload'ы: `CreateViewPayload`, `UpdateViewMetaPayload`, `ColumnConfigPayload`, `columns:[]`, `role_codes:[]`.

**layers/reporter/composables/useReporterAdmin.ts** — admin-клиент:
- sources: `list()`, `databases(id)`, `tables(id,db)`, `columns(id,db,table)`.
- views: `list()`, `get(id)`, `create(payload)`, `updateMeta(id,payload)`, `updateColumns(id,cols)`, `setPermissions(id,codes)`, `preview(id,spec)`, `publish(id)`, `disable(id)`, `remove(id)`.
- roles: `roles()` → `/auth/admin/roles`.

**layers/reporter/utils/viewForm.ts (+ test)** — `columnsToPayload(rows)` (маппинг reactive-строк редактора → ColumnConfigPayload[]); `statusMeta(status)` (label+severity).

**pages/reporter/admin/views/index.vue** — список (DataTable: id-хвост, подключение/БД, таблица, статус-Tag, дата; действия: открыть/опубликовать/отключить/удалить; «Создать» → диалог каскада; middleware auth+admin).

**pages/reporter/admin/views/create.vue** — каскад Источник→База→Таблица (последовательные Select с загрузкой интроспекции) + Название/Slug/Описание → `create()` → `navigateTo(/reporter/admin/views/{id})`.

**pages/reporter/admin/views/[id].vue** — карточка-конструктор:
- секция «Источник» (read-only), статус-Tag.
- «Колонки»: DataTable с editable-ячейками (Checkbox visible/searchable/filterable/sortable/exportable; InputText label; Select display_type; InputNumber position; source_name/source_type read-only) → локальный reactive массив → «Сохранить колонки» `updateColumns`.
- «Права»: MultiSelect ролей → «Сохранить» `setPermissions`.
- «Ограничение строк и параметры»: Select row_scope_mode, Select region/department колонок (из columns), Select keyset, InputNumber page sizes/export limit → «Сохранить параметры» `updateMeta`.
- «Предпросмотр»: кнопка → `preview` → мини-DataTable.
- Действия: «Опубликовать»/«Отключить»/«Удалить» с `Message`-обратной связью.

## Ключевые решения
- **Каскад создания**: три `useAsyncData`/`ref` с `watch` — при выборе источника грузим базы; при выборе базы — таблицы. (Источник по умолчанию — единственный активный.)
- **Редактор колонок**: серверные колонки → локальный reactive-снимок (`rows`); правки не уходят на сервер до «Сохранить» (draft, Принцип 3). После сохранения — `refresh`.
- **display_type** options — фиксированный список (text/number/money/percent/date/datetime/boolean/enum/json/uuid).
- **row-scope колонки** выбираются из физических имён колонок витрины (Select).
- Ошибки действий → `apiErrorMessage` в `Message`/inline; публикация PUBLISH_VALIDATION → понятный текст.
- Все страницы `definePageMeta({ middleware: ['auth','admin'] })`.

## Тестирование
- Unit (vitest): `columnsToPayload` (маппинг флагов/типов), `statusMeta`.
- `pnpm build` (SSR) без ошибок.
- Визуально на стенде (admin): список → создать витрину на demo_transactions → настроить колонки/права/row-scope → preview → publish; проверить отключение/удаление и Empty/Error.

## Риски
- Много inline-редактируемых ячеек в DataTable — используем `#body` со связкой v-model к строке reactive-массива (стабильные ключи по source_name).
- Каскадные Select с async-загрузкой — аккуратный сброс нижних уровней при смене верхнего.
