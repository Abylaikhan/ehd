# Задачи: 005-evga-registry-ui

| # | Задача | Файлы | Статус |
|---|---|---|---|
| T1 | Типы Evga* в shared/api | `shared/api/types.ts` | done |
| T2 | Слой evga: конфиг + composable API | `layers/evga/nuxt.config.ts`, `layers/evga/composables/useEvga.ts` | done |
| T3 | Страница реестра `/evga`: фильтры, DataTable, Paginator, сортировка, экспорт, состояния (unmapped/source/denied) | `layers/evga/pages/evga/index.vue` | done |
| T4 | Карточка `/evga/{id}` | `layers/evga/pages/evga/[id].vue` | done |
| T5 | Секция «ОБМ ЕВГА» в сайдбаре по ролям | `layers/base/components/AppShell.vue` | done |
| T6 | Проверка: typecheck, lint, SSR-рендер на стеке | — | done (06.09.2026) |
