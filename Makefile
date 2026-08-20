COMPOSE = docker compose -f docker-compose.dev.yml

.PHONY: up down destroy ps logs seed

up: ## Собрать и запустить всё окружение (AutoMigrate и seed выполняются при старте ehd-api)
	$(COMPOSE) up -d --build

down: ## Остановить окружение
	$(COMPOSE) down

destroy: ## Остановить и удалить volumes (полный сброс данных)
	$(COMPOSE) down -v

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f ehd-api ehd-web

seed: ## Принудительно пересоздать администратора (обычно не нужно — seed идёт при старте)
	$(COMPOSE) exec ehd-api go run ./cmd/seed
