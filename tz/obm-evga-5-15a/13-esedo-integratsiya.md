# Интеграция с ЕСЭДО — конспект материалов (`tz/ЕСЭДО/`)

Получено 05.09.2026 вместе с ответами на вопросы. Относится к фазе 7 (исходящее, регистрация,
извещения) и частично к фазе 6 (ЭЦП).

## Материалы

| Файл | Что это |
|---|---|
| `требования_к_интеграции ЕСЭДО.ПД.02.10.М.docx` | Официальные требования НИТ (2017) к интеграции ведомственных СЭД с ЕСЭДО-Ц: архитектура (СПД/адаптер, ЕСЭДО-Ц, СЭД), сервис ЭДО, сервис НСИ, форматы сообщений, XSD бизнес-объектов |
| `doc_outgoing(1).txt` | Пример SOAP: отправка исходящего документа (`docOutgoing`) |
| `delivered(1).txt`, `registred(1).txt`, `execution(1).txt`, `finished(1).txt` | Примеры извещений: `stateDelivered`, `stateRegistered`, `stateExecution`, `stateFinished` |
| `stateNotValid.txt` | Пример извещения об отказе: `stateNotValid` c `isValidReason` («Отрицательный результат проверки ЭЦП»), свежий (02.2026) |
| `принципы применения ЭЦП_УЦ_25 12 2014_РУС (2).doc`, `Утв.Принципы применения/` | Принципы применения ЭЦП/УЦ (2013–2014) — регуляторный контекст подписи |

## Сервисы (ответы OQ-03/04)

- **MIND-S-2147** — Универсальный сервис ЕСЭДО для обмена электронными документами (основной канал).
- **MIND-S-0048** — Сервис подсистемы ЕХЭД для временных файлов: вложения передаются через ХЭД
  отдельными запросами (ограничение ШЭП ~15 МБ на сообщение), в `docOutgoing` идут только
  `fileIdentifier` загруженных файлов.
- **MIND-S-0035** — Сервис справочника ЕНСИ (организации) — источник синхронизации `its_cli`.
- Паспорта сервисов: `https://sb.egov.kz/services/passport/<код>`.

## Транспорт

- SOAP через ШЭП/ВШЭП, синхронный канал: метод `SendMessage`
  (`http://bip.bee.kz/SyncChannel/v10/Types`), `serviceId = ESEDO_UNIVERSAL_SERVICE`.
- `requestInfo`: `messageId` (uuid), `messageDate`, `routeId` (= `R_<код получателя>`),
  `sender{senderId,password}`. Доступ к ВШЭП — по VPN-туннелю; учётку выдаёт оператор.
- Полезная нагрузка `requestData/data` типизируется через `xsi:type`:
  - `ns1:docOutgoing` (`http://esedo.nitec.kz/service/model/document`) — исходящий документ;
  - `ns1:state*` (`http://esedo.nitec.kz/service/model/notification`) — извещения.

## `docOutgoing` — ключевые поля (по примеру и XSD)

`attachments[].fileIdentifier` (из ХЭД), `metadataSystem{from, performers, senderOrg, href}`,
`docNo` (напр. `510-02-05/19791`), `docDate`, `preparedDate`, `docLang`, `docKind`, `character`,
`authorNameRu/Kz`, `signerNameRu/Kz`, `executor`, `employeePhone`,
`secondSignEnabled=1` + `secondSignData` — **CMS-подпись ГОСТ (сертификат юрлица) с вложенными
OCSP/TSP** (ответ OQ-05: подпись серверная).

## Извещения (`state*`) — статусная модель канала (ответ OQ-01, типы)

| Тип | Смысл | Ключевые поля | Маппинг на наш процесс |
|---|---|---|---|
| DELIVERED (`stateDelivered`) | Доставлен получателю | `metadataSystem.href` (id документа), `date` | фиксация доставки в `its_out` |
| REGISTERED (`stateRegistered`) | Зарегистрирован / отказ в регистрации | **`regNo`** (напр. `1-010000-19-25814`), `date` | **триггер EVGA-FR-066/067**: номер+дата регистрации → `its_out`, транзакция статуса 4 |
| EXECUTION (`stateExecution`) | Передан на исполнение | `executive`, `execDate` | ход исполнения |
| FINISHED (`stateFinished`) | Исполнен | `author`, `realDate`, `resultCode/Text` | результат отработки |
| NEW_CONTROL / PROLONG_EXDATE / NEW_EXDATE / TAKE_OF_CONTROL | Контроль и сроки | — | сроки/контроль (уточнить применимость) |
| `stateNotValid` | Отказ (например, ЭЦП не прошла) | `isValidReason`, `secondSignNotifData` | обработка отказа отправки |

⚠ В примерах статусных XML кириллица в CP1251 (mojibake при чтении как UTF-8) — при разборе
реальных сообщений уточнить кодировку контура.

## Сообщения интеграции по документу НИТ (разделы 3–4)

`StartProcessRequest` (исходящее задание) / `ReceiveActivity` (входящее),
`ChangeActivityStateRequest` / `ActivityStateChanged` (смена состояния),
`StartProcessResponse` (квитанция ЕСЭДО-Ц), `DictionaryRequest/Response` (НСИ).
XSD бизнес-объектов — в Приложении 1 документа (типы `docOutgoing`, `stateNotValid`,
`stateDelivered`, `stateRegistered`, `stateStartProcessResponse` и др.).

## Ответы аналитика 07.09.2026 (`tz/Вопросы по ЕСЭДО #2 (1).docx`)

- **Приём извещений уже работает на стороне платформы ОБМ**: ЕСЭДО вызывает SOAP-эндпоинт
  `http://10.58.57.38/restapi/soap/esedo_in?wsdl` (настройка — `/static/#/settings/soapdetails/19`).
  ⚠ Следствие для фазы 7: извещения (`stateRegistered` с номером) будут прилетать в старую
  систему и обновлять `its_out`/`its_req` — наш модуль может реагировать на изменения в БД,
  а не поднимать собственный приёмник. Решить при проектировании фазы 7.
- **Сертификат юрлица** выдаёт ДЦГУ МФ РК (запрос письмом, получение через акт).
- **Сервер для подключения** к сервисам отобран, доступ прорабатывается.
- **Синхронизация `its_cli` с ЕНСИ не работает** — сервис ещё не сделан, справочник не обновляется.

## Следствия для плана

1. Фаза 7 проектируется вокруг **MIND-S-2147 + MIND-S-0048**: выгрузка PDF в ХЭД → `docOutgoing`
   с `fileIdentifier` → приём извещений `state*` → обработка `stateRegistered.regNo`.
2. Для отправки нужны: учётка ВШЭП (senderId/password), VPN-туннель, **серверный сертификат
   ГОСТ юрлица** (secondSign), коды организаций (`from`/`senderOrg`/`routeId`) — запросить
   вместе с доступом к БД.
3. Приём извещений — входящий веб-сервис на нашей стороне (или клиент ВШЭП): решить при
   проектировании фазы 7 по требованиям раздела 4 документа НИТ.
4. Проверка подписи — НУЦ РК (`pki.gov.kz`), в ehd2 уже есть NCANode-сайдкар (ГОСТ РК).
