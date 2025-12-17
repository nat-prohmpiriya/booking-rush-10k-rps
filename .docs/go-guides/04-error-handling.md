# 04 - Error Handling (การจัดการข้อผิดพลาด)

## สารบัญ

1. [แนวคิด Error ใน Go](#แนวคิด-error-ใน-go)
2. [Basic Error Handling](#basic-error-handling)
3. [สร้าง Error](#สร้าง-error)
4. [Custom Errors](#custom-errors)
5. [Error Wrapping](#error-wrapping-ห่อหุ้ม-error)
6. [Sentinel Errors](#sentinel-errors)
7. [Panic และ Recover](#panic-และ-recover)
8. [Best Practices](#best-practices)

---

## แนวคิด Error ใน Go

### TypeScript - Exception Model

```typescript
// TypeScript ใช้ try/catch (exception model)
async function getUser(id: string): Promise<User> {
    try {
        const user = await db.findUser(id)
        if (!user) {
            throw new Error("User not found")
        }
        return user
    } catch (error) {
        console.error("Failed to get user:", error)
        throw error  // re-throw
    }
}

// เรียกใช้
try {
    const user = await getUser("123")
} catch (error) {
    // handle error
}
```

### Go - Error as Value

```go
// Go ใช้ error เป็น value (ไม่มี exception)
func getUser(id string) (*User, error) {
    user, err := db.FindUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    if user == nil {
        return nil, errors.New("user not found")
    }
    return user, nil
}

// เรียกใช้
user, err := getUser("123")
if err != nil {
    // handle error
    return
}
// ใช้ user ต่อได้
```

### ทำไม Go ไม่ใช้ Exception?

| Exception (TypeScript) | Error as Value (Go) |
|------------------------|---------------------|
| ซ่อน error flow | เห็น error flow ชัดเจน |
| อาจลืม handle | บังคับให้ handle |
| Stack unwinding ช้า | ไม่มี overhead |
| Error อาจมาจากไหนก็ได้ | รู้ว่า error มาจากไหน |

---

## Basic Error Handling

### Pattern พื้นฐาน: `if err != nil`

```go
// Pattern ที่ใช้บ่อยที่สุดใน Go
func readConfig(path string) (*Config, error) {
    // Step 1: Open file
    file, err := os.Open(path)
    if err != nil {
        return nil, err  // return error ทันที
    }
    defer file.Close()

    // Step 2: Read content
    data, err := io.ReadAll(file)
    if err != nil {
        return nil, err
    }

    // Step 3: Parse JSON
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

### Handle หรือ Return

```go
// Option 1: Handle error - จัดการเอง
func processUser(id string) {
    user, err := getUser(id)
    if err != nil {
        log.Printf("Warning: could not get user %s: %v", id, err)
        // ใช้ default value หรือ skip
        return
    }
    // process user
}

// Option 2: Return error - ส่งต่อให้ caller
func processUser(id string) error {
    user, err := getUser(id)
    if err != nil {
        return err  // ให้ caller จัดการ
    }
    // process user
    return nil
}

// Option 3: Wrap and return - เพิ่ม context แล้วส่งต่อ
func processUser(id string) error {
    user, err := getUser(id)
    if err != nil {
        return fmt.Errorf("process user %s: %w", id, err)
    }
    // process user
    return nil
}
```

### ไม่สนใจ Error (ใช้ระวัง!)

```go
// ใช้ _ เพื่อ ignore error
data, _ := json.Marshal(user)  // ignore error (ไม่แนะนำ)

// กรณีที่รับได้
// - ปิดไฟล์ที่อ่านอย่างเดียว (error ไม่มีผลกระทบ)
defer file.Close()  // ignore error OK

// - Function ที่รู้ว่าไม่มีทาง error
n, _ := fmt.Fprintf(os.Stdout, "hello")  // stdout ไม่ค่อย error
```

---

## สร้าง Error

### errors.New() - Simple Error

```go
import "errors"

// สร้าง error ง่ายๆ
err := errors.New("something went wrong")

// ใช้ใน function
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

### fmt.Errorf() - Formatted Error

```go
import "fmt"

// Error พร้อม format
userID := "123"
err := fmt.Errorf("user %s not found", userID)

// ใช้ใน function
func getUser(id string) (*User, error) {
    user, err := db.FindByID(id)
    if err != nil {
        return nil, fmt.Errorf("database error for user %s: %v", id, err)
    }
    if user == nil {
        return nil, fmt.Errorf("user %s not found", id)
    }
    return user, nil
}
```

---

## Custom Errors

### TypeScript - Custom Error Class

```typescript
class ValidationError extends Error {
    constructor(
        public field: string,
        public message: string
    ) {
        super(message)
        this.name = "ValidationError"
    }
}

throw new ValidationError("email", "Invalid email format")
```

### Go - Custom Error Type

```go
// error interface มีแค่ 1 method
type error interface {
    Error() string
}

// สร้าง custom error โดย implement Error()
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// ใช้งาน
func validateEmail(email string) error {
    if !strings.Contains(email, "@") {
        return &ValidationError{
            Field:   "email",
            Message: "invalid email format",
        }
    }
    return nil
}

// Check error type
err := validateEmail("invalid")
if err != nil {
    var valErr *ValidationError
    if errors.As(err, &valErr) {
        fmt.Printf("Field: %s, Message: %s\n", valErr.Field, valErr.Message)
    }
}
```

### Multiple Error Fields

```go
type APIError struct {
    Code       int    `json:"code"`
    Message    string `json:"message"`
    Details    string `json:"details,omitempty"`
    RequestID  string `json:"request_id,omitempty"`
}

func (e *APIError) Error() string {
    return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// HTTP status code
func (e *APIError) StatusCode() int {
    return e.Code
}

// Constructor functions
func NewNotFoundError(resource string) *APIError {
    return &APIError{
        Code:    404,
        Message: fmt.Sprintf("%s not found", resource),
    }
}

func NewBadRequestError(message string) *APIError {
    return &APIError{
        Code:    400,
        Message: message,
    }
}

func NewInternalError(err error) *APIError {
    return &APIError{
        Code:    500,
        Message: "internal server error",
        Details: err.Error(),
    }
}
```

---

## Error Wrapping (ห่อหุ้ม Error)

Go 1.13+ มี error wrapping ใช้ `%w` และ `errors.Is/As`

### Wrap Error ด้วย %w

```go
// เดิม: ใช้ %v - สูญเสีย original error
err := fmt.Errorf("failed to read config: %v", originalErr)

// ใหม่: ใช้ %w - เก็บ original error ไว้
err := fmt.Errorf("failed to read config: %w", originalErr)
```

### errors.Is() - Check Error Identity

```go
import (
    "errors"
    "os"
)

// Sentinel error
var ErrNotFound = errors.New("not found")

func getUser(id string) (*User, error) {
    user, err := db.FindByID(id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound  // แปลงเป็น domain error
        }
        return nil, fmt.Errorf("database error: %w", err)
    }
    return user, nil
}

// เรียกใช้
user, err := getUser("123")
if err != nil {
    if errors.Is(err, ErrNotFound) {
        // user not found - อาจ return 404
    } else {
        // other error - อาจ return 500
    }
}

// errors.Is ดู wrapped errors ด้วย
err := fmt.Errorf("service error: %w",
    fmt.Errorf("repository error: %w", ErrNotFound))

errors.Is(err, ErrNotFound)  // true!
```

### errors.As() - Extract Error Type

```go
// ดึง error ที่เป็น type ที่ต้องการ
func handleError(err error) {
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        // err หรือ error ที่ wrap อยู่เป็น *APIError
        fmt.Printf("API Error: %d - %s\n", apiErr.Code, apiErr.Message)
        return
    }

    var valErr *ValidationError
    if errors.As(err, &valErr) {
        fmt.Printf("Validation Error: %s - %s\n", valErr.Field, valErr.Message)
        return
    }

    // Unknown error
    fmt.Printf("Unknown error: %v\n", err)
}

// errors.As ดู wrapped errors ด้วย
originalErr := &APIError{Code: 404, Message: "user not found"}
wrappedErr := fmt.Errorf("service error: %w", originalErr)

var apiErr *APIError
errors.As(wrappedErr, &apiErr)  // true! apiErr = originalErr
```

### errors.Unwrap() - Get Original Error

```go
// Unwrap ทีละชั้น
err1 := errors.New("original error")
err2 := fmt.Errorf("wrapped: %w", err1)
err3 := fmt.Errorf("double wrapped: %w", err2)

errors.Unwrap(err3)  // err2
errors.Unwrap(err2)  // err1
errors.Unwrap(err1)  // nil
```

---

## Sentinel Errors

Sentinel error = error ที่ประกาศเป็น package-level variable

```go
// ประกาศ sentinel errors
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
    ErrInvalidInput  = errors.New("invalid input")
    ErrAlreadyExists = errors.New("already exists")
)

// ใช้ใน repository
func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    var user User
    err := r.db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = $1", id).Scan(&user)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("query user: %w", err)
    }
    return &user, nil
}

// ใช้ใน handler
func (h *Handler) GetUser(c *gin.Context) {
    user, err := h.service.GetUser(c, c.Param("id"))
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            c.JSON(404, gin.H{"error": "user not found"})
            return
        }
        c.JSON(500, gin.H{"error": "internal error"})
        return
    }
    c.JSON(200, user)
}
```

---

## Panic และ Recover

### Panic - สำหรับ Unrecoverable Error

```typescript
// TypeScript - throw ได้ทุกที่
throw new Error("Critical error!")
```

```go
// Go - panic สำหรับกรณีร้ายแรงมาก
func mustLoadConfig() *Config {
    config, err := loadConfig()
    if err != nil {
        panic(fmt.Sprintf("failed to load config: %v", err))
    }
    return config
}

// ใช้ panic เมื่อ:
// 1. Programming error (bug)
// 2. Configuration ที่ขาดไม่ได้ตอน startup
// 3. Impossible state ที่ไม่ควรเกิด

// ไม่ควรใช้ panic สำหรับ:
// - User input validation
// - File not found
// - Network errors
// - Database errors
```

### Recover - จับ Panic

```go
// recover() จับ panic ได้เฉพาะใน defer
func safeExecute(fn func()) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic recovered: %v", r)
        }
    }()

    fn()  // ถ้า panic จะถูก recover
    return nil
}

// ใช้งาน
err := safeExecute(func() {
    panic("something went wrong")
})
fmt.Println(err)  // "panic recovered: something went wrong"
```

### HTTP Server Recovery Middleware

```go
// Gin recovery middleware (built-in)
r := gin.Default()  // มี Recovery() อยู่แล้ว

// Custom recovery
func RecoveryMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if r := recover(); r != nil {
                // Log panic
                log.Printf("Panic recovered: %v\nStack: %s", r, debug.Stack())

                // Return 500
                c.AbortWithStatusJSON(500, gin.H{
                    "error": "internal server error",
                })
            }
        }()
        c.Next()
    }
}
```

---

## Best Practices

### 1. Always Handle Errors

```go
// ❌ Bad - ignore error
data, _ := json.Marshal(user)

// ✅ Good - handle error
data, err := json.Marshal(user)
if err != nil {
    return fmt.Errorf("marshal user: %w", err)
}
```

### 2. Add Context When Wrapping

```go
// ❌ Bad - no context
if err != nil {
    return err
}

// ✅ Good - add context
if err != nil {
    return fmt.Errorf("create user %s: %w", user.Email, err)
}
```

### 3. Don't Wrap Twice

```go
// ❌ Bad - wrap then log then wrap again
if err != nil {
    log.Printf("error: %v", err)
    return fmt.Errorf("failed: %w", err)
}

// ✅ Good - wrap once at the boundary
if err != nil {
    return fmt.Errorf("create user: %w", err)
}
```

### 4. Use errors.Is/As Instead of Type Assertion

```go
// ❌ Bad - type assertion
if e, ok := err.(*ValidationError); ok {
    // ...
}

// ✅ Good - errors.As (works with wrapped errors)
var valErr *ValidationError
if errors.As(err, &valErr) {
    // ...
}
```

### 5. Define Errors at Package Level

```go
// ❌ Bad - create error inline
return errors.New("not found")

// ✅ Good - sentinel error
var ErrNotFound = errors.New("not found")
return ErrNotFound
```

### 6. Error Messages

```go
// ❌ Bad - start with capital, end with punctuation
return errors.New("User not found.")

// ✅ Good - lowercase, no punctuation
return errors.New("user not found")

// ❌ Bad - redundant "error" or "failed"
return fmt.Errorf("error: failed to create user: %w", err)

// ✅ Good - concise
return fmt.Errorf("create user: %w", err)
```

---

## ตัวอย่างจาก Booking Rush

```go
// pkg/errors/errors.go
var (
    ErrNotFound         = errors.New("not found")
    ErrUnauthorized     = errors.New("unauthorized")
    ErrForbidden        = errors.New("forbidden")
    ErrInvalidInput     = errors.New("invalid input")
    ErrInsufficientSeat = errors.New("insufficient seats")
    ErrBookingExpired   = errors.New("booking expired")
    ErrPaymentFailed    = errors.New("payment failed")
)

// backend-booking/internal/service/booking_service.go
func (s *BookingService) Reserve(ctx context.Context, req *dto.ReserveRequest) (*dto.ReserveResponse, error) {
    // Validate
    if req.Quantity < 1 || req.Quantity > 10 {
        return nil, fmt.Errorf("%w: quantity must be 1-10", ErrInvalidInput)
    }

    // Check inventory with Redis Lua script
    available, err := s.redis.CheckAndReserve(ctx, req.ZoneID, req.Quantity)
    if err != nil {
        return nil, fmt.Errorf("check inventory: %w", err)
    }
    if !available {
        return nil, ErrInsufficientSeat
    }

    // Create booking
    booking := &domain.Booking{
        ID:          uuid.New().String(),
        UserID:      req.UserID,
        EventID:     req.EventID,
        ZoneID:      req.ZoneID,
        Quantity:    req.Quantity,
        TotalAmount: int64(req.Quantity) * req.UnitPrice,
        Status:      domain.BookingStatusPending,
        ExpiresAt:   time.Now().Add(15 * time.Minute),
    }

    if err := s.repo.Create(ctx, booking); err != nil {
        // Rollback Redis
        _ = s.redis.ReleaseSeats(ctx, req.ZoneID, req.Quantity)
        return nil, fmt.Errorf("create booking: %w", err)
    }

    return &dto.ReserveResponse{
        BookingID: booking.ID,
        ExpiresAt: booking.ExpiresAt,
    }, nil
}

// backend-booking/internal/handler/booking_handler.go
func (h *BookingHandler) Reserve(c *gin.Context) {
    var req dto.ReserveRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    resp, err := h.service.Reserve(c, &req)
    if err != nil {
        switch {
        case errors.Is(err, ErrInvalidInput):
            c.JSON(400, gin.H{"error": err.Error()})
        case errors.Is(err, ErrInsufficientSeat):
            c.JSON(409, gin.H{"error": "not enough seats available"})
        default:
            log.Printf("Reserve error: %v", err)
            c.JSON(500, gin.H{"error": "internal error"})
        }
        return
    }

    c.JSON(200, resp)
}
```

---

## สรุป

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| Error model | Exception (try/catch) | Error as value |
| Create error | `throw new Error()` | `errors.New()` |
| Format error | Template literal | `fmt.Errorf()` |
| Custom error | `class extends Error` | `type + Error() method` |
| Check type | `instanceof` | `errors.As()` |
| Check value | `===` | `errors.Is()` |
| Wrap error | - | `fmt.Errorf("%w", err)` |
| Unrecoverable | `throw` | `panic()` |
| Catch panic | `try/catch` | `recover()` |

### Error Handling Checklist

- [ ] ทุก error ต้อง handle หรือ return
- [ ] เพิ่ม context เมื่อ wrap error
- [ ] ใช้ `%w` สำหรับ wrap error
- [ ] ใช้ `errors.Is/As` แทน type assertion
- [ ] สร้าง sentinel errors สำหรับ domain errors
- [ ] Panic เฉพาะกรณีร้ายแรงจริงๆ

---

## ต่อไป

- [05-collections.md](./05-collections.md) - Arrays, Slices, Maps
