package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/nat-prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/nat-prohmpiriya/booking-rush-10k-rps/pkg/logger"
	"github.com/nat-prohmpiriya/booking-rush-10k-rps/pkg/response"
	"go.uber.org/zap"
)

func main() {
	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Init Logger
	logger.Init(cfg.App.LogLevel)
	logger.Info("Starting API Gateway", zap.String("port", cfg.App.Port))

	// Init Gin
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(LoggerMiddleware())

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":  "up",
			"service": "api-gateway",
		})
	})

	// Readiness Check
	r.GET("/ready", func(c *gin.Context) {
		// TODO: Check dependencies (DB, Redis, Kafka)
		response.Success(c, gin.H{
			"status": "ready",
		})
	})

	// Start Server
	if err := r.Run(":" + cfg.App.Port); err != nil {
		logger.Error("Failed to start server", zap.Error(err))
	}
}

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		status := c.Writer.Status()
		logger.Info("Request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
		)
	}
}
