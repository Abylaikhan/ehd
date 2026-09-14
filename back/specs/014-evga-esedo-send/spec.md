# Спецификация: ОБМ ЕВГА — заготовка отправки исходящего в ЕСЭДО

- **ID**: 014-evga-esedo-send
- **Статус**: draft (заготовка; **не активирована**)
- **Требования ТЗ**: EVGA-FR-065/066, §11.3; интеграция — `../tz/obm-evga-5-15a/13-esedo-integratsiya.md`
- **Дата**: 2026-09-09

## 1. Назначение и жёсткие ограничения

Подготовить **каркас** отправки исходящего уведомления в ЕСЭДО (MIND-S-2147 `SendMessage`,
вложения через ХЭД MIND-S-0048, серверная подпись ГОСТ `secondSignData`), чтобы при получении
доступов ШЭП и сертификата оставалось только вписать реквизиты и включить транспорт.

**Ограничения (действуют строго):**
- **Ничего не отправляется в ЕСЭДО.** Реальный сетевой вызов в скелете `SoapSender` **намеренно
  отключён** (`ErrTransportDisabled`), плюс жёсткий рубильник `Config.Enabled` (по умолчанию false).
- Заготовка **не подключена в composition root** (`internal/app`) — рантайм-поведение модуля не
  меняется: регистрацию по-прежнему инициирует появление `doc_num` в `its_out` (спека 011, вотчер).
- Активация — отдельной задачей после доступов (Бауыржан) и подтверждений (В4 аналитику).

## 2. Открытый вопрос архитектуры (зафиксировать до активации)

Кто фактически шлёт `docOutgoing` в ЕСЭДО:
- **(A)** платформа ОБМ (по ответу аналитика №3-7 «платформа обработает извещения и для наших
  исходящих», приёмник `http://10.58.57.38/restapi/soap/esedo_in`) — тогда наш `Sender` **не нужен**,
  мы только реагируем на `its_out` (текущая схема);
- **(B)** наш модуль (по ТЗ §11 и `13-esedo-integratsiya.md`) — тогда активируем `SoapSender`.

Заготовка покрывает вариант (B); вопрос вынесен в `voprosy-analitiku-4.md` (В4-1 смежно). До ответа
пакет остаётся неактивным.

## 3. Состав заготовки (пакет `internal/modules/evga/esedo`)

| Элемент | Роль |
|---|---|
| `Sender` (интерфейс) | `UploadAttachment` (файл→ХЭД `fileIdentifier`) + `SendOutgoing` (`docOutgoing`→квитанция) |
| `Signer` (интерфейс) | `SignCMS` — CMS-подпись ГОСТ юрлица для `secondSignData` (OQ-05) |
| `DocOutgoing`, `FileRef`, `Attachment`, `SendResult`, `RegisteredNotice` | Типы контракта по `13-esedo-integratsiya.md` |
| `Config` | Реквизиты ШЭП: `Enabled`(false), `Endpoint`, `SenderID`, `Password`, `ServiceID`, `FromOrg`, `CertPath` |
| `StubSender` / `StubSigner` | Безопасные заглушки (по умолчанию): без сети, логируют намерение, возвращают детерминированный результат |
| `SoapSender` | Скелет реального клиента: собирает SOAP-конверт `SendMessage`+`docOutgoing`; **транспорт отключён** |

## 4. Функциональные требования

| ID | Требование |
|---|---|
| FR-1 | `Sender`/`Signer` — интерфейсы; типы `DocOutgoing`/`FileRef`/`Attachment`/`SendResult` отражают поля §13. |
| FR-2 | `StubSender` не выполняет сетевых вызовов; `SendOutgoing` возвращает `Accepted=true` с пометкой «не отправлено», логирует состав. Дефолтная реализация. |
| FR-3 | `SoapSender.SendOutgoing`: при `!Config.Enabled` → `ErrDisabled`; при пустых реквизитах → `ErrNotConfigured`; иначе собирает конверт и возвращает `ErrTransportDisabled` (реальный `http.Do` закомментирован с пометкой TODO(ШЭП)). Ни при каком значении конфигурации заготовка не шлёт запрос. |
| FR-4 | `SoapSender.buildEnvelope` формирует валидный XML SOAP `SendMessage` с `requestInfo`(serviceId/messageId/messageDate/routeId/sender) и `requestData/data` c `xsi:type=ns1:docOutgoing`; форма приближённая — выверить по WSDL контура. |
| FR-5 | `SignAndAttach` демонстрирует шов подписи: ставит `SecondSignEnabled` + `SecondSignData` через `Signer`. |
| FR-6 | Конфиг `EVGA_ESEDO_*` (env, `ENABLED=false` по умолчанию) — место для реквизитов. В рантайме не читается до активации. |
| FR-7 | Юниты: заглушка возвращает результат без сети; `buildEnvelope` содержит ключевые узлы (SendMessage, serviceId, docNo, xsi:type); `SoapSender` с `Enabled=false` → `ErrDisabled`, с `Enabled=true` без сети → доходит до `ErrTransportDisabled` (не паникует, не шлёт). |

## 5. Что нужно для активации (чек-лист)

1. Учётка ВШЭП (`SenderID`/`Password`), `Endpoint`, VPN-туннель — Бауыржан.
2. Серверный сертификат ГОСТ юрлица + реализация `Signer.SignCMS` (через NCANode-сайдкар).
3. Коды организаций: `FromOrg`, `routeId=R_<код получателя>`, `performers` — от заказчика.
4. Выверка XSD `docOutgoing`/квитанции по WSDL контура; замена `ErrTransportDisabled` на реальный вызов.
5. Ответ аналитика по варианту (A)/(B) из §2.
6. Снятие ограничения «в ЕСЭДО не отправлять» и приёмочный прогон на тестовом контуре.

## 5а. Уточнения по ответам аналитика (10.09.2026)

Конкретика, полученная после написания заготовки (см. `../tz/obm-evga-5-15a/voprosy-analitiku-4.md`
Часть 6 и образцы `../tz/eds_temp_files download *.txt`):

- **Сервисы:** `EDS_TEMP_FILES` (временные файлы, MIND-S-0048), `ENSI_SeGetDataGetItems`
  (справочники ЕНСИ), `ESEDO_UNIVERSAL_SERVICE` (основной обмен, MIND-S-2147).
- **Постоянного хранилища нет.** Вложения/файлы идут только через `EDS_TEMP_FILES`:
  `SendMessage`(serviceId=`EDS_TEMP_FILES`) → `TempStorageRequest` (`type` = `UPLOAD`/`DOWNLOAD`).
  UPLOAD: `uploadRequest/fileUploadRequests{fileProcessIdentifier, name, content(base64), mime,
  lifeTime, needToBeConfirmed}` + `credentials{senderId, password}`; ответ — `fileIdentifier`,
  который затем кладётся в `docOutgoing/attachments`. `lifeTime` в образце — 28800000 мс (8 ч) →
  временное; **постоянные копии (PDF, вложения к статусам) хранить у себя в ЕХД** (открытый П-1).
- **Подпись — двухуровневая:** (1) документ подписывает исполнитель; (2) SOAP-конверт подписывается
  **WSSE ds:Signature** сертификатом госслужащего на сервере — алгоритм
  `gostr34102015-gostr34112015-512`, ключ в `wsse:SecurityTokenReference/KeyIdentifier` (X509).
  → В заготовке `Signer` уточняется как «подпись конверта WSSE» (не только CMS `secondSignData`);
  реализация — `signWSSE(gost_path, gost_password, xml, …)` при активации.
- **`docOutgoing` (ESEDO_UNIVERSAL_SERVICE):** получен точный шаблон + пример значений
  (voprosy-analitiku-4.md). `buildEnvelope` воспроизводит его **1:1** (имена/порядок полей,
  `routeId=R_ESEDO`, `secondSignEnabled=1`, `metadataSystem{activityId,from,href,performers,
  senderOrg}`, `documentReceiverKz/Ru`, `docKind`/`character` — коды НСИ). `DocOutgoing` расширен
  под полный набор полей. Маппинг подтверждён юнит-тестом `TestBuildEnvelope`.
- **Наблюдение по 5.0:** присланный шаблон — с `{{lua "signWSSE" …}}` и контекстом `.param`/`.input`,
  то есть это **шаблон низкокодового движка платформы**. Это склоняет к **варианту A (отправляет
  платформа)**. Одновременно аналитик дал реквизиты `EDS_TEMP_FILES` и предложил самим поднять
  VPN к ВШЭП — то есть вариант B тоже открыт. **РЕШЕНО 11.09.2026 (ответ пользователя + скрины платформы):
вариант B — отправляем МЫ.** Модуль формирует уведомление и исходящее и сам шлёт в ЕСЭДО по
нажатию; PDF кладём в `its_out` программно (upload в EDS_TEMP_FILES → fileIdentifier → вложение);
маршрут — один (наш). Заготовку АКТИВИРУЕМ после доступов (П-3). Далее нужна оркестрация:
`OutgoingService.SendToESEDO` (PDF→upload→docOutgoing→SendMessage) + кнопка «Отправить в ЕСЭДО».
  Финальные коды организаций (`from`/`senderOrg`/`performers`/`docKind`/`character`) и учётка
  `ESEDO_UNIVERSAL_SERVICE` — ещё ждём.
- **Ограничение в силе:** до явного разрешения пользователя (П-3) реальные обращения к
  ЕСЭДО/ШЭП/`EDS_TEMP_FILES` не выполняем; транспорт остаётся отключённым.

## 6. Вне объёма

Реальная отправка; приём/разбор извещений `state*` (сейчас на стороне платформы, №3-7);
загрузка PDF в ХЭД «вживую»; оркестрация «создать→подписать→отправить» в `OutgoingService`
(добавится при активации отдельной задачей, чтобы не менять протестированный поток спеки 011).
