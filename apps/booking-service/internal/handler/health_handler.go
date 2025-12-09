package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db    *database.PostgresDB
	redis *redis.Client
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *database.PostgresDB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ReadyResponse represents readiness check response
type ReadyResponse struct {
	Status     string            `json:"status"`
	Timestamp  string            `json:"timestamp"`
	Components map[string]string `json:"components"`
}

// Health returns a simple health check (liveness probe)
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready returns a readiness check (readiness probe)
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	components := make(map[string]string)
	allHealthy := true

	// Check database
	if h.db != nil {
		if err := h.db.HealthCheck(ctx); err != nil {
			components["database"] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			components["database"] = "healthy"
		}
	} else {
		components["database"] = "not configured"
	}

	// Check Redis
	if h.redis != nil {
		if err := h.redis.HealthCheck(ctx); err != nil {
			components["redis"] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			components["redis"] = "healthy"
		}
	} else {
		components["redis"] = "not configured"
	}

	response := ReadyResponse{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Components: components,
	}

	if allHealthy {
		response.Status = "ready"
		c.JSON(http.StatusOK, response)
	} else {
		response.Status = "not ready"
		c.JSON(http.StatusServiceUnavailable, response)
	}
}
