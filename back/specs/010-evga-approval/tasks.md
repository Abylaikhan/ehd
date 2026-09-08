# Задачи: 010-evga-approval

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | domain: номер (формат/сброс года), статусы маршрута, ошибки | FR-8..10 | `evga/domain/approval.go` (+тест) | done |
| T2 | repository (ehd-БД): модель+AutoMigrate route_step, CRUD маршрута | §3 | `evga/repository/route_repo.go` | done |
| T3 | repository (obm_evga): выдача номера FOR UPDATE, смена stat_id, signer, поиск users департамента | FR-3/5/8/9 | `evga/repository/approval_repo.go` | done |
| T4 | application: ApprovalService (RBAC, submit/approve/reject, валидации) | FR-2..7 | `evga/application/approval_service.go` (+тест) | done |
| T5 | transport: 5 эндпоинтов + коды ошибок | FR-1..6 | `evga/transport/http/approval_handlers.go`, router | done |
| T6 | wiring app.go (ehd db + evga db), AutoMigrate route_step | — | `internal/app/app.go` | done |
| T7 | Живой прогон приёмки 1–9 на реплике (номер/год/повтор/approve/reject) | все | стек | done |
