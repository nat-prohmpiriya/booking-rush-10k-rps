package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/middleware"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
)

// EventHandler handles event-related HTTP requests
type EventHandler struct {
	eventService service.EventService
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(eventService service.EventService) *EventHandler {
	return &EventHandler{
		eventService: eventService,
	}
}

// List handles GET /events - lists events with pagination and filters
func (h *EventHandler) List(c *gin.Context) {
	var filter dto.EventListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid query parameters"))
		return
	}

	events, total, err := h.eventService.ListEvents(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list events"))
		return
	}

	eventResponses := make([]*dto.EventResponse, len(events))
	for i, event := range events {
		eventResponses[i] = toEventResponse(event)
	}

	filter.SetDefaults()
	c.JSON(http.StatusOK, response.Paginated(eventResponses, filter.Offset/filter.Limit+1, filter.Limit, int64(total)))
}

// GetBySlug handles GET /events/:slug - retrieves an event by slug
func (h *EventHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Slug is required"))
		return
	}

	event, err := h.eventService.GetEventBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to get event"))
		return
	}

	c.JSON(http.StatusOK, response.Success(toEventResponse(event)))
}

// GetByID handles GET /events/id/:id - retrieves an event by ID
func (h *EventHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	event, err := h.eventService.GetEventByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to get event"))
		return
	}

	c.JSON(http.StatusOK, response.Success(toEventResponse(event)))
}

// Create handles POST /events - creates a new event (Organizer only)
func (h *EventHandler) Create(c *gin.Context) {
	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	// Get tenant ID from JWT context
	tenantID, ok := middleware.GetTenantID(c)
	if !ok || tenantID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("Tenant ID not found in token"))
		return
	}
	req.TenantID = tenantID

	// Validate request
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	event, err := h.eventService.CreateEvent(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrVenueNotFound) {
			c.JSON(http.StatusBadRequest, response.BadRequest("Venue not found"))
			return
		}
		if errors.Is(err, service.ErrEventAlreadyExists) {
			c.JSON(http.StatusConflict, response.Error(response.ErrCodeConflict, "Event with this slug already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to create event"))
		return
	}

	c.JSON(http.StatusCreated, response.Success(toEventResponse(event)))
}

// Update handles PUT /events/:id - updates an event
func (h *EventHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	var req dto.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	// Validate request
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	event, err := h.eventService.UpdateEvent(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to update event"))
		return
	}

	c.JSON(http.StatusOK, response.Success(toEventResponse(event)))
}

// Delete handles DELETE /events/:id - soft deletes an event
func (h *EventHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	err := h.eventService.DeleteEvent(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to delete event"))
		return
	}

	c.JSON(http.StatusOK, response.Success(map[string]string{"message": "Event deleted successfully"}))
}

// Publish handles POST /events/:id/publish - publishes an event
func (h *EventHandler) Publish(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	event, err := h.eventService.PublishEvent(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		if errors.Is(err, service.ErrInvalidEventStatus) {
			c.JSON(http.StatusBadRequest, response.BadRequest("Only draft events can be published"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to publish event"))
		return
	}

	c.JSON(http.StatusOK, response.Success(toEventResponse(event)))
}

// toEventResponse converts a domain event to response DTO
func toEventResponse(event *domain.Event) *dto.EventResponse {
	return &dto.EventResponse{
		ID:          event.ID,
		Name:        event.Name,
		Slug:        event.Slug,
		Description: event.Description,
		VenueID:     event.VenueID,
		StartTime:   event.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		EndTime:     event.EndTime.Format("2006-01-02T15:04:05Z07:00"),
		Status:      event.Status,
		TenantID:    event.TenantID,
		CreatedAt:   event.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   event.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
