```bash
booking-rush-10k-rps/
  ├── .docs/                          # Documentation
  │   ├── 01-spec.md
  │   ├── 02-task.md
  │   ├── 03-issue.md
  │   └── 04-prompt-phrase-1.md
  │
  ├── apps/                           # Microservices
  │   ├── api-gateway/
  │   │   ├── cmd/
  │   │   │   └── main.go
  │   │   ├── internal/
  │   │   │   ├── handler/
  │   │   │   ├── middleware/
  │   │   │   └── router/
  │   │   ├── go.mod
  │   │   └── Dockerfile
  │   │
  │   ├── auth-service/
  │   │   ├── cmd/
  │   │   │   └── main.go
  │   │   ├── internal/
  │   │   │   ├── domain/
  │   │   │   ├── handler/
  │   │   │   ├── repository/
  │   │   │   └── service/
  │   │   ├── go.mod
  │   │   └── Dockerfile
  │   │
  │   ├── ticket-service/
  │   │   ├── cmd/
  │   │   │   └── main.go
  │   │   ├── internal/
  │   │   │   ├── domain/
  │   │   │   ├── handler/
  │   │   │   ├── repository/
  │   │   │   └── service/
  │   │   ├── go.mod
  │   │   └── Dockerfile
  │   │
  │   ├── booking-service/            # Core service - 10k
  RPS
  │   │   ├── cmd/
  │   │   │   └── main.go
  │   │   ├── internal/
  │   │   │   ├── domain/
  │   │   │   ├── handler/
  │   │   │   ├── repository/
  │   │   │   ├── service/
  │   │   │   └── worker/             # Inventory sync worker
  │   │   ├── scripts/
  │   │   │   └── lua/                # Redis Lua scripts
  │   │   │       ├── reserve_seats.lua
  │   │   │       ├── release_seats.lua
  │   │   │       └── confirm_booking.lua
  │   │   ├── go.mod
  │   │   └── Dockerfile
  │   │
  │   └── payment-service/
  │       ├── cmd/
  │       │   └── main.go
  │       ├── internal/
  │       │   ├── domain/
  │       │   ├── handler/
  │       │   ├── repository/
  │       │   ├── service/
  │       │   └── consumer/           # Kafka consumer
  │       ├── go.mod
  │       └── Dockerfile
  │
  ├── pkg/                            # Shared packages
  │   ├── config/                     # Configuration loader 
  (Viper)
  │   ├── logger/                     # Structured logging 
  (Zap/Zerolog)
  │   ├── response/                   # Standard API response
   wrapper
  │   ├── errors/                     # Custom error types
  │   ├── middleware/                 # Common middlewares
  │   ├── database/                   # PostgreSQL connection
   (pgx)
  │   ├── redis/                      # Redis client wrapper
  │   ├── kafka/                      # Kafka 
  producer/consumer (segmentio)
  │   └── telemetry/                  # OpenTelemetry setup
  │
  ├── migrations/                     # Database migrations
  │   ├── 000001_create_tenants.up.sql
  │   ├── 000001_create_tenants.down.sql
  │   ├── 000002_create_users.up.sql
  │   └── ...
  │
  ├── scripts/                        # Build/deploy scripts
  │   ├── migrate.sh
  │   ├── seed.sh
  │   └── load-test.sh
  │
  ├── tests/                          # Integration & load 
  tests
  │   ├── integration/
  │   └── k6/                         # k6 load test scripts
  │       └── booking_load_test.js
  │
  ├── web/                            # Frontend (Phase 6)
  │   └── (Next.js 15 project)
  │
  ├── docker-compose.yml              # Local development
  ├── docker-compose.prod.yml         # Production-like
  ├── go.work                         # Go Workspace
  ├── Makefile                        # Common commands
  ├── .env.example
  ├── .gitignore
  └── README.md

  สรุปโครงสร้างหลัก

  | Directory   | Purpose                               |
  |-------------|---------------------------------------|
  | apps/       | Microservices (5 services)            |
  | pkg/        | Shared packages ใช้ร่วมกันทุก service |
  | migrations/ | Database schema migrations            |
  | scripts/    | Utility scripts                       |
  | tests/      | Integration & load tests              |
  | web/        | Next.js frontend (Phase 6)            |

  Clean Architecture ใน Service

  internal/
  ├── domain/       # Entities, interfaces (ไม่ depend อะไร)
  ├── service/      # Business logic (depend domain)
  ├── repository/   # Data access (depend domain)
  ├── handler/      # HTTP handlers (depend service)
  └── worker/       # Background workers (ถ้ามี)
```