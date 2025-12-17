# 03 - Structs และ Interfaces

## สารบัญ

1. [Structs](#structs-โครงสร้างข้อมูล)
2. [Struct Tags](#struct-tags-แท็ก)
3. [Interfaces](#interfaces-อินเตอร์เฟซ)
4. [Embedding](#embedding-การฝังตัว)
5. [Type Aliases และ Custom Types](#type-aliases-และ-custom-types)
6. [Composition vs Inheritance](#composition-vs-inheritance)

---

## Structs (โครงสร้างข้อมูล)

Struct ใน Go = Class ใน TypeScript แต่ไม่มี inheritance

### TypeScript - Class/Interface

```typescript
// Interface
interface User {
    id: string
    name: string
    email: string
    createdAt: Date
}

// Class
class User {
    constructor(
        public id: string,
        public name: string,
        public email: string,
        public createdAt: Date = new Date()
    ) {}
}

// สร้าง instance
const user = new User("1", "John", "john@example.com")
```

### Go - Struct

```go
// Struct definition
type User struct {
    ID        string
    Name      string
    Email     string
    CreatedAt time.Time
}

// สร้าง instance - หลายวิธี

// วิธี 1: Struct literal (แนะนำ)
user := User{
    ID:        "1",
    Name:      "John",
    Email:     "john@example.com",
    CreatedAt: time.Now(),
}

// วิธี 2: ไม่ระบุ field names (ไม่แนะนำ - ต้องครบทุก field ตามลำดับ)
user := User{"1", "John", "john@example.com", time.Now()}

// วิธี 3: Zero value แล้วค่อย assign
var user User
user.ID = "1"
user.Name = "John"

// วิธี 4: new() - ได้ pointer
userPtr := new(User)  // *User ชี้ไปยัง User{}
userPtr.ID = "1"

// วิธี 5: &Type{} - ได้ pointer (ใช้บ่อยสุด)
userPtr := &User{
    ID:   "1",
    Name: "John",
}
```

### Constructor Pattern

Go ไม่มี constructor - ใช้ factory function แทน

```typescript
// TypeScript
class UserService {
    constructor(
        private readonly userRepo: UserRepository,
        private readonly logger: Logger
    ) {}
}

const service = new UserService(userRepo, logger)
```

```go
// Go - Factory function (convention: NewXxx)
type UserService struct {
    userRepo UserRepository
    logger   Logger
}

func NewUserService(userRepo UserRepository, logger Logger) *UserService {
    return &UserService{
        userRepo: userRepo,
        logger:   logger,
    }
}

// ใช้งาน
service := NewUserService(userRepo, logger)
```

### Access Modifiers (Public/Private)

```typescript
// TypeScript - explicit modifiers
class User {
    public name: string      // public
    private password: string // private
    protected age: number    // protected
}
```

```go
// Go - ใช้ตัวอักษรแรก
type User struct {
    Name     string // Public (ขึ้นต้นตัวใหญ่)
    password string // private (ขึ้นต้นตัวเล็ก)
}

// private = ใช้ได้เฉพาะใน package เดียวกัน
// Go ไม่มี protected
```

### Anonymous Struct

```go
// สร้าง struct แบบไม่ต้องประกาศ type
person := struct {
    Name string
    Age  int
}{
    Name: "John",
    Age:  25,
}

// ใช้บ่อยใน test หรือ temporary data
config := struct {
    Host string
    Port int
}{
    Host: "localhost",
    Port: 8080,
}
```

---

## Struct Tags (แท็ก)

Struct tags = metadata บน field ใช้กับ JSON, validation, ORM

### TypeScript - Decorators

```typescript
class User {
    @Column('varchar')
    id: string

    @Column('varchar')
    name: string

    @Column({ select: false })  // ไม่ดึงมา
    password: string
}
```

### Go - Struct Tags

```go
type User struct {
    ID       string `json:"id" db:"id"`
    Name     string `json:"name" db:"name"`
    Email    string `json:"email" db:"email" validate:"required,email"`
    Password string `json:"-" db:"password"`  // - = ไม่ส่งใน JSON
    Age      int    `json:"age,omitempty"`    // omitempty = ไม่ส่งถ้าเป็น zero value
}

// อ่าน tag ด้วย reflect (ปกติไม่ต้องทำเอง - library ทำให้)
import "reflect"

t := reflect.TypeOf(User{})
field, _ := t.FieldByName("Email")
jsonTag := field.Tag.Get("json")     // "email"
validateTag := field.Tag.Get("validate")  // "required,email"
```

### Common Tags

| Tag | Library | ตัวอย่าง |
|-----|---------|----------|
| `json` | encoding/json | `json:"name,omitempty"` |
| `db` | sqlx | `db:"user_name"` |
| `gorm` | GORM | `gorm:"primaryKey"` |
| `validate` | go-playground/validator | `validate:"required,email"` |
| `binding` | Gin | `binding:"required"` |
| `yaml` | gopkg.in/yaml | `yaml:"server_port"` |
| `env` | caarlos0/env | `env:"DATABASE_URL"` |

### JSON Tag Options

```go
type Response struct {
    // ชื่อ field ใน JSON
    Name string `json:"name"`

    // ไม่ส่งถ้าเป็น zero value
    Age int `json:"age,omitempty"`

    // ไม่ส่งเลย (sensitive data)
    Password string `json:"-"`

    // ส่งเป็น string แทน number
    ID int64 `json:"id,string"`

    // ชื่อเดิมใน Go แต่ไม่ omit
    Status string `json:"Status"`
}

// omitempty กับ pointer - nil = ไม่ส่ง
type UpdateRequest struct {
    Name  *string `json:"name,omitempty"`  // ถ้า nil ไม่ส่ง
    Email *string `json:"email,omitempty"` // ถ้า nil ไม่ส่ง
}
```

---

## Interfaces (อินเตอร์เฟซ)

### TypeScript - Explicit implements

```typescript
// Interface definition
interface Logger {
    log(message: string): void
    error(message: string): void
}

// Explicit implements
class ConsoleLogger implements Logger {
    log(message: string): void {
        console.log(message)
    }
    error(message: string): void {
        console.error(message)
    }
}

// ใช้งาน
function doSomething(logger: Logger) {
    logger.log("Starting...")
}
```

### Go - Implicit implements (Duck Typing)

```go
// Interface definition
type Logger interface {
    Log(message string)
    Error(message string)
}

// ไม่ต้องระบุ implements - ถ้ามี method ครบก็ implement อัตโนมัติ
type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(message string) {
    fmt.Println(message)
}

func (c *ConsoleLogger) Error(message string) {
    fmt.Fprintln(os.Stderr, message)
}

// ConsoleLogger implements Logger โดยอัตโนมัติ!

// ใช้งาน
func doSomething(logger Logger) {
    logger.Log("Starting...")
}

// ตรวจสอบว่า implement ถูกต้อง (compile time)
var _ Logger = (*ConsoleLogger)(nil)
```

### ข้อดีของ Implicit Interface

```go
// สามารถสร้าง interface สำหรับ type ที่เราไม่ได้เป็นเจ้าของ
// เช่น standard library หรือ third-party

// สมมติ os.File มี method Read และ Close
type ReadCloser interface {
    Read(p []byte) (n int, err error)
    Close() error
}

// os.File implements ReadCloser โดยอัตโนมัติ!
func processFile(rc ReadCloser) {
    // ...
}

file, _ := os.Open("data.txt")
processFile(file)  // ใช้ได้เลย
```

### Empty Interface `interface{}` / `any`

```go
// interface{} = any type (เหมือน any ใน TS)
var anything interface{}
anything = 42
anything = "hello"
anything = User{Name: "John"}

// Go 1.18+ มี alias
var anything any  // = interface{}

// ใช้ใน function ที่รับ any type
func printAnything(v interface{}) {
    fmt.Println(v)
}

// ต้อง type assert เพื่อใช้งานจริง
func process(v interface{}) {
    switch val := v.(type) {
    case string:
        fmt.Println("String:", val)
    case int:
        fmt.Println("Int:", val)
    case User:
        fmt.Println("User:", val.Name)
    default:
        fmt.Println("Unknown type")
    }
}
```

### Interface Composition

```go
// ประกอบ interface จากหลาย interface
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

// Compose interfaces
type ReadWriter interface {
    Reader
    Writer
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}

// ใช้ใน standard library
// io.ReadWriter, io.ReadCloser, io.ReadWriteCloser
```

### Common Interface Patterns

```go
// 1. Service interface (for dependency injection)
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}

// Implementation
type PostgresUserRepository struct {
    db *sql.DB
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *User) error {
    // ... PostgreSQL implementation
}

// Mock for testing
type MockUserRepository struct {
    users map[string]*User
}

func (r *MockUserRepository) Create(ctx context.Context, user *User) error {
    r.users[user.ID] = user
    return nil
}
```

---

## Embedding (การฝังตัว)

Go ไม่มี inheritance - ใช้ embedding แทน

### TypeScript - Inheritance

```typescript
class Animal {
    constructor(public name: string) {}

    move(): void {
        console.log(`${this.name} is moving`)
    }
}

class Dog extends Animal {
    bark(): void {
        console.log("Woof!")
    }
}

const dog = new Dog("Buddy")
dog.move()  // inherited
dog.bark()  // own method
```

### Go - Struct Embedding

```go
type Animal struct {
    Name string
}

func (a *Animal) Move() {
    fmt.Printf("%s is moving\n", a.Name)
}

// Embed Animal ใน Dog
type Dog struct {
    Animal  // embedded (ไม่ต้องตั้งชื่อ field)
    Breed string
}

func (d *Dog) Bark() {
    fmt.Println("Woof!")
}

// ใช้งาน
dog := Dog{
    Animal: Animal{Name: "Buddy"},
    Breed:  "Golden Retriever",
}

dog.Move()  // เรียก Animal.Move() ได้เลย
dog.Bark()  // method ของ Dog
dog.Name    // เข้าถึง Name ได้เลย (promoted field)

// หรือระบุชัดเจน
dog.Animal.Move()
dog.Animal.Name
```

### Interface Embedding

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// Embed interfaces
type ReadWriter interface {
    Reader
    Writer
}

// type ที่ implement ReadWriter ต้องมีทั้ง Read และ Write
```

### Multiple Embedding

```go
type Base struct {
    ID        string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Auditable struct {
    CreatedBy string
    UpdatedBy string
}

type User struct {
    Base      // embed Base
    Auditable // embed Auditable
    Name      string
    Email     string
}

// ใช้งาน
user := User{
    Base: Base{
        ID:        "1",
        CreatedAt: time.Now(),
    },
    Auditable: Auditable{
        CreatedBy: "admin",
    },
    Name:  "John",
    Email: "john@example.com",
}

fmt.Println(user.ID)        // promoted from Base
fmt.Println(user.CreatedBy) // promoted from Auditable
```

---

## Type Aliases และ Custom Types

### Type Alias

```go
// Type alias - สร้างชื่อใหม่ให้ type เดิม
type UserID = string     // UserID และ string เป็น type เดียวกัน
type Timestamp = int64   // Timestamp และ int64 เป็น type เดียวกัน

var id UserID = "user-123"
var s string = id  // OK - เป็น type เดียวกัน
```

### Custom Type (Type Definition)

```go
// Custom type - สร้าง type ใหม่จาก type อื่น
type UserID string      // UserID เป็น type ใหม่ (underlying type = string)
type Amount int64       // Amount เป็น type ใหม่ (underlying type = int64)

var id UserID = "user-123"
var s string = id           // Error! ต้อง convert
var s string = string(id)   // OK

// เพิ่ม method ให้ custom type ได้
func (id UserID) Validate() error {
    if id == "" {
        return errors.New("user ID cannot be empty")
    }
    return nil
}

func (a Amount) ToBaht() float64 {
    return float64(a) / 100
}

// ใช้งาน
id := UserID("user-123")
if err := id.Validate(); err != nil {
    // handle error
}

price := Amount(15000)  // 150.00 baht (satang)
fmt.Printf("%.2f baht\n", price.ToBaht())
```

### Enum Pattern

Go ไม่มี enum - ใช้ const + custom type

```typescript
// TypeScript
enum Status {
    Pending = "pending",
    Active = "active",
    Completed = "completed",
}
```

```go
// Go - Custom type + const
type Status string

const (
    StatusPending   Status = "pending"
    StatusActive    Status = "active"
    StatusCompleted Status = "completed"
)

// Validate method
func (s Status) IsValid() bool {
    switch s {
    case StatusPending, StatusActive, StatusCompleted:
        return true
    }
    return false
}

// String method (สำหรับ fmt.Print)
func (s Status) String() string {
    return string(s)
}

// ใช้งาน
status := StatusPending
if !status.IsValid() {
    // handle invalid status
}
```

```go
// iota สำหรับ numeric enum
type Role int

const (
    RoleGuest Role = iota  // 0
    RoleUser               // 1
    RoleAdmin              // 2
    RoleSuperAdmin         // 3
)

func (r Role) String() string {
    switch r {
    case RoleGuest:
        return "guest"
    case RoleUser:
        return "user"
    case RoleAdmin:
        return "admin"
    case RoleSuperAdmin:
        return "super_admin"
    }
    return "unknown"
}
```

---

## Composition vs Inheritance

### TypeScript - Inheritance

```typescript
// Base class
class BaseRepository<T> {
    constructor(protected db: Database) {}

    async findById(id: string): Promise<T | null> {
        return this.db.findOne(id)
    }

    async create(entity: T): Promise<T> {
        return this.db.insert(entity)
    }
}

// Inherit
class UserRepository extends BaseRepository<User> {
    async findByEmail(email: string): Promise<User | null> {
        return this.db.findOne({ email })
    }
}
```

### Go - Composition

```go
// Base struct
type BaseRepository struct {
    db *sql.DB
}

func (r *BaseRepository) FindByID(ctx context.Context, table string, id string, dest interface{}) error {
    query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", table)
    return r.db.QueryRowContext(ctx, query, id).Scan(dest)
}

// Compose
type UserRepository struct {
    BaseRepository  // embed
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{
        BaseRepository: BaseRepository{db: db},
    }
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    var user User
    // เรียกใช้ BaseRepository.FindByID ผ่าน embedding
    err := r.BaseRepository.FindByID(ctx, "users", id, &user)
    return &user, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    query := "SELECT * FROM users WHERE email = $1"
    err := r.db.QueryRowContext(ctx, query, email).Scan(&user)
    return &user, err
}
```

### Interface-based Composition (แนะนำ)

```go
// Define interfaces
type Reader interface {
    Read(ctx context.Context, id string) (*Entity, error)
}

type Writer interface {
    Create(ctx context.Context, entity *Entity) error
    Update(ctx context.Context, entity *Entity) error
}

type Deleter interface {
    Delete(ctx context.Context, id string) error
}

// Compose interfaces
type Repository interface {
    Reader
    Writer
    Deleter
}

// Implementation
type UserRepository struct {
    db *sql.DB
}

// Implement ทุก method
func (r *UserRepository) Read(ctx context.Context, id string) (*Entity, error) { ... }
func (r *UserRepository) Create(ctx context.Context, entity *Entity) error { ... }
func (r *UserRepository) Update(ctx context.Context, entity *Entity) error { ... }
func (r *UserRepository) Delete(ctx context.Context, id string) error { ... }

// ใช้ interface ที่เหมาะสมกับ use case
func readOnlyOperation(r Reader) { ... }
func writeOperation(w Writer) { ... }
func fullAccess(r Repository) { ... }
```

---

## ตัวอย่างจาก Booking Rush

```go
// backend-booking/internal/domain/booking.go
type Booking struct {
    ID          string          `json:"id" db:"id"`
    UserID      string          `json:"user_id" db:"user_id"`
    EventID     string          `json:"event_id" db:"event_id"`
    ZoneID      string          `json:"zone_id" db:"zone_id"`
    ShowID      string          `json:"show_id" db:"show_id"`
    Quantity    int             `json:"quantity" db:"quantity"`
    TotalAmount int64           `json:"total_amount" db:"total_amount"`
    Status      BookingStatus   `json:"status" db:"status"`
    ExpiresAt   time.Time       `json:"expires_at" db:"expires_at"`
    CreatedAt   time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

type BookingStatus string

const (
    BookingStatusPending   BookingStatus = "pending"
    BookingStatusConfirmed BookingStatus = "confirmed"
    BookingStatusCancelled BookingStatus = "cancelled"
    BookingStatusExpired   BookingStatus = "expired"
)

// backend-booking/internal/repository/booking_repository.go
type BookingRepository interface {
    Create(ctx context.Context, booking *domain.Booking) error
    FindByID(ctx context.Context, id string) (*domain.Booking, error)
    FindByUserID(ctx context.Context, userID string) ([]*domain.Booking, error)
    UpdateStatus(ctx context.Context, id string, status domain.BookingStatus) error
}

// PostgreSQL implementation
type PostgresBookingRepository struct {
    db *sqlx.DB
}

func NewPostgresBookingRepository(db *sqlx.DB) *PostgresBookingRepository {
    return &PostgresBookingRepository{db: db}
}

func (r *PostgresBookingRepository) Create(ctx context.Context, booking *domain.Booking) error {
    query := `
        INSERT INTO bookings (id, user_id, event_id, zone_id, show_id, quantity, total_amount, status, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
    _, err := r.db.ExecContext(ctx, query,
        booking.ID, booking.UserID, booking.EventID, booking.ZoneID, booking.ShowID,
        booking.Quantity, booking.TotalAmount, booking.Status, booking.ExpiresAt,
    )
    return err
}
```

---

## สรุป

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| Class | `class User {}` | `type User struct {}` |
| Constructor | `constructor()` | `NewXxx()` function |
| Public | `public` | ขึ้นต้นตัวใหญ่ |
| Private | `private` | ขึ้นต้นตัวเล็ก |
| Interface | `implements` explicit | implicit (duck typing) |
| Inheritance | `extends` | embedding |
| Metadata | decorators | struct tags |
| Enum | `enum Status {}` | `type Status string` + const |
| Any type | `any` | `interface{}` หรือ `any` |

---

## ต่อไป

- [04-error-handling.md](./04-error-handling.md) - Error Handling (การจัดการข้อผิดพลาด)
