package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/service"
)

// mockPaymentService implements service.PaymentService for testing
type mockPaymentService struct {
	payments map[string]*domain.Payment
}

func newMockPaymentService() *mockPaymentService {
	return &mockPaymentService{
		payments: make(map[string]*domain.Payment),
	}
}

func (m *mockPaymentService) CreatePayment(ctx context.Context, req *service.CreatePaymentRequest) (*domain.Payment, error) {
	// Check for duplicate
	for _, p := range m.payments {
		if p.BookingID == req.BookingID {
			return nil, domain.ErrPaymentAlreadyExists
		}
	}

	payment, err := domain.NewPayment(req.BookingID, req.UserID, req.Amount, req.Currency, req.Method)
	if err != nil {
		return nil, err
	}
	payment.Metadata = req.Metadata
	m.payments[payment.ID] = payment
	return payment, nil
}

func (m *mockPaymentService) ProcessPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	payment, ok := m.payments[paymentID]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	if err := payment.MarkProcessing(); err != nil {
		return nil, domain.ErrInvalidPaymentStatus
	}
	if err := payment.Complete("mock-txn-" + paymentID); err != nil {
		return nil, err
	}
	return payment, nil
}

func (m *mockPaymentService) GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	payment, ok := m.payments[paymentID]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	return payment, nil
}

func (m *mockPaymentService) GetPaymentByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error) {
	for _, p := range m.payments {
		if p.BookingID == bookingID {
			return p, nil
		}
	}
	return nil, domain.ErrPaymentNotFound
}

func (m *mockPaymentService) GetUserPayments(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	var result []*domain.Payment
	for _, p := range m.payments {
		if p.UserID == userID {
			result = append(result, p)
		}
	}
	// Apply pagination
	if offset >= len(result) {
		return []*domain.Payment{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *mockPaymentService) RefundPayment(ctx context.Context, paymentID string, reason string) (*domain.Payment, error) {
	payment, ok := m.payments[paymentID]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	if err := payment.Refund(); err != nil {
		return nil, domain.ErrInvalidPaymentStatus
	}
	return payment, nil
}

func (m *mockPaymentService) CancelPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	payment, ok := m.payments[paymentID]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	if err := payment.Cancel(); err != nil {
		return nil, domain.ErrInvalidPaymentStatus
	}
	return payment, nil
}

func setupTestRouter(svc service.PaymentService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewPaymentHandler(svc)
	payments := router.Group("/api/v1/payments")
	{
		payments.POST("", handler.CreatePayment)
		payments.GET("/:id", handler.GetPayment)
		payments.POST("/:id/process", handler.ProcessPayment)
		payments.POST("/:id/refund", handler.RefundPayment)
		payments.POST("/:id/cancel", handler.CancelPayment)
		payments.GET("/booking/:bookingId", handler.GetPaymentByBookingID)
		payments.GET("/user/:userId", handler.GetUserPayments)
	}

	return router
}

func TestPaymentHandler_CreatePayment(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	reqBody := dto.CreatePaymentRequest{
		BookingID: "booking-001",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-001")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response dto.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if !response.Success {
		t.Error("Expected success response")
	}
}

func TestPaymentHandler_CreatePayment_NoUserID(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	reqBody := dto.CreatePaymentRequest{
		BookingID: "booking-002",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestPaymentHandler_CreatePayment_Duplicate(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	reqBody := dto.CreatePaymentRequest{
		BookingID: "booking-dup",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}
	body, _ := json.Marshal(reqBody)

	// First request
	req1, _ := http.NewRequest("POST", "/api/v1/payments", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-User-ID", "user-001")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Second request (duplicate)
	body, _ = json.Marshal(reqBody)
	req2, _ := http.NewRequest("POST", "/api/v1/payments", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-ID", "user-001")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w2.Code)
	}
}

func TestPaymentHandler_CreatePayment_ValidationError(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Missing required fields
	reqBody := map[string]interface{}{
		"amount": 1000.00,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-001")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPaymentHandler_GetPayment(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Create a payment first
	payment, _ := domain.NewPayment("booking-get", "user-001", 500.00, "THB", domain.PaymentMethodDebitCard)
	svc.payments[payment.ID] = payment

	req, _ := http.NewRequest("GET", "/api/v1/payments/"+payment.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response dto.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if !response.Success {
		t.Error("Expected success response")
	}
}

func TestPaymentHandler_GetPayment_NotFound(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	req, _ := http.NewRequest("GET", "/api/v1/payments/non-existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPaymentHandler_GetPaymentByBookingID(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Create a payment first
	payment, _ := domain.NewPayment("booking-by-id", "user-001", 750.00, "THB", domain.PaymentMethodCreditCard)
	svc.payments[payment.ID] = payment

	req, _ := http.NewRequest("GET", "/api/v1/payments/booking/booking-by-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestPaymentHandler_GetUserPayments(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Create multiple payments for user
	for i := 0; i < 3; i++ {
		payment, _ := domain.NewPayment("booking-user-"+string(rune('A'+i)), "user-list", float64(100*(i+1)), "THB", domain.PaymentMethodCreditCard)
		svc.payments[payment.ID] = payment
	}

	req, _ := http.NewRequest("GET", "/api/v1/payments/user/user-list", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response dto.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if !response.Success {
		t.Error("Expected success response")
	}
}

func TestPaymentHandler_ProcessPayment(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Create a pending payment
	payment, _ := domain.NewPayment("booking-process", "user-001", 1000.00, "THB", domain.PaymentMethodCreditCard)
	svc.payments[payment.ID] = payment

	req, _ := http.NewRequest("POST", "/api/v1/payments/"+payment.ID+"/process", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify status changed
	if payment.Status != domain.PaymentStatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", payment.Status)
	}
}

func TestPaymentHandler_ProcessPayment_NotFound(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	req, _ := http.NewRequest("POST", "/api/v1/payments/non-existent/process", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPaymentHandler_RefundPayment(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Create and complete a payment
	payment, _ := domain.NewPayment("booking-refund", "user-001", 2000.00, "THB", domain.PaymentMethodCreditCard)
	payment.MarkProcessing()
	payment.Complete("txn-refund-001")
	svc.payments[payment.ID] = payment

	reqBody := dto.RefundPaymentRequest{
		Reason: "customer request",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/payments/"+payment.ID+"/refund", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify status changed
	if payment.Status != domain.PaymentStatusRefunded {
		t.Errorf("Expected status 'refunded', got '%s'", payment.Status)
	}
}

func TestPaymentHandler_RefundPayment_InvalidStatus(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Create a pending payment (cannot be refunded)
	payment, _ := domain.NewPayment("booking-refund-pending", "user-001", 2000.00, "THB", domain.PaymentMethodCreditCard)
	svc.payments[payment.ID] = payment

	req, _ := http.NewRequest("POST", "/api/v1/payments/"+payment.ID+"/refund", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPaymentHandler_CancelPayment(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Create a pending payment
	payment, _ := domain.NewPayment("booking-cancel", "user-001", 1500.00, "THB", domain.PaymentMethodCreditCard)
	svc.payments[payment.ID] = payment

	req, _ := http.NewRequest("POST", "/api/v1/payments/"+payment.ID+"/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify status changed
	if payment.Status != domain.PaymentStatusCancelled {
		t.Errorf("Expected status 'cancelled', got '%s'", payment.Status)
	}
}

func TestPaymentHandler_CancelPayment_InvalidStatus(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	// Create a completed payment (cannot be cancelled)
	payment, _ := domain.NewPayment("booking-cancel-completed", "user-001", 1500.00, "THB", domain.PaymentMethodCreditCard)
	payment.MarkProcessing()
	payment.Complete("txn-cancel-001")
	svc.payments[payment.ID] = payment

	req, _ := http.NewRequest("POST", "/api/v1/payments/"+payment.ID+"/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPaymentHandler_CreatePayment_WithAutoProcess(t *testing.T) {
	svc := newMockPaymentService()
	router := setupTestRouter(svc)

	reqBody := dto.CreatePaymentRequest{
		BookingID: "booking-auto",
		Amount:    1000.00,
		Currency:  "THB",
		Method:    domain.PaymentMethodCreditCard,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/payments?auto_process=true", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-001")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Verify the payment was processed
	var response dto.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if !response.Success {
		t.Error("Expected success response")
	}

	// Get the payment from response
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}
	status, ok := dataMap["status"].(string)
	if !ok || status != string(domain.PaymentStatusCompleted) {
		t.Errorf("Expected status 'completed', got '%s'", status)
	}
}
