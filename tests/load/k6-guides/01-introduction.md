# K6 คืออะไร? - บทนำสู่ Load Testing

## K6 (เค-ซิกซ์) คืออะไร?

**K6** คือ open-source load testing tool (เครื่องมือทดสอบโหลด) ที่พัฒนาโดย Grafana Labs ออกแบบมาให้เขียน script ด้วย JavaScript และรันได้อย่างมีประสิทธิภาพด้วย Go runtime

### ทำไมต้อง K6?

| ข้อดี | อธิบาย |
|-------|--------|
| **Developer-friendly** (เป็นมิตรกับนักพัฒนา) | เขียน script ด้วย JavaScript ที่คุ้นเคย |
| **High Performance** (ประสิทธิภาพสูง) | รันด้วย Go engine ใช้ resource น้อย |
| **CI/CD Integration** (รวมกับระบบอัตโนมัติ) | ใช้ใน pipeline ได้ง่าย |
| **Modern Protocol Support** | รองรับ HTTP/1.1, HTTP/2, WebSocket, gRPC |
| **Built-in Metrics** (เมตริกในตัว) | มีการวัดผลพร้อมใช้งานทันที |

---

## คำศัพท์พื้นฐานที่ต้องรู้

### 1. Virtual User - VU (วี-ยู)
**Virtual User** (ผู้ใช้จำลอง) คือ user ที่ k6 สร้างขึ้นมาเพื่อจำลองการใช้งานจริง

```
1 VU = 1 ผู้ใช้จำลองที่ส่ง request ไปยัง server
100 VUs = จำลอง 100 คนใช้งานพร้อมกัน
```

### 2. Iteration (อิท-เทอ-เร-ชัน)
**Iteration** คือ 1 รอบการทำงานของ test function ตั้งแต่ต้นจนจบ

```javascript
export default function() {
    // ทุกอย่างในนี้ = 1 iteration
    http.get('https://api.example.com/users');
    sleep(1);
}
// เมื่อจบ function แล้ววนกลับมาใหม่ = iteration ถัดไป
```

### 3. Request Rate - RPS (อาร์-พี-เอส)
**Requests Per Second** (จำนวน request ต่อวินาที) คือจำนวน HTTP requests ที่ส่งได้ใน 1 วินาที

```
1,000 RPS = ส่ง 1,000 requests ต่อวินาที
10,000 RPS = ส่ง 10,000 requests ต่อวินาที (เป้าหมายของโปรเจคนี้)
```

### 4. Response Time / Latency (เลเทนซี)
**Response Time** (เวลาตอบกลับ) คือเวลาตั้งแต่ส่ง request จนได้รับ response

```
< 100ms = เร็วมาก (excellent)
100-500ms = ดี (good)
500ms-1s = พอใช้ (acceptable)
> 1s = ช้า (slow)
```

### 5. Percentile (เปอร์เซ็นไทล์) - p50, p90, p95, p99

**Percentile** คือค่าที่บอกว่า X% ของ requests มีค่าน้อยกว่าหรือเท่ากับค่านี้

| Percentile | ความหมาย | ตัวอย่าง |
|------------|----------|---------|
| **p50** (median) | 50% ของ requests เร็วกว่านี้ | p50 = 100ms หมายถึง ครึ่งหนึ่งเร็วกว่า 100ms |
| **p90** | 90% ของ requests เร็วกว่านี้ | p90 = 200ms หมายถึง 90% เร็วกว่า 200ms |
| **p95** | 95% ของ requests เร็วกว่านี้ | p95 = 300ms (ค่าที่ใช้บ่อยที่สุด) |
| **p99** | 99% ของ requests เร็วกว่านี้ | p99 = 500ms (ดู worst case) |

```
ทำไม p95/p99 สำคัญกว่า average?

Average (ค่าเฉลี่ย) ซ่อนปัญหาได้ เช่น:
- 99 requests ใช้เวลา 10ms
- 1 request ใช้เวลา 10,000ms (10 วินาที!)
- Average = (99×10 + 10000) / 100 = 109ms ← ดูเหมือนดี

แต่ p99 = 10,000ms ← เห็นปัญหาชัดเจน!
```

### 6. Throughput (ทรูพุท)
**Throughput** คือปริมาณงานที่ระบบรองรับได้ในช่วงเวลาหนึ่ง

```
Throughput = จำนวน successful requests / เวลา
เช่น 50,000 requests สำเร็จใน 10 วินาที = 5,000 RPS throughput
```

### 7. Threshold (เทรชโฮลด์)
**Threshold** คือเกณฑ์ที่ตั้งไว้ว่า test ผ่านหรือไม่ผ่าน

```javascript
thresholds: {
    'http_req_duration': ['p(95)<500'],  // 95% ต้องเร็วกว่า 500ms
    'http_req_failed': ['rate<0.01'],    // fail ต้องน้อยกว่า 1%
}
// ถ้าไม่ผ่าน threshold = test FAILED
```

### 8. Scenario (ซีนาริโอ)
**Scenario** คือรูปแบบการทดสอบที่กำหนดว่าจะส่ง load อย่างไร

```javascript
scenarios: {
    smoke: { /* ทดสอบเบาๆ */ },
    stress: { /* ทดสอบหนักๆ */ },
    spike: { /* จำลอง traffic พุ่งสูงทันที */ },
}
```

---

## ติดตั้ง K6

### macOS
```bash
brew install k6
```

### Windows
```bash
choco install k6
# หรือ
winget install k6
```

### Linux (Debian/Ubuntu)
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
    --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
    | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### Docker
```bash
docker run --rm -i grafana/k6 run - <script.js
```

### ตรวจสอบการติดตั้ง
```bash
k6 version
# Output: k6 v0.47.0 (go1.21.0, darwin/arm64)
```

---

## Hello World - Script แรก

สร้างไฟล์ `hello.js`:

```javascript
import http from 'k6/http';
import { sleep } from 'k6';

export default function() {
    // ส่ง GET request
    http.get('https://test.k6.io');

    // รอ 1 วินาทีก่อน iteration ถัดไป
    sleep(1);
}
```

รัน script:

```bash
k6 run hello.js
```

ผลลัพธ์:
```
          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: hello.js
     output: -

  scenarios: (100.00%) 1 scenario, 1 max VUs, 10m30s max duration
           default: 1 iterations for each of 1 VUs (maxDuration: 10m0s)

running (00m01.2s), 0/1 VUs, 1 complete and 0 interrupted iterations
default ✓ [======================================] 1 VUs  00m01.2s/10m0s  1/1 iters, 1 per VU

     data_received..................: 17 kB 14 kB/s
     data_sent......................: 438 B 365 B/s
     http_req_duration..............: avg=215.5ms  min=215.5ms  max=215.5ms  p(90)=215.5ms  p(95)=215.5ms
     http_reqs......................: 1     0.833333/s
     iteration_duration.............: avg=1.22s    min=1.22s    max=1.22s    p(90)=1.22s    p(95)=1.22s
     iterations.....................: 1     0.833333/s
```

---

## ประเภทของ Load Testing

### 1. Smoke Test (สโมค-เทสต์)
**วัตถุประสงค์:** ตรวจสอบว่าระบบทำงานได้ปกติ

```javascript
export const options = {
    vus: 1,
    duration: '30s',
};
```

```
Load: ต่ำมาก (1-5 VUs)
ระยะเวลา: สั้น (30 วินาที - 1 นาที)
ใช้เมื่อ: ก่อน deploy, หลังแก้ bug
```

### 2. Load Test (โหลด-เทสต์)
**วัตถุประสงค์:** ทดสอบภายใต้ load ปกติที่คาดหวัง

```javascript
export const options = {
    stages: [
        { duration: '5m', target: 100 },  // ramp up
        { duration: '10m', target: 100 }, // stay
        { duration: '5m', target: 0 },    // ramp down
    ],
};
```

```
Load: ปกติ (expected traffic)
ระยะเวลา: ปานกลาง (10-30 นาที)
ใช้เมื่อ: ทดสอบ performance ก่อน release
```

### 3. Stress Test (สเตรส-เทสต์)
**วัตถุประสงค์:** หาขีดจำกัดของระบบ

```javascript
export const options = {
    stages: [
        { duration: '2m', target: 100 },
        { duration: '5m', target: 100 },
        { duration: '2m', target: 200 },
        { duration: '5m', target: 200 },
        { duration: '2m', target: 300 },  // เพิ่มไปเรื่อยๆ จนพัง
        { duration: '5m', target: 300 },
        { duration: '2m', target: 0 },
    ],
};
```

```
Load: สูงกว่าปกติ (เกิน expected traffic)
ระยะเวลา: ยาว
ใช้เมื่อ: หา breaking point
```

### 4. Spike Test (สไปค์-เทสต์)
**วัตถุประสงค์:** ทดสอบการรับ traffic ที่พุ่งสูงทันที

```javascript
export const options = {
    stages: [
        { duration: '10s', target: 100 },
        { duration: '1m', target: 100 },
        { duration: '10s', target: 1400 },  // spike!
        { duration: '3m', target: 1400 },
        { duration: '10s', target: 100 },
        { duration: '3m', target: 100 },
        { duration: '10s', target: 0 },
    ],
};
```

```
Load: พุ่งสูงทันที
ระยะเวลา: สั้น (spike) + recovery
ใช้เมื่อ: จำลอง flash sale, viral content
```

### 5. Soak Test / Endurance Test (โซค-เทสต์)
**วัตถุประสงค์:** ทดสอบ memory leak, resource exhaustion

```javascript
export const options = {
    stages: [
        { duration: '5m', target: 100 },
        { duration: '8h', target: 100 },  // รันนานมาก
        { duration: '5m', target: 0 },
    ],
};
```

```
Load: ปกติ
ระยะเวลา: ยาวมาก (หลายชั่วโมง)
ใช้เมื่อ: ตรวจสอบ stability ระยะยาว
```

---

## สรุป

| คำศัพท์ | ความหมาย |
|---------|----------|
| **VU** | ผู้ใช้จำลอง 1 คน |
| **Iteration** | 1 รอบการทำงานของ test function |
| **RPS** | จำนวน requests ต่อวินาที |
| **Latency** | เวลาตอบกลับ |
| **p95** | 95% ของ requests เร็วกว่าค่านี้ |
| **Threshold** | เกณฑ์ผ่าน/ไม่ผ่าน |
| **Scenario** | รูปแบบการทดสอบ |

---

## อ่านต่อ

- [02 - เขียน Script เบื้องต้น](./02-basic-script.md)
- [03 - Scenarios และ Executors](./03-scenarios.md)
- [04 - Metrics และ Thresholds](./04-metrics.md)
