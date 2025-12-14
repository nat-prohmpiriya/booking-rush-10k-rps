package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/gateway"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/service"
)

// PaymentHandler handles payment HTTP endpoints
type PaymentHandler struct {
	paymentService service.PaymentService
	paymentGateway gateway.PaymentGateway
	authServiceURL string
}

// NewPaymentHandler creates a new PaymentHandler
func NewPaymentHandler(paymentService service.PaymentService, paymentGateway gateway.PaymentGateway, authServiceURL string) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		paymentGateway: paymentGateway,
		authServiceURL: authServiceURL,
	}
}

// CreatePayment handles POST /payments
// Creates a new payment and optionally processes it immediately
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req dto.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	// Get tenant ID from context (set by auth middleware) or header
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = c.GetString("tenant_id")
	}
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("UNAUTHORIZED", "tenant_id is required"))
		return
	}

	// Get user ID from context (set by auth middleware) or header for now
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.GetString("user_id")
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("UNAUTHORIZED", "user_id is required"))
		return
	}

	// Create payment request for service
	svcReq := &service.CreatePaymentRequest{
		TenantID:  tenantID,
		BookingID: req.BookingID,
		UserID:    userID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Method:    req.Method,
		Metadata:  req.Metadata,
	}

	payment, err := h.paymentService.CreatePayment(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentAlreadyExists) {
			c.JSON(http.StatusConflict, dto.NewErrorResponse("PAYMENT_EXISTS", "payment already exists for this booking"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("CREATE_FAILED", err.Error()))
		return
	}

	// Check if auto-process is requested
	autoProcess := c.Query("auto_process") == "true"
	if autoProcess {
		payment, err = h.paymentService.ProcessPayment(c.Request.Context(), payment.ID)
		if err != nil {
			// Payment created but processing failed - still return the payment with its current status
			c.JSON(http.StatusAccepted, dto.NewSuccessResponse(dto.FromPayment(payment)))
			return
		}
	}

	c.JSON(http.StatusCreated, dto.NewSuccessResponse(dto.FromPayment(payment)))
}

// GetPayment handles GET /payments/:id
// Returns payment details by ID
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", "payment_id is required"))
		return
	}

	payment, err := h.paymentService.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, dto.NewErrorResponse("NOT_FOUND", "payment not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("GET_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(dto.FromPayment(payment)))
}

// GetPaymentByBookingID handles GET /payments/booking/:bookingId
// Returns payment details by booking ID
func (h *PaymentHandler) GetPaymentByBookingID(c *gin.Context) {
	bookingID := c.Param("bookingId")
	if bookingID == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", "booking_id is required"))
		return
	}

	payment, err := h.paymentService.GetPaymentByBookingID(c.Request.Context(), bookingID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, dto.NewErrorResponse("NOT_FOUND", "payment not found for this booking"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("GET_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(dto.FromPayment(payment)))
}

// GetUserPayments handles GET /payments/user/:userId
// Returns all payments for a user with pagination
func (h *PaymentHandler) GetUserPayments(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		// Try to get from auth context
		userID = c.GetString("user_id")
	}
	if userID == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", "user_id is required"))
		return
	}

	// Parse pagination
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	payments, err := h.paymentService.GetUserPayments(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("GET_FAILED", err.Error()))
		return
	}

	// Convert to response
	paymentResponses := make([]*dto.PaymentResponse, len(payments))
	for i, p := range payments {
		paymentResponses[i] = dto.FromPayment(p)
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.PaymentListResponse{
		Payments: paymentResponses,
		Total:    len(paymentResponses),
	}))
}

// ProcessPayment handles POST /payments/:id/process
// Processes a pending payment
func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", "payment_id is required"))
		return
	}

	payment, err := h.paymentService.ProcessPayment(c.Request.Context(), paymentID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, dto.NewErrorResponse("NOT_FOUND", "payment not found"))
			return
		}
		if errors.Is(err, domain.ErrInvalidPaymentStatus) {
			c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_STATUS", "payment cannot be processed in current status"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("PROCESS_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(dto.FromPayment(payment)))
}

// RefundPayment handles POST /payments/:id/refund
// Refunds a completed payment
func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", "payment_id is required"))
		return
	}

	var req dto.RefundPaymentRequest
	// Request body is optional for full refund
	_ = c.ShouldBindJSON(&req)

	reason := req.Reason
	if reason == "" {
		reason = "customer_request"
	}

	payment, err := h.paymentService.RefundPayment(c.Request.Context(), paymentID, reason)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, dto.NewErrorResponse("NOT_FOUND", "payment not found"))
			return
		}
		if errors.Is(err, domain.ErrInvalidPaymentStatus) {
			c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_STATUS", "payment cannot be refunded in current status"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("REFUND_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(dto.FromPayment(payment)))
}

// CancelPayment handles POST /payments/:id/cancel
// Cancels a pending payment
func (h *PaymentHandler) CancelPayment(c *gin.Context) {
	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", "payment_id is required"))
		return
	}

	payment, err := h.paymentService.CancelPayment(c.Request.Context(), paymentID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, dto.NewErrorResponse("NOT_FOUND", "payment not found"))
			return
		}
		if errors.Is(err, domain.ErrInvalidPaymentStatus) {
			c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_STATUS", "payment cannot be cancelled in current status"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("CANCEL_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(dto.FromPayment(payment)))
}

// CreatePaymentIntent handles POST /payments/intent
// Creates a Stripe PaymentIntent and returns client_secret for frontend
func (h *PaymentHandler) CreatePaymentIntent(c *gin.Context) {
	var req dto.CreatePaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	// Get tenant ID from context
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = c.GetString("tenant_id")
	}
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("UNAUTHORIZED", "tenant_id is required"))
		return
	}

	// Get user ID from context
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.GetString("user_id")
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("UNAUTHORIZED", "user_id is required"))
		return
	}

	// Set default currency
	currency := req.Currency
	if currency == "" {
		currency = "THB"
	}

	// Create payment record first
	svcReq := &service.CreatePaymentRequest{
		TenantID:  tenantID,
		BookingID: req.BookingID,
		UserID:    userID,
		Amount:    req.Amount,
		Currency:  currency,
		Method:    domain.PaymentMethodCreditCard,
	}

	payment, err := h.paymentService.CreatePayment(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentAlreadyExists) {
			// If payment already exists, get it and create new intent
			payment, err = h.paymentService.GetPaymentByBookingID(c.Request.Context(), req.BookingID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("GET_PAYMENT_FAILED", err.Error()))
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("CREATE_FAILED", err.Error()))
			return
		}
	}

	// Create PaymentIntent via gateway
	intentReq := &gateway.PaymentIntentRequest{
		PaymentID:   payment.ID,
		Amount:      req.Amount,
		Currency:    currency,
		Description: "Booking payment for " + req.BookingID,
		Metadata: map[string]string{
			"booking_id": req.BookingID,
			"user_id":    userID,
			"payment_id": payment.ID,
		},
	}

	intentResp, err := h.paymentGateway.CreatePaymentIntent(c.Request.Context(), intentReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("PAYMENT_INTENT_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.PaymentIntentResponse{
		PaymentID:       payment.ID,
		ClientSecret:    intentResp.ClientSecret,
		PaymentIntentID: intentResp.PaymentIntentID,
		Amount:          req.Amount,
		Currency:        currency,
		Status:          intentResp.Status,
	}))
}

// ConfirmPaymentIntent handles POST /payments/intent/confirm
// Confirms payment after Stripe client-side completion
func (h *PaymentHandler) ConfirmPaymentIntent(c *gin.Context) {
	var req dto.ConfirmPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	// Verify PaymentIntent status with Stripe
	intentResp, err := h.paymentGateway.ConfirmPaymentIntent(c.Request.Context(), req.PaymentIntentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("CONFIRM_FAILED", err.Error()))
		return
	}

	// Get the payment
	payment, err := h.paymentService.GetPayment(c.Request.Context(), req.PaymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.NewErrorResponse("NOT_FOUND", "payment not found"))
		return
	}

	// If Stripe says succeeded, process our payment
	if intentResp.Status == "succeeded" {
		processedPayment, err := h.paymentService.ProcessPayment(c.Request.Context(), req.PaymentID)
		if err != nil {
			// ProcessPayment failed, return current payment status
			c.JSON(http.StatusOK, dto.NewSuccessResponse(map[string]interface{}{
				"payment_id":        req.PaymentID,
				"status":            payment.Status,
				"payment_intent_id": req.PaymentIntentID,
				"stripe_status":     intentResp.Status,
				"error":             err.Error(),
			}))
			return
		}
		payment = processedPayment
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(map[string]interface{}{
		"payment_id":        payment.ID,
		"status":            payment.Status,
		"payment_intent_id": req.PaymentIntentID,
		"stripe_status":     intentResp.Status,
	}))
}

// CreatePortalSession handles POST /payments/portal
// Creates a Stripe Customer Portal session for managing payment methods
func (h *PaymentHandler) CreatePortalSession(c *gin.Context) {
	var req dto.CreatePortalSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	// Get user info from headers (set by API Gateway after JWT validation)
	userID := c.GetHeader("X-User-ID")
	userEmail := c.GetHeader("X-User-Email")
	if userID == "" {
		userID = c.GetString("user_id")
	}
	if userEmail == "" {
		userEmail = c.GetString("email")
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("UNAUTHORIZED", "user_id is required"))
		return
	}

	// Get Stripe Customer ID from Auth Service
	stripeCustomerID, err := h.getStripeCustomerID(h.authServiceURL, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("AUTH_SERVICE_ERROR", err.Error()))
		return
	}

	// If user doesn't have a Stripe Customer ID, create one
	if stripeCustomerID == "" {
		customerResp, err := h.paymentGateway.CreateCustomer(c.Request.Context(), &gateway.CreateCustomerRequest{
			UserID: userID,
			Email:  userEmail,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("CREATE_CUSTOMER_FAILED", err.Error()))
			return
		}
		stripeCustomerID = customerResp.CustomerID

		// Save the Stripe Customer ID to Auth Service
		if err := h.updateStripeCustomerID(h.authServiceURL, userID, stripeCustomerID); err != nil {
			// Log the error but continue - portal will still work
			fmt.Printf("Failed to save Stripe Customer ID: %v\n", err)
		}
	}

	// Create Portal Session
	portalResp, err := h.paymentGateway.CreatePortalSession(c.Request.Context(), &gateway.PortalSessionRequest{
		CustomerID: stripeCustomerID,
		ReturnURL:  req.ReturnURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("CREATE_PORTAL_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.PortalSessionResponse{
		URL: portalResp.URL,
	}))
}

// getStripeCustomerID fetches Stripe Customer ID from Auth Service
func (h *PaymentHandler) getStripeCustomerID(authServiceURL, userID string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/auth/users/%s/stripe-customer", authServiceURL, userID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to call auth service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth service error: %s", string(body))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			StripeCustomerID string `json:"stripe_customer_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.StripeCustomerID, nil
}

// updateStripeCustomerID saves Stripe Customer ID to Auth Service
func (h *PaymentHandler) updateStripeCustomerID(authServiceURL, userID, stripeCustomerID string) error {
	url := fmt.Sprintf("%s/api/v1/auth/users/%s/stripe-customer", authServiceURL, userID)

	body, err := json.Marshal(map[string]string{
		"stripe_customer_id": stripeCustomerID,
	})
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call auth service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth service error: %s", string(respBody))
	}

	return nil
}

// ListPaymentMethods handles GET /payments/methods
// Returns saved payment methods for the current user
func (h *PaymentHandler) ListPaymentMethods(c *gin.Context) {
	// Get user ID from headers
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.GetString("user_id")
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("UNAUTHORIZED", "user_id is required"))
		return
	}

	// Get Stripe Customer ID from Auth Service
	stripeCustomerID, err := h.getStripeCustomerID(h.authServiceURL, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("AUTH_SERVICE_ERROR", err.Error()))
		return
	}

	// If user doesn't have a Stripe Customer ID, return empty list
	if stripeCustomerID == "" {
		c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.PaymentMethodsListResponse{
			PaymentMethods: []*dto.PaymentMethodResponse{},
			Total:          0,
		}))
		return
	}

	// Get payment methods from Stripe
	paymentMethods, err := h.paymentGateway.ListPaymentMethods(c.Request.Context(), stripeCustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("LIST_METHODS_FAILED", err.Error()))
		return
	}

	// Convert to response
	methodResponses := make([]*dto.PaymentMethodResponse, len(paymentMethods))
	for i, pm := range paymentMethods {
		methodResponses[i] = &dto.PaymentMethodResponse{
			ID:        pm.ID,
			Type:      pm.Type,
			Brand:     pm.Brand,
			Last4:     pm.Last4,
			ExpMonth:  pm.ExpMonth,
			ExpYear:   pm.ExpYear,
			IsDefault: pm.IsDefault,
		}
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.PaymentMethodsListResponse{
		PaymentMethods: methodResponses,
		Total:          len(methodResponses),
	}))
}
