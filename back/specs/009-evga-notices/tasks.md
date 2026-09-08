# Задачи: 009-evga-notices

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | domain: Notice/NoticeGroup/Rejected/статусы заявки/шаблон текста §9.2 | FR-1/2/11 | `evga/domain/notice.go` | done |
| T2 | repository: PreviewSelect (группировка+причины), SearchCli | FR-1..3 | `evga/repository/notice_repo.go` | done |
| T3 | repository: CreateNotices (транзакция FOR UPDATE, req+notice+снимок+адресат) | FR-4..6 | там же | done |
| T4 | repository: ListNotices/GetNotice/DeleteNotice | FR-7..9 | там же | done |
| T5 | application: NoticeService (RBAC, write-mode, created_by по ИИН) | FR-10 | `evga/application/notice_service.go` | done |
| T6 | юниты: группировка/причины отклонений/RBAC/read-only | FR-1/2/10 | `*_test.go` | done |
| T7 | transport: 6 эндпоинтов + коды ошибок | FR-1..9 | `evga/transport/http/*` | done |
| T8 | wiring app.go | — | `internal/app/app.go` | done |
| T9 | Живой прогон приёмки 1–8 на dev-реплике | все | стек | done |
