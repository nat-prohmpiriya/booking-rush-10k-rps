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
| **Business Metrics** | ✅ | Booking & Payment metrics created |
| **Span Events** | ✅ | State change events (reservation, confirm, cancel, payment) |
| **Error Tracking** | ✅ | Error counters + slow request tracking |
| **Latency Histograms** | ✅ | Request duration with p50/p90/p99 buckets |

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

### 1.6 Business Metrics - Booking & Payment ✅

**Files created:**
- [x] `backend-booking/internal/metrics/metrics.go`
- [x] `backend-payment/internal/metrics/metrics.go`

**Booking Metrics implemented:**

| Metric | Type | Description |
|--------|------|-------------|
| `booking_reservations_total` | Counter | Reservations created |
| `booking_confirmations_total` | Counter | Bookings confirmed |
| `booking_expirations_total` | Counter | Expired reservations |
| `booking_failures_total` | Counter | Failed bookings |
| `booking_cancellations_total` | Counter | Cancelled bookings |
| `queue_joins_total` | Counter | Queue joins |
| `queue_leaves_total` | Counter | Queue leaves |
| `booking_reservation_duration_seconds` | Histogram | Time from reserve to confirm |
| `queue_wait_time_seconds` | Histogram | Queue wait time |
| `booking_active_reservations` | UpDownCounter | Active reservations |
| `queue_depth` | UpDownCounter | Current queue size |

**Payment Metrics implemented:**

| Metric | Type | Description |
|--------|------|-------------|
| `payment_created_total` | Counter | Payments created |
| `payment_processed_total` | Counter | Successful payments |
| `payment_failed_total` | Counter | Failed payments |
| `payment_refunded_total` | Counter | Refunded payments |
| `payment_cancelled_total` | Counter | Cancelled payments |
| `payment_webhooks_received_total` | Counter | Webhooks received |
| `payment_webhooks_processed_total` | Counter | Webhooks processed |
| `payment_webhooks_failed_total` | Counter | Webhooks failed |
| `payment_processing_duration_seconds` | Histogram | Payment processing time |
| `payment_amount` | Histogram | Payment amounts |
| `payment_webhook_processing_seconds` | Histogram | Webhook processing time |
| `payment_pending` | UpDownCounter | Pending payments |

**Where metrics are recorded:**
- [x] `booking_service.go` - ReserveSeats, ConfirmBooking, CancelBooking, ExpireReservations
- [x] `payment_service_impl.go` - CreatePayment, ProcessPayment, RefundPayment, CancelPayment
- [ ] `main.go` - call `metrics.Init()` on startup (TODO)

---

## Phase 2: Supporting Services ✅

### 2.1 Auth Handlers - Add Spans ✅

**Files:**
- [x] `backend-auth/internal/handler/auth_handler.go`
- [x] `backend-auth/internal/handler/tenant_handler.go`

**Key methods instrumented:**
- `Register()`, `Login()`, `RefreshToken()`, `Logout()`, `LogoutAll()`
- `Me()`, `UpdateMe()`, `ValidateToken()`
- `GetStripeCustomerID()`, `UpdateStripeCustomerID()`
- `Create()`, `GetByID()`, `GetBySlug()`, `List()`, `Update()`, `Delete()` (Tenant)

---

### 2.2 Auth Services - Add Spans ✅

**Files:**
- [x] `backend-auth/internal/service/auth_service.go`

**Key methods instrumented:**
- `Register()`, `Login()`, `RefreshToken()`, `Logout()`, `LogoutAll()`
- `ValidateToken()`, `GetUser()`, `UpdateProfile()`
- `GetStripeCustomerID()`, `UpdateStripeCustomerID()`

---

### 2.3 Ticket Handlers - Add Spans ✅

**Files:**
- [x] `backend-ticket/internal/handler/event_handler.go`
- [x] `backend-ticket/internal/handler/show_handler.go`
- [x] `backend-ticket/internal/handler/show_zone_handler.go`

**Key methods instrumented:**
- Event: `List()`, `GetBySlug()`, `GetByID()`, `ListMyEvents()`, `Create()`, `Update()`, `Delete()`, `Publish()`
- Show: `ListByEvent()`, `Create()`, `GetByID()`, `Update()`, `Delete()`
- ShowZone: `ListByShow()`, `Create()`, `GetByID()`, `Update()`, `Delete()`, `ListActive()`

---

### 2.4 Ticket Services - Add Spans

**Files:**
- [ ] `backend-ticket/internal/service/event_service.go`
- [ ] `backend-ticket/internal/service/show_service.go`
- [ ] `backend-ticket/internal/service/show_zone_service.go`
- [ ] `backend-ticket/internal/service/zone_syncer.go`

---

### 2.5 API Gateway - Add Spans ✅

**Files:**
- [x] `backend-api-gateway/internal/proxy/proxy.go`
- [x] `backend-api-gateway/internal/middleware/rate_limiter.go`

**Key functionality instrumented:**
- Proxy: `Handler()` with target service, method, path attributes
- Rate Limiter: `RateLimiter()`, `PerEndpointRateLimiter()` with client_ip, path, allowed attributes

---

## Phase 3: Advanced Metrics & Polish ✅

### 3.1 Latency Histograms ✅

- [x] Add histogram buckets for p50, p90, p99 (buckets: 5ms to 10s)
- [x] Track reservation-to-confirmation latency (`booking_reservation_duration_seconds`)
- [x] Track payment processing latency (`payment_processing_duration_seconds`)
- [x] HTTP request duration (`booking_request_duration_seconds`, `payment_request_duration_seconds`)

### 3.2 Gauge Metrics ✅

- [x] `booking_active_reservations` - current pending reservations
- [x] `queue_depth` - virtual queue depth
- [x] `payment_pending` - current pending payments
- [ ] `rate_limit_remaining` - remaining requests per window (Phase 2 - API Gateway)

### 3.3 Error Tracking ✅

- [x] `booking_errors_total` - Error count by error_type and operation
- [x] `payment_errors_total` - Error count by error_type and operation
- [x] `booking_slow_requests_total` - Slow request tracking (>1s)
- [x] `payment_slow_requests_total` - Slow request tracking (>1s)

**Helper functions added:**
- `metrics.RecordError(ctx, errorType, operation)`
- `metrics.RecordRequestDuration(ctx, operation, durationSeconds)`
- `metrics.RecordSlowRequest(ctx, operation, durationSeconds)`

### 3.4 Span Events ✅

Events added for important state changes:

**Booking Service:**
- `reservation_created` - When reservation is successfully created
- `booking_confirmed` - When booking is confirmed with payment
- `booking_cancelled` - When booking is cancelled

**Payment Service:**
- `payment_created` - When payment record is created
- `payment_completed` - When payment is successfully processed
- `payment_failed` - When payment processing fails

**Example:**
```go
span.AddEvent("payment_completed", trace.WithAttributes(
    attribute.String("payment_id", payment.ID),
    attribute.String("transaction_id", chargeResp.TransactionID),
    attribute.Float64("duration_seconds", durationSeconds),
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

Last Updated: 2025-12-16 (Phase 3 completed, Phase 2 nearly completed - Ticket Services pending)
