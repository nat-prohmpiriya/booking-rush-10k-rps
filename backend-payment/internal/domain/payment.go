package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment (matches DB ENUM)
type PaymentStatus string

const (
	PaymentStatusPending       PaymentStatus = "pending"
	PaymentStatusProcessing    PaymentStatus = "processing"
	PaymentStatusSucceeded     PaymentStatus = "succeeded"
	PaymentStatusFailed        PaymentStatus = "failed"
	PaymentStatusCancelled     PaymentStatus = "cancelled"
	PaymentStatusRefundPending PaymentStatus = "refund_pending"
	PaymentStatusRefunded      PaymentStatus = "refunded"
)

// PaymentMethod represents the method of payment (matches DB ENUM)
type PaymentMethod string

const (
	PaymentMethodCreditCard   PaymentMethod = "credit_card"
	PaymentMethodDebitCard    PaymentMethod = "debit_card"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodPromptPay    PaymentMethod = "promptpay"
	PaymentMethodWallet       PaymentMethod = "wallet"
	PaymentMethodCash         PaymentMethod = "cash"
)

// Payment represents a payment entity (matches microservice schema)
type Payment struct {
	ID                string            `json:"id"`
	TenantID          string            `json:"tenant_id"`
	BookingID         string            `json:"booking_id"`
	UserID            string            `json:"user_id"`
	Amount            float64           `json:"amount"`
	Currency          string            `json:"currency"`
	Method            PaymentMethod     `json:"method,omitempty"`
	Status            PaymentStatus     `json:"status"`
	Gateway           string            `json:"gateway,omitempty"`
	GatewayPaymentID  string            `json:"gateway_payment_id,omitempty"`
	GatewayChargeID   string            `json:"gateway_charge_id,omitempty"`
	GatewayCustomerID string            `json:"gateway_customer_id,omitempty"`
	GatewayResponse   map[string]any    `json:"gateway_response,omitempty"`
	IdempotencyKey    string            `json:"idempotency_key,omitempty"`
	CardLastFour      string            `json:"card_last_four,omitempty"`
	CardBrand         string            `json:"card_brand,omitempty"`
	InitiatedAt       *time.Time        `json:"initiated_at,omitempty"`
	ProcessedAt       *time.Time        `json:"processed_at,omitempty"`
	RefundAmount      *float64          `json:"refund_amount,omitempty"`
	RefundReason      string            `json:"refund_reason,omitempty"`
	RefundedAt        *time.Time        `json:"refunded_at,omitempty"`
	ErrorCode         string            `json:"error_code,omitempty"`
	ErrorMessage      string            `json:"error_message,omitempty"`
	RetryCount        int               `json:"retry_count"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// NewPayment creates a new payment
func NewPayment(tenantID, bookingID, userID string, amount float64, currency string, method PaymentMethod) (*Payment, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}
	if bookingID == "" {
		return nil, errors.New("booking_id is required")
	}
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if currency == "" {
		currency = "THB"
	}

	now := time.Now().UTC()
	return &Payment{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		BookingID:   bookingID,
		UserID:      userID,
		Amount:      amount,
		Currency:    currency,
		Status:      PaymentStatusPending,
		Method:      method,
		Gateway:     "stripe",
		InitiatedAt: &now,
		Metadata:    make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// MarkProcessing marks the payment as processing
func (p *Payment) MarkProcessing() error {
	if p.Status != PaymentStatusPending {
		return errors.New("payment must be pending to start processing")
	}
	p.Status = PaymentStatusProcessing
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Complete marks the payment as succeeded
func (p *Payment) Complete(gatewayPaymentID string) error {
	if p.Status != PaymentStatusProcessing && p.Status != PaymentStatusPending {
		return errors.New("payment must be pending or processing to complete")
	}
	now := time.Now().UTC()
	p.Status = PaymentStatusSucceeded
	p.GatewayPaymentID = gatewayPaymentID
	p.UpdatedAt = now
	p.ProcessedAt = &now
	return nil
}

// Fail marks the payment as failed
func (p *Payment) Fail(errorCode, errorMessage string) error {
	if p.Status != PaymentStatusPending && p.Status != PaymentStatusProcessing {
		return errors.New("payment can only fail from pending or processing status")
	}
	p.Status = PaymentStatusFailed
	p.ErrorCode = errorCode
	p.ErrorMessage = errorMessage
	p.RetryCount++
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Refund marks the payment as refunded
func (p *Payment) Refund(amount float64, reason string) error {
	if p.Status != PaymentStatusSucceeded {
		return errors.New("only succeeded payments can be refunded")
	}
	now := time.Now().UTC()
	p.Status = PaymentStatusRefunded
	p.RefundAmount = &amount
	p.RefundReason = reason
	p.RefundedAt = &now
	p.UpdatedAt = now
	return nil
}

// MarkRefundPending marks the payment as refund pending
func (p *Payment) MarkRefundPending() error {
	if p.Status != PaymentStatusSucceeded {
		return errors.New("only succeeded payments can have pending refund")
	}
	p.Status = PaymentStatusRefundPending
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Cancel marks the payment as cancelled
func (p *Payment) Cancel() error {
	if p.Status != PaymentStatusPending {
		return errors.New("only pending payments can be cancelled")
	}
	p.Status = PaymentStatusCancelled
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// IsFinal returns true if the payment is in a final state
func (p *Payment) IsFinal() bool {
	return p.Status == PaymentStatusSucceeded ||
		p.Status == PaymentStatusFailed ||
		p.Status == PaymentStatusRefunded ||
		p.Status == PaymentStatusCancelled
}

// IsSuccessful returns true if the payment was successful
func (p *Payment) IsSuccessful() bool {
	return p.Status == PaymentStatusSucceeded
}

// SetGatewayInfo sets gateway-related information
func (p *Payment) SetGatewayInfo(gateway, paymentID, chargeID, customerID string) {
	p.Gateway = gateway
	p.GatewayPaymentID = paymentID
	p.GatewayChargeID = chargeID
	p.GatewayCustomerID = customerID
	p.UpdatedAt = time.Now().UTC()
}

// SetCardInfo sets card information
func (p *Payment) SetCardInfo(lastFour, brand string) {
	p.CardLastFour = lastFour
	p.CardBrand = brand
	p.UpdatedAt = time.Now().UTC()
}
