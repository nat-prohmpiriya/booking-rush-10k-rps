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
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/middleware"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
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

	// Initialize OpenTelemetry
	telemetryCfg := &telemetry.Config{
		Enabled:        cfg.OTel.Enabled,
		ServiceName:    "booking-service",
		ServiceVersion: cfg.App.Version,
		Environment:    cfg.App.Environment,
		CollectorAddr:  cfg.OTel.CollectorAddr,
		SampleRatio:    cfg.OTel.SampleRatio,
	}
	if _, err := telemetry.Init(ctx, telemetryCfg); err != nil {
		appLog.Warn(fmt.Sprintf("Failed to initialize telemetry: %v", err))
	} else if telemetryCfg.Enabled {
		appLog.Info(fmt.Sprintf("Telemetry initialized (collector: %s)", telemetryCfg.CollectorAddr))
	}
	defer telemetry.Shutdown(ctx)

	// Initialize database connection with optimized settings for 10k RPS
	// Uses BookingDatabase config (Microservice - each service has its own database)
	var db *database.PostgresDB
	dbCfg := &database.PostgresConfig{
		Host:            cfg.BookingDatabase.Host,
		Port:            cfg.BookingDatabase.Port,
		User:            cfg.BookingDatabase.User,
		Password:        cfg.BookingDatabase.Password,
		Database:        cfg.BookingDatabase.DBName,
		SSLMode:         cfg.BookingDatabase.SSLMode,
		MaxConns:        20,               // Optimized: Virtual Queue controls traffic, Redis handles inventory
		MinConns:        5,
		MaxConnLifetime: 30 * time.Minute, // Reduce to prevent stale connections
		MaxConnIdleTime: 5 * time.Minute,  // Close idle connections sooner
		ConnectTimeout:  5 * time.Second,  // Fast fail
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
		EnableTracing:   cfg.OTel.Enabled,
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
		EnableTracing: cfg.OTel.Enabled,
		ServiceName:   "booking-service",
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

	// Initialize Saga producer and store for saga-based bookings
	var sagaProducer saga.SagaProducer
	var sagaStore pkgsaga.Store
	sagaProducer, err = saga.NewKafkaSagaProducer(ctx, &saga.KafkaSagaProducerConfig{
		Brokers:       cfg.Kafka.Brokers,
		ClientID:      "booking-service-saga-producer",
		MaxRetries:    3,
		RetryInterval: time.Second,
		Logger:        &saga.ZapLogger{},
	})
	if err != nil {
		appLog.Warn(fmt.Sprintf("Saga producer init failed: %v", err))
	} else {
		appLog.Info("Saga producer connected")
		sagaStore = pkgsaga.NewMemoryStore() // In-memory for now, can switch to Redis/Postgres
	}

	// Initialize repositories
	bookingRepo := repository.NewPostgresBookingRepository(db.Pool())
	reservationRepo := repository.NewRedisReservationRepository(redisClient)
	queueRepo := repository.NewRedisQueueRepository(redisClient)

	// Pre-load Lua scripts into Redis
	if err := reservationRepo.LoadScripts(ctx); err != nil {
		appLog.Warn(fmt.Sprintf("Failed to pre-load reservation Lua scripts: %v", err))
	} else {
		appLog.Info("Reservation Lua scripts pre-loaded into Redis")
	}

	if err := queueRepo.LoadScripts(ctx); err != nil {
		appLog.Warn(fmt.Sprintf("Failed to pre-load queue Lua scripts: %v", err))
	} else {
		appLog.Info("Queue Lua scripts pre-loaded into Redis")
	}

	// Check if saga mode is enabled via environment variable
	useSagaForBooking := os.Getenv("USE_SAGA_FOR_BOOKING") == "true"
	if useSagaForBooking && sagaProducer != nil {
		appLog.Info("Saga mode ENABLED for booking - /bookings/reserve will use saga pattern")
	} else {
		appLog.Info("Saga mode DISABLED for booking - using direct sync flow")
	}

	// Build dependency injection container
	container := di.NewContainer(&di.ContainerConfig{
		DB:              db,
		Redis:           redisClient,
		BookingRepo:     bookingRepo,
		ReservationRepo: reservationRepo,
		QueueRepo:       queueRepo,
		EventPublisher:  eventPublisher,
		ServiceConfig: &service.BookingServiceConfig{
			ReservationTTL: 10 * time.Minute,
			MaxPerUser:     10,
		},
		QueueServiceConfig: &service.QueueServiceConfig{
			QueueTTL:             30 * time.Minute,
			MaxQueueSize:         0, // Unlimited
			EstimatedWaitPerUser: 3, // 3 seconds per user
			JWTSecret:            cfg.JWT.Secret,
		},
		TicketServiceURL:  cfg.Services.TicketServiceURL, // For auto-sync zone on ZONE_NOT_FOUND
		SagaProducer:      sagaProducer,
		SagaStore:         sagaStore,
		UseSagaForBooking: useSagaForBooking, // Enable saga-based booking
		SagaServiceConfig: &service.SagaServiceConfig{
			StepTimeout: 30 * time.Second,
			MaxRetries:  2,
		},
	})

	// Setup Gin with optimized settings
	gin.SetMode(gin.ReleaseMode) // Always use release mode for performance
	gin.DisableConsoleColor()

	router := gin.New()

	// Use minimal middleware for performance
	router.Use(gin.Recovery())

	// Add OpenTelemetry tracing middleware if enabled
	if cfg.OTel.Enabled {
		router.Use(telemetry.TracingMiddleware("booking-service"))
		router.Use(telemetry.TraceHeaderMiddleware())
	}

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

		// Queue routes - Virtual Queue for high-demand events
		queue := v1.Group("/queue")
		queue.Use(userIDMiddleware()) // Extract user_id from header
		{
			// Join queue (requires authentication)
			queue.POST("/join", middleware.IdempotencyMiddleware(idempotencyConfig), container.QueueHandler.JoinQueue)

			// Get current position in queue
			queue.GET("/position/:event_id", container.QueueHandler.GetPosition)

			// Leave queue
			queue.DELETE("/leave", container.QueueHandler.LeaveQueue)

			// Get queue status for an event (public)
			queue.GET("/status/:event_id", container.QueueHandler.GetQueueStatus)
		}

		// Admin routes - for managing inventory sync
		admin := v1.Group("/admin")
		{
			// Sync zone availability from PostgreSQL to Redis
			admin.POST("/sync-inventory", container.AdminHandler.SyncInventory)

			// Get inventory status (PostgreSQL vs Redis)
			admin.GET("/inventory-status", container.AdminHandler.GetInventoryStatus)
		}

		// Saga routes - async booking via saga pattern
		sagaRoutes := v1.Group("/saga")
		sagaRoutes.Use(userIDMiddleware()) // Extract user_id from header
		{
			// Start a new booking saga (async)
			sagaRoutes.POST("/bookings", middleware.IdempotencyMiddleware(idempotencyConfig), container.SagaHandler.StartBookingSaga)

			// Get saga status
			sagaRoutes.GET("/bookings/:saga_id", container.SagaHandler.GetSagaStatus)
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

// userIDMiddleware extracts user_id and tenant_id from headers
func userIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			// For load testing, generate a test user ID if not provided
			userID = "test-user-1"
		}
		c.Set("user_id", userID)

		// Extract tenant_id from header (set by API Gateway from JWT)
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID != "" {
			c.Set("tenant_id", tenantID)
		}

		c.Next()
	}
}
