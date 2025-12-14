// Package saga provides the booking saga implementation using the Saga pattern.
//
// # Architecture Overview
//
// This implementation uses an EVENT-DRIVEN approach with Kafka:
//
//  1. Saga Orchestrator (cmd/saga-orchestrator):
//     - Consumes events from workers
//     - Manages saga state transitions
//     - Sends commands to workers via Kafka
//
//  2. Step Workers (separate processes):
//     - saga_step_worker: handles reserve-seats, confirm-booking, release-seats
//     - saga-payment-worker: handles process-payment, refund-payment
//
//  3. Flow:
//     API creates saga instance → sends first command to Kafka →
//     Worker executes → sends event → Orchestrator advances saga →
//     sends next command → repeat until complete or compensate
//
// The step functions (reserveSeatsExecute, processPaymentExecute, etc.) in this file
// are PLACEHOLDER implementations used only for:
// - Defining the saga steps structure (name, timeout, retries)
// - Direct execution mode (not currently used in production)
// - Testing purposes
//
// In production, actual step execution happens in the dedicated workers.
package saga

import (
	"context"
	"fmt"
	"time"

	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

const (
	// BookingSagaName is the name of the booking saga (legacy - for backward compatibility)
	BookingSagaName = "booking-saga"

	// PostPaymentSagaName is the name of the post-payment saga
	// This saga runs AFTER payment success (triggered by payment.success Kafka event)
	PostPaymentSagaName = "post-payment-saga"

	// Step names - these are used by both orchestrator and workers
	// Legacy steps (not used in new flow)
	StepReserveSeats   = "reserve-seats"   // Now handled by fast path (Redis Lua)
	StepProcessPayment = "process-payment" // Now handled by Stripe directly

	// Post-payment saga steps
	StepConfirmBooking   = "confirm-booking"   // Update status, remove TTL
	StepSendNotification = "send-notification" // Optional email notification

	// Compensation steps
	StepRefundPayment = "refund-payment" // Refund via Stripe
	StepReleaseSeats  = "release-seats"  // Release seats back to Redis
)

// BookingSagaData contains the data passed through the booking saga
type BookingSagaData struct {
	// Input data
	BookingID      string  `json:"booking_id"`
	UserID         string  `json:"user_id"`
	TenantID       string  `json:"tenant_id"`
	EventID        string  `json:"event_id"`
	ShowID         string  `json:"show_id"`
	ZoneID         string  `json:"zone_id"`
	Quantity       int     `json:"quantity"`
	TotalPrice     float64 `json:"total_price"`
	Currency       string  `json:"currency"`
	PaymentMethod  string  `json:"payment_method"`
	IdempotencyKey string  `json:"idempotency_key,omitempty"`

	// Step outputs
	ReservationID    string `json:"reservation_id,omitempty"`
	PaymentID        string `json:"payment_id,omitempty"`
	ConfirmationCode string `json:"confirmation_code,omitempty"`
	NotificationID   string `json:"notification_id,omitempty"`
}

// ToMap converts BookingSagaData to map[string]interface{}
func (d *BookingSagaData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"booking_id":        d.BookingID,
		"user_id":           d.UserID,
		"tenant_id":         d.TenantID,
		"event_id":          d.EventID,
		"show_id":           d.ShowID,
		"zone_id":           d.ZoneID,
		"quantity":          d.Quantity,
		"total_price":       d.TotalPrice,
		"currency":          d.Currency,
		"payment_method":    d.PaymentMethod,
		"idempotency_key":   d.IdempotencyKey,
		"reservation_id":    d.ReservationID,
		"payment_id":        d.PaymentID,
		"confirmation_code": d.ConfirmationCode,
		"notification_id":   d.NotificationID,
	}
}

// FromMap populates BookingSagaData from map[string]interface{}
func (d *BookingSagaData) FromMap(m map[string]interface{}) {
	if v, ok := m["booking_id"].(string); ok {
		d.BookingID = v
	}
	if v, ok := m["user_id"].(string); ok {
		d.UserID = v
	}
	if v, ok := m["tenant_id"].(string); ok {
		d.TenantID = v
	}
	if v, ok := m["event_id"].(string); ok {
		d.EventID = v
	}
	if v, ok := m["show_id"].(string); ok {
		d.ShowID = v
	}
	if v, ok := m["zone_id"].(string); ok {
		d.ZoneID = v
	}
	if v, ok := m["quantity"].(int); ok {
		d.Quantity = v
	} else if v, ok := m["quantity"].(float64); ok {
		d.Quantity = int(v)
	}
	if v, ok := m["total_price"].(float64); ok {
		d.TotalPrice = v
	}
	if v, ok := m["currency"].(string); ok {
		d.Currency = v
	}
	if v, ok := m["payment_method"].(string); ok {
		d.PaymentMethod = v
	}
	if v, ok := m["idempotency_key"].(string); ok {
		d.IdempotencyKey = v
	}
	if v, ok := m["reservation_id"].(string); ok {
		d.ReservationID = v
	}
	if v, ok := m["payment_id"].(string); ok {
		d.PaymentID = v
	}
	if v, ok := m["confirmation_code"].(string); ok {
		d.ConfirmationCode = v
	}
	if v, ok := m["notification_id"].(string); ok {
		d.NotificationID = v
	}
}

// SeatReservationService defines the interface for seat reservation operations
type SeatReservationService interface {
	ReserveSeats(ctx context.Context, bookingID, userID, eventID, zoneID string, quantity int) (reservationID string, err error)
	ReleaseSeats(ctx context.Context, bookingID, userID string) error
}

// PaymentService defines the interface for payment operations
type PaymentService interface {
	ProcessPayment(ctx context.Context, bookingID, userID string, amount float64, currency, method string) (paymentID string, err error)
	RefundPayment(ctx context.Context, paymentID, reason string) error
}

// BookingConfirmationService defines the interface for booking confirmation
type BookingConfirmationService interface {
	ConfirmBooking(ctx context.Context, bookingID, userID, paymentID string) (confirmationCode string, err error)
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendBookingConfirmation(ctx context.Context, userID, bookingID, confirmationCode string) (notificationID string, err error)
}

// BookingSagaConfig holds configuration for the booking saga
type BookingSagaConfig struct {
	ReservationService SeatReservationService
	PaymentService     PaymentService
	ConfirmationService BookingConfirmationService
	NotificationService NotificationService
	StepTimeout        time.Duration
	MaxRetries         int
}

// BookingSagaBuilder creates a booking saga definition
type BookingSagaBuilder struct {
	config *BookingSagaConfig
}

// NewBookingSagaBuilder creates a new booking saga builder
func NewBookingSagaBuilder(config *BookingSagaConfig) *BookingSagaBuilder {
	if config.StepTimeout == 0 {
		config.StepTimeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 2
	}
	return &BookingSagaBuilder{config: config}
}

// Build creates the booking saga definition
func (b *BookingSagaBuilder) Build() *pkgsaga.Definition {
	def := pkgsaga.NewDefinition(BookingSagaName, "Booking saga for ticket reservation")
	def.WithTimeout(5 * time.Minute)

	// Step 1: Reserve Seats
	def.AddStep(&pkgsaga.Step{
		Name:        StepReserveSeats,
		Description: "Reserve seats in inventory",
		Execute:     b.reserveSeatsExecute,
		Compensate:  b.reserveSeatsCompensate,
		Timeout:     b.config.StepTimeout,
		Retries:     b.config.MaxRetries,
	})

	// Step 2: Process Payment
	def.AddStep(&pkgsaga.Step{
		Name:        StepProcessPayment,
		Description: "Process payment for booking",
		Execute:     b.processPaymentExecute,
		Compensate:  b.processPaymentCompensate,
		Timeout:     b.config.StepTimeout,
		Retries:     b.config.MaxRetries,
	})

	// Step 3: Confirm Booking
	def.AddStep(&pkgsaga.Step{
		Name:        StepConfirmBooking,
		Description: "Confirm booking after payment",
		Execute:     b.confirmBookingExecute,
		Compensate:  nil, // No compensation needed - if this fails, payment will be refunded
		Timeout:     b.config.StepTimeout,
		Retries:     b.config.MaxRetries,
	})

	// Step 4: Send Notification - TODO: Enable when notification service is ready
	// def.AddStep(&pkgsaga.Step{
	// 	Name:        StepSendNotification,
	// 	Description: "Send booking confirmation notification",
	// 	Execute:     b.sendNotificationExecute,
	// 	Compensate:  nil, // Notification failure is not critical
	// 	Timeout:     b.config.StepTimeout,
	// 	Retries:     b.config.MaxRetries,
	// })

	return def
}

// Step 1: Reserve Seats - Execute
func (b *BookingSagaBuilder) reserveSeatsExecute(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	sagaData := &BookingSagaData{}
	sagaData.FromMap(data)

	if b.config.ReservationService == nil {
		return nil, fmt.Errorf("reservation service is not configured")
	}

	reservationID, err := b.config.ReservationService.ReserveSeats(
		ctx,
		sagaData.BookingID,
		sagaData.UserID,
		sagaData.EventID,
		sagaData.ZoneID,
		sagaData.Quantity,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve seats: %w", err)
	}

	return map[string]interface{}{
		"reservation_id": reservationID,
	}, nil
}

// Step 1: Reserve Seats - Compensate (Release)
func (b *BookingSagaBuilder) reserveSeatsCompensate(ctx context.Context, data map[string]interface{}) error {
	sagaData := &BookingSagaData{}
	sagaData.FromMap(data)

	if b.config.ReservationService == nil {
		return fmt.Errorf("reservation service is not configured")
	}

	if err := b.config.ReservationService.ReleaseSeats(ctx, sagaData.BookingID, sagaData.UserID); err != nil {
		return fmt.Errorf("failed to release seats: %w", err)
	}

	return nil
}

// Step 2: Process Payment - Execute
func (b *BookingSagaBuilder) processPaymentExecute(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	sagaData := &BookingSagaData{}
	sagaData.FromMap(data)

	if b.config.PaymentService == nil {
		return nil, fmt.Errorf("payment service is not configured")
	}

	paymentID, err := b.config.PaymentService.ProcessPayment(
		ctx,
		sagaData.BookingID,
		sagaData.UserID,
		sagaData.TotalPrice,
		sagaData.Currency,
		sagaData.PaymentMethod,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}

	return map[string]interface{}{
		"payment_id": paymentID,
	}, nil
}

// Step 2: Process Payment - Compensate (Refund)
func (b *BookingSagaBuilder) processPaymentCompensate(ctx context.Context, data map[string]interface{}) error {
	sagaData := &BookingSagaData{}
	sagaData.FromMap(data)

	if b.config.PaymentService == nil {
		return fmt.Errorf("payment service is not configured")
	}

	if sagaData.PaymentID == "" {
		// No payment was made, nothing to refund
		return nil
	}

	if err := b.config.PaymentService.RefundPayment(ctx, sagaData.PaymentID, "Booking saga compensation"); err != nil {
		return fmt.Errorf("failed to refund payment: %w", err)
	}

	return nil
}

// Step 3: Confirm Booking - Execute
func (b *BookingSagaBuilder) confirmBookingExecute(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	sagaData := &BookingSagaData{}
	sagaData.FromMap(data)

	if b.config.ConfirmationService == nil {
		return nil, fmt.Errorf("confirmation service is not configured")
	}

	confirmationCode, err := b.config.ConfirmationService.ConfirmBooking(
		ctx,
		sagaData.BookingID,
		sagaData.UserID,
		sagaData.PaymentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm booking: %w", err)
	}

	return map[string]interface{}{
		"confirmation_code": confirmationCode,
	}, nil
}

// Step 4: Send Notification - Execute
func (b *BookingSagaBuilder) sendNotificationExecute(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	sagaData := &BookingSagaData{}
	sagaData.FromMap(data)

	if b.config.NotificationService == nil {
		// Notification is optional, return success if not configured
		return nil, nil
	}

	notificationID, err := b.config.NotificationService.SendBookingConfirmation(
		ctx,
		sagaData.UserID,
		sagaData.BookingID,
		sagaData.ConfirmationCode,
	)
	if err != nil {
		// Log error but don't fail the saga for notification failure
		return map[string]interface{}{
			"notification_error": err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"notification_id": notificationID,
	}, nil
}

// ============================================================================
// POST-PAYMENT SAGA - Runs after payment success (triggered by webhook)
// ============================================================================

// PostPaymentSagaConfig holds configuration for the post-payment saga
type PostPaymentSagaConfig struct {
	StepTimeout time.Duration
	MaxRetries  int
}

// PostPaymentSagaBuilder creates a post-payment saga definition
type PostPaymentSagaBuilder struct {
	config *PostPaymentSagaConfig
}

// NewPostPaymentSagaBuilder creates a new post-payment saga builder
func NewPostPaymentSagaBuilder(config *PostPaymentSagaConfig) *PostPaymentSagaBuilder {
	if config == nil {
		config = &PostPaymentSagaConfig{}
	}
	if config.StepTimeout == 0 {
		config.StepTimeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	return &PostPaymentSagaBuilder{config: config}
}

// Build creates the post-payment saga definition
// This saga runs AFTER payment success and confirms the booking
func (b *PostPaymentSagaBuilder) Build() *pkgsaga.Definition {
	def := pkgsaga.NewDefinition(PostPaymentSagaName, "Post-payment booking confirmation saga")
	def.WithTimeout(1 * time.Minute)

	// Step 1: Confirm Booking
	// - Update booking status to confirmed in PostgreSQL
	// - Remove TTL from Redis (make reservation permanent)
	// - Generate confirmation code
	def.AddStep(&pkgsaga.Step{
		Name:        StepConfirmBooking,
		Description: "Confirm booking after payment success",
		Execute:     nil, // Executed by saga_step_worker
		Compensate:  nil, // Compensation handled separately (refund + release)
		Timeout:     b.config.StepTimeout,
		Retries:     b.config.MaxRetries,
	})

	// Step 2: Send Notification (NON-CRITICAL)
	// - Send booking confirmation email/SMS
	// - If fails: Retry → DLQ (NO refund, NO seat release)
	// - Compensate is nil because notification failure should NOT trigger rollback
	def.AddStep(&pkgsaga.Step{
		Name:        StepSendNotification,
		Description: "Send booking confirmation notification",
		Execute:     nil, // Executed by saga_step_worker (mock for now)
		Compensate:  nil, // NON-CRITICAL: No compensation - just retry and DLQ
		Timeout:     b.config.StepTimeout,
		Retries:     5, // More retries for non-critical step
	})

	return def
}
