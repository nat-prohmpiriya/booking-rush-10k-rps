# Project Specification: Booking Rush (10k RPS)

> **Version:** 2.0 (Draft)
> **Last Updated:** 2025-12-03
> **Status:** Draft - Pending Review

---

## Table of Contents
1. [Project Overview](#1-project-overview)
2. [Business Requirements](#2-business-requirements)
3. [Technical Architecture](#3-technical-architecture)
4. [Service Specifications](#4-service-specifications)
5. [Database Design](#5-database-design)
6. [API Specifications](#6-api-specifications)
7. [Non-Functional Requirements](#7-non-functional-requirements)
8. [Security Requirements](#8-security-requirements)
9. [Observability & Monitoring](#9-observability--monitoring)
10. [Failure Scenarios & Recovery](#10-failure-scenarios--recovery)
11. [Development Phases](#11-development-phases)

---

## 1. Project Overview

### 1.1 Problem Statement
ระบบจองตั๋วคอนเสิร์ตในช่วง Flash Sale มักเผชิญปัญหา:
- **Race Condition:** หลายพันคนกดจองที่นั่งเดียวกันพร้อมกัน
- **Overselling:** ขายตั๋วเกินจำนวนที่มี
- **System Crash:** ระบบล่มเมื่อ traffic พุ่งสูง
- **Poor UX:** User ติด queue นาน หรือจ่ายเงินแล้วไม่ได้ตั๋ว

### 1.2 Solution
**Booking Rush** - High-Concurrency Ticket Booking System ที่ออกแบบมาเพื่อ:
- รองรับ **10,000 Requests Per Second (RPS)**
- **Zero Overselling** ด้วย Atomic Operations
- **Eventual Consistency** - ถ้า Payment ล่ม การจองไม่หาย
- **Horizontal Scalability** - Scale เฉพาะ service ที่โหลดหนัก

### 1.3 Target Users
| User Type | Description |
|-----------|-------------|
| **End User** | ผู้ซื้อบัตรคอนเสิร์ต |
| **Event Organizer** | ผู้จัดงาน สร้าง/จัดการ Event |
| **System Admin** | ดูแลระบบ, Monitor, จัดการ Tenant |

### 1.4 Success Metrics
| Metric | Target | Note |
|--------|--------|------|
| Throughput | ≥ 10,000 RPS | Booking endpoint |
| Latency P50 (Server) | < 20ms | Server processing time only |
| Latency P99 (Server) | < 50ms | Server processing time only |
| Latency P99 (E2E) | < 200ms | Including network latency |
| Error Rate | < 0.1% | Non-5xx errors |
| Availability | 99.9% | 8.76 hrs downtime/year |
| Overselling Rate | 0% | Zero tolerance |

> **Note:** Latency แยกเป็น 2 แบบ: Server Processing Time (วัดใน application) และ End-to-End (วัดจาก client) เพื่อให้ realistic ภายใต้ 10k RPS

---

## 2. Business Requirements

### 2.1 Core Features

#### 2.1.1 User Management
- [ ] Register with email/password
- [ ] Login with JWT authentication
- [ ] OAuth2 (Google, Facebook) - Phase 2
- [ ] Profile management
- [ ] Booking history

#### 2.1.2 Event Management
- [ ] Create/Edit/Delete events (Organizer)
- [ ] Event categories (Concert, Sport, Theater)
- [ ] Multiple show times per event
- [ ] Seat mapping with zones/sections
- [ ] Pricing tiers (VIP, Regular, etc.)

#### 2.1.3 Booking Flow
```
[Browse Events] → [Select Show] → [Select Seats] → [Reserve (Lock)]
    → [Payment] → [Confirm] → [E-Ticket]
```

#### 2.1.4 Virtual Queue (Waiting Room)
- [ ] เมื่อ traffic สูงกว่า threshold, user เข้า waiting room
- [ ] แสดง estimated wait time
- [ ] ปล่อย user เข้าระบบทีละ batch
- [ ] Queue position tracking
- [ ] **Queue Bypass Token** - User ที่ผ่าน queue แล้วได้ signed token

> **Queue Bypass Mechanism:**
> - User ถึงคิว → ได้ Queue Pass Token (JWT, valid 5 min)
> - ส่ง token ใน header `X-Queue-Pass`
> - API Gateway validate แล้ว bypass rate limit
> - ป้องกัน User ผ่านคิวแล้วโดน rate limit, ป้องกัน User ใหม่แทรกคิว

#### 2.1.5 Payment
- [ ] Multiple payment methods (Credit Card, Bank Transfer, E-Wallet)
- [ ] Payment timeout: 10 minutes
- [ ] Auto-release seats if payment fails/timeout
- [ ] Refund flow (request → approve → process)

#### 2.1.6 Notifications
- [ ] Email confirmation after booking
- [ ] SMS reminder before event (optional)
- [ ] Push notification for queue status

#### 2.1.7 Admin Dashboard
- [ ] Real-time sales monitoring
- [ ] Event analytics (views, conversions, revenue)
- [ ] User management
- [ ] System health monitoring

### 2.2 Business Rules

#### 2.2.1 Booking Rules
| Rule | Value | Configurable |
|------|-------|--------------|
| Max tickets per user per event | 4 | Yes (per event) |
| Seat reservation timeout | 10 minutes | Yes (global) |
| Booking window before event | 1 hour | Yes (per event) |
| Minimum age for certain events | 18+ | Yes (per event) |

#### 2.2.2 Pricing Rules
- Dynamic pricing based on demand (optional, Phase 2)
- Early bird discounts
- Promo codes / Vouchers
- Bundle pricing (buy 3 get 10% off)

#### 2.2.3 Refund Rules
| Condition | Refund % |
|-----------|----------|
| > 7 days before event | 100% |
| 3-7 days before event | 50% |
| < 3 days before event | 0% |
| Event cancelled by organizer | 100% + compensation |

### 2.3 Multi-Tenant Support
- แต่ละ Organizer เป็น Tenant แยก
- Data isolation per tenant
- Custom branding per tenant (logo, colors)
- Separate analytics per tenant

---

## 3. Technical Architecture

### 3.1 Architecture Overview
```
                                    ┌─────────────────┐
                                    │   Frontend      │
                                    │   (Next.js)     │
                                    └────────┬────────┘
                                             │
                                    ┌────────▼────────┐
                                    │   API Gateway   │
                                    │   (Go + Gin)    │
                                    │   Rate Limit    │
                                    └────────┬────────┘
                                             │
        ┌────────────────┬───────────────────┼───────────────────┬────────────────┐
        │                │                   │                   │                │
┌───────▼──────┐ ┌───────▼──────┐ ┌──────────▼─────────┐ ┌───────▼──────┐ ┌───────▼──────┐
│ Auth Service │ │Ticket Service│ │  Booking Service   │ │Payment Service│ │Notification  │
│              │ │              │ │      (CORE)        │ │              │ │   Service    │
└───────┬──────┘ └───────┬──────┘ └──────────┬─────────┘ └───────┬──────┘ └───────┬──────┘
        │                │                   │                   │                │
        │                │           ┌───────▼───────┐           │                │
        │                │           │  Redis (Lua)  │           │                │
        │                │           │  Atomic Lock  │           │                │
        │                │           └───────────────┘           │                │
        │                │                   │                   │                │
        │                │           ┌───────▼───────┐           │                │
        │                │           │     Kafka     │◄──────────┘                │
        │                │           │  Event Queue  │────────────────────────────┘
        │                │           └───────────────┘
        │                │                   │
        └────────────────┴───────────────────┼───────────────────────────────────────┘
                                             │
                                    ┌────────▼────────┐
                                    │   PostgreSQL    │
                                    │   (Primary DB)  │
                                    └─────────────────┘
```

### 3.2 Technology Stack

| Layer | Technology | Justification |
|-------|------------|---------------|
| **Frontend** | Next.js 15, TailwindCSS, Shadcn UI | Modern, SSR support, great DX |
| **API Gateway** | Go + Gin | High performance, built-in middleware |
| **Backend** | Go + Gin | Fast, efficient concurrency (goroutines) |
| **Database** | PostgreSQL 16 | ACID, proven reliability, good for transactions |
| **Cache/Lock** | Redis 7 + Lua | Atomic operations, sub-ms latency |
| **Message Queue** | Kafka | High throughput, durable, replayable |
| **Container** | Docker, Docker Compose | Easy local development |
| **Monitoring** | Prometheus, Grafana, Jaeger | Industry standard observability |

### 3.3 Project Structure (Monorepo)
```
booking-rush-10k-rps/
├── apps/
│   ├── api-gateway/          # Entry point, routing, rate limiting
│   ├── auth-service/         # User auth, JWT
│   ├── ticket-service/       # Event & seat management
│   ├── booking-service/      # Core booking logic
│   ├── payment-service/      # Payment processing
│   ├── notification-service/ # Email, SMS, Push
│   └── frontend/             # Next.js app
├── pkg/                      # Shared Go packages
│   ├── config/               # Configuration loader
│   ├── logger/               # Structured logging
│   ├── middleware/           # Common middlewares
│   ├── response/             # Standard API responses
│   ├── errors/               # Custom error types
│   ├── kafka/                # Kafka client wrapper
│   ├── redis/                # Redis client wrapper
│   └── database/             # DB connection & migrations
├── scripts/                  # Utility scripts
│   ├── lua/                  # Redis Lua scripts
│   └── migrations/           # DB migrations
├── deployments/
│   └── docker/               # Docker Compose files
├── docs/                     # Documentation
│   ├── architecture/         # Architecture diagrams
│   └── api/                  # API documentation
├── tests/
│   ├── load/                 # k6 load tests
│   └── e2e/                  # End-to-end tests
├── go.work                   # Go workspace
├── go.work.sum
├── docker-compose.yml
├── docker-compose.prod.yml
├── Makefile
└── README.md
```

---

## 4. Service Specifications

### 4.1 API Gateway
**Responsibility:** Single entry point, routing, cross-cutting concerns

| Feature | Description |
|---------|-------------|
| Routing | Route requests to appropriate services |
| Rate Limiting | Token bucket algorithm, 100 req/min per user |
| Authentication | Validate JWT, inject user context |
| Request ID | Generate unique request ID for tracing |
| CORS | Handle cross-origin requests |
| Request/Response Logging | Log all requests for audit |

**Endpoints:**
- `POST /api/v1/auth/*` → Auth Service
- `GET /api/v1/events/*` → Ticket Service
- `POST /api/v1/bookings/*` → Booking Service
- `POST /api/v1/payments/*` → Payment Service

### 4.2 Auth Service
**Responsibility:** User authentication & authorization

| Feature | Description |
|---------|-------------|
| Register | Email + password, validation |
| Login | Return JWT (access + refresh token) |
| Refresh Token | Issue new access token |
| Password Reset | Email-based reset flow |
| Role Management | User, Organizer, Admin |

**JWT Claims:** sub (user_uuid), email, role, tenant_id, exp, iat

**Token Expiry:** Access Token 15 min, Refresh Token 7 days

### 4.3 Ticket Service
**Responsibility:** Event & seat catalog management (Read-heavy)

| Feature | Description |
|---------|-------------|
| Event CRUD | Create, Read, Update, Delete events |
| Show Management | Multiple shows per event |
| Seat Mapping | Zones, sections, rows, seats |
| Inventory Sync | Sync seat availability to Redis |
| Caching | Cache event data in Redis (TTL: 5 min) |

**Caching Strategy:**
- Event list: Cached, invalidate on update
- Event detail: Cached, invalidate on update
- Seat availability: Real-time from Redis (source of truth for available count)

**Redis-SQL Sync Strategy (Inventory):**
> ⚠️ **Critical Design Decision**

Redis เป็น Source of Truth สำหรับ `available_seats` ระหว่าง Flash Sale:

| Data | Source of Truth | Sync Strategy |
|------|-----------------|---------------|
| Available seats (real-time) | **Redis** | - |
| Available seats (display) | Redis | Read from Redis directly |
| Available seats (DB) | PostgreSQL | Async sync every 5 seconds via Kafka consumer |

**Sync Flow:**
1. Booking Service: Deduct from Redis (Lua) → Produce Kafka event
2. Inventory Sync Worker: Consume events → Batch update PostgreSQL every 5 sec

**Why this approach:**
- ไม่ Write ลง DB ทุก request (ป้องกัน DB bottleneck)
- หน้า Event List อ่านจาก Redis (fast)
- DB เก็บข้อมูลสำหรับ reporting/analytics (eventual consistency OK)
- ถ้า Redis ล่ม สามารถ rebuild จาก DB ได้

### 4.4 Booking Service ⭐ (CORE)
**Responsibility:** Handle high-concurrency seat reservation

| Feature | Description |
|---------|-------------|
| Reserve Seats | Atomic seat lock via Redis Lua |
| Release Seats | Auto-release on timeout/cancel |
| Order Creation | Create booking order, produce Kafka event |
| Idempotency | Prevent duplicate bookings |

**Redis Lua Script:**
> Implementation: `scripts/lua/reserve_seats.lua`

Script ต้องทำ:
1. Check seat availability
2. Check user max limit per event
3. Atomic deduction (DECRBY)
4. Set reservation TTL
5. Return success/error with remaining count

**Kafka Events Produced:**
- `booking.created` - When reservation is made
- `booking.confirmed` - When payment succeeds
- `booking.cancelled` - When booking is cancelled
- `booking.expired` - When reservation times out

### 4.5 Payment Service
**Responsibility:** Process payments, update booking status

| Feature | Description |
|---------|-------------|
| Kafka Consumer | Listen to `booking.created` events |
| Payment Processing | Integrate with payment gateway (Mock) |
| Status Update | Update booking status in DB |
| Retry Logic | Retry failed payments with backoff |
| Produce Events | `payment.success`, `payment.failed` |

**Payment States:** PENDING → PROCESSING → SUCCESS / FAILED → RETRY / TIMEOUT → REFUND

### 4.6 Notification Service
**Responsibility:** Send notifications to users

| Feature | Description |
|---------|-------------|
| Kafka Consumer | Listen to booking/payment events |
| Email | Booking confirmation, e-ticket |
| SMS | Event reminder (optional) |
| Push | Queue status, booking updates |

**Templates:**
- Booking Confirmation
- Payment Success
- Payment Failed
- Event Reminder
- Refund Processed

---

## 5. Database Design

### 5.1 Entity Relationship Diagram
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   tenants   │────<│    users    │     │  categories │
└─────────────┘     └──────┬──────┘     └──────┬──────┘
                           │                   │
                    ┌──────┴──────┐            │
                    │             │            │
              ┌─────▼─────┐ ┌─────▼─────┐ ┌────▼─────┐
              │  bookings │ │  events   │─┤          │
              └─────┬─────┘ └─────┬─────┘ └──────────┘
                    │             │
              ┌─────▼─────┐ ┌─────▼─────┐
              │ payments  │ │   shows   │
              └───────────┘ └─────┬─────┘
                                  │
                            ┌─────▼─────┐
                            │   seats   │
                            └───────────┘
```

### 5.2 Table Definitions
> Implementation: `scripts/migrations/`

#### Tables Overview
| Table | Key Fields | Notes |
|-------|------------|-------|
| `tenants` | id, name, slug, settings, status | Multi-tenant support |
| `users` | id, tenant_id, email, password_hash, role | Roles: user, organizer, admin |
| `categories` | id, name, slug | Event categories |
| `events` | id, tenant_id, organizer_id, title, status | Status: draft, published, cancelled |
| `shows` | id, event_id, show_date, total_seats, available_seats | Multiple shows per event |
| `seat_zones` | id, show_id, name, price, total_seats | VIP, Regular, etc. |
| `bookings` | id, reservation_id, user_id, show_id, status | Status: reserved, confirmed, cancelled, expired |
| `payments` | id, booking_id, amount, status | Status: pending, processing, success, failed |
| `audit_logs` | id, action, entity_type, entity_id, created_at | Partitioned by month |

#### Key Design Decisions:

| Table | Decision | Reason |
|-------|----------|--------|
| `bookings.reservation_id` | Unique idempotency key | Prevent duplicate bookings |
| `bookings.expires_at` | Index with WHERE clause | Fast expiry scan |
| `audit_logs` | Partitioned by month | High-growth table, easy retention |
| All tables | UUID primary keys | Distributed-friendly |
| All tables | `created_at`, `updated_at` | Audit trail |

#### Audit Logs Partitioning:
> ⚠️ **High-Growth Table**

| Aspect | Decision |
|--------|----------|
| Partition Key | `created_at` (monthly) |
| Retention | 3 years (drop old partitions) |
| Archive | Move to cold storage (S3) before drop |
| Alternative | ใช้ Elasticsearch/MongoDB สำหรับ Production |

**Why Partitioning:**
- 10k RPS × logging = ล้าน rows/day
- Query เร็วขึ้นเพราะ scan แค่ partition ที่ต้องการ
- Drop partition ง่ายกว่า DELETE (no vacuum)

### 5.3 Redis Data Structures

| Key Pattern | Type | Description | TTL |
|-------------|------|-------------|-----|
| `event:{id}:show:{id}:zone:{id}:available` | String | Available seat count | - |
| `reservation:{id}` | String | Reservation quantity | 10 min |
| `user:{id}:event:{id}:reserved` | String | User's reserved count for event | 10 min |
| `event:{id}` | Hash | Event cache | 5 min |
| `rate_limit:user:{id}` | String | Rate limit counter | 1 min |
| `queue:event:{id}` | Sorted Set | Virtual queue | - |

---

## 6. API Specifications

### 6.1 API Standards
- **Base URL:** `/api/v1`
- **Authentication:** Bearer JWT in `Authorization` header
- **Content-Type:** `application/json`
- **Request ID:** `X-Request-ID` header for tracing

### 6.2 Response Format
> Implementation: `pkg/response/`

| Type | Fields |
|------|--------|
| **Success** | `success: true`, `data: {...}`, `meta: {page, per_page, total}` |
| **Error** | `success: false`, `error: {code, message, details}` |

### 6.3 Endpoints

#### Auth Service
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/auth/register` | Register new user | - |
| POST | `/auth/login` | Login, get tokens | - |
| POST | `/auth/refresh` | Refresh access token | - |
| POST | `/auth/logout` | Invalidate refresh token | ✓ |
| POST | `/auth/forgot-password` | Request password reset | - |
| POST | `/auth/reset-password` | Reset password | - |
| GET | `/auth/me` | Get current user | ✓ |
| PUT | `/auth/me` | Update profile | ✓ |

#### Ticket Service
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/events` | List events (paginated) | - |
| GET | `/events/:id` | Get event details | - |
| GET | `/events/:id/shows` | Get shows for event | - |
| GET | `/shows/:id/zones` | Get zones & availability | - |
| POST | `/events` | Create event | ✓ Organizer |
| PUT | `/events/:id` | Update event | ✓ Organizer |
| DELETE | `/events/:id` | Delete event | ✓ Organizer |

#### Booking Service
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/bookings/reserve` | Reserve seats | ✓ |
| POST | `/bookings/:id/confirm` | Confirm booking | ✓ |
| POST | `/bookings/:id/cancel` | Cancel booking | ✓ |
| GET | `/bookings` | Get user's bookings | ✓ |
| GET | `/bookings/:id` | Get booking details | ✓ |
| **GET** | **`/bookings/pending`** | **Get pending reservations (resume payment)** | ✓ |
| GET | `/queue/status` | Get queue position | ✓ |
| POST | `/queue/join` | Join virtual queue | ✓ |

> **`/bookings/pending`:** ใช้สำหรับ User ที่จองได้แล้วแต่เน็ตหลุด ให้กลับมา resume การจ่ายเงินได้ภายใน 10 นาที

#### Payment Service
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/payments` | Process payment | ✓ |
| GET | `/payments/:id` | Get payment status | ✓ |
| POST | `/payments/:id/refund` | Request refund | ✓ |

### 6.4 Error Codes
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

## 7. Non-Functional Requirements

### 7.1 Performance
| Metric | Requirement |
|--------|-------------|
| Throughput | ≥ 10,000 RPS (booking endpoint) |
| Latency P50 | < 50ms |
| Latency P99 | < 100ms |
| Latency P999 | < 500ms |
| Concurrent Users | 50,000+ |

### 7.2 Scalability
> ⚠️ **Scaling Triggers แยกตาม Service Type**

| Service | Type | Scale Trigger | Metric |
|---------|------|---------------|--------|
| API Gateway | API | CPU/Memory | CPU > 70% |
| Auth Service | API | CPU/Memory | CPU > 70% |
| Ticket Service | API | CPU/Memory | CPU > 70% |
| **Booking Service** | API | CPU + Request Rate | CPU > 60% OR req/s > 5000 |
| **Payment Service** | Worker | Kafka Lag | Lag > 1000 messages |
| **Notification Service** | Worker | Kafka Lag | Lag > 5000 messages |
| Inventory Sync Worker | Worker | Kafka Lag | Lag > 500 messages |

**Why different triggers:**
- **API Services (Booking):** รับ HTTP request โดยตรง → Scale ตาม CPU/Request rate
- **Worker Services (Payment):** Consume จาก Kafka → Scale ตาม Queue depth (Kafka Lag)

**Scaling Constraints:**
- Min replicas: 2 (HA)
- Max replicas: 10 (cost control)
- Cool-down period: 60 seconds

**Additional:**
- Database read replicas for read-heavy operations
- Redis: Start single node, migration path to cluster documented

### 7.3 Availability
| Tier | Target | Downtime/Year |
|------|--------|---------------|
| Production | 99.9% | 8.76 hours |

- Graceful degradation when dependent services fail
- Circuit breaker pattern for external services
- Health checks for all services

### 7.4 Data Retention
| Data Type | Retention |
|-----------|-----------|
| Booking Records | 7 years (legal requirement) |
| Audit Logs | 3 years |
| Event Data | Indefinite |
| User Sessions | 30 days |
| Analytics Data | 2 years |

### 7.5 Disaster Recovery
| Metric | Target |
|--------|--------|
| RPO (Recovery Point Objective) | 1 hour |
| RTO (Recovery Time Objective) | 4 hours |

- Database: Daily backups, WAL archiving
- Redis: RDB snapshots every hour
- Kafka: Replication factor 3

---

## 8. Security Requirements

### 8.1 Authentication & Authorization
- [x] JWT-based authentication
- [x] Role-Based Access Control (RBAC)
- [x] Refresh token rotation
- [ ] OAuth2 integration (Phase 2)
- [x] Password hashing with bcrypt (cost 12)

### 8.2 Data Protection
- [x] All data encrypted at rest (PostgreSQL encryption)
- [x] All traffic over HTTPS/TLS 1.3
- [x] Sensitive data masking in logs
- [x] PII data encryption in database

### 8.3 Input Validation
- [x] Request payload validation
- [x] SQL injection prevention (parameterized queries)
- [x] XSS prevention (output encoding)
- [x] CSRF protection (SameSite cookies)

### 8.4 Rate Limiting
> ⚠️ **ใช้ Token Bucket Algorithm พร้อม Burst Support**

| Endpoint | Rate | Burst | Window | Note |
|----------|------|-------|--------|------|
| `/auth/login` | 5 req | 3 | 1 min | ป้องกัน brute force |
| `/auth/register` | 3 req | 2 | 1 min | ป้องกัน spam |
| `/bookings/reserve` | 20 req | 10 | 1 min | ยอมให้ burst 10 req แรกทันที |
| General API | 100 req | 20 | 1 min | - |

**Token Bucket Explanation:**
- **Rate:** จำนวน token ที่เติมต่อ window
- **Burst:** จำนวน token สูงสุดที่เก็บได้ (ยอมให้ใช้รวดเดียว)

**Example:** `/bookings/reserve` (Rate: 20, Burst: 10)
- User เข้ามาครั้งแรก → มี 10 tokens พร้อมใช้ทันที
- กดจอง 5 ครั้งรวด → เหลือ 5 tokens
- รอ 30 วิ → เติม 10 tokens → มี 10 tokens (capped at burst)

**Why Burst is important:**
- User ตื่นเต้น กดรัวๆ ตอน Flash Sale เปิด
- เน็ตกระตุก กด retry หลายครั้ง
- ป้องกัน 429 Error ที่ทำให้ UX แย่

### 8.5 Audit Logging
ทุก action ที่สำคัญต้อง log:
- User login/logout
- Booking created/confirmed/cancelled
- Payment processed
- Admin actions
- Data modifications

### 8.6 Security Headers
```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
```

---

## 9. Observability & Monitoring

### 9.0 OpenTelemetry (OTel) - Unified Observability
> ⚠️ **Core Decision: ใช้ OpenTelemetry เป็น standard สำหรับ Traces, Metrics, Logs**

**Why OpenTelemetry:**
- CNCF project, vendor-neutral, industry standard
- Unified SDK สำหรับ Traces, Metrics, Logs
- Flexible export ไปยัง backend ใดก็ได้
- Go SDK มี official support ดีมาก

**Architecture:**
```
┌─────────────────────────────────────────────────────────────┐
│                      Go Services                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │API Gateway  │ │Booking Svc  │ │Payment Svc  │  ...      │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘           │
│         │               │               │                   │
│         └───────────────┼───────────────┘                   │
│                         │                                   │
│              ┌──────────▼──────────┐                        │
│              │   OTel Go SDK       │                        │
│              │ (Traces+Metrics+Logs)│                        │
│              └──────────┬──────────┘                        │
└─────────────────────────┼───────────────────────────────────┘
                          │ OTLP (gRPC/HTTP)
                          ▼
              ┌───────────────────────┐
              │   OTel Collector      │
              │   (Process, Filter,   │
              │    Batch, Export)     │
              └───────────┬───────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│    Jaeger     │ │  Prometheus   │ │     Loki      │
│   (Traces)    │ │  (Metrics)    │ │    (Logs)     │
└───────┬───────┘ └───────┬───────┘ └───────┬───────┘
        │                 │                 │
        └─────────────────┼─────────────────┘
                          ▼
              ┌───────────────────────┐
              │       Grafana         │
              │  (Unified Dashboard)  │
              └───────────────────────┘
```

**OTel Stack Components:**

| Component | Technology | Purpose |
|-----------|------------|---------|
| SDK | `go.opentelemetry.io/otel` | Instrumentation in Go services |
| Collector | `otel/opentelemetry-collector` | Central pipeline for telemetry |
| Traces Backend | Jaeger | Store and query traces |
| Metrics Backend | Prometheus | Store and query metrics |
| Logs Backend | Loki | Store and query logs |
| Visualization | Grafana | Unified dashboards |

**Go Libraries:**
> Implementation: `pkg/telemetry/`

| Category | Libraries |
|----------|-----------|
| Core OTel | `go.opentelemetry.io/otel`, `otel/sdk` |
| Exporters | `otlp/otlptrace/otlptracegrpc`, `otlp/otlpmetric/otlpmetricgrpc` |
| Auto-instrumentation | `otelgin`, `otelhttp`, `otelgrpc` |
| Database & Redis | `otelredis`, `otelsql` |

### 9.1 Logging
- **Format:** Structured JSON
- **Levels:** DEBUG, INFO, WARN, ERROR
- **Fields:** timestamp, level, service, trace_id, span_id, request_id, user_id, message
- **Library:** Zap or Zerolog with OTel bridge
- **Export:** OTel Collector → Loki

> **Log-Trace Correlation:** ใส่ `trace_id` และ `span_id` ใน log ทำให้ click จาก log ไปดู trace ใน Grafana ได้

### 9.2 Metrics (via OTel → Prometheus)
| Metric | Type | Description |
|--------|------|-------------|
| `http_requests_total` | Counter | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | Request latency |
| `booking_reservations_total` | Counter | Total reservations |
| `booking_reservation_failures` | Counter | Failed reservations |
| `redis_operation_duration_seconds` | Histogram | Redis latency |
| `kafka_consumer_lag` | Gauge | Consumer lag |
| `active_reservations` | Gauge | Current active reservations |
| `db_connections_active` | Gauge | Active DB connections |

### 9.3 Tracing (via OTel → Jaeger)
- **Protocol:** OTLP over gRPC to OTel Collector
- **Sampling:** Dev 100%, Prod 10% (configurable)
- **Context Propagation:** W3C TraceContext (`traceparent` header)

**Instrumented Components:**
| Component | Instrumentation |
|-----------|-----------------|
| Gin HTTP | `otelgin` middleware |
| Redis | `otelredis` wrapper |
| PostgreSQL | `otelsql` wrapper |
| Kafka | Manual span injection/extraction |
| HTTP Client | `otelhttp` transport |

### 9.4 OTel Collector
> Configuration: `deployments/docker/otel-collector-config.yaml`

**Pipeline:**
- **Receivers:** OTLP (gRPC :4317, HTTP :4318)
- **Processors:** batch, memory_limiter
- **Exporters:** Jaeger (traces), Prometheus (metrics), Loki (logs)

### 9.5 Alerting
| Alert | Condition | Severity |
|-------|-----------|----------|
| High Error Rate | > 1% errors in 5 min | Critical |
| High Latency | P99 > 500ms for 5 min | Warning |
| Service Down | Health check fails 3x | Critical |
| Low Stock | Available < 10% total | Warning |
| Queue Backlog | Kafka lag > 10,000 | Warning |
| Redis High Memory | > 80% memory used | Warning |

### 9.5 Dashboards (Grafana)
1. **System Overview:** All services health, request rates
2. **Booking Dashboard:** Reservations/min, success rate, queue depth
3. **Payment Dashboard:** Payment status, failure reasons
4. **Infrastructure:** CPU, Memory, Disk, Network

---

## 10. Failure Scenarios & Recovery

### 10.1 Redis Failure
| Scenario | Impact | Recovery |
|----------|--------|----------|
| Redis down | Cannot make new reservations | Failover to replica (if exists), or return "Service Unavailable" |
| Redis slow | High latency | Alert, investigate, scale |
| Data loss | Inconsistent stock | Rebuild from PostgreSQL |

### 10.2 Kafka Failure
| Scenario | Impact | Recovery |
|----------|--------|----------|
| Kafka down | Orders not processed | Queue in memory, retry when up |
| Consumer lag | Delayed payments | Auto-scale consumers |
| Message loss | Lost orders | Idempotent producers, transactional outbox |

### 10.3 PostgreSQL Failure
| Scenario | Impact | Recovery |
|----------|--------|----------|
| Primary down | Cannot write | Promote replica |
| High load | Slow queries | Read from replicas, optimize queries |
| Data corruption | Data loss | Restore from backup |

### 10.4 Payment Gateway Failure
| Scenario | Impact | Recovery |
|----------|--------|----------|
| Gateway timeout | Payment stuck | Retry with exponential backoff |
| Gateway down | Cannot process payments | Queue payments, extend reservation timeout |

### 10.5 Graceful Degradation
เมื่อระบบโหลดสูง:
1. Enable virtual queue (redirect new users to waiting room)
2. Increase cache TTL
3. Disable non-critical features (recommendations, analytics)
4. Return cached data for read operations

---

## 11. Development Phases

> **Detail:** ดู [02-task.md](./02-task.md) สำหรับ tasks ละเอียด

| Phase | Focus | Key Deliverable |
|-------|-------|-----------------|
| 1 | **Foundation** | Monorepo, Docker, shared packages, migrations |
| 2 | **Core Booking** ⭐ | Redis Lua, 10k RPS achieved, zero overselling |
| 3 | **Auth & Events** | JWT, Rate Limit (Token Bucket + Burst), CRUD |
| 4 | **Payment** | Kafka consumer, Idempotency, Transactional Outbox |
| 5 | **Advanced** | Virtual Queue + Bypass Token, Notifications, Audit Log |
| 6 | **Frontend** | Next.js, Booking flow, Queue UI |
| 7 | **Observability** | OpenTelemetry → Jaeger, Prometheus, Loki, Grafana |
| 8 | **Production** | Security audit, E2E load test, Deploy |

### Milestones
- **Phase 2:** Achieve 10,000 RPS with zero overselling
- **Phase 4:** Complete booking-to-payment flow with consistency
- **Phase 7:** Full observability with log-trace correlation
- **Phase 8:** System deployed and running in production

---

## Appendix

### A. Glossary
| Term | Definition |
|------|------------|
| RPS | Requests Per Second |
| Overselling | ขายตั๋วเกินจำนวนที่มี |
| Race Condition | เมื่อหลาย process แย่งเข้าถึง resource เดียวกัน |
| Eventual Consistency | ข้อมูลจะ consistent ในที่สุด ไม่ใช่ทันที |
| Idempotency | ทำซ้ำกี่ครั้งก็ได้ผลเหมือนกัน |

### B. References
- [Redis Lua Scripting](https://redis.io/docs/manual/programmability/eval-intro/)
- [Kafka Best Practices](https://kafka.apache.org/documentation/)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Clean Architecture in Go](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

### C. Decision Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-12-03 | Use Gin over Fiber | Larger ecosystem, more stable |
| 2025-12-03 | Start with Redis single node | Simpler, can migrate to cluster later |
| 2025-12-03 | Use Docker Compose for deployment | Focus on app, not infra |
| 2025-12-03 | Use segmentio/kafka-go | Simpler API than sarama |
| 2025-12-03 | Use OpenTelemetry | Unified SDK for traces, metrics, logs; vendor-neutral |
