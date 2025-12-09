package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
)

// HealthHandler handles health check HTTP requests
type HealthHandler struct {
	db *database.PostgresDB
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *database.PostgresDB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Health returns basic health status
// GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "auth-service",
	})
}

// Ready checks if the service is ready to accept traffic
// GET /ready
func (h *HealthHandler) Ready(c *gin.Context) {
	// Check database connection
	if err := h.db.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "not_ready",
			"service":  "auth-service",
			"database": "disconnected",
			"error":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"service":  "auth-service",
		"database": "connected",
	})
}
