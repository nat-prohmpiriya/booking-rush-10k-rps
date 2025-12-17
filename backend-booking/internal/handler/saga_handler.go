package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// SagaHandler handles saga-based booking HTTP requests
type SagaHandler struct {
	sagaService service.SagaService
}

// NewSagaHandler creates a new saga handler
func NewSagaHandler(sagaService service.SagaService) *SagaHandler {
	return &SagaHandler{
		sagaService: sagaService,
	}
}

// SagaBookingRequest represents a saga-based booking request
type SagaBookingRequest struct {
	EventID       string  `json:"event_id" binding:"required"`
	ZoneID        string  `json:"zone_id" binding:"required"`
	ShowID        string  `json:"show_id"`
	Quantity      int     `json:"quantity" binding:"required,min=1,max=10"`
	TotalPrice    float64 `json:"total_price" binding:"required"`
	Currency      string  `json:"currency"`
	PaymentMethod string  `json:"payment_method"`
}

// SagaBookingResponse represents a saga booking initiation response
type SagaBookingResponse struct {
	SagaID    string `json:"saga_id"`
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// StartBookingSaga handles POST /saga/bookings
// This initiates an async booking process via saga pattern
func (h *SagaHandler) StartBookingSaga(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.saga.start")
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

	var req SagaBookingRequest
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

	// Set defaults
	if req.Currency == "" {
		req.Currency = "THB"
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "card"
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", req.EventID),
		attribute.String("zone_id", req.ZoneID),
		attribute.String("show_id", req.ShowID),
		attribute.Int("quantity", req.Quantity),
		attribute.Float64("total_price", req.TotalPrice),
		attribute.String("currency", req.Currency),
		attribute.String("payment_method", req.PaymentMethod),
	)

	// Create saga data
	sagaData := &saga.BookingSagaData{
		BookingID:     "", // Will be generated
		UserID:        userID,
		EventID:       req.EventID,
		ShowID:        req.ShowID,
		ZoneID:        req.ZoneID,
		Quantity:      req.Quantity,
		TotalPrice:    req.TotalPrice,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
	}

	// Start saga
	sagaID, err := h.sagaService.StartBookingSaga(ctx, sagaData)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "failed to start booking saga",
			Code:    "SAGA_START_FAILED",
			Message: err.Error(),
		})
		return
	}

	span.SetAttributes(attribute.String("saga_id", sagaID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusAccepted, SagaBookingResponse{
		SagaID:  sagaID,
		Status:  "pending",
		Message: "Booking saga started. Check status for updates.",
	})
}

// GetSagaStatus handles GET /saga/bookings/:saga_id
func (h *SagaHandler) GetSagaStatus(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.saga.status")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	sagaID := c.Param("saga_id")
	if sagaID == "" {
		span.SetStatus(codes.Error, "saga_id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "saga_id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(attribute.String("saga_id", sagaID))

	instance, err := h.sagaService.GetSagaStatus(ctx, sagaID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "saga not found",
			Code:    "NOT_FOUND",
			Message: err.Error(),
		})
		return
	}

	span.SetAttributes(
		attribute.String("status", string(instance.Status)),
		attribute.Int("current_step", instance.CurrentStep),
	)
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, gin.H{
		"saga_id":      instance.ID,
		"status":       instance.Status,
		"current_step": instance.CurrentStep,
		"data":         instance.Data,
		"error":        instance.Error,
		"created_at":   instance.CreatedAt,
		"updated_at":   instance.UpdatedAt,
		"completed_at": instance.CompletedAt,
	})
}
