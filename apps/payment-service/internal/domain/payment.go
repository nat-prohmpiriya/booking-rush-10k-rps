package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// PaymentMethod represents the method of payment
type PaymentMethod string

const (
	PaymentMethodCreditCard PaymentMethod = "credit_card"
	PaymentMethodDebitCard  PaymentMethod = "debit_card"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodEWallet    PaymentMethod = "e_wallet"
)

// Payment represents a payment entity
type Payment struct {
	ID            string        `json:"id"`
	BookingID     string        `json:"booking_id"`
	UserID        string        `json:"user_id"`
	Amount        float64       `json:"amount"`
	Currency      string        `json:"currency"`
	Status        PaymentStatus `json:"status"`
	Method        PaymentMethod `json:"method"`
	TransactionID string        `json:"transaction_id,omitempty"`
	FailureReason string        `json:"failure_reason,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
}

// NewPayment creates a new payment
func NewPayment(bookingID, userID string, amount float64, currency string, method PaymentMethod) (*Payment, error) {
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
		return nil, errors.New("currency is required")
	}

	now := time.Now().UTC()
	return &Payment{
		ID:        uuid.New().String(),
		BookingID: bookingID,
		UserID:    userID,
		Amount:    amount,
		Currency:  currency,
		Status:    PaymentStatusPending,
		Method:    method,
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
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

// Complete marks the payment as completed
func (p *Payment) Complete(transactionID string) error {
	if p.Status != PaymentStatusProcessing {
		return errors.New("payment must be processing to complete")
	}
	if transactionID == "" {
		return errors.New("transaction_id is required")
	}
	now := time.Now().UTC()
	p.Status = PaymentStatusCompleted
	p.TransactionID = transactionID
	p.UpdatedAt = now
	p.CompletedAt = &now
	return nil
}

// Fail marks the payment as failed
func (p *Payment) Fail(reason string) error {
	if p.Status != PaymentStatusPending && p.Status != PaymentStatusProcessing {
		return errors.New("payment can only fail from pending or processing status")
	}
	p.Status = PaymentStatusFailed
	p.FailureReason = reason
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Refund marks the payment as refunded
func (p *Payment) Refund() error {
	if p.Status != PaymentStatusCompleted {
		return errors.New("only completed payments can be refunded")
	}
	p.Status = PaymentStatusRefunded
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
	return p.Status == PaymentStatusCompleted ||
		p.Status == PaymentStatusFailed ||
		p.Status == PaymentStatusRefunded ||
		p.Status == PaymentStatusCancelled
}

// IsSuccessful returns true if the payment was successful
func (p *Payment) IsSuccessful() bool {
	return p.Status == PaymentStatusCompleted
}
