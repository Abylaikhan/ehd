# Задачи: 008-evga-statuses

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | Миграции SQL 001…005 + README (dev-реплика сейчас, прод после согласования) | FR-13 | `migrations/evga/` | done |
| T2 | Применить 001–003 (+004/005 по возможности) к dev-реплике | FR-13 | psql | done |
| T3 | Конфиг `EVGA_WRITE_ENABLED`, DSN-режим, compose | FR-1/2 | `config/config.go`, `app.go`, compose | done |
| T4 | domain: матрица переходов + условия входа + ошибки | FR-3..6 | `domain/status_flow.go` | done |
| T5 | Юниты матрицы: полный перебор 8×8 + условия | FR-3..6 | `domain/status_flow_test.go` | done |
| T6 | repository: транзакция FOR UPDATE + UPDATE + журнал; History; UserIDByIIN | FR-9..11 | `repository/status_repo.go` | done |
| T7 | application: RBAC, режим записи, одиночный/bulk, отчёт | FR-7/8 | `application/status_service.go` | done |
| T8 | Юниты application: RBAC, read-only, отчёт bulk | FR-7/8 | `application/status_service_test.go` | done |
| T9 | transport: 3 эндпоинта + коды ошибок | FR-4/5/7/8/12 | `transport/http/*` | done |
| T10 | Живой прогон AT-03…06 + history на dev-реплике | все | стек | done |
