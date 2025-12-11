package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
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
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.JoinQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid request",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	result, err := h.queueService.JoinQueue(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetPosition handles GET /queue/position/:event_id
func (h *QueueHandler) GetPosition(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "event_id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	result, err := h.queueService.GetPosition(c.Request.Context(), userID, eventID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// LeaveQueue handles DELETE /queue/leave
func (h *QueueHandler) LeaveQueue(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.LeaveQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid request",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	result, err := h.queueService.LeaveQueue(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetQueueStatus handles GET /queue/status/:event_id
func (h *QueueHandler) GetQueueStatus(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "event_id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	result, err := h.queueService.GetQueueStatus(c.Request.Context(), eventID)
	if err != nil {
		h.handleError(c, err)
		return
	}

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
