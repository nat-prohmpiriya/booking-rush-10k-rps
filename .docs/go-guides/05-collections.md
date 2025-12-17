# 05 - Collections (โครงสร้างข้อมูล)

## สารบัญ

1. [Arrays](#arrays-อาร์เรย์)
2. [Slices](#slices-สไลซ์)
3. [Maps](#maps-แมป)
4. [Looping](#looping-การวนลูป)
5. [Common Operations](#common-operations)

---

## Arrays (อาร์เรย์)

Array ใน Go มี **ขนาดคงที่** (fixed size) - แตกต่างจาก TypeScript

### TypeScript - Dynamic Array

```typescript
// TypeScript array = dynamic size
const numbers: number[] = [1, 2, 3]
numbers.push(4)  // OK - เพิ่มได้
console.log(numbers.length)  // 4
```

### Go - Fixed Size Array

```go
// Go array = fixed size (ระบุขนาดตอนประกาศ)
var numbers [3]int           // [0, 0, 0]
numbers[0] = 1
numbers[1] = 2
numbers[2] = 3
// numbers[3] = 4            // Error! out of bounds

// ประกาศพร้อมค่า
numbers := [3]int{1, 2, 3}

// ให้ compiler นับขนาด
numbers := [...]int{1, 2, 3, 4, 5}  // [5]int

// Length
len(numbers)  // 5

// Array เป็น value type - copy เมื่อ assign หรือ pass to function
arr1 := [3]int{1, 2, 3}
arr2 := arr1       // copy ทั้ง array!
arr2[0] = 100
fmt.Println(arr1)  // [1 2 3] - ไม่เปลี่ยน
fmt.Println(arr2)  // [100 2 3]
```

### เมื่อไหร่ใช้ Array?

Go array ใช้น้อยมาก เพราะ:
- ขนาดคงที่
- เป็น value type (copy เยอะ)

ใช้เมื่อ:
- ต้องการขนาดแน่นอน เช่น `[32]byte` สำหรับ hash
- Performance critical และรู้ขนาดล่วงหน้า

**ปกติใช้ Slice แทน!**

---

## Slices (สไลซ์)

Slice = **Dynamic array** ของ Go (ใช้แทน TypeScript array)

### Basic Slice

```typescript
// TypeScript
const numbers: number[] = [1, 2, 3]
numbers.push(4)
console.log(numbers)  // [1, 2, 3, 4]
```

```go
// Go - Slice
numbers := []int{1, 2, 3}        // ไม่ระบุขนาด = slice
numbers = append(numbers, 4)     // append return new slice
fmt.Println(numbers)             // [1 2 3 4]

// Zero value = nil
var empty []int                  // nil slice
fmt.Println(empty == nil)        // true
fmt.Println(len(empty))          // 0

// Empty slice (ไม่ใช่ nil)
empty := []int{}                 // empty slice, not nil
fmt.Println(empty == nil)        // false
fmt.Println(len(empty))          // 0
```

### สร้าง Slice ด้วย make()

```go
// make(type, length, capacity)
slice := make([]int, 5)         // len=5, cap=5, filled with zeros
slice := make([]int, 0, 10)     // len=0, cap=10

// length vs capacity
// length = จำนวน elements ปัจจุบัน
// capacity = ขนาดที่รองรับได้ก่อน reallocate

s := make([]int, 3, 10)
fmt.Println(len(s))   // 3
fmt.Println(cap(s))   // 10

// Pre-allocate เพื่อ performance
users := make([]User, 0, 100)   // รู้ว่าจะมีประมาณ 100 คน
for _, row := range rows {
    users = append(users, parseUser(row))
}
```

### Slice Operations

```go
// Access element
slice := []int{10, 20, 30, 40, 50}
first := slice[0]      // 10
last := slice[len(slice)-1]  // 50

// Slice of slice (slicing)
slice[1:3]   // [20, 30] - index 1 to 2
slice[:3]    // [10, 20, 30] - from start to 2
slice[2:]    // [30, 40, 50] - from 2 to end
slice[:]     // [10, 20, 30, 40, 50] - all

// ⚠️ Slicing shares memory!
original := []int{1, 2, 3, 4, 5}
sub := original[1:3]   // [2, 3]
sub[0] = 100
fmt.Println(original)  // [1, 100, 3, 4, 5] - changed!

// Copy เพื่อไม่ share memory
sub := make([]int, 2)
copy(sub, original[1:3])  // copy values
sub[0] = 100
fmt.Println(original)     // [1, 2, 3, 4, 5] - unchanged
```

### Append

```go
// append(slice, elements...)
slice := []int{1, 2, 3}
slice = append(slice, 4)           // [1, 2, 3, 4]
slice = append(slice, 5, 6, 7)     // [1, 2, 3, 4, 5, 6, 7]

// Append another slice
other := []int{8, 9, 10}
slice = append(slice, other...)    // [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

// ⚠️ append อาจ reallocate memory
// ต้อง assign กลับเสมอ!
slice := []int{1, 2}
append(slice, 3)            // ❌ value lost!
slice = append(slice, 3)    // ✅ correct
```

### Remove Element

```go
// Remove by index
slice := []int{1, 2, 3, 4, 5}
i := 2  // remove index 2 (value 3)

// วิธี 1: รักษาลำดับ
slice = append(slice[:i], slice[i+1:]...)  // [1, 2, 4, 5]

// วิธี 2: ไม่รักษาลำดับ (เร็วกว่า)
slice[i] = slice[len(slice)-1]  // swap กับตัวสุดท้าย
slice = slice[:len(slice)-1]    // ตัดตัวสุดท้ายออก

// Remove by value
func remove(slice []int, value int) []int {
    result := make([]int, 0, len(slice))
    for _, v := range slice {
        if v != value {
            result = append(result, v)
        }
    }
    return result
}
```

### Copy Slice

```go
// copy(dst, src) - copy elements
src := []int{1, 2, 3}
dst := make([]int, len(src))
copy(dst, src)

// Shorthand - append to nil
clone := append([]int(nil), src...)

// Deep copy สำหรับ slice of pointers/structs
users := []*User{{Name: "A"}, {Name: "B"}}
clone := make([]*User, len(users))
for i, u := range users {
    copy := *u           // copy struct value
    clone[i] = &copy     // store pointer to copy
}
```

---

## Maps (แมป)

Map = key-value store (เหมือน Object/Map ใน TypeScript)

### TypeScript Object/Map

```typescript
// Object
const scores: Record<string, number> = {}
scores["alice"] = 100
scores["bob"] = 85

// Map
const scores = new Map<string, number>()
scores.set("alice", 100)
scores.get("alice")  // 100
scores.has("alice")  // true
scores.delete("alice")
```

### Go Map

```go
// สร้าง map
var scores map[string]int     // nil map (อ่านได้ เขียนไม่ได้!)
scores := map[string]int{}    // empty map
scores := make(map[string]int) // empty map with make

// ประกาศพร้อมค่า
scores := map[string]int{
    "alice": 100,
    "bob":   85,
}

// Set
scores["charlie"] = 90

// Get
score := scores["alice"]      // 100
missing := scores["unknown"]  // 0 (zero value)

// Check if key exists (comma ok idiom)
score, ok := scores["alice"]
if ok {
    fmt.Println("Alice's score:", score)
} else {
    fmt.Println("Alice not found")
}

// Delete
delete(scores, "bob")

// Length
len(scores)  // จำนวน keys

// ⚠️ nil map - อ่านได้ เขียน panic!
var m map[string]int
_ = m["key"]     // OK - returns 0
m["key"] = 1     // panic: assignment to nil map
```

### Map Operations

```go
// Iterate
for key, value := range scores {
    fmt.Printf("%s: %d\n", key, value)
}

// Keys only
for key := range scores {
    fmt.Println(key)
}

// Values only
for _, value := range scores {
    fmt.Println(value)
}

// ⚠️ Map iteration order is random!
// ถ้าต้องการลำดับ ต้อง sort keys เอง
keys := make([]string, 0, len(scores))
for k := range scores {
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    fmt.Printf("%s: %d\n", k, scores[k])
}
```

### Map กับ Struct

```go
// Map of structs
type User struct {
    Name  string
    Email string
}

users := map[string]User{
    "u1": {Name: "Alice", Email: "alice@example.com"},
    "u2": {Name: "Bob", Email: "bob@example.com"},
}

// ⚠️ ไม่สามารถแก้ไข field ของ struct ใน map โดยตรง
users["u1"].Name = "Alice Updated"  // Error!

// ต้อง copy ออกมา แก้ แล้วใส่กลับ
u := users["u1"]
u.Name = "Alice Updated"
users["u1"] = u

// หรือใช้ pointer
users := map[string]*User{
    "u1": {Name: "Alice", Email: "alice@example.com"},
}
users["u1"].Name = "Alice Updated"  // OK!
```

### Map as Set

Go ไม่มี Set - ใช้ map แทน

```typescript
// TypeScript Set
const seen = new Set<string>()
seen.add("apple")
seen.has("apple")  // true
```

```go
// Go - map[T]bool หรือ map[T]struct{}
seen := map[string]bool{}
seen["apple"] = true

if seen["apple"] {
    fmt.Println("apple exists")
}

// ประหยัด memory กว่า - map[T]struct{}
seen := map[string]struct{}{}
seen["apple"] = struct{}{}

if _, ok := seen["apple"]; ok {
    fmt.Println("apple exists")
}

// Unique values
func unique(items []string) []string {
    seen := map[string]struct{}{}
    result := []string{}

    for _, item := range items {
        if _, ok := seen[item]; !ok {
            seen[item] = struct{}{}
            result = append(result, item)
        }
    }
    return result
}
```

---

## Looping (การวนลูป)

Go มี `for` loop แบบเดียว (ไม่มี while, forEach)

### TypeScript Loops

```typescript
// for
for (let i = 0; i < 10; i++) { }

// for...of (iterate values)
for (const item of items) { }

// for...in (iterate keys)
for (const key in obj) { }

// forEach
items.forEach((item, index) => { })

// while
while (condition) { }
```

### Go - for เท่านั้น

```go
// Classic for
for i := 0; i < 10; i++ {
    fmt.Println(i)
}

// While style
for condition {
    // ...
}

// Infinite loop
for {
    // ...
    if done {
        break
    }
}

// for...range (iterate slice/map/string/channel)
for index, value := range slice {
    fmt.Printf("%d: %v\n", index, value)
}

// Index only
for i := range slice {
    fmt.Println(i)
}

// Value only
for _, v := range slice {
    fmt.Println(v)
}
```

### Range กับ Collections

```go
// Slice
numbers := []int{10, 20, 30}
for i, n := range numbers {
    fmt.Printf("index %d: %d\n", i, n)
}

// Map (random order!)
scores := map[string]int{"a": 1, "b": 2}
for key, value := range scores {
    fmt.Printf("%s: %d\n", key, value)
}

// String (iterate runes)
for i, r := range "Hello ภาษาไทย" {
    fmt.Printf("%d: %c\n", i, r)  // r เป็น rune (Unicode code point)
}

// Channel
for msg := range ch {
    fmt.Println(msg)
}
```

### Break และ Continue

```go
// break - ออกจาก loop
for i := 0; i < 10; i++ {
    if i == 5 {
        break  // ออกเมื่อ i = 5
    }
}

// continue - ข้ามไป iteration ถัดไป
for i := 0; i < 10; i++ {
    if i%2 == 0 {
        continue  // ข้ามเลขคู่
    }
    fmt.Println(i)  // พิมพ์เลขคี่
}

// Label - break/continue nested loops
outer:
for i := 0; i < 3; i++ {
    for j := 0; j < 3; j++ {
        if i == 1 && j == 1 {
            break outer  // ออกจาก outer loop
        }
    }
}
```

---

## Common Operations

### TypeScript Array Methods → Go

```typescript
// TypeScript
const numbers = [1, 2, 3, 4, 5]

// map
const doubled = numbers.map(n => n * 2)

// filter
const evens = numbers.filter(n => n % 2 === 0)

// find
const first = numbers.find(n => n > 3)

// some
const hasEven = numbers.some(n => n % 2 === 0)

// every
const allPositive = numbers.every(n => n > 0)

// reduce
const sum = numbers.reduce((acc, n) => acc + n, 0)

// includes
const has3 = numbers.includes(3)
```

```go
// Go - ต้องเขียน loop เอง (ไม่มี built-in)
numbers := []int{1, 2, 3, 4, 5}

// map equivalent
doubled := make([]int, len(numbers))
for i, n := range numbers {
    doubled[i] = n * 2
}

// filter equivalent
var evens []int
for _, n := range numbers {
    if n%2 == 0 {
        evens = append(evens, n)
    }
}

// find equivalent
func find(numbers []int, predicate func(int) bool) (int, bool) {
    for _, n := range numbers {
        if predicate(n) {
            return n, true
        }
    }
    return 0, false
}
first, found := find(numbers, func(n int) bool { return n > 3 })

// some equivalent
func some(numbers []int, predicate func(int) bool) bool {
    for _, n := range numbers {
        if predicate(n) {
            return true
        }
    }
    return false
}
hasEven := some(numbers, func(n int) bool { return n%2 == 0 })

// every equivalent
func every(numbers []int, predicate func(int) bool) bool {
    for _, n := range numbers {
        if !predicate(n) {
            return false
        }
    }
    return true
}
allPositive := every(numbers, func(n int) bool { return n > 0 })

// reduce equivalent
sum := 0
for _, n := range numbers {
    sum += n
}

// includes equivalent
func contains(numbers []int, target int) bool {
    for _, n := range numbers {
        if n == target {
            return true
        }
    }
    return false
}
has3 := contains(numbers, 3)
```

### Generic Helper Functions (Go 1.18+)

```go
// Go 1.18+ มี generics
package slices

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

// Contains (มีใน slices package)
import "slices"
slices.Contains(numbers, 3)  // Go 1.21+
```

### Sort

```go
import "sort"

// Sort slice of int
numbers := []int{3, 1, 4, 1, 5}
sort.Ints(numbers)           // [1, 1, 3, 4, 5]
sort.Sort(sort.Reverse(sort.IntSlice(numbers)))  // [5, 4, 3, 1, 1]

// Sort slice of string
names := []string{"Charlie", "Alice", "Bob"}
sort.Strings(names)          // [Alice, Bob, Charlie]

// Sort slice of struct
users := []User{
    {Name: "Charlie", Age: 30},
    {Name: "Alice", Age: 25},
    {Name: "Bob", Age: 35},
}

// By field
sort.Slice(users, func(i, j int) bool {
    return users[i].Age < users[j].Age  // sort by Age ascending
})

// Multiple fields
sort.Slice(users, func(i, j int) bool {
    if users[i].Age != users[j].Age {
        return users[i].Age < users[j].Age
    }
    return users[i].Name < users[j].Name  // secondary sort
})

// Go 1.21+ slices package
import "slices"
slices.Sort(numbers)
slices.SortFunc(users, func(a, b User) int {
    return a.Age - b.Age
})
```

---

## ตัวอย่างจาก Booking Rush

```go
// backend-ticket/internal/service/event_service.go

// Get events with filters
func (s *EventService) GetEvents(ctx context.Context, filters *dto.EventFilters) ([]dto.EventResponse, error) {
    events, err := s.repo.FindAll(ctx, filters)
    if err != nil {
        return nil, fmt.Errorf("find events: %w", err)
    }

    // Map domain to DTO
    responses := make([]dto.EventResponse, len(events))
    for i, event := range events {
        responses[i] = dto.EventResponse{
            ID:          event.ID,
            Name:        event.Name,
            Description: event.Description,
            StartDate:   event.StartDate,
            EndDate:     event.EndDate,
        }
    }

    return responses, nil
}

// backend-booking/internal/service/booking_service.go

// Group bookings by status
func (s *BookingService) GetBookingStats(ctx context.Context, userID string) (*dto.BookingStats, error) {
    bookings, err := s.repo.FindByUserID(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Count by status
    stats := map[domain.BookingStatus]int{}
    for _, b := range bookings {
        stats[b.Status]++
    }

    return &dto.BookingStats{
        Total:     len(bookings),
        Pending:   stats[domain.BookingStatusPending],
        Confirmed: stats[domain.BookingStatusConfirmed],
        Cancelled: stats[domain.BookingStatusCancelled],
    }, nil
}

// scripts/lua/reserve_seats.lua helper
func (s *BookingService) CheckAvailability(ctx context.Context, zoneIDs []string) (map[string]int, error) {
    // Get available seats for multiple zones
    availability := make(map[string]int, len(zoneIDs))

    for _, zoneID := range zoneIDs {
        seats, err := s.redis.Get(ctx, fmt.Sprintf("inventory:zone:%s", zoneID))
        if err != nil {
            continue  // skip on error
        }
        availability[zoneID], _ = strconv.Atoi(seats)
    }

    return availability, nil
}
```

---

## สรุป

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| Dynamic array | `number[]` | `[]int` (slice) |
| Fixed array | - | `[5]int` (array) |
| Create empty | `[]` | `make([]T, 0)` |
| Add element | `push()` | `append()` |
| Length | `.length` | `len()` |
| Object/Map | `Record<K,V>` | `map[K]V` |
| Check key exists | `"key" in obj` | `_, ok := m["key"]` |
| Delete key | `delete obj.key` | `delete(m, "key")` |
| Iterate | `for...of`, `forEach` | `for range` |
| map/filter | built-in methods | ต้องเขียน loop |
| Sort | `sort()` | `sort.Slice()` |
| Set | `Set<T>` | `map[T]struct{}` |

### Tips

- ใช้ **slice** แทน array เกือบทุกกรณี
- ใช้ `make()` pre-allocate ถ้ารู้ขนาดล่วงหน้า
- ระวัง **nil map** - เขียนไม่ได้!
- Map iteration **ไม่มีลำดับ**
- ต้อง **assign กลับ** หลัง `append()`

---

## ต่อไป

- [06-concurrency.md](./06-concurrency.md) - Goroutines, Channels
