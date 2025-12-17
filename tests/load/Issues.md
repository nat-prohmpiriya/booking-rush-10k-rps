# Load Test Notes

## Test Data Summary

| Item | จำนวน |
|------|-------|
| Events | 3 |
| Shows | 9 (3 per event) |
| Zones | 45 (5 per show) |
| Seats per zone | 20,000 |
| **Total seats** | **900,000** |

Zone ID format: `b0000000-0000-{show_idx:04d}-{zone_idx:04d}-000000000000`

---

## Issues Found

### Issue #1: ต้อง Clear Data ก่อนทุกครั้งที่ Test

**สาเหตุ:**
- Idempotency keys สะสม (พบ 1.78M keys หลังรัน test)
- Reservation keys สะสม (พบ 101K keys)
- Zone availability ลดลงเรื่อยๆ จนหมด

**ผลกระทบ:**
- Success rate ต่ำมาก (~10.8%) เพราะ seats หมด
- Redis memory สูงขึ้นเรื่อยๆ
- Response time ช้าลง

**วิธีแก้:**
ต้องรัน script clear data ก่อนทุกครั้ง:

```bash
# Clear Redis data
docker exec booking-rush-redis redis-cli -a redis123 --no-auth-warning FLUSHDB

# Sync inventory จาก DB ใหม่
curl -X POST http://localhost:8080/api/v1/admin/sync-inventory \
  -H "Authorization: Bearer $TOKEN"
```

---

### Issue #2: seed_redis.sh ใช้ Zone ID ผิด format

**ปัญหา:**
- Script ใช้ `load-test-zone-1-1`
- แต่ DB ใช้ `b0000000-0000-0001-0001-000000000000`

**วิธีแก้:**
อัพเดท seed_redis.sh ให้ใช้ UUID format ที่ถูกต้อง หรือใช้ `/api/v1/admin/sync-inventory` แทน

---

## Pre-Test Checklist

1. [ ] Stop any running k6 tests
2. [ ] Clear Redis: `docker exec booking-rush-redis redis-cli -a redis123 FLUSHDB`
3. [ ] Sync inventory: `POST /api/v1/admin/sync-inventory`
4. [ ] Verify zones have seats: `GET /api/v1/admin/inventory-status`
5. [ ] Get fresh auth token
6. [ ] Start test

---

## Test Results Log

### Test Run: 2024-12-16 19:30

| Metric | ค่า | Threshold | Status |
|--------|-----|-----------|--------|
| Iterations | 1,106,726 | - | ~1,545 RPS |
| Success Rate | 10.8% | >95% | FAIL |
| p(95) Duration | 1,879ms | <500ms | FAIL |
| p(90) Duration | 1,267ms | - | High |
| Avg Duration | 439ms | <200ms | WARN |

**Root Cause:** Seats หมดเพราะไม่ได้ clear data ก่อน test

---

### Issue #3: Multiple Instances on Single Machine Degraded Performance

**Date:** 2025-12-17

---

#### เรื่องเล่าจากการทดสอบ: เมื่อ "Scale Up" กลายเป็น "Scale Down"

เราเริ่มต้นวันนี้ด้วยความมั่นใจ — booking service ตัวเดียวทำได้ 1,817 RPS แล้ว ถ้าเพิ่มเป็น 3 instances น่าจะได้สัก 5,000 RPS ใกล้เป้า 10k แล้ว!

**ความคาดหวัง:** 3 instances = 3x performance = ~5,400 RPS

**ความจริง:** กลับได้แค่ 656 RPS — *ต่ำกว่าเดิม 3 เท่า*

เกิดอะไรขึ้น?

---

#### บทที่ 1: ปัญหาแรก — Load ไม่กระจาย

เมื่อ scale เป็น 3 instances และรัน sustained test ครั้งแรก ผลออกมาแปลก:

- booking-1: CPU 59%
- booking-2: CPU 83%
- **booking-3: CPU 201%** ← ทำไมรับภาระคนเดียว?

ตรวจสอบพบว่า API Gateway ใช้ `http://booking:8083` เป็น URL เดียว Docker DNS จะ round-robin ก็จริง แต่ HTTP client ของ Go มี **connection pooling** — มันจะ reuse connection เดิมไปที่ instance เดิมตลอด

**บทเรียน:** Docker DNS round-robin ไม่เพียงพอสำหรับ HTTP connection pooling

---

#### บทที่ 2: เพิ่ม nginx Load Balancer

ตัดสินใจเพิ่ม nginx เป็น load balancer หน้า booking services:

```nginx
upstream booking_service {
    least_conn;
    server booking-rush-10k-rps-booking-1:8083;
    server booking-rush-10k-rps-booking-2:8083;
    server booking-rush-10k-rps-booking-3:8083;
}
```

ผลลัพธ์: Load กระจายดีขึ้น! แต่...

- RPS เพิ่มจาก 656 → 970 ✓
- **Error rate พุ่งเป็น 19%!** ✗

Log เต็มไปด้วย `502 Bad Gateway` และ `no live upstreams`

---

#### บทที่ 3: แก้ 502 Errors ด้วย Retry

เพิ่ม configuration ให้ nginx retry เมื่อ upstream fail:

```nginx
proxy_next_upstream error timeout http_502 http_503 http_504;
proxy_next_upstream_tries 3;
max_fails=3 fail_timeout=10s;
```

ผลลัพธ์:
- Error rate ลดจาก 19% → **0.0005%** ✓
- แต่ RPS กลับ *ลดลง* จาก 970 → 699 ✗

เรากำลังวนอยู่ในวงจรที่แปลก — แก้ปัญหาหนึ่ง แต่สร้างปัญหาใหม่

---

#### บทที่ 4: ค้นพบความจริง

หยุดคิดและมองภาพใหญ่: ทุกอย่างรันบน **machine เดียวกัน**

```
┌────────────────────────────────────────────────┐
│           MacBook (11.67 GB RAM)               │
│                                                │
│  booking-1   booking-2   booking-3             │
│   2.75 GB     2.78 GB     2.81 GB              │
│      ↓           ↓           ↓                 │
│      └───────────┼───────────┘                 │
│                  ↓                             │
│            PostgreSQL  ← ทุกคนแย่งกันใช้        │
│            Redis       ← ทุกคนแย่งกันใช้        │
│            CPU cores   ← ทุกคนแย่งกันใช้        │
└────────────────────────────────────────────────┘
```

**Memory:** 3 booking instances ใช้ RAM รวม 8.3 GB จากทั้งหมด 11.67 GB (71%!)

**สิ่งที่เราทำไม่ใช่ "Horizontal Scaling" แต่เป็น "Resource Splitting"**

แทนที่จะเพิ่ม capacity เรากลับ:
- แบ่ง CPU ให้แย่งกัน
- แบ่ง Memory ให้แย่งกัน
- เพิ่ม network hops (gateway → nginx → booking)
- เพิ่ม database connections (100 → 300)

---

#### ตารางเปรียบเทียบการเดินทาง

| ขั้นตอน | Configuration | RPS | Errors | เกิดอะไรขึ้น |
|---------|---------------|-----|--------|-------------|
| 1 | 1 instance, pool=100 | **1,817** | 0% | Baseline ที่ดี |
| 2 | 3 instances (no LB) | 656 | 0.1% | Load ไม่กระจาย |
| 3 | 3 instances + nginx | 970 | 19% | 502 errors |
| 4 | 3 instances + retry | 699 | 0.0005% | ช้าลงเพราะ overhead |

**สรุป:** ยิ่งพยายามแก้ ยิ่งถอยหลัง

---

#### บทเรียนที่ได้

1. **"More instances" ≠ "More performance"** บน single machine
2. **True horizontal scaling** ต้องแยก physical resources
3. **Connection pooling** ทำให้ DNS round-robin ไม่ work
4. **Network hops** แต่ละ hop เพิ่ม latency 1-5ms
5. **Memory pressure** ทำให้ทุกอย่างช้าลง

---

#### สิ่งที่ควรทำ

**สำหรับ Local Testing:**
```yaml
booking:
  deploy:
    replicas: 1  # กลับไปใช้ตัวเดียว
```
ผลลัพธ์: 1,817 RPS, 0% errors — ดีที่สุดแล้วสำหรับ single machine

**สำหรับ Production 10k RPS:**
- Deploy บน Kubernetes cluster (แยก node)
- ใช้ PgBouncer สำหรับ connection pooling
- ใช้ Redis Cluster
- แยก database read replicas

---

#### บทส่งท้าย

วันนี้เราเรียนรู้ว่า "scaling" ไม่ใช่แค่เพิ่มจำนวน containers มันคือการเพิ่ม **resources จริงๆ**

การรัน 3 instances บน machine เดียวเหมือนการให้คน 3 คนแย่งกันใช้คอมพิวเตอร์เครื่องเดียว — งานไม่ได้เร็วขึ้น แต่ทุกคนช้าลง

**Final Score:**
- 1 instance: **1,817 RPS** ✓
- 3 instances: **699 RPS** ✗

บางครั้ง... น้อยกว่า คือ มากกว่า

## ⏺ Virtual Queue สถานะ

  มี Endpoints แล้ว:

  POST   /queue/join           - เข้าคิว
  GET    /queue/position/:id   - เช็คตำแหน่ง
  DELETE /queue/leave          - ออกจากคิว
  GET    /queue/status/:id     - สถานะคิว

  แต่ Load Test ไม่ได้ใช้!

  ตอนนี้ 01-booking-reserve.js เรียก:
  POST /bookings/reserve  ← ตรงไปที่ booking เลย (ไม่ผ่าน queue)

  Flow ที่ควรเป็น (สำหรับ 10k RPS):

  User → Queue (join) → Wait → Get Token → Reserve
           ↓
     ควบคุม rate, fair ordering

  Flow ตอนนี้:

  User → Reserve ← ทุกคนแย่งกันตรงๆ

  ---
  Virtual Queue จะช่วย:
  - ควบคุม rate ที่เข้า booking service
  - Fair ordering (มาก่อนได้ก่อน)
  - ลด contention ที่ DB/Redis

  ต้องการให้สร้าง load test ใหม่ที่ใช้ Virtual Queue ไหมครับ?

---

### Issue #4: Goroutine Leak จาก Kafka Producer

**Date:** 2025-12-17

---

#### อาการที่พบ

หลังรัน load test เสร็จ พบว่า booking service มี resource usage สูงผิดปกติ:

| Metric | ค่าปกติ | ค่าที่พบ |
|--------|--------|---------|
| CPU | 0-10% | **532%** |
| Memory | 40-100 MB | **6.17 GB** |
| Goroutines | < 100 | **895,417** |

---

#### วิเคราะห์ด้วย pprof

```bash
curl -s 'http://localhost:9083/debug/pprof/goroutine?debug=1' | head -5
```

ผลลัพธ์:
```
goroutine profile: total 895,417
447,645 @ franz-go/pkg/kgo.(*Client).produce
```

พบว่า **447,645 goroutines** ค้างอยู่ที่ Kafka producer

---

#### สาเหตุ

```
ทุก booking request:
  1. Reserve seats (Redis)       ← เร็ว, ไม่ blocking
  2. Create booking (PostgreSQL) ← เร็ว, connection pool
  3. Publish event (Kafka)       ← BLOCKING รอ ack!
```

**Code ที่เป็นปัญหา:** `pkg/kafka/producer.go:147`

```go
result := p.client.ProduceSync(ctx, record)  // blocking!
```

- `ProduceSync` จะ block จนกว่า Kafka จะ ack
- Load test: 456K requests × 1+ events = 456K+ goroutines
- Redpanda (Kafka) ตอบไม่ทัน → goroutines สะสม

**Config ปัจจุบัน:**
```go
BatchSize: 100
LingerMs:  10
```

---

#### ผลกระทบ

1. **Memory leak**: Goroutines สะสมจนใช้ RAM 6+ GB
2. **CPU spike**: Scheduler ต้องจัดการ goroutines มากเกินไป
3. **Performance drop**: RPS ลดลงจาก 1,816 → 1,516 (16%)

---

#### วิธีแก้ชั่วคราว

Restart booking service เพื่อ reset goroutines:

```bash
docker-compose -f docker-compose.k6-1instance.yml restart booking
docker exec booking-rush-redis redis-cli -a redis123 FLUSHDB
# restart inventory-worker เพื่อ sync inventory ใหม่
docker-compose -f docker-compose.k6-1instance.yml restart inventory-worker
```

---

#### วิธีแก้ถาวร ✅ FIXED

**แก้ไขแล้ว 2 จุด:**

**1. เปลี่ยนจาก `Produce` (sync) เป็น `ProduceAsync` (non-blocking)**

File: `backend-booking/internal/service/event_publisher.go`

```go
// Before (blocking)
err := p.producer.Produce(ctx, msg)

// After (non-blocking with callback)
p.producer.ProduceAsync(ctx, msg, func(err error) {
    if err != nil && p.logger != nil {
        p.logger.Error(fmt.Sprintf("failed to publish %s event: %v", eventType, err))
    }
})
```

เพิ่ม Logger interface และ ZapLoggerAdapter สำหรับ error callback

**2. ลบ `go func()` wrapper ที่ซ้ำซ้อน**

File: `backend-booking/internal/service/booking_service.go:266`

```go
// Before (double goroutine - leak!)
go func() {
    if pubErr := s.eventPublisher.PublishBookingCreated(context.Background(), booking); pubErr != nil {
        // Log error
    }
}()

// After (ProduceAsync is already non-blocking)
_ = s.eventPublisher.PublishBookingCreated(ctx, booking)
```

**Files changed:**
- `backend-booking/internal/service/event_publisher.go`
  - เปลี่ยน `Produce()` → `ProduceAsync()` ใน `publishEvent()`
  - เพิ่ม `Logger` interface และ `ZapLoggerAdapter`
- `backend-booking/internal/service/booking_service.go`
  - ลบ `go func()` wrapper (line 266)
- `backend-booking/main.go`
  - Pass logger ให้ EventPublisherConfig

---

#### ผลการทดสอบหลังแก้ไข

**Goroutine Count:**

| สถานะ | Goroutines |
|-------|------------|
| ก่อนแก้ | 895,417 |
| หลังแก้ครั้งแรก (ProduceAsync) | 4,620 |
| **หลังแก้ครบ (ลบ go func)** | **28** ✅ |

**Smoke Test Results (2025-12-17 15:56):**

| Metric | ค่า | สถานะ |
|--------|-----|-------|
| Requests | 3,462 | ✅ |
| Success Rate | 100% | ✅ |
| Error Rate | 0% | ✅ |
| Avg Response | 2.38ms | ✅ |
| p(95) Response | 4.93ms | ✅ |
| Goroutines หลัง test | 28 | ✅ ไม่ leak |

---

#### บทเรียน

1. **ProduceSync vs ProduceAsync**: ถ้าไม่ต้องการ guarantee delivery, ใช้ async เพื่อไม่ block request
2. **Double goroutine wrapping**: อย่าใช้ `go func()` ครอบ function ที่เป็น async อยู่แล้ว
3. **pprof เป็น lifesaver**: ช่วยหา goroutine leak ได้ทันที

```bash
# เช็ค goroutine count
curl -s 'http://localhost:9083/debug/pprof/goroutine?debug=1' | head -1
```