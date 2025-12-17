package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// AdminHandler handles admin HTTP requests
type AdminHandler struct {
	redis            *pkgredis.Client
	ticketServiceURL string
	httpClient       *http.Client
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(redis *pkgredis.Client) *AdminHandler {
	ticketURL := os.Getenv("TICKET_SERVICE_URL")
	if ticketURL == "" {
		ticketURL = "http://localhost:8082"
	}

	return &AdminHandler{
		redis:            redis,
		ticketServiceURL: ticketURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SyncInventoryResponse represents the response for sync inventory
type SyncInventoryResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ZonesSynced int    `json:"zones_synced"`
}

// ZoneFromTicketService represents zone data from ticket service API
type ZoneFromTicketService struct {
	ID             string `json:"id"`
	ShowID         string `json:"show_id"`
	Name           string `json:"name"`
	AvailableSeats int    `json:"available_seats"`
	TotalSeats     int    `json:"total_seats"`
	IsActive       bool   `json:"is_active"`
}

// TicketServiceResponse represents the API response from ticket service
type TicketServiceResponse struct {
	Success bool                    `json:"success"`
	Data    []ZoneFromTicketService `json:"data"`
}

// SyncInventory handles POST /admin/sync-inventory
// Syncs zone availability from Ticket Service API to Redis
func (h *AdminHandler) SyncInventory(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.admin.sync_inventory")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	count, err := h.syncZoneAvailability(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "failed to sync inventory",
			Code:    "SYNC_FAILED",
			Message: err.Error(),
		})
		return
	}

	span.SetAttributes(attribute.Int("zones_synced", count))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, SyncInventoryResponse{
		Success:     true,
		Message:     fmt.Sprintf("Successfully synced %d zones to Redis", count),
		ZonesSynced: count,
	})
}

// syncZoneAvailability syncs all zone availability from Ticket Service API to Redis
func (h *AdminHandler) syncZoneAvailability(ctx context.Context) (int, error) {
	// Call ticket service API to get active zones
	url := fmt.Sprintf("%s/api/v1/zones/active", h.ticketServiceURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to call ticket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("ticket service returned status %d", resp.StatusCode)
	}

	var ticketResp TicketServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&ticketResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if !ticketResp.Success {
		return 0, fmt.Errorf("ticket service returned error")
	}

	count := 0
	for _, zone := range ticketResp.Data {
		// Set zone availability in Redis
		key := fmt.Sprintf("zone:availability:%s", zone.ID)
		if err := h.redis.Set(ctx, key, zone.AvailableSeats, 0).Err(); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// GetInventoryStatus handles GET /admin/inventory-status
// Returns current inventory status from Ticket Service API and Redis
func (h *AdminHandler) GetInventoryStatus(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.admin.inventory_status")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	// Call ticket service API to get active zones
	url := fmt.Sprintf("%s/api/v1/zones/active", h.ticketServiceURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "failed to create request",
			Code:    "REQUEST_FAILED",
			Message: err.Error(),
		})
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "failed to call ticket service",
			Code:    "SERVICE_CALL_FAILED",
			Message: err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		span.SetStatus(codes.Error, fmt.Sprintf("ticket service returned status %d", resp.StatusCode))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "ticket service error",
			Code:    "SERVICE_ERROR",
			Message: fmt.Sprintf("ticket service returned status %d", resp.StatusCode),
		})
		return
	}

	var ticketResp TicketServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&ticketResp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "failed to decode response",
			Code:    "DECODE_FAILED",
			Message: err.Error(),
		})
		return
	}

	type ZoneStatus struct {
		ZoneID          string `json:"zone_id"`
		Name            string `json:"name"`
		TicketAvailable int    `json:"ticket_available"`
		TicketTotal     int    `json:"ticket_total"`
		RedisAvailable  int64  `json:"redis_available"`
		InSync          bool   `json:"in_sync"`
	}

	var zones []ZoneStatus
	for _, zone := range ticketResp.Data {
		z := ZoneStatus{
			ZoneID:          zone.ID,
			Name:            zone.Name,
			TicketAvailable: zone.AvailableSeats,
			TicketTotal:     zone.TotalSeats,
		}

		// Get Redis value
		key := fmt.Sprintf("zone:availability:%s", zone.ID)
		val, err := h.redis.Get(ctx, key).Int64()
		if err != nil {
			z.RedisAvailable = -1 // Not set in Redis
			z.InSync = false
		} else {
			z.RedisAvailable = val
			z.InSync = (int64(zone.AvailableSeats) == z.RedisAvailable)
		}

		zones = append(zones, z)
	}

	span.SetAttributes(attribute.Int("zones_count", len(zones)))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    zones,
		"count":   len(zones),
	})
}
