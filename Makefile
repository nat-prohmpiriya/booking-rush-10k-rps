.PHONY: help dev down logs ps build test lint migrate seed clean

# =============================================================================
# Variables
# =============================================================================
DOCKER_COMPOSE = docker-compose
GO = go
MIGRATE = migrate

# Colors
GREEN  := \033[0;32m
YELLOW := \033[0;33m
CYAN   := \033[0;36m
RESET  := \033[0m

# =============================================================================
# Help
# =============================================================================
help: ## Show this help message
	@echo "$(CYAN)Booking Rush 10k RPS - Makefile Commands$(RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'

# =============================================================================
# Docker Commands
# =============================================================================
dev: ## Start all infrastructure services
	@echo "$(YELLOW)Starting infrastructure...$(RESET)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Infrastructure started!$(RESET)"
	@echo ""
	@echo "Services:"
	@echo "  - PostgreSQL:      localhost:5432"
	@echo "  - Redis:           localhost:6379"
	@echo "  - Kafka:           localhost:9093"
	@echo "  - Kafka UI:        http://localhost:8080"
	@echo "  - Redis Commander: http://localhost:8081"

down: ## Stop all services
	@echo "$(YELLOW)Stopping infrastructure...$(RESET)"
	$(DOCKER_COMPOSE) down
	@echo "$(GREEN)Infrastructure stopped!$(RESET)"

down-v: ## Stop all services and remove volumes
	@echo "$(YELLOW)Stopping infrastructure and removing volumes...$(RESET)"
	$(DOCKER_COMPOSE) down -v
	@echo "$(GREEN)Infrastructure stopped and volumes removed!$(RESET)"

logs: ## Show logs from all services
	$(DOCKER_COMPOSE) logs -f

logs-kafka: ## Show Kafka logs
	$(DOCKER_COMPOSE) logs -f kafka

logs-redis: ## Show Redis logs
	$(DOCKER_COMPOSE) logs -f redis

logs-postgres: ## Show PostgreSQL logs
	$(DOCKER_COMPOSE) logs -f postgres

ps: ## Show running containers
	$(DOCKER_COMPOSE) ps

# =============================================================================
# Go Commands
# =============================================================================
build: ## Build all services
	@echo "$(YELLOW)Building all services...$(RESET)"
	@for service in api-gateway auth-service ticket-service booking-service payment-service; do \
		echo "Building $$service..."; \
		cd apps/$$service && $(GO) build -o bin/$$service ./cmd/... && cd ../..; \
	done
	@echo "$(GREEN)Build complete!$(RESET)"

test: ## Run all tests
	@echo "$(YELLOW)Running tests...$(RESET)"
	$(GO) test ./... -v -cover
	@echo "$(GREEN)Tests complete!$(RESET)"

test-coverage: ## Run tests with coverage report
	@echo "$(YELLOW)Running tests with coverage...$(RESET)"
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

lint: ## Run linter
	@echo "$(YELLOW)Running linter...$(RESET)"
	golangci-lint run ./...
	@echo "$(GREEN)Lint complete!$(RESET)"

fmt: ## Format code
	@echo "$(YELLOW)Formatting code...$(RESET)"
	$(GO) fmt ./...
	@echo "$(GREEN)Format complete!$(RESET)"

tidy: ## Tidy go modules
	@echo "$(YELLOW)Tidying modules...$(RESET)"
	@for service in api-gateway auth-service ticket-service booking-service payment-service; do \
		echo "Tidying $$service..."; \
		cd apps/$$service && $(GO) mod tidy && cd ../..; \
	done
	cd pkg && $(GO) mod tidy
	@echo "$(GREEN)Tidy complete!$(RESET)"

# =============================================================================
# Database Commands
# =============================================================================
migrate-up: ## Run database migrations
	@echo "$(YELLOW)Running migrations...$(RESET)"
	$(MIGRATE) -path ./migrations -database "postgres://booking_user:booking_pass@localhost:5432/booking_rush?sslmode=disable" up
	@echo "$(GREEN)Migrations complete!$(RESET)"

migrate-down: ## Rollback last migration
	@echo "$(YELLOW)Rolling back migration...$(RESET)"
	$(MIGRATE) -path ./migrations -database "postgres://booking_user:booking_pass@localhost:5432/booking_rush?sslmode=disable" down 1
	@echo "$(GREEN)Rollback complete!$(RESET)"

migrate-reset: ## Reset all migrations
	@echo "$(YELLOW)Resetting all migrations...$(RESET)"
	$(MIGRATE) -path ./migrations -database "postgres://booking_user:booking_pass@localhost:5432/booking_rush?sslmode=disable" drop -f
	@echo "$(GREEN)Reset complete!$(RESET)"

migrate-create: ## Create a new migration (usage: make migrate-create name=auth_create_users)
	@echo "$(YELLOW)Creating migration: $(name)$(RESET)"
	$(MIGRATE) create -ext sql -dir ./migrations -seq $(name)
	@echo "$(GREEN)Migration created!$(RESET)"

seed: ## Seed database with sample data
	@echo "$(YELLOW)Seeding database...$(RESET)"
	./scripts/seed.sh
	@echo "$(GREEN)Seed complete!$(RESET)"

# =============================================================================
# Service Commands
# =============================================================================
run-gateway: ## Run API Gateway
	cd apps/api-gateway && $(GO) run ./cmd/...

run-auth: ## Run Auth Service
	cd apps/auth-service && $(GO) run ./cmd/...

run-ticket: ## Run Ticket Service
	cd apps/ticket-service && $(GO) run ./cmd/...

run-booking: ## Run Booking Service
	cd apps/booking-service && $(GO) run ./cmd/...

run-payment: ## Run Payment Service
	cd apps/payment-service && $(GO) run ./cmd/...

# =============================================================================
# Load Testing
# =============================================================================
load-test: ## Run k6 load test
	@echo "$(YELLOW)Running load test...$(RESET)"
	k6 run tests/k6/booking_load_test.js
	@echo "$(GREEN)Load test complete!$(RESET)"

# =============================================================================
# Cleanup
# =============================================================================
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning...$(RESET)"
	@rm -rf apps/*/bin
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete!$(RESET)"
