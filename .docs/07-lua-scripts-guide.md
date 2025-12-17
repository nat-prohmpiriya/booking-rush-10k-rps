# Lua Scripts Guide

เอกสารนี้อธิบายการใช้งาน Lua scripts ในระบบ Booking Rush สำหรับจัดการ seat reservation แบบ atomic

## ทำไมต้องใช้ Lua Scripts?

### ปัญหา Race Condition

เมื่อมี requests จำนวนมาก (10,000 RPS) เข้ามาพร้อมกัน การใช้ Redis commands แยกกันจะเกิด race condition:

```
User A: GET available → 100
User B: GET available → 100
User A: SET available → 99  (จอง 1 ที่)
User B: SET available → 99  (จอง 1 ที่) ❌ ควรเป็น 98!
```

ผลลัพธ์: ขายเกิน (overselling) เพราะทั้งคู่เห็น 100 ที่นั่งพร้อมกัน

### วิธีแก้ด้วย Lua Scripts

Lua script รันบน Redis server แบบ **atomic** (single-threaded) ทำให้ทุก operation ภายใน script เสร็จสิ้นก่อนที่ request อื่นจะเข้ามา:

```
User A: [GET + CHECK + DECRBY] → atomic, ได้ที่นั่ง
User B: [GET + CHECK + DECRBY] → รอ User A เสร็จก่อน → เห็น 99 ที่นั่ง
```

---

## Scripts ทั้งหมด

| Script | หน้าที่ | ตำแหน่งไฟล์ |
|--------|--------|-------------|
| `reserve_seats.lua` | จองที่นั่ง | `backend-booking/internal/repository/scripts/` |
| `release_seats.lua` | คืนที่นั่งกลับ inventory | `backend-booking/internal/repository/scripts/` |
| `confirm_booking.lua` | ยืนยันการจองหลังชำระเงิน | `backend-booking/internal/repository/scripts/` |
| `join_queue.lua` | เข้าคิว virtual queue | `backend-booking/internal/repository/scripts/` |

---

## 1. Reserve Seats Script

### หน้าที่
จองที่นั่งแบบ atomic พร้อมตรวจสอบเงื่อนไขทั้งหมดในครั้งเดียว

### Key Structure

```
KEYS[1]: zone:availability:{zone_id}           # จำนวนที่นั่งว่าง (integer)
KEYS[2]: user:reservations:{user_id}:{event_id} # จำนวนที่ user จองไว้ใน event นี้
KEYS[3]: reservation:{booking_id}              # ข้อมูล reservation (hash)
```

### Arguments

| ลำดับ | ชื่อ | คำอธิบาย | ตัวอย่าง |
|-------|------|----------|---------|
| ARGV[1] | quantity | จำนวนที่นั่งที่ต้องการจอง | `2` |
| ARGV[2] | max_per_user | จำนวนสูงสุดที่ user จองได้ต่อ event | `4` |
| ARGV[3] | user_id | UUID ของ user | `550e8400-...` |
| ARGV[4] | booking_id | UUID ของ booking | `6ba7b810-...` |
| ARGV[5] | zone_id | UUID ของ zone | `6ba7b811-...` |
| ARGV[6] | event_id | UUID ของ event | `6ba7b812-...` |
| ARGV[7] | show_id | UUID ของ show | `6ba7b813-...` |
| ARGV[8] | unit_price | ราคาต่อที่นั่ง (satang) | `150000` |
| ARGV[9] | ttl_seconds | เวลา reservation หมดอายุ | `600` (10 นาที) |

### Flow การทำงาน

```
┌─────────────────────────────────────────────────────┐
│              Reserve Seats Flow                      │
├─────────────────────────────────────────────────────┤
│  1. Validate quantity > 0                           │
│                 ↓                                   │
│  2. GET zone:availability:{zone_id}                 │
│     → ตรวจสอบ ที่นั่งว่างพอไหม                       │
│                 ↓                                   │
│  3. GET user:reservations:{user_id}:{event_id}      │
│     → ตรวจสอบ user จองเกิน limit ไหม                │
│                 ↓                                   │
│  4. DECRBY zone:availability (หักที่นั่ง)           │
│                 ↓                                   │
│  5. INCRBY user:reservations (บวก user count)       │
│                 ↓                                   │
│  6. HSET reservation:{booking_id}                   │
│     → สร้าง reservation record                      │
│                 ↓                                   │
│  7. EXPIRE reservation:{booking_id} TTL             │
│     → ตั้ง TTL ให้หมดอายุอัตโนมัติ                   │
└─────────────────────────────────────────────────────┘
```

### Return Values

**สำเร็จ:**
```lua
{1, remaining_seats, total_user_reserved}
-- ตัวอย่าง: {1, 98, 2}  -- เหลือ 98 ที่, user จองไป 2 ที่
```

**ล้มเหลว:**
```lua
{0, error_code, error_message}
```

### Error Codes

| Code | สาเหตุ | คำอธิบาย |
|------|--------|----------|
| `INVALID_QUANTITY` | quantity <= 0 | จำนวนต้องมากกว่า 0 |
| `ZONE_NOT_FOUND` | key ไม่มี | ยังไม่ได้ sync inventory เข้า Redis |
| `INSUFFICIENT_STOCK` | available < quantity | ที่นั่งไม่พอ |
| `USER_LIMIT_EXCEEDED` | reserved + qty > max | user จองเกิน limit |

### ตัวอย่างการเรียกใช้ (Go)

```go
// backend-booking/internal/repository/redis_reservation_repository.go

result, err := r.client.Eval(ctx, reserveScript,
    []string{
        fmt.Sprintf("zone:availability:%s", zoneID),
        fmt.Sprintf("user:reservations:%s:%s", userID, eventID),
        fmt.Sprintf("reservation:%s", bookingID),
    },
    quantity,      // ARGV[1]
    maxPerUser,    // ARGV[2]
    userID,        // ARGV[3]
    bookingID,     // ARGV[4]
    zoneID,        // ARGV[5]
    eventID,       // ARGV[6]
    showID,        // ARGV[7]
    unitPrice,     // ARGV[8]
    ttlSeconds,    // ARGV[9]
).Result()
```

---

## 2. Release Seats Script

### หน้าที่
คืนที่นั่งกลับ inventory เมื่อ:
- User ยกเลิกการจอง
- Reservation หมดอายุ (timeout)
- Payment ล้มเหลว

### Key Structure

```
KEYS[1]: zone:availability:{zone_id}            # จำนวนที่นั่งว่าง
KEYS[2]: user:reservations:{user_id}:{event_id} # จำนวนที่ user จองไว้
KEYS[3]: reservation:{booking_id}               # ข้อมูล reservation
```

### Arguments

| ลำดับ | ชื่อ | คำอธิบาย |
|-------|------|----------|
| ARGV[1] | booking_id | UUID ของ booking ที่จะยกเลิก |
| ARGV[2] | user_id | UUID ของ user (สำหรับ validation) |

### Flow การทำงาน

```
┌─────────────────────────────────────────────────────┐
│              Release Seats Flow                      │
├─────────────────────────────────────────────────────┤
│  1. HGETALL reservation:{booking_id}                │
│     → ดึงข้อมูล reservation                          │
│                 ↓                                   │
│  2. Validate booking_id และ user_id                 │
│     → ตรวจสอบความถูกต้อง                             │
│                 ↓                                   │
│  3. Check status == "reserved"                      │
│     → ถ้า confirmed แล้วไม่สามารถ release            │
│                 ↓                                   │
│  4. INCRBY zone:availability (คืนที่นั่ง)           │
│                 ↓                                   │
│  5. DECRBY user:reservations                        │
│     → ลด user count (ถ้า = 0 ลบ key)                │
│                 ↓                                   │
│  6. DEL reservation:{booking_id}                    │
│     → ลบ reservation record                         │
└─────────────────────────────────────────────────────┘
```

### Return Values

**สำเร็จ:**
```lua
{1, new_available_seats, new_user_reserved}
-- ตัวอย่าง: {1, 100, 0}  -- คืน seats เรียบร้อย
```

### Error Codes

| Code | สาเหตุ |
|------|--------|
| `RESERVATION_NOT_FOUND` | ไม่พบ reservation (อาจหมดอายุแล้ว) |
| `INVALID_BOOKING_ID` | booking_id ไม่ตรง |
| `INVALID_USER_ID` | user_id ไม่ตรง |
| `ALREADY_RELEASED` | status ไม่ใช่ "reserved" |

---

## 3. Confirm Booking Script

### หน้าที่
ยืนยันการจองหลังชำระเงินสำเร็จ ทำให้ reservation เป็น permanent

### Key Structure

```
KEYS[1]: reservation:{booking_id}  # ข้อมูล reservation
```

### Arguments

| ลำดับ | ชื่อ | คำอธิบาย |
|-------|------|----------|
| ARGV[1] | booking_id | UUID ของ booking |
| ARGV[2] | user_id | UUID ของ user |
| ARGV[3] | payment_id | UUID ของ payment (optional) |

### Flow การทำงาน

```
┌─────────────────────────────────────────────────────┐
│              Confirm Booking Flow                    │
├─────────────────────────────────────────────────────┤
│  1. HGETALL reservation:{booking_id}                │
│     → ดึงข้อมูล reservation                          │
│                 ↓                                   │
│  2. Validate booking_id และ user_id                 │
│                 ↓                                   │
│  3. Check status == "reserved"                      │
│     → ถ้า confirmed แล้วไม่ทำซ้ำ                     │
│                 ↓                                   │
│  4. HSET status = "confirmed"                       │
│     + confirmed_at timestamp                        │
│     + payment_id                                    │
│                 ↓                                   │
│  5. PERSIST reservation:{booking_id}                │
│     → ลบ TTL ทำให้เป็น permanent                    │
└─────────────────────────────────────────────────────┘
```

### Return Values

**สำเร็จ:**
```lua
{1, "CONFIRMED", confirmed_at}
-- ตัวอย่าง: {1, "CONFIRMED", "1702800000.123456"}
```

### Error Codes

| Code | สาเหตุ |
|------|--------|
| `RESERVATION_NOT_FOUND` | ไม่พบ reservation |
| `INVALID_BOOKING_ID` | booking_id ไม่ตรง |
| `INVALID_USER_ID` | user_id ไม่ตรง |
| `ALREADY_CONFIRMED` | confirm ไปแล้ว |
| `INVALID_STATUS` | status ไม่ใช่ "reserved" |

---

## 4. Join Queue Script

### หน้าที่
เพิ่ม user เข้า virtual queue แบบ FIFO โดยใช้ Redis Sorted Set

### ใช้เมื่อไหร่?
- Event ที่มีคนต้องการจองมาก (high demand)
- ต้องการควบคุม traffic ไม่ให้ล้น
- ให้ user รอคิวก่อนจะจองได้

### Key Structure

```
KEYS[1]: queue:{event_id}                    # Sorted Set (score = timestamp)
KEYS[2]: queue:user:{event_id}:{user_id}     # Hash ข้อมูลคิวของ user
```

### Arguments

| ลำดับ | ชื่อ | คำอธิบาย | Default |
|-------|------|----------|---------|
| ARGV[1] | user_id | UUID ของ user | - |
| ARGV[2] | event_id | UUID ของ event | - |
| ARGV[3] | token | unique token สำหรับคิว | - |
| ARGV[4] | ttl_seconds | เวลาคิวหมดอายุ | 1800 (30 นาที) |
| ARGV[5] | max_queue_size | จำนวนคิวสูงสุด (0 = unlimited) | 0 |

### Flow การทำงาน

```
┌─────────────────────────────────────────────────────┐
│               Join Queue Flow                        │
├─────────────────────────────────────────────────────┤
│  1. ZSCORE queue:{event_id} user_id                 │
│     → ตรวจสอบ user อยู่ในคิวแล้วหรือไม่              │
│                 ↓                                   │
│  2. ZCARD queue:{event_id}                          │
│     → ตรวจสอบจำนวนคนในคิว                           │
│     → ถ้าเกิน max_queue_size return error           │
│                 ↓                                   │
│  3. ZADD queue:{event_id} timestamp user_id         │
│     → เพิ่ม user เข้าคิว (score = เวลาเข้าคิว)       │
│                 ↓                                   │
│  4. ZRANK queue:{event_id} user_id                  │
│     → หาตำแหน่งในคิว                                │
│                 ↓                                   │
│  5. HSET queue:user:{event_id}:{user_id}            │
│     → เก็บข้อมูล token, position, expires_at        │
│                 ↓                                   │
│  6. EXPIRE queue:user:... TTL                       │
│     → ตั้ง TTL ให้หมดอายุอัตโนมัติ                   │
└─────────────────────────────────────────────────────┘
```

### Return Values

**สำเร็จ:**
```lua
{1, position, total_in_queue, joined_at}
-- ตัวอย่าง: {1, 42, 500, 1702800000.123456}
-- position 42, มีคนในคิว 500 คน
```

### Error Codes

| Code | สาเหตุ |
|------|--------|
| `ALREADY_IN_QUEUE` | user อยู่ในคิวแล้ว |
| `QUEUE_FULL` | คิวเต็ม |

---

## Data Structure ใน Redis

### Reservation Hash

```redis
HGETALL reservation:{booking_id}

1) "booking_id"    2) "6ba7b810-..."
3) "user_id"       4) "550e8400-..."
5) "zone_id"       6) "6ba7b811-..."
7) "event_id"      8) "6ba7b812-..."
9) "show_id"       10) "6ba7b813-..."
11) "quantity"     12) "2"
13) "unit_price"   14) "150000"
15) "status"       16) "reserved"  # หรือ "confirmed"
17) "created_at"   18) "1702800000.123456"
19) "expires_at"   20) "1702800600"  # created_at + TTL
21) "confirmed_at" 22) "1702800300.789"  # (ถ้า confirmed)
23) "payment_id"   24) "7c9e6679-..."    # (ถ้า confirmed)
```

### Zone Availability

```redis
GET zone:availability:{zone_id}
"100"  # จำนวนที่นั่งว่าง
```

### User Reservations

```redis
GET user:reservations:{user_id}:{event_id}
"2"  # จำนวนที่ user จองไว้ใน event นี้
```

### Queue (Sorted Set)

```redis
ZRANGE queue:{event_id} 0 -1 WITHSCORES

1) "user_1_id"      2) "1702800000.123"  # คนแรก
3) "user_2_id"      4) "1702800001.456"  # คนที่สอง
5) "user_3_id"      6) "1702800002.789"  # คนที่สาม
```

---

## การ Sync Inventory เข้า Redis

ก่อนใช้งาน Lua scripts ต้อง sync inventory จาก PostgreSQL เข้า Redis ก่อน:

```bash
# เรียก API sync inventory
curl -X POST http://localhost:8080/api/v1/admin/sync-inventory \
  -H "Authorization: Bearer $TOKEN"
```

API นี้จะ:
1. ดึง zone ทั้งหมดจาก PostgreSQL
2. SET `zone:availability:{zone_id}` = `available_seats` สำหรับแต่ละ zone

---

## TTL และ Expiration

### Reservation TTL
- Default: 600 วินาที (10 นาที)
- หลังหมดเวลา: Redis ลบ key อัตโนมัติ
- **ต้องมี worker** คืน seats กลับ inventory เมื่อหมดอายุ

### User Reservations TTL
- TTL: reservation TTL + 60 วินาที (buffer)
- ลบอัตโนมัติเมื่อ user ไม่มี reservation ค้างอยู่

### Queue TTL
- Default: 1800 วินาที (30 นาที)
- ลบอัตโนมัติเมื่อหมดเวลา

---

## Best Practices

### 1. ใช้ KEYS และ ARGV ให้ถูกต้อง

```go
// ✅ ถูก - ใช้ KEYS สำหรับ key names
client.Eval(ctx, script, []string{"key1", "key2"}, "arg1", "arg2")

// ❌ ผิด - ใส่ key ใน ARGV
client.Eval(ctx, script, nil, "key1", "key2", "arg1")
```

### 2. Validate ก่อน Mutate

```lua
-- ✅ ถูก - ตรวจสอบทุกเงื่อนไขก่อน
if available < quantity then
    return {0, "INSUFFICIENT_STOCK", "..."}
end
-- แล้วค่อย DECRBY

-- ❌ ผิด - DECRBY ก่อนแล้วค่อยตรวจ
local remaining = redis.call("DECRBY", key, quantity)
if remaining < 0 then  -- สายไป!
    redis.call("INCRBY", key, quantity)  -- rollback แต่ไม่ atomic
end
```

### 3. Error Handling ใน Go

```go
result, err := client.Eval(ctx, script, keys, args...).Result()
if err != nil {
    // Redis error (connection, script error, etc.)
    return err
}

// Parse result
arr, ok := result.([]interface{})
if !ok || len(arr) < 2 {
    return errors.New("invalid script result")
}

success := arr[0].(int64)
if success == 0 {
    // Business logic error
    errorCode := arr[1].(string)
    errorMsg := arr[2].(string)
    return fmt.Errorf("%s: %s", errorCode, errorMsg)
}

// Success - parse remaining values
remainingSeats := arr[1].(int64)
userReserved := arr[2].(int64)
```

### 4. Script Caching

Redis cache compiled scripts ด้วย SHA1 hash ใช้ `EVALSHA` แทน `EVAL` สำหรับ production:

```go
// Load script once
sha, _ := client.ScriptLoad(ctx, reserveScript).Result()

// Use EVALSHA for subsequent calls
client.EvalSha(ctx, sha, keys, args...)
```

---

## Debugging

### ดู Script Error

```bash
# ดู Redis logs
docker logs booking-rush-redis

# Test script manually
redis-cli --eval reserve_seats.lua \
  zone:availability:test user:reservations:test:event reservation:test \
  , 1 4 user123 booking123 zone123 event123 show123 15000 600
```

### ดู Keys ใน Redis

```bash
# ดู zone availability
redis-cli GET zone:availability:{zone_id}

# ดู reservation
redis-cli HGETALL reservation:{booking_id}

# ดู user reservations
redis-cli GET user:reservations:{user_id}:{event_id}

# ดู queue
redis-cli ZRANGE queue:{event_id} 0 -1 WITHSCORES
```

---

## สรุป

| Script | เมื่อใช้ | ผลลัพธ์ |
|--------|---------|---------|
| `reserve_seats` | User กดจอง | หัก seats, สร้าง reservation |
| `release_seats` | User ยกเลิก / Timeout | คืน seats, ลบ reservation |
| `confirm_booking` | Payment สำเร็จ | เปลี่ยน status, ลบ TTL |
| `join_queue` | Event high demand | เพิ่ม user เข้าคิว |

ทุก script ทำงานแบบ **atomic** บน Redis server ป้องกัน race condition และ overselling
