# 07 - HTTP Server (เซิร์ฟเวอร์ HTTP)

## สารบัญ

1. [net/http พื้นฐาน](#nethttp-พื้นฐาน)
2. [Gin Framework](#gin-framework)
3. [Request Handling](#request-handling)
4. [Response](#response)
5. [Middleware](#middleware)
6. [Routing](#routing)
7. [Validation](#validation)
8. [Project Structure](#project-structure)

---

## net/http พื้นฐาน

### TypeScript - Express

```typescript
import express from 'express'

const app = express()
app.use(express.json())

app.get('/hello', (req, res) => {
    res.json({ message: 'Hello World' })
})

app.listen(8080, () => {
    console.log('Server running on :8080')
})
```

### Go - net/http (Standard Library)

```go
package main

import (
    "encoding/json"
    "net/http"
)

func main() {
    http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "message": "Hello World",
        })
    })

    http.ListenAndServe(":8080", nil)
}
```

### Handler Interface

```go
// http.Handler interface
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// Implement Handler interface
type HelloHandler struct{}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello World"))
}

// ใช้งาน
http.Handle("/hello", &HelloHandler{})

// หรือใช้ HandlerFunc (adapter)
http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello World"))
})
```

---

## Gin Framework

Gin = Web framework ยอดนิยมของ Go (ใช้ในโปรเจค Booking Rush)

### ติดตั้ง

```bash
go get -u github.com/gin-gonic/gin
```

### Basic Server

```go
package main

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func main() {
    // สร้าง router
    r := gin.Default()  // มี Logger และ Recovery middleware

    // หรือไม่มี middleware
    r := gin.New()

    // Routes
    r.GET("/hello", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "message": "Hello World",
        })
    })

    // Start server
    r.Run(":8080")
}
```

### gin.H คืออะไร?

```go
// gin.H = map[string]interface{} (shorthand)
gin.H{"key": "value"}

// เหมือนกับ
map[string]interface{}{"key": "value"}
```

---

## Request Handling

### TypeScript - Express

```typescript
// Path params
app.get('/users/:id', (req, res) => {
    const id = req.params.id
})

// Query params
app.get('/users', (req, res) => {
    const page = req.query.page
    const limit = req.query.limit
})

// Body
app.post('/users', (req, res) => {
    const { name, email } = req.body
})

// Headers
app.get('/protected', (req, res) => {
    const token = req.headers.authorization
})
```

### Go - Gin

```go
// Path params
r.GET("/users/:id", func(c *gin.Context) {
    id := c.Param("id")
    c.JSON(200, gin.H{"id": id})
})

// Query params
r.GET("/users", func(c *gin.Context) {
    page := c.DefaultQuery("page", "1")
    limit := c.Query("limit")  // "" ถ้าไม่มี

    // Parse เป็น int
    pageInt, _ := strconv.Atoi(page)
})

// Body - JSON
r.POST("/users", func(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    // ใช้ req.Name, req.Email
})

// Body - Form data
r.POST("/upload", func(c *gin.Context) {
    name := c.PostForm("name")
    file, _ := c.FormFile("file")
})

// Headers
r.GET("/protected", func(c *gin.Context) {
    token := c.GetHeader("Authorization")
    // หรือ
    token := c.Request.Header.Get("Authorization")
})
```

### Request Binding

```go
// Bind struct from JSON body
type CreateUserRequest struct {
    Name     string `json:"name" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
    Age      int    `json:"age" binding:"gte=0,lte=150"`
    Password string `json:"password" binding:"required,min=8"`
}

r.POST("/users", func(c *gin.Context) {
    var req CreateUserRequest

    // ShouldBindJSON - return error
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // BindJSON - abort on error
    if err := c.BindJSON(&req); err != nil {
        return  // c.JSON already called
    }
})

// Bind จาก query params
type ListUsersQuery struct {
    Page  int    `form:"page" binding:"gte=1"`
    Limit int    `form:"limit" binding:"gte=1,lte=100"`
    Sort  string `form:"sort"`
}

r.GET("/users", func(c *gin.Context) {
    var query ListUsersQuery
    if err := c.ShouldBindQuery(&query); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
})

// Bind จาก URI params
type GetUserURI struct {
    ID string `uri:"id" binding:"required,uuid"`
}

r.GET("/users/:id", func(c *gin.Context) {
    var uri GetUserURI
    if err := c.ShouldBindUri(&uri); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
})
```

---

## Response

### TypeScript - Express

```typescript
// JSON
res.json({ message: 'success' })

// Status code
res.status(201).json({ id: 1 })

// Error
res.status(404).json({ error: 'Not found' })

// Redirect
res.redirect('/login')

// File
res.sendFile('/path/to/file')
```

### Go - Gin

```go
// JSON
c.JSON(200, gin.H{"message": "success"})
c.JSON(http.StatusOK, user)  // ใช้ constant

// Status only
c.Status(204)

// String
c.String(200, "Hello %s", name)

// HTML
c.HTML(200, "index.html", gin.H{"title": "Home"})

// Redirect
c.Redirect(302, "/login")
c.Redirect(http.StatusFound, "/login")

// File
c.File("/path/to/file")
c.FileAttachment("/path/to/file", "filename.pdf")

// Data (bytes)
c.Data(200, "text/plain", []byte("raw data"))

// Abort (stop middleware chain)
c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
```

### Response Struct

```go
// Standard response format
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
    Page       int `json:"page"`
    Limit      int `json:"limit"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}

// Helper functions
func Success(c *gin.Context, data interface{}) {
    c.JSON(200, Response{
        Success: true,
        Data:    data,
    })
}

func Error(c *gin.Context, status int, message string) {
    c.JSON(status, Response{
        Success: false,
        Error:   message,
    })
}

func Paginated(c *gin.Context, data interface{}, meta *Meta) {
    c.JSON(200, Response{
        Success: true,
        Data:    data,
        Meta:    meta,
    })
}
```

---

## Middleware

### TypeScript - Express

```typescript
// Middleware function
const logger = (req, res, next) => {
    console.log(`${req.method} ${req.path}`)
    next()
}

app.use(logger)
app.use(express.json())
app.use(cors())

// Route-specific
app.get('/admin', authMiddleware, adminHandler)
```

### Go - Gin

```go
// Middleware function
func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        // ก่อน handler
        c.Next()

        // หลัง handler
        latency := time.Since(start)
        log.Printf("%s %s %d %v",
            c.Request.Method,
            c.Request.URL.Path,
            c.Writer.Status(),
            latency,
        )
    }
}

// Global middleware
r := gin.New()
r.Use(Logger())
r.Use(gin.Recovery())

// Route-specific middleware
r.GET("/admin", AuthMiddleware(), AdminHandler)

// Group middleware
admin := r.Group("/admin")
admin.Use(AuthMiddleware())
{
    admin.GET("/users", ListUsers)
    admin.POST("/users", CreateUser)
}
```

### Common Middleware

```go
// Authentication
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }

        // Validate token
        claims, err := validateToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }

        // Set user info in context
        c.Set("user_id", claims.UserID)
        c.Set("role", claims.Role)

        c.Next()
    }
}

// Get user from context
func GetUserID(c *gin.Context) string {
    userID, _ := c.Get("user_id")
    return userID.(string)
}

// CORS
func CORS() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}

// Rate Limiter (simple)
func RateLimiter(rps int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(rps), rps)

    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatusJSON(429, gin.H{"error": "too many requests"})
            return
        }
        c.Next()
    }
}

// Request ID
func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)
        c.Next()
    }
}
```

---

## Routing

### TypeScript - Express

```typescript
// Basic routes
app.get('/users', listUsers)
app.post('/users', createUser)
app.get('/users/:id', getUser)
app.put('/users/:id', updateUser)
app.delete('/users/:id', deleteUser)

// Router
const userRouter = express.Router()
userRouter.get('/', listUsers)
userRouter.post('/', createUser)
app.use('/users', userRouter)
```

### Go - Gin

```go
// Basic routes
r.GET("/users", listUsers)
r.POST("/users", createUser)
r.GET("/users/:id", getUser)
r.PUT("/users/:id", updateUser)
r.DELETE("/users/:id", deleteUser)

// Route groups
api := r.Group("/api/v1")
{
    // /api/v1/users
    users := api.Group("/users")
    {
        users.GET("", listUsers)
        users.POST("", createUser)
        users.GET("/:id", getUser)
        users.PUT("/:id", updateUser)
        users.DELETE("/:id", deleteUser)
    }

    // /api/v1/bookings
    bookings := api.Group("/bookings")
    bookings.Use(AuthMiddleware())
    {
        bookings.POST("/reserve", reserve)
        bookings.POST("/confirm", confirm)
    }
}

// Static files
r.Static("/assets", "./public/assets")
r.StaticFile("/favicon.ico", "./public/favicon.ico")

// No route handler (404)
r.NoRoute(func(c *gin.Context) {
    c.JSON(404, gin.H{"error": "page not found"})
})
```

---

## Validation

### TypeScript - class-validator

```typescript
import { IsEmail, IsString, MinLength } from 'class-validator'

class CreateUserDto {
    @IsString()
    name: string

    @IsEmail()
    email: string

    @MinLength(8)
    password: string
}
```

### Go - Gin Binding Tags

```go
// Built-in validation with binding tags
type CreateUserRequest struct {
    Name     string `json:"name" binding:"required,min=2,max=100"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Age      int    `json:"age" binding:"gte=0,lte=150"`
    Role     string `json:"role" binding:"oneof=admin user guest"`
    Website  string `json:"website" binding:"omitempty,url"`
}

// Common validation tags
// required     - field ต้องมี
// email        - email format
// url          - URL format
// min=n        - string length >= n
// max=n        - string length <= n
// len=n        - string length == n
// gte=n        - number >= n
// lte=n        - number <= n
// gt=n         - number > n
// lt=n         - number < n
// oneof=a b c  - ค่าต้องเป็นหนึ่งใน a, b, c
// uuid         - UUID format
// datetime     - datetime format
// omitempty    - validate เมื่อมีค่าเท่านั้น
```

### Custom Validation

```go
import "github.com/go-playground/validator/v10"

// Custom validator function
func validateBookingStatus(fl validator.FieldLevel) bool {
    status := fl.Field().String()
    validStatuses := []string{"pending", "confirmed", "cancelled"}
    for _, s := range validStatuses {
        if status == s {
            return true
        }
    }
    return false
}

// Register custom validator
func main() {
    if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
        v.RegisterValidation("bookingstatus", validateBookingStatus)
    }

    r := gin.Default()
    // ...
}

// ใช้งาน
type UpdateBookingRequest struct {
    Status string `json:"status" binding:"required,bookingstatus"`
}
```

---

## Project Structure

### โครงสร้างแบบ Clean Architecture

```
backend-booking/
├── main.go                     # Entry point
├── internal/
│   ├── di/
│   │   └── container.go        # Dependency injection
│   ├── handler/
│   │   └── booking_handler.go  # HTTP handlers
│   ├── service/
│   │   └── booking_service.go  # Business logic
│   ├── repository/
│   │   └── booking_repo.go     # Data access
│   ├── domain/
│   │   └── booking.go          # Entities
│   └── dto/
│       └── booking.go          # Request/Response DTOs
└── go.mod
```

### ตัวอย่าง Handler

```go
// internal/handler/booking_handler.go
package handler

type BookingHandler struct {
    service *service.BookingService
}

func NewBookingHandler(service *service.BookingService) *BookingHandler {
    return &BookingHandler{service: service}
}

// RegisterRoutes - register all routes
func (h *BookingHandler) RegisterRoutes(r *gin.RouterGroup) {
    bookings := r.Group("/bookings")
    {
        bookings.POST("/reserve", h.Reserve)
        bookings.POST("/confirm", h.Confirm)
        bookings.GET("/:id", h.GetByID)
        bookings.GET("", h.List)
    }
}

func (h *BookingHandler) Reserve(c *gin.Context) {
    // 1. Parse request
    var req dto.ReserveRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 2. Get user from context (set by auth middleware)
    userID := c.GetString("user_id")
    req.UserID = userID

    // 3. Call service
    resp, err := h.service.Reserve(c.Request.Context(), &req)
    if err != nil {
        // Handle specific errors
        switch {
        case errors.Is(err, ErrInsufficientSeats):
            c.JSON(409, gin.H{"error": "not enough seats"})
        case errors.Is(err, ErrInvalidInput):
            c.JSON(400, gin.H{"error": err.Error()})
        default:
            log.Printf("Reserve error: %v", err)
            c.JSON(500, gin.H{"error": "internal error"})
        }
        return
    }

    // 4. Return response
    c.JSON(200, resp)
}
```

### Main.go

```go
// main.go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "booking-rush/internal/di"
    "booking-rush/internal/handler"
)

func main() {
    // Initialize dependencies
    container := di.NewContainer()

    // Create router
    r := gin.Default()

    // Global middleware
    r.Use(middleware.RequestID())
    r.Use(middleware.CORS())

    // API routes
    api := r.Group("/api/v1")

    // Auth routes (public)
    authHandler := handler.NewAuthHandler(container.AuthService)
    authHandler.RegisterRoutes(api)

    // Protected routes
    protected := api.Group("")
    protected.Use(middleware.AuthMiddleware(container.JWTSecret))

    // Booking routes
    bookingHandler := handler.NewBookingHandler(container.BookingService)
    bookingHandler.RegisterRoutes(protected)

    // Start server
    if err := r.Run(":8083"); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

---

## ตัวอย่างจาก Booking Rush

```go
// backend-auth/internal/handler/auth_handler.go

type AuthHandler struct {
    service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
    return &AuthHandler{service: service}
}

func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
    auth := r.Group("/auth")
    {
        auth.POST("/register", h.Register)
        auth.POST("/login", h.Login)
        auth.POST("/refresh", h.RefreshToken)
    }
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req dto.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }

    resp, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
    if err != nil {
        if errors.Is(err, ErrInvalidCredentials) {
            c.JSON(401, gin.H{
                "success": false,
                "error":   "invalid email or password",
            })
            return
        }
        log.Printf("Login error: %v", err)
        c.JSON(500, gin.H{
            "success": false,
            "error":   "internal error",
        })
        return
    }

    c.JSON(200, gin.H{
        "success": true,
        "data":    resp,
    })
}
```

---

## สรุป

| หัวข้อ | Express (TypeScript) | Gin (Go) |
|--------|---------------------|----------|
| Create app | `express()` | `gin.Default()` |
| Path param | `req.params.id` | `c.Param("id")` |
| Query param | `req.query.page` | `c.Query("page")` |
| Body | `req.body` | `c.ShouldBindJSON(&req)` |
| Header | `req.headers` | `c.GetHeader("key")` |
| JSON response | `res.json()` | `c.JSON()` |
| Status code | `res.status(code)` | `c.JSON(code, data)` |
| Middleware | `app.use(fn)` | `r.Use(fn)` |
| Route group | `express.Router()` | `r.Group("/path")` |
| Validation | class-validator | binding tags |

### Tips

- ใช้ `gin.Default()` สำหรับ development (มี logger)
- ใช้ `gin.New()` สำหรับ production (เลือก middleware เอง)
- ใช้ `ShouldBind*` แทน `Bind*` (ไม่ abort อัตโนมัติ)
- แยก handler, service, repository ชัดเจน
- ใช้ context.Context สำหรับ cancellation

---

## ต่อไป

- [08-json-serialization.md](./08-json-serialization.md) - JSON handling
