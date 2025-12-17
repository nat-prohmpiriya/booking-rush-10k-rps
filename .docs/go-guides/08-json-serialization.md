# 08 - JSON Serialization

## สารบัญ

1. [พื้นฐาน JSON](#พื้นฐาน-json)
2. [Struct Tags](#struct-tags)
3. [Marshal (Encode)](#marshal-encode)
4. [Unmarshal (Decode)](#unmarshal-decode)
5. [Custom JSON](#custom-json)
6. [Working with Dynamic JSON](#working-with-dynamic-json)
7. [Performance Tips](#performance-tips)

---

## พื้นฐาน JSON

### TypeScript

```typescript
// Parse JSON string → object
const user = JSON.parse('{"name": "John", "age": 25}')

// Object → JSON string
const json = JSON.stringify(user)

// With formatting
const pretty = JSON.stringify(user, null, 2)
```

### Go

```go
import "encoding/json"

// Parse JSON string → struct
var user User
err := json.Unmarshal([]byte(`{"name": "John", "age": 25}`), &user)

// Struct → JSON string
data, err := json.Marshal(user)

// With formatting
pretty, err := json.MarshalIndent(user, "", "  ")
```

---

## Struct Tags

### กำหนดชื่อ Field ใน JSON

```go
type User struct {
    ID        string    `json:"id"`           // field name: "id"
    FirstName string    `json:"first_name"`   // snake_case
    LastName  string    `json:"lastName"`     // camelCase
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// JSON output:
// {
//   "id": "123",
//   "first_name": "John",
//   "lastName": "Doe",
//   "email": "john@example.com",
//   "created_at": "2024-01-15T10:30:00Z"
// }
```

### Tag Options

```go
type User struct {
    // ไม่ส่งใน JSON
    Password string `json:"-"`

    // ไม่ส่งถ้าเป็น zero value
    Nickname string `json:"nickname,omitempty"`

    // ส่งเป็น string แทน number
    ID int64 `json:"id,string"`

    // Field name เดิม + omitempty
    Status string `json:"status,omitempty"`
}

// Input struct
user := User{
    Password: "secret",    // จะไม่ส่ง
    Nickname: "",          // จะไม่ส่ง (omitempty + zero value)
    ID:       123,         // ส่งเป็น "123"
    Status:   "active",    // ส่งปกติ
}

// Output:
// {"id": "123", "status": "active"}
```

### Embedded Struct

```go
type BaseModel struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
    BaseModel          // embedded - fields ถูก flatten
    Name      string   `json:"name"`
    Email     string   `json:"email"`
}

// Output:
// {
//   "id": "123",
//   "created_at": "...",
//   "updated_at": "...",
//   "name": "John",
//   "email": "john@example.com"
// }
```

---

## Marshal (Encode)

### พื้นฐาน

```go
user := User{
    ID:    "123",
    Name:  "John",
    Email: "john@example.com",
}

// Struct → []byte
data, err := json.Marshal(user)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))

// Struct → io.Writer (HTTP response, file)
json.NewEncoder(w).Encode(user)

// Pretty print
pretty, _ := json.MarshalIndent(user, "", "  ")
fmt.Println(string(pretty))
```

### Marshal Slice และ Map

```go
// Slice of structs
users := []User{
    {ID: "1", Name: "Alice"},
    {ID: "2", Name: "Bob"},
}
data, _ := json.Marshal(users)
// [{"id":"1","name":"Alice"},{"id":"2","name":"Bob"}]

// Map
scores := map[string]int{
    "alice": 100,
    "bob":   85,
}
data, _ := json.Marshal(scores)
// {"alice":100,"bob":85}
```

### Pointer Fields

```go
type UpdateRequest struct {
    Name  *string `json:"name,omitempty"`
    Email *string `json:"email,omitempty"`
}

// nil pointer = ไม่ส่ง (ถ้ามี omitempty)
req := UpdateRequest{
    Name: nil,
}
data, _ := json.Marshal(req)
// {}

// มีค่า
name := "John"
req := UpdateRequest{
    Name: &name,
}
data, _ := json.Marshal(req)
// {"name":"John"}
```

---

## Unmarshal (Decode)

### พื้นฐาน

```go
jsonStr := `{"id": "123", "name": "John", "email": "john@example.com"}`

var user User
err := json.Unmarshal([]byte(jsonStr), &user)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%+v\n", user)

// จาก io.Reader (HTTP request body)
var user User
err := json.NewDecoder(r.Body).Decode(&user)
```

### Unknown Fields

```go
// Default: ignore unknown fields
jsonStr := `{"id": "123", "name": "John", "unknown_field": "ignored"}`
var user User
json.Unmarshal([]byte(jsonStr), &user)  // OK, unknown_field ถูกข้าม

// Strict mode: error on unknown fields
decoder := json.NewDecoder(strings.NewReader(jsonStr))
decoder.DisallowUnknownFields()
err := decoder.Decode(&user)  // Error!
```

### Missing Fields

```go
// Missing fields = zero value
jsonStr := `{"id": "123"}`  // ไม่มี name, email

var user User
json.Unmarshal([]byte(jsonStr), &user)
fmt.Println(user.Name)   // "" (zero value)
fmt.Println(user.Email)  // "" (zero value)

// ใช้ pointer เพื่อแยก nil vs empty string
type User struct {
    ID    string  `json:"id"`
    Name  *string `json:"name"`
    Email *string `json:"email"`
}

var user User
json.Unmarshal([]byte(jsonStr), &user)
fmt.Println(user.Name == nil)  // true (ไม่ได้ส่งมา)

// vs
jsonStr := `{"id": "123", "name": ""}`
json.Unmarshal([]byte(jsonStr), &user)
fmt.Println(user.Name == nil)  // false
fmt.Println(*user.Name)        // "" (ส่งมาเป็น empty)
```

---

## Custom JSON

### Custom Marshal/Unmarshal

```go
import "time"

// Custom type for date only (no time)
type Date time.Time

func (d Date) MarshalJSON() ([]byte, error) {
    t := time.Time(d)
    return []byte(`"` + t.Format("2006-01-02") + `"`), nil
}

func (d *Date) UnmarshalJSON(data []byte) error {
    // Remove quotes
    str := string(data)
    str = str[1 : len(str)-1]

    t, err := time.Parse("2006-01-02", str)
    if err != nil {
        return err
    }
    *d = Date(t)
    return nil
}

type Event struct {
    Name string `json:"name"`
    Date Date   `json:"date"`
}

// Marshal: {"name": "Concert", "date": "2024-12-25"}
// Unmarshal: Date เป็น time.Time ที่ parse จาก "2024-12-25"
```

### Enum ใน JSON

```go
type Status int

const (
    StatusPending Status = iota
    StatusActive
    StatusCompleted
)

func (s Status) MarshalJSON() ([]byte, error) {
    var str string
    switch s {
    case StatusPending:
        str = "pending"
    case StatusActive:
        str = "active"
    case StatusCompleted:
        str = "completed"
    default:
        str = "unknown"
    }
    return json.Marshal(str)
}

func (s *Status) UnmarshalJSON(data []byte) error {
    var str string
    if err := json.Unmarshal(data, &str); err != nil {
        return err
    }

    switch str {
    case "pending":
        *s = StatusPending
    case "active":
        *s = StatusActive
    case "completed":
        *s = StatusCompleted
    default:
        return fmt.Errorf("unknown status: %s", str)
    }
    return nil
}

type Task struct {
    Name   string `json:"name"`
    Status Status `json:"status"`
}

// {"name": "Task 1", "status": "active"}
```

### Money/Decimal

```go
// เก็บเงินเป็น satang (int64) แต่แสดงเป็น baht (float)
type Money int64

func (m Money) MarshalJSON() ([]byte, error) {
    baht := float64(m) / 100
    return json.Marshal(baht)
}

func (m *Money) UnmarshalJSON(data []byte) error {
    var baht float64
    if err := json.Unmarshal(data, &baht); err != nil {
        return err
    }
    *m = Money(baht * 100)
    return nil
}

type Product struct {
    Name  string `json:"name"`
    Price Money  `json:"price"`
}

// {"name": "Coffee", "price": 55.00}
// แต่เก็บเป็น 5500 satang ใน struct
```

---

## Working with Dynamic JSON

### TypeScript - any object

```typescript
const data = JSON.parse(jsonString)
const name = data.user?.name  // optional chaining
```

### Go - map[string]interface{} หรือ interface{}

```go
// Unknown structure
var data map[string]interface{}
json.Unmarshal([]byte(jsonStr), &data)

// Access with type assertion
if user, ok := data["user"].(map[string]interface{}); ok {
    if name, ok := user["name"].(string); ok {
        fmt.Println(name)
    }
}

// Nested access (verbose)
func getNestedString(data map[string]interface{}, keys ...string) (string, bool) {
    var current interface{} = data
    for _, key := range keys[:len(keys)-1] {
        if m, ok := current.(map[string]interface{}); ok {
            current = m[key]
        } else {
            return "", false
        }
    }
    if m, ok := current.(map[string]interface{}); ok {
        if val, ok := m[keys[len(keys)-1]].(string); ok {
            return val, true
        }
    }
    return "", false
}

name, ok := getNestedString(data, "user", "name")
```

### json.RawMessage - Delay Parsing

```go
// เก็บ JSON ไว้ก่อน parse ทีหลัง
type Event struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`  // raw JSON
}

// Parse
var event Event
json.Unmarshal(data, &event)

// Parse payload ตาม type
switch event.Type {
case "user_created":
    var payload UserCreatedPayload
    json.Unmarshal(event.Payload, &payload)
case "order_placed":
    var payload OrderPlacedPayload
    json.Unmarshal(event.Payload, &payload)
}
```

### gjson Library (Third-party)

```go
import "github.com/tidwall/gjson"

jsonStr := `{"user": {"name": "John", "age": 25}}`

// Simple access
name := gjson.Get(jsonStr, "user.name").String()  // "John"
age := gjson.Get(jsonStr, "user.age").Int()       // 25

// Array access
jsonStr := `{"users": [{"name": "Alice"}, {"name": "Bob"}]}`
gjson.Get(jsonStr, "users.0.name").String()  // "Alice"
gjson.Get(jsonStr, "users.#").Int()          // 2 (length)

// Query
gjson.Get(jsonStr, `users.#(name=="Bob").name`).String()  // "Bob"
```

---

## Performance Tips

### 1. ใช้ json.NewEncoder/Decoder กับ streams

```go
// ❌ อ่านทั้งหมดก่อน
body, _ := io.ReadAll(r.Body)
json.Unmarshal(body, &data)

// ✅ Stream decode
json.NewDecoder(r.Body).Decode(&data)

// ✅ Stream encode
json.NewEncoder(w).Encode(data)
```

### 2. Pre-allocate slices

```go
// ❌ Grow dynamically
var users []User
json.Unmarshal(data, &users)

// ✅ Pre-allocate ถ้ารู้ขนาด
type Response struct {
    Users []User `json:"users"`
    Total int    `json:"total"`
}

// Parse total first, allocate, then parse users
```

### 3. Use json.Number for large integers

```go
// JavaScript Number max safe integer: 9007199254740991
// Go int64 max: 9223372036854775807

// ❌ อาจสูญเสีย precision
var data map[string]interface{}
json.Unmarshal([]byte(`{"id": 9223372036854775807}`), &data)
// data["id"] เป็น float64, อาจผิด!

// ✅ ใช้ json.Number
decoder := json.NewDecoder(strings.NewReader(jsonStr))
decoder.UseNumber()
decoder.Decode(&data)
id, _ := data["id"].(json.Number).Int64()

// ✅ หรือกำหนด type ชัดเจน
type Data struct {
    ID int64 `json:"id"`
}
```

### 4. ใช้ jsoniter สำหรับ performance

```go
import jsoniter "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// ใช้แทน encoding/json ได้เลย
json.Marshal(data)
json.Unmarshal(bytes, &data)

// เร็วกว่า 3-5x
```

---

## ตัวอย่างจาก Booking Rush

```go
// backend-booking/internal/dto/booking.go

type ReserveRequest struct {
    EventID   string `json:"event_id" binding:"required,uuid"`
    ZoneID    string `json:"zone_id" binding:"required,uuid"`
    ShowID    string `json:"show_id" binding:"required,uuid"`
    Quantity  int    `json:"quantity" binding:"required,min=1,max=10"`
    UnitPrice int64  `json:"unit_price" binding:"required,gt=0"`
}

type ReserveResponse struct {
    BookingID   string    `json:"booking_id"`
    TotalAmount int64     `json:"total_amount"`
    ExpiresAt   time.Time `json:"expires_at"`
}

// backend-booking/internal/domain/booking.go

type Booking struct {
    ID          string        `json:"id" db:"id"`
    UserID      string        `json:"user_id" db:"user_id"`
    EventID     string        `json:"event_id" db:"event_id"`
    ZoneID      string        `json:"zone_id" db:"zone_id"`
    ShowID      string        `json:"show_id" db:"show_id"`
    Quantity    int           `json:"quantity" db:"quantity"`
    UnitPrice   int64         `json:"unit_price" db:"unit_price"`
    TotalAmount int64         `json:"total_amount" db:"total_amount"`
    Status      BookingStatus `json:"status" db:"status"`
    ExpiresAt   time.Time     `json:"expires_at" db:"expires_at"`
    ConfirmedAt *time.Time    `json:"confirmed_at,omitempty" db:"confirmed_at"`
    CreatedAt   time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
}

type BookingStatus string

const (
    BookingStatusPending   BookingStatus = "pending"
    BookingStatusConfirmed BookingStatus = "confirmed"
    BookingStatusCancelled BookingStatus = "cancelled"
    BookingStatusExpired   BookingStatus = "expired"
)

// pkg/response/response.go

type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func Success(c *gin.Context, data interface{}) {
    c.JSON(200, Response{
        Success: true,
        Data:    data,
    })
}

func Error(c *gin.Context, status int, code, message string) {
    c.JSON(status, Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    code,
            Message: message,
        },
    })
}
```

---

## สรุป

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| Parse JSON | `JSON.parse(str)` | `json.Unmarshal(bytes, &v)` |
| Stringify | `JSON.stringify(obj)` | `json.Marshal(v)` |
| Field name | - | `json:"name"` tag |
| Omit field | - | `json:"-"` |
| Omit empty | - | `json:",omitempty"` |
| Dynamic JSON | `any` | `map[string]interface{}` |
| Stream | - | `json.NewEncoder/Decoder` |
| Custom format | - | `MarshalJSON/UnmarshalJSON` |

### JSON Tag Cheatsheet

```go
`json:"name"`              // field name
`json:"-"`                 // never include
`json:",omitempty"`        // omit if zero value
`json:"name,omitempty"`    // name + omit if zero
`json:",string"`           // number as string
`json:"name,string"`       // name + number as string
```

---

## ต่อไป

- [09-modules-packages.md](./09-modules-packages.md) - Modules และ Packages
