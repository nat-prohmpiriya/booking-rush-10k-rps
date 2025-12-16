# OpenTelemetry Implementation Checklist

## Overview

เอกสารนี้ track งานที่ต้องทำเพื่อ implement OTel (Logs, Traces, Metrics) ให้ครบถ้วน

### Current Status

| Component | Status | Notes |
|-----------|:------:|-------|
| OTLP Infrastructure | ✅ | Collector, sampling, propagation |
| HTTP Middleware | ✅ | Auto trace ทุก request |
| DB Tracing (pgx) | ✅ | Query-level |
| Redis Tracing | ✅ | Command-level |
| Kafka Tracing | ✅ | Message-level |
| Logger + Trace ID | ✅ | `WithContext()` พร้อมใช้ |
| **Manual Spans** | ✅ | Handlers + Services done |
| **Logger with Context** | ✅ | Saga handlers updated (80+ calls) |
| **Business Metrics** | ❌ | ต้องสร้างใหม่ |

---

## Phase 1: Critical Path (booking + payment)

### 1.1 Booking Handlers - Add Spans ✅

**Files:**
- [x] `backend-booking/internal/handler/booking_handler.go`
- [x] `backend-booking/internal/handler/saga_handler.go`
- [x] `backend-booking/internal/handler/queue_handler.go`
- [x] `backend-booking/internal/handler/admin_handler.go`

**Tasks per file:**
- [x] Import `"booking-rush/pkg/telemetry"`
- [x] Add span at start of each handler method
- [x] Add attributes (event_id, user_id, quantity, etc.)
- [x] Use `defer span.End()`
- [x] Set span status on error

**Example:**
```go
func (h *BookingHandler) Reserve(c *gin.Context) {
    ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.reserve")
    defer span.End()

    // ... existing code ...

    span.SetAttributes(
        attribute.String("event_id", req.EventID),
        attribute.String("user_id", userID),
        attribute.Int("quantity", req.Quantity),
    )

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
}
```

---

### 1.2 Booking Services - Add Spans ✅

**Files:**
- [x] `backend-booking/internal/service/booking_service.go`
- [x] `backend-booking/internal/service/saga_service.go`
- [x] `backend-booking/internal/service/queue_service.go`
- [ ] `backend-booking/internal/service/event_publisher.go` (optional - async operations)

**Tasks per file:**
- [x] Add spans for main business methods
- [x] Add attributes for business context
- [x] Record errors with `span.RecordError()`

**Key methods instrumented:**
- `ReserveSeats()` - ✅
- `ConfirmBooking()` - ✅
- `CancelBooking()` - ✅
- `GetBooking()` - ✅
- `GetUserBookings()` - ✅
- `ExpireReservations()` - ✅
- `StartBookingSaga()` - ✅
- `JoinQueue()`, `GetPosition()`, `LeaveQueue()` - ✅

---

### 1.3 Payment Handlers - Add Spans ✅

**Files:**
- [x] `backend-payment/internal/handler/payment_handler.go`
- [ ] `backend-payment/internal/handler/webhook_handler.go` (webhook uses context from Stripe, different pattern)

**Tasks per file:**
- [x] Add span for each handler method
- [x] Add payment-specific attributes (amount, currency, status)

---

### 1.4 Payment Services - Add Spans ✅

**Files:**
- [x] `backend-payment/internal/service/payment_service_impl.go`

**Key methods instrumented:**
- `CreatePayment()` - ✅
- `ProcessPayment()` - ✅
- `GetPayment()` - ✅
- `GetPaymentByBookingID()` - ✅
- `GetUserPayments()` - ✅
- `RefundPayment()` - ✅
- `CancelPayment()` - ✅
- `CompletePaymentFromWebhook()` - ✅
- `FailPaymentFromWebhook()` - ✅

---

### 1.5 Saga Handlers - Fix Logger Context ✅

**Files (80 logger calls updated):**
- [x] `backend-booking/internal/saga/timeout_handler.go` (15 calls)
- [x] `backend-booking/internal/saga/dlq_handler.go` (6 calls)
- [x] `backend-booking/internal/saga/kafka_consumer.go` (14 calls)
- [x] `backend-booking/internal/saga/orchestrator_handler.go` (11 calls)
- [x] `backend-booking/internal/saga/payment_success_consumer.go` (3 calls)
- [x] `backend-booking/internal/saga/kafka_producer.go` (11 calls)
- [x] `backend-payment/internal/consumer/booking_consumer.go` (20 calls)

**Tasks completed:**
- [x] Updated Logger interface in `backend-booking/internal/saga/types.go` with context methods
- [x] Updated Logger interface in `pkg/saga/orchestrator.go` with context methods
- [x] Added `ZapLogger` context methods in `backend-booking/internal/saga/orchestrator_handler.go`
- [x] Changed `logger.Info(...)` → `logger.InfoContext(ctx, ...)` where ctx available
- [x] Changed `logger.Error(...)` → `logger.ErrorContext(ctx, ...)` where ctx available
- [x] Changed `logger.Warn(...)` → `logger.WarnContext(ctx, ...)` where ctx available

---

### 1.6 Business Metrics - Booking & Payment

**File to create:** `backend-booking/internal/metrics/metrics.go`

```go
package metrics

import "booking-rush/pkg/telemetry"

var (
    BookingsReserved  *telemetry.Counter
    BookingsConfirmed *telemetry.Counter
    BookingsExpired   *telemetry.Counter
    BookingsFailed    *telemetry.Counter

    ReservationDuration *telemetry.Histogram
    SeatsAvailable      *telemetry.Gauge
)

func Init() error {
    var err error

    BookingsReserved, err = telemetry.NewCounter(telemetry.MetricOpts{
        Name:        "bookings_reserved_total",
        Description: "Total number of seat reservations",
        Unit:        "1",
    })
    if err != nil {
        return err
    }

    // ... other metrics
    return nil
}
```

**Metrics to create:**

| Metric | Type | Service | Description |
|--------|------|---------|-------------|
| `bookings_reserved_total` | Counter | booking | Reservations created |
| `bookings_confirmed_total` | Counter | booking | Bookings confirmed |
| `bookings_expired_total` | Counter | booking | Expired reservations |
| `bookings_failed_total` | Counter | booking | Failed bookings |
| `payments_processed_total` | Counter | payment | Successful payments |
| `payments_failed_total` | Counter | payment | Failed payments |
| `payments_refunded_total` | Counter | payment | Refunded payments |
| `reservation_duration_seconds` | Histogram | booking | Time from reserve to confirm |
| `payment_duration_seconds` | Histogram | payment | Payment processing time |
| `seats_available` | Gauge | booking | Available seats per zone |

**Where to increment:**
- [ ] `booking_service.go` - increment counters after operations
- [ ] `payment_service_impl.go` - increment counters after operations
- [ ] `main.go` - call `metrics.Init()` on startup

---

## Phase 2: Supporting Services

### 2.1 Auth Handlers - Add Spans

**Files:**
- [ ] `backend-auth/internal/handler/auth_handler.go`
- [ ] `backend-auth/internal/handler/tenant_handler.go`

**Key methods:**
- `Register()`, `Login()`, `RefreshToken()`
- `CreateTenant()`, `GetTenant()`

---

### 2.2 Auth Services - Add Spans

**Files:**
- [ ] `backend-auth/internal/service/auth_service.go`
- [ ] `backend-auth/internal/service/tenant_service.go`

---

### 2.3 Ticket Handlers - Add Spans

**Files:**
- [ ] `backend-ticket/internal/handler/event_handler.go`
- [ ] `backend-ticket/internal/handler/show_handler.go`
- [ ] `backend-ticket/internal/handler/show_zone_handler.go`

---

### 2.4 Ticket Services - Add Spans

**Files:**
- [ ] `backend-ticket/internal/service/event_service.go`
- [ ] `backend-ticket/internal/service/show_service.go`
- [ ] `backend-ticket/internal/service/show_zone_service.go`
- [ ] `backend-ticket/internal/service/zone_syncer.go`

---

### 2.5 API Gateway - Add Spans

**Files:**
- [ ] `backend-api-gateway/internal/proxy/proxy.go`
- [ ] `backend-api-gateway/internal/middleware/rate_limiter.go`

---

## Phase 3: Advanced Metrics & Polish

### 3.1 Latency Histograms

- [ ] Add histogram buckets for p50, p90, p99
- [ ] Track reservation-to-confirmation latency
- [ ] Track payment processing latency

### 3.2 Gauge Metrics

- [ ] `active_reservations` - current pending reservations
- [ ] `queue_position` - virtual queue depth
- [ ] `rate_limit_remaining` - remaining requests per window

### 3.3 Error Tracking

- [ ] Error rate by service
- [ ] Error rate by error type
- [ ] Slow request tracking (>1s)

### 3.4 Span Events

Add events for important state changes:
```go
span.AddEvent("payment_initiated", trace.WithAttributes(
    attribute.String("payment_method", method),
    attribute.Int64("amount", amount),
))
```

---

## Helper Code Snippets

### Import Statement
```go
import (
    "booking-rush/pkg/telemetry"
    "booking-rush/pkg/logger"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)
```

### Start Span in Handler
```go
ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.{service}.{method}")
defer span.End()
c.Request = c.Request.WithContext(ctx) // Important: update request context
```

### Start Span in Service
```go
ctx, span := telemetry.StartSpan(ctx, "service.{service}.{method}")
defer span.End()
```

### Add Attributes
```go
span.SetAttributes(
    attribute.String("user_id", userID),
    attribute.String("event_id", eventID),
    attribute.Int("quantity", qty),
    attribute.String("status", status),
)
```

### Record Error
```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return err
}
span.SetStatus(codes.Ok, "")
```

### Logger with Context
```go
// Before (wrong)
logger.Info("booking created", zap.String("id", id))

// After (correct)
logger.InfoContext(ctx, "booking created", zap.String("id", id))
```

### Increment Counter
```go
metrics.BookingsReserved.Inc(ctx,
    telemetry.EventIDAttr(eventID),
    telemetry.UserIDAttr(userID),
)
```

---

## Verification Checklist

After implementation, verify:

- [ ] Traces visible in Tempo (Grafana)
- [ ] Logs have `trace_id` field
- [ ] Logs correlate with traces in Grafana
- [ ] Metrics visible in Prometheus/Grafana
- [ ] E2E trace shows: HTTP → Handler → Service → DB/Redis/Kafka
- [ ] Error traces have proper status and error details

---

## Notes

- **Notification Service** (NestJS) - Phase 5, not started yet
- **Analytics Service** (NestJS) - Phase 5, not started yet
- These will need separate OTel setup with `@opentelemetry/sdk-node`

---

Last Updated: 2025-12-16 (Phase 1.4, 1.5 completed)
