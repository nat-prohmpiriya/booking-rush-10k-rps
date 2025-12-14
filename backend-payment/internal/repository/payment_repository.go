package repository

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/domain"
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

	// GetByGatewayPaymentID retrieves a payment by gateway payment ID (e.g., Stripe PaymentIntent ID)
	GetByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (*domain.Payment, error)

	// GetByIdempotencyKey retrieves a payment by idempotency key
	GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Payment, error)
}
