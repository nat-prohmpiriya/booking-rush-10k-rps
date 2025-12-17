package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// QueueHandler handles queue HTTP requests
type QueueHandler struct {
	queueService service.QueueService
}

// NewQueueHandler creates a new queue handler
func NewQueueHandler(queueService service.QueueService) *QueueHandler {
	return &QueueHandler{
		queueService: queueService,
	}
}

// JoinQueue handles POST /queue/join
func (h *QueueHandler) JoinQueue(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.join")
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

	var req dto.JoinQueueRequest
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

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", req.EventID),
	)

	result, err := h.queueService.JoinQueue(ctx, userID, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusCreated, result)
}

// GetPosition handles GET /queue/position/:event_id
func (h *QueueHandler) GetPosition(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.position")
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

	eventID := c.Param("event_id")
	if eventID == "" {
		span.SetStatus(codes.Error, "event_id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "event_id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", eventID),
	)

	result, err := h.queueService.GetPosition(ctx, userID, eventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// LeaveQueue handles DELETE /queue/leave
func (h *QueueHandler) LeaveQueue(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.leave")
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

	var req dto.LeaveQueueRequest
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

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", req.EventID),
	)

	result, err := h.queueService.LeaveQueue(ctx, userID, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// GetQueueStatus handles GET /queue/status/:event_id
func (h *QueueHandler) GetQueueStatus(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.status")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	eventID := c.Param("event_id")
	if eventID == "" {
		span.SetStatus(codes.Error, "event_id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "event_id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(attribute.String("event_id", eventID))

	result, err := h.queueService.GetQueueStatus(ctx, eventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// handleError converts domain errors to HTTP responses
func (h *QueueHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotInQueue):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "NOT_IN_QUEUE",
		})
	case errors.Is(err, domain.ErrAlreadyInQueue):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "ALREADY_IN_QUEUE",
		})
	case errors.Is(err, domain.ErrQueueFull):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "QUEUE_FULL",
		})
	case errors.Is(err, domain.ErrQueueNotOpen):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "QUEUE_NOT_OPEN",
		})
	case errors.Is(err, domain.ErrInvalidQueueToken):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_TOKEN",
		})
	case errors.Is(err, domain.ErrInvalidUserID):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "FORBIDDEN",
		})
	case errors.Is(err, domain.ErrInvalidEventID):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_EVENT_ID",
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}
