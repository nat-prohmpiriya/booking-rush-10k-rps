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
