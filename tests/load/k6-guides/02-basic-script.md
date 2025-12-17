# เขียน K6 Script เบื้องต้น

## โครงสร้างของ K6 Script

K6 script มี 4 ส่วนหลัก:

```javascript
// 1. Import modules (นำเข้าโมดูล)
import http from 'k6/http';
import { check, sleep } from 'k6';

// 2. Options - ตั้งค่าการทดสอบ
export const options = {
    vus: 10,
    duration: '30s',
};

// 3. Setup (optional) - รันครั้งเดียวก่อนเริ่ม test
export function setup() {
    // เตรียมข้อมูล, login, etc.
    return { token: 'xxx' };
}

// 4. Default function - รันซ้ำทุก iteration
export default function(data) {
    // test logic ที่รันซ้ำๆ
    http.get('https://api.example.com');
    sleep(1);
}

// 5. Teardown (optional) - รันครั้งเดียวหลังจบ test
export function teardown(data) {
    // cleanup
}
```

---

## Lifecycle (วงจรชีวิต) ของ K6 Script

```
┌─────────────────────────────────────────────────────────┐
│  1. init code (import, const, etc.) - รัน 1 ครั้งต่อ VU │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  2. setup() - รัน 1 ครั้งเท่านั้น (ก่อนเริ่ม test)       │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  3. default function - รันซ้ำ (iteration) โดยทุก VUs    │
│     ┌──────────────────────────────────────────────┐    │
│     │  VU 1: iteration 1 → 2 → 3 → ...             │    │
│     │  VU 2: iteration 1 → 2 → 3 → ...             │    │
│     │  VU 3: iteration 1 → 2 → 3 → ...             │    │
│     └──────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  4. teardown() - รัน 1 ครั้งเท่านั้น (หลังจบ test)       │
└─────────────────────────────────────────────────────────┘
```

---

## HTTP Requests

### GET Request
```javascript
import http from 'k6/http';

export default function() {
    // Simple GET
    const response = http.get('https://api.example.com/users');

    // GET with headers
    const params = {
        headers: {
            'Authorization': 'Bearer token123',
            'Content-Type': 'application/json',
        },
    };
    const response2 = http.get('https://api.example.com/users', params);
}
```

### POST Request
```javascript
import http from 'k6/http';

export default function() {
    // POST with JSON body
    const payload = JSON.stringify({
        username: 'testuser',
        password: 'password123',
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post('https://api.example.com/login', payload, params);
}
```

### PUT / PATCH / DELETE
```javascript
import http from 'k6/http';

export default function() {
    const url = 'https://api.example.com/users/1';
    const payload = JSON.stringify({ name: 'Updated Name' });
    const params = { headers: { 'Content-Type': 'application/json' } };

    // PUT - แทนที่ทั้ง resource
    http.put(url, payload, params);

    // PATCH - แก้ไขบางส่วน
    http.patch(url, payload, params);

    // DELETE - ลบ
    http.del(url);
}
```

### Response Object (ออบเจกต์ response)
```javascript
const response = http.get('https://api.example.com/users');

// Properties ที่ใช้บ่อย
response.status;          // HTTP status code (200, 404, 500, etc.)
response.body;            // Response body เป็น string
response.json();          // Parse body เป็น JSON object
response.headers;         // Response headers
response.timings.duration; // เวลาทั้งหมด (ms)
response.timings.waiting;  // Time To First Byte (ms)

// ตัวอย่างการใช้
console.log(`Status: ${response.status}`);
console.log(`Duration: ${response.timings.duration}ms`);

const data = response.json();
console.log(`User ID: ${data.id}`);
```

---

## Checks (การตรวจสอบ)

**Check** (เช็ค) คือการตรวจสอบว่า response ถูกต้องหรือไม่ (ไม่ทำให้ test หยุด)

```javascript
import http from 'k6/http';
import { check } from 'k6';

export default function() {
    const response = http.get('https://api.example.com/users');

    // Single check
    check(response, {
        'status is 200': (r) => r.status === 200,
    });

    // Multiple checks
    check(response, {
        'status is 200': (r) => r.status === 200,
        'response time < 500ms': (r) => r.timings.duration < 500,
        'body contains users': (r) => r.body.includes('users'),
        'has correct content-type': (r) => r.headers['Content-Type'] === 'application/json',
    });

    // Check JSON response
    const data = response.json();
    check(data, {
        'has id field': (obj) => obj.id !== undefined,
        'has more than 0 items': (obj) => obj.length > 0,
    });
}
```

### ผลลัพธ์ของ Checks
```
✓ status is 200
✓ response time < 500ms
✗ body contains users
  ↳  95% — ✓ 950 / ✗ 50
```

---

## Sleep (การหน่วงเวลา)

**Sleep** (สลีป) คือการหยุดรอระหว่าง requests เพื่อจำลองพฤติกรรมผู้ใช้จริง

```javascript
import { sleep } from 'k6';

export default function() {
    http.get('https://api.example.com/page1');

    // Sleep แบบต่างๆ
    sleep(1);      // รอ 1 วินาที
    sleep(0.5);    // รอ 500ms
    sleep(0.1);    // รอ 100ms

    http.get('https://api.example.com/page2');
}
```

### Random Sleep (สุ่มเวลารอ)
```javascript
import { sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export default function() {
    http.get('https://api.example.com/page1');

    // สุ่มรอ 1-5 วินาที (จำลอง user อ่านหน้าเว็บ)
    sleep(randomIntBetween(1, 5));

    // สุ่มรอ 100-500ms
    sleep(randomIntBetween(100, 500) / 1000);

    http.get('https://api.example.com/page2');
}
```

---

## Groups (การจัดกลุ่ม)

**Group** (กรุ๊ป) คือการจัดกลุ่ม requests เพื่อดู metrics แยกตามกลุ่ม

```javascript
import http from 'k6/http';
import { group, sleep } from 'k6';

export default function() {
    // Group 1: Login Flow
    group('Login Flow', function() {
        http.get('https://api.example.com/login');
        http.post('https://api.example.com/auth', JSON.stringify({
            username: 'user',
            password: 'pass',
        }));
    });

    sleep(1);

    // Group 2: Browse Products
    group('Browse Products', function() {
        http.get('https://api.example.com/products');
        http.get('https://api.example.com/products/1');
    });

    sleep(1);

    // Group 3: Checkout
    group('Checkout', function() {
        http.post('https://api.example.com/cart/add', JSON.stringify({
            product_id: 1,
            quantity: 2,
        }));
        http.post('https://api.example.com/checkout');
    });
}
```

### ผลลัพธ์แยกตาม Group
```
█ Login Flow
  ✓ status is 200

█ Browse Products
  ✓ status is 200

█ Checkout
  ✗ status is 200
    ↳  90% — ✓ 90 / ✗ 10
```

---

## Tags (แท็ก)

**Tags** คือการติด label ให้ requests เพื่อ filter metrics ได้

```javascript
import http from 'k6/http';

export default function() {
    // Tag ที่ request level
    const params = {
        tags: {
            name: 'GetUsers',
            type: 'api',
        },
    };
    http.get('https://api.example.com/users', params);

    // Tag อีก request
    http.post('https://api.example.com/login', null, {
        tags: { name: 'Login', type: 'auth' },
    });
}
```

### Filter metrics ด้วย Tags
```javascript
export const options = {
    thresholds: {
        // Threshold สำหรับทุก requests
        'http_req_duration': ['p(95)<500'],

        // Threshold เฉพาะ requests ที่มี tag name:GetUsers
        'http_req_duration{name:GetUsers}': ['p(95)<200'],

        // Threshold เฉพาะ requests ที่มี tag type:auth
        'http_req_duration{type:auth}': ['p(95)<300'],
    },
};
```

---

## Environment Variables (ตัวแปรสภาพแวดล้อม)

### กำหนดค่าผ่าน command line
```bash
# วิธีที่ 1: -e flag
k6 run -e BASE_URL=https://api.staging.com -e TOKEN=abc123 script.js

# วิธีที่ 2: export environment variable
export BASE_URL=https://api.staging.com
export TOKEN=abc123
k6 run script.js
```

### ใช้ใน script
```javascript
// อ่านค่าจาก environment variable
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN || 'default-token';

export default function() {
    const params = {
        headers: {
            'Authorization': `Bearer ${TOKEN}`,
        },
    };

    http.get(`${BASE_URL}/api/users`, params);
}
```

---

## ตัวอย่าง Script สมบูรณ์

```javascript
// booking-test.js - ทดสอบ Booking API

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Configuration (การตั้งค่า)
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

// Test options
export const options = {
    vus: 10,
    duration: '1m',
    thresholds: {
        'http_req_duration': ['p(95)<500'],
        'http_req_failed': ['rate<0.05'],
        'checks': ['rate>0.95'],
    },
};

// Setup - Login และเตรียมข้อมูล
export function setup() {
    console.log(`Testing against: ${BASE_URL}`);

    // Login ถ้าไม่มี token
    let token = AUTH_TOKEN;
    if (!token) {
        const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
            email: 'test@example.com',
            password: 'password123',
        }), {
            headers: { 'Content-Type': 'application/json' },
        });

        if (loginRes.status === 200) {
            token = loginRes.json().token;
        }
    }

    return { token };
}

// Main test function
export default function(data) {
    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${data.token}`,
        },
    };

    // Group 1: ดูรายการ Events
    group('Browse Events', function() {
        const eventsRes = http.get(`${BASE_URL}/events`, params);

        check(eventsRes, {
            'events: status 200': (r) => r.status === 200,
            'events: has data': (r) => r.json().data !== undefined,
        });

        sleep(randomIntBetween(1, 3));
    });

    // Group 2: ดูรายละเอียด Event
    group('View Event Detail', function() {
        const eventId = 'event-123';  // ปกติจะสุ่มจาก list
        const detailRes = http.get(`${BASE_URL}/events/${eventId}`, params);

        check(detailRes, {
            'detail: status 200': (r) => r.status === 200,
        });

        sleep(randomIntBetween(2, 5));
    });

    // Group 3: จอง Ticket
    group('Reserve Seats', function() {
        const payload = JSON.stringify({
            event_id: 'event-123',
            zone_id: 'zone-456',
            quantity: randomIntBetween(1, 4),
        });

        const reserveRes = http.post(`${BASE_URL}/bookings/reserve`, payload, {
            ...params,
            tags: { name: 'ReserveSeats' },
        });

        check(reserveRes, {
            'reserve: status 201': (r) => r.status === 201,
            'reserve: has booking_id': (r) => r.json().booking_id !== undefined,
            'reserve: response < 1s': (r) => r.timings.duration < 1000,
        });
    });

    // รอก่อน iteration ถัดไป
    sleep(randomIntBetween(1, 2));
}

// Teardown - สรุปผล
export function teardown(data) {
    console.log('Test completed!');
}
```

### รัน Script
```bash
# รันปกติ
k6 run booking-test.js

# รันพร้อมกำหนด VUs และ duration
k6 run --vus 50 --duration 5m booking-test.js

# รันพร้อม environment variables
k6 run -e BASE_URL=https://api.staging.com -e AUTH_TOKEN=xxx booking-test.js

# รันพร้อม Web Dashboard
k6 run --out web-dashboard booking-test.js
```

---

## สรุป

| Component | หน้าที่ |
|-----------|--------|
| **import** | นำเข้า modules ที่ต้องใช้ |
| **options** | ตั้งค่า VUs, duration, thresholds |
| **setup()** | เตรียมข้อมูลก่อน test (รัน 1 ครั้ง) |
| **default function** | Logic หลักที่รันซ้ำทุก iteration |
| **teardown()** | Cleanup หลังจบ test (รัน 1 ครั้ง) |
| **check()** | ตรวจสอบ response |
| **group()** | จัดกลุ่ม requests |
| **sleep()** | หน่วงเวลาระหว่าง requests |
| **tags** | ติด label เพื่อ filter metrics |

---

## อ่านต่อ

- [03 - Scenarios และ Executors](./03-scenarios.md)
- [04 - Metrics และ Thresholds](./04-metrics.md)
