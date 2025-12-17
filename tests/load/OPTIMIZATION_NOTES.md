# Load Test Optimization Notes

## Test Date: 2025-12-17

## Baseline Results

### ramp_up (0→1000 VUs, 9 min)
| Metric | Result |
|--------|--------|
| Total Requests | 1,032,590 |
| Throughput | ~1,900 RPS |
| Success Rate | 96% |
| p95 Latency | 817ms |
| Server Errors | 4% |

### sustained (5000 RPS target, 5 min)
| Metric | Target | Actual |
|--------|--------|--------|
| Throughput | 5,000 RPS | 762 RPS (15%) |
| Success Rate | >95% | 15% |
| p95 Latency | <500ms | 6,067ms |
| Avg Latency | <200ms | 2,577ms |
| Dropped Requests | 0 | 1,267,069 |

---

## Identified Bottlenecks

### 1. DB Connection Pool (CRITICAL)
**Current:**
```
BOOKING_DATABASE_MAX_OPEN_CONNS=20
BOOKING_DATABASE_MAX_IDLE_CONNS=5
```
**Recommend:**
```
BOOKING_DATABASE_MAX_OPEN_CONNS=100
BOOKING_DATABASE_MAX_IDLE_CONNS=50
```
**Note:** PostgreSQL max_connections=150, enough for pool=100

### 2. Gateway Logging (HIGH)
**Current:**
```
LOG_LEVEL=debug
```
**Recommend:**
```
LOG_LEVEL=error
```
**Impact:** Gateway CPU 90-120% during load test

### 3. Redis Pool Size (MEDIUM)
**Current:** Not set (using default)
**Recommend:**
```
REDIS_POOL_SIZE=100
```
**Note:** Redis not blocking (blocked_clients=0), but should tune for high concurrency

### 4. Single Booking Instance (HIGH)
**Current:** 1 instance
**Recommend:** 3-5 instances
**How to fix:**
1. Remove `container_name` from docker-compose
2. Change port mapping to dynamic
3. Use `docker-compose up --scale booking=3`

---

## NOT Bottlenecks (Confirmed)

| Component | Status | Evidence |
|-----------|--------|----------|
| Saga | Not used in reserve | Only triggered after payment webhook |
| Redis blocking | No issue | blocked_clients=0 |
| PostgreSQL max_connections | OK | 150 > total pool needed |

---

## Action Items

- [ ] Update `.env.local` with new pool sizes
- [ ] Change LOG_LEVEL to error
- [ ] Add REDIS_POOL_SIZE=100
- [ ] Modify docker-compose for booking scaling
- [ ] Re-run load tests after optimization
- [ ] Run spike and stress_10k tests

---

## Files to Modify

1. `.env.local`
   - LOG_LEVEL=error
   - BOOKING_DATABASE_MAX_OPEN_CONNS=100
   - BOOKING_DATABASE_MAX_IDLE_CONNS=50
   - REDIS_POOL_SIZE=100

2. `docker-compose.services.yml`
   - Remove container_name from booking
   - Change ports to dynamic range
   - Add deploy.replicas config
