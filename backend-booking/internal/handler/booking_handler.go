package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

// BookingHandler handles booking HTTP requests
type BookingHandler struct {
	bookingService service.BookingService
	sagaService    service.SagaService
	useSaga        bool
	sagaTimeout    time.Duration
}

// BookingHandlerConfig holds configuration for BookingHandler
type BookingHandlerConfig struct {
	UseSaga     bool          // Enable saga-based booking
	SagaTimeout time.Duration // Timeout for waiting saga completion
}

// NewBookingHandler creates a new booking handler
func NewBookingHandler(bookingService service.BookingService) *BookingHandler {
	return &BookingHandler{
		bookingService: bookingService,
		useSaga:        false,
		sagaTimeout:    30 * time.Second,
	}
}

// NewBookingHandlerWithSaga creates a new booking handler with saga support
func NewBookingHandlerWithSaga(bookingService service.BookingService, sagaService service.SagaService, cfg *BookingHandlerConfig) *BookingHandler {
	if cfg == nil {
		cfg = &BookingHandlerConfig{
			UseSaga:     true,
			SagaTimeout: 30 * time.Second,
		}
	}
	return &BookingHandler{
		bookingService: bookingService,
		sagaService:    sagaService,
		useSaga:        cfg.UseSaga && sagaService != nil,
		sagaTimeout:    cfg.SagaTimeout,
	}
}

// ReserveSeats handles POST /bookings/reserve
func (h *BookingHandler) ReserveSeats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.ReserveSeatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid request",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// Use tenant_id from header if not in request body
	if req.TenantID == "" {
		req.TenantID = c.GetString("tenant_id")
	}

	// Use saga-based booking if enabled
	if h.useSaga {
		h.reserveSeatsViaSaga(c, userID, &req)
		return
	}

	// Fallback to direct booking
	result, err := h.bookingService.ReserveSeats(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// reserveSeatsViaSaga handles booking via saga pattern
func (h *BookingHandler) reserveSeatsViaSaga(c *gin.Context, userID string, req *dto.ReserveSeatsRequest) {
	log := logger.Get()
	ctx := c.Request.Context()

	// Calculate total price
	totalPrice := req.UnitPrice * float64(req.Quantity)
	if totalPrice == 0 {
		totalPrice = float64(req.Quantity) * 100 // Default price if not provided
	}

	// Create saga data
	sagaData := &saga.BookingSagaData{
		UserID:        userID,
		EventID:       req.EventID,
		ZoneID:        req.ZoneID,
		Quantity:      req.Quantity,
		TotalPrice:    totalPrice,
		Currency:      "THB",
		PaymentMethod: "card",
	}

	// Start saga
	sagaID, err := h.sagaService.StartBookingSaga(ctx, sagaData)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to start saga: %v", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "failed to start booking process",
			Code:    "SAGA_START_FAILED",
			Message: err.Error(),
		})
		return
	}

	log.Info(fmt.Sprintf("Saga started: saga_id=%s, user_id=%s", sagaID, userID))

	// Poll for saga completion with timeout
	deadline := time.Now().Add(h.sagaTimeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		instance, err := h.sagaService.GetSagaStatus(ctx, sagaID)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to get saga status: %v", err))
			time.Sleep(pollInterval)
			continue
		}

		switch instance.Status {
		case pkgsaga.StatusCompleted:
			// Saga completed successfully - extract booking info from saga data
			h.handleSagaSuccess(c, instance)
			return

		case pkgsaga.StatusFailed:
			// Saga failed
			h.handleSagaFailure(c, instance)
			return

		case pkgsaga.StatusCompensated:
			// Saga was rolled back
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "booking was rolled back",
				Code:    "BOOKING_ROLLED_BACK",
				Message: instance.Error,
			})
			return

		case pkgsaga.StatusRunning, pkgsaga.StatusPending:
			// Still processing, continue polling
			time.Sleep(pollInterval)
			continue
		}
	}

	// Timeout - return accepted with saga_id for client to poll
	c.JSON(http.StatusAccepted, gin.H{
		"saga_id": sagaID,
		"status":  "processing",
		"message": "Booking is being processed. Use GET /saga/bookings/{saga_id} to check status.",
	})
}

// handleSagaSuccess extracts booking result from completed saga
func (h *BookingHandler) handleSagaSuccess(c *gin.Context, instance *pkgsaga.Instance) {
	// Extract booking data from saga instance
	data := instance.Data
	bookingID, _ := data["booking_id"].(string)
	if bookingID == "" {
		bookingID, _ = data["reservation_id"].(string)
	}
	if bookingID == "" {
		// Try to get from step results array
		for _, result := range instance.StepResults {
			if result.StepName == "reserve-seats" && result.Data != nil {
				if resID, ok := result.Data["reservation_id"].(string); ok {
					bookingID = resID
					break
				}
			}
		}
	}

	// Build response similar to direct booking
	expiresAt := time.Now().Add(10 * time.Minute) // Default TTL
	if expStr, ok := data["expires_at"].(string); ok {
		if exp, err := time.Parse(time.RFC3339, expStr); err == nil {
			expiresAt = exp
		}
	}

	totalPrice, _ := data["total_price"].(float64)
	quantity, _ := data["quantity"].(float64)
	if totalPrice == 0 && quantity > 0 {
		totalPrice = quantity * 100 // Default
	}

	c.JSON(http.StatusCreated, &dto.ReserveSeatsResponse{
		BookingID:  bookingID,
		Status:     "pending", // Saga completed reserve step, waiting for payment
		ExpiresAt:  expiresAt,
		TotalPrice: totalPrice,
	})
}

// handleSagaFailure handles saga failure response
func (h *BookingHandler) handleSagaFailure(c *gin.Context, instance *pkgsaga.Instance) {
	errorCode := "BOOKING_FAILED"
	statusCode := http.StatusInternalServerError

	// Map saga errors to HTTP errors
	if instance.Error != "" {
		switch {
		case contains(instance.Error, "insufficient") || contains(instance.Error, "not enough"):
			errorCode = "INSUFFICIENT_SEATS"
			statusCode = http.StatusConflict
		case contains(instance.Error, "not found"):
			errorCode = "NOT_FOUND"
			statusCode = http.StatusNotFound
		case contains(instance.Error, "max") || contains(instance.Error, "exceeded"):
			errorCode = "MAX_TICKETS_EXCEEDED"
			statusCode = http.StatusConflict
		}
	}

	c.JSON(statusCode, dto.ErrorResponse{
		Error:   instance.Error,
		Code:    errorCode,
		Message: fmt.Sprintf("Saga failed at step: %s", instance.CurrentStep),
	})
}

// contains checks if str contains substr (case-insensitive)
func contains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// ConfirmBooking handles POST /bookings/:id/confirm
func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	bookingID := c.Param("id")
	if bookingID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "booking id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	var req dto.ConfirmBookingRequest
	// PaymentID is optional, so we don't fail if body is empty
	_ = c.ShouldBindJSON(&req)

	result, err := h.bookingService.ConfirmBooking(c.Request.Context(), bookingID, userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ReleaseBooking handles DELETE /bookings/:id
func (h *BookingHandler) ReleaseBooking(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	bookingID := c.Param("id")
	if bookingID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "booking id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	result, err := h.bookingService.ReleaseBooking(c.Request.Context(), bookingID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CancelBooking handles POST /bookings/:id/cancel
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	bookingID := c.Param("id")
	if bookingID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "booking id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	result, err := h.bookingService.CancelBooking(c.Request.Context(), bookingID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetBooking handles GET /bookings/:id
func (h *BookingHandler) GetBooking(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	bookingID := c.Param("id")
	if bookingID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "booking id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	result, err := h.bookingService.GetBooking(c.Request.Context(), bookingID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetUserBookings handles GET /bookings
func (h *BookingHandler) GetUserBookings(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 && n <= 100 {
			pageSize = n
		}
	}

	result, err := h.bookingService.GetUserBookings(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetPendingBookings handles GET /bookings/pending
func (h *BookingHandler) GetPendingBookings(c *gin.Context) {
	// Parse limit parameter
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	result, err := h.bookingService.GetPendingBookings(c.Request.Context(), limit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    result,
	})
}

// handleError converts domain errors to HTTP responses
func (h *BookingHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrBookingNotFound),
		errors.Is(err, domain.ErrReservationNotFound):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "NOT_FOUND",
		})
	case errors.Is(err, domain.ErrZoneNotFound):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   err.Error(),
			Code:    "ZONE_NOT_FOUND",
			Message: "Zone inventory not synced to Redis. Please sync inventory first.",
		})
	case errors.Is(err, domain.ErrInvalidUserID):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "FORBIDDEN",
		})
	case errors.Is(err, domain.ErrInvalidShowID):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_SHOW_ID",
		})
	case errors.Is(err, domain.ErrInsufficientSeats):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INSUFFICIENT_SEATS",
		})
	case errors.Is(err, domain.ErrMaxTicketsExceeded):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "MAX_TICKETS_EXCEEDED",
		})
	case errors.Is(err, domain.ErrAlreadyConfirmed):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "ALREADY_CONFIRMED",
		})
	case errors.Is(err, domain.ErrAlreadyReleased):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "ALREADY_RELEASED",
		})
	case errors.Is(err, domain.ErrBookingExpired),
		errors.Is(err, domain.ErrReservationExpired):
		c.JSON(http.StatusGone, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "EXPIRED",
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}
