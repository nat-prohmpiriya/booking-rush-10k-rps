package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
)

// BookingHandler handles booking HTTP requests
// Uses fast path (Redis Lua + PostgreSQL) for all reservations
// Saga is triggered asynchronously after payment success via webhook
type BookingHandler struct {
	bookingService service.BookingService
}

// NewBookingHandler creates a new booking handler
func NewBookingHandler(bookingService service.BookingService) *BookingHandler {
	return &BookingHandler{
		bookingService: bookingService,
	}
}

// ReserveSeats handles POST /bookings/reserve
// FAST PATH: Uses Redis Lua script for atomic reservation + PostgreSQL for persistence
// Returns immediately with booking_id (< 50ms target latency)
// Payment and confirmation are handled asynchronously via Stripe webhook → Saga
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

	// Fast path: Redis Lua (atomic) + PostgreSQL
	result, err := h.bookingService.ReserveSeats(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
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

// GetUserBookingSummary handles GET /bookings/summary
func (h *BookingHandler) GetUserBookingSummary(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	eventID := c.Query("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "event_id required",
			Code:    "INVALID_REQUEST",
			Message: "Please provide event_id query parameter",
		})
		return
	}

	result, err := h.bookingService.GetUserBookingSummary(c.Request.Context(), userID, eventID)
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
