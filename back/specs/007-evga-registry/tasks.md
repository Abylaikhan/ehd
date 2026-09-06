# Задачи: 007-evga-registry

Трассировка: FR-# из `spec.md` ← EVGA-FR/BR из `../tz/obm-evga-5-15a/`.

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | Конфиг `EVGA_PG_DSN` (необязательный), `Enabled()` | FR-1 | `config/config.go` | done |
| T2 | Compose: `EVGA_PG_DSN` на dev-реплику | FR-1 | `../docker-compose.dev.yml` | done |
| T3 | `UserIIN` в auth/contract + реализация + тест | FR-3 | `auth/contract/contract.go`, `auth/application/service.go`, `service_test.go` | done |
| T4 | Сид ролей `evga_auditor`/`evga_curator` | FR-4 | `auth/repository/seed_reference.go` | done |
| T5 | domain: RiskRecord/Filter/Sort-whitelist/Scope/Reference/ошибки | FR-5..7 | `evga/domain/*.go` | done |
| T6 | repository: GORM-модели obm_evga + RegistryRepo (List/Count/Get/References/UserByIIN, `in$trash`, LATERAL docnum) | FR-5/6/8 | `evga/repository/*.go` | done |
| T7 | application: резолв ИИН→департамент (кэш 15м), scope по ролям, список/карточка/справочники | FR-3/4/9/10 | `evga/application/service.go` | done |
| T8 | Excel-экспорт (excelize, лимит 50 000) | FR-11 | `evga/application/export.go` | done |
| T9 | transport: guard ролей, dto, handlers, router | FR-4..7 | `evga/transport/http/*.go` | done |
| T10 | Wiring в app.go: read-only DSN, пул, readyz, регистрация | FR-1/2/12 | `internal/app/app.go` | done |
| T11 | Unit-тесты: sort whitelist (негатив), scope по ролям, фильтры | П7 | `evga/**/*_test.go` | done |
| T12 | Прогон приёмки 1–9 на dev-реплике + `SHOW transaction_read_only` | все | стек compose | done (06.09.2026; сценарий «аудитор по ИИН» живьём — при подключении фронта, покрыт юнитами) |
