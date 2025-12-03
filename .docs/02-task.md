# Development Roadmap

> **Reference:** [01-spec.md](./01-spec.md)
> **Last Updated:** 2025-12-03

---

## Phase 1: Foundation & Infrastructure
**Goal:** Setup project skeleton and infrastructure

### 1.1 Monorepo Setup
- [ ] Create directory structure (`apps/`, `pkg/`, `scripts/`, `tests/`)
- [ ] Initialize Go Workspaces (`go.work`)
- [ ] Setup each service module (`go mod init`)
- [ ] Create `Makefile` with common commands

### 1.2 Docker Infrastructure
- [ ] Create `docker-compose.yml` with:
  - [ ] PostgreSQL 16 (with healthcheck)
  - [ ] Redis 7 (Alpine)
  - [ ] Kafka + Zookeeper (Bitnami)
  - [ ] Kafka UI (for monitoring)
  - [ ] Redis Commander (for monitoring)
- [ ] Create `.env.example` for configuration
- [ ] Add Docker network configuration

### 1.3 Shared Packages (`pkg/`)
- [ ] `pkg/config` - Configuration loader (Viper)
- [ ] `pkg/logger` - Structured JSON logging (Zap/Zerolog)
- [ ] `pkg/response` - Standard API response wrapper
- [ ] `pkg/errors` - Custom error types with codes
- [ ] `pkg/middleware` - Common middlewares (RequestID, Recovery)
- [ ] `pkg/database` - PostgreSQL connection pool (pgx)
- [ ] `pkg/redis` - Redis client wrapper
- [ ] `pkg/kafka` - Kafka producer/consumer wrapper (segmentio/kafka-go)

### 1.4 Database Setup
- [ ] Create migration tool setup (golang-migrate)
- [ ] Write initial migrations:
  - [ ] `tenants` table
  - [ ] `users` table
  - [ ] `categories` table
  - [ ] `events` table
  - [ ] `shows` table
  - [ ] `seat_zones` table
  - [ ] `bookings` table
  - [ ] `payments` table
  - [ ] `audit_logs` table (partitioned)

### 1.5 API Gateway (Basic)
- [ ] Setup Gin router
- [ ] Implement health check endpoint (`/health`)
- [ ] Implement readiness endpoint (`/ready`)
- [ ] Add request ID middleware
- [ ] Add logging middleware
- [ ] Add CORS middleware
- [ ] Basic reverse proxy to services

**Milestone:** `docker-compose up` runs all infrastructure, API Gateway responds to health checks

---

## Phase 2: Core Booking Engine ⭐
**Goal:** Achieve 10,000 RPS on booking endpoint

### 2.1 Redis Lua Scripts
- [ ] Write `reserve_seats.lua` script:
  - [ ] Check seat availability
  - [ ] Check user max limit per event
  - [ ] Atomic deduction
  - [ ] Set reservation TTL
- [ ] Write `release_seats.lua` script
- [ ] Write `confirm_booking.lua` script
- [ ] Unit test Lua scripts

### 2.2 Booking Service
- [ ] Create service structure (Clean Architecture)
  - [ ] `internal/handler` - HTTP handlers
  - [ ] `internal/service` - Business logic
  - [ ] `internal/repository` - Data access
  - [ ] `internal/domain` - Entities
- [ ] Implement endpoints:
  - [ ] `POST /bookings/reserve` - Reserve seats (Redis Lua)
  - [ ] `POST /bookings/:id/confirm` - Confirm booking
  - [ ] `POST /bookings/:id/cancel` - Cancel booking
  - [ ] `GET /bookings` - List user bookings
  - [ ] `GET /bookings/:id` - Get booking details
  - [ ] `GET /bookings/pending` - Get pending reservations
- [ ] Implement Kafka producer (booking events)
- [ ] Add idempotency (reservation_id)

### 2.3 Inventory Sync Worker
- [ ] Create Kafka consumer for booking events
- [ ] Implement batch update to PostgreSQL (every 5 seconds)
- [ ] Handle Redis-SQL reconciliation

### 2.4 Load Testing
- [ ] Setup k6 load testing tool
- [ ] Write load test script for `/bookings/reserve`
- [ ] Run load test and optimize:
  - [ ] Target: 10,000 RPS
  - [ ] P99 Latency: < 50ms (server)
  - [ ] Error rate: < 0.1%
- [ ] Document performance results

**Milestone:** Achieve 10k RPS with zero overselling on booking endpoint

---

## Phase 3: Authentication & Events
**Goal:** Complete auth flow and event management

### 3.1 Auth Service
- [ ] Implement user registration
  - [ ] Email validation
  - [ ] Password hashing (bcrypt, cost 12)
- [ ] Implement login
  - [ ] JWT generation (access + refresh token)
  - [ ] Token expiry (15 min / 7 days)
- [ ] Implement token refresh
- [ ] Implement logout (invalidate refresh token)
- [ ] Implement password reset flow
- [ ] Add `GET /auth/me` endpoint
- [ ] Add `PUT /auth/me` endpoint (update profile)

### 3.2 JWT Middleware
- [ ] Create JWT validation middleware
- [ ] Inject user context into request
- [ ] Handle expired tokens
- [ ] Add to API Gateway

### 3.3 Ticket Service
- [ ] Implement Event CRUD:
  - [ ] `POST /events` (Organizer only)
  - [ ] `GET /events` (public, paginated)
  - [ ] `GET /events/:id` (public)
  - [ ] `PUT /events/:id` (Organizer only)
  - [ ] `DELETE /events/:id` (Organizer only)
- [ ] Implement Show management:
  - [ ] `GET /events/:id/shows`
  - [ ] `POST /events/:id/shows` (Organizer)
- [ ] Implement Zone/Seat management:
  - [ ] `GET /shows/:id/zones`
  - [ ] `POST /shows/:id/zones` (Organizer)
- [ ] Add Redis caching (TTL: 5 min)
- [ ] Cache invalidation on update

### 3.4 Rate Limiting
- [ ] Implement Token Bucket algorithm with burst
- [ ] Configure limits per endpoint:
  - [ ] `/auth/login`: 5 req/min, burst 3
  - [ ] `/auth/register`: 3 req/min, burst 2
  - [ ] `/bookings/reserve`: 20 req/min, burst 10
  - [ ] General: 100 req/min, burst 20
- [ ] Store rate limit state in Redis
- [ ] Add `X-RateLimit-*` headers

**Milestone:** Users can register, login, browse events, and book tickets

---

## Phase 4: Payment & Consistency
**Goal:** Complete payment flow with data consistency

### 4.1 Payment Service
- [ ] Create Kafka consumer for `booking.created` events
- [ ] Implement payment processing (mock gateway)
- [ ] Implement payment states:
  - [ ] PENDING → PROCESSING → SUCCESS/FAILED
  - [ ] FAILED → RETRY (with backoff)
  - [ ] TIMEOUT → REFUND
- [ ] Produce payment events:
  - [ ] `payment.success`
  - [ ] `payment.failed`
- [ ] Update booking status in DB

### 4.2 Idempotency
- [ ] Add idempotency key to all write operations
- [ ] Store idempotency records in Redis (24h TTL)
- [ ] Return cached response for duplicate requests

### 4.3 Transactional Outbox
- [ ] Create `outbox` table in PostgreSQL
- [ ] Write booking + outbox in same transaction
- [ ] Create outbox poller (publish to Kafka)
- [ ] Mark messages as processed

### 4.4 Retry Logic
- [ ] Implement exponential backoff for failures
- [ ] Add jitter to prevent thundering herd
- [ ] Configure max retry attempts
- [ ] Dead letter queue for failed messages

### 4.5 Reservation Expiry
- [ ] Create reservation expiry worker
- [ ] Scan expired reservations (Redis keyspace notification or cron)
- [ ] Release seats back to inventory
- [ ] Update booking status to `expired`
- [ ] Produce `booking.expired` event

**Milestone:** Complete booking-to-payment flow with consistency guarantees

---

## Phase 5: Advanced Features
**Goal:** Production-level features

### 5.1 Virtual Queue / Waiting Room
- [ ] Implement queue join endpoint (`POST /queue/join`)
- [ ] Use Redis Sorted Set for queue
- [ ] Calculate estimated wait time
- [ ] Implement queue status endpoint (`GET /queue/status`)
- [ ] Generate Queue Pass Token (JWT) when position = 0
- [ ] API Gateway: validate `X-Queue-Pass` header
- [ ] Bypass rate limit for users with valid queue pass

### 5.2 Notification Service
- [ ] Create Kafka consumers for events
- [ ] Implement email notifications:
  - [ ] Booking confirmation
  - [ ] Payment success/failed
  - [ ] Event reminder
  - [ ] Refund processed
- [ ] Email template system
- [ ] Rate limiting for notifications

### 5.3 Audit Logging
- [ ] Create audit log middleware
- [ ] Log all important actions:
  - [ ] User login/logout
  - [ ] Booking CRUD
  - [ ] Payment actions
  - [ ] Admin actions
- [ ] Store in partitioned table
- [ ] Add user agent and IP address

### 5.4 Refund Flow
- [ ] Implement refund request endpoint
- [ ] Apply refund rules (100%/50%/0% based on days)
- [ ] Admin approval workflow
- [ ] Process refund (mock)
- [ ] Update booking status to `refunded`

### 5.5 Multi-Tenant Support
- [ ] Add tenant_id to all relevant tables
- [ ] Implement tenant isolation in queries
- [ ] Tenant-specific configuration
- [ ] Tenant branding (logo, colors)

**Milestone:** Virtual queue working, notifications sent, audit trail complete

---

## Phase 6: Frontend
**Goal:** User-facing web application

### 6.1 Setup
- [ ] Initialize Next.js 15 project (App Router)
- [ ] Setup TailwindCSS
- [ ] Install and configure Shadcn UI
- [ ] Create base layout
- [ ] Setup API client (axios/fetch)
- [ ] Add environment configuration

### 6.2 Public Pages
- [ ] Landing page (hero, featured events)
- [ ] Event list page (with filters)
- [ ] Event detail page
- [ ] Show selection page
- [ ] Seat/Zone selection page

### 6.3 Auth Pages
- [ ] Login page
- [ ] Register page
- [ ] Forgot password page
- [ ] Reset password page
- [ ] Auth state management

### 6.4 Booking Flow
- [ ] Seat selection UI
- [ ] Booking summary page
- [ ] Payment page (mock)
- [ ] Confirmation page
- [ ] E-ticket display

### 6.5 Virtual Queue UI
- [ ] Queue waiting room page
- [ ] Position indicator
- [ ] Estimated wait time
- [ ] Auto-redirect when position = 0

### 6.6 User Dashboard
- [ ] Booking history
- [ ] Pending bookings (resume payment)
- [ ] Profile settings
- [ ] Ticket management

**Milestone:** Users can browse, book, and pay through web UI

---

## Phase 7: Observability (OpenTelemetry)
**Goal:** Production-grade monitoring with unified OTel stack

### 7.1 OpenTelemetry Setup
- [ ] Create `pkg/telemetry` shared package:
  - [ ] OTel SDK initialization
  - [ ] OTLP exporter configuration
  - [ ] Tracer provider setup
  - [ ] Meter provider setup
  - [ ] Resource attributes (service.name, service.version)
- [ ] Add OTel dependencies to all services:
  ```
  go.opentelemetry.io/otel
  go.opentelemetry.io/otel/sdk
  go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
  go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
  ```

### 7.2 Auto-Instrumentation
- [ ] Add Gin middleware: `otelgin`
- [ ] Add Redis instrumentation: `otelredis`
- [ ] Add PostgreSQL instrumentation: `otelsql`
- [ ] Add HTTP client instrumentation: `otelhttp`
- [ ] Add Kafka instrumentation (manual spans):
  - [ ] Producer: inject trace context into message headers
  - [ ] Consumer: extract trace context from message headers

### 7.3 Custom Metrics (via OTel Meter)
- [ ] Implement custom metrics:
  - [ ] `booking_reservations_total` (Counter)
  - [ ] `booking_reservation_failures` (Counter)
  - [ ] `http_request_duration_seconds` (Histogram)
  - [ ] `redis_operation_duration_seconds` (Histogram)
  - [ ] `active_reservations` (Gauge)
  - [ ] `kafka_consumer_lag` (Gauge)
  - [ ] `db_connections_active` (Gauge)

### 7.4 Logging with Trace Correlation
- [ ] Setup Zap/Zerolog with OTel bridge
- [ ] Inject `trace_id` and `span_id` into all logs
- [ ] Configure log export to OTel Collector
- [ ] Verify log-trace correlation in Grafana

### 7.5 Infrastructure Setup (Docker Compose)
- [ ] Add OTel Collector:
  - [ ] Create `otel-collector-config.yaml`
  - [ ] Configure OTLP receivers (gRPC + HTTP)
  - [ ] Configure exporters (Jaeger, Prometheus, Loki)
- [ ] Add Jaeger (Traces backend)
- [ ] Add Prometheus (Metrics backend)
- [ ] Add Loki (Logs backend)
- [ ] Add Grafana (Visualization)
- [ ] Configure data source connections in Grafana

### 7.6 Grafana Dashboards
- [ ] Create dashboards:
  - [ ] System Overview (all services health, request rates)
  - [ ] Booking Dashboard (reservations/min, success rate, latency)
  - [ ] Payment Dashboard (payment status, failure reasons)
  - [ ] Infrastructure Dashboard (CPU, Memory, Connections)
- [ ] Setup Tempo/Jaeger data source for traces
- [ ] Setup log-to-trace linking (Loki → Jaeger)

### 7.7 Alerting
- [ ] Configure Alertmanager in Docker Compose
- [ ] Create alert rules:
  - [ ] High error rate (> 1% for 5 min)
  - [ ] High latency (P99 > 500ms for 5 min)
  - [ ] Service down (health check fails 3x)
  - [ ] Kafka lag (> 10,000 messages)
  - [ ] Redis high memory (> 80%)
- [ ] Setup notification channel (Slack/Email)

**Milestone:** Full observability with OTel - traces, metrics, logs unified in Grafana

---

## Phase 8: Production Hardening
**Goal:** Ready for production deployment

### 8.1 Security Audit
- [ ] Review all endpoints for auth/authz
- [ ] Test for SQL injection
- [ ] Test for XSS vulnerabilities
- [ ] Verify HTTPS configuration
- [ ] Add security headers
- [ ] Review rate limiting
- [ ] Audit log review

### 8.2 Performance Optimization
- [ ] Profile and optimize hot paths
- [ ] Review database indexes
- [ ] Optimize Redis memory usage
- [ ] Connection pool tuning
- [ ] Goroutine leak check

### 8.3 End-to-End Load Testing
- [ ] Write E2E load test scenarios
- [ ] Test full booking flow under load
- [ ] Test payment processing under load
- [ ] Test virtual queue under load
- [ ] Document final performance results:
  - [ ] Target: 10k RPS confirmed
  - [ ] P99 latency confirmed
  - [ ] Zero overselling confirmed

### 8.4 Documentation
- [ ] API documentation (OpenAPI/Swagger)
- [ ] Architecture diagrams (draw.io/Mermaid)
- [ ] Deployment guide
- [ ] Runbook for operations
- [ ] README updates

### 8.5 Deployment
- [ ] Prepare production Docker images
- [ ] Setup Coolify/VPS
- [ ] Configure environment variables
- [ ] Setup database backups
- [ ] Configure monitoring alerts
- [ ] Deploy and verify

**Milestone:** System deployed and running in production

---

## Summary

| Phase | Focus | Key Deliverable |
|-------|-------|-----------------|
| 1 | Foundation | Infrastructure running |
| 2 | Core Booking | 10k RPS achieved |
| 3 | Auth & Events | User can browse and book |
| 4 | Payment | Complete booking-payment flow |
| 5 | Advanced | Virtual queue, notifications |
| 6 | Frontend | Web UI complete |
| 7 | Observability | Full monitoring |
| 8 | Production | Deployed and running |

---

## Progress Tracker

```
Phase 1: ░░░░░░░░░░ 0%
Phase 2: ░░░░░░░░░░ 0%
Phase 3: ░░░░░░░░░░ 0%
Phase 4: ░░░░░░░░░░ 0%
Phase 5: ░░░░░░░░░░ 0%
Phase 6: ░░░░░░░░░░ 0%
Phase 7: ░░░░░░░░░░ 0%
Phase 8: ░░░░░░░░░░ 0%
```
