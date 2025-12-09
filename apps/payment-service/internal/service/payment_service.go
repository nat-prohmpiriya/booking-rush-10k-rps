package service

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/dto"
)

// PaymentService defines the interface for payment business logic
type PaymentService interface {
	// CreatePayment creates a new payment for a booking
	CreatePayment(ctx context.Context, userID string, req *dto.CreatePaymentRequest) (*domain.Payment, error)

	// ProcessPayment processes a payment
	ProcessPayment(ctx context.Context, userID string, req *dto.ProcessPaymentRequest) (*domain.Payment, error)

	// GetPayment retrieves a payment by ID
	GetPayment(ctx context.Context, userID, paymentID string) (*domain.Payment, error)

	// GetPaymentByBookingID retrieves a payment by booking ID
	GetPaymentByBookingID(ctx context.Context, userID, bookingID string) (*domain.Payment, error)

	// GetUserPayments retrieves all payments for a user
	GetUserPayments(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error)

	// RefundPayment refunds a payment
	RefundPayment(ctx context.Context, userID string, req *dto.RefundPaymentRequest) (*domain.Payment, error)

	// CancelPayment cancels a pending payment
	CancelPayment(ctx context.Context, userID, paymentID string) (*domain.Payment, error)

	// HandleWebhook handles payment gateway webhook
	HandleWebhook(ctx context.Context, payload []byte) error
}

// PaymentServiceConfig holds configuration for the payment service
type PaymentServiceConfig struct {
	// Gateway configuration
	GatewayAPIKey    string
	GatewaySecretKey string
	GatewayWebhookSecret string

	// Processing options
	AutoCapture bool
	Currency    string
}
