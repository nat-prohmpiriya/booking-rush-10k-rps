# 11 - Cheatsheet (สรุปรวม)

## Quick Reference สำหรับ TypeScript Developer

---

## Variables และ Types

```go
// ประกาศตัวแปร
var name string = "John"     // explicit type
name := "John"               // short declaration (ใช้บ่อยสุด)
const PI = 3.14159           // constant

// Basic types
var i int = 42               // int (platform dependent)
var i64 int64 = 42           // int64
var f float64 = 3.14         // float64
var s string = "hello"       // string
var b bool = true            // bool
var p *int = nil             // pointer

// Zero values
var s string   // ""
var i int      // 0
var b bool     // false
var p *int     // nil
```

| TypeScript | Go |
|------------|-----|
| `let x = 1` | `x := 1` |
| `const x = 1` | `const x = 1` |
| `number` | `int`, `float64` |
| `string` | `string` |
| `boolean` | `bool` |
| `null/undefined` | `nil` |
| `any` | `interface{}` หรือ `any` |

---

## Functions

```go
// Basic function
func add(a, b int) int {
    return a + b
}

// Multiple returns
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// Named returns
func getStats() (min int, max int) {
    return 1, 100
}

// Variadic
func sum(nums ...int) int {
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}

// Anonymous function
fn := func(x int) int { return x * 2 }
```

| TypeScript | Go |
|------------|-----|
| `function f()` | `func f()` |
| `(a: number) => a * 2` | `func(a int) int { return a * 2 }` |
| `...args` | `args ...int` |
| `return { a, b }` | `return a, b` |

---

## Structs และ Interfaces

```go
// Struct (เหมือน class)
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    email string // unexported (private)
}

// Method
func (u *User) GetEmail() string {
    return u.email
}

// Constructor pattern
func NewUser(name string) *User {
    return &User{Name: name}
}

// Interface (implicit)
type Logger interface {
    Log(msg string)
}

// Embedding (composition)
type Admin struct {
    User      // embedded
    Role string
}
```

| TypeScript | Go |
|------------|-----|
| `class User {}` | `type User struct {}` |
| `constructor()` | `func NewUser() *User` |
| `this.name` | `u.Name` (receiver) |
| `public` | ตัวอักษรใหญ่ `Name` |
| `private` | ตัวอักษรเล็ก `name` |
| `implements` | implicit (duck typing) |
| `extends` | embedding |

---

## Error Handling

```go
// Return error
func getUser(id string) (*User, error) {
    user, err := db.FindByID(id)
    if err != nil {
        return nil, fmt.Errorf("find user: %w", err)
    }
    return user, nil
}

// Handle error
user, err := getUser("123")
if err != nil {
    log.Printf("Error: %v", err)
    return
}

// Check error type
if errors.Is(err, ErrNotFound) { }

// Extract error type
var apiErr *APIError
if errors.As(err, &apiErr) { }
```

| TypeScript | Go |
|------------|-----|
| `throw new Error()` | `return errors.New()` |
| `try {} catch {}` | `if err != nil {}` |
| `instanceof` | `errors.As()` |
| `===` | `errors.Is()` |

---

## Collections

```go
// Slice (dynamic array)
nums := []int{1, 2, 3}
nums = append(nums, 4)
len(nums)  // 4

// Map
scores := map[string]int{
    "alice": 100,
    "bob":   85,
}
scores["charlie"] = 90
value, ok := scores["alice"]  // comma ok idiom
delete(scores, "bob")

// Loop
for i := 0; i < 10; i++ { }
for i, v := range slice { }
for k, v := range map { }
```

| TypeScript | Go |
|------------|-----|
| `number[]` | `[]int` |
| `push()` | `append()` |
| `length` | `len()` |
| `Record<K,V>` | `map[K]V` |
| `"key" in obj` | `_, ok := m["key"]` |
| `for...of` | `for range` |
| `map()`, `filter()` | ต้องเขียน loop |

---

## Concurrency

```go
// Goroutine
go func() {
    // runs concurrently
}()

// Channel
ch := make(chan string)
ch <- "hello"    // send
msg := <-ch      // receive
close(ch)

// Select
select {
case msg := <-ch1:
    fmt.Println(msg)
case <-time.After(5 * time.Second):
    fmt.Println("timeout")
}

// WaitGroup
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    // work
}()
wg.Wait()

// Mutex
var mu sync.Mutex
mu.Lock()
// critical section
mu.Unlock()

// Context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

| TypeScript | Go |
|------------|-----|
| `async/await` | goroutine + channel |
| `Promise.all()` | `sync.WaitGroup` |
| `Promise.race()` | `select` |
| `AbortController` | `context.Context` |

---

## HTTP Server (Gin)

```go
// Setup
r := gin.Default()

// Routes
r.GET("/users/:id", getUser)
r.POST("/users", createUser)

// Handler
func getUser(c *gin.Context) {
    id := c.Param("id")           // path param
    page := c.Query("page")        // query param
    token := c.GetHeader("Authorization")

    c.JSON(200, gin.H{"id": id})
}

func createUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    c.JSON(201, user)
}

// Middleware
r.Use(func(c *gin.Context) {
    // before
    c.Next()
    // after
})

// Start
r.Run(":8080")
```

| Express | Gin |
|---------|-----|
| `req.params.id` | `c.Param("id")` |
| `req.query.page` | `c.Query("page")` |
| `req.body` | `c.ShouldBindJSON(&req)` |
| `res.json()` | `c.JSON()` |
| `app.use(fn)` | `r.Use(fn)` |

---

## JSON

```go
// Struct tags
type User struct {
    ID       string `json:"id"`
    Name     string `json:"name,omitempty"`
    Password string `json:"-"`
}

// Marshal (encode)
data, err := json.Marshal(user)

// Unmarshal (decode)
var user User
err := json.Unmarshal(data, &user)

// Stream
json.NewEncoder(w).Encode(user)
json.NewDecoder(r.Body).Decode(&user)
```

| TypeScript | Go |
|------------|-----|
| `JSON.stringify()` | `json.Marshal()` |
| `JSON.parse()` | `json.Unmarshal()` |

---

## Modules และ Packages

```bash
# สร้าง module
go mod init github.com/user/project

# เพิ่ม dependency
go get github.com/gin-gonic/gin

# Tidy dependencies
go mod tidy

# Build
go build ./...

# Run
go run .

# Test
go test ./...
```

| npm | go |
|-----|-----|
| `npm init` | `go mod init` |
| `npm install` | `go mod tidy` |
| `npm install pkg` | `go get pkg` |
| `npm run build` | `go build` |
| `npm start` | `go run .` |
| `npm test` | `go test ./...` |

---

## Testing

```go
// sum_test.go
func TestSum(t *testing.T) {
    result := Sum(1, 2)
    if result != 3 {
        t.Errorf("got %d, want 3", result)
    }
}

// Table-driven test
func TestSum(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, 1, 0},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := Sum(tt.a, tt.b); got != tt.expected {
                t.Errorf("got %d, want %d", got, tt.expected)
            }
        })
    }
}

// With testify
import "github.com/stretchr/testify/assert"

func TestSum(t *testing.T) {
    assert.Equal(t, 3, Sum(1, 2))
}
```

```bash
go test ./...           # run tests
go test -v ./...        # verbose
go test -cover ./...    # coverage
go test -bench=. ./...  # benchmarks
```

---

## Common Patterns

### Constructor

```go
type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}
```

### Options Pattern

```go
type Option func(*Config)

func WithPort(port int) Option {
    return func(c *Config) { c.Port = port }
}

func NewServer(opts ...Option) *Server {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    return &Server{cfg: cfg}
}

// ใช้งาน
server := NewServer(WithPort(8080))
```

### Functional Iteration

```go
// Map
func Map[T, U any](slice []T, fn func(T) U) []U {
    result := make([]U, len(slice))
    for i, v := range slice {
        result[i] = fn(v)
    }
    return result
}

// Filter
func Filter[T any](slice []T, fn func(T) bool) []T {
    var result []T
    for _, v := range slice {
        if fn(v) {
            result = append(result, v)
        }
    }
    return result
}
```

---

## Common Imports

```go
import (
    // Standard library
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

    // Third-party
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "github.com/go-redis/redis/v8"
    "go.uber.org/zap"
)
```

---

## Symbols Cheatsheet

| Symbol | ชื่อ | ความหมาย |
|--------|------|----------|
| `:=` | short declaration | ประกาศ + กำหนดค่า |
| `*` | pointer/dereference | ชี้ไปที่ address / เข้าถึงค่า |
| `&` | address-of | เอา address ของตัวแปร |
| `<-` | channel operator | ส่ง/รับข้อมูลผ่าน channel |
| `...` | variadic/spread | รับหลายค่า / กระจายค่า |
| `_` | blank identifier | ไม่สนใจค่านี้ |
| `{}` | composite literal | สร้าง struct/slice/map |

---

## Format Verbs (fmt)

| Verb | ใช้กับ | ตัวอย่าง |
|------|--------|----------|
| `%v` | ค่าใดๆ | `fmt.Printf("%v", user)` |
| `%+v` | struct + field names | `{Name:John Age:25}` |
| `%#v` | Go syntax | `main.User{Name:"John"}` |
| `%T` | type | `main.User` |
| `%s` | string | `"hello"` |
| `%d` | integer | `42` |
| `%f` | float | `3.140000` |
| `%.2f` | float 2 decimal | `3.14` |
| `%t` | bool | `true` |
| `%p` | pointer | `0xc0000...` |
| `%q` | quoted string | `"hello"` |

---

## สุดท้าย - คำแนะนำ

1. **เรียนรู้จากโค้ดจริง** - อ่าน `backend-booking/internal/` ในโปรเจค
2. **ฝึกเขียน tests** - Go มี testing ที่ดีมาก
3. **ใช้ go fmt** - format code อัตโนมัติ
4. **อ่าน error ให้ดี** - Go compiler error messages ชัดเจน
5. **ใช้ IDE ที่ดี** - VS Code + Go extension หรือ GoLand

---

## Links

- [Go Tour](https://go.dev/tour/) - Interactive tutorial
- [Go by Example](https://gobyexample.com/) - Code examples
- [Effective Go](https://go.dev/doc/effective_go) - Best practices
- [Go Proverbs](https://go-proverbs.github.io/) - Philosophy

---

**Happy Coding! จาก TypeScript สู่ Go** 🚀
