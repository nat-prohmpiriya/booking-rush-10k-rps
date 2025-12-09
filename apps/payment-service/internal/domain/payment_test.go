package domain

import (
	"testing"
)

func TestNewPayment(t *testing.T) {
	tests := []struct {
		name      string
		bookingID string
		userID    string
		amount    float64
		currency  string
		method    PaymentMethod
		wantErr   bool
	}{
		{
			name:      "valid payment",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    100.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   false,
		},
		{
			name:      "missing booking_id",
			bookingID: "",
			userID:    "user-123",
			amount:    100.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "missing user_id",
			bookingID: "booking-123",
			userID:    "",
			amount:    100.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "zero amount",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    0,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "negative amount",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    -50.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "missing currency",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    100.00,
			currency:  "",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := NewPayment(tt.bookingID, tt.userID, tt.amount, tt.currency, tt.method)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if payment.ID == "" {
				t.Error("Expected payment ID to be set")
			}
			if payment.BookingID != tt.bookingID {
				t.Errorf("Expected booking_id %s, got %s", tt.bookingID, payment.BookingID)
			}
			if payment.UserID != tt.userID {
				t.Errorf("Expected user_id %s, got %s", tt.userID, payment.UserID)
			}
			if payment.Amount != tt.amount {
				t.Errorf("Expected amount %f, got %f", tt.amount, payment.Amount)
			}
			if payment.Status != PaymentStatusPending {
				t.Errorf("Expected status pending, got %s", payment.Status)
			}
		})
	}
}

func TestPayment_MarkProcessing(t *testing.T) {
	payment, _ := NewPayment("booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	err := payment.MarkProcessing()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusProcessing {
		t.Errorf("Expected status processing, got %s", payment.Status)
	}

	// Should fail if called again
	err = payment.MarkProcessing()
	if err == nil {
		t.Error("Expected error when marking processing again")
	}
}

func TestPayment_Complete(t *testing.T) {
	payment, _ := NewPayment("booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	// Should fail from pending status
	err := payment.Complete("txn-123")
	if err == nil {
		t.Error("Expected error when completing from pending status")
	}

	// Mark as processing first
	payment.MarkProcessing()

	// Should fail without transaction ID
	err = payment.Complete("")
	if err == nil {
		t.Error("Expected error without transaction ID")
	}

	// Should succeed
	err = payment.Complete("txn-123")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusCompleted {
		t.Errorf("Expected status completed, got %s", payment.Status)
	}
	if payment.TransactionID != "txn-123" {
		t.Errorf("Expected transaction_id txn-123, got %s", payment.TransactionID)
	}
	if payment.CompletedAt == nil {
		t.Error("Expected completed_at to be set")
	}
}

func TestPayment_Fail(t *testing.T) {
	payment, _ := NewPayment("booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	err := payment.Fail("insufficient funds")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusFailed {
		t.Errorf("Expected status failed, got %s", payment.Status)
	}
	if payment.FailureReason != "insufficient funds" {
		t.Errorf("Expected failure_reason 'insufficient funds', got '%s'", payment.FailureReason)
	}
}

func TestPayment_Refund(t *testing.T) {
	payment, _ := NewPayment("booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	// Should fail from pending status
	err := payment.Refund()
	if err == nil {
		t.Error("Expected error when refunding from pending status")
	}

	// Complete the payment first
	payment.MarkProcessing()
	payment.Complete("txn-123")

	// Should succeed
	err = payment.Refund()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusRefunded {
		t.Errorf("Expected status refunded, got %s", payment.Status)
	}
}

func TestPayment_Cancel(t *testing.T) {
	payment, _ := NewPayment("booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	err := payment.Cancel()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusCancelled {
		t.Errorf("Expected status cancelled, got %s", payment.Status)
	}

	// Should fail if called again
	payment2, _ := NewPayment("booking-456", "user-123", 100.00, "THB", PaymentMethodCreditCard)
	payment2.MarkProcessing()

	err = payment2.Cancel()
	if err == nil {
		t.Error("Expected error when cancelling processing payment")
	}
}

func TestPayment_IsFinal(t *testing.T) {
	payment, _ := NewPayment("booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	if payment.IsFinal() {
		t.Error("Pending payment should not be final")
	}

	payment.MarkProcessing()
	if payment.IsFinal() {
		t.Error("Processing payment should not be final")
	}

	payment.Complete("txn-123")
	if !payment.IsFinal() {
		t.Error("Completed payment should be final")
	}
}

func TestPayment_IsSuccessful(t *testing.T) {
	payment, _ := NewPayment("booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	if payment.IsSuccessful() {
		t.Error("Pending payment should not be successful")
	}

	payment.MarkProcessing()
	payment.Complete("txn-123")

	if !payment.IsSuccessful() {
		t.Error("Completed payment should be successful")
	}
}
