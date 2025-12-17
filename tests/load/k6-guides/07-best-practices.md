# K6 Best Practices

## 1. Script Best Practices

### ใช้ Environment Variables
```javascript
// ❌ ไม่ดี: Hardcode ค่า
const BASE_URL = 'http://localhost:8080';
const TOKEN = 'abc123';

// ✅ ดี: ใช้ environment variables
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN || '';
```

### ใช้ SharedArray สำหรับ Test Data
```javascript
import { SharedArray } from 'k6/data';

// ❌ ไม่ดี: ทุก VU โหลด data ซ้ำ (เปลือง memory)
const users = JSON.parse(open('./users.json'));

// ✅ ดี: โหลดครั้งเดียว share ทุก VU
const users = new SharedArray('users', function() {
    return JSON.parse(open('./users.json'));
});
```

### หลีกเลี่ยง open() ใน Default Function
```javascript
// ❌ ไม่ดี: อ่านไฟล์ทุก iteration
export default function() {
    const data = JSON.parse(open('./data.json'));  // ช้า!
}

// ✅ ดี: อ่านไฟล์ครั้งเดียวตอน init
const data = JSON.parse(open('./data.json'));

export default function() {
    // ใช้ data
}
```

### ใช้ Tags อย่างเหมาะสม
```javascript
// ✅ ติด tags เพื่อ filter metrics ได้
http.get(url, {
    tags: {
        name: 'GetUserProfile',
        endpoint: '/users/profile',
        type: 'read',
    },
});

// ตั้ง thresholds ตาม tags
export const options = {
    thresholds: {
        'http_req_duration{type:read}': ['p(95)<200'],
        'http_req_duration{type:write}': ['p(95)<500'],
    },
};
```

---

## 2. Test Data Best Practices

### เตรียม Data ล่วงหน้า
```javascript
// ✅ Pre-generate test data
// สร้างไฟล์ users.json ก่อนรัน test

// users.json
[
    {"id": "user-1", "email": "user1@test.com", "token": "xxx"},
    {"id": "user-2", "email": "user2@test.com", "token": "yyy"},
    // ... 1000 users
]

// script.js
const users = new SharedArray('users', function() {
    return JSON.parse(open('./users.json'));
});

export default function() {
    const user = users[__VU % users.length];  // กระจาย users
    http.get(url, {
        headers: { 'Authorization': `Bearer ${user.token}` }
    });
}
```

### สุ่มข้อมูลอย่างถูกวิธี
```javascript
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const eventIds = ['event-1', 'event-2', 'event-3'];
const zones = ['zone-a', 'zone-b', 'zone-c'];

export default function() {
    const eventId = randomItem(eventIds);
    const zoneId = randomItem(zones);
    const quantity = randomIntBetween(1, 4);

    http.post(url, JSON.stringify({
        event_id: eventId,
        zone_id: zoneId,
        quantity: quantity,
    }));
}
```

### ใช้ Unique Data ต่อ Request
```javascript
// สร้าง unique idempotency key
const idempotencyKey = `${__VU}-${__ITER}-${Date.now()}`;

http.post(url, payload, {
    headers: {
        'X-Idempotency-Key': idempotencyKey,
    },
});
```

---

## 3. Scenario Best Practices

### เริ่มจาก Smoke Test เสมอ
```javascript
const scenarios = {
    // 1. เริ่มจาก smoke test ก่อน
    smoke: {
        executor: 'constant-vus',
        vus: 1,
        duration: '30s',
    },

    // 2. ถ้าผ่านค่อยเพิ่ม load
    load: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '2m', target: 100 },
            { duration: '5m', target: 100 },
            { duration: '2m', target: 0 },
        ],
    },
};
```

### ใช้ Ramp Up/Down เสมอ
```javascript
// ❌ ไม่ดี: เริ่มทันที
export const options = {
    vus: 1000,
    duration: '5m',
};

// ✅ ดี: ค่อยๆ เพิ่ม/ลด
export const options = {
    stages: [
        { duration: '1m', target: 1000 },  // ramp up
        { duration: '3m', target: 1000 },  // steady state
        { duration: '1m', target: 0 },     // ramp down
    ],
};
```

### กำหนด preAllocatedVUs เพียงพอ
```javascript
// สำหรับ arrival-rate executors
export const options = {
    scenarios: {
        high_load: {
            executor: 'constant-arrival-rate',
            rate: 10000,
            timeUnit: '1s',
            duration: '5m',

            // ✅ กำหนด VUs เผื่อไว้
            preAllocatedVUs: 2000,  // เริ่มต้น
            maxVUs: 5000,           // สูงสุดที่อนุญาต
        },
    },
};
```

---

## 4. Threshold Best Practices

### ตั้ง Thresholds ที่ Realistic
```javascript
export const options = {
    thresholds: {
        // ✅ ค่าที่เหมาะสม
        'http_req_duration': ['p(95)<500', 'p(99)<1000'],
        'http_req_failed': ['rate<0.01'],

        // ❌ ไม่สมจริง
        // 'http_req_duration': ['p(99)<10'],  // เร็วเกินไป
        // 'http_req_failed': ['rate<0.0001'], // ต่ำเกินไป
    },
};
```

### ใช้ abortOnFail สำหรับ Critical Failures
```javascript
export const options = {
    thresholds: {
        'http_req_failed': [
            {
                threshold: 'rate<0.1',    // fail < 10%
                abortOnFail: true,        // หยุดทันทีถ้าเกิน
                delayAbortEval: '30s',    // รอ 30 วินาทีก่อนเช็ค
            },
        ],
    },
};
```

---

## 5. Check Best Practices

### Check ทุก Response ที่สำคัญ
```javascript
export default function() {
    const response = http.post(url, payload);

    // ✅ Check หลายเงื่อนไข
    check(response, {
        'status is 201': (r) => r.status === 201,
        'has booking_id': (r) => {
            const body = r.json();
            return body.booking_id !== undefined;
        },
        'response time OK': (r) => r.timings.duration < 1000,
    });
}
```

### Track Error Types
```javascript
import { Counter } from 'k6/metrics';

const errors = {
    client_error: new Counter('errors_4xx'),
    server_error: new Counter('errors_5xx'),
    timeout: new Counter('errors_timeout'),
    validation: new Counter('errors_validation'),
};

export default function() {
    const response = http.post(url, payload);

    if (response.status >= 400 && response.status < 500) {
        errors.client_error.add(1);
    } else if (response.status >= 500) {
        errors.server_error.add(1);
    }

    if (response.timings.duration > 10000) {
        errors.timeout.add(1);
    }
}
```

---

## 6. Performance Best Practices

### ลด Memory Usage
```javascript
// ✅ ใช้ SharedArray
const data = new SharedArray('data', () => JSON.parse(open('./data.json')));

// ✅ อย่าเก็บ response body ถ้าไม่จำเป็น
const response = http.get(url);
// ใช้ response.status, response.timings แทน response.body

// ✅ ใช้ discardResponseBodies สำหรับ high-throughput tests
export const options = {
    discardResponseBodies: true,  // ไม่เก็บ body
};
```

### Sleep อย่างเหมาะสม
```javascript
import { sleep } from 'k6';

export default function() {
    http.get(url);

    // ❌ ไม่ดี: ไม่มี sleep (overwhelm server)
    // ไม่มี sleep

    // ✅ ดี: มี sleep จำลองพฤติกรรมจริง
    sleep(1);  // รอ 1 วินาที

    // ✅ ดีกว่า: สุ่ม sleep
    sleep(Math.random() * 2 + 0.5);  // 0.5-2.5 วินาที
}
```

### Batch Requests ถ้าเป็นไปได้
```javascript
import http from 'k6/http';

// ❌ ไม่ดี: ส่งทีละ request
http.get('http://api.com/users/1');
http.get('http://api.com/users/2');
http.get('http://api.com/users/3');

// ✅ ดี: ส่งพร้อมกัน
const responses = http.batch([
    ['GET', 'http://api.com/users/1'],
    ['GET', 'http://api.com/users/2'],
    ['GET', 'http://api.com/users/3'],
]);
```

---

## 7. Infrastructure Best Practices

### รัน K6 จาก Machine ที่เหมาะสม
```
❌ ไม่ดี:
- รัน K6 บน Macbook ที่ต่อ WiFi
- รัน K6 จาก network ที่ไกลจาก server

✅ ดี:
- รัน K6 บน dedicated machine
- รัน K6 ใน same region กับ server
- ใช้ wired connection
```

### Monitor K6 Machine
```bash
# ดู resource usage ขณะรัน
htop
# หรือ
top -p $(pgrep k6)
```

### กำหนด File Descriptors
```bash
# เพิ่ม limit สำหรับ high-load tests
ulimit -n 65535
```

---

## 8. Common Mistakes (ข้อผิดพลาดที่พบบ่อย)

### ❌ ไม่ทำ Smoke Test ก่อน
```javascript
// เริ่มด้วย 10,000 VUs ทันที → Server crash
export const options = { vus: 10000, duration: '5m' };

// ✅ เริ่มจาก 1 VU ก่อน
export const options = { vus: 1, duration: '30s' };
```

### ❌ ใช้ constant-vus สำหรับ RPS target
```javascript
// ❌ ไม่รู้ RPS จริงๆ
export const options = {
    vus: 1000,  // RPS แปรผันตาม response time
    duration: '5m',
};

// ✅ กำหนด RPS ตรงๆ
export const options = {
    scenarios: {
        target_rps: {
            executor: 'constant-arrival-rate',
            rate: 10000,  // 10,000 RPS ตายตัว
            timeUnit: '1s',
            duration: '5m',
            preAllocatedVUs: 2000,
            maxVUs: 5000,
        },
    },
};
```

### ❌ ไม่ใส่ Timeout
```javascript
// ❌ ไม่มี timeout → รอไม่รู้จบ
http.get(url);

// ✅ ใส่ timeout
http.get(url, { timeout: '10s' });
```

### ❌ Test Data ซ้ำกัน
```javascript
// ❌ ทุก request ใช้ data เดียวกัน
http.post(url, JSON.stringify({ user_id: 'user-1' }));

// ✅ สุ่ม data
const userId = users[__VU % users.length].id;
http.post(url, JSON.stringify({ user_id: userId }));
```

---

## 9. Checklist ก่อนรัน Load Test

### Pre-Test
```
□ ทำ Smoke test ผ่านแล้ว
□ Test data พร้อม (users, tokens, etc.)
□ Server ว่างจาก traffic อื่น
□ Monitoring พร้อม (CPU, Memory, DB)
□ Log level ไม่ verbose เกินไป
□ K6 machine มี resource เพียงพอ
□ Network stable
```

### During Test
```
□ Monitor K6 machine (CPU, Memory)
□ Monitor server metrics
□ ดู error logs
□ พร้อมหยุด test ถ้ามีปัญหา
```

### Post-Test
```
□ บันทึกผลลัพธ์
□ วิเคราะห์ bottlenecks
□ Document findings
□ Plan improvements
```

---

## 10. สรุป Best Practices

| หมวด | Best Practice |
|------|---------------|
| **Script** | ใช้ ENV vars, SharedArray, Tags |
| **Data** | Pre-generate, unique per request |
| **Scenario** | Smoke first, ramp up/down |
| **Threshold** | Realistic, abortOnFail |
| **Check** | ทุก response สำคัญ |
| **Performance** | ลด memory, sleep, batch |
| **Infra** | Same region, enough resources |

---

## Resources เพิ่มเติม

- [K6 Documentation](https://k6.io/docs/)
- [K6 Examples](https://github.com/grafana/k6/tree/master/examples)
- [K6 Blog](https://k6.io/blog/)
- [K6 Community](https://community.k6.io/)
