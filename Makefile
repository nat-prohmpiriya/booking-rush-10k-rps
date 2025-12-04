.PHONY: up down logs ps help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

up: ## Start all services in background
	docker-compose up -d

down: ## Stop all services
	docker-compose down

logs: ## Tail logs for all services
	docker-compose logs -f

ps: ## Check service status
	docker-compose ps

test: ## Run all tests
	go test ./...

lint: ## Run linter
	golangci-lint run
