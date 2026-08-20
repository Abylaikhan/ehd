---
description: Составить план реализации фронтенда (plan.md, компоненты, контракты) по спеке
argument-hint: <NNN или slug фичи>
---

Построй план для `specs/$ARGUMENTS/`.

1. Прочитай `spec.md` и `.specify/memory/constitution.md`. Есть открытые `[NEEDS CLARIFICATION]` — останови, предложи `/clarify`.
2. Влияние на код через Graphify (Grep/Glob запрещены): `graphify_subgraph`/`graphify_impact` — какие layers/компоненты/composables/страницы; `graphify_path` — поток данных (страница → composable → useApi → API).
3. Создай по `.specify/templates/plan-template.md`:
   - `plan.md` — в какой layer, какие PrimeVue-компоненты (из реестра ТЗ), какие composables/сторы, состояние в URL, дизайн-токены; заполни Constitution Check.
   - `components.md` — список новых/изменяемых компонентов с props/emits и состояниями.
   - `contracts.md` — какие эндпоинты и типы из `shared/api` используются (сверься с `../back/openapi/`).
4. PrimeVue — единственная UI-библиотека; токены, не inline-стили; фронт не расширяет права; сервер-стейт не дублируется в Pinia.

Заверши Constitution Check по всем 7 принципам; конфликт → фиксируй для `/analyze`.
