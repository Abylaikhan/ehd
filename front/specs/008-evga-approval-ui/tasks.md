# Задачи: 008-evga-approval-ui

| # | Задача | Файлы | Статус |
|---|---|---|---|
| T1 | Типы EvgaRoute*/EvgaParticipant | `shared/api/types.ts` | done |
| T2 | API-методы route/participants/submit/approve/reject | `layers/evga/composables/useEvga.ts` | done |
| T3 | routeComplete + vitest | `layers/evga/utils/routeForm.ts` (+test) | done |
| T4 | Компонент маршрута (редактор + ход исполнения + действия) | `layers/evga/components/EvgaRouteCard.vue` | done |
| T5 | Интеграция в карточку уведомления | `layers/evga/pages/evga/notices/[id].vue` | done |
| T6 | Проверка: vitest, SSR, живой цикл submit→approve через UI-эндпоинты | — | done |
