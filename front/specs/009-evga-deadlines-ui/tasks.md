# Задачи: 009-evga-deadlines-ui

| # | Задача | FR | Файлы | Статус |
|---|---|---|---|---|
| T1 | Типы: `exec_due`/`deadline_state` в `EvgaRecord`; `confirmed_no_decision` в `EvgaRegistryParams` | FR-5 | `shared/api/types.ts`, `layers/evga/composables/useEvga.ts` | done |
| T2 | Реестр: колонка «Срок» (дата ДД.ММ.ГГГГ + бейдж истёк/истекает), подсветка строк по `deadline_state` | FR-1/2/4 | `layers/evga/pages/evga/index.vue` | done |
| T3 | Быстрый фильтр «Подтверждено без решения» (ToggleButton, немедленное применение, сброс) | FR-3 | `layers/evga/pages/evga/index.vue` | done |
| T4 | Проверка: `pnpm test` (vitest) зелёный; SSR-рендер `/evga` без ошибок; визуальная сверка подсветки/фильтра на стеке | приёмка | стек | done |
