# Задачи: 011-evga-outgoing

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | domain: AddBusinessDays + тесты | §3 | `evga/domain/businessdays.go` (+test) | done |
| T2 | StatusRepo: каскад как явная опция + системная простановка для регистрации | FR-3 | `evga/repository/status_repo.go` | done |
| T3 | OutgoingRepo: создание its_out, чтение регистрации, транзакция Register | FR-1..4 | `evga/repository/outgoing_repo.go` | done |
| T4 | application: OutgoingService (создание, Register, симуляция) | FR-1..6 | `evga/application/outgoing_service.go` (+тест) | done |
| T5 | transport: POST /outgoing, POST /register (admin) | FR-1/6 | `evga/transport/http/*` | done |
| T6 | Вотчер в app.go + конфиг интервала | FR-5 | `internal/app/app.go`, `config/config.go`, compose | done |
| T7 | UI: кнопка «Создать исходящее» в маршруте, given исходящего в карточке | FR-7 | `front/layers/evga/*` | done |
| T8 | Живой прогон приёмки 1–5 на реплике | все | стек | done |
