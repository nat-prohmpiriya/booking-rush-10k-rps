Race Condition ใน PostgreSQL

  Race Condition คือ สถานการณ์ที่ 2+ transactions พยายามอ่าน/เขียนข้อมูลเดียวกันพร้อมกัน ทำให้ผลลัพธ์ผิดพลาด

  ตัวอย่างง่ายๆ: ขายตั๋ว

  เหลือที่นั่ง = 1

  Transaction A                    Transaction B
  ─────────────────────────────────────────────────────
  SELECT available FROM seats
  → เห็น available = 1
                                   SELECT available FROM seats
                                   → เห็น available = 1

  UPDATE seats SET available = 0
  → สำเร็จ! ขายได้
                                   UPDATE seats SET available = 0
                                   → สำเร็จ! ขายได้ (แต่ไม่มีที่แล้ว!)

  ❌ ผลลัพธ์: ขายไป 2 ที่ แต่มีแค่ 1 = OVERSELLING!

  ---
  🔐 วิธีป้องกัน Race Condition

  1. Pessimistic Locking (SELECT ... FOR UPDATE)

  -- Lock row ไว้เลย ใครมาทีหลังต้องรอ
  BEGIN;
  SELECT available FROM seats WHERE zone_id = 'A' FOR UPDATE;
  -- ตอนนี้ row ถูก lock, Transaction อื่นต้องรอ
  UPDATE seats SET available = available - 1 WHERE zone_id = 'A';
  COMMIT;

  2. Optimistic Locking (Version/Timestamp)

  -- ใช้ version number ตรวจสอบ
  UPDATE seats
  SET available = available - 1, version = version + 1
  WHERE zone_id = 'A' AND version = 5;  -- ต้องตรงกับที่อ่านมา

  -- ถ้า rows_affected = 0 แสดงว่ามีคนอื่นแก้ไปแล้ว → retry

  3. Atomic Update (ดีที่สุดสำหรับ counter)

  -- อัพเดทพร้อมเช็คใน command เดียว
  UPDATE seats
  SET available = available - 1
  WHERE zone_id = 'A' AND available >= 1
  RETURNING available;

  -- ถ้า available < 1 จะไม่ update → ไม่ oversell

  4. SERIALIZABLE Isolation

  BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
  -- PostgreSQL จะ detect conflict อัตโนมัติ
  -- ถ้าชน จะ error → ต้อง retry
  COMMIT;

  ---
  📊 เปรียบเทียบ

  | วิธี            | Pros       | Cons                |
  |---------------|------------|---------------------|
  | FOR UPDATE    | ป้องกัน 100% | Lock รอกัน, ช้า        |
  | Optimistic    | ไม่ lock    | ต้อง retry เยอะตอนชุก |
  | Atomic Update | เร็ว, ง่าย    | ใช้ได้แค่ simple cases |
  | SERIALIZABLE  | ปลอดภัยสุด   | Performance แย่      |

  ---
  🎯 สำหรับ Booking System

  แนะนำ: ใช้ Redis Lua (ที่มีอยู่แล้ว) + PostgreSQL Atomic Update

  1. Redis Lua     →  Reserve ที่นั่ง (เร็ว, atomic)
  2. PostgreSQL    →  บันทึก booking record
  3. ถ้า PG fail   →  Redis release คืน

  เหตุผลที่ไม่ใช้ PostgreSQL อย่างเดียว:
  - FOR UPDATE lock รอกัน → throughput ต่ำ
  - 10K RPS ใช้ PostgreSQL lock ไม่ไหว