package gateway

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

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
