# Makefile фронтенда ЕХД (ehd-web). Запускать из папки front/.
# Docker-таргеты работают с общим стеком через ../docker-compose.dev.yml.

COMPOSE  = docker compose -f ../docker-compose.dev.yml
GRAPHIFY = uvx --from graphifyy graphify
PNPM     = pnpm

.DEFAULT_GOAL := help
.PHONY: help install dev build preview typecheck lint test clean \
        logs sh restart \
        graph graph-update graph-fresh graph-hook graph-clean

help: ## Показать список команд
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# --- Nuxt / pnpm ---
install: ## Установить зависимости (pnpm install)
	$(PNPM) install

dev: ## Nuxt dev-сервер
	$(PNPM) dev

build: ## Сборка SSR (nuxt build)
	$(PNPM) build

preview: ## Предпросмотр production-сборки
	$(PNPM) preview

typecheck: ## Проверка типов (нужен скрипт typecheck: nuxt typecheck)
	@$(PNPM) run typecheck 2>/dev/null || echo "Скрипт 'typecheck' ещё не настроен (добавь \"typecheck\": \"nuxt typecheck\" + vue-tsc/typescript)."

lint: ## Линт (нужен скрипт lint)
	@$(PNPM) run lint 2>/dev/null || echo "Скрипт 'lint' ещё не настроен."

test: ## Тесты (нужен скрипт test: vitest)
	@$(PNPM) run test 2>/dev/null || echo "Скрипт 'test' ещё не настроен (добавь vitest + @nuxt/test-utils)."

clean: ## Очистить артефакты сборки
	rm -rf .nuxt .output .data

# --- Docker (общий стек) ---
logs: ## Логи контейнера ehd-web
	$(COMPOSE) logs -f ehd-web

sh: ## Шелл внутри контейнера ehd-web
	$(COMPOSE) exec ehd-web sh

restart: ## Перезапустить контейнер ehd-web
	$(COMPOSE) restart ehd-web

# --- Graphify Knowledge Graph (граф в ./graphify-out) ---
graph: ## Построить граф с нуля (AST, без API-ключа)
	$(GRAPHIFY) extract . --code-only

graph-update: ## Инкрементально обновить граф (без LLM)
	$(GRAPHIFY) update .

graph-fresh: ## Проверить, не устарел ли граф
	$(GRAPHIFY) check-update .

graph-hook: ## Установить git-hook авто-обновления графа
	$(GRAPHIFY) hook install

graph-clean: ## Удалить граф
	rm -rf graphify-out
