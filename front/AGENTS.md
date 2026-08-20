# ЕХД Frontend (`ehd-web`) — операционные правила агента

Ты работаешь над **фронтендом** ЕХД: Nuxt 4 SSR, TypeScript, PrimeVue 4, доменные Nuxt Layers. Отдельный harness со своим Knowledge Graph. Режим — **Spec-Driven Development (SDD)**.

Правила ОБЯЗАТЕЛЬНЫ и приоритетны над привычками по умолчанию.

---

## 1. MCP-first. Grep/Glob для анализа структуры ЗАПРЕЩЕНЫ

| Задача | ЕДИНСТВЕННО правильный инструмент | Запрещено |
|---|---|---|
| «Где компонент/composable/страница X, где используется» | `mcp__graphify` → `graphify_locate`, `graphify_query` | Grep, Glob, `find` |
| «Как устроен layer / что с чем связано» | `graphify_overview`, `graphify_subgraph`, `graphify_explain`, `graphify_path` | чтение вслепую |
| «Что сломает изменение компонента/типа» | `graphify_impact` | догадки |
| История, blame, diff | `mcp__git` | `Bash(git ...)` наугад |
| Контейнеры/логи/compose | `mcp__docker` | лишние `docker` вызовы |
| Прочитать конкретный известный файл | `Read` | — |

- Grep/Glob перехватываются hook-ом и **блокируются**; обход через `Bash(grep/find …)` — то же нарушение.
- Перед структурными выводами — `graphify_freshness`; пусто/устарело → `graphify_build update=true`. После изменений в коде — снова `graphify_build update=true`.

## 2. Анти-галлюцинации (жёстко)

- Ни одного утверждения о коде без подтверждения инструментом: путь файла, имя компонента/props/composable/стора, ключ дизайн-токена, маршрут, поле API-ответа.
- Не выдумывай API Nuxt 4 / PrimeVue 4 / Pinia. Не уверен — открой исходник/типы через `Read` или проверь версию в `package.json` и документацию.
- Контракт API берётся из сгенерированных типов (`shared/api`, источник — `../back/openapi/`). Поля ответа не выдумывай — сверяйся с типами.
- Требования ТЗ — `../.tz_extracted.txt` (`REP-FR/BR`, Приложение D — реестр экранов). Ссылайся на ID.
- Инструмент вернул ошибку/пусто — так и скажи; не выдавай догадку за факт.

## 3. SDD-гейт: код только после спецификации

`/specify` → `/clarify` → `/plan` → `/tasks` → `/analyze` → `/implement`.

- Нет кода экрана/компонента без принятых `spec.md`, `plan.md`, `tasks.md`.
- Каждая задача трассируется к требованию ТЗ и к затронутым файлам (`graphify_impact`).
- Спека — поведение и UX-контракт (состояния, права, данные), не реализация.
- Соответствие `.specify/memory/constitution.md` обязательно.

## 4. Проверка перед «готово»

- Сборка SSR: `pnpm build`. Типы: `pnpm typecheck` (без ошибок), lint без ошибок.
- Компонентные/юнит-тесты (Vitest + @nuxt/test-utils) для query-state, form validation, permission-состояний; e2e — основной сценарий.
- Визуальная проверка на запущенном стеке (`docker compose -f docker-compose.dev.yml`) на ширинах 1024/1280/1440; проверь Loading/Empty/Error/Access denied.
- Не помечай задачу готовой без наблюдаемого результата.

## 5. Запрещено

- UI-библиотека кроме PrimeVue; inline-стили вместо дизайн-токенов (var(--ehd-*) / токены PrimeVue).
- Хардкод ролей/колонок и «безопасность через скрытие»: capability приходят из API, backend авторизует повторно — фронт НЕ расширяет права.
- Дублирование серверного состояния в Pinia: сервер-стейт через `useAsyncData`/`$fetch`; Pinia только для кросс-страничного (сессия, навигация, справочники, draft).
- `git push`, `rm -rf`, `docker compose down -v` без явного запроса пользователя.
- Новая архитектура вместо Nuxt Layers / новая UI-библиотека — только через ADR и решение пользователя.

## Карта проекта (ориентир; детали — через Graphify)

- `app/` — корень: `app.vue`, `theme.ts` (navy preset), `layouts/`, `assets/css` (дизайн-токены).
- `layers/base` — design system: AppShell, PageHeader, StatusTile, `useApi`.
- `layers/auth` — страницы/стор Auth Module. `layers/reporter` — каталог витрин, `/reporter/{slug}`, admin-конструктор.
- `shared/api` — типы/клиент из OpenAPI. ТЗ: `../.tz_extracted.txt`. Спеки: `specs/`. Конституция: `.specify/memory/constitution.md`.
