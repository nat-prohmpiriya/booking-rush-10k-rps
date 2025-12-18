# Architecture Diagrams

Mermaid diagrams สำหรับระบบ Booking Rush - สร้างจาก codebase จริง

## วิธีดู Diagram

1. **GitHub/GitLab** - render อัตโนมัติ
2. **VS Code** - ติดตั้ง extension "Markdown Preview Mermaid Support"
3. **Online** - ไปที่ [mermaid.live](https://mermaid.live) แล้ว paste code
4. **Export PNG/SVG** - ใช้ mermaid.live แล้วกด export

---

## 1. Virtual Queue - Fair Access Control

### 1.1 Overview Flowchart

```mermaid
flowchart TB
    subgraph Client["👤 Client"]
        User[User]
        SSEClient[SSE Client]
    end

    subgraph APIGateway["API Gateway :8080"]
        GW[Reverse Proxy<br/>Rate Limit + JWT]
    end

    subgraph BookingService["Booking Service :8083"]
        QueueHandler[Queue Handler]
        BookingHandler[Booking Handler]
        QueueReleaseWorker[Queue Release Worker<br/>runs every 1 second]
    end

    subgraph Redis["Redis"]
        SortedSet[("queue:{event_id}<br/>Sorted Set<br/>score = timestamp")]
        QueuePass[("queue:pass:{event_id}:{user_id}<br/>JWT String, 5-min TTL")]
        Config[("queue:config:{event_id}<br/>Hash: max_concurrent")]
        PubSub[["Pub/Sub Channel<br/>queue:pass:{event_id}:{user_id}"]]
    end

    subgraph PostgreSQL["PostgreSQL"]
        BookingDB[(booking_db)]
    end

    %% Flow 1: Join Queue
    User -->|"1. POST /queue/join"| GW
    GW -->|proxy| QueueHandler
    QueueHandler -->|"ZADD score=timestamp"| SortedSet
    QueueHandler -->|"position + wait time"| User

    %% Flow 2: SSE Connection
    User -->|"2. GET /queue/position/{event_id}/stream"| GW
    GW -->|proxy| QueueHandler
    QueueHandler -->|"SUBSCRIBE"| PubSub
    PubSub -.->|"keep alive, wait"| SSEClient

    %% Flow 3: Queue Release Worker
    QueueReleaseWorker -->|"a. GET max_concurrent"| Config
    QueueReleaseWorker -->|"b. COUNT active passes"| QueuePass
    QueueReleaseWorker -->|"c. ZPOPMIN (FIFO)"| SortedSet
    QueueReleaseWorker -->|"d. SET JWT pass"| QueuePass
    QueueReleaseWorker -->|"e. PUBLISH"| PubSub
    PubSub -->|"f. pass received instantly"| SSEClient

    %% Flow 4: Booking with Queue Pass
    User -->|"4. POST /bookings/reserve<br/>+ JWT Queue Pass"| GW
    GW -->|proxy| BookingHandler
    BookingHandler -->|"validate JWT"| QueuePass
    BookingHandler -->|"reserve seats"| Redis
    BookingHandler -->|"create booking"| BookingDB
    BookingHandler -->|"DELETE pass (one-time)"| QueuePass

    %% Styling
    classDef redis fill:#dc382d,color:#fff
    classDef postgres fill:#336791,color:#fff
    classDef service fill:#2d6a4f,color:#fff
    classDef gateway fill:#1a73e8,color:#fff
    classDef worker fill:#f59e0b,color:#000

    class SortedSet,QueuePass,Config,PubSub redis
    class BookingDB postgres
    class QueueHandler,BookingHandler service
    class GW gateway
    class QueueReleaseWorker worker
```

### 1.2 Sequence Diagram (ละเอียด)

```mermaid
sequenceDiagram
    autonumber
    participant U as User
    participant GW as API Gateway<br/>:8080
    participant QH as Queue Handler<br/>(Booking Service :8083)
    participant BH as Booking Handler<br/>(Booking Service :8083)
    participant W as Queue Release Worker
    participant R as Redis

    Note over U,R: Phase 1: Join Queue
    U->>GW: POST /queue/join
    GW->>QH: proxy request
    QH->>R: ZADD queue:{event_id} {timestamp} {user_id}
    R-->>QH: OK
    QH-->>U: {position: 150, estimated_wait: "2 min"}

    Note over U,R: Phase 2: SSE Connection (keep alive)
    U->>GW: GET /queue/position/{event_id}/stream
    GW->>QH: proxy (SSE)
    QH->>R: SUBSCRIBE queue:pass:{event_id}:{user_id}
    Note over QH,U: Connection kept alive...

    Note over W,R: Phase 3: Queue Release Worker (every 1s)
    loop Every 1 second
        W->>R: GET queue:config:{event_id}
        R-->>W: {max_concurrent: 100}
        W->>R: COUNT keys queue:pass:{event_id}:*
        R-->>W: active_passes = 80
        Note over W: available = 100 - 80 = 20
        W->>R: ZPOPMIN queue:{event_id} 20
        R-->>W: [user_1, user_2, ... user_20]
        loop For each user
            W->>W: Generate JWT Queue Pass (5-min TTL)
            W->>R: SET queue:pass:{event_id}:{user_id} {jwt}
            W->>R: PUBLISH queue:pass:{event_id}:{user_id}
        end
    end

    Note over U,R: User receives pass via SSE
    R-->>QH: PUBLISH message received
    QH-->>U: SSE event: {queue_pass: "eyJhbG..."}

    Note over U,R: Phase 4: Booking with Queue Pass
    U->>GW: POST /bookings/reserve + Header: X-Queue-Pass
    GW->>BH: proxy request
    BH->>R: GET queue:pass:{event_id}:{user_id}
    R-->>BH: JWT pass (validate: not expired)
    BH->>R: Lua: reserve_seats.lua (atomic)
    R-->>BH: {success, remaining_seats}
    BH->>BH: Create booking in PostgreSQL
    BH->>R: DEL queue:pass:{event_id}:{user_id}
    BH-->>U: 201 {booking_id: "..."}
```

### 1.3 Data Structures

| Key Pattern | Type | Description | TTL |
|-------------|------|-------------|-----|
| `queue:{event_id}` | Sorted Set | Waiting users, score = join timestamp | - |
| `queue:pass:{event_id}:{user_id}` | String | JWT queue pass | 5 min |
| `queue:config:{event_id}` | Hash | max_concurrent, pass_ttl | - |

### 1.4 Performance Comparison

| Method | Queries/sec | Efficiency |
|--------|-------------|------------|
| Polling (every 500ms) | 2,000,000 | ❌ High Load |
| Pub/Sub + SSE | 10,000 | ✅ Efficient |

---

## 2. Overall System Architecture

```mermaid
flowchart TB
    subgraph Frontend["Frontend"]
        Web[Next.js :3000]
    end

    subgraph Gateway["API Gateway :8080"]
        GW[Gin Server<br/>Rate Limit / JWT / Proxy]
    end

    subgraph Services["Microservices (Go + Gin)"]
        Auth[Auth Service :8081]
        Ticket[Ticket Service :8082]
        Booking[Booking Service :8083]
        Payment[Payment Service :8084]
    end

    subgraph Workers["Background Workers"]
        QueueWorker[Queue Release Worker]
        SagaOrch[Saga Orchestrator]
        SagaStep[Saga Step Worker]
        SeatRelease[Seat Release Worker]
        Inventory[Inventory Worker]
    end

    subgraph DataLayer["Data Layer"]
        PG[(PostgreSQL<br/>auth_db / ticket_db<br/>booking_db / payment_db)]
        Redis[(Redis<br/>Lua Scripts / Pub-Sub<br/>Rate Limit / Queue)]
        Kafka[/Redpanda<br/>Kafka-compatible/]
    end

    subgraph Observability["Observability"]
        Tempo[Tempo<br/>Traces]
        Prometheus[Prometheus<br/>Metrics]
        Loki[Loki<br/>Logs]
        Grafana[Grafana<br/>Dashboards]
    end

    Web --> GW
    GW --> Auth
    GW --> Ticket
    GW --> Booking
    GW --> Payment

    Auth --> PG
    Ticket --> PG
    Ticket --> Redis
    Booking --> PG
    Booking --> Redis
    Booking --> Kafka
    Payment --> PG
    Payment --> Kafka

    QueueWorker --> Redis
    SagaOrch --> Kafka
    SagaOrch --> PG
    SagaStep --> Kafka
    SagaStep --> Redis
    SeatRelease --> Redis
    Inventory --> Redis
    Inventory --> PG

    Kafka --> SagaOrch
    Kafka --> SagaStep

    Services --> Tempo
    Services --> Prometheus
    Workers --> Prometheus
    Tempo --> Grafana
    Prometheus --> Grafana
    Loki --> Grafana
```

---

## 3. Fast Path - Seat Reservation

```mermaid
sequenceDiagram
    autonumber
    participant U as User
    participant GW as API Gateway
    participant B as Booking Service
    participant R as Redis
    participant PG as PostgreSQL
    participant K as Kafka

    Note over U,K: Fast Path: < 100ms latency

    U->>GW: POST /bookings/reserve
    GW->>GW: Rate Limit Check
    GW->>GW: JWT Validation
    GW->>B: Proxy Request

    opt If REQUIRE_QUEUE_PASS=true
        B->>R: Validate Queue Pass
        R-->>B: Valid / Invalid
    end

    B->>R: Lua: reserve_seats.lua (ATOMIC)
    Note over R: Single-threaded execution:<br/>1. GET availability<br/>2. CHECK quantity<br/>3. CHECK user limit<br/>4. DECRBY seats<br/>5. SET reservation + TTL
    R-->>B: {1, remaining_seats, user_total}

    B->>PG: INSERT booking (async)
    B->>K: PUBLISH booking.reserved

    B-->>U: 201 {booking_id, status: "RESERVED"}

    Note over U,K: Reservation expires in 10 minutes<br/>if not confirmed
```

---

## 4. Post-Payment Saga

```mermaid
sequenceDiagram
    autonumber
    participant Stripe as Stripe Webhook
    participant P as Payment Service
    participant K as Kafka
    participant SO as Saga Orchestrator
    participant PG as PostgreSQL
    participant SW as Saga Step Worker
    participant R as Redis
    participant B as Booking DB

    Note over Stripe,B: Post-Payment Saga (Orchestration Pattern)

    Stripe->>P: POST /webhooks/stripe (payment.success)
    P->>K: PUBLISH payment-success

    K->>SO: Consume payment-success
    SO->>PG: INSERT saga_instances (status: PENDING)
    SO->>K: PUBLISH confirm-booking command

    K->>SW: Consume confirm-booking
    SW->>R: Lua: confirm_booking.lua
    Note over R: 1. Update status: reserved → confirmed<br/>2. Remove TTL (permanent)
    R-->>SW: {1, "CONFIRMED"}
    SW->>B: UPDATE booking SET status='CONFIRMED'
    SW->>K: PUBLISH booking-confirmed event

    K->>SO: Consume booking-confirmed
    SO->>PG: UPDATE saga_instances SET status='COMPLETED'

    opt Send Notification
        SO->>K: PUBLISH send-notification command
    end
```

---

## 5. Redis Lua Scripts - Atomic Operations

```mermaid
flowchart LR
    subgraph Problem["❌ Race Condition (without Lua)"]
        direction TB
        P1[Thread 1: GET seats = 1]
        P2[Thread 2: GET seats = 1]
        P3[Thread 1: DECRBY 1 → 0]
        P4[Thread 2: DECRBY 1 → -1]
        P5[Result: OVERSOLD!]
        P1 --> P2 --> P3 --> P4 --> P5
    end

    subgraph Solution["✅ Atomic Lua Script"]
        direction TB
        S1[Lua Script Starts]
        S2[GET seats = 1]
        S3[CHECK quantity <= seats?]
        S4[DECRBY 1 → 0]
        S5[SET reservation]
        S6[Return success]
        S1 --> S2 --> S3 --> S4 --> S5 --> S6

        S7[Single Thread<br/>No Interruption]
        S7 -.- S1
    end

    Problem -.->|"Solution"| Solution
```

### Lua Scripts Reference

| Script | File | Purpose |
|--------|------|---------|
| reserve_seats.lua | `scripts/lua/reserve_seats.lua` | Atomic seat reservation |
| confirm_booking.lua | `scripts/lua/confirm_booking.lua` | Mark confirmed, remove TTL |
| release_seats.lua | `scripts/lua/release_seats.lua` | Return seats to inventory |

---

## 6. Service Ports Reference

| Service | Port | Technology | Responsibility |
|---------|------|------------|----------------|
| Frontend | 3000 | Next.js | Web UI |
| API Gateway | 8080 | Go + Gin | Rate limit, JWT, Proxy |
| Auth Service | 8081 | Go + Gin | Authentication, JWT |
| Ticket Service | 8082 | Go + Gin | Events, Zones catalog |
| Booking Service | 8083 | Go + Gin | Reservations, Queue, Saga |
| Payment Service | 8084 | Go + Gin | Stripe integration |

---

## 7. Key File Locations

```
backend-booking/
├── main.go                                    # Entry point
├── cmd/
│   ├── queue-worker/main.go                   # Queue Release Worker
│   ├── saga-orchestrator/main.go              # Saga Orchestrator
│   ├── saga-step-worker/main.go               # Saga Step Worker
│   └── seat-release-worker/main.go            # Seat Release Worker
├── internal/
│   ├── handler/
│   │   ├── booking_handler.go                 # /bookings/* endpoints
│   │   └── queue_handler.go                   # /queue/* endpoints
│   ├── service/
│   │   ├── booking_service.go                 # Booking business logic
│   │   └── queue_service.go                   # Queue business logic
│   ├── repository/
│   │   ├── redis_reservation_repository.go   # Redis Lua scripts
│   │   └── redis_queue_repository.go         # Queue Redis operations
│   ├── worker/
│   │   └── queue_release_worker.go           # Worker implementation
│   └── saga/
│       └── booking_saga.go                    # Saga definition

scripts/lua/
├── reserve_seats.lua                          # Atomic reservation
├── confirm_booking.lua                        # Confirm after payment
└── release_seats.lua                          # Release expired
```
