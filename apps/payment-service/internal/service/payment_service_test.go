package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/gateway"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
)

func skipIfNoIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupTestService(t *testing.T) (PaymentService, *database.PostgresDB, func()) {
	ctx := context.Background()

	cfg := &database.PostgresConfig{
		Host:            getEnv("POSTGRES_HOST", "100.104.0.42"),
		Port:            5432,
		User:            getEnv("POSTGRES_USER", "postgres"),
		Password:        getEnv("POSTGRES_PASSWORD", ""),
		Database:        getEnv("POSTGRES_DB", "booking_rush"),
		SSLMode:         "disable",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: 1 * time.Minute,
		ConnectTimeout:  5 * time.Second,
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
	}

	db, err := database.NewPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create table if not exists
	_, err = db.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS payments (
			id VARCHAR(36) PRIMARY KEY,
			booking_id VARCHAR(36) NOT NULL UNIQUE,
			user_id VARCHAR(36) NOT NULL,
			amount DECIMAL(12,2) NOT NULL,
			currency VARCHAR(3) NOT NULL DEFAULT 'THB',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			method VARCHAR(20) NOT NULL,
			transaction_id VARCHAR(255),
			failure_reason TEXT,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create payments table: %v", err)
	}

	repo := repository.NewPostgresPaymentRepository(db)
	gw := gateway.NewMockGateway(&gateway.MockGatewayConfig{
		SuccessRate: 1.0,
		DelayMs:     0,
	})

	svc := NewPaymentService(repo, gw, &PaymentServiceConfig{
		Currency:        "THB",
		GatewayType:     "mock",
		MockSuccessRate: 1.0,
		MockDelayMs:     0,
	})

	cleanup := func() {
		ctx := context.Background()
		db.Pool().Exec(ctx, "DELETE FROM payments WHERE booking_id LIKE 'test-svc-booking-%'")
		db.Close()
	}

	return svc, db, cleanup
}

func TestPaymentService_CreateAndProcess_Integration(t *testing.T) {
	skipIfNoIntegration(t)

	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create payment
	req := &CreatePaymentRequest{
		BookingID: "test-svc-booking-001",
		UserID:    "test-user-001",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
		Metadata: map[string]string{
			"event_id": "event-123",
		},
	}

	payment, err := svc.CreatePayment(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create payment: %v", err)
	}

	if payment.Status != domain.PaymentStatusPending {
		t.Errorf("Expected status 'pending', got '%s'", payment.Status)
	}

	// Process payment
	processed, err := svc.ProcessPayment(ctx, payment.ID)
	if err != nil {
		t.Fatalf("Failed to process payment: %v", err)
	}

	if processed.Status != domain.PaymentStatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", processed.Status)
	}

	if processed.TransactionID == "" {
		t.Error("Expected TransactionID to be set")
	}
}

func TestPaymentService_GetPayment_Integration(t *testing.T) {
	skipIfNoIntegration(t)

	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create payment
	req := &CreatePaymentRequest{
		BookingID: "test-svc-booking-002",
		UserID:    "test-user-002",
		Amount:    2000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodDebitCard,
	}

	created, err := svc.CreatePayment(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create payment: %v", err)
	}

	// Get by ID
	byID, err := svc.GetPayment(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get payment by ID: %v", err)
	}

	if byID.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, byID.ID)
	}

	// Get by booking ID
	byBooking, err := svc.GetPaymentByBookingID(ctx, "test-svc-booking-002")
	if err != nil {
		t.Fatalf("Failed to get payment by booking ID: %v", err)
	}

	if byBooking.BookingID != "test-svc-booking-002" {
		t.Errorf("Expected booking ID 'test-svc-booking-002', got '%s'", byBooking.BookingID)
	}
}

func TestPaymentService_GetUserPayments_Integration(t *testing.T) {
	skipIfNoIntegration(t)

	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	testUserID := "test-svc-user-list"

	// Create multiple payments
	for i := 0; i < 5; i++ {
		req := &CreatePaymentRequest{
			BookingID: "test-svc-booking-list-" + string(rune('A'+i)),
			UserID:    testUserID,
			Amount:    float64(100 * (i + 1)),
			Currency:  "THB",
			Method:    domain.PaymentMethodCreditCard,
		}
		_, err := svc.CreatePayment(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create payment %d: %v", i, err)
		}
	}

	// Get all payments
	payments, err := svc.GetUserPayments(ctx, testUserID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get user payments: %v", err)
	}

	if len(payments) != 5 {
		t.Errorf("Expected 5 payments, got %d", len(payments))
	}

	// Test pagination
	page1, err := svc.GetUserPayments(ctx, testUserID, 2, 0)
	if err != nil {
		t.Fatalf("Failed to get page 1: %v", err)
	}

	if len(page1) != 2 {
		t.Errorf("Expected 2 payments on page 1, got %d", len(page1))
	}

	page2, err := svc.GetUserPayments(ctx, testUserID, 2, 2)
	if err != nil {
		t.Fatalf("Failed to get page 2: %v", err)
	}

	if len(page2) != 2 {
		t.Errorf("Expected 2 payments on page 2, got %d", len(page2))
	}
}

func TestPaymentService_RefundPayment_Integration(t *testing.T) {
	skipIfNoIntegration(t)

	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create and process payment
	req := &CreatePaymentRequest{
		BookingID: "test-svc-booking-refund",
		UserID:    "test-user-refund",
		Amount:    3000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}

	payment, _ := svc.CreatePayment(ctx, req)
	processed, _ := svc.ProcessPayment(ctx, payment.ID)

	// Refund payment
	refunded, err := svc.RefundPayment(ctx, processed.ID, "customer requested")
	if err != nil {
		t.Fatalf("Failed to refund payment: %v", err)
	}

	if refunded.Status != domain.PaymentStatusRefunded {
		t.Errorf("Expected status 'refunded', got '%s'", refunded.Status)
	}
}

func TestPaymentService_CancelPayment_Integration(t *testing.T) {
	skipIfNoIntegration(t)

	svc, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Create payment (don't process)
	req := &CreatePaymentRequest{
		BookingID: "test-svc-booking-cancel",
		UserID:    "test-user-cancel",
		Amount:    1500.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}

	payment, _ := svc.CreatePayment(ctx, req)

	// Cancel payment
	cancelled, err := svc.CancelPayment(ctx, payment.ID)
	if err != nil {
		t.Fatalf("Failed to cancel payment: %v", err)
	}

	if cancelled.Status != domain.PaymentStatusCancelled {
		t.Errorf("Expected status 'cancelled', got '%s'", cancelled.Status)
	}
}

func TestPaymentService_FailedPayment_Integration(t *testing.T) {
	skipIfNoIntegration(t)

	ctx := context.Background()

	cfg := &database.PostgresConfig{
		Host:            getEnv("POSTGRES_HOST", "100.104.0.42"),
		Port:            5432,
		User:            getEnv("POSTGRES_USER", "postgres"),
		Password:        getEnv("POSTGRES_PASSWORD", ""),
		Database:        getEnv("POSTGRES_DB", "booking_rush"),
		SSLMode:         "disable",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: 1 * time.Minute,
		ConnectTimeout:  5 * time.Second,
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
	}

	db, err := database.NewPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	defer func() {
		db.Pool().Exec(ctx, "DELETE FROM payments WHERE booking_id LIKE 'test-svc-booking-%'")
	}()

	// Create service with failing gateway
	repo := repository.NewPostgresPaymentRepository(db)
	failingGw := gateway.NewMockGateway(&gateway.MockGatewayConfig{
		SuccessRate:    0.0, // Always fail
		DelayMs:        0,
		FailureReasons: []string{"card_declined"},
	})

	svc := NewPaymentService(repo, failingGw, nil)

	// Create and process payment
	req := &CreatePaymentRequest{
		BookingID: "test-svc-booking-fail",
		UserID:    "test-user-fail",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}

	payment, _ := svc.CreatePayment(ctx, req)
	processed, err := svc.ProcessPayment(ctx, payment.ID)
	if err != nil {
		t.Fatalf("ProcessPayment returned error: %v", err)
	}

	if processed.Status != domain.PaymentStatusFailed {
		t.Errorf("Expected status 'failed', got '%s'", processed.Status)
	}

	if processed.FailureReason == "" {
		t.Error("Expected failure reason to be set")
	}
}
