# Задачи: 007-evga-notices-ui

| # | Задача | Файлы | Статус |
|---|---|---|---|
| T1 | Типы EvgaNotice* | `shared/api/types.ts` | done |
| T2 | API-методы notices/cli в composable | `layers/evga/composables/useEvga.ts` | done |
| T3 | Логика canCreate + vitest | `layers/evga/utils/noticeForm.ts`, `noticeForm.test.ts` | done |
| T4 | Кнопка «Сформировать уведомление» в реестре + передача ids | `layers/evga/pages/evga/index.vue` | done |
| T5 | Экран группировки `/evga/notices/create` | `layers/evga/pages/evga/notices/create.vue` | done |
| T6 | Список `/evga/notices` | `layers/evga/pages/evga/notices/index.vue` | done |
| T7 | Карточка `/evga/notices/[id]` + удаление проекта | `layers/evga/pages/evga/notices/[id].vue` | done |
| T8 | Пункт «Уведомления» в сайдбаре | `layers/base/components/AppShell.vue` | done |
| T9 | Проверка: vitest, SSR-рендеры | — | done |
