package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
)

// getPostgresPool creates a PostgreSQL connection pool for testing
func getPostgresPool(t *testing.T) *pgxpool.Pool {
	skipIfNoIntegration(t)

	host := os.Getenv("TEST_POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("TEST_POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("TEST_POSTGRES_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("TEST_POSTGRES_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dbname := os.Getenv("TEST_POSTGRES_DB")
	if dbname == "" {
		dbname = "booking_rush_test"
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Failed to ping PostgreSQL: %v", err)
	}

	// Clean up test data
	cleanupTestData(t, pool)

	return pool
}

func cleanupTestData(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	// Clean up in reverse order of dependencies
	tables := []string{
		"bookings",
		// Add other tables if needed
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, "DELETE FROM "+table+" WHERE id::text LIKE 'test-%' OR id::text LIKE '%test%'")
		if err != nil {
			t.Logf("Warning: failed to clean up %s: %v", table, err)
		}
	}
}

func createTestBooking(tenantID, userID, eventID, showID, zoneID string) *domain.Booking {
	now := time.Now()
	return &domain.Booking{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		UserID:     userID,
		EventID:    eventID,
		ShowID:     showID,
		ZoneID:     zoneID,
		Quantity:   2,
		UnitPrice:  100.00,
		TotalPrice: 200.00,
		Currency:   "THB",
		Status:     domain.BookingStatusReserved,
		ReservedAt: now,
		ExpiresAt:  now.Add(10 * time.Minute),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestPostgresBookingRepository_Create(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	// Note: You'll need valid tenant_id, user_id, event_id, show_id, zone_id
	// that exist in the database due to foreign key constraints
	// For now, we'll skip this test if there's no test data
	t.Skip("Skipping: requires existing tenant, user, event, show, zone records")

	booking := createTestBooking(
		"test-tenant-id",
		"test-user-id",
		"test-event-id",
		"test-show-id",
		"test-zone-id",
	)

	err := repo.Create(ctx, booking)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify booking was created
	retrieved, err := repo.GetByID(ctx, booking.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.ID != booking.ID {
		t.Errorf("GetByID() ID = %v, want %v", retrieved.ID, booking.ID)
	}

	if retrieved.Quantity != booking.Quantity {
		t.Errorf("GetByID() Quantity = %v, want %v", retrieved.Quantity, booking.Quantity)
	}

	if retrieved.TotalPrice != booking.TotalPrice {
		t.Errorf("GetByID() TotalPrice = %v, want %v", retrieved.TotalPrice, booking.TotalPrice)
	}

	// Cleanup
	_ = repo.Delete(ctx, booking.ID)
}

func TestPostgresBookingRepository_GetByID_NotFound(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New().String())
	if err != domain.ErrBookingNotFound {
		t.Errorf("GetByID() error = %v, want %v", err, domain.ErrBookingNotFound)
	}
}

func TestPostgresBookingRepository_UpdateStatus(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	// Skip if no test data
	t.Skip("Skipping: requires existing booking record")

	bookingID := "existing-booking-id"

	err := repo.UpdateStatus(ctx, bookingID, domain.BookingStatusConfirmed)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	// Verify status was updated
	retrieved, err := repo.GetByID(ctx, bookingID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Status != domain.BookingStatusConfirmed {
		t.Errorf("Status = %v, want %v", retrieved.Status, domain.BookingStatusConfirmed)
	}
}

func TestPostgresBookingRepository_Delete_NotFound(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New().String())
	if err != domain.ErrBookingNotFound {
		t.Errorf("Delete() error = %v, want %v", err, domain.ErrBookingNotFound)
	}
}

func TestPostgresBookingRepository_Confirm(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	t.Skip("Skipping: requires existing booking record")

	bookingID := "existing-reserved-booking-id"
	paymentID := "payment-" + uuid.New().String()

	err := repo.Confirm(ctx, bookingID, paymentID)
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}

	// Verify confirmation
	retrieved, err := repo.GetByID(ctx, bookingID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Status != domain.BookingStatusConfirmed {
		t.Errorf("Status = %v, want %v", retrieved.Status, domain.BookingStatusConfirmed)
	}

	if retrieved.PaymentID != paymentID {
		t.Errorf("PaymentID = %v, want %v", retrieved.PaymentID, paymentID)
	}

	if retrieved.ConfirmedAt == nil {
		t.Error("ConfirmedAt should not be nil")
	}
}

func TestPostgresBookingRepository_Cancel(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	t.Skip("Skipping: requires existing booking record")

	bookingID := "existing-reserved-booking-id"

	err := repo.Cancel(ctx, bookingID)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	// Verify cancellation
	retrieved, err := repo.GetByID(ctx, bookingID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if retrieved.Status != domain.BookingStatusCancelled {
		t.Errorf("Status = %v, want %v", retrieved.Status, domain.BookingStatusCancelled)
	}

	if retrieved.CancelledAt == nil {
		t.Error("CancelledAt should not be nil")
	}
}

func TestPostgresBookingRepository_GetByUserID(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	// This test checks that the query works, even if it returns empty
	userID := uuid.New().String()

	bookings, err := repo.GetByUserID(ctx, userID, 10, 0)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}

	// For a random UUID, should return empty list
	if len(bookings) != 0 {
		t.Errorf("GetByUserID() returned %d bookings, want 0", len(bookings))
	}
}

func TestPostgresBookingRepository_CountByUserAndEvent(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	// Test with random UUIDs - should return 0
	count, err := repo.CountByUserAndEvent(ctx, uuid.New().String(), uuid.New().String())
	if err != nil {
		t.Fatalf("CountByUserAndEvent() error = %v", err)
	}

	if count != 0 {
		t.Errorf("CountByUserAndEvent() = %d, want 0", count)
	}
}

func TestPostgresBookingRepository_GetExpiredReservations(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	// Test the query works
	bookings, err := repo.GetExpiredReservations(ctx, 100)
	if err != nil {
		t.Fatalf("GetExpiredReservations() error = %v", err)
	}

	// Just verify no error - actual count depends on test data
	t.Logf("Found %d expired reservations", len(bookings))
}

func TestPostgresBookingRepository_MarkAsExpired(t *testing.T) {
	skipIfNoIntegration(t)

	pool := getPostgresPool(t)
	defer pool.Close()

	repo := NewPostgresBookingRepository(pool)
	ctx := context.Background()

	// Test with non-existent ID
	err := repo.MarkAsExpired(ctx, uuid.New().String())
	if err != domain.ErrBookingNotFound {
		t.Errorf("MarkAsExpired() error = %v, want %v", err, domain.ErrBookingNotFound)
	}
}
