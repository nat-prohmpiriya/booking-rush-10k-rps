package gateway

import (
	"context"
)

// PaymentGateway defines the interface for payment processing
type PaymentGateway interface {
	// Charge processes a payment charge
	Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error)

	// Refund processes a refund
	Refund(ctx context.Context, transactionID string, amount float64) error

	// GetTransaction retrieves transaction details
	GetTransaction(ctx context.Context, transactionID string) (*TransactionInfo, error)

	// Name returns the gateway name
	Name() string
}

// ChargeRequest represents a charge request
type ChargeRequest struct {
	PaymentID   string
	Amount      float64
	Currency    string
	Method      string
	Description string
	Metadata    map[string]string

	// Card details (for direct card payments)
	CardToken string

	// Customer info
	CustomerID    string
	CustomerEmail string
}

// ChargeResponse represents a charge response
type ChargeResponse struct {
	Success       bool
	TransactionID string
	Status        string
	FailureReason string
	FailureCode   string
	Metadata      map[string]string
}

// TransactionInfo represents transaction details
type TransactionInfo struct {
	TransactionID string
	Status        string
	Amount        float64
	Currency      string
	Method        string
	CreatedAt     string
	Metadata      map[string]string
}

// GatewayConfig holds common gateway configuration
type GatewayConfig struct {
	APIKey       string
	SecretKey    string
	WebhookSecret string
	Environment  string // "test" or "live"
}
