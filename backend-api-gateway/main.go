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
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logCfg := &logger.Config{
		Level:       cfg.App.Environment,
		ServiceName: "api-gateway",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	log := logger.Get()
	log.Info("Starting API Gateway...")

	ctx := context.Background()

	// Initialize database connection (optional, for /ready check)
	var db *database.PostgresDB
	dbCfg := &database.PostgresConfig{
		Host:          cfg.Database.Host,
		Port:          cfg.Database.Port,
		User:          cfg.Database.User,
		Password:      cfg.Database.Password,
		Database:      cfg.Database.DBName,
		SSLMode:       cfg.Database.SSLMode,
		MaxConns:      int32(cfg.Database.MaxOpenConns),
		MinConns:      int32(cfg.Database.MaxIdleConns),
		MaxRetries:    3,
		RetryInterval: 2 * time.Second,
	}
	db, err = database.NewPostgres(ctx, dbCfg)
	if err != nil {
		log.Warn("Database connection failed, /ready will report unhealthy")
	} else {
		defer db.Close()
		log.Info("Database connected")
	}

	// Initialize Redis connection (optional, for /ready check)
	var redis *pkgredis.Client
	redisCfg := &pkgredis.Config{
		Host:          cfg.Redis.Host,
		Port:          cfg.Redis.Port,
		Password:      cfg.Redis.Password,
		DB:            cfg.Redis.DB,
		PoolSize:      cfg.Redis.PoolSize,
		MaxRetries:    3,
		RetryInterval: 2 * time.Second,
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
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(log))
	router.Use(middleware.CORS())

	// Configure per-endpoint rate limiting
	rateLimitConfig := middleware.DefaultPerEndpointConfig()
	if redis != nil {
		rateLimitConfig.UseRedis = true
		rateLimitConfig.RedisClient = redis
		log.Info("Rate limiting enabled (Redis-backed, distributed)")
	} else {
		log.Info("Rate limiting enabled (local, non-distributed)")
	}
	router.Use(middleware.PerEndpointRateLimiter(rateLimitConfig))

	// Health check handlers
	healthHandler := handler.NewHealthHandler(db, redis)
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
