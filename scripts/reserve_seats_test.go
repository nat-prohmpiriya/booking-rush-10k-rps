package scripts

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func getTestRedisClient() *redis.Client {
	host := os.Getenv("TEST_REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	password := os.Getenv("TEST_REDIS_PASSWORD")

	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", host),
		Password: password,
		DB:       1, // Use DB 1 for tests
	})
}

func skipIfNoRedis(t *testing.T, client *redis.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping: Redis not available: %v", err)
	}
}

func TestReserveSeats_Success(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	client := getTestRedisClient()
	defer client.Close()
	skipIfNoRedis(t, client)

	ctx := context.Background()

	// Setup test data
	zoneID := uuid.New().String()
	userID := uuid.New().String()
	eventID := uuid.New().String()
	showID := uuid.New().String()
	bookingID := uuid.New().String()

	zoneKey := fmt.Sprintf("zone:availability:%s", zoneID)

	// Initialize zone with 100 seats
	client.Set(ctx, zoneKey, 100, 0)
	defer client.Del(ctx, zoneKey)

	// Reserve 2 seats
	result, err := ReserveSeats(ctx, client, ReserveSeatsParams{
		ZoneID:     zoneID,
		UserID:     userID,
		EventID:    eventID,
		ShowID:     showID,
		BookingID:  bookingID,
		Quantity:   2,
		MaxPerUser: 10,
		UnitPrice:  500.00,
		TTLSeconds: 60,
	})

	if err != nil {
		t.Fatalf("ReserveSeats failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s - %s", result.ErrorCode, result.ErrorMessage)
	}

	if result.RemainingSeats != 98 {
		t.Errorf("Expected 98 remaining seats, got %d", result.RemainingSeats)
	}

	if result.UserTotalReserved != 2 {
		t.Errorf("Expected user total reserved 2, got %d", result.UserTotalReserved)
	}

	// Verify reservation key exists
	reservationKey := fmt.Sprintf("reservation:%s", bookingID)
	exists, _ := client.Exists(ctx, reservationKey).Result()
	if exists != 1 {
		t.Error("Expected reservation key to exist")
	}

	// Cleanup
	client.Del(ctx, reservationKey)
	client.Del(ctx, fmt.Sprintf("user:reservations:%s:%s", userID, eventID))
}

func TestReserveSeats_InsufficientStock(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	client := getTestRedisClient()
	defer client.Close()
	skipIfNoRedis(t, client)

	ctx := context.Background()

	zoneID := uuid.New().String()
	zoneKey := fmt.Sprintf("zone:availability:%s", zoneID)

	// Initialize zone with only 5 seats
	client.Set(ctx, zoneKey, 5, 0)
	defer client.Del(ctx, zoneKey)

	// Try to reserve 10 seats
	result, err := ReserveSeats(ctx, client, ReserveSeatsParams{
		ZoneID:     zoneID,
		UserID:     uuid.New().String(),
		EventID:    uuid.New().String(),
		ShowID:     uuid.New().String(),
		BookingID:  uuid.New().String(),
		Quantity:   10,
		MaxPerUser: 10,
		UnitPrice:  500.00,
	})

	if err != nil {
		t.Fatalf("ReserveSeats failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure due to insufficient stock")
	}

	if result.ErrorCode != ErrInsufficientStock {
		t.Errorf("Expected error code %s, got %s", ErrInsufficientStock, result.ErrorCode)
	}
}

func TestReserveSeats_UserLimitExceeded(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	client := getTestRedisClient()
	defer client.Close()
	skipIfNoRedis(t, client)

	ctx := context.Background()

	zoneID := uuid.New().String()
	userID := uuid.New().String()
	eventID := uuid.New().String()
	zoneKey := fmt.Sprintf("zone:availability:%s", zoneID)
	userKey := fmt.Sprintf("user:reservations:%s:%s", userID, eventID)

	// Initialize zone with 100 seats
	client.Set(ctx, zoneKey, 100, 0)
	// User already has 8 reserved
	client.Set(ctx, userKey, 8, 0)
	defer client.Del(ctx, zoneKey, userKey)

	// Try to reserve 5 more (would exceed limit of 10)
	result, err := ReserveSeats(ctx, client, ReserveSeatsParams{
		ZoneID:     zoneID,
		UserID:     userID,
		EventID:    eventID,
		ShowID:     uuid.New().String(),
		BookingID:  uuid.New().String(),
		Quantity:   5, // 8 + 5 = 13 > 10
		MaxPerUser: 10,
		UnitPrice:  500.00,
	})

	if err != nil {
		t.Fatalf("ReserveSeats failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure due to user limit exceeded")
	}

	if result.ErrorCode != ErrUserLimitExceeded {
		t.Errorf("Expected error code %s, got %s", ErrUserLimitExceeded, result.ErrorCode)
	}
}

func TestReserveSeats_ZoneNotFound(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	client := getTestRedisClient()
	defer client.Close()
	skipIfNoRedis(t, client)

	ctx := context.Background()

	// Don't create zone key
	result, err := ReserveSeats(ctx, client, ReserveSeatsParams{
		ZoneID:     uuid.New().String(),
		UserID:     uuid.New().String(),
		EventID:    uuid.New().String(),
		ShowID:     uuid.New().String(),
		BookingID:  uuid.New().String(),
		Quantity:   2,
		MaxPerUser: 10,
		UnitPrice:  500.00,
	})

	if err != nil {
		t.Fatalf("ReserveSeats failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure due to zone not found")
	}

	if result.ErrorCode != ErrZoneNotFound {
		t.Errorf("Expected error code %s, got %s", ErrZoneNotFound, result.ErrorCode)
	}
}

func TestReserveSeats_InvalidQuantity(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	client := getTestRedisClient()
	defer client.Close()
	skipIfNoRedis(t, client)

	ctx := context.Background()

	zoneID := uuid.New().String()
	zoneKey := fmt.Sprintf("zone:availability:%s", zoneID)
	client.Set(ctx, zoneKey, 100, 0)
	defer client.Del(ctx, zoneKey)

	// Try to reserve 0 seats
	result, err := ReserveSeats(ctx, client, ReserveSeatsParams{
		ZoneID:     zoneID,
		UserID:     uuid.New().String(),
		EventID:    uuid.New().String(),
		ShowID:     uuid.New().String(),
		BookingID:  uuid.New().String(),
		Quantity:   0,
		MaxPerUser: 10,
		UnitPrice:  500.00,
	})

	if err != nil {
		t.Fatalf("ReserveSeats failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure due to invalid quantity")
	}

	if result.ErrorCode != ErrInvalidQuantity {
		t.Errorf("Expected error code %s, got %s", ErrInvalidQuantity, result.ErrorCode)
	}
}

func TestReserveSeats_ConcurrentReservations(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	client := getTestRedisClient()
	defer client.Close()
	skipIfNoRedis(t, client)

	ctx := context.Background()

	zoneID := uuid.New().String()
	eventID := uuid.New().String()
	showID := uuid.New().String()
	zoneKey := fmt.Sprintf("zone:availability:%s", zoneID)

	// Initialize zone with 50 seats
	client.Set(ctx, zoneKey, 50, 0)
	defer client.Del(ctx, zoneKey)

	// Run 100 concurrent reservations of 1 seat each
	// Only 50 should succeed
	concurrency := 100
	results := make(chan *ReserveSeatsResult, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			result, err := ReserveSeats(ctx, client, ReserveSeatsParams{
				ZoneID:     zoneID,
				UserID:     uuid.New().String(), // Different user each time
				EventID:    eventID,
				ShowID:     showID,
				BookingID:  uuid.New().String(),
				Quantity:   1,
				MaxPerUser: 10,
				UnitPrice:  500.00,
				TTLSeconds: 60,
			})
			if err != nil {
				t.Logf("Error in goroutine %d: %v", idx, err)
				results <- nil
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	successCount := 0
	failCount := 0
	for i := 0; i < concurrency; i++ {
		result := <-results
		if result == nil {
			continue
		}
		if result.Success {
			successCount++
		} else {
			failCount++
		}
	}

	t.Logf("Success: %d, Failed: %d", successCount, failCount)

	// Exactly 50 should succeed (no overselling)
	if successCount != 50 {
		t.Errorf("Expected exactly 50 successful reservations, got %d", successCount)
	}

	// Verify final availability is 0
	remaining, _ := client.Get(ctx, zoneKey).Int()
	if remaining != 0 {
		t.Errorf("Expected 0 remaining seats, got %d", remaining)
	}
}
