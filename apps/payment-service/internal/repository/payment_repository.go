package repository

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
)

// PaymentRepository defines the interface for payment data access
type PaymentRepository interface {
	// Create creates a new payment record
	Create(ctx context.Context, payment *domain.Payment) error

	// GetByID retrieves a payment by its ID
	GetByID(ctx context.Context, id string) (*domain.Payment, error)

	// GetByBookingID retrieves a payment by booking ID
	GetByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error)

	// GetByUserID retrieves all payments for a user
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error)

	// Update updates an existing payment
	Update(ctx context.Context, payment *domain.Payment) error

	// GetByTransactionID retrieves a payment by transaction ID
	GetByTransactionID(ctx context.Context, transactionID string) (*domain.Payment, error)
}
