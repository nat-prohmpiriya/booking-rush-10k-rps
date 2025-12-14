# Booking Rush

High-Performance Ticket Booking System designed for **10,000 RPS** with **zero overselling**.

Built with Go microservices for critical path operations and Redis Lua scripts for atomic seat reservations.

## Highlights

| Metric | Value |
|--------|-------|
| Throughput | **10,000 RPS** validated |
| Latency | **< 100ms** p99 |
| DB Write Reduction | **99%** via Kafka batching |
| Infrastructure Cost | **$6.5/month** (4 CPU / 8GB RAM) |
| Overselling Rate | **0%** guaranteed |

## Key Features

- **Zero Overselling**: Multi-layer defense with Redis Lua + Saga + DB constraints
- **Virtual Queue**: Redis Sorted Set for fair flash sale access (FIFO)
- **High Concurrency**: 10,000 concurrent users with controlled release
- **Idempotent APIs**: Safe retry handling for all booking operations
- **Event-Driven**: Transactional Outbox + Saga pattern
- **Full Observability**: OpenTelemetry → Tempo, Prometheus, Loki

## Why This Tech Stack?

| Service Type | Language | Reason |
|--------------|----------|--------|
| Critical Path (Booking, Auth) | **Go + Gin** | 50k RPS, 2ms latency, 10MB binary |
| Async Workers (Notification) | **NestJS** | Better DX, rich ecosystem |

> Different problems need different tools. Critical path optimizes for performance, async workers optimize for developer experience.

## Architecture

```
┌──────────┐      ┌───────────────┐      ┌───────────────┐
│   User   │─────▶│   Frontend    │─────▶│  API Gateway  │
└──────────┘      │  Next.js:3000 │      │     :8080     │
                  └───────────────┘      └───────┬───────┘
                                                 │
                  ┌──────────┬───────────────────┼───────────┐
                  ▼          ▼                   ▼           ▼
            ┌──────────┐ ┌──────────┐     ┌──────────┐ ┌──────────┐
            │   Auth   │ │  Ticket  │     │ Booking  │ │ Payment  │
            │  :8081   │ │  :8082   │     │  :8083   │ │  :8084   │
            └────┬─────┘ └────┬─────┘     └────┬─────┘ └────┬─────┘
                 │            │                │            │
                 ▼            ▼                ▼            ▼
            ┌─────────┐ ┌─────────┐      ┌─────────┐  ┌─────────┐
            │ auth_db │ │ticket_db│      │booking_db│ │payment_db│
            └─────────┘ └─────────┘      └────┬─────┘ └─────────┘
                                              │
                                              ▼
                                        ┌──────────┐
                                        │  Redis   │ ← Lua Scripts
                                        └──────────┘
```

### Observability

```
┌──────────────────────────────────────────────────────────────┐
│                      All Services                             │
│  (API Gateway, Auth, Ticket, Booking, Payment)               │
└──────────────────────────┬───────────────────────────────────┘
                           │ OTLP (traces, metrics, logs)
                           ▼
                  ┌─────────────────┐
                  │  OTel Collector │
                  │     :4317       │
                  └────────┬────────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
    ┌──────────┐    ┌────────────┐   ┌──────────┐
    │  Tempo   │    │ Prometheus │   │   Loki   │
    │ (traces) │    │ (metrics)  │   │  (logs)  │
    └────┬─────┘    └─────┬──────┘   └────┬─────┘
         │                │               │
         └────────────────┼───────────────┘
                          ▼
                    ┌──────────┐
                    │ Grafana  │
                    │  :3000   │
                    └──────────┘
```

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24 + Gin |
| Frontend | Next.js 16 + TypeScript |
| Database | PostgreSQL 17 |
| Cache | Redis 8 |
| Message Queue | Redpanda (Kafka-compatible) |
| Tracing | OpenTelemetry → Tempo |
| Metrics | Prometheus |
| Payment | Stripe |

## Services

| Service | Port | Description |
|---------|------|-------------|
| API Gateway | 8080 | Reverse proxy, rate limiting, JWT validation |
| Auth Service | 8081 | User authentication, JWT generation |
| Ticket Service | 8082 | Event catalog, show/zone management |
| Booking Service | 8083 | Core booking logic, Redis Lua, Saga orchestration |
| Payment Service | 8084 | Payment processing, Stripe webhooks |
| Frontend Web | 3000 | Customer & organizer portal |

## Project Structure

```
booking-rush-10k-rps/
├── backend-api-gateway/     # API Gateway service
├── backend-auth/            # Authentication service
├── backend-booking/         # Booking service (critical path)
├── backend-ticket/          # Event catalog service
├── backend-payment/         # Payment service
├── frontend-web/            # Next.js frontend
├── pkg/                     # Shared Go packages
│   ├── config/              # Configuration loader
│   ├── database/            # PostgreSQL connection pool
│   ├── redis/               # Redis client + Lua support
│   ├── kafka/               # Redpanda producer
│   ├── middleware/          # JWT, idempotency, rate limit
│   ├── logger/              # Structured JSON logging
│   └── telemetry/           # OpenTelemetry setup
├── scripts/
│   ├── lua/                 # Redis Lua scripts
│   └── migrations/          # Database migrations
├── tests/
│   ├── integration/         # Go integration tests
│   └── load/                # k6 load tests
└── infra/
    ├── k8s/                 # Kubernetes manifests
    └── helm-values/         # Helm chart values
```

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 20+
- Docker & Docker Compose
- pnpm

### 1. Start Infrastructure

```bash
# Start PostgreSQL, Redis, Redpanda
make dev
```

### 2. Run Migrations

```bash
make migrate-all-up
```

### 3. Start Services

```bash
# Terminal 1: API Gateway
make run-gateway

# Terminal 2: Auth Service
make run-auth

# Terminal 3: Ticket Service
make run-ticket

# Terminal 4: Booking Service
make run-booking

# Terminal 5: Payment Service
make run-payment

# Terminal 6: Frontend
cd frontend-web && pnpm dev
```

Or use Docker Compose with hot reload:

```bash
make docker-dev
```

### 4. Seed Test Data

```bash
node scripts/01-seed-users.mjs
node scripts/02-seed-events.mjs
```

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=15m
REFRESH_TOKEN_EXPIRY=7d

# Stripe
STRIPE_API_KEY=sk_test_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
```

## API Endpoints

### Auth (`/api/v1/auth`)
```
POST   /register    - User registration
POST   /login       - User login
POST   /refresh     - Refresh token
GET    /me          - Get current user
```

### Events (`/api/v1/events`)
```
GET    /            - List events
GET    /:slug       - Get event by slug
POST   /            - Create event (organizer)
PUT    /:id         - Update event (organizer)
```

### Bookings (`/api/v1/bookings`)
```
POST   /reserve     - Reserve seats (idempotent)
POST   /:id/confirm - Confirm booking
POST   /:id/cancel  - Cancel booking
GET    /            - Get user bookings
```

### Payments (`/api/v1/payments`)
```
POST   /intent      - Create PaymentIntent
POST   /intent/confirm - Confirm payment
GET    /:id         - Get payment details
```

## Critical Path: Booking Flow

```
                    ┌─────────────┐
                    │   Client    │
                    └──────┬──────┘
                           │ POST /bookings/reserve
                           ▼
                    ┌─────────────┐
                    │ API Gateway │
                    └──────┬──────┘
                           │ Idempotency Check
                           ▼
                    ┌─────────────┐
                    │   Booking   │
                    │   Service   │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
   ┌───────────┐    ┌───────────┐    ┌───────────┐
   │   Redis   │    │   Kafka   │    │ PostgreSQL│
   │ Lua Script│    │   Event   │    │  Booking  │
   └───────────┘    └───────────┘    └───────────┘
```

### Redis Lua Scripts

- `reserve_seats.lua` - Atomic seat reservation with TTL
- `release_seats.lua` - Return seats to inventory
- `confirm_booking.lua` - Mark reservation as confirmed

## Zero Overselling: Multi-Layer Defense

```
Layer 1: Redis Lua Scripts     → Atomic check-and-deduct (primary guard)
    ↓
Layer 2: Saga Compensation     → Auto-rollback on payment failure
    ↓
Layer 3: PostgreSQL Constraint → CHECK (available_seats >= 0)
    ↓
Layer 4: Audit & Reconciliation → Redis-PostgreSQL sync every 5s
```

**Defense in depth**: If one layer fails, the next catches it. Result: 0% overselling rate.

## Virtual Queue System

For flash sales with 500,000 users competing for 50,000 seats:

```
┌─────────────┐    ZADD (timestamp)    ┌─────────────────┐
│    User     │ ──────────────────────▶│  Redis Sorted   │
│  Joins Queue│                        │      Set        │
└─────────────┘                        └────────┬────────┘
                                                │
                                       Worker releases batch
                                                │
                                                ▼
                                       ┌─────────────────┐
                                       │   Queue Pass    │
                                       │   JWT Token     │
                                       └────────┬────────┘
                                                │
                                                ▼
                                       ┌─────────────────┐
                                       │  Purchase with  │
                                       │   Queue Pass    │
                                       └─────────────────┘
```

- **FIFO ordering**: First come, first served (fair!)
- **Controlled release**: 500 users/second batch
- **5-minute expiry**: Queue pass auto-expires

## Load Testing

```bash
# Smoke test (1 VU)
make load-smoke

# Ramp-up test (0→1000 VUs)
make load-ramp

# Sustained load (5000 RPS)
make load-sustained

# Spike test (1000→10000 RPS)
make load-spike

# Full 10k RPS stress test
make load-10k
```

## Development

### Run Tests

```bash
# All tests
make test

# Unit tests only
make test-unit

# Integration tests
INTEGRATION_TEST=true make test-integration
```

### Code Quality

```bash
make lint    # Run golangci-lint
make fmt     # Format code
make tidy    # Tidy Go modules
```

### Build

```bash
make build           # Build all services
make build-gateway   # Build specific service
```

## Deployment

### Kubernetes

```bash
kubectl apply -f infra/k8s/namespace.yaml
kubectl apply -f infra/k8s/
```

### Docker Compose (Production)

```bash
docker-compose -f docker-compose.yml up -d
```

## Monitoring

| Tool | URL | Purpose |
|------|-----|---------|
| Grafana | :3000 | Dashboards |
| Tempo | :3200 | Distributed tracing |
| Prometheus | :9090 | Metrics |
| Redpanda Console | :8088 | Kafka UI |

### Health Checks

```bash
# API Gateway
curl http://localhost:8080/health

# Booking Service (with readiness)
curl http://localhost:8083/ready
```

## Business Rules

- Maximum **10 tickets** per user per event
- Reservation timeout: **10 minutes**
- Auto-release unpaid reservations
- Zone-based seating (not individual seats)

## Resilience Patterns

- **Idempotency**: Redis-backed idempotency keys
- **Rate Limiting**: Per-endpoint, distributed via Redis
- **Retry**: Exponential backoff with jitter
- **Graceful Shutdown**: 30-second timeout for in-flight requests
- **Saga Pattern**: Distributed transaction orchestration

## Documentation

- [Product Specification](.docs/01-spec.md)
- [Technical Architecture](.docs/02-plan.md)
- [Development Tasks](.docs/03-task.md)
- [Load Test Guide](tests/load/README.md)

## Technical Deep Dives

Detailed explanations of key architectural decisions:

| Topic | Description |
|-------|-------------|
| [Redis Lua Scripts](.interview/01-redis-lua-scripts.md) | Preventing race conditions at 10k RPS |
| [Saga Pattern](.interview/02-saga-pattern.md) | Distributed transaction orchestration |
| [Zero Overselling](.interview/03-zero-overselling.md) | Multi-layer defense architecture |
| [Idempotency Middleware](.interview/04-idempotency-middleware.md) | Safe retry handling |
| [Dirty Scenarios Testing](.interview/05-dirty-scenarios-testing.md) | Testing edge cases |
| [Transactional Outbox](.interview/06-transactional-outbox.md) | Reliable event publishing |
| [Virtual Queue](.interview/07-virtual-queue.md) | Redis Sorted Set for flash sales |
| [Tech Stack Decisions](.interview/08-tech-stack-decisions.md) | Why Go + NestJS |
| [OpenTelemetry Stack](.interview/09-opentelemetry-grafana-stack.md) | Observability setup |

## Patterns Used

- **Microservices** with database-per-service
- **Clean Architecture** (Handler → Service → Repository)
- **Saga Pattern** (Orchestration) for distributed transactions
- **Transactional Outbox** for reliable event publishing
- **CQRS** for read/write separation
- **Virtual Queue** for traffic control
- **Idempotency** for safe retries

## License

MIT
