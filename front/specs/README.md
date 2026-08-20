# Спецификации фронтенда ЕХД (SDD)

Каждая фича/экран — папка `NNN-<slug>/`:

```
NNN-<slug>/
  spec.md          # /specify → /clarify  (что и зачем, UX-контракт)
  plan.md          # /plan                (layer, PrimeVue-компоненты, потоки)
  components.md    # /plan                (компоненты: props/emits/состояния)
  contracts.md     # /plan                (эндпоинты и типы из shared/api)
  tasks.md         # /tasks               (упорядоченные трассируемые задачи)
```

## Поток

`/specify` → `/clarify` → `/plan` → `/tasks` → `/analyze` → `/implement`

Правила — в `../CLAUDE.md`. Принципы — в `../.specify/memory/constitution.md`. Требования — в `../../.tz_extracted.txt` (`REP-FR/BR`, Приложение D). Типы API — из `../../back/openapi/` → `shared/api`.

Анализ структуры только через Graphify MCP (Grep/Glob заблокированы). Первый шаг в новой сессии: `graphify_freshness`, при необходимости `graphify_build`.
