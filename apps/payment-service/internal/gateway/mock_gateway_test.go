package gateway

import (
	"context"
	"testing"
)

func TestNewMockGateway(t *testing.T) {
	gw := NewMockGateway(nil)
	if gw == nil {
		t.Fatal("Expected non-nil gateway")
	}

	if gw.Name() != "mock" {
		t.Errorf("Expected name 'mock', got '%s'", gw.Name())
	}
}

func TestMockGateway_Charge_Success(t *testing.T) {
	gw := NewMockGateway(&MockGatewayConfig{
		SuccessRate: 1.0, // 100% success
		DelayMs:     0,
	})

	ctx := context.Background()
	req := &ChargeRequest{
		PaymentID: "pay-123",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    "credit_card",
	}

	resp, err := gw.Charge(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("Expected successful charge")
	}

	if resp.TransactionID == "" {
		t.Error("Expected transaction ID")
	}

	if resp.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", resp.Status)
	}
}

func TestMockGateway_Charge_Failure(t *testing.T) {
	gw := NewMockGateway(&MockGatewayConfig{
		SuccessRate:    0.0, // 0% success
		DelayMs:        0,
		FailureReasons: []string{"card_declined"},
	})

	ctx := context.Background()
	req := &ChargeRequest{
		PaymentID: "pay-123",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    "credit_card",
	}

	resp, err := gw.Charge(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Success {
		t.Error("Expected failed charge")
	}

	if resp.FailureReason == "" {
		t.Error("Expected failure reason")
	}
}

func TestMockGateway_Charge_NilRequest(t *testing.T) {
	gw := NewMockGateway(nil)

	ctx := context.Background()
	_, err := gw.Charge(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}
}

func TestMockGateway_Refund(t *testing.T) {
	gw := NewMockGateway(&MockGatewayConfig{
		SuccessRate: 1.0,
		DelayMs:     0,
	})

	ctx := context.Background()

	// First create a charge
	req := &ChargeRequest{
		PaymentID: "pay-123",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    "credit_card",
	}

	resp, _ := gw.Charge(ctx, req)

	// Now refund
	err := gw.Refund(ctx, resp.TransactionID, 1000.00)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify transaction is refunded
	txn, err := gw.GetTransaction(ctx, resp.TransactionID)
	if err != nil {
		t.Fatalf("Failed to get transaction: %v", err)
	}

	if txn.Status != "refunded" {
		t.Errorf("Expected status 'refunded', got '%s'", txn.Status)
	}
}

func TestMockGateway_Refund_NotFound(t *testing.T) {
	gw := NewMockGateway(nil)

	ctx := context.Background()
	err := gw.Refund(ctx, "non-existent", 1000.00)
	if err == nil {
		t.Error("Expected error for non-existent transaction")
	}
}

func TestMockGateway_GetTransaction(t *testing.T) {
	gw := NewMockGateway(&MockGatewayConfig{
		SuccessRate: 1.0,
		DelayMs:     0,
	})

	ctx := context.Background()

	// Create a charge
	req := &ChargeRequest{
		PaymentID: "pay-123",
		Amount:    500.00,
		Currency:  "THB",
		Method:    "credit_card",
	}

	resp, _ := gw.Charge(ctx, req)

	// Get transaction
	txn, err := gw.GetTransaction(ctx, resp.TransactionID)
	if err != nil {
		t.Fatalf("Failed to get transaction: %v", err)
	}

	if txn.Amount != 500.00 {
		t.Errorf("Expected amount 500.00, got %f", txn.Amount)
	}

	if txn.Currency != "THB" {
		t.Errorf("Expected currency 'THB', got '%s'", txn.Currency)
	}
}

func TestMockGateway_GetTransaction_NotFound(t *testing.T) {
	gw := NewMockGateway(nil)

	ctx := context.Background()
	_, err := gw.GetTransaction(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent transaction")
	}
}

func TestMockGateway_SetSuccessRate(t *testing.T) {
	gw := NewMockGateway(&MockGatewayConfig{
		SuccessRate: 0.5,
	})

	if gw.GetSuccessRate() != 0.5 {
		t.Errorf("Expected success rate 0.5, got %f", gw.GetSuccessRate())
	}

	gw.SetSuccessRate(0.8)
	if gw.GetSuccessRate() != 0.8 {
		t.Errorf("Expected success rate 0.8, got %f", gw.GetSuccessRate())
	}

	// Test bounds
	gw.SetSuccessRate(-0.5)
	if gw.GetSuccessRate() != 0.0 {
		t.Errorf("Expected success rate 0.0, got %f", gw.GetSuccessRate())
	}

	gw.SetSuccessRate(1.5)
	if gw.GetSuccessRate() != 1.0 {
		t.Errorf("Expected success rate 1.0, got %f", gw.GetSuccessRate())
	}
}

func TestNewPaymentGateway_Mock(t *testing.T) {
	gw, err := NewPaymentGateway("mock", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if gw.Name() != "mock" {
		t.Errorf("Expected name 'mock', got '%s'", gw.Name())
	}
}

func TestNewPaymentGateway_Empty(t *testing.T) {
	gw, err := NewPaymentGateway("", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if gw.Name() != "mock" {
		t.Errorf("Expected default to mock, got '%s'", gw.Name())
	}
}

func TestNewPaymentGateway_Stripe_NoKey(t *testing.T) {
	_, err := NewPaymentGateway("stripe", nil)
	if err == nil {
		t.Error("Expected error for stripe without key")
	}
}

func TestNewPaymentGateway_Unknown(t *testing.T) {
	_, err := NewPaymentGateway("unknown", nil)
	if err == nil {
		t.Error("Expected error for unknown gateway type")
	}
}
