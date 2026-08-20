# Задачи: <название> (NNN-<slug>)

Порядок исполнения = порядок в списке. `[P]` — параллелизуемые. Тест раньше реализации.

| ID | Задача | Тип | Трассировка | Затронутые файлы (graphify_impact) | Готово когда |
|---|---|---|---|---|---|
| T001 | Типы из OpenAPI для … | types | REP-FR-### | shared/api/… | pnpm typecheck ok |
| T002 | Composable useXxx (данные + состояния) | composable | REP-FR-### | layers/reporter/composables/… | тест зелёный |
| T003 | Компонент со всеми состояниями | component | Прил. D | layers/…/components/… | Loading/Empty/Error/AccessDenied видны |
| … | | | | | |

## Порядок по умолчанию
типы (OpenAPI) → composables/сторы → компоненты (все состояния) → страница/маршрут → тесты (Vitest) → визуальная проверка.

## Обязательные задачи
- Состояния Loading/Empty/Error/Access denied.
- Проверка, что UI не расширяет права (capability из API).
