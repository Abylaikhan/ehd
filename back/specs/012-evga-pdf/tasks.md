# Задачи: 012-evga-pdf

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | Ассеты: эмблема+водяной знак из образца, шрифты DejaVu (embed) | FR-2 | `evga/pdf/assets/*` | done |
| T2 | Зависимость go-pdf/fpdf (ADR) | — | `go.mod` | done |
| T3 | Пакет pdf: Render (4 секции, шапка, водяной знак, таблица, подвал) | FR-2/4 | `evga/pdf/render.go` | done |
| T4 | Repo: сбор данных PDF (адресат, подписант+должность, исполнитель, строки) | FR-3 | `evga/repository/notice_repo.go` | done |
| T5 | Сервис+эндпоинт GET /notices/:id/pdf | FR-1 | `evga/application`, `transport/http` | done |
| T6 | Юниты: валидный PDF, форматы | FR-6 | `evga/pdf/render_test.go` | done |
| T7 | UI: кнопка «Скачать PDF» в карточке | FR-5 | `front/layers/evga` | done |
| T8 | Живой прогон: скачать PDF уведомления с реплики, визуальная сверка | приёмка | стек | done |
