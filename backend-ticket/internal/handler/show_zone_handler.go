package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
)

// ShowZoneHandler handles show zone-related HTTP requests
type ShowZoneHandler struct {
	showZoneService service.ShowZoneService
	showService     service.ShowService
}

// NewShowZoneHandler creates a new ShowZoneHandler
func NewShowZoneHandler(showZoneService service.ShowZoneService, showService service.ShowService) *ShowZoneHandler {
	return &ShowZoneHandler{
		showZoneService: showZoneService,
		showService:     showService,
	}
}

// ListByShow handles GET /shows/:id/zones - lists zones for a show
func (h *ShowZoneHandler) ListByShow(c *gin.Context) {
	showID := c.Param("id")
	if showID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Show ID is required"))
		return
	}

	var filter dto.ShowZoneListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid query parameters"))
		return
	}

	zones, total, err := h.showZoneService.ListZonesByShow(c.Request.Context(), showID, &filter)
	if err != nil {
		if errors.Is(err, service.ErrShowNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list zones"))
		return
	}

	zoneResponses := make([]*dto.ShowZoneResponse, len(zones))
	for i, zone := range zones {
		zoneResponses[i] = toShowZoneResponse(zone)
	}

	filter.SetDefaults()
	c.JSON(http.StatusOK, response.Paginated(zoneResponses, filter.Offset/filter.Limit+1, filter.Limit, int64(total)))
}

// Create handles POST /shows/:id/zones - creates a new zone for a show
func (h *ShowZoneHandler) Create(c *gin.Context) {
	showID := c.Param("id")
	if showID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("Show ID is required"))
		return
	}

	var req dto.CreateShowZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	req.ShowID = showID

	// Validate request
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	zone, err := h.showZoneService.CreateShowZone(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrShowNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to create zone"))
		return
	}

	c.JSON(http.StatusCreated, response.Success(toShowZoneResponse(zone)))
}

// GetByID handles GET /zones/:id - retrieves a zone by ID
func (h *ShowZoneHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	zone, err := h.showZoneService.GetShowZoneByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrShowZoneNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Zone not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to get zone"))
		return
	}

	c.JSON(http.StatusOK, response.Success(toShowZoneResponse(zone)))
}

// Update handles PUT /zones/:id - updates a zone
func (h *ShowZoneHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	var req dto.UpdateShowZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	// Validate request
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	zone, err := h.showZoneService.UpdateShowZone(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrShowZoneNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Zone not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to update zone"))
		return
	}

	c.JSON(http.StatusOK, response.Success(toShowZoneResponse(zone)))
}

// Delete handles DELETE /zones/:id - soft deletes a zone
func (h *ShowZoneHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	err := h.showZoneService.DeleteShowZone(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrShowZoneNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("Zone not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to delete zone"))
		return
	}

	c.JSON(http.StatusOK, response.Success(map[string]string{"message": "Zone deleted successfully"}))
}

// ListActive handles GET /zones/active - lists all active zones for inventory sync
func (h *ShowZoneHandler) ListActive(c *gin.Context) {
	zones, err := h.showZoneService.ListActiveZones(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list active zones"))
		return
	}

	zoneResponses := make([]*dto.ShowZoneResponse, len(zones))
	for i, zone := range zones {
		zoneResponses[i] = toShowZoneResponse(zone)
	}

	c.JSON(http.StatusOK, response.Success(zoneResponses))
}

// toShowZoneResponse converts a domain show zone to response DTO
func toShowZoneResponse(zone *domain.ShowZone) *dto.ShowZoneResponse {
	resp := &dto.ShowZoneResponse{
		ID:             zone.ID,
		ShowID:         zone.ShowID,
		Name:           zone.Name,
		Description:    zone.Description,
		Color:          zone.Color,
		Price:          zone.Price,
		Currency:       zone.Currency,
		TotalSeats:     zone.TotalSeats,
		AvailableSeats: zone.AvailableSeats,
		ReservedSeats:  zone.ReservedSeats,
		SoldSeats:      zone.SoldSeats,
		MinPerOrder:    zone.MinPerOrder,
		MaxPerOrder:    zone.MaxPerOrder,
		IsActive:       zone.IsActive,
		SortOrder:      zone.SortOrder,
		CreatedAt:      zone.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      zone.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if zone.SaleStartAt != nil {
		t := zone.SaleStartAt.Format("2006-01-02T15:04:05Z07:00")
		resp.SaleStartAt = &t
	}
	if zone.SaleEndAt != nil {
		t := zone.SaleEndAt.Format("2006-01-02T15:04:05Z07:00")
		resp.SaleEndAt = &t
	}

	return resp
}
