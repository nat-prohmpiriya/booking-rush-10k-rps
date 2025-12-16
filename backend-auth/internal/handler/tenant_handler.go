package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// TenantHandler handles tenant management HTTP requests
type TenantHandler struct {
	tenantService service.TenantService
}

// NewTenantHandler creates a new TenantHandler
func NewTenantHandler(tenantService service.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

// Create handles tenant creation
// POST /api/v1/tenants
func (h *TenantHandler) Create(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.tenant.create")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	span.SetAttributes(
		attribute.String("tenant_name", req.Name),
		attribute.String("tenant_slug", req.Slug),
	)

	// Validate slug format
	if valid, msg := req.ValidateSlug(); !valid {
		span.SetStatus(codes.Error, "invalid slug")
		c.JSON(http.StatusBadRequest, response.Error("INVALID_SLUG", msg))
		return
	}

	result, err := h.tenantService.Create(ctx, &req)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrTenantAlreadyExists) {
			span.SetStatus(codes.Error, "tenant exists")
			c.JSON(http.StatusConflict, response.Error("TENANT_EXISTS", "Tenant with this slug already exists"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetAttributes(attribute.String("tenant_id", result.ID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusCreated, response.Success(result))
}

// GetByID handles retrieving a tenant by ID
// GET /api/v1/tenants/:id
func (h *TenantHandler) GetByID(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.tenant.get_by_id")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	if id == "" {
		span.SetStatus(codes.Error, "tenant_id required")
		c.JSON(http.StatusBadRequest, response.BadRequest("Tenant ID is required"))
		return
	}

	span.SetAttributes(attribute.String("tenant_id", id))

	result, err := h.tenantService.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrTenantNotFound) {
			span.SetStatus(codes.Error, "tenant not found")
			c.JSON(http.StatusNotFound, response.NotFound("Tenant not found"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(result))
}

// GetBySlug handles retrieving a tenant by slug
// GET /api/v1/tenants/slug/:slug
func (h *TenantHandler) GetBySlug(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.tenant.get_by_slug")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	slug := c.Param("slug")
	if slug == "" {
		span.SetStatus(codes.Error, "slug required")
		c.JSON(http.StatusBadRequest, response.BadRequest("Slug is required"))
		return
	}

	span.SetAttributes(attribute.String("tenant_slug", slug))

	result, err := h.tenantService.GetBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrTenantNotFound) {
			span.SetStatus(codes.Error, "tenant not found")
			c.JSON(http.StatusNotFound, response.NotFound("Tenant not found"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetAttributes(attribute.String("tenant_id", result.ID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(result))
}

// List handles retrieving all tenants with pagination
// GET /api/v1/tenants
func (h *TenantHandler) List(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.tenant.list")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	var query dto.ListTenantsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid query params")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	span.SetAttributes(
		attribute.Int("page", query.Page),
		attribute.Int("limit", query.Limit),
	)

	result, err := h.tenantService.List(ctx, &query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetAttributes(attribute.Int("total_count", result.TotalCount))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(result))
}

// Update handles tenant update
// PUT /api/v1/tenants/:id
func (h *TenantHandler) Update(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.tenant.update")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	if id == "" {
		span.SetStatus(codes.Error, "tenant_id required")
		c.JSON(http.StatusBadRequest, response.BadRequest("Tenant ID is required"))
		return
	}

	span.SetAttributes(attribute.String("tenant_id", id))

	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	// Validate that at least one field is provided
	if valid, msg := req.Validate(); !valid {
		span.SetStatus(codes.Error, "validation error")
		c.JSON(http.StatusBadRequest, response.Error("INVALID_UPDATE", msg))
		return
	}

	result, err := h.tenantService.Update(ctx, id, &req)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrTenantNotFound) {
			span.SetStatus(codes.Error, "tenant not found")
			c.JSON(http.StatusNotFound, response.NotFound("Tenant not found"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(result))
}

// Delete handles tenant soft deletion
// DELETE /api/v1/tenants/:id
func (h *TenantHandler) Delete(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.tenant.delete")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	id := c.Param("id")
	if id == "" {
		span.SetStatus(codes.Error, "tenant_id required")
		c.JSON(http.StatusBadRequest, response.BadRequest("Tenant ID is required"))
		return
	}

	span.SetAttributes(attribute.String("tenant_id", id))

	err := h.tenantService.Delete(ctx, id)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrTenantNotFound) {
			span.SetStatus(codes.Error, "tenant not found")
			c.JSON(http.StatusNotFound, response.NotFound("Tenant not found"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(gin.H{"message": "Tenant deleted successfully"}))
}
