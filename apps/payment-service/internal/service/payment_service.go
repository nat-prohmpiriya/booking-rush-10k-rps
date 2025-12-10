package service

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
)

// CreatePaymentRequest represents a request to create a payment (internal)
type CreatePaymentRequest struct {
	BookingID string
	UserID    string
	Amount    float64
	Currency  string
	Method    domain.PaymentMethod
	Metadata  map[string]string
}

// PaymentService defines the interface for payment business logic
type PaymentService interface {
	// CreatePayment creates a new payment for a booking
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*domain.Payment, error)

	// ProcessPayment processes a payment by ID
	ProcessPayment(ctx context.Context, paymentID string) (*domain.Payment, error)

	// GetPayment retrieves a payment by ID
	GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error)

	// GetPaymentByBookingID retrieves a payment by booking ID
	GetPaymentByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error)

	// GetUserPayments retrieves all payments for a user
	GetUserPayments(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error)

	// RefundPayment refunds a payment
	RefundPayment(ctx context.Context, paymentID string, reason string) (*domain.Payment, error)

	// CancelPayment cancels a pending payment
	CancelPayment(ctx context.Context, paymentID string) (*domain.Payment, error)
}

// PaymentServiceConfig holds configuration for the payment service
type PaymentServiceConfig struct {
	// Gateway type: "mock" or "stripe"
	GatewayType string

	// Gateway configuration
	GatewayAPIKey        string
	GatewaySecretKey     string
	GatewayWebhookSecret string

	// Processing options
	AutoCapture bool
	Currency    string

	// Mock gateway settings
	MockSuccessRate float64 // 0.0 to 1.0, default 0.95 (95% success)
	MockDelayMs     int     // Simulated processing delay in milliseconds
}
