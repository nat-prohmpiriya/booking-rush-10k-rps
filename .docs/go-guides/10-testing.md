# 10 - Testing (การทดสอบ)

## สารบัญ

1. [พื้นฐาน Testing](#พื้นฐาน-testing)
2. [Table-Driven Tests](#table-driven-tests)
3. [Assertions](#assertions)
4. [Mocking](#mocking)
5. [HTTP Testing](#http-testing)
6. [Benchmarks](#benchmarks)
7. [Test Coverage](#test-coverage)

---

## พื้นฐาน Testing

### TypeScript - Jest/Vitest

```typescript
// sum.ts
export function sum(a: number, b: number): number {
    return a + b
}

// sum.test.ts
import { describe, it, expect } from 'vitest'
import { sum } from './sum'

describe('sum', () => {
    it('should add two numbers', () => {
        expect(sum(1, 2)).toBe(3)
    })

    it('should handle negative numbers', () => {
        expect(sum(-1, 1)).toBe(0)
    })
})
```

### Go - testing package

```go
// sum.go
package math

func Sum(a, b int) int {
    return a + b
}

// sum_test.go (ต้องลงท้ายด้วย _test.go)
package math

import "testing"

func TestSum(t *testing.T) {
    result := Sum(1, 2)
    expected := 3

    if result != expected {
        t.Errorf("Sum(1, 2) = %d; want %d", result, expected)
    }
}

func TestSumNegative(t *testing.T) {
    result := Sum(-1, 1)
    expected := 0

    if result != expected {
        t.Errorf("Sum(-1, 1) = %d; want %d", result, expected)
    }
}
```

### รัน Tests

```bash
# รัน tests ทั้งหมดใน package ปัจจุบัน
go test

# รัน tests ทุก packages
go test ./...

# Verbose output
go test -v

# รัน test function เฉพาะ
go test -run TestSum

# รัน test ที่ match pattern
go test -run "TestSum.*"

# รันพร้อม race detector
go test -race
```

### Test Function Rules

```go
// 1. File ต้องลงท้ายด้วย _test.go
//    user_test.go ✓
//    user_tests.go ✗

// 2. Function ต้องขึ้นต้นด้วย Test
//    func TestXxx(t *testing.T) ✓
//    func testXxx(t *testing.T) ✗

// 3. รับ parameter *testing.T
func TestExample(t *testing.T) {
    // test code
}
```

### t.Error vs t.Fatal

```go
func TestExample(t *testing.T) {
    // t.Error - รายงาน error แต่ test ยังทำงานต่อ
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
    // code นี้ยังทำงาน

    // t.Fatal - รายงาน error และหยุด test ทันที
    if critical != nil {
        t.Fatalf("critical error: %v", critical)
    }
    // code นี้ไม่ทำงาน

    // t.Skip - ข้าม test
    if !hasDatabase {
        t.Skip("skipping database test")
    }
}
```

---

## Table-Driven Tests

Pattern ยอดนิยมใน Go - ทดสอบหลาย cases ในที่เดียว

### TypeScript - Jest each

```typescript
describe('sum', () => {
    it.each([
        [1, 2, 3],
        [0, 0, 0],
        [-1, 1, 0],
        [100, 200, 300],
    ])('sum(%i, %i) = %i', (a, b, expected) => {
        expect(sum(a, b)).toBe(expected)
    })
})
```

### Go - Table-Driven

```go
func TestSum(t *testing.T) {
    // Define test cases
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 1, 2, 3},
        {"zeros", 0, 0, 0},
        {"negative numbers", -1, 1, 0},
        {"large numbers", 100, 200, 300},
    }

    // Run test cases
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Sum(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Sum(%d, %d) = %d; want %d",
                    tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

### t.Run - Subtests

```go
func TestUser(t *testing.T) {
    // Setup
    db := setupTestDB()
    defer db.Close()

    t.Run("Create", func(t *testing.T) {
        // test create
    })

    t.Run("Get", func(t *testing.T) {
        // test get
    })

    t.Run("Update", func(t *testing.T) {
        // test update
    })

    t.Run("Delete", func(t *testing.T) {
        // test delete
    })
}

// รันเฉพาะ subtest
// go test -run TestUser/Create
```

---

## Assertions

### Go Standard - Manual Comparison

```go
func TestExample(t *testing.T) {
    got := DoSomething()
    want := "expected"

    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}
```

### testify - Assertion Library (แนะนำ)

```bash
go get github.com/stretchr/testify
```

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWithAssert(t *testing.T) {
    // assert - รายงาน error แต่ไปต่อ
    assert.Equal(t, 3, Sum(1, 2))
    assert.NotEqual(t, 0, Sum(1, 2))
    assert.True(t, IsValid())
    assert.False(t, IsEmpty())
    assert.Nil(t, err)
    assert.NotNil(t, user)
    assert.Contains(t, "hello world", "hello")
    assert.Len(t, items, 5)
    assert.Empty(t, slice)
    assert.Error(t, err)
    assert.NoError(t, err)

    // require - หยุดทันทีถ้า fail
    require.NoError(t, err)
    require.NotNil(t, user)
    // code ต่อจากนี้ไม่ทำงานถ้า require fail
}

// Struct comparison
func TestUser(t *testing.T) {
    got := GetUser("123")
    want := &User{ID: "123", Name: "John"}

    assert.Equal(t, want, got)  // deep compare
}
```

---

## Mocking

### TypeScript - Jest mocks

```typescript
// Jest mock
jest.mock('./database')
const mockDB = database as jest.Mocked<typeof database>
mockDB.findUser.mockResolvedValue({ id: '123', name: 'John' })
```

### Go - Interface + Mock Implementation

```go
// repository.go
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    Create(ctx context.Context, user *User) error
}

// mock_repository.go (ใน _test.go หรือ package mock)
type MockUserRepository struct {
    Users map[string]*User
    Error error
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    if m.Error != nil {
        return nil, m.Error
    }
    return m.Users[id], nil
}

func (m *MockUserRepository) Create(ctx context.Context, user *User) error {
    if m.Error != nil {
        return m.Error
    }
    m.Users[user.ID] = user
    return nil
}

// service_test.go
func TestGetUser(t *testing.T) {
    // Setup mock
    mockRepo := &MockUserRepository{
        Users: map[string]*User{
            "123": {ID: "123", Name: "John"},
        },
    }

    // Create service with mock
    service := NewUserService(mockRepo)

    // Test
    user, err := service.GetUser(context.Background(), "123")
    require.NoError(t, err)
    assert.Equal(t, "John", user.Name)
}

func TestGetUserNotFound(t *testing.T) {
    mockRepo := &MockUserRepository{
        Users: map[string]*User{},
    }

    service := NewUserService(mockRepo)

    user, err := service.GetUser(context.Background(), "999")
    assert.Error(t, err)
    assert.Nil(t, user)
}
```

### testify/mock

```go
import "github.com/stretchr/testify/mock"

type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func TestWithTestifyMock(t *testing.T) {
    mockRepo := new(MockUserRepository)

    // Setup expectations
    mockRepo.On("FindByID", mock.Anything, "123").
        Return(&User{ID: "123", Name: "John"}, nil)

    service := NewUserService(mockRepo)
    user, err := service.GetUser(context.Background(), "123")

    require.NoError(t, err)
    assert.Equal(t, "John", user.Name)

    // Verify expectations
    mockRepo.AssertExpectations(t)
}
```

### gomock (Google)

```bash
go install github.com/golang/mock/mockgen@latest
```

```go
//go:generate mockgen -source=repository.go -destination=mock_repository.go -package=mocks

// Generate mock
// mockgen -source=repository.go -destination=mocks/mock_repository.go -package=mocks

// ใช้งาน
func TestWithGoMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockRepo.EXPECT().
        FindByID(gomock.Any(), "123").
        Return(&User{ID: "123", Name: "John"}, nil)

    service := NewUserService(mockRepo)
    user, err := service.GetUser(context.Background(), "123")

    require.NoError(t, err)
    assert.Equal(t, "John", user.Name)
}
```

---

## HTTP Testing

### TypeScript - Supertest

```typescript
import request from 'supertest'
import app from './app'

describe('GET /users/:id', () => {
    it('returns user', async () => {
        const res = await request(app)
            .get('/users/123')
            .expect(200)

        expect(res.body.name).toBe('John')
    })
})
```

### Go - httptest

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func TestGetUser(t *testing.T) {
    // Setup
    gin.SetMode(gin.TestMode)
    router := setupRouter()  // สร้าง router

    // Create request
    req, _ := http.NewRequest("GET", "/users/123", nil)
    w := httptest.NewRecorder()

    // Execute
    router.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "John")
}

func TestCreateUser(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := setupRouter()

    // JSON body
    body := `{"name": "John", "email": "john@example.com"}`
    req, _ := http.NewRequest("POST", "/users", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, 201, w.Code)

    // Parse response
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "John", response["name"])
}

func TestWithAuth(t *testing.T) {
    router := setupRouter()

    req, _ := http.NewRequest("GET", "/protected", nil)
    req.Header.Set("Authorization", "Bearer valid-token")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
}
```

### Test Server

```go
func TestExternalAPI(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"status": "ok"}`))
    }))
    defer server.Close()

    // Use server.URL in client
    client := NewAPIClient(server.URL)
    resp, err := client.CheckStatus()

    require.NoError(t, err)
    assert.Equal(t, "ok", resp.Status)
}
```

---

## Benchmarks

### TypeScript - ไม่มี built-in

```typescript
// ใช้ external library หรือ manual timing
console.time('operation')
for (let i = 0; i < 1000000; i++) {
    doSomething()
}
console.timeEnd('operation')
```

### Go - Built-in Benchmarks

```go
// sum_test.go
func BenchmarkSum(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Sum(1, 2)
    }
}

func BenchmarkSumLarge(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Sum(1000000, 2000000)
    }
}

// รัน benchmark
// go test -bench=.
// go test -bench=BenchmarkSum
// go test -bench=. -benchmem  // แสดง memory allocations

// Output:
// BenchmarkSum-8      1000000000    0.3 ns/op    0 B/op    0 allocs/op
```

### Benchmark with Setup

```go
func BenchmarkComplexOperation(b *testing.B) {
    // Setup (ไม่นับเวลา)
    data := prepareTestData()

    b.ResetTimer()  // reset timer หลัง setup

    for i := 0; i < b.N; i++ {
        processData(data)
    }
}

func BenchmarkParallel(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            doParallelWork()
        }
    })
}
```

---

## Test Coverage

### รัน Tests พร้อม Coverage

```bash
# Generate coverage
go test -cover ./...

# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# View coverage by function
go tool cover -func=coverage.out
```

### Coverage Output

```
=== RUN   TestSum
--- PASS: TestSum (0.00s)
=== RUN   TestSumNegative
--- PASS: TestSumNegative (0.00s)
PASS
coverage: 85.7% of statements
```

---

## ตัวอย่างจาก Booking Rush

```go
// backend-booking/internal/service/booking_service_test.go

func TestBookingService_Reserve(t *testing.T) {
    tests := []struct {
        name          string
        req           *dto.ReserveRequest
        mockSetup     func(*MockBookingRepository, *MockRedisClient)
        expectedError error
    }{
        {
            name: "successful reservation",
            req: &dto.ReserveRequest{
                EventID:   "event-1",
                ZoneID:    "zone-1",
                ShowID:    "show-1",
                Quantity:  2,
                UnitPrice: 500,
            },
            mockSetup: func(repo *MockBookingRepository, redis *MockRedisClient) {
                redis.On("CheckAndReserve", mock.Anything, "zone-1", 2).
                    Return(true, nil)
                repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Booking")).
                    Return(nil)
            },
            expectedError: nil,
        },
        {
            name: "insufficient seats",
            req: &dto.ReserveRequest{
                EventID:   "event-1",
                ZoneID:    "zone-1",
                ShowID:    "show-1",
                Quantity:  100,
                UnitPrice: 500,
            },
            mockSetup: func(repo *MockBookingRepository, redis *MockRedisClient) {
                redis.On("CheckAndReserve", mock.Anything, "zone-1", 100).
                    Return(false, nil)
            },
            expectedError: ErrInsufficientSeats,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            mockRepo := new(MockBookingRepository)
            mockRedis := new(MockRedisClient)
            tt.mockSetup(mockRepo, mockRedis)

            // Create service
            service := NewBookingService(mockRepo, mockRedis)

            // Execute
            _, err := service.Reserve(context.Background(), tt.req)

            // Assert
            if tt.expectedError != nil {
                assert.ErrorIs(t, err, tt.expectedError)
            } else {
                assert.NoError(t, err)
            }

            mockRepo.AssertExpectations(t)
            mockRedis.AssertExpectations(t)
        })
    }
}

// backend-booking/internal/handler/booking_handler_test.go

func TestBookingHandler_Reserve(t *testing.T) {
    gin.SetMode(gin.TestMode)

    t.Run("success", func(t *testing.T) {
        // Setup mock service
        mockService := new(MockBookingService)
        mockService.On("Reserve", mock.Anything, mock.AnythingOfType("*dto.ReserveRequest")).
            Return(&dto.ReserveResponse{
                BookingID:   "booking-123",
                TotalAmount: 1000,
            }, nil)

        // Setup router
        handler := NewBookingHandler(mockService)
        router := gin.New()
        router.POST("/reserve", handler.Reserve)

        // Create request
        body := `{
            "event_id": "event-1",
            "zone_id": "zone-1",
            "show_id": "show-1",
            "quantity": 2,
            "unit_price": 500
        }`
        req, _ := http.NewRequest("POST", "/reserve", strings.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        // Execute
        router.ServeHTTP(w, req)

        // Assert
        assert.Equal(t, 200, w.Code)

        var response dto.ReserveResponse
        json.Unmarshal(w.Body.Bytes(), &response)
        assert.Equal(t, "booking-123", response.BookingID)

        mockService.AssertExpectations(t)
    })
}
```

---

## สรุป

| หัวข้อ | Jest/Vitest (TypeScript) | Go testing |
|--------|--------------------------|------------|
| Test file | `*.test.ts`, `*.spec.ts` | `*_test.go` |
| Test function | `it()`, `test()` | `func TestXxx(t *testing.T)` |
| Assertion | `expect().toBe()` | Manual หรือ testify |
| Mock | `jest.mock()` | Interface + Mock struct |
| HTTP test | supertest | httptest |
| Benchmark | External lib | `func BenchmarkXxx(b *testing.B)` |
| Coverage | `--coverage` | `-cover`, `-coverprofile` |
| Run | `npm test` | `go test` |
| Watch | `--watch` | ไม่มี built-in |

### Testing Commands

```bash
# รัน tests
go test ./...

# Verbose
go test -v ./...

# รัน test เฉพาะ
go test -run TestName

# รันพร้อม race detector
go test -race ./...

# Coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmark
go test -bench=. ./...
go test -bench=. -benchmem ./...
```

---

## ต่อไป

- [11-cheatsheet.md](./11-cheatsheet.md) - Quick Reference (สรุปรวม)
