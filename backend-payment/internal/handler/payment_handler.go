package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment-service/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment-service/internal/service"
)

// PaymentHandler handles payment HTTP endpoints
type PaymentHandler struct {
	paymentService service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler
func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
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
