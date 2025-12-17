# Metrics และ Thresholds

## Metrics (เมตริก) คืออะไร?

**Metrics** คือ "ตัวชี้วัด" ที่ K6 เก็บระหว่างการทดสอบ เช่น:
- จำนวน requests
- เวลาตอบกลับ
- จำนวน errors
- Data ที่รับ/ส่ง

---

## ประเภทของ Metrics

K6 มี metrics 4 ประเภท:

| ประเภท | คำอธิบาย | ตัวอย่าง |
|--------|----------|---------|
| **Counter** (เคาน์เตอร์) | นับจำนวนสะสม | `http_reqs`, `iterations` |
| **Gauge** (เกจ) | ค่าปัจจุบัน ณ ขณะนั้น | `vus`, `vus_max` |
| **Rate** (เรท) | อัตราส่วน (0-1) | `http_req_failed`, `checks` |
| **Trend** (เทรนด์) | ค่าสถิติ (min, max, avg, percentiles) | `http_req_duration` |

---

## Built-in Metrics (เมตริกในตัว)

### HTTP Metrics

| Metric | ประเภท | คำอธิบาย |
|--------|--------|----------|
| `http_reqs` | Counter | จำนวน HTTP requests ทั้งหมด |
| `http_req_duration` | Trend | เวลาทั้งหมดของ request (ms) |
| `http_req_blocked` | Trend | เวลาที่รอก่อนเริ่ม request |
| `http_req_connecting` | Trend | เวลาที่ใช้สร้าง TCP connection |
| `http_req_tls_handshaking` | Trend | เวลา TLS handshake |
| `http_req_sending` | Trend | เวลาส่ง data ไป server |
| `http_req_waiting` | Trend | เวลารอ response (TTFB) |
| `http_req_receiving` | Trend | เวลารับ data จาก server |
| `http_req_failed` | Rate | อัตราส่วน request ที่ fail |

### การแยกส่วนของ http_req_duration

```
http_req_duration = http_req_sending
                  + http_req_waiting (TTFB)
                  + http_req_receiving

┌─────────────────────────────────────────────────────────────────┐
│                     http_req_duration                           │
├────────────────┬───────────────────────────┬────────────────────┤
│ http_req_      │ http_req_waiting          │ http_req_          │
│ sending        │ (Time To First Byte)      │ receiving          │
│                │                           │                    │
│ ส่ง request    │ รอ server ประมวลผล          │ รับ response       │
└────────────────┴───────────────────────────┴────────────────────┘
```

### Other Built-in Metrics

| Metric | ประเภท | คำอธิบาย |
|--------|--------|----------|
| `vus` | Gauge | จำนวน VUs ที่ active ขณะนั้น |
| `vus_max` | Gauge | จำนวน VUs สูงสุดที่กำหนด |
| `iterations` | Counter | จำนวน iterations ทั้งหมด |
| `iteration_duration` | Trend | เวลาต่อ 1 iteration |
| `data_received` | Counter | ข้อมูลที่รับ (bytes) |
| `data_sent` | Counter | ข้อมูลที่ส่ง (bytes) |
| `checks` | Rate | อัตราส่วน checks ที่ผ่าน |

---

## Custom Metrics (เมตริกกำหนดเอง)

### สร้าง Counter
```javascript
import { Counter } from 'k6/metrics';

// สร้าง counter
const successfulBookings = new Counter('successful_bookings');
const failedBookings = new Counter('failed_bookings');

export default function() {
    const response = http.post(url, payload);

    if (response.status === 201) {
        successfulBookings.add(1);  // นับสำเร็จ
    } else {
        failedBookings.add(1);      // นับไม่สำเร็จ
    }
}
```

### สร้าง Gauge
```javascript
import { Gauge } from 'k6/metrics';

// สร้าง gauge
const availableSeats = new Gauge('available_seats');

export default function() {
    const response = http.get(`${BASE_URL}/zones/1`);
    const data = response.json();

    availableSeats.add(data.available_seats);  // บันทึกค่าปัจจุบัน
}
```

### สร้าง Rate
```javascript
import { Rate } from 'k6/metrics';

// สร้าง rate
const bookingSuccessRate = new Rate('booking_success_rate');
const oversellRate = new Rate('oversell_rate');

export default function() {
    const response = http.post(url, payload);

    // บันทึก success/fail
    bookingSuccessRate.add(response.status === 201);

    // บันทึก oversell
    if (response.status === 409) {
        oversellRate.add(1);  // 1 = เกิด oversell
    } else {
        oversellRate.add(0);  // 0 = ปกติ
    }
}
```

### สร้าง Trend
```javascript
import { Trend } from 'k6/metrics';

// สร้าง trend
const bookingDuration = new Trend('booking_duration');
const paymentDuration = new Trend('payment_duration');

export default function() {
    // วัดเวลา booking
    const startBooking = Date.now();
    http.post(`${BASE_URL}/bookings/reserve`, payload);
    bookingDuration.add(Date.now() - startBooking);

    // วัดเวลา payment
    const startPayment = Date.now();
    http.post(`${BASE_URL}/payments/process`, paymentPayload);
    paymentDuration.add(Date.now() - startPayment);
}
```

---

## Thresholds (เทรชโฮลด์)

**Threshold** คือ "เกณฑ์" ที่ตั้งไว้ว่า test ผ่านหรือไม่ผ่าน

### Basic Thresholds
```javascript
export const options = {
    thresholds: {
        // Counter: ค่าสะสม
        'http_reqs': ['count>1000'],          // ต้องมี > 1000 requests

        // Gauge: ค่าปัจจุบัน
        'vus': ['value<=100'],                // VUs ต้อง <= 100

        // Rate: อัตราส่วน (0-1)
        'http_req_failed': ['rate<0.01'],     // fail < 1%
        'checks': ['rate>0.95'],              // checks ผ่าน > 95%

        // Trend: ค่าสถิติ
        'http_req_duration': ['p(95)<500'],   // 95% เร็วกว่า 500ms
    },
};
```

### Threshold Syntax

#### สำหรับ Counter
```javascript
thresholds: {
    'http_reqs': [
        'count>1000',       // จำนวนรวม > 1000
        'rate>100',         // rate > 100/s
    ],
}
```

#### สำหรับ Gauge
```javascript
thresholds: {
    'vus': [
        'value<=100',       // ค่าปัจจุบัน <= 100
    ],
}
```

#### สำหรับ Rate
```javascript
thresholds: {
    'http_req_failed': [
        'rate<0.01',        // อัตราส่วน < 1% (0.01)
        'rate<0.05',        // อัตราส่วน < 5% (0.05)
    ],
    'checks': [
        'rate>0.95',        // อัตราส่วน > 95%
        'rate==1.0',        // ต้อง 100%
    ],
}
```

#### สำหรับ Trend
```javascript
thresholds: {
    'http_req_duration': [
        'avg<200',          // ค่าเฉลี่ย < 200ms
        'min<100',          // ค่าต่ำสุด < 100ms
        'max<1000',         // ค่าสูงสุด < 1000ms
        'med<150',          // median < 150ms
        'p(90)<300',        // 90th percentile < 300ms
        'p(95)<500',        // 95th percentile < 500ms
        'p(99)<1000',       // 99th percentile < 1000ms
    ],
}
```

---

## Threshold พร้อม Tags

Filter metrics ตาม tags:

```javascript
export const options = {
    thresholds: {
        // Threshold สำหรับทุก requests
        'http_req_duration': ['p(95)<500'],

        // Threshold เฉพาะ requests ที่มี tag name:GetEvents
        'http_req_duration{name:GetEvents}': ['p(95)<200'],

        // Threshold เฉพาะ requests ที่มี tag name:ReserveSeats
        'http_req_duration{name:ReserveSeats}': ['p(95)<1000'],

        // Threshold เฉพาะ scenario
        'http_req_duration{scenario:browse}': ['p(95)<100'],
        'http_req_duration{scenario:reserve}': ['p(95)<500'],
    },
};

export default function() {
    // Tag requests
    http.get(url, { tags: { name: 'GetEvents' } });
    http.post(url, payload, { tags: { name: 'ReserveSeats' } });
}
```

---

## Abort on Threshold Fail

หยุด test ทันทีถ้า threshold fail:

```javascript
export const options = {
    thresholds: {
        'http_req_failed': [
            {
                threshold: 'rate<0.1',      // fail < 10%
                abortOnFail: true,          // หยุดทันทีถ้า fail เกิน 10%
                delayAbortEval: '10s',      // รอ 10 วินาทีก่อนเริ่มเช็ค
            },
        ],
        'http_req_duration': [
            {
                threshold: 'p(95)<2000',    // p95 < 2 วินาที
                abortOnFail: true,
            },
        ],
    },
};
```

---

## ตัวอย่างจากโปรเจค Booking Rush

```javascript
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const reserveSuccessRate = new Rate('reserve_success_rate');
const reserveFailRate = new Rate('reserve_fail_rate');
const reserveDuration = new Trend('reserve_duration');
const insufficientSeatsErrors = new Counter('insufficient_seats_errors');
const serverErrors = new Counter('server_errors');

export const options = {
    thresholds: {
        // Built-in metrics
        'http_req_duration': ['p(95)<500', 'p(99)<1000'],
        'http_req_failed': ['rate<0.05'],

        // Custom metrics
        'reserve_success_rate': ['rate>0.95'],
        'reserve_duration': ['p(95)<500', 'avg<200'],
    },
};

export function reserveSeats() {
    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/bookings/reserve`, payload, params);
    const duration = Date.now() - startTime;

    // Record custom metrics
    reserveDuration.add(duration);

    const success = response.status === 201;
    reserveSuccessRate.add(success);
    reserveFailRate.add(!success);

    // Track specific error types
    if (!success) {
        if (response.status === 409) {
            insufficientSeatsErrors.add(1);
        } else if (response.status >= 500) {
            serverErrors.add(1);
        }
    }
}
```

---

## ผลลัพธ์ Thresholds

### เมื่อ Test ผ่าน
```
     ✓ http_req_duration..............: avg=89.21ms  min=45.12ms  med=82.34ms  max=432.21ms p(90)=142.32ms p(95)=189.43ms
     ✓ http_req_failed................: 0.23%   ✓ 23       ✗ 9977
     ✓ reserve_success_rate...........: 99.77%  ✓ 9977     ✗ 23
     ✓ reserve_duration...............: avg=85.43ms  p(95)=178.21ms
```

### เมื่อ Test ไม่ผ่าน
```
     ✗ http_req_duration..............: avg=892.21ms min=145.12ms med=782.34ms max=4322.1ms p(90)=1423.2ms p(95)=1894.3ms
       ✗ p(95)<500
       ✗ p(99)<1000
     ✗ http_req_failed................: 12.34%  ✓ 1234     ✗ 8766
       ✗ rate<0.05
     ✗ reserve_success_rate...........: 87.66%  ✓ 8766     ✗ 1234
       ✗ rate>0.95
```

---

## Export Metrics

### JSON Output
```bash
k6 run --out json=results.json script.js
```

### CSV Output
```bash
k6 run --out csv=results.csv script.js
```

### InfluxDB (สำหรับ Grafana)
```bash
k6 run --out influxdb=http://localhost:8086/k6 script.js
```

### Prometheus
```bash
# ใช้ k6 extension
k6 run --out experimental-prometheus-rw script.js
```

---

## สรุป Metrics ที่ควรดู

| เป้าหมาย | Metric | Threshold แนะนำ |
|----------|--------|----------------|
| **Response Time** | `http_req_duration` | `p(95)<500ms` |
| **Error Rate** | `http_req_failed` | `rate<0.01` (1%) |
| **Throughput** | `http_reqs` | `rate>1000` |
| **Success Rate** | `checks` | `rate>0.95` |
| **Availability** | custom rate | `rate>0.99` |

---

## อ่านต่อ

- [05 - อ่านกราฟ Dashboard](./05-dashboard.md)
- [06 - วิเคราะห์ผลลัพธ์](./06-analyze-results.md)
