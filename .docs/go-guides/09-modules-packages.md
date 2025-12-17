# 09 - Modules และ Packages

## สารบัญ

1. [Go Modules](#go-modules)
2. [Packages](#packages-แพ็คเกจ)
3. [Import](#import)
4. [Visibility](#visibility-การมองเห็น)
5. [Project Structure](#project-structure-โครงสร้างโปรเจค)
6. [Internal Packages](#internal-packages)
7. [Dependency Management](#dependency-management)

---

## Go Modules

### TypeScript - npm/package.json

```json
{
    "name": "my-app",
    "version": "1.0.0",
    "dependencies": {
        "express": "^4.18.0",
        "lodash": "^4.17.0"
    },
    "devDependencies": {
        "typescript": "^5.0.0"
    }
}
```

### Go - go.mod

```go
// go.mod
module github.com/username/my-app

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/go-redis/redis/v8 v8.11.5
)

require (
    // indirect dependencies (transitive)
    github.com/cespare/xxhash/v2 v2.1.2 // indirect
)
```

### Commands เปรียบเทียบ

| npm | go mod | คำอธิบาย |
|-----|--------|----------|
| `npm init` | `go mod init <module>` | สร้าง project |
| `npm install` | `go mod tidy` | ติดตั้ง dependencies |
| `npm install pkg` | `go get pkg` | เพิ่ม dependency |
| `npm install pkg@1.0.0` | `go get pkg@v1.0.0` | ติดตั้ง version เฉพาะ |
| `npm uninstall pkg` | ลบจาก import แล้ว `go mod tidy` | ลบ dependency |
| `npm update` | `go get -u ./...` | อัพเดท dependencies |
| `npm run build` | `go build` | Build |
| `npm start` | `go run .` | Run |
| `npm test` | `go test ./...` | Test |

### สร้าง Module ใหม่

```bash
# สร้าง directory
mkdir my-project && cd my-project

# Initialize module
go mod init github.com/username/my-project

# ไฟล์ go.mod จะถูกสร้าง
```

### go.sum

```
// go.sum - checksums ของ dependencies
github.com/gin-gonic/gin v1.9.1 h1:4idEAncQnU5cB7BeOkPtxjfCSye0AAm1R0RVIqJ+Jmg=
github.com/gin-gonic/gin v1.9.1/go.mod h1:hPrL7YrpYKXt5YId3A/Dn+qVz=...
```

- `go.sum` เหมือน `package-lock.json`
- ใช้ verify integrity ของ dependencies
- **Commit ทั้ง go.mod และ go.sum**

---

## Packages (แพ็คเกจ)

### TypeScript - Files และ exports

```typescript
// utils/string.ts
export function capitalize(s: string): string {
    return s.charAt(0).toUpperCase() + s.slice(1)
}

// utils/index.ts
export * from './string'

// main.ts
import { capitalize } from './utils'
```

### Go - Package = Directory

```go
// utils/string.go
package utils

func Capitalize(s string) string {
    return strings.ToUpper(s[:1]) + s[1:]
}

// main.go
package main

import "github.com/username/my-app/utils"

func main() {
    result := utils.Capitalize("hello")
}
```

### Package Rules

1. **ทุกไฟล์ใน directory เดียวกันต้องเป็น package เดียวกัน**

```
utils/
├── string.go   // package utils
├── number.go   // package utils
└── helper.go   // package utils
```

2. **Package name = directory name (convention)**

```
mypackage/
└── file.go     // package mypackage
```

3. **ยกเว้น main package**

```
cmd/server/
└── main.go     // package main (executable)
```

### Multiple Files in Package

```go
// utils/string.go
package utils

func Capitalize(s string) string { ... }

// utils/number.go
package utils

func Max(a, b int) int { ... }

// ทั้งสองไฟล์อยู่ใน package utils
// เข้าถึง function กันได้โดยไม่ต้อง import
```

---

## Import

### TypeScript

```typescript
// Named imports
import { Router, Request, Response } from 'express'

// Default import
import express from 'express'

// Import all
import * as utils from './utils'

// Relative path
import { helper } from '../helpers'
```

### Go

```go
import (
    // Standard library
    "fmt"
    "net/http"
    "encoding/json"

    // Third-party packages
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"

    // Local packages (module path + directory)
    "github.com/username/my-app/internal/handler"
    "github.com/username/my-app/pkg/utils"
)
```

### Import Alias

```go
import (
    // Alias
    "fmt"
    json "encoding/json"          // ใช้ json แทน
    myfmt "github.com/pkg/fmt"    // หลีกเลี่ยง conflict

    // Dot import (import ทุก exported identifier) - ไม่แนะนำ
    . "fmt"  // ใช้ Println() แทน fmt.Println()

    // Blank import (import for side effects only)
    _ "github.com/lib/pq"  // register postgres driver
)

// ใช้งาน
json.Marshal(data)      // ใช้ alias
myfmt.Println("hello")  // ใช้ alias
Println("hello")        // dot import (ไม่แนะนำ)
```

### ความแตกต่างจาก TypeScript

```go
// Go ไม่มี named imports
// import ทั้ง package แล้วใช้ package.Function

import "github.com/gin-gonic/gin"

// ใช้
r := gin.Default()
c.JSON(200, gin.H{})

// ไม่มี:
// import { Default, Context } from "gin"  // ไม่ได้!
```

---

## Visibility (การมองเห็น)

### TypeScript - export keyword

```typescript
// Exported (public)
export function publicFunc() {}
export class User {}
export const PI = 3.14

// Not exported (private to file)
function privateFunc() {}
class InternalClass {}
```

### Go - Capital Letter = Exported

```go
package mypackage

// Exported (public) - ขึ้นต้นตัวใหญ่
func PublicFunc() {}
type User struct {
    Name string   // exported field
    age  int      // unexported field
}
const PI = 3.14
var GlobalVar = "hello"

// Unexported (private to package) - ขึ้นต้นตัวเล็ก
func privateFunc() {}
type internalStruct struct {}
const maxRetries = 3
var configCache = make(map[string]string)
```

### ตัวอย่าง Visibility

```go
// models/user.go
package models

type User struct {
    ID       string  // exported - เข้าถึงจากนอก package ได้
    Name     string  // exported
    password string  // unexported - เข้าถึงได้เฉพาะใน package models
}

func NewUser(name string) *User {
    return &User{
        Name:     name,
        password: generatePassword(),  // OK - อยู่ใน package เดียวกัน
    }
}

func (u *User) SetPassword(pwd string) {
    u.password = pwd  // OK
}

func (u *User) CheckPassword(pwd string) bool {
    return u.password == pwd  // OK
}

// main.go
package main

import "myapp/models"

func main() {
    user := models.NewUser("John")
    user.Name = "Jane"            // OK - exported
    user.password = "secret"      // Error! unexported
    user.SetPassword("secret")    // OK - ผ่าน method
}
```

---

## Project Structure (โครงสร้างโปรเจค)

### TypeScript - Common Structure

```
my-app/
├── src/
│   ├── controllers/
│   ├── services/
│   ├── models/
│   ├── routes/
│   └── index.ts
├── tests/
├── package.json
└── tsconfig.json
```

### Go - Standard Layout

```
my-app/
├── cmd/                    # Entry points (main packages)
│   ├── server/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── internal/               # Private packages (ไม่ให้ import จากนอก module)
│   ├── handler/
│   ├── service/
│   ├── repository/
│   ├── domain/
│   └── dto/
├── pkg/                    # Public packages (ให้ import จากนอกได้)
│   ├── logger/
│   ├── config/
│   └── response/
├── api/                    # API specs (OpenAPI, protobuf)
├── scripts/                # Scripts
├── configs/                # Config files
├── go.mod
└── go.sum
```

### Booking Rush Structure

```
booking-rush-10k-rps/
├── backend-auth/
│   ├── cmd/
│   │   └── main.go
│   └── internal/
│       ├── handler/
│       ├── service/
│       ├── repository/
│       ├── domain/
│       └── dto/
├── backend-booking/
│   ├── cmd/
│   │   └── main.go
│   └── internal/
│       └── ...
├── pkg/                    # Shared packages
│   ├── config/
│   ├── logger/
│   ├── database/
│   ├── redis/
│   └── response/
├── scripts/
│   └── lua/               # Redis Lua scripts
├── go.work                # Go workspace
├── go.work.sum
└── docker-compose.yml
```

### Go Workspace (go.work)

```go
// go.work - สำหรับ multi-module development
go 1.21

use (
    ./backend-auth
    ./backend-booking
    ./backend-ticket
    ./backend-payment
    ./pkg
)
```

---

## Internal Packages

### `internal/` = Private to Module

```
my-app/
├── internal/           # ไม่ให้ import จากนอก module
│   └── secret/
│       └── secret.go
├── pkg/                # ให้ import จากนอกได้
│   └── utils/
│       └── utils.go
└── go.mod
```

```go
// ภายนอก module (module อื่น)
import "github.com/username/my-app/internal/secret"  // Error!
import "github.com/username/my-app/pkg/utils"        // OK

// ภายใน module เดียวกัน
import "github.com/username/my-app/internal/secret"  // OK
```

### Internal ที่ Sub-directory

```
my-app/
├── pkg/
│   └── database/
│       ├── internal/       # private to database package
│       │   └── pool.go
│       └── database.go
```

```go
// pkg/database/database.go
import "github.com/username/my-app/pkg/database/internal/pool"  // OK

// pkg/other/other.go
import "github.com/username/my-app/pkg/database/internal/pool"  // Error!
```

---

## Dependency Management

### เพิ่ม Dependency

```bash
# เพิ่ม package
go get github.com/gin-gonic/gin

# เพิ่ม version เฉพาะ
go get github.com/gin-gonic/gin@v1.9.0
go get github.com/gin-gonic/gin@latest

# เพิ่มจาก branch
go get github.com/user/repo@branch-name

# เพิ่มจาก commit
go get github.com/user/repo@abc1234
```

### อัพเดท Dependencies

```bash
# อัพเดททุก packages
go get -u ./...

# อัพเดท package เฉพาะ
go get -u github.com/gin-gonic/gin

# อัพเดทเฉพาะ patch versions
go get -u=patch ./...
```

### ลบ Dependencies ที่ไม่ใช้

```bash
# ลบ unused dependencies + เพิ่ม missing
go mod tidy
```

### ดู Dependencies

```bash
# แสดง dependencies ทั้งหมด
go list -m all

# แสดง dependency graph
go mod graph

# ดูว่าทำไมต้อง package นี้
go mod why github.com/some/package
```

### Vendor Dependencies

```bash
# สร้าง vendor folder (copy dependencies ลง project)
go mod vendor

# Build ด้วย vendor
go build -mod=vendor ./...
```

---

## ตัวอย่างจาก Booking Rush

```go
// backend-booking/go.mod
module github.com/booking-rush/backend-booking

go 1.21

require (
    github.com/booking-rush/pkg v0.0.0
    github.com/gin-gonic/gin v1.9.1
    github.com/go-redis/redis/v8 v8.11.5
    github.com/google/uuid v1.3.0
    github.com/jmoiron/sqlx v1.3.5
    github.com/lib/pq v1.10.9
)

// pkg/go.mod
module github.com/booking-rush/pkg

go 1.21

require (
    github.com/spf13/viper v1.16.0
    go.uber.org/zap v1.25.0
)

// go.work
go 1.21

use (
    ./backend-auth
    ./backend-booking
    ./backend-ticket
    ./backend-payment
    ./backend-api-gateway
    ./pkg
)
```

```go
// backend-booking/internal/handler/booking_handler.go
package handler

import (
    // Standard library
    "net/http"

    // Third-party
    "github.com/gin-gonic/gin"

    // Local - internal packages
    "github.com/booking-rush/backend-booking/internal/dto"
    "github.com/booking-rush/backend-booking/internal/service"

    // Local - shared packages
    "github.com/booking-rush/pkg/response"
)

type BookingHandler struct {
    service *service.BookingService
}

func NewBookingHandler(svc *service.BookingService) *BookingHandler {
    return &BookingHandler{service: svc}
}

func (h *BookingHandler) Reserve(c *gin.Context) {
    var req dto.ReserveRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
        return
    }

    resp, err := h.service.Reserve(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "RESERVE_FAILED", err.Error())
        return
    }

    response.Success(c, resp)
}
```

---

## สรุป

| หัวข้อ | npm (TypeScript) | Go Modules |
|--------|------------------|------------|
| Config file | `package.json` | `go.mod` |
| Lock file | `package-lock.json` | `go.sum` |
| Init | `npm init` | `go mod init` |
| Install | `npm install` | `go mod tidy` |
| Add package | `npm i pkg` | `go get pkg` |
| Update | `npm update` | `go get -u` |
| Remove | `npm uninstall` | ลบ import + `go mod tidy` |
| Run | `npm start` | `go run .` |
| Build | `npm run build` | `go build` |
| Test | `npm test` | `go test ./...` |
| Private | file scope | `internal/` directory |
| Public exports | `export` | Capital letter |

### Package Naming Convention

- ใช้ lowercase
- สั้น กระชับ
- ไม่ใช้ underscore หรือ camelCase
- เป็นคำนาม ไม่ใช่กริยา

```go
// ✅ Good
package user
package http
package json
package booking

// ❌ Bad
package userService    // ใช้ camelCase
package user_handler   // ใช้ underscore
package getUsers       // เป็นกริยา
```

---

## ต่อไป

- [10-testing.md](./10-testing.md) - Testing (การทดสอบ)
