# Спецификация: Reporter — Data View (представление таблицы)

- **ID**: 004-reporter-data-view
- **Статус**: implemented
- **Требования ТЗ**: REP-FR-040, REP-FR-041, REP-FR-042, REP-FR-043, REP-FR-044; REP-BR-001, REP-BR-007; разделы «Создание Data View», «Настройка колонок», «Соответствие типов и фильтров», «Предпросмотр и публикация»
- **Дата**: 2026-08-24

## 1. Цель и ценность
Второй кирпич вертикали Reporter: администратор из настроенного источника (slice 003) собирает **опубликованное представление** одной таблицы — выбирает базу/таблицу, настраивает колонки (видимость, подпись, тип отображения, порядок, флаги поиска/фильтра/сортировки/экспорта), задаёт права ролей и параметры поведения, затем **публикует**. Публикация фиксирует неизменяемый snapshot, который позже читает Query Engine и конечный пользователь. Представление живёт по статусной модели ТЗ.

## 2. Пользовательские сценарии
- Администратор создаёт черновик: название, slug, источник → база → таблица; backend автоматически загружает колонки из интроспекции (все `visible=false`).
- Администратор настраивает колонки: подпись, `display_type` (авто по типу ClickHouse), порядок, `visible`, `searchable/filterable/sortable/exportable`.
- Администратор задаёт параметры: пагинация (50 по умолчанию, 20–200), сортировка по умолчанию, лимит экспорта (≤100 000), режим row scope.
- Администратор назначает роли, которым доступно представление.
- Администратор **публикует**: backend проверяет источник, наличие ≥1 видимой колонки, валидный уникальный slug и ≥1 роль → статус `published`, фиксируется snapshot и `schema_hash`.
- Редактирование опубликованного представления переводит его в `draft`, но **пользователи продолжают работать с текущим snapshot** до повторной публикации.
- Администратор может отключить представление (`disabled`) без удаления.

## 3. Функциональные требования
| ID | Требование (поведение) | Источник ТЗ |
|---|---|---|
| FR-1 | Создание черновика: name, уникальный slug (`^[a-z0-9-]+$`), описание, ссылка на **активный** источник, база (не системная), таблица/VIEW/MATVIEW | «Создание Data View»; REP-BR-001 |
| FR-2 | При создании/пересинхронизации колонки берутся из интроспекции; `source_name`/`source_type` не редактируются; `visible=false` по умолчанию | «Настройка колонок» |
| FR-3 | Настройка колонки: label, display_type, position, visible, searchable, filterable, sortable, exportable (+ хранение format/mask_rule/width/null_label) | «Настройка колонок» |
| FR-4 | `display_type` авто-выводится из `source_type` по таблице соответствия ТЗ; `Nullable(T)` разворачивается в T | «Соответствие типов» |
| FR-5 | Параметры представления: page_size (default 50, min 20, max 200), сортировка по умолчанию, export_row_limit ≤ 100 000, row_scope_mode (`by_profile`\|`unrestricted`) | «Создание Data View» |
| FR-6 | Права: набор кодов ролей; администратор имеет доступ всегда | «Создание Data View», Права |
| FR-7 | Публикация разрешена только при: активный источник, ≥1 видимая колонка, валидный уникальный slug, ≥1 роль (REP-FR-041) | REP-FR-041 |
| FR-8 | Публикация фиксирует snapshot (db/table + whitelist колонок + параметры + права + schema_hash) и заменяет предыдущий; история версий не ведётся | REP-FR-042 |
| FR-9 | Редактирование колонок/параметров/источника переводит в `draft`; текущий published snapshot остаётся доступным до повторной публикации | REP-FR-043 |
| FR-10 | Отключение (`disabled`) без удаления; статусная модель draft/published/disabled/schema_error/archived | REP-FR-044 |
| FR-11 | Скрытая колонка (`visible=false`) не попадает в snapshot-whitelist (основа для REP-BR-007 в Query Engine) | REP-BR-007 |
| FR-12 | Все `/reporter/admin/*` — под сессией администратора (RBAC на backend) | REP-FR-002 |

## 4. Контракты API (раздел 10)
Все — под сессией администратора; ответы содержат `request_id`; ошибки — единый контракт.
- `POST   /reporter/admin/views` — создать черновик (+ авто-загрузка колонок).
- `GET    /reporter/admin/views` — список представлений.
- `GET    /reporter/admin/views/{id}` — карточка: мета, параметры, колонки, права, published-инфо.
- `PATCH  /reporter/admin/views/{id}` — мета и параметры (name, slug, description, пагинация, сортировка, export limit, row_scope_mode).
- `DELETE /reporter/admin/views/{id}` — удалить черновик.
- `PUT    /reporter/admin/views/{id}/columns` — массовое обновление конфигурации колонок.
- `POST   /reporter/admin/views/{id}/columns/refresh` — пересинхронизировать колонки из интроспекции.
- `PUT    /reporter/admin/views/{id}/permissions` — задать коды ролей.
- `POST   /reporter/admin/views/{id}/publish` — валидация и публикация.
- `POST   /reporter/admin/views/{id}/disable` — отключить.

## 5. Сущности данных
- `data_views`: id, name, slug (uniq), description, data_source_id, database_name, table_name, source_mode (`physical_object`), status, revision, schema_hash, page_size_default/min/max, default_sort_column, default_sort_dir, export_row_limit, row_scope_mode, published_snapshot (jsonb), published_at, created_at, updated_at.
- `view_columns`: id, view_id, source_name, source_type, label, display_type, position, visible, searchable, filterable, sortable, exportable, format (jsonb), mask_rule, width, null_label.
- `view_permissions`: id, view_id, role_code (уникально в паре view_id+role_code).

## 6. Нефункциональные ограничения
- Публикация атомарна (транзакция): snapshot и статус меняются вместе.
- `schema_hash` = SHA-256 по упорядоченному набору видимых колонок (`source_name:source_type`).
- slug валидируется и хранится в нижнем регистре; уникальность на уровне БД и проверки.
- ClickHouse не читается при управлении представлением, кроме интроспекции при создании/refresh (read-only).
- Metadata API p95 ≤ 500 мс.

## 7. Критерии приёмки
- **AC-1**: create с активным источником и существующей таблицей → 201, статус `draft`, колонки подгружены (`visible=false`).
- **AC-2**: create с несуществующей таблицей/системной базой/неактивным источником → 4xx с понятным кодом.
- **AC-3**: slug не по маске или дубль → `VALIDATION_ERROR`/409.
- **AC-4**: `display_type` авто-корректен: `LowCardinality(String)`→text, `Decimal(18,2)`→number/money, `DateTime`→datetime, `Nullable(Int64)`→number.
- **AC-5**: publish без видимых колонок / без ролей / неактивный источник → 422 с указанием причины; НЕ публикуется.
- **AC-6**: publish валидного черновика → `published`, `published_at` задан, `schema_hash` непустой, snapshot содержит только видимые колонки (скрытые отсутствуют).
- **AC-7**: PATCH/columns опубликованного представления → статус `draft`, но `published_snapshot` сохранён.
- **AC-8**: disable → `disabled`.
- **AC-9**: не-администратор → 403; без сессии → 401.

## 8. Вне объёма
- **Query Engine** и предпросмотр строк (REP-FR-040) — следующий слайс (snapshot уже готовится здесь).
- Пользовательский маршрут `/reporter/{slug}` и рендер таблицы (REP-FR-050+).
- Автоопределение дрейфа схемы и авто-перевод в `schema_error` (проверка при выполнении запроса — в Query Engine).
- Экспорт XLSX; managed_query режим (Phase 2); валидация существования ролей в auth (коды принимаются как есть — из admin-справочника ролей).
- Семантика форматирования (format/mask_rule применяются позже при отдаче данных).
