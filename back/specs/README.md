# Спецификации бэкенда ЕХД (SDD)

Каждая фича — папка `NNN-<slug>/` со своими артефактами:

```
NNN-<slug>/
  spec.md          # /specify → /clarify  (что и зачем)
  plan.md          # /plan                (как, по слоям)
  data-model.md    # /plan                (сущности, GORM-модели, схема)
  contracts/       # /plan                (контракты эндпоинтов → OpenAPI)
  tasks.md         # /tasks               (упорядоченные трассируемые задачи)
```

## Поток

`/specify` → `/clarify` → `/plan` → `/tasks` → `/analyze` → `/implement`

Правила потока — в `../CLAUDE.md`. Принципы — в `../.specify/memory/constitution.md`. Требования — в `../../.tz_extracted.txt` (`REP-FR/BR`).

Анализ структуры только через Graphify MCP (Grep/Glob заблокированы). Первый шаг в новой сессии: `graphify_freshness`, при необходимости `graphify_build`.
