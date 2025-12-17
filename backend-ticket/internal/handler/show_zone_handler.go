package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show_zone.ListByShow")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	showID := c.Param("id")
	span.SetAttributes(attribute.String("show_id", showID))

	if showID == "" {
		span.RecordError(errors.New("show ID is required"))
		span.SetStatus(codes.Error, "Show ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("Show ID is required"))
		return
	}

	var filter dto.ShowZoneListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid query parameters")
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid query parameters"))
		return
	}

	zones, total, err := h.showZoneService.ListZonesByShow(ctx, showID, &filter)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrShowNotFound) {
			span.SetStatus(codes.Error, "Show not found")
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		span.SetStatus(codes.Error, "Failed to list zones")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list zones"))
		return
	}

	zoneResponses := make([]*dto.ShowZoneResponse, len(zones))
	for i, zone := range zones {
		zoneResponses[i] = toShowZoneResponse(zone)
	}

	filter.SetDefaults()
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Paginated(zoneResponses, filter.Offset/filter.Limit+1, filter.Limit, int64(total)))
}

// Create handles POST /shows/:id/zones - creates a new zone for a show
func (h *ShowZoneHandler) Create(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show_zone.Create")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	showID := c.Param("id")
	span.SetAttributes(attribute.String("show_id", showID))

	if showID == "" {
		span.RecordError(errors.New("show ID is required"))
		span.SetStatus(codes.Error, "Show ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("Show ID is required"))
		return
	}

	var req dto.CreateShowZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	req.ShowID = showID

	// Validate request
	if valid, msg := req.Validate(); !valid {
		span.RecordError(errors.New(msg))
		span.SetStatus(codes.Error, msg)
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	zone, err := h.showZoneService.CreateShowZone(ctx, &req)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrShowNotFound) {
			span.SetStatus(codes.Error, "Show not found")
			c.JSON(http.StatusNotFound, response.NotFound("Show not found"))
			return
		}
		span.SetStatus(codes.Error, "Failed to create zone")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to create zone"))
		return
	}

	span.SetAttributes(attribute.String("zone_id", zone.ID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusCreated, response.Success(toShowZoneResponse(zone)))
}

// GetByID handles GET /zones/:id - retrieves a zone by ID
func (h *ShowZoneHandler) GetByID(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show_zone.GetByID")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	span.SetAttributes(attribute.String("zone_id", id))

	if id == "" {
		span.RecordError(errors.New("ID is required"))
		span.SetStatus(codes.Error, "ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	zone, err := h.showZoneService.GetShowZoneByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrShowZoneNotFound) {
			span.SetStatus(codes.Error, "Zone not found")
			c.JSON(http.StatusNotFound, response.NotFound("Zone not found"))
			return
		}
		span.SetStatus(codes.Error, "Failed to get zone")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to get zone"))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(toShowZoneResponse(zone)))
}

// Update handles PUT /zones/:id - updates a zone
func (h *ShowZoneHandler) Update(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show_zone.Update")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	span.SetAttributes(attribute.String("zone_id", id))

	if id == "" {
		span.RecordError(errors.New("ID is required"))
		span.SetStatus(codes.Error, "ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	var req dto.UpdateShowZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest("Invalid request body"))
		return
	}

	// Validate request
	if valid, msg := req.Validate(); !valid {
		span.RecordError(errors.New(msg))
		span.SetStatus(codes.Error, msg)
		c.JSON(http.StatusBadRequest, response.BadRequest(msg))
		return
	}

	zone, err := h.showZoneService.UpdateShowZone(ctx, id, &req)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrShowZoneNotFound) {
			span.SetStatus(codes.Error, "Zone not found")
			c.JSON(http.StatusNotFound, response.NotFound("Zone not found"))
			return
		}
		span.SetStatus(codes.Error, "Failed to update zone")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to update zone"))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(toShowZoneResponse(zone)))
}

// Delete handles DELETE /zones/:id - soft deletes a zone
func (h *ShowZoneHandler) Delete(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show_zone.Delete")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	span.SetAttributes(attribute.String("zone_id", id))

	if id == "" {
		span.RecordError(errors.New("ID is required"))
		span.SetStatus(codes.Error, "ID is required")
		c.JSON(http.StatusBadRequest, response.BadRequest("ID is required"))
		return
	}

	err := h.showZoneService.DeleteShowZone(ctx, id)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrShowZoneNotFound) {
			span.SetStatus(codes.Error, "Zone not found")
			c.JSON(http.StatusNotFound, response.NotFound("Zone not found"))
			return
		}
		span.SetStatus(codes.Error, "Failed to delete zone")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to delete zone"))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(map[string]string{"message": "Zone deleted successfully"}))
}

// ListActive handles GET /zones/active - lists all active zones for inventory sync
func (h *ShowZoneHandler) ListActive(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.show_zone.ListActive")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	zones, err := h.showZoneService.ListActiveZones(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list active zones")
		c.JSON(http.StatusInternalServerError, response.InternalError("Failed to list active zones"))
		return
	}

	zoneResponses := make([]*dto.ShowZoneResponse, len(zones))
	for i, zone := range zones {
		zoneResponses[i] = toShowZoneResponse(zone)
	}

	span.SetStatus(codes.Ok, "")
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
