package gateway

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

// alphanumericChars for generating Stripe-compatible IDs
const alphanumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// randomAlphanumeric generates a random alphanumeric string of given length
func randomAlphanumeric(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = alphanumericChars[rand.Intn(len(alphanumericChars))]
	}
	return string(b)
}

// MockGateway implements PaymentGateway for testing and load testing
type MockGateway struct {
	config       *MockGatewayConfig
	transactions sync.Map
	mu           sync.RWMutex
}

// MockGatewayConfig holds configuration for the mock gateway
type MockGatewayConfig struct {
	// SuccessRate is the probability of successful payment (0.0 to 1.0)
	SuccessRate float64

	// DelayMs is the simulated processing delay in milliseconds
	DelayMs int

	// FailureReasons is a list of possible failure reasons
	FailureReasons []string
}

// DefaultMockGatewayConfig returns default configuration
func DefaultMockGatewayConfig() *MockGatewayConfig {
	return &MockGatewayConfig{
		SuccessRate: 0.95, // 95% success rate
		DelayMs:     100,  // 100ms delay
		FailureReasons: []string{
			"insufficient_funds",
			"card_declined",
			"expired_card",
			"processing_error",
			"fraud_detected",
		},
	}
}

// NewMockGateway creates a new mock gateway
func NewMockGateway(config *MockGatewayConfig) *MockGateway {
	if config == nil {
		config = DefaultMockGatewayConfig()
	}

	// Validate success rate
	if config.SuccessRate < 0 {
		config.SuccessRate = 0
	}
	if config.SuccessRate > 1 {
		config.SuccessRate = 1
	}

	return &MockGateway{
		config: config,
	}
}

// Charge processes a mock payment charge
func (g *MockGateway) Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("charge request is required")
	}

	// Simulate processing delay
	if g.config.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(g.config.DelayMs) * time.Millisecond):
		}
	}

	// Generate transaction ID
	transactionID := fmt.Sprintf("mock_txn_%s", uuid.New().String()[:8])

	// Determine success or failure
	success := rand.Float64() < g.config.SuccessRate

	resp := &ChargeResponse{
		TransactionID: transactionID,
		Metadata:      req.Metadata,
	}

	if success {
		resp.Success = true
		resp.Status = "completed"

		// Store transaction
		g.transactions.Store(transactionID, &TransactionInfo{
			TransactionID: transactionID,
			Status:        "completed",
			Amount:        req.Amount,
			Currency:      req.Currency,
			Method:        req.Method,
			CreatedAt:     time.Now().Format(time.RFC3339),
			Metadata:      req.Metadata,
		})
	} else {
		resp.Success = false
		resp.Status = "failed"

		// Pick a random failure reason
		if len(g.config.FailureReasons) > 0 {
			idx := rand.Intn(len(g.config.FailureReasons))
			resp.FailureReason = g.config.FailureReasons[idx]
			resp.FailureCode = resp.FailureReason
		} else {
			resp.FailureReason = "payment_failed"
			resp.FailureCode = "payment_failed"
		}
	}

	return resp, nil
}

// Refund processes a mock refund
func (g *MockGateway) Refund(ctx context.Context, transactionID string, amount float64) error {
	if transactionID == "" {
		return fmt.Errorf("transaction ID is required")
	}

	// Simulate processing delay
	if g.config.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(g.config.DelayMs) * time.Millisecond):
		}
	}

	// Check if transaction exists
	txn, ok := g.transactions.Load(transactionID)
	if !ok {
		return fmt.Errorf("transaction not found: %s", transactionID)
	}

	// Update transaction status
	info := txn.(*TransactionInfo)
	info.Status = "refunded"
	g.transactions.Store(transactionID, info)

	return nil
}

// GetTransaction retrieves transaction details
func (g *MockGateway) GetTransaction(ctx context.Context, transactionID string) (*TransactionInfo, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("transaction ID is required")
	}

	txn, ok := g.transactions.Load(transactionID)
	if !ok {
		return nil, fmt.Errorf("transaction not found: %s", transactionID)
	}

	return txn.(*TransactionInfo), nil
}

// Name returns the gateway name
func (g *MockGateway) Name() string {
	return "mock"
}

// SetSuccessRate updates the success rate (for testing)
func (g *MockGateway) SetSuccessRate(rate float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	g.config.SuccessRate = rate
}

// GetSuccessRate returns the current success rate
func (g *MockGateway) GetSuccessRate() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.config.SuccessRate
}

// CreatePaymentIntent creates a mock PaymentIntent
func (g *MockGateway) CreatePaymentIntent(ctx context.Context, req *PaymentIntentRequest) (*PaymentIntentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("payment intent request is required")
	}

	// Simulate processing delay
	if g.config.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(g.config.DelayMs) * time.Millisecond):
		}
	}

	// Generate mock IDs - Stripe-compatible format (alphanumeric only)
	paymentIntentID := fmt.Sprintf("pi_mock_%s", randomAlphanumeric(24))
	clientSecret := fmt.Sprintf("%s_secret_%s", paymentIntentID, randomAlphanumeric(24))

	// Store mock payment intent
	g.transactions.Store(paymentIntentID, &TransactionInfo{
		TransactionID: paymentIntentID,
		Status:        "requires_payment_method",
		Amount:        req.Amount,
		Currency:      req.Currency,
		CreatedAt:     time.Now().Format(time.RFC3339),
		Metadata:      req.Metadata,
	})

	return &PaymentIntentResponse{
		PaymentIntentID: paymentIntentID,
		ClientSecret:    clientSecret,
		Status:          "requires_payment_method",
		Amount:          req.Amount,
		Currency:        req.Currency,
	}, nil
}

// ConfirmPaymentIntent confirms a mock PaymentIntent
func (g *MockGateway) ConfirmPaymentIntent(ctx context.Context, paymentIntentID string) (*PaymentIntentResponse, error) {
	if paymentIntentID == "" {
		return nil, fmt.Errorf("payment intent ID is required")
	}

	// Simulate processing delay
	if g.config.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(g.config.DelayMs) * time.Millisecond):
		}
	}

	// Get the payment intent
	txn, ok := g.transactions.Load(paymentIntentID)
	if !ok {
		return nil, fmt.Errorf("payment intent not found: %s", paymentIntentID)
	}

	info := txn.(*TransactionInfo)

	// Determine success or failure
	success := rand.Float64() < g.config.SuccessRate

	if success {
		info.Status = "succeeded"
	} else {
		info.Status = "failed"
	}

	g.transactions.Store(paymentIntentID, info)

	return &PaymentIntentResponse{
		PaymentIntentID: paymentIntentID,
		ClientSecret:    "",
		Status:          info.Status,
		Amount:          info.Amount,
		Currency:        info.Currency,
	}, nil
}

// CreateCustomer creates a mock Stripe Customer
func (g *MockGateway) CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*CustomerResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create customer request is required")
	}
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Simulate processing delay
	if g.config.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(g.config.DelayMs) * time.Millisecond):
		}
	}

	// Generate mock customer ID
	customerID := fmt.Sprintf("cus_mock_%s", uuid.New().String()[:12])

	return &CustomerResponse{
		CustomerID: customerID,
		Email:      req.Email,
		Name:       req.Name,
	}, nil
}

// CreatePortalSession creates a mock Customer Portal session
func (g *MockGateway) CreatePortalSession(ctx context.Context, req *PortalSessionRequest) (*PortalSessionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("portal session request is required")
	}
	if req.CustomerID == "" {
		return nil, fmt.Errorf("customer ID is required")
	}
	if req.ReturnURL == "" {
		return nil, fmt.Errorf("return URL is required")
	}

	// Simulate processing delay
	if g.config.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(g.config.DelayMs) * time.Millisecond):
		}
	}

	// Generate mock portal URL
	portalURL := fmt.Sprintf("https://billing.stripe.com/mock/session/%s", uuid.New().String()[:16])

	return &PortalSessionResponse{
		URL: portalURL,
	}, nil
}

// ListPaymentMethods returns mock saved payment methods
func (g *MockGateway) ListPaymentMethods(ctx context.Context, customerID string) ([]*PaymentMethodInfo, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customer ID is required")
	}

	// Simulate processing delay
	if g.config.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(g.config.DelayMs) * time.Millisecond):
		}
	}

	// Return mock saved cards
	return []*PaymentMethodInfo{
		{
			ID:        "pm_mock_visa_4242",
			Type:      "card",
			Brand:     "visa",
			Last4:     "4242",
			ExpMonth:  12,
			ExpYear:   2025,
			IsDefault: true,
		},
		{
			ID:        "pm_mock_mastercard_5555",
			Type:      "card",
			Brand:     "mastercard",
			Last4:     "5555",
			ExpMonth:  6,
			ExpYear:   2026,
			IsDefault: false,
		},
	}, nil
}
