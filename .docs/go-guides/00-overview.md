# 00 - ภาพรวม Go vs TypeScript

## Go คืออะไร?

**Go** (หรือ Golang) เป็นภาษาโปรแกรมที่ Google สร้างขึ้นในปี 2009 ออกแบบมาเพื่อ:

- **ความเร็ว** - compile เป็น binary ทำงานเร็วมาก
- **ความเรียบง่าย** - syntax น้อย เรียนรู้ง่าย
- **Concurrency** (การทำงานพร้อมกัน) - มี goroutine และ channel ในตัว
- **Scalability** (ขยายขนาด) - เหมาะกับระบบขนาดใหญ่

---

## ทำไมต้องเรียน Go?

| เหตุผล | คำอธิบาย |
|--------|----------|
| **Performance** (ประสิทธิภาพ) | เร็วกว่า Node.js 5-10 เท่าในงานหนักๆ |
| **Memory** (หน่วยความจำ) | ใช้ RAM น้อยกว่ามาก |
| **Concurrency** | Goroutine เบากว่า Thread มาก (สร้างได้เป็นล้าน) |
| **Single binary** | compile ได้ไฟล์เดียว deploy ง่าย |
| **Type safety** | Strong typing จับ bug ตั้งแต่ compile time |

### ตัวอย่างในโปรเจค Booking Rush

โปรเจคนี้ต้องรองรับ **10,000 RPS** (requests per second) จึงเลือก Go สำหรับ critical path:

```
บริการที่ใช้ Go (งานหนัก ต้องเร็ว):
├── api-gateway      → Rate limiting, routing
├── auth-service     → JWT authentication
├── ticket-service   → Event catalog (read-heavy)
├── booking-service  → Core booking logic (critical!)
└── payment-service  → Payment processing

บริการที่ใช้ NestJS/TypeScript (งานเบา):
├── notification-service  → ส่ง email (async)
└── analytics-service     → Dashboard (ไม่ urgent)
```

---

## เปรียบเทียบ TypeScript vs Go

### Philosophy (ปรัชญา)

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| **Paradigm** (แนวคิด) | Multi-paradigm (OOP, FP) | Imperative, Concurrent |
| **Typing** | Static (optional) | Static (strict) |
| **Compilation** | → JavaScript (interpreted) | → Binary (compiled) |
| **Error handling** | try/catch (exceptions) | Explicit return (no exceptions) |
| **OOP** | Class, inheritance | Struct, composition |
| **Null safety** | Optional chaining `?.` | Explicit nil checks |

### Syntax Overview (ภาพรวม Syntax)

```typescript
// TypeScript
import express from 'express'

interface User {
    id: number
    name: string
}

class UserService {
    private users: User[] = []

    async getUser(id: number): Promise<User | null> {
        return this.users.find(u => u.id === id) ?? null
    }
}

const app = express()
app.get('/users/:id', async (req, res) => {
    const user = await userService.getUser(parseInt(req.params.id))
    res.json(user)
})
app.listen(8080)
```

```go
// Go
package main

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

type UserService struct {
    users []User
}

func (s *UserService) GetUser(id int) *User {
    for _, u := range s.users {
        if u.ID == id {
            return &u
        }
    }
    return nil
}

func main() {
    r := gin.Default()
    r.GET("/users/:id", func(c *gin.Context) {
        id, _ := strconv.Atoi(c.Param("id"))
        user := userService.GetUser(id)
        c.JSON(http.StatusOK, user)
    })
    r.Run(":8080")
}
```

---

## ความแตกต่างหลัก 10 ข้อ

### 1. ไม่มี Class - ใช้ Struct แทน

```typescript
// TypeScript - มี class
class User {
    constructor(public name: string) {}
    greet() { return `Hello ${this.name}` }
}
```

```go
// Go - ใช้ struct + method
type User struct {
    Name string
}

func (u *User) Greet() string {
    return "Hello " + u.Name
}
```

### 2. ไม่มี Inheritance - ใช้ Composition

```typescript
// TypeScript - inheritance
class Animal { move() {} }
class Dog extends Animal { bark() {} }
```

```go
// Go - composition (embedding)
type Animal struct{}
func (a *Animal) Move() {}

type Dog struct {
    Animal  // embed Animal
}
func (d *Dog) Bark() {}
// Dog มี Move() ด้วยอัตโนมัติ
```

### 3. Interface เป็น Implicit (Duck Typing)

```typescript
// TypeScript - explicit implements
interface Logger { log(msg: string): void }
class ConsoleLogger implements Logger {
    log(msg: string) { console.log(msg) }
}
```

```go
// Go - implicit (ไม่ต้อง implements)
type Logger interface {
    Log(msg string)
}

type ConsoleLogger struct{}
func (c *ConsoleLogger) Log(msg string) {
    fmt.Println(msg)
}
// ConsoleLogger implements Logger อัตโนมัติ!
```

### 4. Error เป็น Value - ไม่มี try/catch

```typescript
// TypeScript
try {
    const result = await riskyOperation()
} catch (error) {
    console.error(error)
}
```

```go
// Go - error เป็น return value
result, err := riskyOperation()
if err != nil {
    log.Println(err)
    return
}
```

### 5. Multiple Return Values

```typescript
// TypeScript - ต้อง return object
function divide(a: number, b: number): { result: number; error?: string } {
    if (b === 0) return { result: 0, error: "div by zero" }
    return { result: a / b }
}
```

```go
// Go - return หลายค่าได้เลย
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("div by zero")
    }
    return a / b, nil
}
```

### 6. Goroutine แทน async/await

```typescript
// TypeScript
async function fetchAll() {
    const [a, b] = await Promise.all([fetchA(), fetchB()])
}
```

```go
// Go - goroutine + channel
func fetchAll() {
    chA := make(chan Result)
    chB := make(chan Result)

    go func() { chA <- fetchA() }()
    go func() { chB <- fetchB() }()

    a := <-chA
    b := <-chB
}
```

### 7. Public/Private ด้วยตัวอักษรตัวแรก

```typescript
// TypeScript
class User {
    public name: string   // public
    private age: number   // private
}
```

```go
// Go - Capital = public, lowercase = private
type User struct {
    Name string  // Public (exported)
    age  int     // private (unexported)
}
```

### 8. ไม่มี Generics แบบเดียวกับ TS (มีใน Go 1.18+)

```typescript
// TypeScript
function first<T>(arr: T[]): T | undefined {
    return arr[0]
}
```

```go
// Go 1.18+ - มี generics แล้ว
func First[T any](arr []T) T {
    return arr[0]
}
```

### 9. Package System ต่างจาก npm

```typescript
// TypeScript
import { Router } from 'express'
import { myFunc } from './utils'
```

```go
// Go - import ทั้ง package แล้วใช้ชื่อ package
import (
    "github.com/gin-gonic/gin"
    "myapp/internal/utils"
)

// ใช้งาน
r := gin.Default()
utils.MyFunc()
```

### 10. Zero Values (ค่าเริ่มต้นอัตโนมัติ)

```typescript
// TypeScript - undefined ถ้าไม่กำหนด
let name: string  // undefined
let count: number // undefined
```

```go
// Go - มี zero value ทุก type
var name string  // "" (empty string)
var count int    // 0
var ok bool      // false
var user *User   // nil
```

---

## เมื่อไหร่ควรใช้ Go vs TypeScript?

### ใช้ Go เมื่อ:

- ต้องการ **performance สูง** (high throughput)
- งาน **CPU-intensive** (ประมวลผลหนัก)
- ต้อง **scale** รองรับ traffic มาก
- สร้าง **CLI tools** หรือ **system utilities**
- งาน **concurrent** หนักๆ (หลายงานพร้อมกัน)

### ใช้ TypeScript/Node.js เมื่อ:

- งาน **I/O-bound** ทั่วไป (API ธรรมดา)
- ต้องการ **ecosystem** กว้าง (npm packages)
- ทีม **familiar** กับ JavaScript
- **Prototype** เร็วๆ
- **Full-stack** (share code frontend-backend)

---

## สรุป

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| Learning curve | คุ้นเคย JS อยู่แล้ว | ต้องเรียนใหม่ แต่ syntax ง่าย |
| Performance | ดี | ดีมาก |
| Concurrency | async/await | goroutine + channel |
| Error handling | try/catch | if err != nil |
| OOP | class + inheritance | struct + composition |
| Type system | Flexible | Strict |
| Deployment | ต้องมี Node.js | Single binary |
| Best for | Web apps, APIs | High-performance services |

---

## ต่อไป

- [01-variables-types.md](./01-variables-types.md) - ตัวแปรและ Types
- [glossary.md](./glossary.md) - คำศัพท์ Go-Thai-English
