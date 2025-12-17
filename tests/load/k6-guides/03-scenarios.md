# Scenarios และ Executors

## Scenario (ซีนาริโอ) คืออะไร?

**Scenario** คือ "ฉากการทดสอบ" ที่กำหนดว่า:
- จะส่ง load แบบไหน (รูปแบบ)
- ใช้กี่ VUs หรือกี่ RPS
- นานเท่าไหร่
- รัน function ไหน

### ทำไมต้องใช้ Scenarios?

```javascript
// ❌ แบบเก่า: กำหนด vus + duration ตรงๆ
export const options = {
    vus: 100,
    duration: '5m',
};

// ✅ แบบใหม่: ใช้ scenarios ยืดหยุ่นกว่า
export const options = {
    scenarios: {
        smoke: { /* ทดสอบเบาๆ */ },
        load: { /* ทดสอบ load ปกติ */ },
        stress: { /* ทดสอบหนักๆ */ },
    },
};
```

**ข้อดีของ Scenarios:**
- รัน test หลายรูปแบบพร้อมกัน
- กำหนด executor (ตัวควบคุม) ที่แตกต่างกัน
- เลือกรัน function ต่างกันได้
- ตั้งเวลาเริ่มต้นของแต่ละ scenario ได้

---

## Executor (เอ็กซีคิวเตอร์) คืออะไร?

**Executor** คือ "ตัวควบคุม" ที่กำหนดว่าจะสร้าง load อย่างไร

K6 มี executor 6 แบบ:

| Executor | ควบคุมด้วย | เหมาะกับ |
|----------|-----------|---------|
| `shared-iterations` | จำนวน iterations ทั้งหมด | ทดสอบ N ครั้งให้จบ |
| `per-vu-iterations` | iterations ต่อ VU | ทุก VU ทำ N ครั้ง |
| `constant-vus` | จำนวน VUs คงที่ | Load test ทั่วไป |
| `ramping-vus` | VUs เปลี่ยนตามเวลา | Ramp up/down |
| `constant-arrival-rate` | RPS คงที่ | ทดสอบ throughput เป้าหมาย |
| `ramping-arrival-rate` | RPS เปลี่ยนตามเวลา | Spike test |

---

## 1. shared-iterations (แชร์-อิทเทอเรชัน)

**หลักการ:** กำหนดจำนวน iterations ทั้งหมด แล้วให้ VUs แบ่งกันทำ

```javascript
export const options = {
    scenarios: {
        shared_iterations_test: {
            executor: 'shared-iterations',
            vus: 10,              // จำนวน VUs
            iterations: 100,      // iterations ทั้งหมด
            maxDuration: '5m',    // เวลาสูงสุด
        },
    },
};
```

**การทำงาน:**
```
VUs = 10, Iterations = 100
แต่ละ VU หยิบ iteration มาทำจาก pool รวม
- VU 1 ทำไป 12 iterations
- VU 2 ทำไป 8 iterations
- VU 3 ทำไป 15 iterations
- ...
รวมกัน = 100 iterations แล้วหยุด
```

**เหมาะกับ:**
- ทดสอบ N ครั้งแล้วจบ
- ไม่สนใจว่า VU ไหนทำกี่ครั้ง
- ต้องการให้เสร็จเร็วที่สุด

---

## 2. per-vu-iterations (เปอร์-วียู-อิทเทอเรชัน)

**หลักการ:** กำหนดว่าแต่ละ VU ต้องทำกี่ iterations

```javascript
export const options = {
    scenarios: {
        per_vu_test: {
            executor: 'per-vu-iterations',
            vus: 10,              // จำนวน VUs
            iterations: 20,       // iterations ต่อ VU
            maxDuration: '5m',
        },
    },
};
```

**การทำงาน:**
```
VUs = 10, Iterations per VU = 20
- VU 1 ทำ 20 iterations
- VU 2 ทำ 20 iterations
- VU 3 ทำ 20 iterations
- ...
รวม = 10 × 20 = 200 iterations
```

**เหมาะกับ:**
- ทดสอบว่าแต่ละ user ทำ N ครั้งได้ไหม
- ต้องการความเท่าเทียมของ VUs

---

## 3. constant-vus (คอนสแตนท์-วียู)

**หลักการ:** รักษาจำนวน VUs คงที่ตลอดเวลา

```javascript
export const options = {
    scenarios: {
        constant_load: {
            executor: 'constant-vus',
            vus: 50,              // จำนวน VUs คงที่
            duration: '5m',       // ระยะเวลา
        },
    },
};
```

**การทำงาน:**
```
     VUs
      │
   50 │ ─────────────────────────
      │
      └──────────────────────────► เวลา
        0                    5m
```

**เหมาะกับ:**
- Load test ทั่วไป
- ทดสอบ concurrent users คงที่

---

## 4. ramping-vus (แรมปิ้ง-วียู) ⭐

**หลักการ:** ค่อยๆ เพิ่ม/ลด VUs ตาม stages ที่กำหนด

```javascript
export const options = {
    scenarios: {
        ramping_test: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '2m', target: 100 },   // 0 → 100 ใน 2 นาที
                { duration: '5m', target: 100 },   // คงที่ 100 ไว้ 5 นาที
                { duration: '2m', target: 200 },   // 100 → 200 ใน 2 นาที
                { duration: '5m', target: 200 },   // คงที่ 200 ไว้ 5 นาที
                { duration: '2m', target: 0 },     // 200 → 0 ใน 2 นาที
            ],
        },
    },
};
```

**การทำงาน:**
```
     VUs
      │
  200 │              ┌───────────┐
      │             /             \
  100 │     ┌──────┘               \
      │    /                        \
    0 │───┘                          └───
      └──────────────────────────────────► เวลา
         2m   7m    9m    14m   16m
```

**เหมาะกับ:**
- Ramp up test (ค่อยๆ เพิ่ม load)
- Stress test (เพิ่มจนหา breaking point)
- Soak test (รักษา load นานๆ)

### ตัวอย่าง Patterns

#### Pattern 1: Simple Ramp Up/Down
```javascript
stages: [
    { duration: '5m', target: 100 },   // ramp up
    { duration: '10m', target: 100 },  // steady
    { duration: '5m', target: 0 },     // ramp down
]
```

#### Pattern 2: Steps (ขั้นบันได)
```javascript
stages: [
    { duration: '2m', target: 50 },
    { duration: '3m', target: 50 },
    { duration: '2m', target: 100 },
    { duration: '3m', target: 100 },
    { duration: '2m', target: 150 },
    { duration: '3m', target: 150 },
    { duration: '2m', target: 0 },
]
```

#### Pattern 3: Spike
```javascript
stages: [
    { duration: '1m', target: 50 },
    { duration: '10s', target: 500 },  // spike!
    { duration: '1m', target: 500 },
    { duration: '10s', target: 50 },   // drop
    { duration: '2m', target: 50 },
    { duration: '1m', target: 0 },
]
```

---

## 5. constant-arrival-rate (คอนสแตนท์-อะไรวัล-เรท) ⭐⭐

**หลักการ:** รักษา Request Rate (RPS) คงที่ โดยไม่สนใจว่าต้องใช้กี่ VUs

```javascript
export const options = {
    scenarios: {
        constant_rps: {
            executor: 'constant-arrival-rate',
            rate: 1000,               // 1000 iterations ต่อ timeUnit
            timeUnit: '1s',           // ต่อวินาที = 1000 RPS
            duration: '5m',
            preAllocatedVUs: 100,     // VUs ที่เตรียมไว้
            maxVUs: 500,              // VUs สูงสุดที่อนุญาต
        },
    },
};
```

**การทำงาน:**
```
     RPS
      │
 1000 │ ─────────────────────────
      │
      └──────────────────────────► เวลา
        0                    5m

K6 จะ spawn VUs ตามความจำเป็นเพื่อรักษา 1000 RPS
- ถ้า server เร็ว → ใช้ VUs น้อย
- ถ้า server ช้า → ใช้ VUs มาก (สูงสุด maxVUs)
```

### ความแตกต่างระหว่าง constant-vus vs constant-arrival-rate

| | constant-vus | constant-arrival-rate |
|---|-------------|----------------------|
| **ควบคุม** | จำนวน VUs | จำนวน RPS |
| **RPS** | แปรผันตาม response time | คงที่ตามที่กำหนด |
| **VUs** | คงที่ | แปรผันตาม response time |
| **เมื่อ server ช้า** | RPS ลดลง | เพิ่ม VUs |

**ตัวอย่างเปรียบเทียบ:**
```
constant-vus: 100 VUs, response time = 100ms
→ RPS = 100 × (1000/100) = 1000 RPS

constant-vus: 100 VUs, response time = 500ms (server ช้าลง)
→ RPS = 100 × (1000/500) = 200 RPS ← RPS ลดลง!

constant-arrival-rate: 1000 RPS, response time = 500ms
→ VUs needed = 1000 × (500/1000) = 500 VUs ← เพิ่ม VUs แทน
```

**เหมาะกับ:**
- ทดสอบว่า server รับ X RPS ได้ไหม (เป้าหมายของโปรเจคนี้: 10,000 RPS)
- Benchmark ที่ต้องการ rate คงที่

---

## 6. ramping-arrival-rate (แรมปิ้ง-อะไรวัล-เรท) ⭐⭐

**หลักการ:** ค่อยๆ เพิ่ม/ลด RPS ตาม stages

```javascript
export const options = {
    scenarios: {
        spike_test: {
            executor: 'ramping-arrival-rate',
            startRate: 100,           // เริ่มที่ 100 RPS
            timeUnit: '1s',
            stages: [
                { duration: '1m', target: 100 },     // คงที่ 100 RPS
                { duration: '30s', target: 10000 },  // 100 → 10000 RPS ใน 30 วินาที
                { duration: '2m', target: 10000 },   // คงที่ 10000 RPS
                { duration: '30s', target: 100 },    // 10000 → 100 RPS
                { duration: '1m', target: 100 },     // คงที่ 100 RPS
            ],
            preAllocatedVUs: 1000,
            maxVUs: 5000,
        },
    },
};
```

**การทำงาน:**
```
     RPS
      │
10000 │              ┌──────────┐
      │             /            \
      │            /              \
  100 │───────────┘                └─────────
      └──────────────────────────────────────► เวลา
         1m   1.5m          3.5m  4m    5m
```

**เหมาะกับ:**
- Spike test (จำลอง flash sale, viral content)
- ทดสอบ auto-scaling ของ server

---

## รันหลาย Scenarios พร้อมกัน

```javascript
export const options = {
    scenarios: {
        // Scenario 1: API Browse (read-heavy)
        browse: {
            executor: 'constant-arrival-rate',
            rate: 5000,
            timeUnit: '1s',
            duration: '5m',
            preAllocatedVUs: 500,
            maxVUs: 1000,
            exec: 'browseEvents',        // รัน function browseEvents
            tags: { scenario: 'browse' },
        },

        // Scenario 2: API Reserve (write-heavy)
        reserve: {
            executor: 'constant-arrival-rate',
            rate: 1000,
            timeUnit: '1s',
            duration: '5m',
            preAllocatedVUs: 200,
            maxVUs: 500,
            exec: 'reserveSeats',        // รัน function reserveSeats
            startTime: '30s',            // เริ่มหลังจาก 30 วินาที
            tags: { scenario: 'reserve' },
        },
    },
};

// Function สำหรับ browse scenario
export function browseEvents() {
    http.get(`${BASE_URL}/events`);
}

// Function สำหรับ reserve scenario
export function reserveSeats() {
    http.post(`${BASE_URL}/bookings/reserve`, payload);
}

// Default function (ไม่ใช้ถ้ากำหนด exec ใน scenario)
export default function() {
    // ...
}
```

---

## ตัวอย่างจากโปรเจค Booking Rush

```javascript
const allScenarios = {
    // 1. Smoke Test - ทดสอบว่าระบบทำงาน
    smoke: {
        executor: 'constant-vus',
        vus: 1,
        duration: '30s',
        exec: 'reserveSeats',
    },

    // 2. Ramp Up - ค่อยๆ เพิ่ม VUs
    ramp_up: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 100 },
            { duration: '2m', target: 500 },
            { duration: '3m', target: 1000 },
            { duration: '2m', target: 500 },
            { duration: '1m', target: 0 },
        ],
        exec: 'reserveSeats',
    },

    // 3. Sustained Load - รักษา 5000 RPS
    sustained: {
        executor: 'constant-arrival-rate',
        rate: 5000,
        timeUnit: '1s',
        duration: '5m',
        preAllocatedVUs: 1000,
        maxVUs: 2000,
        exec: 'reserveSeats',
    },

    // 4. Spike Test - traffic พุ่ง
    spike: {
        executor: 'ramping-arrival-rate',
        startRate: 1000,
        timeUnit: '1s',
        stages: [
            { duration: '30s', target: 1000 },
            { duration: '10s', target: 10000 },  // spike!
            { duration: '1m', target: 10000 },
            { duration: '10s', target: 1000 },
            { duration: '1m', target: 1000 },
        ],
        preAllocatedVUs: 2000,
        maxVUs: 5000,
        exec: 'reserveSeats',
    },

    // 5. Stress 10K RPS - เป้าหมายหลัก
    stress_10k: {
        executor: 'constant-arrival-rate',
        rate: 10000,
        timeUnit: '1s',
        duration: '5m',
        preAllocatedVUs: 2000,
        maxVUs: 5000,
        exec: 'reserveSeats',
    },
};
```

### รัน Scenario เฉพาะ
```bash
# Smoke test
SCENARIO=smoke k6 run 01-booking-reserve.js

# Spike test
SCENARIO=spike k6 run 01-booking-reserve.js

# Stress 10K RPS
SCENARIO=stress_10k k6 run 01-booking-reserve.js

# รันทุก scenarios
SCENARIO=all k6 run 01-booking-reserve.js
```

---

## สรุป: เลือก Executor อย่างไร?

| ต้องการทดสอบ | Executor ที่เหมาะ |
|--------------|-----------------|
| ทำ N ครั้งแล้วจบ | `shared-iterations` |
| แต่ละ user ทำ N ครั้ง | `per-vu-iterations` |
| X users พร้อมกัน | `constant-vus` |
| ค่อยๆ เพิ่ม/ลด users | `ramping-vus` |
| รักษา X RPS คงที่ | `constant-arrival-rate` |
| Spike/Flash sale | `ramping-arrival-rate` |

---

## อ่านต่อ

- [04 - Metrics และ Thresholds](./04-metrics.md)
- [05 - อ่านกราฟ Dashboard](./05-dashboard.md)
