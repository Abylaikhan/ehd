---
description: Разбить план на упорядоченный, трассируемый список задач (tasks.md)
argument-hint: <NNN или slug фичи>
---

Сгенерируй `specs/$ARGUMENTS/tasks.md` по `.specify/templates/tasks-template.md`.

1. Прочитай `spec.md`, `plan.md`, `components.md`, `contracts.md`.
2. Атомарные задачи в порядке исполнения (тест раньше реализации). Для каждой:
   - ID (`T001`…), действие.
   - Трассировка к `REP-FR/BR` / Приложению D и разделу спеки.
   - Затронутые файлы — через `graphify_impact` (не угадывай).
   - Тип: test | component | page | composable | store | types | docs. `[P]` — независимые.
   - Критерий готовности (команда/наблюдаемый результат).
3. Порядок по умолчанию: типы из OpenAPI → composables/сторы → компоненты (со всеми состояниями) → страница/маршрут → тесты (Vitest) → визуальная проверка.
4. Обязательные задачи на состояния Loading/Empty/Error/Access denied и на проверку, что UI не расширяет права.

Не выдумывай пути и имена — только из Graphify/плана.
