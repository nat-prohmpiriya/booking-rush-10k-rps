package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/gateway"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/service"
)

// mockPaymentService implements service.PaymentService for testing
type mockPaymentService struct {
	createPaymentFunc  func(ctx context.Context, req *service.CreatePaymentRequest) (*domain.Payment, error)
	processPaymentFunc func(ctx context.Context, paymentID string) (*domain.Payment, error)
}

func (m *mockPaymentService) CreatePayment(ctx context.Context, req *service.CreatePaymentRequest) (*domain.Payment, error) {
	if m.createPaymentFunc != nil {
		return m.createPaymentFunc(ctx, req)
	}
	payment, _ := domain.NewPayment(req.BookingID, req.UserID, req.Amount, req.Currency, req.Method)
	return payment, nil
}

func (m *mockPaymentService) ProcessPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	if m.processPaymentFunc != nil {
		return m.processPaymentFunc(ctx, paymentID)
	}
	payment, _ := domain.NewPayment("booking-123", "user-123", 100, "THB", domain.PaymentMethodCreditCard)
	payment.MarkProcessing()
	payment.Complete("txn-123")
	return payment, nil
}

func (m *mockPaymentService) GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	return nil, nil
}

func (m *mockPaymentService) GetPaymentByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error) {
	return nil, nil
}

func (m *mockPaymentService) GetUserPayments(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	return nil, nil
}

func (m *mockPaymentService) RefundPayment(ctx context.Context, paymentID string, reason string) (*domain.Payment, error) {
	return nil, nil
}

func (m *mockPaymentService) CancelPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	return nil, nil
}

func TestBookingEventUnmarshal(t *testing.T) {
	jsonData := `{
		"event_id": "evt-123",
		"event_type": "booking.created",
		"occurred_at": "2024-01-01T12:00:00Z",
		"version": 1,
		"data": {
			"booking_id": "booking-123",
			"user_id": "user-456",
			"event_id": "event-789",
			"zone_id": "zone-A",
			"quantity": 2,
			"unit_price": 500.00,
			"total_price": 1000.00,
			"currency": "THB",
			"status": "pending"
		}
	}`

	var event BookingEvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("Failed to unmarshal booking event: %v", err)
	}

	if event.EventID != "evt-123" {
		t.Errorf("Expected event_id 'evt-123', got '%s'", event.EventID)
	}

	if event.EventType != BookingEventCreated {
		t.Errorf("Expected event_type 'booking.created', got '%s'", event.EventType)
	}

	if event.BookingData == nil {
		t.Fatal("Expected booking data to be present")
	}

	if event.BookingData.BookingID != "booking-123" {
		t.Errorf("Expected booking_id 'booking-123', got '%s'", event.BookingData.BookingID)
	}

	if event.BookingData.TotalPrice != 1000.00 {
		t.Errorf("Expected total_price 1000.00, got %f", event.BookingData.TotalPrice)
	}
}

func TestPaymentEventMarshal(t *testing.T) {
	event := &PaymentEvent{
		EventID:    "evt-123",
		EventType:  PaymentEventSuccess,
		OccurredAt: time.Now(),
		Version:    1,
		PaymentData: &PaymentEventData{
			PaymentID:     "pay-123",
			BookingID:     "booking-456",
			UserID:        "user-789",
			Amount:        1000.00,
			Currency:      "THB",
			Status:        "completed",
			Method:        "credit_card",
			TransactionID: "txn-abc",
			ProcessedAt:   time.Now(),
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal payment event: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty JSON data")
	}

	// Verify it can be unmarshalled back
	var parsed PaymentEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal payment event: %v", err)
	}

	if parsed.PaymentData.PaymentID != "pay-123" {
		t.Errorf("Expected payment_id 'pay-123', got '%s'", parsed.PaymentData.PaymentID)
	}
}

func TestPaymentEventTopic(t *testing.T) {
	event := &PaymentEvent{
		EventID:   "evt-123",
		EventType: PaymentEventSuccess,
	}

	if event.Topic() != "payment-events" {
		t.Errorf("Expected topic 'payment-events', got '%s'", event.Topic())
	}
}

func TestPaymentEventKey(t *testing.T) {
	event := &PaymentEvent{
		EventID:   "evt-123",
		EventType: PaymentEventSuccess,
		PaymentData: &PaymentEventData{
			BookingID: "booking-456",
		},
	}

	if event.Key() != "booking-456" {
		t.Errorf("Expected key 'booking-456', got '%s'", event.Key())
	}

	// Without payment data, should return event ID
	event2 := &PaymentEvent{
		EventID:   "evt-789",
		EventType: PaymentEventSuccess,
	}

	if event2.Key() != "evt-789" {
		t.Errorf("Expected key 'evt-789', got '%s'", event2.Key())
	}
}

func TestDefaultBookingConsumerConfig(t *testing.T) {
	cfg := DefaultBookingConsumerConfig()

	if len(cfg.Brokers) == 0 {
		t.Error("Expected at least one broker")
	}

	if cfg.GroupID == "" {
		t.Error("Expected non-empty group ID")
	}

	if cfg.Topic == "" {
		t.Error("Expected non-empty topic")
	}

	if cfg.WorkerCount <= 0 {
		t.Error("Expected positive worker count")
	}
}

func TestPaymentServiceIntegration(t *testing.T) {
	// Create in-memory repository
	repo := repository.NewMemoryPaymentRepository()

	// Create mock gateway with 100% success rate
	gw := gateway.NewMockGateway(&gateway.MockGatewayConfig{
		SuccessRate: 1.0,
		DelayMs:     0,
	})

	// Create payment service
	svc := service.NewPaymentService(repo, gw, nil)

	ctx := context.Background()

	// Create payment
	req := &service.CreatePaymentRequest{
		BookingID: "booking-123",
		UserID:    "user-456",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create payment: %v", err)
	}

	if payment.Status != domain.PaymentStatusPending {
		t.Errorf("Expected status pending, got %s", payment.Status)
	}

	// Process payment
	processedPayment, err := svc.ProcessPayment(ctx, payment.ID)
	if err != nil {
		t.Fatalf("Failed to process payment: %v", err)
	}

	if processedPayment.Status != domain.PaymentStatusCompleted {
		t.Errorf("Expected status completed, got %s", processedPayment.Status)
	}

	if processedPayment.TransactionID == "" {
		t.Error("Expected transaction ID to be set")
	}
}

func TestPaymentServiceFailure(t *testing.T) {
	// Create in-memory repository
	repo := repository.NewMemoryPaymentRepository()

	// Create mock gateway with 0% success rate
	gw := gateway.NewMockGateway(&gateway.MockGatewayConfig{
		SuccessRate:    0.0,
		DelayMs:        0,
		FailureReasons: []string{"card_declined"},
	})

	// Create payment service
	svc := service.NewPaymentService(repo, gw, nil)

	ctx := context.Background()

	// Create payment
	req := &service.CreatePaymentRequest{
		BookingID: "booking-123",
		UserID:    "user-456",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create payment: %v", err)
	}

	// Process payment - should fail
	processedPayment, err := svc.ProcessPayment(ctx, payment.ID)
	if err != nil {
		t.Fatalf("ProcessPayment returned error: %v", err)
	}

	if processedPayment.Status != domain.PaymentStatusFailed {
		t.Errorf("Expected status failed, got %s", processedPayment.Status)
	}

	if processedPayment.FailureReason == "" {
		t.Error("Expected failure reason to be set")
	}
}
