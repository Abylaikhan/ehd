# Задачи: 014-evga-esedo-send (заготовка, не активирована)

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | Типы контракта ЕСЭДО (DocOutgoing/FileRef/Attachment/SendResult/RegisteredNotice) | FR-1 | `evga/esedo/types.go` | done |
| T2 | Интерфейсы Sender/Signer, Config, ошибки (ErrDisabled/ErrNotConfigured/ErrTransportDisabled) | FR-1/3 | `evga/esedo/sender.go` | done |
| T3 | StubSender/StubSigner — безопасные заглушки без сети (дефолт) | FR-2/5 | `evga/esedo/stub.go` | done |
| T4 | SoapSender — скелет: buildEnvelope, гейты Enabled/реквизиты, транспорт отключён | FR-3/4/5 | `evga/esedo/soap.go` | done |
| T5 | Юниты: заглушка без сети; buildEnvelope содержит ключевые узлы; гейты SoapSender | FR-7 | `evga/esedo/esedo_test.go` | done |
| T6 | Конфиг EVGA_ESEDO_* (ENABLED=false), место под реквизиты; в рантайме не читается | FR-6 | `config/config.go` | done |
| T7 | go build/vet/test ./... зелёные | приёмка | стек | done |
| T8 | Оркестрация `ESEDOOutgoingService.SendToESEDO` (PDF→upload→docOutgoing→подпись→send) на заглушке | вариант B | `evga/application/esedo_service.go` | done |
| T9 | Юниты оркестрации (спай-отправитель, ReadOnly-гейт) | — | `evga/application/esedo_service_test.go` | done |
| T10 | Эндпоинт `POST /notices/:id/esedo-send` + маппинг ошибок ЕСЭДО (503 ESEDO_NOT_ACTIVE) | вариант B | `evga/transport/http/{handlers,approval_handlers,router}.go` | done |
| T11 | Wiring в composition root: Stub/Soap по `EVGA_ESEDO_ENABLED`, StubSigner (TODO реальный ГОСТ) | — | `internal/app/app.go` | done |
| T12 | UI: кнопка «Отправить в ЕСЭДО» в карточке (при наличии исходящего) | вариант B | `front/layers/evga/{composables/useEvga.ts, pages/evga/notices/[id].vue}` | done |
| T13 | Живой смоук на реплике через заглушку (accepted, в сеть ничего не ушло) | приёмка | стек | done |

**Примечание:** пакет `esedo` теперь ПОДКЛЮЧЁН в рантайм (T11), но реальный транспорт остаётся
отключённым (StubSender по умолчанию; SoapSender → ErrTransportDisabled). В ЕСЭДО ничего не уходит.
