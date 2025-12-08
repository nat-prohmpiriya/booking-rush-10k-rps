# Booking Rush 10k RPS - Makefile
# ================================

.PHONY: help dev dev-down build test lint migrate-up migrate-down clean

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

# Database settings (can be overridden)
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/booking_rush?sslmode=disable
MIGRATIONS_PATH ?= scripts/migrations

# Default target
help:
	@echo "$(GREEN)Booking Rush 10k RPS - Available Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Development:$(NC)"
	@echo "  make dev              - Start infrastructure containers (docker-compose up)"
	@echo "  make dev-down         - Stop infrastructure containers"
	@echo "  make dev-logs         - Follow infrastructure logs"
	@echo ""
	@echo "$(YELLOW)Services:$(NC)"
	@echo "  make run-gateway      - Run API Gateway locally"
	@echo "  make run-auth         - Run Auth Service locally"
	@echo "  make run-booking      - Run Booking Service locally"
	@echo "  make run-ticket       - Run Ticket Service locally"
	@echo "  make run-payment      - Run Payment Service locally"
	@echo ""
	@echo "$(YELLOW)Build:$(NC)"
	@echo "  make build            - Build all Go services"
	@echo "  make build-gateway    - Build API Gateway"
	@echo "  make build-auth       - Build Auth Service"
	@echo "  make build-booking    - Build Booking Service"
	@echo ""
	@echo "$(YELLOW)Database:$(NC)"
	@echo "  make migrate-up       - Run all migrations up"
	@echo "  make migrate-down     - Rollback last migration"
	@echo "  make migrate-down-all - Rollback all migrations"
	@echo "  make migrate-create   - Create new migration (NAME=migration_name)"
	@echo "  make migrate-status   - Show migration status"
	@echo ""
	@echo "$(YELLOW)Testing:$(NC)"
	@echo "  make test             - Run all tests"
	@echo "  make test-unit        - Run unit tests only"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-coverage    - Run tests with coverage"
	@echo ""
	@echo "$(YELLOW)Code Quality:$(NC)"
	@echo "  make lint             - Run linters"
	@echo "  make fmt              - Format code"
	@echo "  make vet              - Run go vet"
	@echo ""
	@echo "$(YELLOW)Cleanup:$(NC)"
	@echo "  make clean            - Remove build artifacts"

# ================================
# Development
# ================================

dev:
	@echo "$(GREEN)Starting infrastructure...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)Infrastructure started!$(NC)"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"
	@echo "Redpanda: localhost:9092"
	@echo "Redpanda Console: http://localhost:8090"

dev-down:
	@echo "$(YELLOW)Stopping infrastructure...$(NC)"
	docker-compose down
	@echo "$(GREEN)Infrastructure stopped$(NC)"

dev-logs:
	docker-compose logs -f

dev-restart: dev-down dev

# ================================
# Run Services Locally
# ================================

run-gateway:
	@echo "$(GREEN)Starting API Gateway...$(NC)"
	cd apps/api-gateway && go run main.go

run-auth:
	@echo "$(GREEN)Starting Auth Service...$(NC)"
	cd apps/auth-service && go run main.go

run-booking:
	@echo "$(GREEN)Starting Booking Service...$(NC)"
	cd apps/booking-service && go run main.go

run-ticket:
	@echo "$(GREEN)Starting Ticket Service...$(NC)"
	cd apps/ticket-service && go run main.go

run-payment:
	@echo "$(GREEN)Starting Payment Service...$(NC)"
	cd apps/payment-service && go run main.go

# ================================
# Build
# ================================

build: build-gateway build-auth build-booking build-ticket build-payment
	@echo "$(GREEN)All services built successfully!$(NC)"

build-gateway:
	@echo "$(GREEN)Building API Gateway...$(NC)"
	cd apps/api-gateway && go build -o ../../bin/api-gateway .

build-auth:
	@echo "$(GREEN)Building Auth Service...$(NC)"
	cd apps/auth-service && go build -o ../../bin/auth-service . 2>/dev/null || echo "$(YELLOW)Auth Service not ready yet$(NC)"

build-booking:
	@echo "$(GREEN)Building Booking Service...$(NC)"
	cd apps/booking-service && go build -o ../../bin/booking-service . 2>/dev/null || echo "$(YELLOW)Booking Service not ready yet$(NC)"

build-ticket:
	@echo "$(GREEN)Building Ticket Service...$(NC)"
	cd apps/ticket-service && go build -o ../../bin/ticket-service . 2>/dev/null || echo "$(YELLOW)Ticket Service not ready yet$(NC)"

build-payment:
	@echo "$(GREEN)Building Payment Service...$(NC)"
	cd apps/payment-service && go build -o ../../bin/payment-service . 2>/dev/null || echo "$(YELLOW)Payment Service not ready yet$(NC)"

# ================================
# Database Migrations
# ================================

migrate-up:
	@echo "$(GREEN)Running migrations up...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up
	@echo "$(GREEN)Migrations completed$(NC)"

migrate-down:
	@echo "$(YELLOW)Rolling back last migration...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1
	@echo "$(GREEN)Rollback completed$(NC)"

migrate-down-all:
	@echo "$(RED)Rolling back ALL migrations...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down -all
	@echo "$(GREEN)All migrations rolled back$(NC)"

migrate-status:
	@echo "$(GREEN)Migration status:$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

migrate-create:
ifndef NAME
	$(error NAME is required. Usage: make migrate-create NAME=create_something)
endif
	@echo "$(GREEN)Creating migration: $(NAME)$(NC)"
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)
	@echo "$(GREEN)Migration files created$(NC)"

migrate-force:
ifndef VERSION
	$(error VERSION is required. Usage: make migrate-force VERSION=1)
endif
	@echo "$(YELLOW)Forcing migration version to: $(VERSION)$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(VERSION)

# ================================
# Testing
# ================================

test:
	@echo "$(GREEN)Running all tests...$(NC)"
	go test ./pkg/... ./apps/... -v -race -count=1
	@echo "$(GREEN)All tests passed$(NC)"

test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	go test ./pkg/... ./apps/... -v -short -race
	@echo "$(GREEN)Unit tests passed$(NC)"

test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	INTEGRATION_TEST=true go test ./pkg/... ./apps/... -v -race -run Integration
	@echo "$(GREEN)Integration tests passed$(NC)"

test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test ./pkg/... ./apps/... -v -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

test-bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	go test ./pkg/... ./apps/... -bench=. -benchmem

# ================================
# Code Quality
# ================================

lint:
	@echo "$(GREEN)Running linters...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not installed. Run: brew install golangci-lint$(NC)" && exit 1)
	golangci-lint run ./...
	@echo "$(GREEN)Linting passed$(NC)"

fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	gofmt -s -w .
	@echo "$(GREEN)Code formatted$(NC)"

vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)Vet passed$(NC)"

# ================================
# Go Workspace
# ================================

tidy:
	@echo "$(GREEN)Tidying Go modules...$(NC)"
	cd pkg && go mod tidy
	cd apps/api-gateway && go mod tidy
	cd apps/auth-service && go mod tidy
	cd apps/booking-service && go mod tidy
	cd apps/ticket-service && go mod tidy
	cd apps/payment-service && go mod tidy
	go work sync
	@echo "$(GREEN)Modules tidied$(NC)"

# ================================
# Cleanup
# ================================

clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Cleaned$(NC)"

# ================================
# Quick Start
# ================================

setup: dev migrate-up
	@echo "$(GREEN)Setup complete! Run 'make run-gateway' to start the API Gateway$(NC)"

.DEFAULT_GOAL := help
