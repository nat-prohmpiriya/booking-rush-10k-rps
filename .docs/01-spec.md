# Product Specification: Booking Rush (10k RPS)

> **Version:** 3.0
> **Last Updated:** 2025-12-07
> **Status:** Draft - Pending Review

---

## Table of Contents
1. [Project Overview](#1-project-overview)
2. [User Personas](#2-user-personas)
3. [User Journeys](#3-user-journeys)
4. [Core Features](#4-core-features)
5. [Business Rules](#5-business-rules)
6. [Success Metrics](#6-success-metrics)
7. [Edge Cases & Error Scenarios](#7-edge-cases--error-scenarios)

---

## 1. Project Overview

### 1.1 Problem Statement
ระบบจองตั๋วคอนเสิร์ตในช่วง Flash Sale มักเผชิญปัญหา:
- **Race Condition:** หลายพันคนกดจองที่นั่งเดียวกันพร้อมกัน
- **Overselling:** ขายตั๋วเกินจำนวนที่มี
- **System Crash:** ระบบล่มเมื่อ traffic พุ่งสูง
- **Poor UX:** User ติด queue นาน หรือจ่ายเงินแล้วไม่ได้ตั๋ว

### 1.2 Solution
**Booking Rush** - High-Concurrency Ticket Booking System ที่ออกแบบมาเพื่อ:
- รองรับ **10,000 Requests Per Second (RPS)**
- **Zero Overselling** ด้วย Atomic Operations
- **Eventual Consistency** - ถ้า Payment ล่ม การจองไม่หาย
- **Horizontal Scalability** - Scale เฉพาะ service ที่โหลดหนัก

### 1.3 Project Scope
| In Scope | Out of Scope |
|----------|--------------|
| User registration & authentication | Mobile native apps |
| Event & show management | Seat-level selection (จองเป็น zone/section) |
| High-concurrency booking | Dynamic pricing (Phase 2) |
| Payment processing (mock) | Real payment gateway integration |
| Virtual queue / waiting room | Multi-language support |
| Email notifications | SMS/Push notifications (Phase 2) |
| Admin dashboard basics | Advanced analytics & reporting |

---

## 2. User Personas

### 2.1 End User (Ticket Buyer)
| Attribute | Description |
|-----------|-------------|
| **Who** | คนทั่วไปที่ต้องการซื้อตั๋วคอนเสิร์ต/อีเวนต์ |
| **Age** | 18-45 ปี |
| **Tech Savvy** | ปานกลาง - สูง |
| **Behavior** | - เข้ามารอก่อนเวลาเปิดขาย<br>- เปิดหลาย tab/device พร้อมกัน<br>- ต้องการความเร็ว ไม่อยากรอนาน<br>- กังวลว่าจะพลาดตั๋ว |
| **Pain Points** | - ระบบล่มตอน Flash Sale<br>- กดจองได้แต่จ่ายเงินไม่ทัน<br>- ไม่รู้ว่าต้องรออีกนานแค่ไหน<br>- จ่ายเงินแล้วไม่ได้ตั๋ว |
| **Goals** | - จองตั๋วได้สำเร็จ<br>- รู้สถานะตลอดเวลา<br>- ได้รับการยืนยันทันที |

### 2.2 Event Organizer
| Attribute | Description |
|-----------|-------------|
| **Who** | บริษัทจัดคอนเสิร์ต, ผู้จัดงาน |
| **Behavior** | - สร้างและจัดการ Event<br>- ตั้งราคาและจำนวนที่นั่ง<br>- ติดตามยอดขาย real-time |
| **Pain Points** | - ขายตั๋วเกินจำนวน (overselling)<br>- ไม่มีข้อมูล real-time<br>- จัดการ refund ยุ่งยาก |
| **Goals** | - ขายตั๋วได้ครบตามจำนวน<br>- ไม่มี overselling<br>- เห็น dashboard ยอดขาย |

### 2.3 System Admin
| Attribute | Description |
|-----------|-------------|
| **Who** | ทีม IT ที่ดูแลระบบ |
| **Behavior** | - Monitor ระบบ<br>- จัดการ user/tenant<br>- แก้ไขปัญหาเร่งด่วน |
| **Pain Points** | - ไม่รู้ว่าระบบมีปัญหาตรงไหน<br>- Debug ยาก<br>- Scale ไม่ทัน |
| **Goals** | - เห็น health ของระบบทั้งหมด<br>- Alert เมื่อมีปัญหา<br>- Scale ได้อัตโนมัติ |

---

## 3. User Journeys

### 3.1 Journey: ซื้อตั๋วคอนเสิร์ต (Happy Path)

**Scenario:** นาย A ต้องการซื้อตั๋วคอนเสิร์ต BTS ที่เปิดขาย Flash Sale วันที่ 1 ม.ค. เวลา 10:00 น.

```
Timeline: ก่อนเปิดขาย → เปิดขาย → จอง → จ่ายเงิน → รับตั๋ว

[09:50] นาย A เข้าเว็บไซต์ เห็นหน้า Event Detail
        → ปุ่ม "จองตั๋ว" ยัง disabled พร้อม countdown "เปิดจองในอีก 10:00 นาที"

[09:59] Countdown ใกล้หมด นาย A refresh หน้า
        → เห็น "เตรียมตัวให้พร้อม กดจองได้ในอีก 60 วินาที"

[10:00] Flash Sale เปิด! นาย A กดปุ่ม "จองตั๋ว"
        → ระบบตรวจสอบว่า traffic สูง → redirect ไป Virtual Queue
        → เห็น "คุณอยู่ในคิวลำดับที่ 1,234 รอประมาณ 3 นาที"

[10:03] ถึงคิวแล้ว! ได้รับ Queue Pass Token
        → Redirect ไปหน้าเลือก Zone/Section
        → เห็นจำนวนที่นั่งว่างแต่ละ Zone (อัพเดท real-time)

[10:04] นาย A เลือก Zone A (VIP) จำนวน 2 ใบ กด "Reserve"
        → ระบบ lock ที่นั่ง 10 นาที
        → เห็น countdown "จ่ายเงินภายใน 10:00 นาที"
        → เห็น Order Summary: Zone A x 2 = 6,000 บาท

[10:05] นาย A กรอกข้อมูลการจ่ายเงิน กด "ชำระเงิน"
        → แสดง loading "กำลังดำเนินการ..."
        → Payment สำเร็จ!

[10:06] เห็นหน้า Confirmation
        → "จองสำเร็จ! หมายเลขการจอง: BK-20250101-001234"
        → ได้รับ email ยืนยันพร้อม E-Ticket (QR Code)
```

### 3.2 Journey: เน็ตหลุดระหว่างจ่ายเงิน (Recovery Path)

**Scenario:** นาย B จองตั๋วได้แล้ว แต่เน็ตหลุดตอนจ่ายเงิน

```
[10:04] นาย B เลือก Zone B จำนวน 2 ใบ กด "Reserve" สำเร็จ
        → ที่นั่งถูก lock 10 นาที

[10:05] นาย B กด "ชำระเงิน" แต่เน็ตหลุด!
        → หน้าค้าง, connection timeout

[10:07] นาย B กลับมาออนไลน์ เข้าเว็บใหม่
        → Login เข้าระบบ
        → เห็น banner "คุณมีการจองที่รอชำระเงิน (เหลือเวลา 7:00 นาที)"
        → กดที่ banner → ไปหน้า Payment ต่อ

[10:08] นาย B จ่ายเงินสำเร็จ
        → ได้รับ E-Ticket
```

### 3.3 Journey: จ่ายเงินไม่ทัน (Timeout Path)

**Scenario:** นาย C จองได้แต่ไม่จ่ายเงินภายใน 10 นาที

```
[10:04] นาย C เลือก Zone C จำนวน 1 ใบ กด "Reserve" สำเร็จ

[10:05] นาย C ต้องไปทำธุระด่วน ไม่ได้จ่ายเงิน

[10:14] หมดเวลา 10 นาที
        → ระบบ auto-release ที่นั่งกลับคืน
        → Booking status เปลี่ยนเป็น "Expired"
        → ส่ง email แจ้ง "การจองของคุณหมดอายุแล้ว"

[10:30] นาย C กลับมา เห็นหน้า Booking History
        → "การจอง BK-xxx หมดอายุแล้ว เนื่องจากไม่ได้ชำระเงินภายในเวลาที่กำหนด"
        → ปุ่ม "จองใหม่" (ถ้ายังมีที่นั่งว่าง)
```

### 3.4 Journey: ตั๋วหมด (Sold Out Path)

**Scenario:** นางสาว D เข้ามาช้า ตั๋วหมดแล้ว

```
[10:30] นางสาว D เข้าเว็บไซต์
        → เห็นหน้า Event Detail
        → ป้าย "SOLD OUT" สีแดงที่ปุ่มจอง
        → ปุ่มเปลี่ยนเป็น "แจ้งเตือนเมื่อมีตั๋วว่าง"

[10:31] นางสาว D กดปุ่มแจ้งเตือน
        → กรอก email → "ระบบจะแจ้งเตือนเมื่อมีตั๋วว่าง (จาก cancellation)"

[11:00] มีคนยกเลิกการจอง ตั๋วว่าง 2 ใบ
        → นางสาว D ได้รับ email "มีตั๋ว Zone B ว่างแล้ว! รีบจองเลย"
```

### 3.5 Journey: Event Organizer สร้าง Event

**Scenario:** ผู้จัดงานสร้าง Event คอนเสิร์ตใหม่

```
[Day 1] ผู้จัดงาน Login เข้า Admin Dashboard
        → กด "Create New Event"
        → กรอกข้อมูล: ชื่อ, รายละเอียด, รูปภาพ, Category

[Day 1] ตั้งค่า Shows (รอบการแสดง)
        → เพิ่ม Show: 1 ม.ค. 2025 เวลา 19:00
        → เพิ่ม Show: 2 ม.ค. 2025 เวลา 19:00

[Day 1] ตั้งค่า Zones และราคา
        → Zone A (VIP): 3,000 บาท x 500 ที่นั่ง
        → Zone B (Regular): 1,500 บาท x 1,000 ที่นั่ง
        → Zone C (Economy): 800 บาท x 2,000 ที่นั่ง

[Day 1] ตั้งค่า Sales Settings
        → เปิดขาย: 1 ม.ค. 2025 เวลา 10:00
        → จำกัด: 4 ใบต่อคนต่อ Event
        → เวลาจ่ายเงิน: 10 นาที

[Day 1] กด "Publish" → Event เปลี่ยนสถานะเป็น Published
        → เห็นบนหน้าเว็บสาธารณะแล้ว (แต่ยังจองไม่ได้)

[Sale Day] ดู Real-time Dashboard
        → เห็นกราฟยอดขายทะลุขึ้น
        → เห็นจำนวนคนในคิว
        → เห็นจำนวนที่นั่งคงเหลือแต่ละ Zone
```

---

## 4. Core Features

### 4.1 User Management
| Feature | Description | Priority |
|---------|-------------|----------|
| Register | สมัครสมาชิกด้วย Email + Password | Must Have |
| Login | เข้าสู่ระบบด้วย JWT | Must Have |
| Profile | ดูและแก้ไขข้อมูลส่วนตัว | Must Have |
| Booking History | ดูประวัติการจองทั้งหมด | Must Have |
| Password Reset | รีเซ็ตรหัสผ่านผ่าน Email | Must Have |
| OAuth2 (Google) | Login ด้วย Google | Nice to Have |

### 4.2 Event Management (Organizer)
| Feature | Description | Priority |
|---------|-------------|----------|
| Create Event | สร้าง Event ใหม่พร้อมรายละเอียด | Must Have |
| Manage Shows | เพิ่ม/แก้ไข รอบการแสดง | Must Have |
| Manage Zones | ตั้งค่า Zone, ราคา, จำนวนที่นั่ง | Must Have |
| Publish/Unpublish | ควบคุมการแสดงผล Event | Must Have |
| Sales Settings | ตั้งเวลาเปิดขาย, จำกัดจำนวนต่อคน | Must Have |
| Event Analytics | ดูสถิติการขาย | Nice to Have |

### 4.3 Booking Flow
| Feature | Description | Priority |
|---------|-------------|----------|
| Browse Events | ดูรายการ Event ที่เปิดขาย | Must Have |
| Event Detail | ดูรายละเอียด Event และ Shows | Must Have |
| Zone Selection | เลือก Zone และจำนวนตั๋ว | Must Have |
| Reserve Seats | Lock ที่นั่งชั่วคราว (10 นาที) | Must Have |
| Payment | ชำระเงิน (Mock) | Must Have |
| Confirmation | แสดงผลการจองสำเร็จ | Must Have |
| E-Ticket | แสดง QR Code สำหรับเข้างาน | Must Have |
| Resume Payment | กลับมาจ่ายเงินต่อถ้าหลุด | Must Have |
| Cancel Booking | ยกเลิกการจอง (ตามเงื่อนไข) | Must Have |

### 4.4 Virtual Queue (Waiting Room)
| Feature | Description | Priority |
|---------|-------------|----------|
| Auto Queue | Redirect ไป Queue เมื่อ traffic สูง | Must Have |
| Position Display | แสดงลำดับในคิว | Must Have |
| Wait Time | แสดงเวลารอโดยประมาณ | Must Have |
| Queue Pass | Token สำหรับ bypass rate limit | Must Have |
| Auto Redirect | ไปหน้าจองเมื่อถึงคิว | Must Have |

### 4.5 Notifications
| Feature | Description | Priority |
|---------|-------------|----------|
| Booking Confirmation | Email ยืนยันการจอง | Must Have |
| Payment Receipt | Email ใบเสร็จ | Must Have |
| Booking Expired | Email แจ้งการจองหมดอายุ | Must Have |
| Event Reminder | Email เตือนก่อนวัน Event | Nice to Have |
| Waitlist Alert | Email แจ้งเมื่อมีตั๋วว่าง | Nice to Have |

### 4.6 Admin Dashboard
| Feature | Description | Priority |
|---------|-------------|----------|
| User Management | ดู/จัดการ users | Must Have |
| Event Overview | ดูสรุป Events ทั้งหมด | Must Have |
| Real-time Sales | ดูยอดขาย real-time | Must Have |
| System Health | ดูสถานะระบบ | Must Have |

---

## 5. Business Rules

### 5.1 Booking Rules
| Rule | Value | Configurable |
|------|-------|--------------|
| Max tickets per user per event | 4 | Yes (per event) |
| Seat reservation timeout | 10 minutes | Yes (global) |
| Booking window before event | 1 hour | Yes (per event) |
| Minimum age for certain events | 18+ | Yes (per event) |

### 5.2 Pricing Rules
| Rule | Description |
|------|-------------|
| Zone-based Pricing | แต่ละ Zone มีราคาต่างกัน |
| Early Bird | ส่วนลดสำหรับจองล่วงหน้า (Phase 2) |
| Promo Codes | รหัสส่วนลด (Phase 2) |
| Bundle Discount | ซื้อ 3 ใบ ลด 10% (Phase 2) |

### 5.3 Refund Rules
| Condition | Refund % |
|-----------|----------|
| > 7 days before event | 100% |
| 3-7 days before event | 50% |
| < 3 days before event | 0% |
| Event cancelled by organizer | 100% + compensation |

### 5.4 Queue Rules
| Rule | Value |
|------|-------|
| Queue activation threshold | > 1,000 concurrent users |
| Batch release size | 100 users per batch |
| Queue pass validity | 5 minutes |
| Re-queue on pass expiry | Yes |

---

## 6. Success Metrics

### 6.1 Technical Metrics
| Metric | Target | Note |
|--------|--------|------|
| Throughput | ≥ 10,000 RPS | Booking endpoint |
| Latency P50 (Server) | < 20ms | Server processing time only |
| Latency P99 (Server) | < 50ms | Server processing time only |
| Latency P99 (E2E) | < 200ms | Including network latency |
| Error Rate | < 0.1% | Non-5xx errors |
| Availability | 99.9% | 8.76 hrs downtime/year |
| **Overselling Rate** | **0%** | **Zero tolerance** |

### 6.2 Business Metrics
| Metric | Target | Note |
|--------|--------|------|
| Booking Completion Rate | > 80% | Reserved → Paid |
| Cart Abandonment | < 20% | Reserved but not paid |
| Queue Drop-off | < 10% | Left queue before turn |
| Customer Satisfaction | > 4.0/5 | Post-purchase survey |

### 6.3 User Experience Metrics
| Metric | Target | Note |
|--------|--------|------|
| Time to Book | < 3 minutes | From event page to confirmation |
| Queue Wait Accuracy | ±30 seconds | Estimated vs actual wait |
| Page Load Time | < 2 seconds | First contentful paint |
| Mobile Responsiveness | 100% | All features work on mobile |

---

## 7. Edge Cases & Error Scenarios

### 7.1 High Traffic Scenarios
| Scenario | Expected Behavior |
|----------|-------------------|
| 50,000 users hit booking at same time | Virtual queue activates, orderly processing |
| User refreshes page while in queue | Maintains queue position (via session/cookie) |
| User opens multiple tabs | Same queue position across tabs |
| Flash sale starts at midnight | System handles timezone correctly |

### 7.2 Payment Scenarios
| Scenario | Expected Behavior |
|----------|-------------------|
| Payment gateway timeout | Show "กำลังตรวจสอบ" + retry 3 times |
| Payment fails | Show error, allow retry, keep reservation |
| Double payment attempt | Idempotent - charge only once |
| Network disconnection during payment | Resume payment flow when back online |
| Payment succeeds but DB write fails | Saga compensation, refund automatically |

### 7.3 Inventory Scenarios
| Scenario | Expected Behavior |
|----------|-------------------|
| Last seat, 2 users click simultaneously | Only 1 succeeds (atomic operation) |
| User reserves then abandons | Auto-release after 10 min timeout |
| Admin manually adjusts inventory | Real-time update, notify affected users |
| Negative inventory attempt | Prevent, return "Sold Out" |

### 7.4 Session Scenarios
| Scenario | Expected Behavior |
|----------|-------------------|
| Token expires during booking | Refresh token automatically |
| User logs in from new device | Invalidate old sessions (optional) |
| Cookie blocked by browser | Fallback to localStorage/sessionStorage |

### 7.5 System Failure Scenarios
| Scenario | Expected Behavior |
|----------|-------------------|
| Redis down | Return "Service Temporarily Unavailable" |
| Database down | Graceful degradation, cache serving |
| Message queue down | Buffer locally, retry when recovered |
| Single service crash | Other services continue, affected feature unavailable |

### 7.6 User Error Scenarios
| Scenario | Expected Behavior |
|----------|-------------------|
| Invalid email format | Show validation error immediately |
| Password too weak | Show requirements, prevent submit |
| Select more than max allowed | Disable + or show limit message |
| Try to book past event | Show "Event has ended" |
| Try to book before sale starts | Show countdown, disable button |

---

## Appendix

### A. Glossary
| Term | Definition |
|------|------------|
| Flash Sale | ช่วงเวลาที่เปิดขายตั๋ว มี traffic สูงมาก |
| Overselling | ขายตั๋วเกินจำนวนที่มีจริง |
| Race Condition | เมื่อหลาย request แย่งเข้าถึง resource เดียวกัน |
| Virtual Queue | ระบบคิวเพื่อควบคุม traffic |
| Queue Pass | Token ที่ได้รับเมื่อถึงคิว ใช้ bypass rate limit |
| Reservation | การจองชั่วคราวก่อนจ่ายเงิน |
| Idempotency | ทำซ้ำกี่ครั้งก็ได้ผลเหมือนกัน |

### B. References
- [Original Technical Spec](./02-plan.md) - รายละเอียดทางเทคนิค
- [Development Tasks](./03-task.md) - รายการงานที่ต้องทำ
