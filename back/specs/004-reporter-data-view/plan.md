# План реализации: Reporter — Data View

- **Спека**: [spec.md](./spec.md)
- **Дата**: 2026-08-24

## Архитектура
`transport(http) → application(service) → repository(GORM) + application.Connector(интроспекция)`; `domain` — чистый. Переиспользуем slice 003: `SourceRepo` (проверка активного источника) и `Connector`/`SourceConn` (интроспекция колонок при создании/refresh).

## Слои и файлы
**domain** (`reporter/domain/`)
- `data_view.go` — расширить: добавить поля параметров (пагинация, сортировка, export limit, row_scope_mode) и константы `RowScope*`, `DisplayType*`, `SortDir*`. Уже есть статусы и source mode.
- `view_column.go` — сущность `ViewColumn` + `MaskRule*`.
- `view_permission.go` — сущность `ViewPermission` (view_id, role_code).
- `snapshot.go` — `PublishedSnapshot` (db/table + видимые колонки + параметры + права + schema_hash) — то, что читает Query Engine.
- `display_type.go` — чистый маппер `DisplayTypeFor(chType) string` (таблица «Соответствие типов»), разворот `Nullable(T)`.
- `errors.go` — добавить `ErrViewNotFound`, `ErrSlugTaken`, `ErrPublishValidation`, `ErrTableNotFound`, `ErrSourceInactive`.

**repository** (`reporter/repository/`)
- `models.go` — добавить `DataViewModel`, `ViewColumnModel`, `ViewPermissionModel` в AutoMigrate; `published_snapshot` — jsonb (тип `datatypes.JSON` или `[]byte`).
- `data_view_repo.go` — CRUD представления, колонок, прав; транзакция публикации (snapshot + статус); slug-uniqueness.

**application** (`reporter/application/`)
- `view_service.go` — `ViewService`: Create (валидация источника/таблицы + авто-загрузка колонок через Connector), Get/List, UpdateMeta, UpdateColumns, RefreshColumns, SetPermissions, Publish (валидация REP-FR-041 + сборка snapshot + schema_hash в транзакции), Disable, Delete. Editing published → status=draft.
- зависимости: `ViewRepo`, `SourceRepo` (из 003), `Connector` (из 003).

**transport/http** (`reporter/transport/http/`)
- `view_dto.go` — DTO представления/колонок/прав/публикации.
- `view_handlers.go` — обработчики; маппинг доменных ошибок (расширить `mapErr`).
- `router.go` — добавить группу `/views` под тем же admin-guard.

**wiring** (`internal/app/app.go`)
- Собрать `ViewService` (тот же db, SourceRepo, Connector) и `views`-handlers; зарегистрировать.

## Ключевые решения
- **Snapshot как jsonb** на `data_views.published_snapshot`: публикация фиксирует неизменяемую конфигурацию для Query Engine; редактирование меняет мутабельные `view_columns`/параметры и переводит в `draft`, snapshot остаётся (REP-FR-043).
- **schema_hash** = SHA-256 от `strings.Join(sorted("name:type" видимых колонок), "|")`.
- **display_type** — чистая функция в domain (легко тестируется по таблице ТЗ).
- **Публикация — транзакция**: пересчёт snapshot + schema_hash + смена статуса атомарно.
- **Роли** принимаются как коды-строки (валидация существования — вне слайса; источник кодов — auth admin roles).
- Автозагрузка колонок при create использует уже готовый `Connector.Columns` (slice 003).

## Тестирование
- Unit (без внешнего CH): `DisplayTypeFor` по всем строкам таблицы ТЗ (+Nullable); slug-валидация; publish-валидация (нет видимых колонок / нет ролей / неактивный источник → отказ); snapshot содержит только видимые колонки; edit published → draft.
- Integration (стенд): create на demo-таблице (колонки подгружены) → настроить видимость → задать роль → publish → snapshot/schema_hash; disable; 401/403.
- Негативные: несуществующая таблица, дубль slug, публикация без прав/видимых колонок.

## Риски
- Тип jsonb в GORM: используем `datatypes.JSON` (gorm.io/datatypes) или `[]byte` c ручным маршалингом. Проверить наличие зависимости; при отсутствии — `[]byte` + `json.Marshal`.
- Каскадное удаление колонок/прав при удалении черновика — удаляем явно в транзакции (FK отключены в AutoMigrate).
