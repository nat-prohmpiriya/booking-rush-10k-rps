package repository

import (
	"context"
	"os"
	"testing"
	"time"

	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// skipIfNoIntegration skips the test if INTEGRATION_TEST env var is not set
func skipIfNoIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}
}

// getRedisClient creates a Redis client for testing
func getRedisClient(t *testing.T) *pkgredis.Client {
	host := os.Getenv("TEST_REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	password := os.Getenv("TEST_REDIS_PASSWORD")

	cfg := &pkgredis.Config{
		Host:          host,
		Port:          6379,
		Password:      password,
		DB:            15, // Use DB 15 for testing
		PoolSize:      10,
		MinIdleConns:  2,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		MaxRetries:    3,
		RetryInterval: time.Second,
	}

	ctx := context.Background()
	client, err := pkgredis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	// Flush test database
	if err := client.Client().FlushDB(ctx).Err(); err != nil {
		t.Fatalf("Failed to flush test database: %v", err)
	}

	return client
}

func TestRedisReservationRepository_ReserveSeats(t *testing.T) {
	skipIfNoIntegration(t)

	ctx := context.Background()
	client := getRedisClient(t)
	defer client.Close()

	repo := NewRedisReservationRepository(client)

	// Load scripts
	if err := repo.LoadScripts(ctx); err != nil {
		t.Fatalf("Failed to load scripts: %v", err)
	}

	// Initialize zone availability
	zoneID := "zone-test-001"
	if err := repo.SetZoneAvailability(ctx, zoneID, 100); err != nil {
		t.Fatalf("Failed to set zone availability: %v", err)
	}

	tests := []struct {
		name        string
		params      ReserveParams
		wantSuccess bool
		wantError   string
	}{
		{
			name: "successful reservation",
			params: ReserveParams{
				ZoneID:     zoneID,
				UserID:     "user-001",
				EventID:    "event-001",
				Quantity:   2,
				MaxPerUser: 4,
				TTLSeconds: 600,
				Price:      100.00,
			},
			wantSuccess: true,
		},
		{
			name: "user limit exceeded",
			params: ReserveParams{
				ZoneID:     zoneID,
				UserID:     "user-001",
				EventID:    "event-001",
				Quantity:   5, // Already reserved 2, max is 4
				MaxPerUser: 4,
				TTLSeconds: 600,
				Price:      100.00,
			},
			wantSuccess: false,
			wantError:   "USER_LIMIT_EXCEEDED",
		},
		{
			name: "zone not found",
			params: ReserveParams{
				ZoneID:     "zone-not-exists",
				UserID:     "user-002",
				EventID:    "event-001",
				Quantity:   1,
				MaxPerUser: 4,
				TTLSeconds: 600,
				Price:      100.00,
			},
			wantSuccess: false,
			wantError:   "ZONE_NOT_FOUND",
		},
		{
			name: "invalid quantity",
			params: ReserveParams{
				ZoneID:     zoneID,
				UserID:     "user-003",
				EventID:    "event-001",
				Quantity:   0,
				MaxPerUser: 4,
				TTLSeconds: 600,
				Price:      100.00,
			},
			wantSuccess: false,
			wantError:   "INVALID_QUANTITY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.ReserveSeats(ctx, tt.params)
			if err != nil {
				t.Fatalf("ReserveSeats() error = %v", err)
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("ReserveSeats() success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if !tt.wantSuccess && result.ErrorCode != tt.wantError {
				t.Errorf("ReserveSeats() errorCode = %v, want %v", result.ErrorCode, tt.wantError)
			}

			if tt.wantSuccess {
				if result.BookingID == "" {
					t.Error("ReserveSeats() bookingID should not be empty")
				}
			}
		})
	}
}

func TestRedisReservationRepository_ReserveSeats_InsufficientStock(t *testing.T) {
	skipIfNoIntegration(t)

	ctx := context.Background()
	client := getRedisClient(t)
	defer client.Close()

	repo := NewRedisReservationRepository(client)

	if err := repo.LoadScripts(ctx); err != nil {
		t.Fatalf("Failed to load scripts: %v", err)
	}

	// Initialize zone with only 5 seats
	zoneID := "zone-limited"
	if err := repo.SetZoneAvailability(ctx, zoneID, 5); err != nil {
		t.Fatalf("Failed to set zone availability: %v", err)
	}

	// Try to reserve 10 seats
	result, err := repo.ReserveSeats(ctx, ReserveParams{
		ZoneID:     zoneID,
		UserID:     "user-001",
		EventID:    "event-001",
		Quantity:   10,
		MaxPerUser: 20,
		TTLSeconds: 600,
		Price:      100.00,
	})

	if err != nil {
		t.Fatalf("ReserveSeats() error = %v", err)
	}

	if result.Success {
		t.Error("ReserveSeats() should fail with insufficient stock")
	}

	if result.ErrorCode != "INSUFFICIENT_STOCK" {
		t.Errorf("ReserveSeats() errorCode = %v, want INSUFFICIENT_STOCK", result.ErrorCode)
	}
}

func TestRedisReservationRepository_ReleaseSeats(t *testing.T) {
	skipIfNoIntegration(t)

	ctx := context.Background()
	client := getRedisClient(t)
	defer client.Close()

	repo := NewRedisReservationRepository(client)

	if err := repo.LoadScripts(ctx); err != nil {
		t.Fatalf("Failed to load scripts: %v", err)
	}

	// Initialize zone
	zoneID := "zone-release-test"
	initialSeats := int64(100)
	if err := repo.SetZoneAvailability(ctx, zoneID, initialSeats); err != nil {
		t.Fatalf("Failed to set zone availability: %v", err)
	}

	// Reserve seats first
	reserveResult, err := repo.ReserveSeats(ctx, ReserveParams{
		ZoneID:     zoneID,
		UserID:     "user-release",
		EventID:    "event-release",
		Quantity:   3,
		MaxPerUser: 10,
		TTLSeconds: 600,
		Price:      100.00,
	})

	if err != nil || !reserveResult.Success {
		t.Fatalf("Failed to reserve seats: %v, %+v", err, reserveResult)
	}

	// Verify availability decreased
	available, err := repo.GetZoneAvailability(ctx, zoneID)
	if err != nil {
		t.Fatalf("Failed to get availability: %v", err)
	}
	if available != initialSeats-3 {
		t.Errorf("Available seats = %d, want %d", available, initialSeats-3)
	}

	// Release seats
	releaseResult, err := repo.ReleaseSeats(ctx, reserveResult.BookingID, "user-release")
	if err != nil {
		t.Fatalf("ReleaseSeats() error = %v", err)
	}

	if !releaseResult.Success {
		t.Errorf("ReleaseSeats() failed: %s - %s", releaseResult.ErrorCode, releaseResult.ErrorMessage)
	}

	// Verify seats restored
	available, err = repo.GetZoneAvailability(ctx, zoneID)
	if err != nil {
		t.Fatalf("Failed to get availability: %v", err)
	}
	if available != initialSeats {
		t.Errorf("Available seats after release = %d, want %d", available, initialSeats)
	}
}

func TestRedisReservationRepository_ConfirmBooking(t *testing.T) {
	skipIfNoIntegration(t)

	ctx := context.Background()
	client := getRedisClient(t)
	defer client.Close()

	repo := NewRedisReservationRepository(client)

	if err := repo.LoadScripts(ctx); err != nil {
		t.Fatalf("Failed to load scripts: %v", err)
	}

	// Initialize zone
	zoneID := "zone-confirm-test"
	if err := repo.SetZoneAvailability(ctx, zoneID, 100); err != nil {
		t.Fatalf("Failed to set zone availability: %v", err)
	}

	// Reserve seats first
	reserveResult, err := repo.ReserveSeats(ctx, ReserveParams{
		ZoneID:     zoneID,
		UserID:     "user-confirm",
		EventID:    "event-confirm",
		Quantity:   2,
		MaxPerUser: 10,
		TTLSeconds: 600,
		Price:      100.00,
	})

	if err != nil || !reserveResult.Success {
		t.Fatalf("Failed to reserve seats: %v, %+v", err, reserveResult)
	}

	// Confirm booking
	confirmResult, err := repo.ConfirmBooking(ctx, reserveResult.BookingID, "user-confirm", "payment-123")
	if err != nil {
		t.Fatalf("ConfirmBooking() error = %v", err)
	}

	if !confirmResult.Success {
		t.Errorf("ConfirmBooking() failed: %s - %s", confirmResult.ErrorCode, confirmResult.ErrorMessage)
	}

	if confirmResult.Status != "CONFIRMED" {
		t.Errorf("ConfirmBooking() status = %v, want CONFIRMED", confirmResult.Status)
	}

	// Try to confirm again - should fail
	confirmAgain, err := repo.ConfirmBooking(ctx, reserveResult.BookingID, "user-confirm", "payment-456")
	if err != nil {
		t.Fatalf("ConfirmBooking() error = %v", err)
	}

	if confirmAgain.Success {
		t.Error("ConfirmBooking() should fail when already confirmed")
	}

	if confirmAgain.ErrorCode != "ALREADY_CONFIRMED" {
		t.Errorf("ConfirmBooking() errorCode = %v, want ALREADY_CONFIRMED", confirmAgain.ErrorCode)
	}
}

func TestRedisReservationRepository_GetZoneAvailability(t *testing.T) {
	skipIfNoIntegration(t)

	ctx := context.Background()
	client := getRedisClient(t)
	defer client.Close()

	repo := NewRedisReservationRepository(client)

	// Set availability
	zoneID := "zone-avail-test"
	expectedSeats := int64(500)
	if err := repo.SetZoneAvailability(ctx, zoneID, expectedSeats); err != nil {
		t.Fatalf("Failed to set zone availability: %v", err)
	}

	// Get availability
	available, err := repo.GetZoneAvailability(ctx, zoneID)
	if err != nil {
		t.Fatalf("GetZoneAvailability() error = %v", err)
	}

	if available != expectedSeats {
		t.Errorf("GetZoneAvailability() = %d, want %d", available, expectedSeats)
	}

	// Get non-existent zone
	available, err = repo.GetZoneAvailability(ctx, "non-existent-zone")
	if err != nil {
		t.Fatalf("GetZoneAvailability() error = %v", err)
	}

	if available != 0 {
		t.Errorf("GetZoneAvailability() for non-existent zone = %d, want 0", available)
	}
}

func TestRedisReservationRepository_ConcurrentReservations(t *testing.T) {
	skipIfNoIntegration(t)

	ctx := context.Background()
	client := getRedisClient(t)
	defer client.Close()

	repo := NewRedisReservationRepository(client)

	if err := repo.LoadScripts(ctx); err != nil {
		t.Fatalf("Failed to load scripts: %v", err)
	}

	// Initialize zone with limited seats
	zoneID := "zone-concurrent"
	totalSeats := int64(10)
	if err := repo.SetZoneAvailability(ctx, zoneID, totalSeats); err != nil {
		t.Fatalf("Failed to set zone availability: %v", err)
	}

	// Run concurrent reservations
	numReservations := 20
	results := make(chan *ReserveResult, numReservations)

	for i := 0; i < numReservations; i++ {
		go func(userNum int) {
			result, err := repo.ReserveSeats(ctx, ReserveParams{
				ZoneID:     zoneID,
				UserID:     "user-concurrent-" + string(rune('0'+userNum)),
				EventID:    "event-concurrent",
				Quantity:   1,
				MaxPerUser: 1,
				TTLSeconds: 600,
				Price:      100.00,
			})
			if err != nil {
				t.Logf("Reservation error for user %d: %v", userNum, err)
				results <- nil
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	successCount := 0
	insufficientCount := 0
	for i := 0; i < numReservations; i++ {
		result := <-results
		if result == nil {
			continue
		}
		if result.Success {
			successCount++
		} else if result.ErrorCode == "INSUFFICIENT_STOCK" {
			insufficientCount++
		}
	}

	// Exactly 10 should succeed (since we have 10 seats)
	if successCount != int(totalSeats) {
		t.Errorf("Concurrent reservations: %d succeeded, want %d", successCount, totalSeats)
	}

	// The rest should fail with insufficient stock
	if insufficientCount != numReservations-int(totalSeats) {
		t.Errorf("Concurrent reservations: %d failed with insufficient, want %d", insufficientCount, numReservations-int(totalSeats))
	}

	// Verify final availability is 0
	available, err := repo.GetZoneAvailability(ctx, zoneID)
	if err != nil {
		t.Fatalf("Failed to get availability: %v", err)
	}
	if available != 0 {
		t.Errorf("Final availability = %d, want 0", available)
	}
}
