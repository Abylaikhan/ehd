# План реализации: <название> (NNN-<slug>)

## Техническое решение
В какой layer (`base`/`auth`/`reporter`); PrimeVue-компоненты (из реестра ТЗ); composables/сторы; URL-состояние; дизайн-токены.

## Затронутый код (через Graphify)
- `graphify_subgraph`/`graphify_impact`: layers/компоненты/composables/страницы.
- `graphify_path`: поток данных (страница → composable → useApi → API).

## Компоненты
Ссылка на `components.md` (props/emits/состояния).

## Контракты
Ссылка на `contracts.md` (эндпоинты и типы из `shared/api`).

## Риски и решения
| Риск | Митигация |
|---|---|

## Constitution Check
| Принцип | Статус | Комментарий |
|---|---|---|
| 1. Nuxt Layers | PASS/риск | |
| 2. PrimeVue-only + токены | PASS/риск | |
| 3. Сервер-стейт / Pinia scope | PASS/риск | |
| 4. Права на бэкенде | PASS/риск | |
| 5. Контракт API | PASS/риск | |
| 6. UX и доступность | PASS/риск | |
| 7. Тестирование | PASS/риск | |
