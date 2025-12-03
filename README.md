# Booking Rush 10k RPS

High-performance ticket booking microservices system designed to handle **10,000 Requests Per Second** without overselling.

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23 |
| Framework | Gin |
| Database | PostgreSQL 16 |
| Cache | Redis 7 (Lua Scripts) |
| Message Broker | Kafka |
| Observability | OpenTelemetry |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
│                    (Rate Limit, Auth, Routing)                   │
└─────────────────────────────────────────────────────────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        ▼                        ▼                        ▼
┌───────────────┐       ┌───────────────┐       ┌───────────────┐
│  Auth Service │       │ Ticket Service│       │Booking Service│
│   (JWT Auth)  │       │ (Events/Shows)│       │ (10k RPS Core)│
└───────────────┘       └───────────────┘       └───────┬───────┘
                                                        │
                                                        ▼ Kafka
                                                ┌───────────────┐
                                                │Payment Service│
                                                │   (Consumer)  │
                                                └───────────────┘
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker & Docker Compose
- Make

### Start Infrastructure

```bash
# Start PostgreSQL, Redis, Kafka, and monitoring tools
make dev

# Check running services
make ps
```

### Services Available

| Service | URL |
|---------|-----|
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| Kafka | localhost:9093 |
| Kafka UI | http://localhost:8080 |
| Redis Commander | http://localhost:8081 |

### Run Migrations

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
make migrate-up
```

### Run Services

```bash
# Run individual services
make run-gateway
make run-auth
make run-ticket
make run-booking
make run-payment
```

## Project Structure

```
booking-rush-10k-rps/
├── apps/                   # Microservices
│   ├── api-gateway/        # Entry point, routing, rate limiting
│   ├── auth-service/       # JWT authentication
│   ├── ticket-service/     # Event & show management
│   ├── booking-service/    # Core booking (Redis Lua + Kafka)
│   └── payment-service/    # Payment processing (Kafka consumer)
│
├── pkg/                    # Shared packages
│   ├── config/             # Configuration (Viper)
│   ├── logger/             # Structured logging (Zap)
│   ├── response/           # API response wrapper
│   ├── errors/             # Custom error types
│   ├── middleware/         # Common middlewares
│   ├── database/           # PostgreSQL (pgx)
│   ├── redis/              # Redis client
│   ├── kafka/              # Kafka producer/consumer
│   └── telemetry/          # OpenTelemetry setup
│
├── migrations/             # Database migrations
├── scripts/                # Utility scripts
├── tests/                  # Integration & load tests
│   ├── integration/
│   └── k6/                 # k6 load test scripts
│
├── web/                    # Frontend (Next.js 15)
├── docker-compose.yml      # Local development
├── go.work                 # Go Workspace
└── Makefile                # Common commands
```

## Make Commands

```bash
make help           # Show all commands

# Docker
make dev            # Start infrastructure
make down           # Stop infrastructure
make logs           # View logs

# Go
make build          # Build all services
make test           # Run tests
make lint           # Run linter

# Database
make migrate-up     # Run migrations
make migrate-down   # Rollback migration
make migrate-create name=xxx  # Create new migration

# Load Testing
make load-test      # Run k6 load test
```

## Key Features

- **10k RPS Booking**: Redis Lua scripts for atomic seat reservation
- **Zero Overselling**: Atomic operations prevent race conditions
- **Event-Driven**: Kafka for async processing
- **Rate Limiting**: Token Bucket with burst support
- **Virtual Queue**: Waiting room for flash sales
- **Observability**: OpenTelemetry with Jaeger, Prometheus, Grafana

## Documentation

- [Specification](.docs/01-spec.md)
- [Development Roadmap](.docs/02-task.md)
- [Known Issues](.docs/03-issue.md)

## Performance Targets

| Metric | Target |
|--------|--------|
| RPS (Reserve endpoint) | 10,000 |
| Server P99 Latency | < 50ms |
| E2E P99 Latency | < 200ms |
| Error Rate | < 0.1% |
| Overselling | 0 |

## License

MIT
