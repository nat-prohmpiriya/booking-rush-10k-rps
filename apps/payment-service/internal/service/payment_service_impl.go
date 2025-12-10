package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/gateway"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/repository"
)

// paymentServiceImpl implements PaymentService
type paymentServiceImpl struct {
	repo    repository.PaymentRepository
	gateway gateway.PaymentGateway
	config  *PaymentServiceConfig
	mu      sync.RWMutex
}

// NewPaymentService creates a new PaymentService
func NewPaymentService(
	repo repository.PaymentRepository,
	gw gateway.PaymentGateway,
	config *PaymentServiceConfig,
) PaymentService {
	if config == nil {
		config = &PaymentServiceConfig{
			GatewayType:     "mock",
			Currency:        "THB",
			MockSuccessRate: 0.95,
			MockDelayMs:     100,
		}
	}

	return &paymentServiceImpl{
		repo:    repo,
		gateway: gw,
		config:  config,
	}
}

// CreatePayment creates a new payment for a booking
func (s *paymentServiceImpl) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*domain.Payment, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	// Check if payment already exists for this booking
	existing, err := s.repo.GetByBookingID(ctx, req.BookingID)
	if err == nil && existing != nil {
		return nil, domain.ErrPaymentAlreadyExists
	}

	// Create new payment
	payment, err := domain.NewPayment(
		req.BookingID,
		req.UserID,
		req.Amount,
		req.Currency,
		req.Method,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Set metadata
	if req.Metadata != nil {
		payment.Metadata = req.Metadata
	}

	// Save to repository
	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	return payment, nil
}

// ProcessPayment processes a payment by ID
func (s *paymentServiceImpl) ProcessPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	// Get payment
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Mark as processing
	if err := payment.MarkProcessing(); err != nil {
		return nil, fmt.Errorf("failed to mark payment as processing: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	// Process through gateway
	chargeReq := &gateway.ChargeRequest{
		PaymentID: payment.ID,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		Method:    string(payment.Method),
		Metadata:  payment.Metadata,
	}

	chargeResp, err := s.gateway.Charge(ctx, chargeReq)
	if err != nil {
		// Mark as failed
		payment.Fail(err.Error())
		s.repo.Update(ctx, payment)
		return payment, nil
	}

	// Update payment based on gateway response
	if chargeResp.Success {
		if err := payment.Complete(chargeResp.TransactionID); err != nil {
			return nil, fmt.Errorf("failed to complete payment: %w", err)
		}
	} else {
		if err := payment.Fail(chargeResp.FailureReason); err != nil {
			return nil, fmt.Errorf("failed to mark payment as failed: %w", err)
		}
	}

	// Save final status
	if err := s.repo.Update(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	return payment, nil
}

// GetPayment retrieves a payment by ID
func (s *paymentServiceImpl) GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	return s.repo.GetByID(ctx, paymentID)
}

// GetPaymentByBookingID retrieves a payment by booking ID
func (s *paymentServiceImpl) GetPaymentByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error) {
	return s.repo.GetByBookingID(ctx, bookingID)
}

// GetUserPayments retrieves all payments for a user
func (s *paymentServiceImpl) GetUserPayments(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetByUserID(ctx, userID, limit, offset)
}

// RefundPayment refunds a payment
func (s *paymentServiceImpl) RefundPayment(ctx context.Context, paymentID string, reason string) (*domain.Payment, error) {
	// Get payment
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Process refund through gateway
	if err := s.gateway.Refund(ctx, payment.TransactionID, payment.Amount); err != nil {
		return nil, fmt.Errorf("failed to process refund: %w", err)
	}

	// Mark as refunded
	if err := payment.Refund(); err != nil {
		return nil, fmt.Errorf("failed to mark payment as refunded: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	return payment, nil
}

// CancelPayment cancels a pending payment
func (s *paymentServiceImpl) CancelPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	// Get payment
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Cancel payment
	if err := payment.Cancel(); err != nil {
		return nil, fmt.Errorf("failed to cancel payment: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	return payment, nil
}
