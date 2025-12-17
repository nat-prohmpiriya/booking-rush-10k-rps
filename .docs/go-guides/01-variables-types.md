# 01 - ตัวแปรและ Types

## สารบัญ

1. [การประกาศตัวแปร](#การประกาศตัวแปร)
2. [Basic Types](#basic-types-ประเภทพื้นฐาน)
3. [Type Conversion](#type-conversion-แปลงประเภท)
4. [Constants](#constants-ค่าคงที่)
5. [Zero Values](#zero-values-ค่าเริ่มต้น)
6. [Pointers](#pointers-พอยน์เตอร์)

---

## การประกาศตัวแปร

### TypeScript

```typescript
// ใช้ let/const
let name: string = "John"
const age: number = 25
let isActive: boolean = true

// Type inference - ไม่ต้องระบุ type
let city = "Bangkok"  // inferred as string
```

### Go

```go
// แบบที่ 1: var + type (verbose)
var name string = "John"
var age int = 25
var isActive bool = true

// แบบที่ 2: var + type inference
var city = "Bangkok"  // inferred as string

// แบบที่ 3: Short declaration := (ใช้บ่อยสุด!)
name := "John"
age := 25
isActive := true

// ประกาศหลายตัวพร้อมกัน
var (
    firstName string = "John"
    lastName  string = "Doe"
    age       int    = 25
)

// หรือแบบสั้น
firstName, lastName, age := "John", "Doe", 25
```

### ข้อแตกต่างสำคัญ

| TypeScript | Go | หมายเหตุ |
|------------|-----|----------|
| `let x = 1` | `x := 1` | `:=` ใช้ได้ใน function เท่านั้น |
| `const x = 1` | `const x = 1` หรือ `x := 1` | Go const ใช้กับ compile-time values เท่านั้น |
| `let x: number` | `var x int` | Go ต้องมีค่าเสมอ (zero value) |

### Short Declaration `:=` Rules

```go
// ใช้ได้ใน function เท่านั้น
func main() {
    name := "John"  // OK
}

// ใช้นอก function ไม่ได้!
name := "John"  // Error!

// ต้องใช้ var แทน
var name = "John"  // OK (package level)
```

---

## Basic Types (ประเภทพื้นฐาน)

### Numbers (ตัวเลข)

```typescript
// TypeScript - มีแค่ number
let count: number = 42
let price: number = 99.99
let big: bigint = 9007199254740991n
```

```go
// Go - แยกประเภทชัดเจน

// Integers (จำนวนเต็ม)
var i int = 42         // ขนาดขึ้นกับ platform (32 หรือ 64 bit)
var i8 int8 = 127      // -128 to 127
var i16 int16 = 32767  // -32768 to 32767
var i32 int32 = 2147483647
var i64 int64 = 9223372036854775807

// Unsigned integers (จำนวนเต็มบวก)
var u uint = 42
var u8 uint8 = 255     // 0 to 255 (เหมือน byte)
var u16 uint16 = 65535
var u32 uint32 = 4294967295
var u64 uint64 = 18446744073709551615

// Floats (ทศนิยม)
var f32 float32 = 3.14
var f64 float64 = 3.141592653589793  // ใช้บ่อยสุด

// Aliases
var b byte = 255       // = uint8
var r rune = 'ก'       // = int32 (Unicode code point)
```

### เลือก Type ไหนดี?

| สถานการณ์ | แนะนำใช้ | เหตุผล |
|-----------|----------|--------|
| ตัวเลขทั่วไป | `int` | ขนาดเหมาะกับ platform |
| ทศนิยม | `float64` | ความแม่นยำสูงกว่า float32 |
| Loop index | `int` | มาตรฐาน |
| File size, offset | `int64` | รองรับไฟล์ใหญ่ |
| JSON number | `float64` หรือ `int64` | JSON spec |
| เงิน (satang) | `int64` | หลีกเลี่ยง float |

### Strings (ข้อความ)

```typescript
// TypeScript
let name: string = "John"
let greeting: string = `Hello ${name}`
let multiline: string = `
    Line 1
    Line 2
`
```

```go
// Go
name := "John"

// ไม่มี template literal แบบ TS - ใช้ fmt.Sprintf
greeting := fmt.Sprintf("Hello %s", name)

// Raw string literal (ใช้ backtick)
multiline := `
    Line 1
    Line 2
`

// String เป็น immutable - แก้ไขตรงๆ ไม่ได้
name[0] = 'j'  // Error!

// ต้องสร้างใหม่
name = "john"  // OK
```

### Format Specifiers (ตัวระบุรูปแบบ)

```go
name := "John"
age := 25
price := 99.99
active := true

fmt.Sprintf("%s", name)      // "John" - string
fmt.Sprintf("%d", age)       // "25" - integer
fmt.Sprintf("%f", price)     // "99.990000" - float
fmt.Sprintf("%.2f", price)   // "99.99" - float 2 ทศนิยม
fmt.Sprintf("%t", active)    // "true" - boolean
fmt.Sprintf("%v", anything)  // value ใดๆ (default format)
fmt.Sprintf("%+v", struct)   // struct พร้อม field names
fmt.Sprintf("%#v", value)    // Go syntax representation
fmt.Sprintf("%T", value)     // type ของ value
```

### Booleans (บูลีน)

```typescript
// TypeScript
let isActive: boolean = true
let isEmpty: boolean = false
```

```go
// Go - ใช้ bool
isActive := true
isEmpty := false

// Go ไม่มี truthy/falsy แบบ JS
// if "" { }     // Error! ใช้ string เป็น condition ไม่ได้
// if 0 { }      // Error! ใช้ number เป็น condition ไม่ได้

// ต้องเปรียบเทียบชัดเจน
if name != "" { }
if count != 0 { }
if user != nil { }
```

---

## Type Conversion (แปลงประเภท)

### TypeScript - Implicit + Explicit

```typescript
// TypeScript - implicit conversion บางกรณี
let num: number = 42
let str: string = num + ""  // implicit to string

// Explicit conversion
let str2: string = String(num)
let num2: number = parseInt("42")
let float: number = parseFloat("3.14")
```

### Go - Explicit เท่านั้น

```go
// Go ไม่มี implicit conversion - ต้อง explicit เสมอ
var i int = 42
var f float64 = float64(i)    // int → float64
var u uint = uint(f)          // float64 → uint

// String conversions
import "strconv"

// int → string
s := strconv.Itoa(42)           // "42"
s := fmt.Sprintf("%d", 42)      // "42"

// string → int
i, err := strconv.Atoi("42")    // 42, nil
if err != nil {
    // handle error - "42abc" จะ error
}

// string → int64
i64, err := strconv.ParseInt("42", 10, 64)

// string → float64
f, err := strconv.ParseFloat("3.14", 64)

// float → string
s := strconv.FormatFloat(3.14, 'f', 2, 64)  // "3.14"
s := fmt.Sprintf("%.2f", 3.14)              // "3.14"

// bool → string
s := strconv.FormatBool(true)  // "true"

// string → bool
b, err := strconv.ParseBool("true")  // true, nil
```

### Type Assertion (สำหรับ interface)

```go
// เมื่อมี interface{} (any) ต้อง assert type
var data interface{} = "hello"

// Type assertion
str := data.(string)  // ถ้าไม่ใช่ string จะ panic!

// Safe type assertion (comma ok idiom)
str, ok := data.(string)
if ok {
    fmt.Println("It's a string:", str)
} else {
    fmt.Println("Not a string")
}

// Type switch
switch v := data.(type) {
case string:
    fmt.Println("string:", v)
case int:
    fmt.Println("int:", v)
default:
    fmt.Println("unknown type")
}
```

---

## Constants (ค่าคงที่)

### TypeScript

```typescript
// TypeScript - const กับทุก value
const PI = 3.14159
const MAX_SIZE = 100
const APP_NAME = "MyApp"
const CONFIG = { port: 8080 }  // object ก็ได้
```

### Go

```go
// Go - const ใช้กับ compile-time values เท่านั้น
const PI = 3.14159
const MaxSize = 100
const AppName = "MyApp"

// const CONFIG = map[string]int{}  // Error! map ไม่ได้

// ประกาศหลายตัว
const (
    StatusPending = "pending"
    StatusActive  = "active"
    StatusDone    = "done"
)

// iota - auto increment (เหมือน enum)
const (
    Sunday = iota    // 0
    Monday           // 1
    Tuesday          // 2
    Wednesday        // 3
    Thursday         // 4
    Friday           // 5
    Saturday         // 6
)

// iota patterns
const (
    _  = iota             // 0 (skip)
    KB = 1 << (10 * iota) // 1 << 10 = 1024
    MB                    // 1 << 20 = 1048576
    GB                    // 1 << 30 = 1073741824
)

// Typed constants
const (
    MaxInt8   int8  = 127
    MaxUint16 uint16 = 65535
)
```

---

## Zero Values (ค่าเริ่มต้น)

### TypeScript - undefined

```typescript
// TypeScript - ไม่กำหนดค่า = undefined
let name: string   // undefined
let count: number  // undefined
let items: string[] // undefined
```

### Go - Zero Value ทุก Type

```go
// Go - ทุก type มี zero value (ไม่มี undefined)
var s string    // "" (empty string)
var i int       // 0
var f float64   // 0.0
var b bool      // false
var p *int      // nil (pointer)
var sl []int    // nil (slice)
var m map[string]int  // nil (map)
var ch chan int       // nil (channel)
var fn func()         // nil (function)
var iface interface{} // nil (interface)

// struct - ทุก field เป็น zero value
type User struct {
    Name string
    Age  int
}
var u User  // User{Name: "", Age: 0}
```

### ตาราง Zero Values

| Type | Zero Value | หมายเหตุ |
|------|------------|----------|
| `bool` | `false` | |
| `int`, `int8`...`int64` | `0` | |
| `uint`, `uint8`...`uint64` | `0` | |
| `float32`, `float64` | `0.0` | |
| `string` | `""` | empty string |
| `*T` (pointer) | `nil` | |
| `[]T` (slice) | `nil` | len=0, cap=0 |
| `map[K]V` | `nil` | ใช้ไม่ได้จนกว่า make() |
| `chan T` | `nil` | |
| `func` | `nil` | |
| `interface{}` | `nil` | |
| `struct` | ทุก field = zero | |

---

## Pointers (พอยน์เตอร์)

### ทำไมต้องมี Pointer?

TypeScript ไม่มี pointer แต่มี reference types (object, array) ที่ pass by reference อัตโนมัติ

Go ทุกอย่างเป็น pass by value ดังนั้น pointer ใช้เมื่อ:
- ต้องการแก้ไขค่าใน function
- ไม่อยาก copy ข้อมูลขนาดใหญ่
- ต้องการแสดงว่าค่าอาจเป็น nil

### Syntax พื้นฐาน

```go
// ประกาศ pointer
var p *int      // p เป็น pointer ไปยัง int (ยังไม่ชี้อะไร = nil)

// สร้าง pointer จากตัวแปร
x := 42
p = &x          // & = address-of (เอา address ของ x)

// อ่านค่าจาก pointer
fmt.Println(*p) // * = dereference (เข้าถึงค่า) → 42

// เปลี่ยนค่าผ่าน pointer
*p = 100        // x เปลี่ยนเป็น 100 ด้วย!
fmt.Println(x)  // 100
```

### เปรียบเทียบกับ TypeScript

```typescript
// TypeScript - object เป็น reference อัตโนมัติ
function updateUser(user: { name: string }) {
    user.name = "Jane"  // แก้ไขค่าจริงได้เลย
}

const u = { name: "John" }
updateUser(u)
console.log(u.name)  // "Jane"
```

```go
// Go - ต้องใช้ pointer ถ้าจะแก้ไข
func updateUser(user *User) {
    user.Name = "Jane"  // แก้ไขผ่าน pointer
}

u := User{Name: "John"}
updateUser(&u)         // ส่ง address
fmt.Println(u.Name)    // "Jane"

// ถ้าไม่ใช้ pointer - แก้ไข copy
func updateUserCopy(user User) {
    user.Name = "Jane"  // แก้ไข copy เท่านั้น
}

u := User{Name: "John"}
updateUserCopy(u)      // ส่ง copy
fmt.Println(u.Name)    // "John" (ไม่เปลี่ยน!)
```

### new() vs &T{}

```go
// สร้าง pointer 2 วิธี

// วิธี 1: new() - ได้ pointer ไปยัง zero value
p := new(User)           // *User ชี้ไปยัง User{}
p.Name = "John"

// วิธี 2: &T{} - ได้ pointer พร้อมกำหนดค่า (ใช้บ่อยกว่า)
p := &User{Name: "John"} // *User ชี้ไปยัง User{Name: "John"}

// สำหรับ basic types
x := new(int)   // *int ชี้ไปยัง 0
*x = 42

// หรือ
val := 42
x := &val       // *int ชี้ไปยัง val
```

### Pointer ใน Struct Methods

```go
type Counter struct {
    value int
}

// Value receiver - รับ copy (แก้ไขไม่ได้)
func (c Counter) GetValue() int {
    return c.value
}

// Pointer receiver - รับ pointer (แก้ไขได้)
func (c *Counter) Increment() {
    c.value++  // แก้ไขค่าจริง
}

// ใช้งาน
counter := Counter{value: 0}
counter.Increment()  // Go แปลง &counter ให้อัตโนมัติ
fmt.Println(counter.GetValue())  // 1
```

### เมื่อไหร่ใช้ Pointer?

| สถานการณ์ | ใช้ Pointer? | เหตุผล |
|-----------|--------------|--------|
| Method แก้ไข struct | ใช้ `*T` | ต้องแก้ไขค่าจริง |
| Struct ขนาดใหญ่ | ใช้ `*T` | ไม่ต้อง copy |
| อาจเป็น nil (optional) | ใช้ `*T` | แสดงว่าอาจไม่มีค่า |
| Struct เล็ก, read only | ไม่ใช้ | copy เร็วกว่า |
| Basic types (int, string) | ไม่ใช้ | copy เร็ว |
| Slice, Map, Channel | ไม่ใช้ | เป็น reference อยู่แล้ว |

---

## ตัวอย่างจาก Booking Rush

```go
// backend-auth/internal/domain/user.go
type User struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Password  string    `json:"-"`          // - = ไม่ส่งใน JSON
    Role      string    `json:"role"`
    TenantID  string    `json:"tenant_id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// backend-booking/internal/dto/booking.go
type ReserveRequest struct {
    EventID   string `json:"event_id" binding:"required"`
    ZoneID    string `json:"zone_id" binding:"required"`
    ShowID    string `json:"show_id" binding:"required"`
    Quantity  int    `json:"quantity" binding:"required,min=1,max=10"`
    UnitPrice int64  `json:"unit_price" binding:"required"`
}

// Pointer ใช้กับ optional fields
type UpdateUserRequest struct {
    Name  *string `json:"name"`   // อาจไม่ส่งมา
    Email *string `json:"email"`  // อาจไม่ส่งมา
}
```

---

## สรุป

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| ประกาศตัวแปร | `let x = 1` | `x := 1` |
| Const | `const x = 1` | `const x = 1` |
| Number types | `number` | `int`, `int64`, `float64`, etc. |
| String template | `` `${var}` `` | `fmt.Sprintf("%s", var)` |
| Type conversion | Implicit/explicit | Explicit เท่านั้น |
| Zero value | `undefined` | มี default ทุก type |
| Pointer | ไม่มี | `*T`, `&x`, `*p` |
| Null check | `x ?? default` | `if x != nil` |

---

## ต่อไป

- [02-functions.md](./02-functions.md) - Functions (ฟังก์ชัน)
