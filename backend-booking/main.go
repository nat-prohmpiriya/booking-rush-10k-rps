package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Import pprof for profiling
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/di"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/middleware"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

func main() {
	// Optimize Go runtime for high concurrency
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logCfg := &logger.Config{
		Level:       cfg.App.Environment,
		ServiceName: "booking-service",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Booking Service...")

	ctx := context.Background()

	// Initialize database connection with optimized settings for 10k RPS
	var db *database.PostgresDB
	dbCfg := &database.PostgresConfig{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxConns:        200,              // Increased for 10k RPS
		MinConns:        50,               // Keep minimum pool ready
		MaxConnLifetime: 30 * time.Minute, // Reduce to prevent stale connections
		MaxConnIdleTime: 5 * time.Minute,  // Close idle connections sooner
		ConnectTimeout:  5 * time.Second,  // Fast fail
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
	}
	db, err = database.NewPostgres(ctx, dbCfg)
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Database connection failed: %v", err))
	}
	defer db.Close()
	appLog.Info(fmt.Sprintf("Database connected (pool: min=%d, max=%d)", dbCfg.MinConns, dbCfg.MaxConns))

	// Initialize Redis connection with optimized settings for 10k RPS
	var redisClient *pkgredis.Client
	redisCfg := &pkgredis.Config{
		Host:          cfg.Redis.Host,
		Port:          cfg.Redis.Port,
		Password:      cfg.Redis.Password,
		DB:            cfg.Redis.DB,
		PoolSize:      500, // Large pool for 10k RPS
		MinIdleConns:  100, // Keep connections ready
		MaxRetries:    3,
		RetryInterval: 100 * time.Millisecond,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		PoolTimeout:   4 * time.Second,
	}
	redisClient, err = pkgredis.NewClient(ctx, redisCfg)
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Redis connection failed: %v", err))
	}
	defer redisClient.Close()
	appLog.Info(fmt.Sprintf("Redis connected (pool: %d, minIdle: %d)", redisCfg.PoolSize, redisCfg.MinIdleConns))

	// Initialize Kafka event publisher
	var eventPublisher service.EventPublisher
	eventPubCfg := &service.EventPublisherConfig{
		Brokers:     cfg.Kafka.Brokers,
		Topic:       "booking-events",
		ServiceName: "booking-service",
		ClientID:    cfg.Kafka.ClientID,
	}
	eventPublisher, err = service.NewKafkaEventPublisher(ctx, eventPubCfg)
	if err != nil {
		appLog.Warn(fmt.Sprintf("Kafka connection failed, using no-op publisher: %v", err))
		eventPublisher = service.NewNoOpEventPublisher()
	} else {
		appLog.Info("Kafka event publisher connected")
	}

	// Initialize repositories
	bookingRepo := repository.NewPostgresBookingRepository(db.Pool())
	reservationRepo := repository.NewRedisReservationRepository(redisClient)

	// Pre-load Lua scripts into Redis
	if err := reservationRepo.LoadScripts(ctx); err != nil {
		appLog.Warn(fmt.Sprintf("Failed to pre-load Lua scripts: %v", err))
	} else {
		appLog.Info("Lua scripts pre-loaded into Redis")
	}

	// Build dependency injection container
	container := di.NewContainer(&di.ContainerConfig{
		DB:              db,
		Redis:           redisClient,
		BookingRepo:     bookingRepo,
		ReservationRepo: reservationRepo,
		EventPublisher:  eventPublisher,
		ServiceConfig: &service.BookingServiceConfig{
			ReservationTTL: 10 * time.Minute,
			MaxPerUser:     10,
		},
	})

	// Setup Gin with optimized settings
	gin.SetMode(gin.ReleaseMode) // Always use release mode for performance
	gin.DisableConsoleColor()

	router := gin.New()

	// Use minimal middleware for performance
	router.Use(gin.Recovery())

	// Health check endpoints
	router.GET("/health", container.HealthHandler.Health)
	router.GET("/ready", container.HealthHandler.Ready)

	// Metrics endpoint for monitoring
	router.GET("/metrics", func(c *gin.Context) {
		stats := db.Stats()
		c.JSON(http.StatusOK, gin.H{
			"db_pool": gin.H{
				"total_conns":        stats.TotalConns(),
				"acquired_conns":     stats.AcquiredConns(),
				"idle_conns":         stats.IdleConns(),
				"max_conns":          stats.MaxConns(),
				"constructing_conns": stats.ConstructingConns(),
			},
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Status endpoint
		v1.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"version": cfg.App.Version,
				"service": "booking-service",
			})
		})

		// Booking routes - simplified middleware for performance
		bookings := v1.Group("/bookings")
		bookings.Use(userIDMiddleware()) // Extract user_id from header

		// Configure idempotency middleware for write operations
		idempotencyConfig := middleware.DefaultIdempotencyConfig(redisClient.Client())
		idempotencyConfig.SkipPaths = []string{"/health", "/ready", "/metrics"}

		{
			// Write operations with idempotency
			bookings.POST("/reserve", middleware.IdempotencyMiddleware(idempotencyConfig), container.BookingHandler.ReserveSeats)
			bookings.POST("/:id/confirm", middleware.IdempotencyMiddleware(idempotencyConfig), container.BookingHandler.ConfirmBooking)
			bookings.POST("/:id/cancel", middleware.IdempotencyMiddleware(idempotencyConfig), container.BookingHandler.CancelBooking)
			bookings.DELETE("/:id", middleware.IdempotencyMiddleware(idempotencyConfig), container.BookingHandler.ReleaseBooking)

			// Read operations without idempotency
			bookings.GET("", container.BookingHandler.GetUserBookings)
			bookings.GET("/pending", container.BookingHandler.GetPendingBookings)
			bookings.GET("/:id", container.BookingHandler.GetBooking)
		}
	}

	// Create HTTP server with optimized settings
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	// Start pprof server on separate port for profiling
	go func() {
		pprofAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port+1000)
		appLog.Info(fmt.Sprintf("pprof server listening on %s", pprofAddr))
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			appLog.Error(fmt.Sprintf("pprof server error: %v", err))
		}
	}()

	// Start server in goroutine
	go func() {
		appLog.Info(fmt.Sprintf("Booking Service listening on %s", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLog.Fatal(fmt.Sprintf("Failed to start server: %v", err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLog.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLog.Fatal(fmt.Sprintf("Server forced to shutdown: %v", err))
	}

	appLog.Info("Server exited gracefully")
}

// userIDMiddleware extracts user_id from X-User-ID header for load testing
func userIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			// For load testing, generate a test user ID if not provided
			userID = "test-user-1"
		}
		c.Set("user_id", userID)
		c.Next()
	}
}
