package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/gateway"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/metrics"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	ctx, span := telemetry.StartSpan(ctx, "service.payment.create")
	defer span.End()

	if req == nil {
		span.RecordError(fmt.Errorf("request is required"))
		span.SetStatus(codes.Error, "request is required")
		return nil, fmt.Errorf("request is required")
	}

	span.SetAttributes(
		attribute.String("booking_id", req.BookingID),
		attribute.String("user_id", req.UserID),
		attribute.Float64("amount", req.Amount),
		attribute.String("currency", req.Currency),
		attribute.String("method", string(req.Method)),
	)

	// Check if payment already exists for this booking
	existing, err := s.repo.GetByBookingID(ctx, req.BookingID)
	if err == nil && existing != nil {
		span.RecordError(domain.ErrPaymentAlreadyExists)
		span.SetStatus(codes.Error, "payment already exists")
		return nil, domain.ErrPaymentAlreadyExists
	}

	// Create new payment with TenantID
	payment, err := domain.NewPayment(
		req.TenantID,
		req.BookingID,
		req.UserID,
		req.Amount,
		req.Currency,
		req.Method,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Set metadata
	if req.Metadata != nil {
		payment.Metadata = req.Metadata
	}

	// Save to repository
	if err := s.repo.Create(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	// Record metrics
	metrics.RecordPaymentCreated(ctx, payment.BookingID, string(payment.Method), payment.Currency, payment.Amount)

	// Add span event for payment created
	span.AddEvent("payment_created", trace.WithAttributes(
		attribute.String("payment_id", payment.ID),
		attribute.String("booking_id", payment.BookingID),
		attribute.Float64("amount", payment.Amount),
		attribute.String("currency", payment.Currency),
		attribute.String("method", string(payment.Method)),
	))

	span.SetAttributes(attribute.String("payment_id", payment.ID))
	span.SetStatus(codes.Ok, "")
	return payment, nil
}

// ProcessPayment processes a payment by ID
func (s *paymentServiceImpl) ProcessPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.payment.process")
	defer span.End()
	startTime := time.Now()

	span.SetAttributes(attribute.String("payment_id", paymentID))

	// Get payment
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("booking_id", payment.BookingID),
		attribute.String("user_id", payment.UserID),
		attribute.Float64("amount", payment.Amount),
		attribute.String("currency", payment.Currency),
	)

	// Mark as processing
	if err := payment.MarkProcessing(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to mark payment as processing: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
		// Mark as failed with error details
		span.RecordError(err)
		span.SetAttributes(attribute.String("failure_reason", "GATEWAY_ERROR"))
		payment.Fail("GATEWAY_ERROR", err.Error())
		s.repo.Update(ctx, payment)
		// Record metrics
		metrics.RecordPaymentFailed(ctx, payment.BookingID, string(payment.Method), "GATEWAY_ERROR")
		span.SetStatus(codes.Ok, "") // Payment failed but operation succeeded
		return payment, nil
	}

	// Update payment based on gateway response
	if chargeResp.Success {
		if err := payment.Complete(chargeResp.TransactionID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("failed to complete payment: %w", err)
		}
		span.SetAttributes(
			attribute.String("transaction_id", chargeResp.TransactionID),
			attribute.String("status", "completed"),
		)
	} else {
		if err := payment.Fail("PAYMENT_FAILED", chargeResp.FailureReason); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("failed to mark payment as failed: %w", err)
		}
		span.SetAttributes(
			attribute.String("failure_reason", chargeResp.FailureReason),
			attribute.String("status", "failed"),
		)
	}

	// Save final status
	if err := s.repo.Update(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Record metrics
	durationSeconds := time.Since(startTime).Seconds()
	if chargeResp.Success {
		metrics.RecordPaymentProcessed(ctx, payment.BookingID, string(payment.Method), payment.Currency, durationSeconds)
		// Add span event for payment completed
		span.AddEvent("payment_completed", trace.WithAttributes(
			attribute.String("payment_id", payment.ID),
			attribute.String("transaction_id", chargeResp.TransactionID),
			attribute.Float64("duration_seconds", durationSeconds),
		))
	} else {
		metrics.RecordPaymentFailed(ctx, payment.BookingID, string(payment.Method), chargeResp.FailureReason)
		// Add span event for payment failed
		span.AddEvent("payment_failed", trace.WithAttributes(
			attribute.String("payment_id", payment.ID),
			attribute.String("failure_reason", chargeResp.FailureReason),
			attribute.Float64("duration_seconds", durationSeconds),
		))
	}

	span.SetStatus(codes.Ok, "")
	return payment, nil
}

// GetPayment retrieves a payment by ID
func (s *paymentServiceImpl) GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.payment.get")
	defer span.End()

	span.SetAttributes(attribute.String("payment_id", paymentID))

	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return payment, nil
}

// GetPaymentByBookingID retrieves a payment by booking ID
func (s *paymentServiceImpl) GetPaymentByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.payment.get_by_booking")
	defer span.End()

	span.SetAttributes(attribute.String("booking_id", bookingID))

	payment, err := s.repo.GetByBookingID(ctx, bookingID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.String("payment_id", payment.ID))
	span.SetStatus(codes.Ok, "")
	return payment, nil
}

// GetUserPayments retrieves all payments for a user
func (s *paymentServiceImpl) GetUserPayments(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.payment.get_user_payments")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
	)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	payments, err := s.repo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("result_count", len(payments)))
	span.SetStatus(codes.Ok, "")
	return payments, nil
}

// RefundPayment refunds a payment
func (s *paymentServiceImpl) RefundPayment(ctx context.Context, paymentID string, reason string) (*domain.Payment, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.payment.refund")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment_id", paymentID),
		attribute.String("refund_reason", reason),
	)

	// Get payment
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("booking_id", payment.BookingID),
		attribute.Float64("amount", payment.Amount),
	)

	// Process refund through gateway using GatewayPaymentID
	if err := s.gateway.Refund(ctx, payment.GatewayPaymentID, payment.Amount); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to process refund: %w", err)
	}

	// Mark as refunded with amount and reason
	if err := payment.Refund(payment.Amount, reason); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to mark payment as refunded: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Record metrics
	metrics.RecordPaymentRefunded(ctx, payment.BookingID, reason, payment.Amount)

	span.SetStatus(codes.Ok, "")
	return payment, nil
}

// CancelPayment cancels a pending payment
func (s *paymentServiceImpl) CancelPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.payment.cancel")
	defer span.End()

	span.SetAttributes(attribute.String("payment_id", paymentID))

	// Get payment
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("booking_id", payment.BookingID),
		attribute.String("current_status", string(payment.Status)),
	)

	// Cancel payment
	if err := payment.Cancel(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to cancel payment: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Record metrics
	metrics.RecordPaymentCancelled(ctx, payment.BookingID)

	span.SetStatus(codes.Ok, "")
	return payment, nil
}

// CompletePaymentFromWebhook marks payment as completed from Stripe webhook
// This is called when payment_intent.succeeded webhook is received
func (s *paymentServiceImpl) CompletePaymentFromWebhook(ctx context.Context, paymentID string, gatewayPaymentID string) (*domain.Payment, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.payment.complete_from_webhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment_id", paymentID),
		attribute.String("gateway_payment_id", gatewayPaymentID),
	)

	// Get payment
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	span.SetAttributes(
		attribute.String("booking_id", payment.BookingID),
		attribute.String("current_status", string(payment.Status)),
	)

	// Skip if already in final state
	if payment.IsFinal() {
		span.SetAttributes(attribute.Bool("skipped_final_state", true))
		span.SetStatus(codes.Ok, "")
		return payment, nil
	}

	// Complete payment directly (no gateway call needed - Stripe already processed it)
	if err := payment.Complete(gatewayPaymentID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to complete payment: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	span.SetAttributes(attribute.String("new_status", string(payment.Status)))
	span.SetStatus(codes.Ok, "")
	return payment, nil
}

// FailPaymentFromWebhook marks payment as failed from Stripe webhook
// This is called when payment_intent.payment_failed webhook is received
func (s *paymentServiceImpl) FailPaymentFromWebhook(ctx context.Context, paymentID string, errorCode string, errorMessage string) (*domain.Payment, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.payment.fail_from_webhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment_id", paymentID),
		attribute.String("error_code", errorCode),
		attribute.String("error_message", errorMessage),
	)

	// Get payment
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	span.SetAttributes(
		attribute.String("booking_id", payment.BookingID),
		attribute.String("current_status", string(payment.Status)),
	)

	// Skip if already in final state
	if payment.IsFinal() {
		span.SetAttributes(attribute.Bool("skipped_final_state", true))
		span.SetStatus(codes.Ok, "")
		return payment, nil
	}

	// Fail payment
	if err := payment.Fail(errorCode, errorMessage); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to mark payment as failed: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	span.SetAttributes(attribute.String("new_status", string(payment.Status)))
	span.SetStatus(codes.Ok, "")
	return payment, nil
}
