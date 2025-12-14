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

// ListByEvent handles GET /events/:slug/shows - lists shows for an event by slug
func (h *ShowHandler) ListByEvent(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Event slug is required"))
		return
	}

	// Get event by slug to get event ID
	event, err := h.eventService.GetEventBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to get event"))
		return
	}

	var filter dto.ShowListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid query parameters"))
		return
	}

	shows, total, err := h.showService.ListShowsByEvent(c.Request.Context(), event.ID, &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list shows: "+err.Error()))
		return
	}

	showResponses := make([]*dto.ShowResponse, len(shows))
	for i, show := range shows {
		showResponses[i] = toShowResponse(show)
	}

	filter.SetDefaults()
	c.JSON(http.StatusOK, response.Paginated(showResponses, filter.Offset/filter.Limit+1, filter.Limit, int64(total)))
}

// Create handles POST /events/:id/shows - creates a new show for an event
func (h *ShowHandler) Create(c *gin.Context) {
	eventID := c.Param("id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Event ID is required"))
		return
	}

	var req dto.CreateShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	req.EventID = eventID

	// Validate request
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	show, err := h.showService.CreateShow(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Event not found"))
			return
		}
		// Check for validation errors (invalid format, etc.)
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to create show: "+err.Error()))
		return
	}

	c.JSON(http.StatusCreated, response.Success(toShowResponse(show)))
}

// GetByID handles GET /shows/:id - retrieves a show by ID
func (h *ShowHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	show, err := h.showService.GetShowByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrShowNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to get show"))
		return
	}

	c.JSON(http.StatusOK, response.Success(toShowResponse(show)))
}

// Update handles PUT /shows/:id - updates a show
func (h *ShowHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	var req dto.UpdateShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	// Validate request
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	show, err := h.showService.UpdateShow(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrShowNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		// Check for validation errors (invalid format, etc.)
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to update show: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(toShowResponse(show)))
}

// Delete handles DELETE /shows/:id - soft deletes a show
func (h *ShowHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	err := h.showService.DeleteShow(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrShowNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to delete show"))
		return
	}

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
