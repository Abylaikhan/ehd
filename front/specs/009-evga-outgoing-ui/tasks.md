# Задачи: 009-evga-outgoing-ui (проход 1)

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | Логика шага `currentNoticeStep` + метки | FR-1 | `layers/evga/utils/noticeSteps.ts` | done |
| T2 | Юнит `currentNoticeStep` (все ветки) | FR-7 | `layers/evga/utils/noticeSteps.test.ts` | done |
| T3 | Компонент степпера (Создан→Подписание→Канцелярия→Выполнен, цвета) | FR-1/2 | `layers/evga/components/EvgaNoticeSteps.vue` | done |
| T4 | Разбивка карточки на вкладки (PrimeVue Tabs) + блок отправителя/вложения | FR-3/4/5/6 | `layers/evga/pages/evga/notices/[id].vue` | done |
| T5 | vitest зелёный; SSR 200 | приёмка | стек | done |
| T6 | Визуальная проверка 1024/1280/1440 (табы, степпер, состояния) | приёмка | браузер | ⏳ вручную |

**Проход 2 (позже):** блок «Регистрационные реквизиты» (коды НСИ docKind/character) и вкладка
«Информация о доставке» (извещения state*, П-4) — блокированы данными от заказчика.
