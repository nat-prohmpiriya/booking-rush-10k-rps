package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// ShowHandler handles show-related HTTP requests
type ShowHandler struct {
	showService  service.ShowService
	eventService service.EventService
}

// NewShowHandler creates a new ShowHandler
func NewShowHandler(showService service.ShowService, eventService service.EventService) *ShowHandler {
	return &ShowHandler{
		showService:  showService,
		eventService: eventService,
	}
}

// ListByEvent handles GET /events/slug/:slug/shows - lists shows for an event by slug
func (h *ShowHandler) ListByEvent(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show.ListByEvent")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	slug := c.Param("slug")
	span.SetAttributes(attribute.String("event.slug", slug))

	if slug == "" {
		span.RecordError(errors.New("event slug is required"))
		span.SetStatus(codes.Error, "event slug is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("Event slug is required"))
		return
	}

	// Get event by slug to get event ID
	event, err := h.eventService.GetEventBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrEventNotFound) {
			span.SetStatus(codes.Error, "event not found")
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		span.SetStatus(codes.Error, "failed to get event")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to get event"))
		return
	}

	span.SetAttributes(attribute.String("event.id", event.ID))

	var filter dto.ShowListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid query parameters")
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid query parameters"))
		return
	}

	shows, total, err := h.showService.ListShowsByEvent(ctx, event.ID, &filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to list shows")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list shows: "+err.Error()))
		return
	}

	span.SetAttributes(attribute.Int("shows.count", len(shows)))
	span.SetAttributes(attribute.Int("shows.total", int(total)))

	showResponses := make([]*dto.ShowResponse, len(shows))
	for i, show := range shows {
		showResponses[i] = toShowResponse(show)
	}

	filter.SetDefaults()
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Paginated(showResponses, filter.Offset/filter.Limit+1, filter.Limit, int64(total)))
}

// Create handles POST /events/:id/shows - creates a new show for an event
func (h *ShowHandler) Create(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show.Create")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	eventID := c.Param("id")
	span.SetAttributes(attribute.String("event.id", eventID))

	if eventID == "" {
		span.RecordError(errors.New("event ID is required"))
		span.SetStatus(codes.Error, "event ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("Event ID is required"))
		return
	}

	var req dto.CreateShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	req.EventID = eventID

	// Validate request
	if valid, msg := req.Validate(); !valid {
		span.RecordError(errors.New(msg))
		span.SetStatus(codes.Error, "validation failed")
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	show, err := h.showService.CreateShow(ctx, &req)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrEventNotFound) {
			span.SetStatus(codes.Error, "event not found")
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		// Check for validation errors (invalid format, etc.)
		if isValidationError(err) {
			span.SetStatus(codes.Error, "validation error")
			c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
			return
		}
		span.SetStatus(codes.Error, "failed to create show")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to create show: "+err.Error()))
		return
	}

	span.SetAttributes(attribute.String("show.id", show.ID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusCreated, response.Success(toShowResponse(show)))
}

// GetByID handles GET /shows/:id - retrieves a show by ID
func (h *ShowHandler) GetByID(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show.GetByID")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	span.SetAttributes(attribute.String("show.id", id))

	if id == "" {
		span.RecordError(errors.New("ID is required"))
		span.SetStatus(codes.Error, "ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	show, err := h.showService.GetShowByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrShowNotFound) {
			span.SetStatus(codes.Error, "show not found")
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		span.SetStatus(codes.Error, "failed to get show")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to get show"))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(toShowResponse(show)))
}

// Update handles PUT /shows/:id - updates a show
func (h *ShowHandler) Update(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show.Update")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	span.SetAttributes(attribute.String("show.id", id))

	if id == "" {
		span.RecordError(errors.New("ID is required"))
		span.SetStatus(codes.Error, "ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	var req dto.UpdateShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	// Validate request
	if valid, msg := req.Validate(); !valid {
		span.RecordError(errors.New(msg))
		span.SetStatus(codes.Error, "validation failed")
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	show, err := h.showService.UpdateShow(ctx, id, &req)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrShowNotFound) {
			span.SetStatus(codes.Error, "show not found")
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		// Check for validation errors (invalid format, etc.)
		if isValidationError(err) {
			span.SetStatus(codes.Error, "validation error")
			c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
			return
		}
		span.SetStatus(codes.Error, "failed to update show")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to update show: "+err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(toShowResponse(show)))
}

// Delete handles DELETE /shows/:id - soft deletes a show
func (h *ShowHandler) Delete(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show.Delete")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	span.SetAttributes(attribute.String("show.id", id))

	if id == "" {
		span.RecordError(errors.New("ID is required"))
		span.SetStatus(codes.Error, "ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	err := h.showService.DeleteShow(ctx, id)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrShowNotFound) {
			span.SetStatus(codes.Error, "show not found")
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		span.SetStatus(codes.Error, "failed to delete show")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to delete show"))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(map[string]string{"message": "Show deleted successfully"}))
}

// isValidationError checks if error is a validation error (should return 400 BadRequest)
func isValidationError(err error) bool {
	msg := err.Error()
	return strings.HasPrefix(msg, "invalid ") ||
		strings.Contains(msg, "is required") ||
		strings.Contains(msg, "format")
}

// toShowResponse converts a domain show to response DTO
func toShowResponse(show *domain.Show) *dto.ShowResponse {
	// Return full ISO timestamps by combining show_date with time
	resp := &dto.ShowResponse{
		ID:            show.ID,
		EventID:       show.EventID,
		Name:          show.Name,
		ShowDate:      show.ShowDate.Format("2006-01-02"),
		StartTime:     show.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		EndTime:       show.EndTime.Format("2006-01-02T15:04:05Z07:00"),
		Status:        show.Status,
		TotalCapacity: show.TotalCapacity,
		ReservedCount: show.ReservedCount,
		SoldCount:     show.SoldCount,
		CreatedAt:     show.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     show.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if show.DoorsOpenAt != nil {
		t := show.DoorsOpenAt.Format("2006-01-02T15:04:05Z07:00")
		resp.DoorsOpenAt = &t
	}
	if show.SaleStartAt != nil {
		t := show.SaleStartAt.Format("2006-01-02T15:04:05Z07:00")
		resp.SaleStartAt = &t
	}
	if show.SaleEndAt != nil {
		t := show.SaleEndAt.Format("2006-01-02T15:04:05Z07:00")
		resp.SaleEndAt = &t
	}

	return resp
}
