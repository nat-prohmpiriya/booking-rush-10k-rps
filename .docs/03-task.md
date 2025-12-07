# Development Tasks: Booking Rush (10k RPS)

> **Version:** 2.0
> **Last Updated:** 2025-12-07
> **Reference:** [01-spec.md](./01-spec.md) | [02-plan.md](./02-plan.md)

---

## Task Format

แต่ละ Task ประกอบด้วย:
- **Task ID:** รหัส unique (P{phase}-{number})
- **Name:** ชื่อที่สื่อความหมาย
- **Description:** สิ่งที่ต้องทำโดยละเอียด
- **Technical Context:** ไฟล์/function ที่เกี่ยวข้อง
- **Acceptance Criteria:** เกณฑ์ที่บอกว่างานเสร็จ

---

## Phase 1: Foundation & Infrastructure

**Goal:** Setup project skeleton and infrastructure

### [x]  P1-01: Create Monorepo Directory Structure
| Field | Value |
|-------|-----------|
| **Description** | สร้างโครงสร้างโฟลเดอร์ตาม monorepo pattern |
| **Technical Context** | `apps/`, `pkg/`, `scripts/`, `tests/`, `infra/`, `libs/` |
| **Acceptance Criteria** | - โครงสร้างโฟลเดอร์ตรงตาม 02-plan.md Section 4.1<br>- `go.work` initialized<br>- `pnpm-workspace.yaml` created |

- [x] Create `apps/` directory with service placeholders
- [x] Create `pkg/` directory for shared Go packages
- [x] Create `scripts/lua/` and `scripts/migrations/`
- [x] Create `infra/` for observability configs
- [x] Initialize `go.work` for Go workspace
- [x] Create `pnpm-workspace.yaml` for TypeScript

---

### [x]  P1-02: Initialize Go Service Modules
| Field | Value |
|-------|-----------|
| **Description** | Initialize go modules สำหรับแต่ละ service |
| **Technical Context** | `apps/api-gateway/go.mod`, `apps/auth-service/go.mod`, etc. |
| **Acceptance Criteria** | - ทุก service มี `go.mod`<br>- `go work sync` ทำงานได้<br>- Shared pkg import ได้ |

- [x] `go mod init` for api-gateway
- [x] `go mod init` for auth-service
- [x] `go mod init` for ticket-service
- [x] `go mod init` for booking-service
- [x] `go mod init` for payment-service
- [x] `go mod init` for pkg/
- [x] Run `go work use ./apps/* ./pkg`

---

### [-]  P1-03: Docker Compose - Core Infrastructure (SKIPPED - ใช้ remote server แทน)
| Field | Value |
|-------|-----------|
| **Description** | สร้าง docker-compose.yml สำหรับ databases และ message queue |
| **Technical Context** | `docker-compose.yml` |
| **Acceptance Criteria** | - `docker-compose up` รันได้ไม่มี error<br>- PostgreSQL accessible on :5432<br>- Redis accessible on :6379<br>- Redpanda accessible on :9092 |

- [-] Add PostgreSQL 16 service with healthcheck (ใช้ remote server)
- [-] Add Redis 7 service (ใช้ remote server)
- [-] Add Redpanda service (Kafka-compatible) (ใช้ remote server)
- [-] Add MongoDB 7 service (ใช้ remote server)
- [-] Add Redpanda Console (Kafka UI)
- [-] Create `.env.example` with all variables
- [-] Create Docker network configuration

---

### [x]  P1-04: Shared Package - Config Loader
| Field | Value |
|-------|-----------|
| **Description** | สร้าง config loader ด้วย Viper |
| **Technical Context** | `pkg/config/config.go` |
| **Acceptance Criteria** | - Load config จาก env vars<br>- Load config จาก .env file<br>- Type-safe config struct |

- [x] Create `Config` struct with all app settings
- [x] Implement `Load()` function with Viper
- [x] Support environment variable override
- [x] Add config validation
- [x] Write unit tests

---

### [x]  P1-05: Shared Package - Logger
| Field | Value |
|-------|-------|
| **Description** | สร้าง structured JSON logger ด้วย Zap |
| **Technical Context** | `pkg/logger/logger.go` |
| **Acceptance Criteria** | - Log output เป็น JSON format<br>- มี fields: timestamp, level, service, trace_id<br>- Support log levels: DEBUG, INFO, WARN, ERROR |

- [x] Create Zap logger wrapper
- [x] Add structured fields (service, trace_id)
- [x] Implement log level configuration
- [x] Add context-aware logging
- [x] Write unit tests

---

### [x]  P1-06: Shared Package - Response Wrapper
| Field | Value |
|-------|-------|
| **Description** | สร้าง standard API response wrapper |
| **Technical Context** | `pkg/response/response.go` |
| **Acceptance Criteria** | - Success response มี `success: true`, `data`<br>- Error response มี `success: false`, `error.code`, `error.message`<br>- Support pagination meta |

- [x] Create `SuccessResponse` function
- [x] Create `ErrorResponse` function
- [x] Create `PaginatedResponse` function
- [x] Define error code constants
- [x] Write unit tests

---

### [x]  P1-07: Shared Package - Database Connection
| Field | Value |
|-------|-------|
| **Description** | สร้าง PostgreSQL connection pool ด้วย pgx |
| **Technical Context** | `pkg/database/postgres.go` |
| **Acceptance Criteria** | - Connection pool ทำงานได้<br>- Graceful shutdown<br>- Health check function |

- [x] Create connection pool with pgx/v5
- [x] Implement connection retry logic
- [x] Add `Ping()` health check
- [x] Add `Close()` graceful shutdown
- [x] Write integration tests

---

### [x]  P1-08: Shared Package - Redis Client
| Field | Value |
|-------|-------|
| **Description** | สร้าง Redis client wrapper |
| **Technical Context** | `pkg/redis/redis.go` |
| **Acceptance Criteria** | - Connect to Redis ได้<br>- Execute Lua scripts ได้<br>- Health check function |

- [x] Create Redis client with go-redis/v9
- [x] Implement `EvalSha` for Lua scripts
- [x] Add `Ping()` health check
- [x] Add script loading utility
- [x] Write integration tests

---

### [x]  P1-09: Database Migrations - Core Tables
| Field | Value |
|-------|-------|
| **Description** | สร้าง SQL migrations สำหรับ core tables |
| **Technical Context** | `scripts/migrations/000001_*.sql` |
| **Acceptance Criteria** | - Migration up/down ทำงานได้<br>- Tables: tenants, users, categories, events, shows, seat_zones<br>- Indexes created |

- [x] Create `000001_create_tenants.up.sql`
- [x] Create `000001_create_tenants.down.sql`
- [x] Create `000002_create_users.up.sql`
- [x] Create `000003_create_categories.up.sql`
- [x] Create `000004_create_events.up.sql`
- [x] Create `000005_create_shows.up.sql`
- [x] Create `000006_create_seat_zones.up.sql`
- [x] Add all required indexes
- [x] Test migrations up and down

---

### [x]  P1-10: Database Migrations - Booking Tables
| Field | Value |
|-------|-------|
| **Description** | สร้าง SQL migrations สำหรับ booking และ payment tables |
| **Technical Context** | `scripts/migrations/000007_*.sql` |
| **Acceptance Criteria** | - Tables: bookings, payments, outbox, audit_logs<br>- Partitioned audit_logs table<br>- All indexes created |

- [x] Create `000007_create_bookings.up.sql`
- [x] Create `000008_create_payments.up.sql`
- [x] Create `000009_create_outbox.up.sql`
- [x] Create `000010_create_audit_logs.up.sql` (partitioned)
- [x] Create partition for current month
- [x] Test migrations

---

### [ ]  P1-11: API Gateway - Basic Setup
| Field | Value |
|-------|-------|
| **Description** | Setup API Gateway ด้วย Gin framework |
| **Technical Context** | `apps/api-gateway/main.go`, `apps/api-gateway/internal/` |
| **Acceptance Criteria** | - Server starts on :8080<br>- `/health` returns 200<br>- `/ready` checks DB & Redis |

- [ ] Create `main.go` entry point
- [ ] Setup Gin router
- [ ] Implement `/health` endpoint
- [ ] Implement `/ready` endpoint (checks dependencies)
- [ ] Add request ID middleware
- [ ] Add logging middleware
- [ ] Add CORS middleware
- [ ] Write tests

---

### [ ]  P1-12: Makefile Commands
| Field | Value |
|-------|-------|
| **Description** | สร้าง Makefile สำหรับ common commands |
| **Technical Context** | `Makefile` |
| **Acceptance Criteria** | - `make dev` starts infrastructure<br>- `make migrate-up` runs migrations<br>- `make test` runs all tests |

- [ ] Add `dev` command (docker-compose up)
- [ ] Add `dev-down` command
- [ ] Add `migrate-up` and `migrate-down`
- [ ] Add `test` command
- [ ] Add `build` command for all services
- [ ] Add `lint` command

---

### [ ]  P1-13: Basic OTel Setup (Early Observability)
| Field | Value |
|-------|-------|
| **Description** | Setup basic OpenTelemetry tracing ตั้งแต่ Phase 1 เพื่อใช้ debug race conditions และ performance ใน Phase 2+ |
| **Technical Context** | `pkg/telemetry/`, `infra/otel/`, `docker-compose.yml` |
| **Acceptance Criteria** | - OTel Collector running in Docker<br>- Jaeger accessible for traces<br>- Basic tracing middleware พร้อมใช้<br>- Trace ID appears in logs |

- [ ] Add OTel Collector to docker-compose.yml
- [ ] Add Jaeger to docker-compose.yml (all-in-one image)
- [ ] Create basic `pkg/telemetry/tracer.go` with TracerProvider
- [ ] Inject trace_id into Zap logger context
- [ ] Create Gin middleware for auto-tracing
- [ ] Document how to view traces in Jaeger UI
- [ ] Test trace propagation works

---

**Phase 1 Milestone:** `docker-compose up` runs all infrastructure, API Gateway responds to health checks, **basic tracing operational**

---

## Phase 2: Core Booking Engine ⭐

**Goal:** Achieve 10,000 RPS on booking endpoint with zero overselling

### [ ]  P2-01: Redis Lua Script - Reserve Seats
| Field | Value |
|-------|-------|
| **Description** | เขียน Lua script สำหรับ atomic seat reservation |
| **Technical Context** | `scripts/lua/reserve_seats.lua` |
| **Acceptance Criteria** | - Check seat availability atomically<br>- Check user max limit per event<br>- Deduct seats and set TTL<br>- Return remaining count or error |

- [ ] Implement availability check logic
- [ ] Implement user limit check
- [ ] Implement atomic DECRBY
- [ ] Set reservation TTL (10 min)
- [ ] Return structured response (success/error + remaining)
- [ ] Write Lua script tests

---

### [ ]  P2-02: Redis Lua Script - Release Seats
| Field | Value |
|-------|-------|
| **Description** | เขียน Lua script สำหรับ release seats กลับ inventory |
| **Technical Context** | `scripts/lua/release_seats.lua` |
| **Acceptance Criteria** | - Increment seat count atomically<br>- Delete reservation key<br>- Update user reserved count |

- [ ] Implement atomic INCRBY
- [ ] Delete reservation record
- [ ] Update user's reserved count
- [ ] Write tests

---

### [ ]  P2-03: Redis Lua Script - Confirm Booking
| Field | Value |
|-------|-------|
| **Description** | เขียน Lua script สำหรับ confirm booking |
| **Technical Context** | `scripts/lua/confirm_booking.lua` |
| **Acceptance Criteria** | - Validate reservation exists<br>- Mark as confirmed<br>- Remove TTL |

- [ ] Check reservation exists and not expired
- [ ] Update reservation status
- [ ] Remove TTL (permanent)
- [ ] Write tests

---

### [ ]  P2-04: Booking Service - Project Structure
| Field | Value |
|-------|-------|
| **Description** | สร้าง Clean Architecture structure สำหรับ Booking Service |
| **Technical Context** | `apps/booking-service/internal/` |
| **Acceptance Criteria** | - Layers: handler, service, repository, domain<br>- Dependency injection setup<br>- Service starts without error |

- [ ] Create `internal/handler/` directory
- [ ] Create `internal/service/` directory
- [ ] Create `internal/repository/` directory
- [ ] Create `internal/domain/` directory
- [ ] Create `internal/dto/` directory
- [ ] Setup dependency injection
- [ ] Create `main.go`

---

### [ ]  P2-05: Booking Service - Domain Models
| Field | Value |
|-------|-------|
| **Description** | สร้าง domain entities สำหรับ booking |
| **Technical Context** | `apps/booking-service/internal/domain/booking.go` |
| **Acceptance Criteria** | - Booking entity with all fields<br>- Status enum (reserved, confirmed, cancelled, expired)<br>- Validation methods |

- [ ] Create `Booking` struct
- [ ] Create `BookingStatus` enum
- [ ] Create `Reservation` struct (Redis)
- [ ] Add validation methods
- [ ] Write tests

---

### [ ]  P2-06: Booking Service - Repository Layer
| Field | Value |
|-------|-------|
| **Description** | Implement repository สำหรับ PostgreSQL และ Redis |
| **Technical Context** | `apps/booking-service/internal/repository/` |
| **Acceptance Criteria** | - PostgreSQL CRUD operations<br>- Redis Lua script execution<br>- Transaction support |

- [ ] Create `BookingRepository` interface
- [ ] Implement `PostgresBookingRepository`
- [ ] Create `RedisRepository` interface
- [ ] Implement `RedisInventoryRepository`
- [ ] Implement `ReserveSeats()` using Lua
- [ ] Implement `ReleaseSeats()` using Lua
- [ ] Write integration tests

---

### [ ]  P2-07: Booking Service - Service Layer
| Field | Value |
|-------|-------|
| **Description** | Implement business logic สำหรับ booking |
| **Technical Context** | `apps/booking-service/internal/service/booking_service.go` |
| **Acceptance Criteria** | - Reserve seats with idempotency<br>- Confirm booking<br>- Cancel booking<br>- List user bookings |

- [ ] Create `BookingService` struct
- [ ] Implement `ReserveSeats()` with idempotency key
- [ ] Implement `ConfirmBooking()`
- [ ] Implement `CancelBooking()`
- [ ] Implement `GetUserBookings()`
- [ ] Implement `GetPendingBookings()`
- [ ] Write unit tests with mocks

---

### [ ]  P2-08: Booking Service - HTTP Handlers
| Field | Value |
|-------|-------|
| **Description** | Implement HTTP handlers สำหรับ booking endpoints |
| **Technical Context** | `apps/booking-service/internal/handler/booking_handler.go` |
| **Acceptance Criteria** | - `POST /bookings/reserve` works<br>- `POST /bookings/:id/confirm` works<br>- `POST /bookings/:id/cancel` works<br>- `GET /bookings` works<br>- `GET /bookings/pending` works |

- [ ] Create `BookingHandler` struct
- [ ] Implement `Reserve()` handler
- [ ] Implement `Confirm()` handler
- [ ] Implement `Cancel()` handler
- [ ] Implement `List()` handler
- [ ] Implement `GetPending()` handler
- [ ] Add request validation
- [ ] Write API tests

---

### [ ]  P2-09: Booking Service - Kafka Producer
| Field | Value |
|-------|-------|
| **Description** | Implement Kafka producer สำหรับ booking events |
| **Technical Context** | `apps/booking-service/internal/service/event_publisher.go` |
| **Acceptance Criteria** | - Publish `booking.created` event<br>- Publish `booking.confirmed` event<br>- Publish `booking.cancelled` event<br>- Publish `booking.expired` event |

- [ ] Create `EventPublisher` interface
- [ ] Implement Kafka producer wrapper
- [ ] Define event schemas
- [ ] Publish events after state changes
- [ ] Write tests

---

### [ ]  P2-10: Inventory Sync Worker
| Field | Value |
|-------|-------|
| **Description** | สร้าง worker สำหรับ sync Redis → PostgreSQL |
| **Technical Context** | `apps/booking-service/cmd/inventory-worker/` |
| **Acceptance Criteria** | - Consume booking events from Kafka<br>- Batch update PostgreSQL every 5 seconds<br>- Handle Redis rebuild on startup |

- [ ] Create Kafka consumer
- [ ] Implement batch aggregation (5 sec window)
- [ ] Implement PostgreSQL batch update
- [ ] Add startup Redis rebuild from DB
- [ ] Write tests

---

### [ ]  P2-11: Load Testing Setup
| Field | Value |
|-------|-------|
| **Description** | Setup k6 สำหรับ load testing |
| **Technical Context** | `tests/load/booking_reserve.js` |
| **Acceptance Criteria** | - k6 script for `/bookings/reserve`<br>- Test scenarios defined<br>- Can run 10k RPS test |

- [ ] Install and configure k6
- [ ] Create test script for reserve endpoint
- [ ] Define test scenarios (ramp-up, sustained, spike)
- [ ] Create test data seeding script
- [ ] Document how to run tests

---

### [ ]  P2-12: Performance Optimization
| Field | Value |
|-------|-------|
| **Description** | Optimize จนได้ 10k RPS |
| **Technical Context** | All booking service code |
| **Acceptance Criteria** | - Achieve 10,000 RPS<br>- P99 latency < 50ms (server)<br>- Error rate < 0.1%<br>- Zero overselling |

- [ ] Run initial load test and document baseline
- [ ] Profile and identify bottlenecks
- [ ] Optimize connection pools
- [ ] Optimize Lua scripts
- [ ] Re-run tests until targets met
- [ ] Document final performance results

---

### [ ]  P2-13: Dirty Scenario Testing
| Field | Value |
|-------|-------|
| **Description** | Test edge cases ที่อาจเกิดขึ้นจริง เช่น client disconnect, timeout, concurrent booking |
| **Technical Context** | `tests/load/dirty_scenarios.js`, booking service |
| **Acceptance Criteria** | - Client disconnect หลัง reserve แต่ก่อน payment → seats released after TTL<br>- Network timeout mid-request → no duplicate reservations<br>- Concurrent booking for last seat → only 1 succeeds<br>- Payment timeout → reservation released, refund triggered |

- [ ] Test: Client disconnects after reserve, before payment
  - Expected: Seats released after 10 min TTL
  - Verify: No orphaned reservations
- [ ] Test: Client retries with same idempotency key
  - Expected: Same response returned, no double-booking
- [ ] Test: 100 concurrent requests for last 1 seat
  - Expected: Exactly 1 success, 99 failures with INSUFFICIENT_STOCK
  - Verify: Total seat count unchanged (no negative inventory)
- [ ] Test: Payment service times out
  - Expected: Saga compensates, seats released
- [ ] Test: Kafka consumer crashes mid-processing
  - Expected: Message reprocessed after restart (at-least-once)
  - Verify: Idempotency prevents duplicate side effects
- [ ] Test: Redis crashes during reservation
  - Expected: Service returns 503, graceful degradation
- [ ] Document all dirty scenarios and expected behaviors

---

### [ ]  P2-14: Thundering Herd Rejection Efficiency
| Field | Value |
|-------|-------|
| **Description** | ตรวจสอบว่า Booking Service ปฏิเสธ requests เกินกำลังได้อย่างมีประสิทธิภาพ (ก่อน Virtual Queue พร้อมใน Phase 6) |
| **Technical Context** | `apps/booking-service/`, `apps/api-gateway/` |
| **Acceptance Criteria** | - 429 responses returned < 5ms<br>- No resource exhaustion under 20k RPS spike<br>- Error messages เข้าใจง่ายสำหรับ client |

- [ ] Test API Gateway rate limiting under 20k RPS
  - Verify: 429 responses fast (< 5ms)
  - Verify: X-RateLimit headers correct
  - Verify: Retry-After header present
- [ ] Test Booking Service rejection when sold out
  - Verify: Lua script returns immediately (< 1ms)
  - Verify: No DB connections used for rejections
- [ ] Monitor resource usage under rejection load
  - Verify: CPU/Memory stable
  - Verify: No goroutine leaks
- [ ] Create client-side retry guidelines doc
  - Document: Exponential backoff strategy
  - Document: When to stop retrying

---

**Phase 2 Milestone:** Achieve 10,000 RPS with zero overselling, **dirty scenarios handled correctly**, **thundering herd rejection efficient**

---

## Phase 3: Authentication & Events

**Goal:** Complete auth flow and event management

### [ ]  P3-01: Auth Service - Project Structure
| Field | Value |
|-------|-------|
| **Description** | Setup Auth Service structure |
| **Technical Context** | `apps/auth-service/internal/` |
| **Acceptance Criteria** | - Clean Architecture layers<br>- Service starts on :8081<br>- Health check works |

- [ ] Create project structure (handler, service, repository, domain)
- [ ] Setup dependency injection
- [ ] Create `main.go`
- [ ] Add health check endpoint

---

### [ ]  P3-02: Auth Service - User Registration
| Field | Value |
|-------|-------|
| **Description** | Implement user registration |
| **Technical Context** | `apps/auth-service/internal/handler/auth_handler.go` |
| **Acceptance Criteria** | - `POST /auth/register` creates user<br>- Email validation<br>- Password hashed with bcrypt (cost 12)<br>- Returns user without password |

- [ ] Create User domain model
- [ ] Implement UserRepository
- [ ] Implement registration service logic
- [ ] Create registration handler
- [ ] Add email format validation
- [ ] Add password strength validation
- [ ] Write tests

---

### [ ]  P3-03: Auth Service - Login & JWT
| Field | Value |
|-------|-------|
| **Description** | Implement login with JWT |
| **Technical Context** | `apps/auth-service/internal/service/auth_service.go` |
| **Acceptance Criteria** | - `POST /auth/login` returns access + refresh token<br>- Access token: 15 min expiry<br>- Refresh token: 7 days expiry<br>- JWT contains: sub, email, role, tenant_id |

- [ ] Implement password verification
- [ ] Implement JWT generation (access token)
- [ ] Implement refresh token generation
- [ ] Store refresh token in DB
- [ ] Create login handler
- [ ] Write tests

---

### [ ]  P3-04: Auth Service - Token Refresh
| Field | Value |
|-------|-------|
| **Description** | Implement token refresh |
| **Technical Context** | `apps/auth-service/internal/handler/auth_handler.go` |
| **Acceptance Criteria** | - `POST /auth/refresh` returns new access token<br>- Validates refresh token<br>- Rotates refresh token (security) |

- [ ] Validate refresh token
- [ ] Generate new access token
- [ ] Rotate refresh token
- [ ] Invalidate old refresh token
- [ ] Write tests

---

### [ ]  P3-05: Auth Service - Logout
| Field | Value |
|-------|-------|
| **Description** | Implement logout |
| **Technical Context** | `apps/auth-service/internal/handler/auth_handler.go` |
| **Acceptance Criteria** | - `POST /auth/logout` invalidates refresh token<br>- Requires authentication |

- [ ] Invalidate refresh token in DB
- [ ] Return success response
- [ ] Write tests

---

### [ ]  P3-06: Auth Service - Profile Endpoints
| Field | Value |
|-------|-------|
| **Description** | Implement profile management |
| **Technical Context** | `apps/auth-service/internal/handler/auth_handler.go` |
| **Acceptance Criteria** | - `GET /auth/me` returns current user<br>- `PUT /auth/me` updates profile |

- [ ] Implement `GET /auth/me`
- [ ] Implement `PUT /auth/me`
- [ ] Validate update fields
- [ ] Write tests

---

### [ ]  P3-07: JWT Middleware
| Field | Value |
|-------|-------|
| **Description** | Create JWT validation middleware |
| **Technical Context** | `pkg/middleware/jwt.go` |
| **Acceptance Criteria** | - Validates JWT from Authorization header<br>- Injects user context into request<br>- Returns 401 for invalid/expired tokens |

- [ ] Create JWT middleware
- [ ] Parse and validate token
- [ ] Extract claims and inject into context
- [ ] Handle expired tokens
- [ ] Write tests

---

### [ ]  P3-08: Ticket Service - Project Structure
| Field | Value |
|-------|-------|
| **Description** | Setup Ticket Service structure |
| **Technical Context** | `apps/ticket-service/internal/` |
| **Acceptance Criteria** | - Clean Architecture layers<br>- Service starts on :8082<br>- Health check works |

- [ ] Create project structure
- [ ] Setup dependency injection
- [ ] Create `main.go`
- [ ] Add health check endpoint

---

### [ ]  P3-09: Ticket Service - Event CRUD
| Field | Value |
|-------|-------|
| **Description** | Implement Event CRUD operations |
| **Technical Context** | `apps/ticket-service/internal/handler/event_handler.go` |
| **Acceptance Criteria** | - `GET /events` lists events (paginated)<br>- `GET /events/:slug` returns event detail<br>- `POST /events` creates event (Organizer only)<br>- `PUT /events/:id` updates event<br>- `DELETE /events/:id` soft deletes event |

- [ ] Create Event domain model
- [ ] Implement EventRepository
- [ ] Implement list with pagination and filters
- [ ] Implement get by slug
- [ ] Implement create (with slug generation)
- [ ] Implement update
- [ ] Implement delete
- [ ] Add authorization checks (Organizer role)
- [ ] Write tests

---

### [ ]  P3-10: Ticket Service - Show Management
| Field | Value |
|-------|-------|
| **Description** | Implement Show management |
| **Technical Context** | `apps/ticket-service/internal/handler/show_handler.go` |
| **Acceptance Criteria** | - `GET /events/:slug/shows` lists shows<br>- `POST /events/:id/shows` creates show |

- [ ] Create Show domain model
- [ ] Implement ShowRepository
- [ ] Implement list shows for event
- [ ] Implement create show
- [ ] Write tests

---

### [ ]  P3-11: Ticket Service - Zone Management
| Field | Value |
|-------|-------|
| **Description** | Implement Zone/Seat management |
| **Technical Context** | `apps/ticket-service/internal/handler/zone_handler.go` |
| **Acceptance Criteria** | - `GET /shows/:id/zones` lists zones with availability<br>- `POST /shows/:id/zones` creates zone |

- [ ] Create Zone domain model
- [ ] Implement ZoneRepository
- [ ] Implement list zones with real-time availability (from Redis)
- [ ] Implement create zone
- [ ] Sync initial inventory to Redis
- [ ] Write tests

---

### [ ]  P3-12: Ticket Service - Redis Caching
| Field | Value |
|-------|-------|
| **Description** | Add Redis caching สำหรับ events |
| **Technical Context** | `apps/ticket-service/internal/repository/cache_repo.go` |
| **Acceptance Criteria** | - Event list cached (TTL: 5 min)<br>- Event detail cached (TTL: 5 min)<br>- Cache invalidation on update |

- [ ] Implement cache layer
- [ ] Cache event list
- [ ] Cache event detail
- [ ] Implement cache invalidation
- [ ] Write tests

---

### [ ]  P3-13: API Gateway - Rate Limiting
| Field | Value |
|-------|-------|
| **Description** | Implement Token Bucket rate limiting |
| **Technical Context** | `apps/api-gateway/internal/middleware/rate_limit.go` |
| **Acceptance Criteria** | - Token Bucket algorithm with burst<br>- Per-endpoint configuration<br>- Store state in Redis<br>- Return 429 with `Retry-After` header |

- [ ] Implement Token Bucket algorithm
- [ ] Configure per-endpoint limits
- [ ] Store tokens in Redis
- [ ] Add `X-RateLimit-*` headers
- [ ] Add `Retry-After` header on 429
- [ ] Write tests

---

### [ ]  P3-14: API Gateway - Service Routing
| Field | Value |
|-------|-------|
| **Description** | Implement routing to backend services |
| **Technical Context** | `apps/api-gateway/internal/proxy/` |
| **Acceptance Criteria** | - Route `/auth/*` to Auth Service<br>- Route `/events/*` to Ticket Service<br>- Route `/bookings/*` to Booking Service<br>- JWT middleware on protected routes |

- [ ] Implement reverse proxy
- [ ] Configure route mappings
- [ ] Add JWT middleware to protected routes
- [ ] Pass user context to backend services
- [ ] Write tests

---

**Phase 3 Milestone:** Users can register, login, browse events, and book tickets

---

## Phase 4: Payment & Saga Pattern

**Goal:** Complete payment flow with data consistency using Saga

### [ ]  P4-01: Payment Service - Project Structure
| Field | Value |
|-------|-------|
| **Description** | Setup Payment Service structure |
| **Technical Context** | `apps/payment-service/internal/` |
| **Acceptance Criteria** | - Clean Architecture layers<br>- Service starts on :8083<br>- Health check works |

- [ ] Create project structure
- [ ] Setup dependency injection
- [ ] Create `main.go`
- [ ] Add health check endpoint

---

### [ ]  P4-02: Payment Service - Kafka Consumer
| Field | Value |
|-------|-------|
| **Description** | Implement Kafka consumer สำหรับ booking events |
| **Technical Context** | `apps/payment-service/internal/consumer/` |
| **Acceptance Criteria** | - Consume `booking.created` events<br>- Process payment automatically<br>- Produce payment result events |

- [ ] Create Kafka consumer
- [ ] Handle `booking.created` event
- [ ] Trigger payment processing
- [ ] Handle consumer errors and retries
- [ ] Write tests

---

### [ ]  P4-03: Payment Service - Payment Processing
| Field | Value |
|-------|-------|
| **Description** | Implement payment processing (mock) |
| **Technical Context** | `apps/payment-service/internal/service/payment_service.go` |
| **Acceptance Criteria** | - Create payment record<br>- Process payment (mock success/failure)<br>- Update payment status<br>- Produce `payment.success` or `payment.failed` |

- [ ] Create Payment domain model
- [ ] Implement PaymentRepository
- [ ] Implement mock payment gateway
- [ ] Handle payment states (PENDING → PROCESSING → SUCCESS/FAILED)
- [ ] Produce result events
- [ ] Write tests

---

### [ ]  P4-04: Payment Service - HTTP Endpoints
| Field | Value |
|-------|-------|
| **Description** | Implement payment HTTP endpoints |
| **Technical Context** | `apps/payment-service/internal/handler/` |
| **Acceptance Criteria** | - `POST /payments` initiates payment<br>- `GET /payments/:id` returns status<br>- `POST /payments/:id/refund` requests refund |

- [ ] Implement payment initiation endpoint
- [ ] Implement payment status endpoint
- [ ] Implement refund request endpoint
- [ ] Write tests

---

### [ ]  P4-05: Saga Orchestrator - Setup
| Field | Value |
|-------|-------|
| **Description** | Create Saga orchestrator framework |
| **Technical Context** | `pkg/saga/` |
| **Acceptance Criteria** | - Saga definition struct<br>- Step execution with compensation<br>- Saga state persistence |

- [ ] Create `Saga` struct
- [ ] Create `SagaStep` struct with Execute/Compensate
- [ ] Create `SagaOrchestrator`
- [ ] Implement step execution logic
- [ ] Implement compensation on failure
- [ ] Write tests

---

### [ ]  P4-06: Saga Orchestrator - State Machine
| Field | Value |
|-------|-------|
| **Description** | Implement Saga state machine |
| **Technical Context** | `pkg/saga/state.go` |
| **Acceptance Criteria** | - States: CREATED, RESERVED, PAID, CONFIRMED, FAILED<br>- State transitions validated<br>- State persisted in DB |

- [ ] Define SagaState enum
- [ ] Create saga_instances table (migration)
- [ ] Implement state transitions
- [ ] Persist state changes
- [ ] Write tests

---

### [ ]  P4-07: Booking Saga Implementation
| Field | Value |
|-------|-------|
| **Description** | Implement Booking Saga with all steps |
| **Technical Context** | `apps/booking-service/internal/saga/booking_saga.go` |
| **Acceptance Criteria** | - Step 1: Reserve Seats (compensate: Release)<br>- Step 2: Process Payment (compensate: Refund)<br>- Step 3: Confirm Booking<br>- Step 4: Send Notification |

- [ ] Define BookingSaga with steps
- [ ] Implement Reserve step
- [ ] Implement Release compensation
- [ ] Implement Payment step
- [ ] Implement Refund compensation
- [ ] Implement Confirm step
- [ ] Implement Notification step
- [ ] Write integration tests

---

### [ ]  P4-08: Saga Kafka Integration
| Field | Value |
|-------|-------|
| **Description** | Integrate Saga with Kafka commands/events |
| **Technical Context** | `apps/booking-service/internal/saga/` |
| **Acceptance Criteria** | - Produce saga command topics<br>- Consume saga event topics<br>- Handle timeouts |

- [ ] Define saga command topics
- [ ] Define saga event topics
- [ ] Produce commands from orchestrator
- [ ] Consume events and advance saga
- [ ] Handle step timeouts
- [ ] Write tests

---

### [ ]  P4-09: Idempotency Implementation
| Field | Value |
|-------|-------|
| **Description** | Implement idempotency for all write operations |
| **Technical Context** | `pkg/middleware/idempotency.go` |
| **Acceptance Criteria** | - Idempotency key from header/body<br>- Store in Redis (24h TTL)<br>- Return cached response for duplicates |

- [ ] Create idempotency middleware
- [ ] Store request hash in Redis
- [ ] Return cached response for duplicates
- [ ] Add to booking and payment endpoints
- [ ] Write tests

---

### [ ]  P4-10: Transactional Outbox
| Field | Value |
|-------|-------|
| **Description** | Implement Transactional Outbox pattern |
| **Technical Context** | `apps/booking-service/internal/repository/outbox_repo.go` |
| **Acceptance Criteria** | - Write booking + outbox in same transaction<br>- Outbox poller publishes to Kafka<br>- Mark messages as processed |

- [ ] Create OutboxRepository
- [ ] Write outbox entry in booking transaction
- [ ] Create outbox poller worker
- [ ] Publish events to Kafka
- [ ] Mark as processed
- [ ] Write tests

---

### [ ]  P4-11: Reservation Expiry Worker
| Field | Value |
|-------|-------|
| **Description** | Implement worker สำหรับ expire reservations |
| **Technical Context** | `apps/booking-service/cmd/expiry-worker/` |
| **Acceptance Criteria** | - Scan expired reservations<br>- Release seats back to inventory<br>- Update booking status to `expired`<br>- Produce `booking.expired` event |

- [ ] Create expiry worker
- [ ] Scan for expired reservations (Redis keyspace notification or cron)
- [ ] Release seats via Lua script
- [ ] Update DB status
- [ ] Produce expired event
- [ ] Write tests

---

### [ ]  P4-12: Retry Logic with Backoff
| Field | Value |
|-------|-------|
| **Description** | Implement retry logic with exponential backoff |
| **Technical Context** | `pkg/retry/retry.go` |
| **Acceptance Criteria** | - Exponential backoff (1s, 2s, 4s)<br>- Jitter to prevent thundering herd<br>- Max retries configurable<br>- Dead letter queue for failed messages |

- [ ] Create retry utility
- [ ] Implement exponential backoff
- [ ] Add jitter
- [ ] Configure max retries
- [ ] Implement DLQ publishing
- [ ] Write tests

---

**Phase 4 Milestone:** Complete booking-to-payment flow with Saga pattern and consistency guarantees

---

## Phase 5: NestJS Services

**Goal:** Implement Notification and Analytics services with NestJS + MongoDB

### [ ]  P5-01: Notification Service - Project Setup
| Field | Value |
|-------|-------|
| **Description** | Setup NestJS project for Notification Service |
| **Technical Context** | `apps/notification-service/` |
| **Acceptance Criteria** | - NestJS project initialized<br>- MongoDB connected<br>- Kafka consumer configured<br>- Service starts on :8084 |

- [ ] Initialize NestJS project
- [ ] Setup MongoDB with Mongoose
- [ ] Configure Kafka consumer
- [ ] Create module structure
- [ ] Add health check endpoint
- [ ] Write tests

---

### [ ]  P5-02: Notification Service - MongoDB Schemas
| Field | Value |
|-------|-------|
| **Description** | Create MongoDB schemas for notifications |
| **Technical Context** | `apps/notification-service/src/schemas/` |
| **Acceptance Criteria** | - `notifications` collection schema<br>- `notification_templates` collection schema<br>- Indexes created |

- [ ] Create Notification schema
- [ ] Create NotificationTemplate schema
- [ ] Add required indexes
- [ ] Seed default templates
- [ ] Write tests

---

### [ ]  P5-03: Notification Service - Email Module
| Field | Value |
|-------|-------|
| **Description** | Implement email sending module |
| **Technical Context** | `apps/notification-service/src/modules/email/` |
| **Acceptance Criteria** | - Send emails via Nodemailer/SendGrid<br>- Support HTML templates (Handlebars)<br>- Retry on failure |

- [ ] Create EmailModule
- [ ] Integrate Nodemailer or SendGrid
- [ ] Implement Handlebars template rendering
- [ ] Add retry logic
- [ ] Write tests

---

### [ ]  P5-04: Notification Service - Kafka Consumer
| Field | Value |
|-------|-------|
| **Description** | Consume booking/payment events |
| **Technical Context** | `apps/notification-service/src/modules/kafka/` |
| **Acceptance Criteria** | - Consume `booking.created` → booking confirmation<br>- Consume `booking.confirmed` → e-ticket<br>- Consume `payment.success` → payment receipt<br>- Consume `booking.expired` → expiry notice |

- [ ] Create KafkaModule
- [ ] Handle booking.created event
- [ ] Handle booking.confirmed event
- [ ] Handle payment.success event
- [ ] Handle payment.failed event
- [ ] Handle booking.expired event
- [ ] Write tests

---

### [ ]  P5-05: Analytics Service - Project Setup
| Field | Value |
|-------|-------|
| **Description** | Setup NestJS project for Analytics Service |
| **Technical Context** | `apps/analytics-service/` |
| **Acceptance Criteria** | - NestJS project initialized<br>- MongoDB connected<br>- Kafka consumer configured<br>- Service starts on :8085 |

- [ ] Initialize NestJS project
- [ ] Setup MongoDB with Mongoose
- [ ] Configure Kafka consumer
- [ ] Create module structure
- [ ] Add health check endpoint

---

### [ ]  P5-06: Analytics Service - MongoDB Schemas
| Field | Value |
|-------|-------|
| **Description** | Create MongoDB schemas for analytics |
| **Technical Context** | `apps/analytics-service/src/schemas/` |
| **Acceptance Criteria** | - `events_raw` collection (raw event log)<br>- `analytics_daily` collection (aggregated)<br>- `analytics_realtime` collection (TTL index) |

- [ ] Create EventsRaw schema
- [ ] Create AnalyticsDaily schema
- [ ] Create AnalyticsRealtime schema (with TTL)
- [ ] Add required indexes
- [ ] Write tests

---

### [ ]  P5-07: Analytics Service - Event Aggregation
| Field | Value |
|-------|-------|
| **Description** | Consume events and aggregate metrics |
| **Technical Context** | `apps/analytics-service/src/modules/aggregation/` |
| **Acceptance Criteria** | - Consume all booking/payment events<br>- Store raw events<br>- Update daily aggregations<br>- Update real-time stats |

- [ ] Create AggregationModule
- [ ] Store raw events
- [ ] Implement daily aggregation logic
- [ ] Implement real-time stats update
- [ ] Write tests

---

### [ ]  P5-08: Analytics Service - REST API
| Field | Value |
|-------|-------|
| **Description** | Implement REST API for dashboard |
| **Technical Context** | `apps/analytics-service/src/modules/api/` |
| **Acceptance Criteria** | - `GET /analytics/events/:id/realtime` returns real-time stats<br>- `GET /analytics/events/:id/daily` returns daily stats<br>- `GET /analytics/dashboard` returns overview |

- [ ] Create ApiModule
- [ ] Implement realtime stats endpoint
- [ ] Implement daily stats endpoint
- [ ] Implement dashboard overview endpoint
- [ ] Write tests

---

**Phase 5 Milestone:** NestJS services (Notification + Analytics) operational with MongoDB

---

## Phase 6: Virtual Queue & Advanced Features

**Goal:** Implement virtual queue and audit logging

### [ ]  P6-01: Virtual Queue - Join Queue
| Field | Value |
|-------|-------|
| **Description** | Implement queue join endpoint |
| **Technical Context** | `apps/booking-service/internal/handler/queue_handler.go` |
| **Acceptance Criteria** | - `POST /queue/join` adds user to queue<br>- Uses Redis Sorted Set (score = timestamp)<br>- Returns queue position |

- [ ] Create QueueHandler
- [ ] Implement join queue logic
- [ ] Store in Redis Sorted Set
- [ ] Return position and estimated wait
- [ ] Write tests

---

### [ ]  P6-02: Virtual Queue - Queue Status
| Field | Value |
|-------|-------|
| **Description** | Implement queue status endpoint |
| **Technical Context** | `apps/booking-service/internal/handler/queue_handler.go` |
| **Acceptance Criteria** | - `GET /queue/status` returns current position<br>- Returns estimated wait time<br>- Returns `ready` status when position = 0 |

- [ ] Implement status endpoint
- [ ] Calculate current position
- [ ] Calculate estimated wait time
- [ ] Indicate when ready to proceed
- [ ] Write tests

---

### [ ]  P6-03: Virtual Queue - Queue Pass Token
| Field | Value |
|-------|-------|
| **Description** | Generate Queue Pass Token when user reaches front |
| **Technical Context** | `apps/booking-service/internal/service/queue_service.go` |
| **Acceptance Criteria** | - Generate JWT token when position = 0<br>- Token valid for 5 minutes<br>- Store in Redis for validation |

- [ ] Implement Queue Pass generation
- [ ] Sign as JWT with expiry
- [ ] Store in Redis
- [ ] Return in queue status response
- [ ] Write tests

---

### [ ]  P6-04: API Gateway - Queue Pass Validation
| Field | Value |
|-------|-------|
| **Description** | Validate Queue Pass in API Gateway |
| **Technical Context** | `apps/api-gateway/internal/middleware/queue_pass.go` |
| **Acceptance Criteria** | - Check `X-Queue-Pass` header<br>- Validate token signature and expiry<br>- Bypass rate limit for valid passes<br>- Block users without pass during high traffic |

- [ ] Create Queue Pass middleware
- [ ] Validate JWT signature
- [ ] Check expiry
- [ ] Bypass rate limit if valid
- [ ] Block if no pass during queue mode
- [ ] Write tests

---

### [ ]  P6-05: Queue Release Batch Worker
| Field | Value |
|-------|-------|
| **Description** | Release users from queue in batches |
| **Technical Context** | `apps/booking-service/cmd/queue-worker/` |
| **Acceptance Criteria** | - Release 100 users per batch<br>- Generate Queue Pass for released users<br>- Run continuously |

- [ ] Create queue release worker
- [ ] Pop users from Sorted Set
- [ ] Generate Queue Pass tokens
- [ ] Configure batch size
- [ ] Write tests

---

### [ ]  P6-06: Audit Logging Middleware
| Field | Value |
|-------|-------|
| **Description** | Implement audit logging middleware |
| **Technical Context** | `pkg/middleware/audit.go` |
| **Acceptance Criteria** | - Log all write operations<br>- Capture: user_id, action, entity, old/new values<br>- Capture: IP address, user agent<br>- Store in partitioned table |

- [ ] Create audit middleware
- [ ] Capture request details
- [ ] Log to audit_logs table
- [ ] Handle async logging (don't block request)
- [ ] Write tests

---

### [ ]  P6-07: Audit Log Endpoints
| Field | Value |
|-------|-------|
| **Description** | Implement audit log viewing endpoints |
| **Technical Context** | `apps/admin-service/` or API Gateway |
| **Acceptance Criteria** | - `GET /admin/audit-logs` lists logs (paginated)<br>- Filter by user, action, entity<br>- Admin only access |

- [ ] Create audit log repository
- [ ] Implement list endpoint with filters
- [ ] Add pagination
- [ ] Restrict to admin role
- [ ] Write tests

---

**Phase 6 Milestone:** Virtual queue working, audit trail complete

---

## Phase 7: Frontend

**Goal:** User-facing web application

### [ ]  P7-01: Next.js Project Setup
| Field | Value |
|-------|-------|
| **Description** | Initialize Next.js 15 project |
| **Technical Context** | `apps/web/` |
| **Acceptance Criteria** | - Next.js 15 with App Router<br>- TailwindCSS configured<br>- Shadcn UI installed<br>- Development server runs |

- [ ] Create Next.js project
- [ ] Configure TailwindCSS
- [ ] Install Shadcn UI
- [ ] Setup environment variables
- [ ] Create base layout

---

### [ ]  P7-02: API Client Setup
| Field | Value |
|-------|-------|
| **Description** | Setup API client with Axios |
| **Technical Context** | `apps/web/lib/api.ts` |
| **Acceptance Criteria** | - Axios instance with base URL<br>- JWT token injection<br>- Error handling<br>- Token refresh logic |

- [ ] Create Axios instance
- [ ] Add JWT interceptor
- [ ] Handle 401 (token refresh)
- [ ] Handle errors globally
- [ ] Write tests

---

### [ ]  P7-03: Auth State Management
| Field | Value |
|-------|-------|
| **Description** | Implement auth state with Zustand |
| **Technical Context** | `apps/web/store/auth.ts` |
| **Acceptance Criteria** | - Login/logout actions<br>- Persist tokens in localStorage<br>- User state available globally |

- [ ] Create auth store
- [ ] Implement login action
- [ ] Implement logout action
- [ ] Persist tokens
- [ ] Create auth provider

---

### [ ]  P7-04: Auth Pages
| Field | Value |
|-------|-------|
| **Description** | Create auth pages |
| **Technical Context** | `apps/web/app/auth/` |
| **Acceptance Criteria** | - `/auth/login` page<br>- `/auth/register` page<br>- Form validation<br>- Error handling |

- [ ] Create login page
- [ ] Create register page
- [ ] Add form validation (Zod + React Hook Form)
- [ ] Handle errors
- [ ] Redirect on success

---

### [ ]  P7-05: Event List Page
| Field | Value |
|-------|-------|
| **Description** | Create event listing page |
| **Technical Context** | `apps/web/app/page.tsx` |
| **Acceptance Criteria** | - Homepage shows event list<br>- Search and filter support<br>- Pagination<br>- Responsive design |

- [ ] Create event list component
- [ ] Implement search
- [ ] Implement filters (category)
- [ ] Add pagination
- [ ] Make responsive

---

### [ ]  P7-06: Event Detail Page
| Field | Value |
|-------|-------|
| **Description** | Create event detail page |
| **Technical Context** | `apps/web/app/events/[slug]/page.tsx` |
| **Acceptance Criteria** | - `/events/:slug` shows event details<br>- Shows available shows<br>- Shows countdown to sale start<br>- "Book Now" button |

- [ ] Create event detail page
- [ ] Fetch event by slug
- [ ] Display shows
- [ ] Show countdown timer
- [ ] Handle "Book Now" click

---

### [ ]  P7-07: Booking Flow - Zone Selection
| Field | Value |
|-------|-------|
| **Description** | Create zone selection page |
| **Technical Context** | `apps/web/app/events/[slug]/booking/page.tsx` |
| **Acceptance Criteria** | - Show available zones<br>- Real-time availability<br>- Quantity selector (max 4)<br>- Price calculation |

- [ ] Create zone selection page
- [ ] Fetch zones with availability
- [ ] Implement quantity selector
- [ ] Show total price
- [ ] Handle "Reserve" click

---

### [ ]  P7-08: Booking Flow - Payment
| Field | Value |
|-------|-------|
| **Description** | Create payment page |
| **Technical Context** | `apps/web/app/events/[slug]/payment/page.tsx` |
| **Acceptance Criteria** | - Show order summary<br>- Countdown timer (10 min)<br>- Payment form (mock)<br>- Handle timeout |

- [ ] Create payment page
- [ ] Show order summary
- [ ] Implement countdown timer
- [ ] Create payment form
- [ ] Handle payment submission
- [ ] Handle timeout redirect

---

### [ ]  P7-09: Booking Confirmation
| Field | Value |
|-------|-------|
| **Description** | Create confirmation page |
| **Technical Context** | `apps/web/app/bookings/[id]/page.tsx` |
| **Acceptance Criteria** | - Show booking details<br>- Show E-Ticket (QR code)<br>- Download option |

- [ ] Create confirmation page
- [ ] Display booking details
- [ ] Generate QR code
- [ ] Add download button

---

### [ ]  P7-10: Virtual Queue UI
| Field | Value |
|-------|-------|
| **Description** | Create queue waiting room page |
| **Technical Context** | `apps/web/app/queue/page.tsx` |
| **Acceptance Criteria** | - Show queue position<br>- Show estimated wait time<br>- Auto-refresh status<br>- Auto-redirect when ready |

- [ ] Create queue page
- [ ] Display position
- [ ] Display estimated wait
- [ ] Poll for status updates
- [ ] Auto-redirect on ready

---

### [ ]  P7-11: User Dashboard
| Field | Value |
|-------|-------|
| **Description** | Create user dashboard pages |
| **Technical Context** | `apps/web/app/bookings/` |
| **Acceptance Criteria** | - Booking history list<br>- Pending bookings with "Resume Payment"<br>- Profile settings |

- [ ] Create booking history page
- [ ] Show pending bookings with resume option
- [ ] Create profile settings page
- [ ] Handle profile update

---

**Phase 7 Milestone:** Users can browse, book, and pay through web UI

---

## Phase 8: Observability

**Goal:** Production-grade monitoring with unified OTel stack

### [ ]  P8-01: OpenTelemetry Package
| Field | Value |
|-------|-------|
| **Description** | Create shared OTel package |
| **Technical Context** | `pkg/telemetry/` |
| **Acceptance Criteria** | - OTel SDK initialization<br>- OTLP exporter configured<br>- Tracer and Meter providers setup |

- [ ] Create telemetry package
- [ ] Setup TracerProvider
- [ ] Setup MeterProvider
- [ ] Configure OTLP exporters
- [ ] Add resource attributes
- [ ] Write tests

---

### [ ]  P8-02: Service Instrumentation
| Field | Value |
|-------|-------|
| **Description** | Add OTel instrumentation to all services |
| **Technical Context** | All Go services |
| **Acceptance Criteria** | - Gin middleware (otelgin)<br>- Redis instrumentation<br>- PostgreSQL instrumentation<br>- HTTP client instrumentation |

- [ ] Add otelgin middleware
- [ ] Add otelredis instrumentation
- [ ] Add otelsql instrumentation
- [ ] Add otelhttp transport
- [ ] Add Kafka span injection/extraction
- [ ] Verify traces in Jaeger

---

### [ ]  P8-03: Custom Metrics
| Field | Value |
|-------|-------|
| **Description** | Implement custom business metrics |
| **Technical Context** | All Go services |
| **Acceptance Criteria** | - booking_reservations_total<br>- booking_reservation_failures<br>- active_reservations<br>- kafka_consumer_lag |

- [ ] Create Counter for reservations
- [ ] Create Counter for failures
- [ ] Create Gauge for active reservations
- [ ] Create Gauge for Kafka lag
- [ ] Create Histogram for latencies
- [ ] Verify metrics in Prometheus

---

### [ ]  P8-04: Log-Trace Correlation
| Field | Value |
|-------|-------|
| **Description** | Add trace context to logs |
| **Technical Context** | `pkg/logger/` |
| **Acceptance Criteria** | - trace_id in all logs<br>- span_id in all logs<br>- Logs export to Loki<br>- Click-through from log to trace |

- [ ] Inject trace context into Zap
- [ ] Configure Loki export
- [ ] Verify in Grafana
- [ ] Test log-trace linking

---

### [ ]  P8-05: Infrastructure - OTel Collector
| Field | Value |
|-------|-------|
| **Description** | Setup OTel Collector in Docker Compose |
| **Technical Context** | `infra/otel/`, `docker-compose.yml` |
| **Acceptance Criteria** | - OTLP receivers (gRPC + HTTP)<br>- Export to Jaeger, Prometheus, Loki<br>- Processing pipelines configured |

- [ ] Create otel-collector-config.yaml
- [ ] Add to docker-compose.yml
- [ ] Configure receivers
- [ ] Configure processors (batch, memory_limiter)
- [ ] Configure exporters
- [ ] Test pipeline

---

### [ ]  P8-06: Infrastructure - Observability Stack
| Field | Value |
|-------|-------|
| **Description** | Setup Jaeger, Prometheus, Loki, Grafana |
| **Technical Context** | `docker-compose.yml` |
| **Acceptance Criteria** | - All services running<br>- Grafana accessible on :3000<br>- Data sources configured |

- [ ] Add Jaeger to docker-compose
- [ ] Add Prometheus to docker-compose
- [ ] Add Loki to docker-compose
- [ ] Add Grafana to docker-compose
- [ ] Configure Grafana data sources
- [ ] Verify all connections

---

### [ ]  P8-07: Grafana Dashboards
| Field | Value |
|-------|-------|
| **Description** | Create Grafana dashboards |
| **Technical Context** | `infra/grafana/dashboards/` |
| **Acceptance Criteria** | - System Overview dashboard<br>- Booking dashboard<br>- Payment dashboard<br>- Infrastructure dashboard |

- [ ] Create System Overview dashboard
- [ ] Create Booking dashboard
- [ ] Create Payment dashboard
- [ ] Create Infrastructure dashboard
- [ ] Export as JSON for provisioning

---

### [ ]  P8-08: Alerting Rules
| Field | Value |
|-------|-------|
| **Description** | Configure alerting rules |
| **Technical Context** | `infra/prometheus/alerts.yml` |
| **Acceptance Criteria** | - High error rate alert<br>- High latency alert<br>- Service down alert<br>- Kafka lag alert |

- [ ] Create alert rules file
- [ ] Add error rate alert (> 1% for 5 min)
- [ ] Add latency alert (P99 > 500ms)
- [ ] Add service down alert
- [ ] Add Kafka lag alert
- [ ] Test alerts fire correctly

---

**Phase 8 Milestone:** Full observability with OTel - traces, metrics, logs unified in Grafana

---

## Phase 9: Production Hardening

**Goal:** Ready for production deployment

### [ ]  P9-01: Security Audit
| Field | Value |
|-------|-------|
| **Description** | Comprehensive security review |
| **Technical Context** | All services |
| **Acceptance Criteria** | - No SQL injection vulnerabilities<br>- No XSS vulnerabilities<br>- All endpoints properly authorized<br>- Security headers configured |

- [ ] Review all endpoints for auth/authz
- [ ] Test for SQL injection
- [ ] Test for XSS
- [ ] Verify HTTPS configuration
- [ ] Add security headers
- [ ] Review rate limiting
- [ ] Document findings and fixes

---

### [ ]  P9-02: Performance Profiling
| Field | Value |
|-------|-------|
| **Description** | Profile and optimize hot paths |
| **Technical Context** | All Go services |
| **Acceptance Criteria** | - Identified and fixed bottlenecks<br>- Connection pools optimized<br>- Memory usage acceptable<br>- No goroutine leaks |

- [ ] Profile with pprof
- [ ] Identify hot paths
- [ ] Optimize connection pools
- [ ] Check for goroutine leaks
- [ ] Optimize memory usage
- [ ] Document optimizations

---

### [ ]  P9-03: End-to-End Load Test
| Field | Value |
|-------|-------|
| **Description** | Full system load test |
| **Technical Context** | `tests/load/` |
| **Acceptance Criteria** | - 10k RPS confirmed<br>- P99 latency < 50ms (server)<br>- Zero overselling<br>- Error rate < 0.1% |

- [ ] Create E2E load test scenarios
- [ ] Test full booking flow under load
- [ ] Test payment processing under load
- [ ] Test virtual queue under load
- [ ] Document final results
- [ ] Fix any issues found

---

### [ ]  P9-04: API Documentation
| Field | Value |
|-------|-------|
| **Description** | Create OpenAPI/Swagger documentation |
| **Technical Context** | All APIs |
| **Acceptance Criteria** | - All endpoints documented<br>- Request/response schemas<br>- Authentication documented<br>- Error codes documented |

- [ ] Generate OpenAPI spec
- [ ] Document all endpoints
- [ ] Add request/response examples
- [ ] Document authentication
- [ ] Document error codes
- [ ] Setup Swagger UI

---

### [ ]  P9-05: Operations Runbook
| Field | Value |
|-------|-------|
| **Description** | Create runbook for operations |
| **Technical Context** | `docs/runbook.md` |
| **Acceptance Criteria** | - Deployment procedures<br>- Rollback procedures<br>- Incident response<br>- Common issues and solutions |

- [ ] Document deployment procedures
- [ ] Document rollback procedures
- [ ] Create incident response guide
- [ ] Document common issues
- [ ] Document monitoring/alerting

---

### [ ]  P9-06: Production Deployment
| Field | Value |
|-------|-------|
| **Description** | Deploy to production environment |
| **Technical Context** | Docker images, infrastructure |
| **Acceptance Criteria** | - All services deployed<br>- Database migrations applied<br>- Monitoring active<br>- System operational |

- [ ] Build production Docker images
- [ ] Setup production environment
- [ ] Configure environment variables
- [ ] Run database migrations
- [ ] Deploy services
- [ ] Verify health checks
- [ ] Configure monitoring alerts
- [ ] Final smoke test

---

**Phase 9 Milestone:** System deployed and running in production

---

## Progress Summary

| Phase | Tasks | Completed |
|-------|-------|-----------|
| Phase 1: Foundation | 13 | 0 |
| Phase 2: Core Booking | 14 | 0 |
| Phase 3: Auth & Events | 14 | 0 |
| Phase 4: Payment & Saga | 12 | 0 |
| Phase 5: NestJS Services | 8 | 0 |
| Phase 6: Virtual Queue | 7 | 0 |
| Phase 7: Frontend | 11 | 0 |
| Phase 8: Observability | 8 | 0 |
| Phase 9: Production | 6 | 0 |
| **Total** | **93** | **0** |

---

## References
- [Product Specification](./01-spec.md)
- [Technical Plan](./02-plan.md)
