# Задачи: 013-evga-deadlines

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | Домен: `DeadlineState(execDue string, statusID *int64, warnDays int, now)` + константа порога; поля `ExecDue`/`DeadlineState` в `RiskRecord`; флаг `ConfirmedNoDecision` в `Filter` | FR-1/2/3 | `evga/domain/registry.go` | done |
| T2 | Юниты доменной функции (границы, пустой срок, чужой статус) | FR-2 | `evga/domain/registry_deadline_test.go` | done |
| T3 | Repo: `exec_due` в `noticeLateral`+`recordColumns`, скан в `rowScan`; условие `confirmed_no_decision` в `base` | FR-1/3/5 | `evga/repository/registry_repo.go` | done |
| T4 | Конфиг `EVGA_DEADLINE_WARN_DAYS` (env, дефолт 3) + проброс в сервис | FR-4 | `config/config.go`, `internal/app/app.go` | done |
| T5 | Сервис: заполнение `DeadlineState` по `warnDays`+now; парс `confirmed_no_decision` | FR-1/2/3 | `evga/application/service.go` | done |
| T6 | Transport: query-параметр `confirmed_no_decision`, поля `exec_due`/`deadline_state` в DTO реестра/карточки | FR-1/3 | `evga/transport/http/{dto,handlers}.go` | done |
| T7 | `go build/vet/test ./...`; живой прогон на реплике (запись со сроком, фильтр 11) | приёмка | стек | done |
