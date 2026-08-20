# Makefile бэкенда ЕХД (ehd-api). Запускать из папки back/.
# Docker-таргеты работают с общим стеком через ../docker-compose.dev.yml.

COMPOSE  = docker compose -f ../docker-compose.dev.yml
GRAPHIFY = uvx --from graphifyy graphify

.DEFAULT_GOAL := help
.PHONY: help tidy build run test vet fmt fmt-check seed \
        logs sh restart \
        graph graph-update graph-fresh graph-hook graph-clean

help: ## Показать список команд
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# --- Go ---
tidy: ## go mod tidy
	go mod tidy

build: ## Собрать все пакеты (go build ./...)
	go build ./...

run: ## Запустить API локально (нужны переменные окружения)
	go run ./cmd/ehd-api

test: ## Прогнать тесты (go test ./...)
	go test ./...

vet: ## Статический анализ (go vet ./...)
	go vet ./...

fmt: ## Отформатировать код (gofmt -w .)
	gofmt -w .

fmt-check: ## Проверить форматирование без изменений (для CI)
	@out="$$(gofmt -l .)"; test -z "$$out" || { echo "gofmt: не отформатировано:"; echo "$$out"; exit 1; }

seed: ## Пересоздать администратора (GORM)
	go run ./cmd/seed

# --- Docker (общий стек) ---
logs: ## Логи контейнера ehd-api
	$(COMPOSE) logs -f ehd-api

sh: ## Шелл внутри контейнера ehd-api
	$(COMPOSE) exec ehd-api sh

restart: ## Перезапустить контейнер ehd-api
	$(COMPOSE) restart ehd-api

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
