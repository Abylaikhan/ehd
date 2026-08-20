# Agent Harness — Spec-Driven Development (ЕХД)

Два независимых харнесса: **`back/`** (бэкенд) и **`front/`** (фронтенд). У каждого свой Knowledge Graph (Graphify), свои MCP-серверы, свои строгие правила, свой SDD-поток.

## Активация

Claude Code читает `.mcp.json`, `.claude/settings.json` (hooks/permissions) и слэш-команды из **корня рабочей директории**. Поэтому харнесс активируется, когда агент запущен в папке приложения:

```bash
cd back  && claude     # активны back/.mcp.json, hooks, /specify…/implement, субагенты
cd front && claude      # активны front/.mcp.json, hooks, команды, субагенты
```

Из корня `ehd2` нессированные `CLAUDE.md` подхватываются как контекст, но MCP-серверы и hook Grep/Glob — только при запуске из `back/` или `front/`.

## MCP-серверы

| Сервер | back | front | Назначение | Команда |
|---|---|---|---|---|
| **graphify** | ✅ (граф `back/`) | ✅ (граф `front/`) | Knowledge Graph кода: структура, связи, impact | `uvx --from git+https://github.com/yasinyaman/graphify-mcp graphify-mcp-server` |
| **git** | ✅ | ✅ | История, blame, diff (репозиторий — корень `ehd2`) | `uvx mcp-server-git --repository …/ehd2` |
| **docker** | ✅ | ✅ | Контейнеры, логи, compose | `uvx docker-mcp` |
| **jetbrains** | ✅ | — | GoLand MCP: навигация по Go-символам, find usages, рефактор | `npx -y @jetbrains/mcp-proxy` |

Граф каждого приложения хранится отдельно в `<app>/.graphify-out/` (в git не коммитится).

## Строгие правила (анти-галлюцинации, MCP-first)

- Анализ структуры кода — **только Graphify** (`graphify_locate/query/overview/subgraph/impact`). **Grep и Glob заблокированы** PreToolUse-hook'ом (`.claude/hooks/redirect-to-graphify.sh`) и перенаправляют на Graphify. `Read` конкретного файла разрешён.
- Ни одного утверждения о коде без подтверждения инструментом; не выдумывать пути, символы, API, поля.
- Код — только после принятых spec → plan → tasks; каждая задача трассируется к `REP-FR/BR`.
- Полные правила: `back/CLAUDE.md`, `front/CLAUDE.md`. Принципы: `<app>/.specify/memory/constitution.md`.

## Makefile приложения

У каждого приложения свой `Makefile` (запускать из его папки). `make` без аргументов — список команд.

- **back/**: `tidy build run test vet fmt fmt-check seed` · `logs sh restart` (контейнер ehd-api).
- **front/**: `install dev build preview typecheck lint test clean` · `logs sh restart` (контейнер ehd-web).
- **Graphify (оба)**: `make graph` — построить граф (AST, без API-ключа, `graphify extract . --code-only`); `make graph-update` — инкрементально (без LLM); `make graph-fresh` — проверка свежести; `make graph-hook` — git-hook авто-обновления; `make graph-clean` — удалить. CLI берётся через `uvx --from graphifyy graphify` (пакет `graphifyy`); граф пишется в `<app>/graphify-out/`, откуда его читает Graphify MCP.

## SDD-поток

`/specify` → `/clarify` → `/plan` → `/tasks` → `/analyze` → `/implement` (+ `/constitution`).
Артефакты — в `<app>/specs/NNN-<slug>/`. Шаблоны — в `<app>/.specify/templates/`.
Субагенты: `spec-author`, `planner`, `implementer`, `verifier` (в `<app>/.claude/agents/`).

## Предпосылки окружения

1. **uv (даёт `uvx`)** — нужен для graphify/git/docker MCP:
   `brew install uv` (или `curl -LsSf https://astral.sh/uv/install.sh | sh`).
2. **git-репозиторий** — git MCP и Graphify freshness требуют репозиторий в корне:
   `git init` в `…/ehd2` (одно моно-репо на оба приложения).
3. **Первый прогон Graphify** — построить граф из Makefile приложения (см. ниже): `make graph`. Инкрементально — `make graph-update`; авто-обновление по коммитам — `make graph-hook`. Внутри агента доступен MCP-инструмент `graphify_build`.
4. **GoLand MCP (бэкенд)** — GoLand 2025.2+ с открытым проектом `back/` и включённым плагином MCP Server; внешний клиент — через `@jetbrains/mcp-proxy`.
