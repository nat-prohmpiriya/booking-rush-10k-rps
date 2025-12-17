package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// BookingHandler handles booking HTTP requests
// Uses fast path (Redis Lua + PostgreSQL) for all reservations
// Saga is triggered asynchronously after payment success via webhook
type BookingHandler struct {
	bookingService   service.BookingService
	queueService     service.QueueService
	requireQueuePass bool
}

// BookingHandlerConfig contains configuration for booking handler
type BookingHandlerConfig struct {
	RequireQueuePass bool
}

// NewBookingHandler creates a new booking handler
func NewBookingHandler(bookingService service.BookingService, queueService service.QueueService, cfg *BookingHandlerConfig) *BookingHandler {
	requireQueuePass := false
	if cfg != nil {
		requireQueuePass = cfg.RequireQueuePass
	}
	return &BookingHandler{
		bookingService:   bookingService,
		queueService:     queueService,
		requireQueuePass: requireQueuePass,
	}
}

// ReserveSeats handles POST /bookings/reserve
// FAST PATH: Uses Redis Lua script for atomic reservation + PostgreSQL for persistence
// Returns immediately with booking_id (< 50ms target latency)
// Payment and confirmation are handled asynchronously via Stripe webhook → Saga
func (h *BookingHandler) ReserveSeats(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.reserve")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.ReserveSeatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request")
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

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", req.EventID),
		attribute.String("zone_id", req.ZoneID),
		attribute.String("show_id", req.ShowID),
		attribute.Int("quantity", req.Quantity),
		attribute.Bool("require_queue_pass", h.requireQueuePass),
	)

	// Validate queue pass if required
	if h.requireQueuePass {
		if err := h.queueService.ValidateQueuePass(ctx, userID, req.EventID, req.QueuePass); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			h.handleError(c, err)
			return
		}
		span.SetAttributes(attribute.Bool("queue_pass_valid", true))
	}

	// Fast path: Redis Lua (atomic) + PostgreSQL
	result, err := h.bookingService.ReserveSeats(ctx, userID, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	// Delete queue pass after successful reservation (one-time use)
	if h.requireQueuePass && h.queueService != nil {
		// Run in background - don't block the response
		go func() {
			_ = h.queueService.DeleteQueuePass(ctx, userID, req.EventID)
		}()
	}

	span.SetAttributes(attribute.String("booking_id", result.BookingID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusCreated, result)
}

// ConfirmBooking handles POST /bookings/:id/confirm
func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.confirm")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	bookingID := c.Param("id")
	if bookingID == "" {
		span.SetStatus(codes.Error, "booking id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "booking id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(
		attribute.String("booking_id", bookingID),
		attribute.String("user_id", userID),
	)

	var req dto.ConfirmBookingRequest
	// PaymentID is optional, so we don't fail if body is empty
	_ = c.ShouldBindJSON(&req)

	if req.PaymentID != "" {
		span.SetAttributes(attribute.String("payment_id", req.PaymentID))
	}

	result, err := h.bookingService.ConfirmBooking(ctx, bookingID, userID, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// ReleaseBooking handles DELETE /bookings/:id
func (h *BookingHandler) ReleaseBooking(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.release")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	bookingID := c.Param("id")
	if bookingID == "" {
		span.SetStatus(codes.Error, "booking id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "booking id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(
		attribute.String("booking_id", bookingID),
		attribute.String("user_id", userID),
	)

	result, err := h.bookingService.ReleaseBooking(ctx, bookingID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// CancelBooking handles POST /bookings/:id/cancel
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.cancel")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	bookingID := c.Param("id")
	if bookingID == "" {
		span.SetStatus(codes.Error, "booking id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "booking id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(
		attribute.String("booking_id", bookingID),
		attribute.String("user_id", userID),
	)

	result, err := h.bookingService.CancelBooking(ctx, bookingID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// GetBooking handles GET /bookings/:id
func (h *BookingHandler) GetBooking(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.get")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	bookingID := c.Param("id")
	if bookingID == "" {
		span.SetStatus(codes.Error, "booking id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "booking id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(
		attribute.String("booking_id", bookingID),
		attribute.String("user_id", userID),
	)

	result, err := h.bookingService.GetBooking(ctx, bookingID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// GetUserBookings handles GET /bookings
func (h *BookingHandler) GetUserBookings(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.list")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
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

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	result, err := h.bookingService.GetUserBookings(ctx, userID, page, pageSize)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// GetUserBookingSummary handles GET /bookings/summary
func (h *BookingHandler) GetUserBookingSummary(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.summary")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	eventID := c.Query("event_id")
	if eventID == "" {
		span.SetStatus(codes.Error, "event_id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "event_id required",
			Code:    "INVALID_REQUEST",
			Message: "Please provide event_id query parameter",
		})
		return
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", eventID),
	)

	result, err := h.bookingService.GetUserBookingSummary(ctx, userID, eventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// GetPendingBookings handles GET /bookings/pending
func (h *BookingHandler) GetPendingBookings(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.booking.pending")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	// Parse limit parameter
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	span.SetAttributes(attribute.Int("limit", limit))

	result, err := h.bookingService.GetPendingBookings(ctx, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
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
	// Queue pass errors
	case errors.Is(err, domain.ErrQueuePassRequired):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error:   err.Error(),
			Code:    "QUEUE_PASS_REQUIRED",
			Message: "Please join the queue and wait for your turn to book",
		})
	case errors.Is(err, domain.ErrInvalidQueuePass):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_QUEUE_PASS",
		})
	case errors.Is(err, domain.ErrQueuePassExpired):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error:   err.Error(),
			Code:    "QUEUE_PASS_EXPIRED",
			Message: "Your queue pass has expired. Please rejoin the queue.",
		})
	case errors.Is(err, domain.ErrQueuePassUserMismatch),
		errors.Is(err, domain.ErrQueuePassEventMismatch):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "QUEUE_PASS_MISMATCH",
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}
