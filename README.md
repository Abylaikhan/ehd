# ЕХД — sandbox локальной разработки

Единый сервис «Единое хранилище данных»: модульный монолит — один Go backend (Auth Module + Reporter Module), один Nuxt 4 SSR frontend, PostgreSQL как основная БД, ClickHouse как read-only источник Reporter.

## Стек

| Компонент | Технология |
|---|---|
| Backend (`back/`) | Go 1.26, Fiber, GORM (AutoMigrate), zap, viper, clickhouse-go; архитектура по [go-clean-template](https://github.com/evrone/go-clean-template) |
| Frontend (`front/`) | Nuxt 4 SSR (Node 24 LTS), TypeScript, PrimeVue 4 (кастомный navy-preset), Pinia; доменные Nuxt Layers |
| БД | PostgreSQL 18 (alpine); схемы создаются GORM AutoMigrate при старте |
| Источник | ClickHouse (read-only пользователь) |

## Быстрый старт

```bash
cp .env.example .env
make up        # сборка и запуск; при старте ehd-api сам делает AutoMigrate + seed админа
```

Точки входа:

- Frontend: http://localhost:3000 (панель мониторинга состояния sandbox)
- API: http://localhost:8080 (`/livez`, `/readyz`, `/api/v1/auth/ping`, `/api/v1/reporter/ping`)
- PostgreSQL: `localhost:5433` (`ehd`/`ehd_local`) — host-порт вынесен в `POSTGRES_HOST_PORT`
- ClickHouse: `localhost:8123` (тестовая таблица `ehd_src.demo_transactions`, read-only юзер `reporter_ro`)

Первый администратор создаётся автоматически из переменных `ADMIN_*` при старте.
Остальные команды: `make down`, `make destroy` (сброс данных), `make logs`, `make ps`, `make seed` (принудительный ре-seed).

## Структура

```
back/                     # ehd-api: Go, модульный монолит
  cmd/ehd-api/            # entrypoint
  cmd/seed/               # ручной seed администратора (GORM)
  config/                 # конфигурация через viper (env)
  internal/app/           # composition root: AutoMigrate, wiring, graceful shutdown
  internal/modules/
    auth/                 # domain / repository (GORM+seed) / transport/http / contract
    reporter/             # domain / repository (GORM) / transport/http
  pkg/                    # httpserver (Fiber), postgres (GORM), clickhouse, logger (zap)
  openapi/                # OpenAPI 3.1 — источник типов frontend
front/                    # ehd-web: Nuxt 4 SSR
  app/                    # корень: app.vue, theme.ts (navy preset), layouts, assets/css
  layers/base/            # design system: AppShell, PageHeader, StatusTile, useApi
  layers/auth/            # страницы и состояние Auth Module
  layers/reporter/        # каталог витрин, динамическая таблица, admin-конструктор
  shared/api/             # сгенерированный OpenAPI-клиент (позже)
infra/clickhouse/init/    # тестовый источник: БД, таблица, read-only пользователь
```

Правила архитектуры backend: зависимости направлены внутрь (`transport → repository → domain`); модули общаются только через `contract`-интерфейсы, без сетевых вызовов; схемы `auth`/`reporter` и таблицы создаются GORM AutoMigrate при старте приложения (для sandbox; в production вернёмся к контролируемым миграциям по ТЗ). SIGTERM → graceful shutdown с закрытием активных соединений.

## Agent Harness (SDD)

В `back/` и `front/` заложены отдельные харнессы для Spec-Driven Development с Graphify Knowledge Graph и MCP-серверами. Активируются запуском агента из папки приложения (`cd back && claude`). Полное описание, MCP-серверы и предпосылки — в `HARNESS.md`.

Полное ТЗ: `TZ_Reporter_ClickHouse.docx`.
