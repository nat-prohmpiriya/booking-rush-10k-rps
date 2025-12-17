package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-api-gateway/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-api-gateway/internal/middleware"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-api-gateway/internal/proxy"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger with OTLP export support
	logCfg := &logger.Config{
		Level:        cfg.App.Environment,
		ServiceName:  "api-gateway",
		Development:  cfg.IsDevelopment(),
		OTLPEnabled:  cfg.OTel.Enabled && cfg.OTel.LogExportEnabled,
		OTLPEndpoint: cfg.OTel.CollectorAddr,
		OTLPInsecure: true,
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	log := logger.Get()
	log.Info("Starting API Gateway...")

	ctx := context.Background()

	// Initialize OpenTelemetry
	telemetryCfg := &telemetry.Config{
		Enabled:        cfg.OTel.Enabled,
		ServiceName:    "api-gateway",
		ServiceVersion: cfg.App.Version,
		Environment:    cfg.App.Environment,
		CollectorAddr:  cfg.OTel.CollectorAddr,
		SampleRatio:    cfg.OTel.SampleRatio,
	}
	if _, err := telemetry.Init(ctx, telemetryCfg); err != nil {
		log.Warn(fmt.Sprintf("Failed to initialize telemetry: %v", err))
	} else if telemetryCfg.Enabled {
		log.Info(fmt.Sprintf("Telemetry initialized (collector: %s)", telemetryCfg.CollectorAddr))
	}
	defer telemetry.Shutdown(ctx)

	// API Gateway does NOT connect to any database directly (Microservice pattern)
	// Each service manages its own database connection
	// Gateway only uses Redis for rate limiting and health checks

	// Initialize Redis connection (for rate limiting and /ready check)
	var redis *pkgredis.Client
	redisCfg := &pkgredis.Config{
		Host:          cfg.Redis.Host,
		Port:          cfg.Redis.Port,
		Password:      cfg.Redis.Password,
		DB:            cfg.Redis.DB,
		PoolSize:      cfg.Redis.PoolSize,
		MaxRetries:    3,
		RetryInterval: 2 * time.Second,
		EnableTracing: cfg.OTel.Enabled,
		ServiceName:   "api-gateway",
	}
	redis, err = pkgredis.NewClient(ctx, redisCfg)
	if err != nil {
		log.Warn("Redis connection failed, /ready will report unhealthy")
	} else {
		defer redis.Close()
		log.Info("Redis connected")
	}

	// Setup Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Apply global middlewares
	router.Use(gin.Recovery())

	// Add OpenTelemetry tracing middleware if enabled
	if cfg.OTel.Enabled {
		router.Use(telemetry.TracingMiddleware("api-gateway"))
		router.Use(telemetry.TraceHeaderMiddleware())
	}

	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(log))
	router.Use(middleware.CORS())

	// Configure per-endpoint rate limiting (can be disabled via ENV for load testing)
	if os.Getenv("RATE_LIMIT_ENABLED") != "false" {
		rateLimitConfig := middleware.DefaultPerEndpointConfig()
		if redis != nil {
			rateLimitConfig.UseRedis = true
			rateLimitConfig.RedisClient = redis
			log.Info("Rate limiting enabled (Redis-backed, distributed)")
		} else {
			log.Info("Rate limiting enabled (local, non-distributed)")
		}
		router.Use(middleware.PerEndpointRateLimiter(rateLimitConfig))
	} else {
		log.Warn("Rate limiting DISABLED (RATE_LIMIT_ENABLED=false)")
	}

	// Health check handlers (no database - microservice pattern)
	healthHandler := handler.NewHealthHandler(nil, redis)
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	// API version prefix
	v1 := router.Group("/api/v1")
	{
		// Status endpoint for clients
		v1.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"version": cfg.App.Version,
				"service": "api-gateway",
			})
		})
	}

	// Configure reverse proxy for backend services
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8081")
	ticketServiceURL := getEnv("TICKET_SERVICE_URL", "http://localhost:8082")
	bookingServiceURL := getEnv("BOOKING_SERVICE_URL", "http://localhost:8083")
	paymentServiceURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8084")

	proxyConfig := proxy.ConfigFromEnv(
		authServiceURL,
		ticketServiceURL,
		bookingServiceURL,
		paymentServiceURL,
		cfg.JWT.Secret,
	)

	reverseProxy := proxy.NewReverseProxy(proxyConfig)
	proxyRouter := proxy.NewRouter(reverseProxy, cfg.JWT.Secret)

	// Use catch-all handler for proxied routes
	router.NoRoute(proxyRouter.MatchHandler())

	log.Info(fmt.Sprintf("Proxy configured: auth=%s, ticket=%s, booking=%s, payment=%s",
		authServiceURL, ticketServiceURL, bookingServiceURL, paymentServiceURL))

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info(fmt.Sprintf("API Gateway listening on %s", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(fmt.Sprintf("Failed to start server: %v", err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(fmt.Sprintf("Server forced to shutdown: %v", err))
	}

	log.Info("Server exited gracefully")
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
