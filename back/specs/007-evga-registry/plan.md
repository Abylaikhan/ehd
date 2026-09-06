# План реализации: 007-evga-registry

- **Спека**: `spec.md` (007-evga-registry)
- **Дата**: 2026-09-06
- **Соответствие конституции**: Принципы 1 (слои, межмодульно только contract), 2 (стек без новых
  библиотек), 3 (whitelist сортировок/фильтров, типизированные параметры, запрет SELECT * —
  явные списки колонок), 4 (RBAC/видимость на сервере), 6 (request_id, readyz), 7 (тесты-гейт).
  Отступление от П5: AutoMigrate НЕ применяется к внешней БД `obm_evga` (чужая прод-БД,
  только чтение) — зафиксировано в spec FR-2.

## Архитектура

Новый модуль `internal/modules/evga` по образцу `reporter`:

```
internal/modules/evga/
  domain/        registry.go (RiskRecord, Filter, Page, DepartmentScope, Reference), errors.go
  repository/    models.go (GORM-модели obm_evga, column-теги с "in$trash"/"sys$uuid")
                 registry_repo.go (List/Count/Get/References/UserByIIN)
  application/   service.go (резолв ИИН→департамент + кэш, RBAC-scope, список/карточка/справочники/экспорт)
  transport/http/ guard.go (роли evga_auditor|evga_curator|admin через auth/contract)
                 dto.go, handlers.go, router.go
```

Подключение: отдельный `*gorm.DB` через существующий `pkg/postgres.New`; к DSN добавляется
runtime-параметр pgx `default_transaction_read_only=on` (сессионный read-only). Пул: 5/2.

Ключевые решения:
- **ИИН**: `auth/contract.Provider` расширяется методом `UserIIN(ctx, userID) (iin string,
  verified bool, err error)` — расшифровка `IINEnc` внутри auth (cipher не покидает модуль).
  ИИН передаётся в evga только в памяти, не логируется.
- **№ уведомления**: LEFT JOIN LATERAL — последний действующий снимок
  `its_risk_notice_5_15a` по `tb_5_15a_id` → `its_risk_notice` → `its_req.docnum`.
- **№ исходящего**: `t.its_out_id → its_out.doc_num`.
- **Роли**: сид `evga_auditor` («Аудитор ДВГА»), `evga_curator` («Куратор КВГА») —
  идемпотентно в `authrepo.SeedReference` (FirstOrCreate, как SeedAdmin).
- **Excel**: excelize (уже в go.mod), StreamWriter, лимит 50 000, буфер в память
  (как reporter exportView).
- **readyz**: ping внешней БД только при включённом модуле.

## Изменяемые файлы

| Файл | Изменение |
|---|---|
| `config/config.go` | + `EVGA{DSN string}`; DSN необязателен, `Enabled()` |
| `../docker-compose.dev.yml` | + `EVGA_PG_DSN` на dev-реплику `postgres:5432/obm_evga` |
| `internal/modules/auth/contract/contract.go` | + `UserIIN` в `Provider` |
| `internal/modules/auth/application/service.go` | + реализация `UserIIN` |
| `internal/modules/auth/application/service_test.go` | + тест `UserIIN` (verified/невериф.) |
| `internal/modules/auth/repository/seed_reference.go` | + сид ролей evga |
| `internal/modules/evga/**` | новый модуль (8 файлов + тесты) |
| `internal/app/app.go` | wiring evga при `cfg.EVGA.Enabled()`; readyz |

## Тесты (гейт)

- Unit domain/application: whitelist сортировок (негативный — FR-7/INVALID_FILTER),
  расчёт scope по ролям (аудитор/куратор/админ/unmapped), маппинг фильтров.
- Unit repository: сборка условий фильтра (SQL-фрагменты через gorm DryRun; негативный —
  неизвестная сортировка отклонена до SQL).
- Интеграционный прогон вручную на dev-реплике: сценарии приёмки 1–9 спеки (curl),
  `SHOW transaction_read_only`.

## Риски

- Разница коллаций/регистронезависимый поиск `sendername` — ILIKE (индексов нет, на проде
  замерить; при деградации — trigram-индекс через миграцию фазы 2 по согласованию).
- `paymentdate` в витрине — тип date (по DDL) — фильтры от/до включительно.
- Кэш ИИН→департамент в памяти процесса: при горизонтальном масштабировании допустимо
  (TTL 15 мин, данные квазистатичные).
