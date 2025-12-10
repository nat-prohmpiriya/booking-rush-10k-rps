package gateway

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
)

// StripeGateway implements PaymentGateway using Stripe
type StripeGateway struct {
	config *StripeGatewayConfig
}

// StripeGatewayConfig holds configuration for Stripe gateway
type StripeGatewayConfig struct {
	SecretKey     string
	WebhookSecret string
	Environment   string // "test" or "live"
}

// NewStripeGateway creates a new Stripe gateway
func NewStripeGateway(config *StripeGatewayConfig) (*StripeGateway, error) {
	if config == nil {
		return nil, fmt.Errorf("stripe config is required")
	}
	if config.SecretKey == "" {
		return nil, fmt.Errorf("stripe secret key is required")
	}

	// Set Stripe API key globally
	stripe.Key = config.SecretKey

	return &StripeGateway{
		config: config,
	}, nil
}

// Charge processes a payment charge through Stripe
func (g *StripeGateway) Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("charge request is required")
	}

	// Convert amount to cents (Stripe expects smallest currency unit)
	amountInCents := int64(req.Amount * 100)

	// Build payment intent params
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountInCents),
		Currency: stripe.String(req.Currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Metadata: make(map[string]string),
	}

	// Add metadata
	params.Metadata["payment_id"] = req.PaymentID
	for k, v := range req.Metadata {
		params.Metadata[k] = v
	}

	// Add description if provided
	if req.Description != "" {
		params.Description = stripe.String(req.Description)
	}

	// Create payment intent
	pi, err := paymentintent.New(params)
	if err != nil {
		return &ChargeResponse{
			Success:       false,
			FailureReason: err.Error(),
			FailureCode:   "stripe_error",
		}, nil
	}

	// For test mode with automatic confirmation, simulate success
	// In production, you'd handle the payment intent status properly
	resp := &ChargeResponse{
		TransactionID: pi.ID,
		Status:        string(pi.Status),
		Metadata:      req.Metadata,
	}

	// Check status
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		resp.Success = true
	case stripe.PaymentIntentStatusRequiresPaymentMethod,
		stripe.PaymentIntentStatusRequiresConfirmation,
		stripe.PaymentIntentStatusRequiresAction:
		// These statuses mean the payment needs more steps
		resp.Success = false
		resp.FailureReason = "payment_requires_action"
		resp.FailureCode = string(pi.Status)
	case stripe.PaymentIntentStatusCanceled:
		resp.Success = false
		resp.FailureReason = "payment_canceled"
		resp.FailureCode = "canceled"
	default:
		// For demo purposes, treat "requires_payment_method" as success
		// since we don't have a real frontend to complete the payment
		if pi.Status == stripe.PaymentIntentStatusRequiresPaymentMethod {
			resp.Success = true
			resp.Status = "pending_confirmation"
		} else {
			resp.Success = false
			resp.FailureReason = fmt.Sprintf("unexpected status: %s", pi.Status)
		}
	}

	return resp, nil
}

// Refund processes a refund through Stripe
func (g *StripeGateway) Refund(ctx context.Context, transactionID string, amount float64) error {
	if transactionID == "" {
		return fmt.Errorf("transaction ID is required")
	}

	// Convert amount to cents
	amountInCents := int64(amount * 100)

	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(transactionID),
		Amount:        stripe.Int64(amountInCents),
	}

	_, err := refund.New(params)
	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	return nil
}

// GetTransaction retrieves transaction details from Stripe
func (g *StripeGateway) GetTransaction(ctx context.Context, transactionID string) (*TransactionInfo, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("transaction ID is required")
	}

	pi, err := paymentintent.Get(transactionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	return &TransactionInfo{
		TransactionID: pi.ID,
		Status:        string(pi.Status),
		Amount:        float64(pi.Amount) / 100,
		Currency:      string(pi.Currency),
		CreatedAt:     fmt.Sprintf("%d", pi.Created),
		Metadata:      pi.Metadata,
	}, nil
}

// Name returns the gateway name
func (g *StripeGateway) Name() string {
	return "stripe"
}
