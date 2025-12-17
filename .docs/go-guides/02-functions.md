# 02 - Functions (ฟังก์ชัน)

## สารบัญ

1. [Basic Functions](#basic-functions-ฟังก์ชันพื้นฐาน)
2. [Parameters](#parameters-พารามิเตอร์)
3. [Return Values](#return-values-ค่าส่งกลับ)
4. [Multiple Returns](#multiple-returns-คืนค่าหลายตัว)
5. [Named Returns](#named-returns)
6. [Variadic Functions](#variadic-functions-รับหลายค่า)
7. [Anonymous Functions](#anonymous-functions-ฟังก์ชันไม่มีชื่อ)
8. [Closures](#closures-คลอเชอร์)
9. [Defer](#defer-เลื่อนการทำงาน)
10. [Methods](#methods-เมธอด)

---

## Basic Functions (ฟังก์ชันพื้นฐาน)

### TypeScript

```typescript
// Function declaration
function greet(name: string): string {
    return `Hello, ${name}!`
}

// Arrow function
const greetArrow = (name: string): string => `Hello, ${name}!`

// Function expression
const greetExpr = function(name: string): string {
    return `Hello, ${name}!`
}
```

### Go

```go
// Function declaration (วิธีเดียว)
func greet(name string) string {
    return "Hello, " + name + "!"
}

// Go ไม่มี arrow function
// แต่มี anonymous function (function expression)
greetExpr := func(name string) string {
    return "Hello, " + name + "!"
}
```

### Syntax ต่างกัน

```typescript
// TypeScript: function name(param: type): returnType
function add(a: number, b: number): number {
    return a + b
}
```

```go
// Go: func name(param type) returnType
func add(a int, b int) int {
    return a + b
}

// Short syntax - same type parameters
func add(a, b int) int {
    return a + b
}
```

---

## Parameters (พารามิเตอร์)

### TypeScript

```typescript
// Required parameters
function required(a: number, b: number): number {
    return a + b
}

// Optional parameters
function optional(a: number, b?: number): number {
    return a + (b ?? 0)
}

// Default parameters
function withDefault(a: number, b: number = 10): number {
    return a + b
}

// Destructured parameters
function config({ host, port }: { host: string; port: number }) {
    console.log(host, port)
}
```

### Go

```go
// Required parameters (Go ไม่มี optional parameters)
func required(a int, b int) int {
    return a + b
}

// ไม่มี optional parameters - ใช้ pointer หรือ struct แทน
func optional(a int, b *int) int {
    if b == nil {
        return a
    }
    return a + *b
}

// ไม่มี default parameters - ใช้ variadic หรือ options pattern
func withDefault(a int, opts ...int) int {
    b := 10  // default value
    if len(opts) > 0 {
        b = opts[0]
    }
    return a + b
}

// Options pattern (แนะนำสำหรับ config)
type Config struct {
    Host string
    Port int
}

func configFunc(cfg Config) {
    fmt.Println(cfg.Host, cfg.Port)
}

// Functional options pattern (advanced)
type Option func(*Config)

func WithPort(port int) Option {
    return func(c *Config) {
        c.Port = port
    }
}

func NewConfig(opts ...Option) *Config {
    cfg := &Config{
        Host: "localhost",  // default
        Port: 8080,         // default
    }
    for _, opt := range opts {
        opt(cfg)
    }
    return cfg
}

// ใช้งาน
cfg := NewConfig(WithPort(9000))
```

---

## Return Values (ค่าส่งกลับ)

### TypeScript

```typescript
// Single return
function add(a: number, b: number): number {
    return a + b
}

// No return (void)
function log(msg: string): void {
    console.log(msg)
}

// Return object
function getUser(): { name: string; age: number } {
    return { name: "John", age: 25 }
}
```

### Go

```go
// Single return
func add(a, b int) int {
    return a + b
}

// No return (ไม่ต้องระบุ void)
func log(msg string) {
    fmt.Println(msg)
}

// Return struct
type User struct {
    Name string
    Age  int
}

func getUser() User {
    return User{Name: "John", Age: 25}
}

// Return pointer (ถ้าจะ return nil ได้)
func findUser(id string) *User {
    // ... หา user
    if notFound {
        return nil
    }
    return &User{Name: "John", Age: 25}
}
```

---

## Multiple Returns (คืนค่าหลายตัว)

**จุดเด่นสำคัญของ Go!** - คืนค่าได้หลายตัว โดยเฉพาะ `(result, error)`

### TypeScript - ต้อง return object/tuple

```typescript
// Return object
function divide(a: number, b: number): { result: number; error?: string } {
    if (b === 0) {
        return { result: 0, error: "division by zero" }
    }
    return { result: a / b }
}

const { result, error } = divide(10, 2)
if (error) {
    console.error(error)
} else {
    console.log(result)
}

// Return tuple
function divideAsTuple(a: number, b: number): [number, string | null] {
    if (b === 0) {
        return [0, "division by zero"]
    }
    return [a / b, null]
}

const [result2, error2] = divideAsTuple(10, 2)
```

### Go - Multiple return values (built-in)

```go
// คืนค่าหลายตัวได้เลย
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// รับค่า
result, err := divide(10, 2)
if err != nil {
    fmt.Println("Error:", err)
    return
}
fmt.Println("Result:", result)

// ไม่สนใจค่าที่ return ใช้ _ (blank identifier)
result, _ := divide(10, 2)  // ignore error (ไม่แนะนำ)
_, err := divide(10, 0)      // ignore result
```

### Common Patterns

```go
// Pattern 1: value, error (ใช้บ่อยสุด)
func ReadFile(path string) ([]byte, error) {
    // ...
}

data, err := ReadFile("config.json")
if err != nil {
    return nil, fmt.Errorf("failed to read config: %w", err)
}

// Pattern 2: value, ok (comma ok idiom)
func GetCache(key string) (string, bool) {
    value, exists := cache[key]
    return value, exists
}

value, ok := GetCache("user:123")
if !ok {
    // cache miss
}

// Pattern 3: ค่าหลายตัว
func MinMax(numbers []int) (min int, max int) {
    // ...
    return minVal, maxVal
}

min, max := MinMax(numbers)
```

---

## Named Returns

Go สามารถตั้งชื่อ return values ได้

```go
// Named return values
func divide(a, b int) (result int, err error) {
    if b == 0 {
        err = errors.New("division by zero")
        return  // naked return - return result, err
    }
    result = a / b
    return  // naked return - return result, err
}

// ประโยชน์: เห็น signature ชัดเจนว่า return อะไร
func GetUserStats(id string) (posts int, followers int, following int, err error) {
    // ...
}

// Document ชัดเจนกว่า
func GetUserStats(id string) (int, int, int, error) {
    // posts? followers? following? ไม่รู้ลำดับ
}
```

### ข้อควรระวัง Named Returns

```go
// Shadowing - ระวัง!
func example() (result int, err error) {
    result := 10  // สร้างตัวแปรใหม่! (shadowing)
    return        // return 0, nil (ไม่ใช่ 10!)
}

// ถูกต้อง
func example() (result int, err error) {
    result = 10   // assign ไม่ใช่ :=
    return        // return 10, nil
}
```

---

## Variadic Functions (รับหลายค่า)

### TypeScript - Rest parameters

```typescript
// Rest parameters
function sum(...numbers: number[]): number {
    return numbers.reduce((acc, n) => acc + n, 0)
}

sum(1, 2, 3)        // 6
sum(1, 2, 3, 4, 5)  // 15

// Spread array
const nums = [1, 2, 3]
sum(...nums)  // 6
```

### Go - Variadic

```go
// Variadic function
func sum(numbers ...int) int {
    total := 0
    for _, n := range numbers {
        total += n
    }
    return total
}

sum(1, 2, 3)        // 6
sum(1, 2, 3, 4, 5)  // 15

// Spread slice
nums := []int{1, 2, 3}
sum(nums...)  // 6 - ใช้ ... หลัง slice

// ผสม fixed + variadic
func printf(format string, args ...interface{}) {
    // format = fixed parameter
    // args = variadic (0 หรือมากกว่า)
}
```

---

## Anonymous Functions (ฟังก์ชันไม่มีชื่อ)

### TypeScript

```typescript
// Arrow function
const double = (n: number): number => n * 2

// IIFE (Immediately Invoked Function Expression)
const result = ((n: number) => n * 2)(5)  // 10

// Callback
numbers.map((n) => n * 2)
numbers.filter((n) => n > 5)
```

### Go

```go
// Anonymous function
double := func(n int) int {
    return n * 2
}

// IIFE
result := func(n int) int {
    return n * 2
}(5)  // 10

// Callback (ไม่มี map/filter built-in)
// ต้องเขียน helper เอง
func Map(numbers []int, fn func(int) int) []int {
    result := make([]int, len(numbers))
    for i, n := range numbers {
        result[i] = fn(n)
    }
    return result
}

doubled := Map(numbers, func(n int) int {
    return n * 2
})

// หรือใช้ loop ปกติ (Go style)
var doubled []int
for _, n := range numbers {
    doubled = append(doubled, n*2)
}
```

---

## Closures (คลอเชอร์)

Closure = ฟังก์ชันที่เข้าถึงตัวแปรนอก scope ได้

### TypeScript

```typescript
function counter(): () => number {
    let count = 0
    return () => {
        count++
        return count
    }
}

const increment = counter()
console.log(increment())  // 1
console.log(increment())  // 2
console.log(increment())  // 3
```

### Go

```go
func counter() func() int {
    count := 0
    return func() int {
        count++  // เข้าถึงตัวแปรนอก scope
        return count
    }
}

increment := counter()
fmt.Println(increment())  // 1
fmt.Println(increment())  // 2
fmt.Println(increment())  // 3

// ระวัง closure ใน loop!
// ผิด - ทุก goroutine ใช้ค่า i เดียวกัน
for i := 0; i < 3; i++ {
    go func() {
        fmt.Println(i)  // อาจได้ 3, 3, 3
    }()
}

// ถูก - capture ค่า i ในแต่ละ iteration
for i := 0; i < 3; i++ {
    go func(n int) {
        fmt.Println(n)  // 0, 1, 2 (อาจสลับลำดับ)
    }(i)
}

// หรือ (Go 1.22+)
for i := 0; i < 3; i++ {
    i := i  // shadow variable
    go func() {
        fmt.Println(i)
    }()
}
```

---

## Defer (เลื่อนการทำงาน)

`defer` = รันตอนจบ function (เหมือน `finally` แต่ไม่เหมือนซะทีเดียว)

### TypeScript - try/finally

```typescript
async function readFile(path: string): Promise<string> {
    let file: FileHandle | null = null
    try {
        file = await fs.open(path, 'r')
        return await file.readFile('utf-8')
    } finally {
        if (file) {
            await file.close()  // ปิดไฟล์เสมอ
        }
    }
}
```

### Go - defer

```go
func readFile(path string) (string, error) {
    file, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer file.Close()  // จะ Close ตอนจบ function

    data, err := io.ReadAll(file)
    if err != nil {
        return "", err  // file.Close() ยังทำงาน!
    }
    return string(data), nil
}
```

### defer หลายตัว = LIFO (Stack)

```go
func example() {
    defer fmt.Println("1")
    defer fmt.Println("2")
    defer fmt.Println("3")
    fmt.Println("Start")
}
// Output:
// Start
// 3
// 2
// 1

// Real use case: cleanup resources
func transaction() error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()  // จะ rollback ถ้า function จบก่อน commit

    // ... do something ...

    if err := tx.Commit(); err != nil {
        return err  // Rollback จะถูกเรียก
    }
    return nil  // Rollback ถูกเรียกแต่ไม่ทำอะไรหลัง commit
}
```

### defer argument evaluation

```go
// Arguments ถูก evaluate ตอน defer ไม่ใช่ตอนรัน
func example() {
    i := 0
    defer fmt.Println(i)  // i=0 ณ ตอนนี้
    i++
    fmt.Println(i)
}
// Output:
// 1
// 0  (ไม่ใช่ 1!)

// ถ้าอยากได้ค่าตอนรัน ใช้ closure
func example2() {
    i := 0
    defer func() {
        fmt.Println(i)  // เข้าถึง i ตอนรัน
    }()
    i++
    fmt.Println(i)
}
// Output:
// 1
// 1
```

---

## Methods (เมธอด)

Method = function ที่ผูกกับ type (เหมือน class method ใน TS)

### TypeScript - Class methods

```typescript
class User {
    constructor(
        public name: string,
        private age: number
    ) {}

    greet(): string {
        return `Hello, ${this.name}`
    }

    setAge(age: number): void {
        this.age = age
    }
}
```

### Go - Struct with methods

```go
type User struct {
    Name string
    age  int  // lowercase = private
}

// Method with receiver
// (u User) = value receiver
func (u User) Greet() string {
    return "Hello, " + u.Name
}

// (u *User) = pointer receiver (สามารถแก้ไขได้)
func (u *User) SetAge(age int) {
    u.age = age  // แก้ไข original
}

// ใช้งาน
user := User{Name: "John", age: 25}
fmt.Println(user.Greet())  // "Hello, John"
user.SetAge(26)            // Go แปลง &user ให้อัตโนมัติ
```

### Value vs Pointer Receiver

```go
// Value receiver - รับ copy (แก้ไขไม่ได้)
func (u User) UpdateNameValue(name string) {
    u.Name = name  // แก้ไข copy เท่านั้น!
}

// Pointer receiver - รับ pointer (แก้ไขได้)
func (u *User) UpdateName(name string) {
    u.Name = name  // แก้ไข original
}

user := User{Name: "John"}

user.UpdateNameValue("Jane")
fmt.Println(user.Name)  // "John" (ไม่เปลี่ยน!)

user.UpdateName("Jane")
fmt.Println(user.Name)  // "Jane" (เปลี่ยน!)
```

### เมื่อไหร่ใช้ Pointer Receiver?

| สถานการณ์ | Value `(t T)` | Pointer `(t *T)` |
|-----------|---------------|------------------|
| ต้องแก้ไข struct | ❌ | ✅ |
| Struct ใหญ่ (หลีกเลี่ยง copy) | ❌ | ✅ |
| Consistency (receiver type เดียวกัน) | - | ✅ แนะนำ |
| Struct เล็ก + read-only | ✅ | - |

**Rule of thumb:** ถ้า struct มี pointer receiver method ใดก็ตาม → ทุก method ควรเป็น pointer receiver

---

## ตัวอย่างจาก Booking Rush

```go
// backend-auth/internal/service/auth_service.go

type AuthService struct {
    userRepo   repository.UserRepository
    jwtSecret  string
    tokenTTL   time.Duration
}

// Constructor function (ไม่มี constructor แบบ class)
func NewAuthService(
    userRepo repository.UserRepository,
    jwtSecret string,
    tokenTTL time.Duration,
) *AuthService {
    return &AuthService{
        userRepo:  userRepo,
        jwtSecret: jwtSecret,
        tokenTTL:  tokenTTL,
    }
}

// Methods with pointer receiver
func (s *AuthService) Login(ctx context.Context, email, password string) (*dto.TokenResponse, error) {
    // Get user
    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return nil, fmt.Errorf("invalid credentials")
    }

    // Verify password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
        return nil, fmt.Errorf("invalid credentials")
    }

    // Generate token
    token, err := s.generateToken(user)
    if err != nil {
        return nil, fmt.Errorf("failed to generate token: %w", err)
    }

    return &dto.TokenResponse{
        AccessToken: token,
        ExpiresIn:   int64(s.tokenTTL.Seconds()),
    }, nil
}

// Private method (lowercase)
func (s *AuthService) generateToken(user *domain.User) (string, error) {
    claims := jwt.MapClaims{
        "sub":       user.ID,
        "email":     user.Email,
        "role":      user.Role,
        "tenant_id": user.TenantID,
        "exp":       time.Now().Add(s.tokenTTL).Unix(),
        "iat":       time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.jwtSecret))
}
```

---

## สรุป

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| ประกาศ function | `function name()` | `func name()` |
| Arrow function | `() => {}` | ไม่มี (ใช้ anonymous) |
| Optional params | `param?: type` | ไม่มี (ใช้ pointer/variadic) |
| Default params | `param = value` | ไม่มี (ใช้ pattern) |
| Multiple returns | return object/tuple | `func() (int, error)` |
| Rest params | `...args` | `args ...type` |
| Spread | `...array` | `slice...` |
| Method | `class { method() }` | `func (r *Type) Method()` |
| this | `this` | receiver `(r *Type)` |
| Cleanup | `try/finally` | `defer` |

---

## ต่อไป

- [03-structs-interfaces.md](./03-structs-interfaces.md) - Struct และ Interface
