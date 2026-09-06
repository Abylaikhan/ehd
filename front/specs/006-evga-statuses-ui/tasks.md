# Задачи: 006-evga-statuses-ui

| # | Задача | Файлы | Статус |
|---|---|---|---|
| T1 | Типы: history, bulk-отчёт, activities в references | `shared/api/types.ts` | done |
| T2 | API: changeStatus/bulkStatus/history в composable | `layers/evga/composables/useEvga.ts` | done |
| T3 | Клиентская матрица переходов + vitest | `layers/evga/utils/statusFlow.ts`, `statusFlow.test.ts` | done |
| T4 | Диалог смены статуса (общий для реестра и карточки) | `layers/evga/components/EvgaStatusDialog.vue` | done |
| T5 | Реестр: чекбоксы, кнопка «Сменить статус», отчёт bulk | `layers/evga/pages/evga/index.vue` | done |
| T6 | Карточка: кнопка смены + история статусов | `layers/evga/pages/evga/[id].vue` | done |
| T7 | Проверка: vitest, SSR-рендер, живой bulk через UI-эндпоинты | — | done |
