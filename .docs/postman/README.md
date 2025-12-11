# Booking Rush Postman Collection

Postman collection สำหรับ Booking Rush 10K RPS API

## Files

| File | Description |
|------|-------------|
| `booking-rush.postman_collection.json` | Main collection รวมทุก endpoints |
| `booking-rush.postman_environment.json` | Environment สำหรับ local development |
| `booking-rush-remote.postman_environment.json` | Environment สำหรับ remote server (100.104.0.42) |

## How to Import

1. เปิด Postman
2. Click **Import** button
3. Drag & drop files หรือเลือก files จาก folder นี้
4. Import ทั้ง collection และ environment ที่ต้องการใช้

## Services & Ports

| Service | Port | Description |
|---------|------|-------------|
| API Gateway | 8080 | Rate limiting, routing, JWT validation |
| Auth Service | 8081 | Authentication, JWT, Tenants |
| Ticket Service | 8082 | Events, Shows, Zones |
| Booking Service | 8083 | Seat reservation (Redis Lua) |
| Payment Service | 8084 | Payment processing |

## Collection Structure

```
Booking Rush 10K RPS
├── Health Checks/          # Health & readiness endpoints
├── Auth Service/
│   ├── Authentication/     # Register, Login, Refresh, Logout
│   └── Tenants/            # CRUD tenants (Admin only)
├── Ticket Service/
│   ├── Events/             # CRUD events
│   ├── Shows/              # CRUD shows
│   └── Zones/              # CRUD zones
├── Booking Service/        # Reserve, Confirm, Cancel bookings
├── Payment Service/        # Create, Process, Refund payments
└── Workflows/
    └── Complete Booking Flow/  # End-to-end booking flow
```

## Quick Start

### 1. เลือก Environment
- **Local**: ใช้ `Booking Rush - Local` environment
- **Remote**: ใช้ `Booking Rush - Remote` environment

### 2. Login เพื่อรับ Token
1. ไปที่ **Auth Service > Authentication > Login**
2. แก้ไข email/password ตามต้องการ
3. Send request
4. Token จะถูก save ไว้ใน collection variables อัตโนมัติ

### 3. ใช้งาน Authenticated Endpoints
- ทุก request ที่ต้องการ authentication จะใช้ `{{access_token}}` อัตโนมัติ
- ถ้า token หมดอายุ ใช้ **Refresh Token** endpoint

## Variables

Collection variables ที่ใช้:

| Variable | Description |
|----------|-------------|
| `base_url` | Base URL (http://localhost หรือ http://100.104.0.42) |
| `gateway_url` | API Gateway URL |
| `auth_url` | Auth Service URL |
| `ticket_url` | Ticket Service URL |
| `booking_url` | Booking Service URL |
| `payment_url` | Payment Service URL |
| `access_token` | JWT access token (auto-saved after login) |
| `refresh_token` | JWT refresh token (auto-saved after login) |
| `user_id` | Current user ID |
| `tenant_id` | Current tenant ID |
| `event_id` | Selected event ID |
| `show_id` | Selected show ID |
| `zone_id` | Selected zone ID |
| `booking_id` | Current booking ID |
| `payment_id` | Current payment ID |

## Test Scripts

หลาย requests มี test scripts ที่:
- ดึง ID จาก response แล้ว save เป็น variable
- ช่วยให้ใช้งานต่อเนื่องได้ง่าย (เช่น Login แล้วใช้ token ต่อได้เลย)

## Workflows

### Complete Booking Flow
ลำดับ requests สำหรับ booking flow:
1. **Login** - รับ token
2. **Browse Events** - ดูรายการ events
3. **Get Event Shows** - ดูรอบการแสดง
4. **Get Show Zones** - ดู zones และราคา
5. **Reserve Seats** - จองที่นั่ง
6. **Confirm Booking** - ยืนยันการจอง
7. **Check Booking Status** - ตรวจสอบสถานะ

## Notes

- **Idempotency**: Booking และ Payment services รองรับ idempotent operations
- **TTL**: การจองมี TTL 10-15 นาที ต้อง confirm ก่อนหมดเวลา
- **Redis Lua**: Booking ใช้ Redis Lua scripts สำหรับ atomic seat deduction
