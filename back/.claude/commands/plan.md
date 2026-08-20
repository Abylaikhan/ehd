---
description: Составить план реализации (plan.md, data-model, contracts) по спеке
argument-hint: <NNN или slug фичи>
---

Построй план реализации для `specs/$ARGUMENTS/`.

1. Прочитай `spec.md` и `.specify/memory/constitution.md`. Спека без открытых `[NEEDS CLARIFICATION]` — иначе останови и предложи `/clarify`.
2. Проанализируй затрагиваемый код **через Graphify** (Grep/Glob запрещены): `graphify_subgraph`, `graphify_impact` — какие пакеты/файлы/слои меняются; `graphify_path` — как проходит запрос. Для Go-символов — GoLand MCP.
3. Создай по шаблону `.specify/templates/plan-template.md`:
   - `plan.md` — техрешение по слоям (transport/repository/domain), выбор в рамках фиксированного стека, риски, соответствие каждому принципу конституции (заполни Constitution Check).
   - `data-model.md` — сущности, GORM-модели (в `repository/`), схема/индексы, инварианты.
   - `contracts/` — контракты эндпоинтов (запрос/ответ/ошибки) как основа для OpenAPI.
4. Никакого нового стека вне конституции без ADR. Никаких `SELECT *`, нарушений RBAC/RLS, секретов в логах.
5. Каждое проектное решение опирается на факт из инструмента, а не на догадку.

Заверши блоком Constitution Check: по каждому принципу — PASS/риск. Есть непокрытый конфликт → зафиксируй для `/analyze`.
