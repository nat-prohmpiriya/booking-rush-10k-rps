# ออกเสียงแบบนี้ครับ:

  | Pattern       | ออกเสียง         | ความหมาย
                   |
  |---------------|------------------|--------------------------
  ------------------|
  | Orchestration | ออ-เคส-เทร-ชั่น  | การควบคุมแบบวงออเคสตรา
  (มีวาทยกรคุม)       |
  | Choreography  | คอ-รี-ออก-กระ-ฟี | การเต้นรำแบบออกแบบท่า
  (เต้นเองตาม pattern) |

  เปรียบเทียบง่ายๆ

  Orchestration 🎻 = วงออเคสตรา
  - มี conductor (วาทยกร) คอยสั่งว่าใครเล่นเมื่อไหร่
  - ทุกคนทำตาม conductor

  Choreography 💃 = การเต้นรำ
  - ไม่มีคนสั่ง แต่ทุกคน รู้ท่าเต้นของตัวเอง
  - พอเห็น signal ก็ทำต่อเอง

  ---
  ในโปรเจคเรา Orchestrator คือ "วาทยกร" ที่สั่ง:
  1. "Booking Service - จองที่นั่ง!"
  2. "Payment Service - ตัดเงิน!"
  3. "Notification - ส่งอีเมล!"
# ============================================================
# Saga Pattern มี 2 แบบหลักๆ:

  Saga Pattern Types

  1. Orchestration (ที่เราใช้) 🎯

                      ┌─────────────────┐
                      │  Orchestrator   │  ← Central coordinator
                      └────────┬────────┘
             ┌─────────────────┼─────────────────┐
             ▼                 ▼                 ▼
        ┌─────────┐      ┌─────────┐      ┌─────────┐
        │ Step 1  │      │ Step 2  │      │ Step 3  │
        │ Reserve │ ───► │ Payment │ ───► │ Confirm │
        └─────────┘      └─────────┘      └─────────┘
  - มี central controller คอยสั่งการทุก step
  - Orchestrator รู้ flow ทั้งหมด
  - ง่ายต่อการ debug และ monitor
  - เหมาะกับ: Complex flows, ต้องการ visibility สูง

  2. Choreography (อีกแบบ)

        ┌─────────┐      ┌─────────┐      ┌─────────┐
        │ Step 1  │ ───► │ Step 2  │ ───► │ Step 3  │
        │ Reserve │      │ Payment │      │ Confirm │
        └────┬────┘      └────┬────┘      └────┬────┘
             │                │                │
             └────────────────┴────────────────┘
                      Event Bus (Kafka)
  - ไม่มี central controller - แต่ละ service ฟัง events แล้วทำต่อเอง
  - Decoupled มากกว่า
  - ยากต่อการ debug (distributed)
  - เหมาะกับ: Simple flows, ต้องการ loose coupling

  ---
  ทำไมเราเลือก Orchestration?

  สำหรับ Booking Rush (10k RPS) เลือก Orchestration เพราะ:

  1. Booking flow ซับซ้อน - Reserve → Pay → Confirm ต้องเป็น
  sequence
  2. ต้องการ compensation ที่ชัดเจน - ถ้า payment fail ต้อง release
  seats ทันที
  3. Visibility สำคัญ - ต้องรู้ว่า booking อยู่ state ไหน
  4. Easier to debug - เมื่อมีปัญหาสามารถ trace ได้ง่าย

  ใน pkg/saga/ ที่สร้างไป:
  - orchestrator.go = Central coordinator
  - state.go = State machine (CREATED → RESERVED → PAID →
  CONFIRMED)
# ============================================================
# STEP TEST
## setup stripe cli
### Login
stripe login
### Forward webhooks to localhost
stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe
##  strip credit card
⏺ Stripe Test Cards สำหรับทดสอบครับ:

  บัตรที่ผ่าน:
  | Card Number         | Description           |
  |---------------------|-----------------------|
  | 4242 4242 4242 4242 | Visa - สำเร็จเสมอ       |
  | 5555 5555 5555 4444 | Mastercard - สำเร็จเสมอ |

  บัตรที่ถูกปฏิเสธ:
  | Card Number         | Error              |
  |---------------------|--------------------|
  | 4000 0000 0000 0002 | Card declined      |
  | 4000 0000 0000 9995 | Insufficient funds |
  | 4000 0000 0000 9987 | Lost card          |
  | 4000 0000 0000 0069 | Expired card       |

  ต้องการ 3D Secure:
  | Card Number         | Description            |
  |---------------------|------------------------|
  | 4000 0025 0000 3155 | 3DS required - จะผ่าน    |
  | 4000 0000 0000 3220 | 3DS required - จะ fail |

  ใช้กับทุก test card:
  - Expiration: อนาคตใดก็ได้ เช่น 12/26
  - CVC: เลข 3 หลักใดก็ได้ เช่น 123
  - ZIP: เลขใดก็ได้ เช่น 10110