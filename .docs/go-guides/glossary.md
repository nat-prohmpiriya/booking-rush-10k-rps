# Glossary - คำศัพท์ Go-Thai-English

คำศัพท์ที่ใช้บ่อยในการเขียน Go สำหรับ TypeScript Developer

---

## A

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Array** | อาร์เรย์ | ชุดข้อมูลขนาดคงที่ `[5]int` |
| **Append** | เพิ่มต่อท้าย | ฟังก์ชันเพิ่มข้อมูลใน slice `append(slice, item)` |
| **Anonymous function** | ฟังก์ชันไม่ระบุชื่อ | ฟังก์ชันที่ไม่มีชื่อ เหมือน arrow function ใน TS |

## B

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Boolean** | บูลีน | ค่าจริง/เท็จ (`true`/`false`) ใน Go ใช้ `bool` |
| **Buffer** | บัฟเฟอร์ | พื้นที่เก็บข้อมูลชั่วคราว เช่น buffered channel |
| **Blocking** | บล็อก/รอ | การหยุดรอจนกว่าจะมีผลลัพธ์ |

## C

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Channel** | ช่องทาง | ท่อส่งข้อมูลระหว่าง goroutines `chan T` |
| **Concurrency** | การทำงานพร้อมกัน | รันหลายงานพร้อมกัน (ไม่จำเป็นต้อง parallel) |
| **Compiler** | คอมไพเลอร์ | โปรแกรมแปลงโค้ดเป็น binary |
| **Closure** | คลอเชอร์ | ฟังก์ชันที่เข้าถึงตัวแปรนอก scope ได้ |
| **Context** | บริบท | object สำหรับส่ง cancellation, timeout, values |
| **Composite type** | ประเภทประกอบ | type ที่สร้างจาก type อื่น เช่น struct, slice, map |

## D

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Defer** | เลื่อนไปทำทีหลัง | รันตอนจบ function `defer file.Close()` |
| **Declaration** | การประกาศ | การสร้างตัวแปร/ฟังก์ชัน |
| **Dependency** | การพึ่งพา | package ที่โปรเจคต้องใช้ |
| **Duck typing** | - | ถ้ามี method ครบก็ถือว่า implement interface |

## E

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Error** | ข้อผิดพลาด | interface สำหรับแสดงความผิดพลาด |
| **Exported** | ส่งออก/เปิดเผย | ชื่อขึ้นต้นตัวใหญ่ = public |
| **Embedding** | ฝังตัว | การนำ struct/interface มารวมกัน (แทน inheritance) |

## F

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Function** | ฟังก์ชัน | บล็อกโค้ดที่ทำงานเฉพาะ |
| **Field** | ฟิลด์ | ตัวแปรภายใน struct |
| **Format string** | รูปแบบสตริง | `fmt.Sprintf("%s %d", str, num)` |

## G

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Goroutine** | โกรูทีน | lightweight thread ของ Go |
| **Go module** | โมดูล | ระบบจัดการ dependencies ของ Go |
| **Garbage collection** | เก็บขยะอัตโนมัติ | ระบบจัดการ memory อัตโนมัติ |

## H

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Handler** | ตัวจัดการ | ฟังก์ชันรับ request และส่ง response |
| **Heap** | ฮีป | พื้นที่ memory สำหรับ dynamic allocation |

## I

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Interface** | อินเตอร์เฟซ | สัญญาว่าต้องมี method อะไรบ้าง |
| **Import** | นำเข้า | การเรียกใช้ package อื่น |
| **Inference** | อนุมาน | compiler เดา type ให้อัตโนมัติ |
| **Initialize** | กำหนดค่าเริ่มต้น | ตั้งค่าตัวแปรครั้งแรก |
| **Idiomatic** | เป็นแบบฉบับ | วิธีเขียนที่ชุมชน Go ยอมรับ |

## J

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **JSON tag** | แท็ก JSON | กำหนดชื่อ field ใน JSON `json:"name"` |

## K

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Keyword** | คำสงวน | คำที่ Go จองไว้ เช่น `func`, `var`, `if` |

## L

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Literal** | ค่าตรงๆ | ค่าที่เขียนตรงๆ เช่น `"hello"`, `123` |
| **Loop** | วนซ้ำ | Go มีแค่ `for` (ไม่มี while) |

## M

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Map** | แมป | โครงสร้าง key-value `map[string]int` |
| **Method** | เมธอด | ฟังก์ชันที่ผูกกับ type |
| **Mutex** | มิวเท็กซ์ | ล็อคป้องกัน race condition |
| **Module** | โมดูล | หน่วยจัดการ dependencies |

## N

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Nil** | นิล | ค่าว่าง (เหมือน null/undefined ใน TS) |
| **Non-blocking** | ไม่บล็อก | ไม่ต้องรอ ทำงานต่อได้เลย |

## O

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Operator** | ตัวดำเนินการ | เครื่องหมาย เช่น `+`, `-`, `==` |

## P

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Package** | แพ็คเกจ | กลุ่มไฟล์ Go ที่อยู่ใน folder เดียวกัน |
| **Pointer** | พอยน์เตอร์ | ตัวแปรเก็บ address ของตัวแปรอื่น `*T` |
| **Panic** | แพนิค | error ร้ายแรงที่หยุดโปรแกรม |
| **Parameter** | พารามิเตอร์ | ตัวแปรที่รับเข้ามาใน function |
| **Parallel** | ขนาน | รันพร้อมกันจริงๆ บน multi-core |

## R

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Receiver** | ตัวรับ | struct ที่ method ผูกอยู่ `(s *Service)` |
| **Return** | คืนค่า | ส่งค่ากลับจาก function |
| **Range** | ช่วง | วนลูป slice/map `for i, v := range slice` |
| **Race condition** | - | bug จากหลาย goroutine แย่งแก้ไขข้อมูล |
| **Recover** | กู้คืน | จับ panic ไม่ให้โปรแกรมตาย |

## S

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Struct** | สตรัค | โครงสร้างข้อมูล (เหมือน class แต่ไม่มี inheritance) |
| **Slice** | สไลซ์ | อาร์เรย์ขนาดไม่คงที่ `[]int` |
| **Select** | เลือก | switch สำหรับ channel |
| **Scope** | ขอบเขต | ส่วนที่ตัวแปรเข้าถึงได้ |
| **Short declaration** | ประกาศแบบสั้น | `:=` ประกาศและกำหนดค่าพร้อมกัน |
| **Stack** | สแตก | พื้นที่ memory สำหรับ local variables |
| **String** | สตริง | ข้อความ (immutable ใน Go) |
| **Synchronous** | ซิงโครนัส | ทำทีละอย่าง รอจบก่อนทำต่อ |

## T

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Type** | ประเภท | ชนิดข้อมูล เช่น `int`, `string`, `User` |
| **Type assertion** | ยืนยันประเภท | แปลง interface เป็น type จริง `v.(Type)` |
| **Tag** | แท็ก | metadata บน struct field `json:"name"` |
| **Test** | ทดสอบ | ไฟล์ `_test.go` และฟังก์ชัน `TestXxx` |

## U

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Unexported** | ไม่ส่งออก/ซ่อน | ชื่อขึ้นต้นตัวเล็ก = private |

## V

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Variable** | ตัวแปร | ที่เก็บข้อมูล |
| **Value** | ค่า | ข้อมูลที่เก็บในตัวแปร |
| **Variadic** | รับหลายค่า | function รับ arguments ไม่จำกัด `func(args ...int)` |

## W

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **WaitGroup** | กลุ่มรอ | รอหลาย goroutines ทำงานเสร็จ |
| **Wrapper** | ตัวห่อหุ้ม | function/type ที่ห่อหุ้มอีกตัว |

## Z

| English | Thai | คำอธิบาย |
|---------|------|----------|
| **Zero value** | ค่าศูนย์ | ค่าเริ่มต้นอัตโนมัติ (`0`, `""`, `false`, `nil`) |

---

## สัญลักษณ์สำคัญใน Go

| สัญลักษณ์ | ชื่อ | ความหมาย |
|-----------|------|----------|
| `:=` | short declaration | ประกาศและกำหนดค่า |
| `*` | pointer/dereference | ชี้ไปที่ address หรือเข้าถึงค่า |
| `&` | address-of | เอา address ของตัวแปร |
| `<-` | channel send/receive | ส่ง/รับข้อมูลผ่าน channel |
| `...` | spread/variadic | กระจายค่า หรือรับหลายค่า |
| `_` | blank identifier | ไม่สนใจค่านี้ |

---

## เปรียบเทียบศัพท์ TypeScript → Go

| TypeScript | Go | หมายเหตุ |
|------------|-----|----------|
| `interface` | `interface` | Go เป็น implicit (duck typing) |
| `class` | `struct` + methods | Go ไม่มี class |
| `extends` | embedding | Go ไม่มี inheritance |
| `implements` | (automatic) | ไม่ต้องระบุ ถ้ามี method ครบก็ implement |
| `constructor` | factory function | เช่น `NewUserService()` |
| `this` | receiver `(s *Service)` | ระบุชัดเจนว่าเป็นตัวไหน |
| `private` | ขึ้นต้นตัวเล็ก | `name` = private |
| `public` | ขึ้นต้นตัวใหญ่ | `Name` = public |
| `async/await` | goroutine + channel | หรือใช้ WaitGroup |
| `Promise` | channel | หรือ callback |
| `null/undefined` | `nil` | ใช้กับ pointer, slice, map, channel, interface |
| `any` | `interface{}` หรือ `any` | Go 1.18+ มี `any` |
| `Array<T>` | `[]T` | slice |
| `Record<K,V>` | `map[K]V` | map |
| `try/catch` | `if err != nil` | Go ไม่มี exception |
| `throw` | `return error` หรือ `panic` | ปกติใช้ return error |
| `import { x }` | `import "pkg"` แล้วใช้ `pkg.X` | ไม่มี named import |
| `export` | ชื่อขึ้นต้นตัวใหญ่ | Capital letter = exported |
