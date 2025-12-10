package repository

import (
	"context"
	"testing"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
)

func TestNewMemoryPaymentRepository(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}

	if repo.Count() != 0 {
		t.Error("Expected empty repository")
	}
}

func TestMemoryPaymentRepository_Create(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	payment, _ := domain.NewPayment("booking-123", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)

	err := repo.Create(ctx, payment)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if repo.Count() != 1 {
		t.Errorf("Expected count 1, got %d", repo.Count())
	}
}

func TestMemoryPaymentRepository_Create_Duplicate(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	payment1, _ := domain.NewPayment("booking-123", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	payment2, _ := domain.NewPayment("booking-123", "user-456", 500.00, "THB", domain.PaymentMethodCreditCard)

	repo.Create(ctx, payment1)
	err := repo.Create(ctx, payment2)

	if err == nil {
		t.Error("Expected error for duplicate booking")
	}
}

func TestMemoryPaymentRepository_GetByID(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	payment, _ := domain.NewPayment("booking-123", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	repo.Create(ctx, payment)

	found, err := repo.GetByID(ctx, payment.ID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if found.ID != payment.ID {
		t.Errorf("Expected ID %s, got %s", payment.ID, found.ID)
	}
}

func TestMemoryPaymentRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent payment")
	}
}

func TestMemoryPaymentRepository_GetByBookingID(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	payment, _ := domain.NewPayment("booking-123", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	repo.Create(ctx, payment)

	found, err := repo.GetByBookingID(ctx, "booking-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if found.BookingID != "booking-123" {
		t.Errorf("Expected booking ID 'booking-123', got '%s'", found.BookingID)
	}
}

func TestMemoryPaymentRepository_GetByUserID(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	// Create multiple payments for the same user
	payment1, _ := domain.NewPayment("booking-1", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	payment2, _ := domain.NewPayment("booking-2", "user-456", 500.00, "THB", domain.PaymentMethodCreditCard)
	payment3, _ := domain.NewPayment("booking-3", "user-789", 750.00, "THB", domain.PaymentMethodCreditCard)

	repo.Create(ctx, payment1)
	repo.Create(ctx, payment2)
	repo.Create(ctx, payment3)

	payments, err := repo.GetByUserID(ctx, "user-456", 10, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(payments) != 2 {
		t.Errorf("Expected 2 payments, got %d", len(payments))
	}
}

func TestMemoryPaymentRepository_GetByUserID_Pagination(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	// Create multiple payments
	for i := 0; i < 5; i++ {
		payment, _ := domain.NewPayment("booking-"+string(rune('A'+i)), "user-456", 100.00, "THB", domain.PaymentMethodCreditCard)
		repo.Create(ctx, payment)
	}

	// Get first page
	page1, _ := repo.GetByUserID(ctx, "user-456", 2, 0)
	if len(page1) != 2 {
		t.Errorf("Expected 2 payments on page 1, got %d", len(page1))
	}

	// Get second page
	page2, _ := repo.GetByUserID(ctx, "user-456", 2, 2)
	if len(page2) != 2 {
		t.Errorf("Expected 2 payments on page 2, got %d", len(page2))
	}

	// Get last page
	page3, _ := repo.GetByUserID(ctx, "user-456", 2, 4)
	if len(page3) != 1 {
		t.Errorf("Expected 1 payment on page 3, got %d", len(page3))
	}
}

func TestMemoryPaymentRepository_Update(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	payment, _ := domain.NewPayment("booking-123", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	repo.Create(ctx, payment)

	// Update payment
	payment.MarkProcessing()
	payment.Complete("txn-123")

	err := repo.Update(ctx, payment)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify update
	found, _ := repo.GetByID(ctx, payment.ID)
	if found.Status != domain.PaymentStatusCompleted {
		t.Errorf("Expected status completed, got %s", found.Status)
	}

	if found.TransactionID != "txn-123" {
		t.Errorf("Expected transaction ID 'txn-123', got '%s'", found.TransactionID)
	}
}

func TestMemoryPaymentRepository_Update_NotFound(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	payment, _ := domain.NewPayment("booking-123", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)

	err := repo.Update(ctx, payment)
	if err == nil {
		t.Error("Expected error for non-existent payment")
	}
}

func TestMemoryPaymentRepository_GetByTransactionID(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	payment, _ := domain.NewPayment("booking-123", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	repo.Create(ctx, payment)

	payment.MarkProcessing()
	payment.Complete("txn-abc-123")
	repo.Update(ctx, payment)

	found, err := repo.GetByTransactionID(ctx, "txn-abc-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if found.TransactionID != "txn-abc-123" {
		t.Errorf("Expected transaction ID 'txn-abc-123', got '%s'", found.TransactionID)
	}
}

func TestMemoryPaymentRepository_Clear(t *testing.T) {
	repo := NewMemoryPaymentRepository()
	ctx := context.Background()

	payment, _ := domain.NewPayment("booking-123", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	repo.Create(ctx, payment)

	if repo.Count() != 1 {
		t.Error("Expected count 1 before clear")
	}

	repo.Clear()

	if repo.Count() != 0 {
		t.Error("Expected count 0 after clear")
	}
}
