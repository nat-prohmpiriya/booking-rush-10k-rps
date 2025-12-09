package dto

import (
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
)

// CreatePaymentRequest represents a request to create a payment
type CreatePaymentRequest struct {
	BookingID string             `json:"booking_id" binding:"required"`
	Amount    float64            `json:"amount" binding:"required,gt=0"`
	Currency  string             `json:"currency" binding:"required"`
	Method    domain.PaymentMethod `json:"method" binding:"required"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
}

// ProcessPaymentRequest represents a request to process a payment
type ProcessPaymentRequest struct {
	PaymentID string `json:"payment_id" binding:"required"`
	// Additional fields for payment gateway
	CardToken   string `json:"card_token,omitempty"`
	ReturnURL   string `json:"return_url,omitempty"`
	WebhookURL  string `json:"webhook_url,omitempty"`
}

// RefundPaymentRequest represents a request to refund a payment
type RefundPaymentRequest struct {
	PaymentID string  `json:"payment_id" binding:"required"`
	Amount    float64 `json:"amount,omitempty"` // Optional: partial refund amount
	Reason    string  `json:"reason,omitempty"`
}

// PaymentResponse represents a payment response
type PaymentResponse struct {
	ID            string               `json:"id"`
	BookingID     string               `json:"booking_id"`
	UserID        string               `json:"user_id"`
	Amount        float64              `json:"amount"`
	Currency      string               `json:"currency"`
	Status        domain.PaymentStatus `json:"status"`
	Method        domain.PaymentMethod `json:"method"`
	TransactionID string               `json:"transaction_id,omitempty"`
	FailureReason string               `json:"failure_reason,omitempty"`
	Metadata      map[string]string    `json:"metadata,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	CompletedAt   *time.Time           `json:"completed_at,omitempty"`
}

// FromPayment converts a domain Payment to PaymentResponse
func FromPayment(p *domain.Payment) *PaymentResponse {
	return &PaymentResponse{
		ID:            p.ID,
		BookingID:     p.BookingID,
		UserID:        p.UserID,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Status:        p.Status,
		Method:        p.Method,
		TransactionID: p.TransactionID,
		FailureReason: p.FailureReason,
		Metadata:      p.Metadata,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		CompletedAt:   p.CompletedAt,
	}
}

// PaymentListResponse represents a list of payments
type PaymentListResponse struct {
	Payments []*PaymentResponse `json:"payments"`
	Total    int                `json:"total"`
}
