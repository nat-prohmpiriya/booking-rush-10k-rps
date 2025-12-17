# 06 - Concurrency (การทำงานพร้อมกัน)

## สารบัญ

1. [แนวคิด Concurrency](#แนวคิด-concurrency)
2. [Goroutines](#goroutines-โกรูทีน)
3. [Channels](#channels-ช่องทางสื่อสาร)
4. [Select](#select)
5. [WaitGroup](#waitgroup)
6. [Mutex](#mutex)
7. [Context](#context-บริบท)
8. [Common Patterns](#common-patterns)

---

## แนวคิด Concurrency

### TypeScript - Single Thread + Event Loop

```typescript
// TypeScript/Node.js = Single thread + async/await
async function fetchAll() {
    // Promise.all = รันพร้อมกัน (concurrent I/O)
    const [users, products] = await Promise.all([
        fetch('/users').then(r => r.json()),
        fetch('/products').then(r => r.json()),
    ])
    return { users, products }
}

// setTimeout = schedule later
setTimeout(() => console.log('delayed'), 1000)

// Worker threads สำหรับ CPU-intensive work
import { Worker } from 'worker_threads'
```

### Go - Goroutines + Channels

```go
// Go = Multiple goroutines (lightweight threads)
func fetchAll() ([]User, []Product) {
    usersCh := make(chan []User)
    productsCh := make(chan []Product)

    // go = spawn goroutine
    go func() { usersCh <- fetchUsers() }()
    go func() { productsCh <- fetchProducts() }()

    // รอรับผลลัพธ์
    users := <-usersCh
    products := <-productsCh

    return users, products
}
```

### ความแตกต่างหลัก

| TypeScript | Go |
|------------|-----|
| Single thread + event loop | Multiple goroutines |
| async/await | go keyword + channels |
| Promise.all | WaitGroup / Channels |
| Callbacks/Promises | Channels |
| Worker threads (heavy) | Goroutines (lightweight) |
| ~10MB per thread | ~2KB per goroutine |

---

## Goroutines (โกรูทีน)

Goroutine = lightweight thread ที่ Go runtime จัดการ

### สร้าง Goroutine

```go
// go + function call
go doSomething()

// go + anonymous function
go func() {
    fmt.Println("Running in goroutine")
}()

// go + method
go server.Start()

// ⚠️ main function จบ = ทุก goroutine ตายด้วย
func main() {
    go fmt.Println("Hello")  // อาจไม่ทันพิมพ์!
}  // main จบ, goroutine ตาย

// ต้องรอ goroutine
func main() {
    go fmt.Println("Hello")
    time.Sleep(100 * time.Millisecond)  // รอ (วิธีนี้ไม่ดี)
}
```

### Goroutine กับ Loop

```go
// ⚠️ ผิด! - closure capture variable
for i := 0; i < 5; i++ {
    go func() {
        fmt.Println(i)  // อาจได้ 5, 5, 5, 5, 5
    }()
}

// ✅ ถูก - pass as parameter
for i := 0; i < 5; i++ {
    go func(n int) {
        fmt.Println(n)  // 0, 1, 2, 3, 4 (ลำดับอาจสลับ)
    }(i)
}

// ✅ ถูก - shadow variable (Go 1.22+ แก้ปัญหานี้แล้ว)
for i := 0; i < 5; i++ {
    i := i  // shadow
    go func() {
        fmt.Println(i)
    }()
}
```

---

## Channels (ช่องทางสื่อสาร)

Channel = ท่อส่งข้อมูลระหว่าง goroutines

### TypeScript - No direct equivalent

```typescript
// TypeScript ไม่มี channel
// ใช้ EventEmitter / Observable แทน
import { EventEmitter } from 'events'

const emitter = new EventEmitter()
emitter.on('data', (data) => console.log(data))
emitter.emit('data', 'hello')
```

### Go - Channels

```go
// สร้าง channel
ch := make(chan string)      // unbuffered channel
ch := make(chan string, 10)  // buffered channel (size 10)

// ส่งข้อมูล (send)
ch <- "hello"

// รับข้อมูล (receive)
msg := <-ch

// ปิด channel
close(ch)
```

### Unbuffered vs Buffered

```go
// Unbuffered - ต้องมีคนรอรับถึงจะส่งได้ (synchronous)
ch := make(chan int)

go func() {
    ch <- 42  // block จนกว่าจะมีคนรับ
}()

value := <-ch  // รับค่า

// Buffered - ส่งได้จนกว่า buffer เต็ม (async)
ch := make(chan int, 3)

ch <- 1  // OK - buffer ยังว่าง
ch <- 2  // OK
ch <- 3  // OK - buffer เต็ม
ch <- 4  // block! รอจนกว่าจะมีคนรับ

value := <-ch  // รับ 1, buffer มีที่ว่างแล้ว
```

### Channel Direction

```go
// Send-only channel
func producer(ch chan<- int) {
    ch <- 42
    // <-ch  // Error! ส่งได้อย่างเดียว
}

// Receive-only channel
func consumer(ch <-chan int) {
    value := <-ch
    // ch <- 1  // Error! รับได้อย่างเดียว
}

// Bidirectional
func process(ch chan int) {
    ch <- 1
    v := <-ch
}
```

### Range over Channel

```go
ch := make(chan int)

go func() {
    for i := 0; i < 5; i++ {
        ch <- i
    }
    close(ch)  // ต้อง close เพื่อให้ range จบ
}()

// รับจนกว่า channel จะถูก close
for value := range ch {
    fmt.Println(value)  // 0, 1, 2, 3, 4
}
```

### Check Channel Closed

```go
ch := make(chan int)
close(ch)

// วิธี 1: comma ok
value, ok := <-ch
if !ok {
    fmt.Println("channel closed")
}

// วิธี 2: range (จบเมื่อ closed)
for v := range ch {
    fmt.Println(v)
}
```

---

## Select

Select = switch สำหรับ channels (รอหลาย channels พร้อมกัน)

### TypeScript - Promise.race

```typescript
// รอตัวแรกที่เสร็จ
const result = await Promise.race([
    fetch('/api1'),
    fetch('/api2'),
    new Promise((_, reject) =>
        setTimeout(() => reject('timeout'), 5000)
    )
])
```

### Go - Select

```go
// select รอหลาย channels
select {
case msg := <-ch1:
    fmt.Println("from ch1:", msg)
case msg := <-ch2:
    fmt.Println("from ch2:", msg)
case ch3 <- "hello":
    fmt.Println("sent to ch3")
case <-time.After(5 * time.Second):
    fmt.Println("timeout!")
}
```

### Select Patterns

```go
// Non-blocking receive
select {
case msg := <-ch:
    fmt.Println(msg)
default:
    fmt.Println("no message available")
}

// Non-blocking send
select {
case ch <- msg:
    fmt.Println("sent")
default:
    fmt.Println("channel full, dropped")
}

// Timeout
select {
case result := <-resultCh:
    return result, nil
case <-time.After(10 * time.Second):
    return nil, errors.New("timeout")
}

// Multiple channels in loop
for {
    select {
    case msg := <-msgCh:
        process(msg)
    case err := <-errCh:
        handleError(err)
    case <-done:
        return
    }
}
```

---

## WaitGroup

WaitGroup = รอหลาย goroutines ให้เสร็จ

### TypeScript - Promise.all

```typescript
// รอทั้งหมดเสร็จ
await Promise.all([
    doTask1(),
    doTask2(),
    doTask3(),
])
```

### Go - sync.WaitGroup

```go
import "sync"

func main() {
    var wg sync.WaitGroup

    for i := 0; i < 5; i++ {
        wg.Add(1)  // บอกว่ามี goroutine เพิ่ม 1 ตัว

        go func(n int) {
            defer wg.Done()  // บอกว่า goroutine นี้เสร็จแล้ว
            fmt.Println("Worker", n)
        }(i)
    }

    wg.Wait()  // block จนกว่าทุกตัว Done()
    fmt.Println("All done")
}
```

### WaitGroup with Error Handling

```go
func processAll(items []Item) error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(items))  // buffered for all errors

    for _, item := range items {
        wg.Add(1)
        go func(it Item) {
            defer wg.Done()
            if err := process(it); err != nil {
                errCh <- err
            }
        }(item)
    }

    // รอทั้งหมดเสร็จ
    wg.Wait()
    close(errCh)

    // Check errors
    for err := range errCh {
        if err != nil {
            return err  // return first error
        }
    }
    return nil
}
```

### errgroup (golang.org/x/sync/errgroup)

```go
import "golang.org/x/sync/errgroup"

func processAll(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, item := range items {
        item := item  // capture
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait()  // return first error หรือ nil
}
```

---

## Mutex

Mutex = ล็อคป้องกัน race condition

### Race Condition

```go
// ⚠️ Race condition!
counter := 0
var wg sync.WaitGroup

for i := 0; i < 1000; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        counter++  // หลาย goroutines แก้ไขพร้อมกัน
    }()
}

wg.Wait()
fmt.Println(counter)  // อาจไม่ใช่ 1000!
```

### แก้ด้วย Mutex

```go
import "sync"

counter := 0
var mu sync.Mutex
var wg sync.WaitGroup

for i := 0; i < 1000; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        mu.Lock()
        counter++
        mu.Unlock()
    }()
}

wg.Wait()
fmt.Println(counter)  // 1000 แน่นอน
```

### RWMutex (Read-Write Lock)

```go
import "sync"

type SafeCache struct {
    mu    sync.RWMutex
    items map[string]string
}

// Read - หลายตัวอ่านพร้อมกันได้
func (c *SafeCache) Get(key string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.items[key]
    return val, ok
}

// Write - ต้องรอคนเดียว
func (c *SafeCache) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = value
}
```

### Atomic Operations

```go
import "sync/atomic"

// Atomic counter - เร็วกว่า mutex
var counter int64

for i := 0; i < 1000; i++ {
    go func() {
        atomic.AddInt64(&counter, 1)
    }()
}

// Go 1.19+ atomic types
var counter atomic.Int64

for i := 0; i < 1000; i++ {
    go func() {
        counter.Add(1)
    }()
}

fmt.Println(counter.Load())
```

---

## Context (บริบท)

Context = ส่ง cancellation, timeout, values ระหว่าง goroutines

### TypeScript - AbortController

```typescript
// Cancellation
const controller = new AbortController()
const signal = controller.signal

fetch('/api', { signal })
    .then(r => r.json())
    .catch(err => {
        if (err.name === 'AbortError') {
            console.log('Cancelled')
        }
    })

// Cancel after 5 seconds
setTimeout(() => controller.abort(), 5000)
```

### Go - context.Context

```go
import "context"

// สร้าง context
ctx := context.Background()           // root context
ctx := context.TODO()                 // placeholder

// Context with cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()  // ควร cancel เสมอ

// Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Context with deadline
deadline := time.Now().Add(10 * time.Second)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

// Context with value (use sparingly!)
ctx := context.WithValue(parentCtx, "userID", "123")
userID := ctx.Value("userID").(string)
```

### ใช้ Context ใน Function

```go
// ฟังก์ชันที่รับ context - ตรวจสอบ cancellation
func doWork(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()  // cancelled or timeout
        default:
            // ทำงาน
            if err := processItem(); err != nil {
                return err
            }
        }
    }
}

// HTTP request with timeout
func fetchWithTimeout(url string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err  // จะได้ context.DeadlineExceeded ถ้า timeout
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}
```

### Context in Web Handler

```go
// Gin - context มี ctx.Request.Context()
func (h *Handler) GetUser(c *gin.Context) {
    ctx := c.Request.Context()

    // Pass context ให้ service
    user, err := h.service.GetUser(ctx, c.Param("id"))
    if err != nil {
        if errors.Is(err, context.Canceled) {
            return  // client disconnected
        }
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, user)
}
```

---

## Common Patterns

### Worker Pool

```go
func workerPool(jobs <-chan Job, results chan<- Result, numWorkers int) {
    var wg sync.WaitGroup

    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            for job := range jobs {
                result := process(job)
                results <- result
            }
        }(i)
    }

    wg.Wait()
    close(results)
}

// ใช้งาน
jobs := make(chan Job, 100)
results := make(chan Result, 100)

go workerPool(jobs, results, 10)  // 10 workers

// ส่ง jobs
for _, j := range jobList {
    jobs <- j
}
close(jobs)

// รับ results
for r := range results {
    fmt.Println(r)
}
```

### Fan-out, Fan-in

```go
// Fan-out: 1 input → multiple workers
func fanOut(input <-chan int, numWorkers int) []<-chan int {
    outputs := make([]<-chan int, numWorkers)

    for i := 0; i < numWorkers; i++ {
        output := make(chan int)
        outputs[i] = output

        go func(out chan<- int) {
            for n := range input {
                out <- process(n)
            }
            close(out)
        }(output)
    }

    return outputs
}

// Fan-in: multiple inputs → 1 output
func fanIn(inputs ...<-chan int) <-chan int {
    output := make(chan int)
    var wg sync.WaitGroup

    for _, input := range inputs {
        wg.Add(1)
        go func(in <-chan int) {
            defer wg.Done()
            for n := range in {
                output <- n
            }
        }(input)
    }

    go func() {
        wg.Wait()
        close(output)
    }()

    return output
}
```

### Semaphore (Limit Concurrent)

```go
// จำกัด concurrent goroutines
type Semaphore chan struct{}

func NewSemaphore(max int) Semaphore {
    return make(chan struct{}, max)
}

func (s Semaphore) Acquire() {
    s <- struct{}{}
}

func (s Semaphore) Release() {
    <-s
}

// ใช้งาน
sem := NewSemaphore(10)  // max 10 concurrent

for _, item := range items {
    sem.Acquire()
    go func(it Item) {
        defer sem.Release()
        process(it)
    }(item)
}
```

### Rate Limiter

```go
import "time"

// Simple rate limiter with ticker
func rateLimited(items []Item, rps int) {
    ticker := time.NewTicker(time.Second / time.Duration(rps))
    defer ticker.Stop()

    for _, item := range items {
        <-ticker.C  // wait for tick
        go process(item)
    }
}

// Token bucket (golang.org/x/time/rate)
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(10, 1)  // 10 requests/sec, burst 1

for _, item := range items {
    if err := limiter.Wait(ctx); err != nil {
        return err
    }
    process(item)
}
```

---

## ตัวอย่างจาก Booking Rush

```go
// backend-booking/internal/service/booking_service.go

// Process multiple bookings concurrently
func (s *BookingService) ProcessBatch(ctx context.Context, bookingIDs []string) error {
    g, ctx := errgroup.WithContext(ctx)

    // Limit concurrent processing
    sem := make(chan struct{}, 10)

    for _, id := range bookingIDs {
        id := id  // capture

        g.Go(func() error {
            sem <- struct{}{}        // acquire
            defer func() { <-sem }() // release

            return s.processBooking(ctx, id)
        })
    }

    return g.Wait()
}

// backend-api-gateway/internal/middleware/rate_limiter.go

// Rate limiter middleware
func RateLimiter(rps int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(rps), rps)

    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatusJSON(429, gin.H{
                "error": "rate limit exceeded",
            })
            return
        }
        c.Next()
    }
}

// backend-booking/internal/worker/expiry_worker.go

// Background worker to expire bookings
func (w *ExpiryWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Println("Expiry worker stopped")
            return
        case <-ticker.C:
            if err := w.expireBookings(ctx); err != nil {
                log.Printf("Error expiring bookings: %v", err)
            }
        }
    }
}
```

---

## สรุป

| หัวข้อ | TypeScript | Go |
|--------|------------|-----|
| Concurrent unit | Worker thread (heavy) | Goroutine (lightweight) |
| Create | `new Worker()` | `go func()` |
| Communication | EventEmitter | Channel |
| Wait all | `Promise.all` | `sync.WaitGroup` |
| Wait first | `Promise.race` | `select` |
| Shared state | Locks | Mutex / Channels |
| Cancellation | AbortController | context.Context |
| Rate limiting | Custom | `time.Ticker` / `rate.Limiter` |

### Concurrency Rules

1. **"Don't communicate by sharing memory; share memory by communicating"**
   - ใช้ channel แทน shared state เมื่อเป็นไปได้

2. **ใช้ WaitGroup รอ goroutines**
   - อย่าใช้ `time.Sleep`

3. **ใช้ Context สำหรับ cancellation**
   - Pass context เป็น parameter แรกเสมอ

4. **ใช้ Mutex เมื่อจำเป็น**
   - ล็อคให้สั้นที่สุด
   - ใช้ RWMutex ถ้า read มากกว่า write

5. **ระวัง goroutine leak**
   - ตรวจสอบว่าทุก goroutine จบได้

---

## ต่อไป

- [07-http-server.md](./07-http-server.md) - HTTP Server with Gin
