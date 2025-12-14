package handler

import (
	"errors"
	"net/http"
	"strconv"

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
	showService  service.ShowService
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(eventService service.EventService, showService service.ShowService) *EventHandler {
	return &EventHandler{
		eventService: eventService,
		showService:  showService,
	}
}

// List handles GET /events - lists published events for public
func (h *EventHandler) List(c *gin.Context) {
	// Parse pagination params
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

	events, total, err := h.eventService.ListPublishedEvents(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list events"))
		return
	}

	eventResponses := make([]*dto.EventResponse, len(events))
	for i, event := range events {
		// Fetch shows for this event to calculate sale status
		shows, _, _ := h.showService.ListShowsByEvent(c.Request.Context(), event.ID, nil)
		saleStatus := calculateSaleStatus(shows)
		eventResponses[i] = toEventResponse(event, saleStatus)
	}

	c.JSON(http.StatusOK, response.Paginated(eventResponses, offset/limit+1, limit, int64(total)))
}

// GetBySlug handles GET /events/:slug - retrieves an event by slug
// For non-published events, only the owner can view
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

	// If event is not published, only owner can view
	if event.Status != domain.EventStatusPublished {
		userID, _ := middleware.GetUserID(c)
		if userID != event.OrganizerID {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
	}

	// Fetch shows for this event to calculate sale status
	shows, _, _ := h.showService.ListShowsByEvent(c.Request.Context(), event.ID, nil)
	saleStatus := calculateSaleStatus(shows)

	c.JSON(http.StatusOK, response.Success(toEventResponse(event, saleStatus)))
}

// GetByID handles GET /events/id/:id - retrieves an event by ID
// For non-published events, only the owner can view
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

	// If event is not published, only owner can view
	if event.Status != domain.EventStatusPublished {
		userID, _ := middleware.GetUserID(c)
		if userID != event.OrganizerID {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
	}

	// Fetch shows for this event to calculate sale status
	shows, _, _ := h.showService.ListShowsByEvent(c.Request.Context(), event.ID, nil)
	saleStatus := calculateSaleStatus(shows)

	c.JSON(http.StatusOK, response.Success(toEventResponse(event, saleStatus)))
}

// ListMyEvents handles GET /events/my - lists events owned by current user (Organizer)
func (h *EventHandler) ListMyEvents(c *gin.Context) {
	// Get user ID from JWT context
	userID, ok := middleware.GetUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User ID not found in token"))
		return
	}

	// Parse pagination params
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

	filter := &dto.EventListFilter{
		OrganizerID: userID,
		Status:      c.Query("status"),
		Search:      c.Query("search"),
		Limit:       limit,
		Offset:      offset,
	}

	events, total, err := h.eventService.ListEvents(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list events"))
		return
	}

	eventResponses := make([]*dto.EventResponse, len(events))
	for i, event := range events {
		// Fetch shows for this event to calculate sale status
		shows, _, _ := h.showService.ListShowsByEvent(c.Request.Context(), event.ID, nil)
		saleStatus := calculateSaleStatus(shows)
		eventResponses[i] = toEventResponse(event, saleStatus)
	}

	c.JSON(http.StatusOK, response.Paginated(eventResponses, offset/limit+1, limit, int64(total)))
}

// Create handles POST /events - creates a new event (Organizer only)
func (h *EventHandler) Create(c *gin.Context) {
	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	// Get tenant ID and user ID from JWT context
	tenantID, ok := middleware.GetTenantID(c)
	if !ok || tenantID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("Tenant ID not found in token"))
		return
	}
	req.TenantID = tenantID

	userID, ok := middleware.GetUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User ID not found in token"))
		return
	}
	req.OrganizerID = userID

	// Validate request
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	event, err := h.eventService.CreateEvent(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrEventAlreadyExists) {
			c.JSON(http.StatusConflict, response.Error(response.ErrCodeConflict, "Event with this slug already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to create event"))
		return
	}

	// New event has no shows yet, default to "scheduled"
	c.JSON(http.StatusCreated, response.Success(toEventResponse(event, "scheduled")))
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

	// Fetch shows for this event to calculate sale status
	shows, _, _ := h.showService.ListShowsByEvent(c.Request.Context(), event.ID, nil)
	saleStatus := calculateSaleStatus(shows)

	c.JSON(http.StatusOK, response.Success(toEventResponse(event, saleStatus)))
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

	// Fetch shows for this event to calculate sale status
	shows, _, _ := h.showService.ListShowsByEvent(c.Request.Context(), event.ID, nil)
	saleStatus := calculateSaleStatus(shows)

	c.JSON(http.StatusOK, response.Success(toEventResponse(event, saleStatus)))
}

// calculateSaleStatus determines the aggregated sale status from shows
// Priority: on_sale > scheduled > sold_out > completed > cancelled
func calculateSaleStatus(shows []*domain.Show) string {
	if len(shows) == 0 {
		return "scheduled" // Default if no shows
	}

	hasOnSale := false
	hasScheduled := false
	hasSoldOut := false
	hasCompleted := false
	hasCancelled := false

	for _, show := range shows {
		switch show.Status {
		case domain.ShowStatusOnSale:
			hasOnSale = true
		case domain.ShowStatusScheduled:
			hasScheduled = true
		case domain.ShowStatusSoldOut:
			hasSoldOut = true
		case domain.ShowStatusCompleted:
			hasCompleted = true
		case domain.ShowStatusCancelled:
			hasCancelled = true
		}
	}

	// Return based on priority
	if hasOnSale {
		return domain.ShowStatusOnSale
	}
	if hasScheduled {
		return domain.ShowStatusScheduled
	}
	if hasSoldOut {
		return domain.ShowStatusSoldOut
	}
	if hasCompleted {
		return domain.ShowStatusCompleted
	}
	if hasCancelled {
		return domain.ShowStatusCancelled
	}

	return "scheduled"
}

// toEventResponse converts a domain event to response DTO
func toEventResponse(event *domain.Event, saleStatus string) *dto.EventResponse {
	resp := &dto.EventResponse{
		ID:                event.ID,
		TenantID:          event.TenantID,
		OrganizerID:       event.OrganizerID,
		CategoryID:        event.CategoryID,
		Name:              event.Name,
		Slug:              event.Slug,
		Description:       event.Description,
		ShortDescription:  event.ShortDescription,
		PosterURL:         event.PosterURL,
		BannerURL:         event.BannerURL,
		Gallery:           event.Gallery,
		VenueName:         event.VenueName,
		VenueAddress:      event.VenueAddress,
		City:              event.City,
		Country:           event.Country,
		Latitude:          event.Latitude,
		Longitude:         event.Longitude,
		MaxTicketsPerUser: event.MaxTicketsPerUser,
		Status:            event.Status,
		SaleStatus:        saleStatus,
		IsFeatured:        event.IsFeatured,
		IsPublic:          event.IsPublic,
		MetaTitle:         event.MetaTitle,
		MetaDescription:   event.MetaDescription,
		MinPrice:          event.MinPrice,
		CreatedAt:         event.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         event.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if event.BookingStartAt != nil {
		t := event.BookingStartAt.Format("2006-01-02T15:04:05Z07:00")
		resp.BookingStartAt = &t
	}
	if event.BookingEndAt != nil {
		t := event.BookingEndAt.Format("2006-01-02T15:04:05Z07:00")
		resp.BookingEndAt = &t
	}
	if event.PublishedAt != nil {
		t := event.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.PublishedAt = &t
	}
	if resp.Gallery == nil {
		resp.Gallery = []string{}
	}

	return resp
}
