# Technical Plan: Booking Rush (10k RPS)

> **Version:** 1.0
> **Last Updated:** 2025-12-07
> **Reference:** [01-spec.md](./01-spec.md) (Product Specification)

---

## Table of Contents
1. [System Architecture](#1-system-architecture)
2. [Data Model / Schema](#2-data-model--schema)
3. [API Definition](#3-api-definition)
4. [Component Structure](#4-component-structure)
5. [Third-party Integrations](#5-third-party-integrations)
6. [Security & Scalability](#6-security--scalability)
7. [Saga Pattern](#7-saga-pattern)
8. [Observability](#8-observability)
9. [Development Phases](#9-development-phases)

---

## 1. System Architecture

### 1.1 Architecture Overview
```
                                    +-------------------+
                                    |     Frontend      |
                                    |    (Next.js)      |
                                    +---------+---------+
                                              |
                                    +---------v---------+
                                    |   API Gateway     |
                                    |   (Go + Gin)      |
                                    |   Rate Limit      |
                                    +---------+---------+
                                              |
        +----------------+----------------+---+---+----------------+
        |                |                |       |                |
+-------v------+ +-------v------+ +-------v-------+ +-------v------+
| Auth Service | |Ticket Service| | Booking Svc   | |Payment Service|
|  (Go + Gin)  | |  (Go + Gin)  | | (Go+Gin) [*]  | |  (Go + Gin)  |
+-------+------+ +-------+------+ +-------+-------+ +-------+------+
        |                |                |                |
        |                |        +-------v-------+        |
        |                |        |  Redis (Lua)  |        |
        |                |        |  Atomic Lock  |        |
        |                |        +---------------+        |
        |                |                |                |
        |                |        +-------v-------+        |
        |                |        |   Redpanda    |<-------+
        |                |        | (Kafka-compat)|
        |                |        +-------+-------+
        |                |                |
        |                |     +----------+----------+
        |                |     |          |          |
        |                |     v          v          v
        |                | +--------+ +--------+ +--------+
        |                | |Notific.| |Analytics| |Inventory|
        |                | |Service | | Service | |  Sync   |
        |                | |(NestJS)| | (NestJS)| | Worker  |
        |                | +---+----+ +----+----+ +--------+
        |                |     |           |
        |                |     v           v
        |                | +-------------------+
        |                | |      MongoDB      |
        |                | | (Notifications &  |
        |                | |    Analytics)     |
        |                | +-------------------+
        |                |         |
        +----------------+---------+--------------------------+
                                   |
                         +---------v---------+
                         |    PostgreSQL     |
                         |   (Primary DB)    |
                         +-------------------+
```

### 1.2 Architecture Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Critical Path (Go + Gin)** | API Gateway, Auth, Ticket, Booking, Payment - require high performance |
| **Non-Critical Path (NestJS)** | Notification, Analytics - async workers, don't need low latency |
| **Redpanda** | Kafka-compatible but faster, uses less resources |
| **Redis Lua** | Atomic operations for seat reservation |

### 1.3 Technology Stack

| Layer | Technology | Justification |
|-------|------------|---------------|
| **Frontend** | Next.js 15, TailwindCSS, Shadcn UI | Modern, SSR support, great DX |
| **API Gateway** | Go + Gin | High performance, built-in middleware |
| **Backend (Critical)** | Go + Gin | Fast, efficient concurrency (goroutines), 10k RPS target |
| **Backend (Non-Critical)** | NestJS (TypeScript) | Great DX, decorator-based, good for async workers |
| **Database (Primary)** | PostgreSQL 16 | ACID, proven reliability, good for transactions |
| **Database (Analytics)** | MongoDB 7 | Flexible schema, good for logs/analytics |
| **Cache/Lock** | Redis 7 + Lua | Atomic operations, sub-ms latency |
| **Message Queue** | Redpanda | Kafka-compatible, faster, less resource usage |
| **Container** | Docker, Docker Compose | Easy local development |
| **Monitoring** | Prometheus, Grafana, Tempo | Industry standard observability |

### 1.4 Service Responsibilities

| Service | Language | Responsibility |
|---------|----------|----------------|
| API Gateway | Go | Rate limiting, routing, JWT validation |
| Auth Service | Go | User auth, JWT generation |
| Ticket Service | Go | Event & seat catalog (read-heavy, cached) |
| Booking Service [*] | Go | Core booking logic, Redis Lua, Saga orchestration |
| Payment Service | Go | Payment processing, Kafka consumer |
| Notification Service | NestJS | Email notifications, Kafka consumer |
| Analytics Service | NestJS | Dashboard data, aggregations |

---

## 2. Data Model / Schema

### 2.1 Entity Relationship Diagram
```
+-------------+     +-------------+     +-------------+
|   tenants   |---->|    users    |     |  categories |
+-------------+     +------+------+     +------+------+
                           |                   |
                    +------+------+            |
                    |             |            |
              +-----v-----+ +-----v-----+ +----v------+
              |  bookings | |  events   |-|           |
              +-----+-----+ +-----+-----+ +-----------+
                    |             |
              +-----v-----+ +-----v-----+
              | payments  | |   shows   |
              +-----------+ +-----+-----+
                                  |
                            +-----v-----+
                            |seat_zones |
                            +-----------+
```

### 2.2 PostgreSQL Tables

#### tenants
```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    settings JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### users
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    role VARCHAR(20) DEFAULT 'user',  -- user, organizer, admin
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant ON users(tenant_id);
```

#### categories
```sql
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### events
```sql
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    organizer_id UUID REFERENCES users(id),
    category_id UUID REFERENCES categories(id),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    venue VARCHAR(255),
    image_url VARCHAR(500),
    status VARCHAR(20) DEFAULT 'draft',  -- draft, published, cancelled
    sale_start_at TIMESTAMPTZ,
    max_per_user INT DEFAULT 4,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_slug ON events(slug);
CREATE INDEX idx_events_sale_start ON events(sale_start_at);
```

#### shows
```sql
CREATE TABLE shows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    show_date TIMESTAMPTZ NOT NULL,
    total_seats INT NOT NULL,
    available_seats INT NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_shows_event ON shows(event_id);
CREATE INDEX idx_shows_date ON shows(show_date);
```

#### seat_zones
```sql
CREATE TABLE seat_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    show_id UUID REFERENCES shows(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    total_seats INT NOT NULL,
    available_seats INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_zones_show ON seat_zones(show_id);
```

#### bookings
```sql
CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id VARCHAR(100) UNIQUE NOT NULL,  -- idempotency key
    user_id UUID REFERENCES users(id),
    show_id UUID REFERENCES shows(id),
    zone_id UUID REFERENCES seat_zones(id),
    quantity INT NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'reserved',  -- reserved, confirmed, cancelled, expired
    expires_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_bookings_user ON bookings(user_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_expires ON bookings(expires_at) WHERE status = 'reserved';
```

#### payments
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID REFERENCES bookings(id),
    amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',  -- pending, processing, success, failed
    payment_method VARCHAR(50),
    transaction_id VARCHAR(255),
    failed_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_payments_booking ON payments(booking_id);
CREATE INDEX idx_payments_status ON payments(status);
```

#### audit_logs (Partitioned)
```sql
CREATE TABLE audit_logs (
    id UUID DEFAULT gen_random_uuid(),
    user_id UUID,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions
CREATE TABLE audit_logs_2025_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

#### outbox (Transactional Outbox Pattern)
```sql
CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(100) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_outbox_unprocessed ON outbox(created_at) WHERE processed = FALSE;
```

### 2.3 Redis Data Structures

| Key Pattern | Type | Description | TTL |
|-------------|------|-------------|-----|
| `show:{id}:zone:{id}:available` | String | Available seat count | - |
| `reservation:{id}` | Hash | Reservation details | 10 min |
| `user:{id}:event:{id}:reserved` | String | User's reserved count | 10 min |
| `event:{id}` | Hash | Event cache | 5 min |
| `rate_limit:user:{id}` | String | Rate limit counter | 1 min |
| `queue:event:{id}` | Sorted Set | Virtual queue (score = timestamp) | - |
| `queue_pass:{token}` | String | Queue pass token | 5 min |

### 2.4 MongoDB Collections

#### notifications
```javascript
{
  _id: ObjectId,
  userId: UUID,
  type: "email" | "sms" | "push",
  channel: "booking_confirmation" | "payment_success" | "reminder",
  recipient: string,
  subject: string,
  body: string,
  status: "pending" | "sent" | "failed",
  attempts: number,
  sentAt: Date,
  createdAt: Date
}
```

#### analytics_daily
```javascript
{
  _id: ObjectId,
  date: Date,
  eventId: UUID,
  metrics: {
    totalBookings: number,
    confirmedBookings: number,
    totalRevenue: number,
    conversionRate: number
  },
  hourlyBreakdown: [
    { hour: 0, bookings: 10, revenue: 5000 }
  ]
}
```

---

## 3. API Definition

### 3.1 API Standards
- **Base URL:** `/api/v1`
- **Authentication:** Bearer JWT in `Authorization` header
- **Content-Type:** `application/json`
- **Request ID:** `X-Request-ID` header for tracing

### 3.2 Response Format

**Success:**
```json
{
  "success": true,
  "data": { ... },
  "meta": { "page": 1, "per_page": 20, "total": 100 }
}
```

**Error:**
```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Not enough seats available",
    "details": { "requested": 5, "available": 2 }
  }
}
```

### 3.3 Endpoints

#### Auth Service
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/auth/register` | Register new user | - |
| POST | `/auth/login` | Login, get tokens | - |
| POST | `/auth/refresh` | Refresh access token | - |
| POST | `/auth/logout` | Invalidate refresh token | Yes |
| POST | `/auth/forgot-password` | Request password reset | - |
| POST | `/auth/reset-password` | Reset password | - |
| GET | `/auth/me` | Get current user | Yes |
| PUT | `/auth/me` | Update profile | Yes |

#### Ticket Service
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/events` | List events (paginated) | - |
| GET | `/events/:slug` | Get event by slug | - |
| GET | `/events/:slug/shows` | Get shows for event | - |
| GET | `/shows/:id/zones` | Get zones & availability | - |
| POST | `/events` | Create event | Organizer |
| PUT | `/events/:id` | Update event | Organizer |
| DELETE | `/events/:id` | Delete event | Organizer |

#### Booking Service
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/bookings/reserve` | Reserve seats | Yes |
| POST | `/bookings/:id/confirm` | Confirm booking (after payment) | Yes |
| POST | `/bookings/:id/cancel` | Cancel booking | Yes |
| GET | `/bookings` | Get user's bookings | Yes |
| GET | `/bookings/:id` | Get booking details | Yes |
| GET | `/bookings/pending` | Get pending reservations | Yes |
| POST | `/queue/join` | Join virtual queue | Yes |
| GET | `/queue/status` | Get queue position | Yes |

#### Payment Service
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/payments` | Process payment | Yes |
| GET | `/payments/:id` | Get payment status | Yes |
| POST | `/payments/:id/refund` | Request refund | Yes |

### 3.4 Error Codes
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Invalid or missing token |
| `FORBIDDEN` | 403 | No permission |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid input |
| `INSUFFICIENT_STOCK` | 409 | Not enough seats |
| `MAX_PER_USER_EXCEEDED` | 409 | User limit reached |
| `RESERVATION_EXPIRED` | 410 | Reservation timed out |
| `PAYMENT_FAILED` | 402 | Payment processing failed |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

---

## 4. Component Structure

### 4.1 Project Structure (Monorepo)
```
booking-rush-10k-rps/
├── backend-
│   ├── api-gateway/          # Go - Entry point, routing, rate limiting
│   │   ├── cmd/
│   │   ├── internal/
│   │   │   ├── handler/
│   │   │   ├── middleware/
│   │   │   └── proxy/
│   │   ├── go.mod
│   │   └── main.go
│   ├── auth-service/         # Go - User auth, JWT
│   │   ├── cmd/
│   │   ├── internal/
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   ├── repository/
│   │   │   └── domain/
│   │   └── go.mod
│   ├── ticket-service/       # Go - Event & seat management
│   ├── booking-service/      # Go - Core booking logic (10k RPS)
│   ├── payment-service/      # Go - Payment processing
│   ├── notification-service/ # NestJS - Email, SMS, Push
│   │   ├── src/
│   │   │   ├── modules/
│   │   │   │   ├── kafka/
│   │   │   │   ├── email/
│   │   │   │   └── template/
│   │   │   └── main.ts
│   │   └── package.json
│   ├── analytics-service/    # NestJS - Reports, dashboard data
│   └── web/                  # Next.js app
│       ├── app/
│       ├── components/
│       └── package.json
├── pkg/                      # Shared Go packages
│   ├── config/               # Configuration loader (Viper)
│   ├── logger/               # Structured logging (Zap)
│   ├── middleware/           # Common middlewares
│   ├── response/             # Standard API responses
│   ├── errors/               # Custom error types
│   ├── kafka/                # Redpanda/Kafka wrapper
│   ├── redis/                # Redis client wrapper
│   ├── saga/                 # Saga orchestration
│   ├── database/             # DB connection & migrations
│   └── telemetry/            # OpenTelemetry setup
├── libs/                     # Shared TypeScript packages (for NestJS)
│   ├── common/               # Shared DTOs, interfaces
│   └── kafka/                # Kafka consumer/producer wrapper
├── scripts/
│   ├── lua/                  # Redis Lua scripts
│   │   ├── reserve_seats.lua
│   │   ├── release_seats.lua
│   │   └── confirm_booking.lua
│   └── migrations/           # DB migrations
├── infra/
│   ├── prometheus/
│   ├── grafana/
│   ├── otel/
│   └── loki/
├── tests/
│   ├── load/                 # k6 load tests
│   └── e2e/                  # End-to-end tests
├── go.work                   # Go workspace
├── pnpm-workspace.yaml       # pnpm workspace config
├── docker-compose.yml
├── Makefile
└── README.md
```

### 4.2 Go Service Structure (Clean Architecture)
```
service/
├── cmd/
│   └── main.go              # Entry point
├── internal/
│   ├── handler/             # HTTP handlers (Controllers)
│   │   └── booking_handler.go
│   ├── service/             # Business logic (Use Cases)
│   │   └── booking_service.go
│   ├── repository/          # Data access (Gateways)
│   │   ├── booking_repo.go
│   │   └── redis_repo.go
│   ├── domain/              # Entities
│   │   └── booking.go
│   └── dto/                 # Data transfer objects
│       ├── request.go
│       └── response.go
├── go.mod
└── go.sum
```

---

## 5. Third-party Integrations

### 5.1 Go Libraries

| Category | Library | Purpose |
|----------|---------|---------|
| Web Framework | `gin-gonic/gin` | HTTP routing |
| Database | `jackc/pgx/v5` | PostgreSQL driver |
| Redis | `redis/go-redis/v9` | Redis client |
| Kafka | `segmentio/kafka-go` | Redpanda/Kafka client |
| JWT | `golang-jwt/jwt/v5` | JWT handling |
| Config | `spf13/viper` | Configuration |
| Logging | `uber-go/zap` | Structured logging |
| Validation | `go-playground/validator/v10` | Input validation |
| Migration | `golang-migrate/migrate/v4` | DB migrations |
| UUID | `google/uuid` | UUID generation |
| OTel | `go.opentelemetry.io/otel` | Observability |

### 5.2 NestJS Libraries

| Category | Library | Purpose |
|----------|---------|---------|
| Kafka | `@nestjs/microservices` + `kafkajs` | Kafka consumer |
| MongoDB | `@nestjs/mongoose` | MongoDB ODM |
| Email | `@nestjs-modules/mailer` | Email sending |
| Validation | `class-validator` | Input validation |
| Config | `@nestjs/config` | Configuration |

### 5.3 Frontend Libraries

| Category | Library | Purpose |
|----------|---------|---------|
| Framework | Next.js 15 | React framework |
| Styling | TailwindCSS | Utility CSS |
| UI Components | Shadcn UI | Component library |
| State | Zustand | State management |
| HTTP | Axios | API client |
| Forms | React Hook Form | Form handling |
| Validation | Zod | Schema validation |

### 5.4 Infrastructure

| Service | Technology | Purpose |
|---------|------------|---------|
| PostgreSQL | PostgreSQL 16 | Primary database |
| MongoDB | MongoDB 7 | Analytics/Notifications |
| Redis | Redis 7 | Cache, Locks, Queue |
| Message Queue | Redpanda | Event streaming |
| Tracing | Tempo | Distributed tracing |
| Metrics | Prometheus | Metrics collection |
| Logs | Loki | Log aggregation |
| Visualization | Grafana | Dashboards |
| OTel Collector | OpenTelemetry | Telemetry pipeline |

---

## 6. Security & Scalability

### 6.1 Security Measures

#### Authentication & Authorization
- JWT-based authentication (access + refresh tokens)
- Role-Based Access Control (RBAC): user, organizer, admin
- Token refresh rotation
- Password hashing with bcrypt (cost 12)

#### Data Protection
- All data encrypted at rest (PostgreSQL encryption)
- All traffic over HTTPS/TLS 1.3
- Sensitive data masking in logs
- PII encryption in database

#### Input Validation
- Request payload validation (validator/v10)
- SQL injection prevention (parameterized queries)
- XSS prevention (output encoding)
- CSRF protection (SameSite cookies)

#### Rate Limiting (Token Bucket + Burst)
| Endpoint | Rate | Burst | Window |
|----------|------|-------|--------|
| `/auth/login` | 5 req | 3 | 1 min |
| `/auth/register` | 3 req | 2 | 1 min |
| `/bookings/reserve` | 20 req | 10 | 1 min |
| General API | 100 req | 20 | 1 min |

#### Security Headers
```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
```

### 6.2 Scalability

#### Scaling Triggers
| Service | Type | Scale Trigger |
|---------|------|---------------|
| API Gateway | API | CPU > 70% |
| Auth Service | API | CPU > 70% |
| Ticket Service | API | CPU > 70% |
| **Booking Service** | API | CPU > 60% OR req/s > 5000 |
| **Payment Service** | Worker | Kafka Lag > 1000 |
| **Notification Service** | Worker | Kafka Lag > 5000 |

#### Scaling Constraints
- Min replicas: 2 (HA)
- Max replicas: 10 (cost control)
- Cool-down period: 60 seconds

#### High Availability
- Database read replicas for read-heavy operations
- Redis: Start single node, migration path to cluster documented
- Graceful degradation when dependent services fail
- Circuit breaker pattern for external services

### 6.3 Performance Targets

| Metric | Target |
|--------|--------|
| Throughput | >= 10,000 RPS (booking endpoint) |
| Latency P50 (Server) | < 20ms |
| Latency P99 (Server) | < 50ms |
| Error Rate | < 0.1% |
| Availability | 99.9% |

---

### 7.1 Overview
**Saga Pattern** is used for managing the post-payment confirmation workflow to ensure consistency between Payment and Booking services.

**Architecture Decision:**
We use a **Hybrid Approach** instead of a full lifecycle Saga to handle the 10k RPS requirement:

1.  **Reservation Phase (Fast Path):**
    -   Uses Redis Lua Scripts for atomic locking.
    -   Synchronous, extremely low latency.
    -   No Saga overhead here to prevent bottlenecks during high traffic bursts.

2.  **Payment Phase (Client-Side):**
    -   User completes payment directly with Stripe.
    -   Booking has a TTL (e.g., 10 mins) in Redis.

3.  **Confirmation Phase (Saga - Post-Payment):**
    -   Triggered asynchronously via Webhook -> Kafka (`payment.success`).
    -   Uses **Orchestration-based Saga** to finalize the booking.
    -   Ensures "Eventual Consistency" between Payment (Paid) and Booking (Confirmed).

### 7.2 Post-Payment Saga Flow

```
User Action:      [Pay at Stripe] --> [Webhook] --> [Payment Service]
                                                          |
                                                          | (Produce 'payment.success')
                                                          v
                                                  +----------------+
                                                  |  Kafka Topic   |
                                                  +-------+--------+
                                                          |
+=========================================================|=================+
|                   BOOKING SAGA (Orchestrator)           v                 |
+===========================================================================+
|                                                                           |
|   Step 1: Confirm Booking                                                 |
|  +---------------------------+                                            |
|  | Booking Service           |                                            |
|  | - Update DB: 'CONFIRMED'  |                                            |
|  | - Redis: Remove TTL       |                                            |
|  |   (Make Permanent)        |                                            |
|  +-------------+-------------+                                            |
|                | [Success]                                                |
|                v                                                          |
|                                                                           |
|   Step 2: Send Notification                                               |
|  +---------------------------+                                            |
|  | Notification Service      |                                            |
|  | - Send Email / SMS        |                                            |
|  +---------------------------+                                            |
|                                                                           |
+===========================================================================+
```

### 7.3 Saga Steps Definition

**Saga Name:** `post-payment-saga`

| Step | Action | Service | Compensating Action | Note |
|------|--------|---------|---------------------|------|
| 1 | `confirm-booking` | Booking Service | `release-seats` * | This makes the booking permanent. If it fails, we must refund & release. |
| 2 | `send-notification` | Notification Service | - | Non-critical. Failure does NOT trigger rollback. |

*Note: In the current implementation, `confirm-booking` is designed to be retried until success. Compensation (Refunding) happens only if the booking was already expired/released by the time payment arrived (race condition).*

### 7.4 Kafka Events for Saga

**Trigger Event:**
| Topic | Source | Description |
|-------|--------|-------------|
| `payment.success` | Payment Service | Webhook received, payment charged successfully. Starts the Saga. |

**Internal Saga Commands:**
| Topic | Description |
|-------|-------------|
| `saga.booking.confirm` | Orchestrator tells Booking Service to finalize the record. |
| `saga.notification.send` | Orchestrator tells Notification Service to send email. |

### 7.5 Saga Timeout & Retry

| Config | Value | Reason |
|--------|-------|--------|
| Step Timeout | 30 seconds | Fast fail to retry quickly. |
| Saga Timeout | 1 minute | Post-payment processing should be fast. |
| Max Retries | 3 | Handle transient network glitches. |

---

## 8. Observability

### 8.1 OpenTelemetry Stack

```
+-------------------------------------------------------------+
|                      Go Services                             |
|  +-------------+ +-------------+ +-------------+             |
|  |API Gateway  | |Booking Svc  | |Payment Svc  |  ...        |
|  +------+------+ +------+------+ +------+------+             |
|         |               |               |                    |
|         +---------------+---------------+                    |
|                         |                                    |
|              +----------v----------+                         |
|              |    OTel Go SDK      |                         |
|              | (Traces+Metrics+Logs)|                        |
|              +----------+----------+                         |
+-------------------------------------------------------------+
                          | OTLP (gRPC/HTTP)
                          v
              +-----------------------+
              |    OTel Collector     |
              +-----------+-----------+
                          |
        +-----------------+-----------------+
        |                 |                 |
        v                 v                 v
+---------------+ +---------------+ +---------------+
|    Tempo      | |  Prometheus   | |     Loki      |
|   (Traces)    | |  (Metrics)    | |    (Logs)     |
+-------+-------+ +-------+-------+ +-------+-------+
        |                 |                 |
        +-----------------+-----------------+
                          |
                          v
              +-----------------------+
              |       Grafana         |
              |  (Unified Dashboard)  |
              +-----------------------+
```

### 8.2 Key Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `http_requests_total` | Counter | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | Request latency |
| `booking_reservations_total` | Counter | Total reservations |
| `booking_reservation_failures` | Counter | Failed reservations |
| `active_reservations` | Gauge | Current active reservations |
| `kafka_consumer_lag` | Gauge | Consumer lag |

### 8.3 Alerting Rules

| Alert | Condition | Severity |
|-------|-----------|----------|
| High Error Rate | > 1% errors in 5 min | Critical |
| High Latency | P99 > 500ms for 5 min | Warning |
| Service Down | Health check fails 3x | Critical |
| Kafka Lag | Lag > 10,000 messages | Warning |

---

## 9. Development Phases

| Phase | Focus | Key Deliverable |
|-------|-------|-----------------|
| 1 | **Foundation** | Monorepo, Docker, shared packages, migrations, **Basic OTel** |
| 2 | **Core Booking** [*] | Redis Lua, 10k RPS achieved, zero overselling, **Thundering Herd Handling** |
| 3 | **Auth & Events** | JWT, Rate Limit (Token Bucket), CRUD |
| 4 | **Payment & Saga** | Kafka consumer, Idempotency, Saga Pattern |
| 5 | **NestJS Services** | Notification + Analytics (NestJS + MongoDB) |
| 6 | **Virtual Queue** | Queue + Bypass Token, Audit Log |
| 7 | **Frontend** | Next.js, Booking flow, Queue UI |
| 8 | **Observability** | Full OTel stack, Grafana dashboards, Alerting |
| 9 | **Production** | Security audit, E2E load test, Deploy |

### Early Observability (Phase 1-2)
> **Important:** Basic tracing/metrics should be implemented early (Phase 1) to enable debugging of:
> - Race conditions in Phase 2 (Core Booking)
> - Performance bottlenecks during 10k RPS optimization
> - Saga flow issues in Phase 4
>
> Phase 8 focuses on *production-grade* observability with dashboards, alerting, and log-trace correlation.

### Thundering Herd Handling (Phase 2-3)
> **Note:** Before Virtual Queue is ready (Phase 6), the system must handle "thundering herd" scenarios:
> - API Gateway rate limiting provides first line of defense
> - Booking Service must efficiently reject over-capacity requests
> - Redis Lua scripts return immediately on insufficient stock
> - Client-side should handle 429 responses gracefully with retry-after

### Milestones
- **Phase 1:** Basic OTel tracing operational for debugging
- **Phase 2:** Achieve 10,000 RPS with zero overselling
- **Phase 4:** Complete booking-to-payment flow with Saga
- **Phase 5:** NestJS services operational with MongoDB
- **Phase 8:** Full observability with log-trace correlation, dashboards, alerting
- **Phase 9:** System deployed and running in production

---

## Appendix

### A. Decision Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-12-03 | Use Gin over Fiber | Larger ecosystem, more stable |
| 2025-12-03 | Start with Redis single node | Simpler, can migrate to cluster later |
| 2025-12-06 | Use Redpanda instead of Kafka | Kafka-compatible, faster, less resource |
| 2025-12-06 | Use NestJS for non-critical services | Good for learning, async workers |
| 2025-12-06 | Use Orchestration-based Saga | Better visibility over complex flows |
| 2025-12-07 | Split spec into Product + Technical | Better separation of concerns |

### B. References
- [Product Specification](./01-spec.md)
- [Development Tasks](./03-task.md)
- [Redis Lua Scripting](https://redis.io/docs/manual/programmability/eval-intro/)
- [Saga Pattern](https://microservices.io/patterns/data/saga.html)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
