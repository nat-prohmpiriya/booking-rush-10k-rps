# Kafka/Redpanda Guide (คู่มือ Kafka/Redpanda)

คู่มือนี้อธิบายการใช้งาน Kafka/Redpanda ในโปรเจค Booking Rush พร้อมคำอ่านศัพท์เทคนิค

---

## สารบัญ

1. [พื้นฐานและคำศัพท์](#1-พื้นฐานและคำศัพท์)
2. [Redpanda คืออะไร](#2-redpanda-คืออะไร)
3. [Architecture ในโปรเจค](#3-architecture-ในโปรเจค)
4. [Topics ที่ใช้ในระบบ](#4-topics-ที่ใช้ในระบบ)
5. [Producer และ Consumer](#5-producer-และ-consumer)
6. [Saga Pattern กับ Kafka](#6-saga-pattern-กับ-kafka)
7. [Dead Letter Queue (DLQ)](#7-dead-letter-queue-dlq)
8. [Configuration](#8-configuration)
9. [Best Practices](#9-best-practices)

---

## 1. พื้นฐานและคำศัพท์

### คำศัพท์สำคัญ

| คำศัพท์ | คำอ่าน | ความหมาย |
|---------|--------|----------|
| **Kafka** | คาฟ-ก้า | Distributed streaming platform สำหรับส่งข้อมูลระหว่าง services |
| **Redpanda** | เรด-แพน-ด้า | Kafka-compatible streaming platform ที่เร็วกว่าและใช้ resource น้อยกว่า |
| **Topic** | ท็อป-ปิก | ช่องทางหรือ "หมวดหมู่" สำหรับจัดกลุ่ม messages |
| **Partition** | พาร์-ทิ-ชั่น | การแบ่ง topic ออกเป็นส่วนๆ เพื่อ parallelism |
| **Producer** | โปร-ดิว-เซอร์ | ผู้ส่ง message ไปยัง topic |
| **Consumer** | คอน-ซู-เมอร์ | ผู้รับ message จาก topic |
| **Consumer Group** | คอน-ซู-เมอร์ กรุ๊ป | กลุ่มของ consumers ที่ทำงานร่วมกัน |
| **Offset** | ออฟ-เซ็ท | ตำแหน่งของ message ใน partition |
| **Broker** | โบร-เกอร์ | Server ที่รัน Kafka/Redpanda |
| **Message** | เมส-เสจ | ข้อมูลที่ส่งผ่าน Kafka (key + value + headers) |
| **Commit** | คอม-มิท | การยืนยันว่าได้ประมวลผล message แล้ว |
| **Rebalance** | รี-บา-ลานซ์ | การจัดสรร partitions ใหม่เมื่อ consumers เปลี่ยนแปลง |

### แนวคิดพื้นฐาน

```
┌─────────────┐     ┌─────────────────────┐     ┌─────────────┐
│  Producer   │────▶│   Topic (Kafka)     │────▶│  Consumer   │
│  (Booking)  │     │  [P0][P1][P2]       │     │  (Payment)  │
└─────────────┘     └─────────────────────┘     └─────────────┘
```

**หลักการทำงาน:**
1. **Producer** ส่ง message ไปยัง **Topic**
2. Message ถูกเก็บใน **Partition** ตาม key
3. **Consumer** อ่าน message จาก partition
4. เมื่อประมวลผลสำเร็จจะ **Commit** offset

---

## 2. Redpanda คืออะไร

### ทำไมใช้ Redpanda แทน Kafka?

**Redpanda** (เรด-แพน-ด้า) เป็น Kafka-compatible streaming platform ที่:

| คุณสมบัติ | Kafka | Redpanda |
|-----------|-------|----------|
| **ภาษา** | Java + Scala | C++ |
| **ZooKeeper** | ต้องใช้ (หรือ KRaft) | ไม่ต้องใช้ |
| **Memory** | สูง (~4GB+) | ต่ำ (~1GB) |
| **Latency** | ~5-10ms | ~1-2ms |
| **Setup** | ซับซ้อน | ง่าย |

### Configuration ในโปรเจค

```yaml
# infra/helm-values/redpanda.yaml
statefulset:
  replicas: 1

resources:
  memory:
    container:
      max: 1536Mi
    redpanda:
      memory: 1Gi
      reserveMemory: 256Mi

storage:
  persistentVolume:
    size: 5Gi

# ปิด features ที่ไม่จำเป็น
auth:
  sasl:
    enabled: false
tls:
  enabled: false
console:
  enabled: false
```

---

## 3. Architecture ในโปรเจค

### ภาพรวมการใช้ Kafka ในระบบ Booking

```
┌──────────────────────────────────────────────────────────────────┐
│                        BOOKING SYSTEM                            │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐                      ┌─────────────────────┐   │
│  │   API       │                      │   REDPANDA          │   │
│  │   Gateway   │                      │   (Kafka)           │   │
│  └──────┬──────┘                      │                     │   │
│         │                             │  ┌───────────────┐  │   │
│         ▼                             │  │ Commands      │  │   │
│  ┌─────────────┐     Produce          │  │ - reserve     │  │   │
│  │   Booking   │─────────────────────▶│  │ - payment     │  │   │
│  │   Service   │                      │  │ - confirm     │  │   │
│  └─────────────┘                      │  └───────────────┘  │   │
│                                       │                     │   │
│  ┌─────────────┐     Consume          │  ┌───────────────┐  │   │
│  │   Saga      │◀─────────────────────│  │ Events        │  │   │
│  │ Orchestrator│                      │  │ - success     │  │   │
│  └─────────────┘                      │  │ - failure     │  │   │
│         │                             │  └───────────────┘  │   │
│         │ Produce                     │                     │   │
│         ▼                             │  ┌───────────────┐  │   │
│  ┌─────────────┐     Consume          │  │ DLQ           │  │   │
│  │   Step      │◀─────────────────────│  │ (Dead Letter) │  │   │
│  │   Workers   │                      │  └───────────────┘  │   │
│  └─────────────┘                      │                     │   │
│         │                             │                     │   │
│         │ Produce (Events)            │                     │   │
│         └────────────────────────────▶│                     │   │
│                                       └─────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### Workers ที่ใช้ Kafka

| Worker | หน้าที่ | Topics ที่ Subscribe |
|--------|---------|---------------------|
| **saga-orchestrator** | จัดการ state ของ saga | Events ทั้งหมด |
| **saga-step-worker** | ทำงาน reserve/confirm/release | Commands |
| **saga-payment-worker** | ทำงาน payment/refund | Payment commands |
| **inventory-worker** | Sync inventory กับ Redis | Inventory events |

---

## 4. Topics ที่ใช้ในระบบ

### Topic Naming Convention

```
saga.{domain}.{step}.{type}
```

- **domain**: booking, payment, notification
- **step**: reserve-seats, process-payment, confirm-booking
- **type**: command หรือ event

### Command Topics (คอม-มานด์ ท็อป-ปิก)

Commands คือ "คำสั่ง" ที่ส่งจาก Orchestrator ไปยัง Workers

```go
// backend-booking/internal/saga/kafka_topics.go

// Command topics - สั่งให้ทำงาน
TopicSagaReserveSeatsCommand    = "saga.booking.reserve-seats.command"
TopicSagaProcessPaymentCommand  = "saga.booking.process-payment.command"
TopicSagaConfirmBookingCommand  = "saga.booking.confirm-booking.command"
TopicSagaSendNotificationCommand = "saga.booking.send-notification.command"

// Compensation commands - สั่งให้ย้อนกลับ
TopicSagaReleaseSeatsCommand   = "saga.booking.release-seats.command"
TopicSagaRefundPaymentCommand  = "saga.booking.refund-payment.command"
```

### Event Topics (อี-เว้นท์ ท็อป-ปิก)

Events คือ "ผลลัพธ์" ที่ส่งกลับจาก Workers

```go
// Success events
TopicSagaSeatsReservedEvent    = "saga.booking.seats-reserved.event"
TopicSagaPaymentProcessedEvent = "saga.booking.payment-processed.event"
TopicSagaBookingConfirmedEvent = "saga.booking.booking-confirmed.event"
TopicSagaNotificationSentEvent = "saga.booking.notification-sent.event"

// Failure events
TopicSagaSeatsReservationFailedEvent = "saga.booking.seats-reservation-failed.event"
TopicSagaPaymentFailedEvent          = "saga.booking.payment-failed.event"

// Lifecycle events
TopicSagaStartedEvent    = "saga.booking.started.event"
TopicSagaCompletedEvent  = "saga.booking.completed.event"
TopicSagaFailedEvent     = "saga.booking.failed.event"
TopicSagaCompensatedEvent = "saga.booking.compensated.event"
```

### Flow Diagram

```
                     COMMANDS                           EVENTS
                        │                                  │
    Orchestrator        │                                  │        Workers
    ────────────        ▼                                  ▼        ───────
         │        ┌──────────┐                      ┌──────────┐        │
         │        │ reserve- │                      │ seats-   │        │
         ├───────▶│ seats    │                      │ reserved │◀───────┤
         │        │ .command │                      │ .event   │        │
         │        └──────────┘                      └──────────┘        │
         │              │                                  ▲            │
         │              │                                  │            │
         │              └──────────────────────────────────┘            │
         │                   Worker ประมวลผลแล้วส่ง event กลับ            │
         │                                                              │
```

---

## 5. Producer และ Consumer

### Producer Implementation

```go
// pkg/kafka/producer.go

// ProducerConfig - การตั้งค่า Producer
type ProducerConfig struct {
    Brokers       []string      // รายชื่อ broker addresses
    ClientID      string        // ชื่อ client สำหรับ tracking
    MaxRetries    int           // จำนวนครั้งที่ retry
    RetryInterval time.Duration // เวลารอระหว่าง retry
    BatchSize     int           // จำนวน messages ต่อ batch
    LingerMs      int           // เวลารอก่อนส่ง batch
}

// สร้าง Producer
producer, err := kafka.NewProducer(ctx, &kafka.ProducerConfig{
    Brokers:    []string{"localhost:9092"},
    ClientID:   "booking-service",
    MaxRetries: 3,
})

// ส่ง Message แบบ Synchronous
err := producer.ProduceJSON(ctx,
    "saga.booking.reserve-seats.command",  // topic
    "booking-123",                          // key (ใช้จัด partition)
    commandData,                            // payload
    headers,                                // metadata
)

// ส่ง Message แบบ Asynchronous
producer.ProduceAsync(ctx, msg, func(err error) {
    if err != nil {
        log.Error("Failed to produce", "error", err)
    }
})
```

### Consumer Implementation

```go
// pkg/kafka/consumer.go

// ConsumerConfig - การตั้งค่า Consumer
type ConsumerConfig struct {
    Brokers          []string      // broker addresses
    GroupID          string        // consumer group ID
    Topics           []string      // topics ที่จะ subscribe
    SessionTimeout   time.Duration // timeout ก่อน rebalance
    RebalanceTimeout time.Duration // เวลาสำหรับ rebalance
    AutoCommit       bool          // commit อัตโนมัติหรือไม่
}

// สร้าง Consumer
consumer, err := kafka.NewConsumer(ctx, &kafka.ConsumerConfig{
    Brokers: []string{"localhost:9092"},
    GroupID: "saga-orchestrator",
    Topics:  []string{"saga.booking.seats-reserved.event"},
})

// Poll และประมวลผล messages
for {
    records, err := consumer.Poll(ctx)
    if err != nil {
        log.Error("Poll failed", "error", err)
        continue
    }

    for _, record := range records {
        // Extract trace context (สำหรับ distributed tracing)
        ctx, span := record.StartProcessingSpan(ctx)

        // ประมวลผล message
        if err := processRecord(ctx, record); err != nil {
            // Handle error...
        }

        span.End()
    }

    // Commit offsets
    consumer.CommitRecords(ctx, records)
}
```

### Message Structure

```go
// Message ที่ส่งผ่าน Kafka
type Message struct {
    Topic     string            // ชื่อ topic
    Key       []byte            // key สำหรับ partitioning
    Value     []byte            // payload (มักเป็น JSON)
    Headers   map[string]string // metadata
    Timestamp time.Time         // เวลาที่สร้าง
}

// Record ที่ได้จากการ consume
type Record struct {
    Topic     string
    Partition int32             // partition number
    Offset    int64             // offset ใน partition
    Key       []byte
    Value     []byte
    Headers   map[string]string
    Timestamp time.Time
}
```

---

## 6. Saga Pattern กับ Kafka

### Saga คืออะไร?

**Saga** (ซา-ก้า) เป็น pattern สำหรับจัดการ distributed transactions โดยแบ่งเป็นขั้นตอนย่อยๆ ถ้าขั้นตอนใดล้มเหลว จะทำ compensation (ย้อนกลับ)

### Saga Messages

```go
// SagaCommand - คำสั่งสำหรับแต่ละ step
type SagaCommand struct {
    MessageID      string                 // ID ของ message
    CorrelationID  string                 // Saga instance ID
    SagaID         string                 // เหมือน CorrelationID
    SagaName       string                 // ชื่อ saga (เช่น "booking-saga")
    StepName       string                 // ชื่อ step (เช่น "reserve-seats")
    StepIndex      int                    // ลำดับของ step
    IdempotencyKey string                 // key สำหรับป้องกัน duplicate
    TimeoutAt      time.Time              // เวลา timeout
    RetryCount     int                    // จำนวนครั้งที่ retry แล้ว
    MaxRetries     int                    // retry สูงสุด
    Data           map[string]interface{} // ข้อมูลสำหรับ step
}

// SagaEvent - ผลลัพธ์จาก step
type SagaEvent struct {
    MessageID    string
    SagaID       string
    StepName     string
    Success      bool                   // สำเร็จหรือไม่
    ErrorMessage string                 // ข้อความ error (ถ้ามี)
    ErrorCode    string                 // code ของ error
    Data         map[string]interface{} // ข้อมูลผลลัพธ์
    StartedAt    time.Time              // เวลาเริ่ม
    FinishedAt   time.Time              // เวลาจบ
    Duration     time.Duration          // ระยะเวลา
}
```

### Saga Flow ในระบบ Booking

```
┌────────────────────────────────────────────────────────────────────────┐
│                        BOOKING SAGA FLOW                               │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  1. API สร้าง Booking        2. ส่ง Command               3. Worker    │
│  ┌─────────────────┐        ┌─────────────────┐        ┌────────────┐ │
│  │  POST /booking  │───────▶│ reserve-seats   │───────▶│ Step       │ │
│  │  /reserve       │        │ .command        │        │ Worker     │ │
│  └─────────────────┘        └─────────────────┘        └─────┬──────┘ │
│                                                              │        │
│  6. Orchestrator            5. Event                   4. ทำงาน      │
│  ┌─────────────────┐        ┌─────────────────┐            │         │
│  │  อ่าน Event     │◀───────│ seats-reserved  │◀───────────┘         │
│  │  ส่ง Command    │        │ .event          │                      │
│  │  ถัดไป          │        └─────────────────┘                      │
│  └────────┬────────┘                                                 │
│           │                                                          │
│           ▼                                                          │
│  7. ถ้าสำเร็จ: ส่ง confirm-booking.command                            │
│     ถ้าล้มเหลว: ส่ง release-seats.command (compensation)              │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

### Saga Orchestrator

```go
// Orchestrator ทำหน้าที่:
// 1. รับ events จาก workers
// 2. อัปเดต saga state
// 3. ตัดสินใจว่าจะทำอะไรต่อ
// 4. ส่ง command ถัดไป หรือ compensation

// เมื่อได้รับ success event
func (o *Orchestrator) handleSuccessEvent(event *SagaEvent) {
    // อัปเดต state
    saga.CurrentStep++
    saga.Status = "in_progress"

    // ถ้ายังมี step ถัดไป
    if saga.CurrentStep < len(saga.Steps) {
        // ส่ง command ถัดไป
        nextCommand := createCommand(saga, saga.Steps[saga.CurrentStep])
        producer.SendCommand(ctx, nextCommand)
    } else {
        // สำเร็จทั้งหมด
        saga.Status = "completed"
        producer.SendSagaCompletedEvent(ctx, saga)
    }
}

// เมื่อได้รับ failure event
func (o *Orchestrator) handleFailureEvent(event *SagaEvent) {
    // เริ่ม compensation (ย้อนกลับ)
    saga.Status = "compensating"

    // ส่ง compensation commands ย้อนกลับ
    for i := saga.CurrentStep - 1; i >= 0; i-- {
        compensationCmd := createCompensationCommand(saga, saga.Steps[i])
        producer.SendCompensationCommand(ctx, compensationCmd)
    }
}
```

---

## 7. Dead Letter Queue (DLQ)

### DLQ คืออะไร?

**Dead Letter Queue** (เด็ด เล็ต-เตอร์ คิว) หรือ DLQ คือที่เก็บ messages ที่ประมวลผลไม่สำเร็จหลังจาก retry หลายครั้ง

```
┌────────────┐    retry 1    ┌────────────┐    retry 2    ┌────────────┐
│  Message   │──────────────▶│  Worker    │──────────────▶│  Worker    │
│            │    fail       │            │    fail       │            │
└────────────┘               └────────────┘               └─────┬──────┘
                                                                │
                                                          retry 3 fail
                                                                │
                                                                ▼
                                                         ┌────────────┐
                                                         │    DLQ     │
                                                         │ (เก็บไว้   │
                                                         │  ตรวจสอบ)  │
                                                         └────────────┘
```

### DLQ Implementation

```go
// backend-booking/internal/saga/dlq_handler.go

const (
    DLQTopic = "saga.booking.dlq"  // topic สำหรับ DLQ
    MaxRetryAttempts = 3          // retry สูงสุดก่อนส่ง DLQ
)

// DLQMessage - โครงสร้างข้อมูลใน DLQ
type DLQMessage struct {
    ID             string                 // ID ของ DLQ message
    OriginalTopic  string                 // topic ต้นทาง
    SagaID         string                 // saga ID
    MessageKey     string                 // key ของ message
    MessageValue   map[string]interface{} // payload
    ErrorMessage   string                 // ข้อความ error
    ErrorCode      string                 // code
    RetryCount     int                    // retry กี่ครั้ง
    FirstFailedAt  time.Time              // ครั้งแรกที่ fail
    LastFailedAt   time.Time              // ครั้งล่าสุดที่ fail
}

// HandleFailedMessage - ส่ง message ไป DLQ
func (h *DLQHandler) HandleFailedMessage(ctx context.Context,
    originalTopic string,
    messageKey string,
    messageValue []byte,
    err error,
    retryCount int) error {

    // Log alert
    log.Error("[ALERT] Message failed, sending to DLQ",
        "topic", originalTopic,
        "retry_count", retryCount,
        "error", err)

    dlqMsg := &DLQMessage{
        ID:            generateID(),
        OriginalTopic: originalTopic,
        MessageKey:    messageKey,
        ErrorMessage:  err.Error(),
        RetryCount:    retryCount,
        LastFailedAt:  time.Now(),
    }

    // บันทึกลง PostgreSQL
    h.store.SaveDeadLetter(ctx, dlqMsg)

    // ส่งไป Kafka DLQ topic (สำหรับ alerting)
    h.producer.Publish(ctx, DLQTopic, messageKey, dlqMsg)

    return nil
}
```

### Non-Retryable Errors

บาง errors ไม่ควร retry เพราะจะ fail เหมือนเดิม:

```go
// isNonRetryableError - ตรวจสอบว่า error ควร retry หรือไม่
func isNonRetryableError(err error) bool {
    nonRetryablePatterns := []string{
        "invalid request",   // request ผิด format
        "validation failed", // ข้อมูลไม่ถูกต้อง
        "not found",         // ไม่พบข้อมูล
        "unauthorized",      // ไม่มีสิทธิ์
        "forbidden",         // ถูกห้าม
        "duplicate",         // ซ้ำ
        "already exists",    // มีอยู่แล้ว
    }

    for _, pattern := range nonRetryablePatterns {
        if containsIgnoreCase(err.Error(), pattern) {
            return true  // ไม่ต้อง retry
        }
    }
    return false  // retry ได้
}
```

### DLQ Stats

```go
// ดูสถิติ DLQ
type DLQStats struct {
    TotalMessages      int64            // จำนวนทั้งหมด
    UnprocessedCount   int64            // ยังไม่ได้จัดการ
    ProcessedCount     int64            // จัดการแล้ว
    OldestMessageTime  time.Time        // message เก่าสุด
    ByTopic            map[string]int64 // จำนวนแยกตาม topic
}
```

---

## 8. Configuration

### Environment Variables

```bash
# Kafka/Redpanda connection
KAFKA_BROKERS=localhost:9092
KAFKA_CLIENT_ID=booking-service

# Producer settings
KAFKA_BATCH_SIZE=100
KAFKA_LINGER_MS=5
KAFKA_MAX_RETRIES=3
KAFKA_RETRY_INTERVAL=2s

# Consumer settings
KAFKA_CONSUMER_GROUP=saga-orchestrator
KAFKA_SESSION_TIMEOUT=30s
KAFKA_REBALANCE_TIMEOUT=60s
```

### Code Configuration

```go
// pkg/config/config.go

type KafkaConfig struct {
    Brokers          []string      `mapstructure:"brokers"`
    ClientID         string        `mapstructure:"client_id"`
    ConsumerGroup    string        `mapstructure:"consumer_group"`
    MaxRetries       int           `mapstructure:"max_retries"`
    RetryInterval    time.Duration `mapstructure:"retry_interval"`
    SessionTimeout   time.Duration `mapstructure:"session_timeout"`
    RebalanceTimeout time.Duration `mapstructure:"rebalance_timeout"`
}
```

---

## 9. Best Practices

### 1. Message Key Selection

```go
// ใช้ key ที่เกี่ยวข้องกับ business logic
// เพื่อให้ messages ของ entity เดียวกันอยู่ partition เดียวกัน

// ดี: ใช้ booking_id เป็น key
producer.ProduceJSON(ctx, topic, bookingID, data, headers)

// ไม่ดี: ใช้ random key (messages จะกระจายแบบสุ่ม)
producer.ProduceJSON(ctx, topic, uuid.New().String(), data, headers)
```

### 2. Idempotency (ไอ-เดม-โพ-เท็น-ซี่)

ทำให้การประมวลผลซ้ำได้ผลลัพธ์เหมือนเดิม:

```go
// สร้าง idempotency key
idempotencyKey := fmt.Sprintf("%s:%s", sagaID, stepName)

// ตรวจสอบก่อนประมวลผล
if h.hasProcessed(ctx, idempotencyKey) {
    return nil // ข้าม เพราะทำไปแล้ว
}

// ประมวลผล
result := processStep(ctx, data)

// บันทึกว่าทำแล้ว
h.markProcessed(ctx, idempotencyKey, result)
```

### 3. Error Handling

```go
func processMessage(ctx context.Context, record *kafka.Record) error {
    // 1. Parse message
    var cmd SagaCommand
    if err := json.Unmarshal(record.Value, &cmd); err != nil {
        // Parse error = non-retryable
        return fmt.Errorf("invalid message format: %w", err)
    }

    // 2. Validate
    if cmd.SagaID == "" {
        // Validation error = non-retryable
        return fmt.Errorf("missing saga_id")
    }

    // 3. Process with timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    if err := executeStep(ctx, &cmd); err != nil {
        // Execution error = might be retryable
        return fmt.Errorf("step execution failed: %w", err)
    }

    return nil
}
```

### 4. Graceful Shutdown

```go
func main() {
    // สร้าง consumer
    consumer, _ := kafka.NewConsumer(ctx, config)
    defer consumer.Close()

    // รอ signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Info("Shutting down...")

        // หยุด consumer อย่างสุภาพ
        consumer.Close()
    }()

    // Process loop
    for {
        records, err := consumer.Poll(ctx)
        if err != nil {
            if ctx.Err() != nil {
                break // shutdown
            }
            continue
        }
        // ...
    }
}
```

### 5. Monitoring

ควรติดตาม metrics เหล่านี้:

| Metric | ความหมาย |
|--------|----------|
| **Consumer lag** | จำนวน messages ที่ยังไม่ได้ประมวลผล |
| **Message rate** | จำนวน messages ต่อวินาที |
| **Processing time** | เวลาประมวลผลต่อ message |
| **Error rate** | อัตราส่วน errors |
| **DLQ count** | จำนวน messages ใน DLQ |
| **Rebalance frequency** | ความถี่ที่เกิด rebalance |

---

## สรุป

### เมื่อไหร่ควรใช้ Kafka/Redpanda?

| Use Case | เหมาะสม? | เหตุผล |
|----------|----------|--------|
| Service-to-service async | ใช่ | Decoupling, reliability |
| Event sourcing | ใช่ | Message retention |
| Saga orchestration | ใช่ | Distributed transactions |
| Real-time sync | ไม่ | ใช้ Redis หรือ direct call |
| Simple queue | ไม่เสมอไป | พิจารณา Redis หรือ RabbitMQ |

### Key Takeaways

1. **Redpanda** = Kafka-compatible แต่เบากว่า
2. **Topic** = ช่องทางสื่อสาร แบ่งเป็น Commands และ Events
3. **Producer** = ส่ง message, **Consumer** = รับ message
4. **Saga Pattern** = จัดการ distributed transactions ผ่าน Kafka
5. **DLQ** = ที่เก็บ messages ที่ fail หลัง retry
6. **Idempotency** = สำคัญมากสำหรับ retry scenarios
