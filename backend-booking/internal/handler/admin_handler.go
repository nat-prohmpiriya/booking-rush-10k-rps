package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// AdminHandler handles admin HTTP requests
type AdminHandler struct {
	db    *database.PostgresDB
	redis *pkgredis.Client
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db *database.PostgresDB, redis *pkgredis.Client) *AdminHandler {
	return &AdminHandler{
		db:    db,
		redis: redis,
	}
}

// SyncInventoryResponse represents the response for sync inventory
type SyncInventoryResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ZonesSynced int    `json:"zones_synced"`
}

// SyncInventory handles POST /admin/sync-inventory
// Syncs zone availability from PostgreSQL to Redis
func (h *AdminHandler) SyncInventory(c *gin.Context) {
	ctx := c.Request.Context()

	count, err := h.syncZoneAvailability(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "failed to sync inventory",
			Code:    "SYNC_FAILED",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SyncInventoryResponse{
		Success:     true,
		Message:     fmt.Sprintf("Successfully synced %d zones to Redis", count),
		ZonesSynced: count,
	})
}

// syncZoneAvailability syncs all zone availability from PostgreSQL to Redis
func (h *AdminHandler) syncZoneAvailability(ctx context.Context) (int, error) {
	// Query all active seat zones from PostgreSQL
	query := `
		SELECT id, available_seats
		FROM seat_zones
		WHERE is_active = true AND deleted_at IS NULL
	`

	rows, err := h.db.Pool().Query(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to query seat zones: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var zoneID string
		var availableSeats int64

		if err := rows.Scan(&zoneID, &availableSeats); err != nil {
			continue
		}

		// Set zone availability in Redis
		key := fmt.Sprintf("zone:availability:%s", zoneID)
		if err := h.redis.Set(ctx, key, availableSeats, 0).Err(); err != nil {
			continue
		}

		count++
	}

	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("error iterating rows: %w", err)
	}

	return count, nil
}

// GetInventoryStatus handles GET /admin/inventory-status
// Returns current inventory status from both PostgreSQL and Redis
func (h *AdminHandler) GetInventoryStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Query zones from PostgreSQL
	query := `
		SELECT id, name, available_seats, reserved_seats, sold_seats, total_seats
		FROM seat_zones
		WHERE is_active = true AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := h.db.Pool().Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "failed to query zones",
			Code:    "QUERY_FAILED",
			Message: err.Error(),
		})
		return
	}
	defer rows.Close()

	type ZoneStatus struct {
		ZoneID         string `json:"zone_id"`
		Name           string `json:"name"`
		PGAvailable    int64  `json:"pg_available"`
		PGReserved     int64  `json:"pg_reserved"`
		PGSold         int64  `json:"pg_sold"`
		PGTotal        int64  `json:"pg_total"`
		RedisAvailable int64  `json:"redis_available"`
		InSync         bool   `json:"in_sync"`
	}

	var zones []ZoneStatus
	for rows.Next() {
		var z ZoneStatus
		if err := rows.Scan(&z.ZoneID, &z.Name, &z.PGAvailable, &z.PGReserved, &z.PGSold, &z.PGTotal); err != nil {
			continue
		}

		// Get Redis value
		key := fmt.Sprintf("zone:availability:%s", z.ZoneID)
		val, err := h.redis.Get(ctx, key).Int64()
		if err != nil {
			z.RedisAvailable = -1 // Not set in Redis
			z.InSync = false
		} else {
			z.RedisAvailable = val
			z.InSync = (z.PGAvailable == z.RedisAvailable)
		}

		zones = append(zones, z)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    zones,
		"count":   len(zones),
	})
}
