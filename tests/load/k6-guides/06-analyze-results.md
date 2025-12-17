# วิเคราะห์ผลลัพธ์ Load Test

## อ่านผลลัพธ์จาก Terminal

หลังรัน k6 จะแสดงผลลัพธ์แบบนี้:

```
     data_received..................: 1.2 GB  4.0 MB/s
     data_sent......................: 890 MB  3.0 MB/s
     http_req_blocked...............: avg=1.2ms    min=0s       med=1µs      max=1.2s     p(90)=2µs      p(95)=3µs
     http_req_connecting............: avg=800µs    min=0s       med=0s       max=800ms    p(90)=0s       p(95)=0s
     http_req_duration..............: avg=89.21ms  min=12.34ms  med=72.45ms  max=2.34s    p(90)=142.32ms p(95)=234.56ms
       { expected_response:true }...: avg=85.12ms  min=12.34ms  med=70.23ms  max=1.89s    p(90)=138.45ms p(95)=223.67ms
     http_req_failed................: 2.34%   ✓ 2340      ✗ 97660
     http_req_receiving.............: avg=234µs    min=12µs     med=89µs     max=123ms    p(90)=345µs    p(95)=567µs
     http_req_sending...............: avg=45µs     min=5µs      med=23µs     max=45ms     p(90)=56µs     p(95)=89µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s       p(90)=0s       p(95)=0s
     http_req_waiting...............: avg=88.93ms  min=12.12ms  med=72.12ms  max=2.34s    p(90)=141.89ms p(95)=233.89ms
     http_reqs......................: 100000  333.33/s
     iteration_duration.............: avg=2.98s    min=1.01s    med=2.87s    max=12.34s   p(90)=4.23s    p(95)=5.67s
     iterations.....................: 100000  333.33/s
     vus............................: 1000    min=1000    max=1000
     vus_max........................: 1000    min=1000    max=1000
```

---

## Metrics ที่ต้องดู

### 1. HTTP Request Duration (สำคัญที่สุด)
```
http_req_duration..............: avg=89.21ms  min=12.34ms  med=72.45ms  max=2.34s  p(90)=142ms p(95)=234ms
```

| ค่า | ความหมาย | ค่าที่ดี |
|-----|----------|---------|
| **avg** | ค่าเฉลี่ย | < 200ms |
| **min** | เร็วที่สุด | - |
| **med** | ค่ากลาง (p50) | < 100ms |
| **max** | ช้าที่สุด | < 5s |
| **p(90)** | 90% เร็วกว่านี้ | < 300ms |
| **p(95)** | 95% เร็วกว่านี้ | < 500ms |

### 2. HTTP Request Failed
```
http_req_failed................: 2.34%   ✓ 2340      ✗ 97660
```

| ค่า | ความหมาย |
|-----|----------|
| **2.34%** | เปอร์เซ็นต์ที่ fail |
| **✓ 2340** | จำนวนที่ fail |
| **✗ 97660** | จำนวนที่สำเร็จ |

**เกณฑ์:**
- < 1% = ดีมาก
- 1-5% = ยอมรับได้
- > 5% = มีปัญหา

### 3. HTTP Reqs (Throughput)
```
http_reqs......................: 100000  333.33/s
```

| ค่า | ความหมาย |
|-----|----------|
| **100000** | requests ทั้งหมด |
| **333.33/s** | requests ต่อวินาที (RPS) |

### 4. VUs
```
vus............................: 1000    min=1000    max=1000
vus_max........................: 1000    min=1000    max=1000
```

จำนวน Virtual Users ที่ใช้

---

## วิเคราะห์ Timing Breakdown

### ส่วนประกอบของ http_req_duration

```
http_req_duration = blocked + connecting + sending + waiting + receiving

┌──────────────────────────────────────────────────────────────┐
│                    http_req_duration (89ms)                  │
├─────────┬─────────┬────────┬───────────────────────┬─────────┤
│ blocked │connecting│ sending │ waiting (TTFB)       │receiving│
│ 1.2ms   │ 0.8ms   │ 0.05ms │ 87ms                  │ 0.2ms   │
└─────────┴─────────┴────────┴───────────────────────┴─────────┘
```

### ดู Bottleneck จาก Timing

| ถ้า | สูง | แสดงว่า |
|-----|-----|---------|
| **blocked** สูง | > 10ms | K6 machine ทำงานไม่ทัน |
| **connecting** สูง | > 50ms | Network latency หรือ server รับ connection ไม่ทัน |
| **sending** สูง | > 10ms | Request body ใหญ่ หรือ network ช้า |
| **waiting** สูง | > 200ms | Server ประมวลผลช้า (ปัญหาหลัก) |
| **receiving** สูง | > 50ms | Response body ใหญ่ หรือ network ช้า |

---

## Pattern ปัญหาที่พบบ่อย

### 1. Server Overload (เซิร์ฟเวอร์รับไม่ไหว)

**อาการ:**
```
http_req_duration: avg=500ms → 1s → 2s → 5s (เพิ่มขึ้นเรื่อยๆ)
http_req_failed: 0% → 1% → 5% → 20% (เพิ่มขึ้นเรื่อยๆ)
http_reqs: 5000/s → 3000/s → 1000/s (ลดลงเรื่อยๆ)
```

**สาเหตุ:**
- CPU/Memory ไม่พอ
- Connection pool หมด
- Database queries ช้า
- Thread pool exhaustion

**แก้ไข:**
- เพิ่ม server resources (scale up)
- เพิ่มจำนวน server (scale out)
- Optimize queries
- เพิ่ม connection pool size

---

### 2. Connection Pool Exhaustion (Connection หมด)

**อาการ:**
```
http_req_blocked: avg=1ms → 100ms → 1s → 5s
http_req_connecting: avg=0s → 500ms → 2s
http_req_waiting: คงที่ประมาณ 50ms
```

**สาเหตุ:**
- Database connection pool เต็ม
- HTTP client connection pool เต็ม
- Too many concurrent connections

**แก้ไข:**
```go
// เพิ่ม connection pool size
db.SetMaxOpenConns(100)     // เพิ่มจาก default
db.SetMaxIdleConns(50)      // เพิ่ม idle connections
db.SetConnMaxLifetime(5*time.Minute)
```

---

### 3. Memory Leak / GC Pressure

**อาการ:**
```
http_req_duration: ส่วนใหญ่ดี แต่มี spike เป็นระยะ
p95=100ms, p99=2000ms (ห่างกันมาก)
กราฟเป็นฟันปลา (sawtooth pattern)
```

**สาเหตุ:**
- Garbage Collection (GC) หยุดการทำงานชั่วคราว
- Memory leak ทำให้ GC ทำงานบ่อย

**แก้ไข:**
```go
// ลด GC pressure
// ใช้ sync.Pool สำหรับ objects ที่สร้างบ่อย
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}
```

---

### 4. Database Bottleneck (ฐานข้อมูลช้า)

**อาการ:**
```
http_req_waiting: avg=500ms (สูงมาก)
http_req_sending, receiving: ต่ำ
```

**สาเหตุ:**
- Slow queries
- Missing indexes
- Lock contention
- Connection pool exhaustion

**วิธีหา:**
```sql
-- PostgreSQL: ดู slow queries
SELECT query, calls, mean_time, total_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;

-- ดู locks
SELECT * FROM pg_locks WHERE NOT granted;
```

---

### 5. Network Saturation (Network เต็ม)

**อาการ:**
```
data_received: 100 MB/s (สูงมาก)
http_req_receiving: avg=500ms (สูง)
http_req_waiting: avg=50ms (ต่ำ)
```

**สาเหตุ:**
- Response body ใหญ่เกินไป
- Network bandwidth ไม่พอ

**แก้ไข:**
- Enable compression (gzip)
- Pagination
- ลด response size

---

### 6. Rate Limiting (ถูกจำกัด Rate)

**อาการ:**
```
http_req_failed: สูงมาก
status 429 Too Many Requests
```

**สาเหตุ:**
- API rate limit
- DDoS protection
- Firewall rules

**แก้ไข:**
- ลด RPS ในการทดสอบ
- Whitelist test IP
- ปรับ rate limit config

---

## วิเคราะห์ด้วย Percentiles

### ทำไม p95/p99 สำคัญ?

```
ตัวอย่าง: 1000 requests
- avg = 100ms
- p95 = 100ms → ดี! 95% เร็วพอ
- p99 = 500ms → มี 10 requests (1%) ที่ช้ามาก

vs

- avg = 100ms
- p95 = 200ms → มี 50 requests (5%) ที่ช้า
- p99 = 2000ms → มี 10 requests (1%) ที่ช้ามากๆ
```

### การตีความ

| ถ้า p99 >> p95 | แสดงว่า |
|----------------|---------|
| p99 สูงกว่า 2-3 เท่า | มี outliers บางส่วน (ปกติ) |
| p99 สูงกว่า 5+ เท่า | มีปัญหากับบาง requests |
| p99 สูงกว่า 10+ เท่า | มีปัญหาร้ายแรง ต้องตรวจสอบ |

---

## Checklist วิเคราะห์ผลลัพธ์

### ✅ Performance OK
```
□ http_req_duration p95 < 500ms
□ http_req_failed < 1%
□ http_reqs ได้ตามเป้า
□ p99 ไม่ห่างจาก p95 มาก (< 2x)
□ ค่าคงที่ตลอดการทดสอบ
```

### ⚠️ ต้องตรวจสอบ
```
□ http_req_duration เพิ่มขึ้นเรื่อยๆ
□ http_req_failed เพิ่มขึ้นเรื่อยๆ
□ p99 >> p95 (มาก)
□ Sawtooth pattern ในกราฟ
□ RPS ลดลงขณะที่ VUs คงที่
```

### ❌ มีปัญหา
```
□ http_req_duration p95 > 2s
□ http_req_failed > 5%
□ Server errors (5xx) เยอะ
□ http_req_blocked สูง
□ Test ไม่สามารถ ramp up ได้
```

---

## Export ผลลัพธ์สำหรับวิเคราะห์

### JSON Output (วิเคราะห์ละเอียด)
```bash
k6 run --out json=results.json script.js

# ดู summary
cat results.json | jq '.metrics.http_req_duration'
```

### CSV Output (สำหรับ Excel/Sheets)
```bash
k6 run --out csv=results.csv script.js
```

### HTML Report
```bash
# ใช้ k6-reporter
k6 run --out json=results.json script.js
# แปลงเป็น HTML
npx k6-html-reporter results.json
```

---

## ตัวอย่างการวิเคราะห์

### Case 1: Test ผ่าน ✅
```
http_req_duration: avg=85ms p95=180ms p99=250ms
http_req_failed: 0.1%
http_reqs: 10000/s

วิเคราะห์:
✅ Response time ดีมาก (p95 < 200ms)
✅ Error rate ต่ำมาก (< 1%)
✅ ได้ throughput ตามเป้า (10,000 RPS)
✅ p99/p95 ratio = 1.4 (ปกติ)
```

### Case 2: มีปัญหา Response Time ⚠️
```
http_req_duration: avg=450ms p95=1200ms p99=3500ms
http_req_failed: 3%
http_reqs: 5000/s

วิเคราะห์:
⚠️ Response time สูง (p95 > 1s)
⚠️ Error rate พอรับได้ (< 5%)
❌ ได้ throughput แค่ 50% ของเป้า
⚠️ p99/p95 ratio = 2.9 (มี outliers)

แนวทางแก้ไข:
1. ตรวจสอบ slow queries
2. เพิ่ม database indexes
3. เพิ่ม caching
```

### Case 3: Server ล้มเหลว ❌
```
http_req_duration: avg=5000ms p95=10000ms
http_req_failed: 45%
http_reqs: 500/s

วิเคราะห์:
❌ Response time สูงมาก (5 วินาที avg)
❌ Error rate สูงมาก (45%)
❌ Throughput ต่ำมาก (500/s)

แนวทางแก้ไข:
1. ลด load และทดสอบใหม่
2. เพิ่ม server resources
3. Profile หา bottleneck
4. ตรวจสอบ logs หา error
```

---

## อ่านต่อ

- [07 - Best Practices](./07-best-practices.md)
