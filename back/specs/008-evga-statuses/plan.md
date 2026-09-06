# План реализации: 008-evga-statuses

- **Спека**: `spec.md` · Дата: 2026-09-06
- **Конституция**: Принципы 1, 3 (типизированные параметры, без SELECT *), 4 (RBAC на сервере),
  7 (негативные тесты — гейт). Отступление П5 сохраняется: миграции ТОЛЬКО ручными SQL,
  AutoMigrate к obm_evga не применяется.

## Архитектура

- `domain/status_flow.go` — матрица переходов и валидация условий (чистая логика без БД):
  `ValidateTransition(current, target, attrs) error` + типизированные ошибки причин.
- `repository/status_repo.go` — в транзакции: `SELECT … FOR UPDATE` записей,
  UPDATE полей отработки, INSERT журнала; `History(recordID)`; `UserIDByIIN`.
- `application/status_service.go` — RBAC (аудитор своего департамента/админ; куратор — отказ),
  режим записи (`EVGA_WRITE_ENABLED`), резолв `changed_by`, одиночный и bulk сценарии,
  отчёт «обработано/отклонено».
- `transport/http` — `POST /registry/:id/status`, `POST /registry/status/bulk`,
  `GET /registry/:id/history`; коды `EVGA_READ_ONLY`, `TRANSITION_NOT_ALLOWED`,
  `TRANSITION_CONDITION_FAILED`.
- Конфиг: `EVGA{DSN, WriteEnabled}`; app.go: read-only параметр DSN только при `!WriteEnabled`;
  compose: `EVGA_WRITE_ENABLED=true` (dev-реплика).
- Миграции: `back/migrations/evga/00{1..5}_*.sql` + README (порядок, прод-предупреждение).

## Изменяемые файлы

`config/config.go`, `../docker-compose.dev.yml`, `internal/app/app.go`,
`internal/modules/evga/domain/{status_flow.go,registry.go,errors.go}`,
`internal/modules/evga/repository/status_repo.go`,
`internal/modules/evga/application/{status_service.go,service.go}`,
`internal/modules/evga/transport/http/{handlers.go,router.go,dto.go}`,
`back/migrations/evga/*.sql`, тесты `*_test.go`.

## Тесты

- Юнит domain: полный перебор матрицы (все пары 8×8), все условия входа, статус 4 вручную.
- Юнит application: RBAC (куратор/чужой департамент/админ), режим чтения, отчёт bulk.
- Живой прогон на dev-реплике: AT-03…06 + журнал + history.

## Риски

- Условия матрицы — трактовка «условия входа» (Clarifications) — подтвердить заказчиком.
- 004/005 (уникальные индексы) на проде могут не создаться при существующих дублях —
  скрипты сначала диагностируют дубли SELECT-ом (комментарий в файле).
