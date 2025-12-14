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
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/di"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/middleware"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
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
		ServiceName: "ticket-service",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Ticket Service...")

	ctx := context.Background()

	// Initialize OpenTelemetry
	telemetryCfg := &telemetry.Config{
		Enabled:        cfg.OTel.Enabled,
		ServiceName:    "ticket-service",
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

	// Initialize database connection (uses TicketDatabase config)
	var db *database.PostgresDB
	dbCfg := &database.PostgresConfig{
		Host:            cfg.TicketDatabase.Host,
		Port:            cfg.TicketDatabase.Port,
		User:            cfg.TicketDatabase.User,
		Password:        cfg.TicketDatabase.Password,
		Database:        cfg.TicketDatabase.DBName,
		SSLMode:         cfg.TicketDatabase.SSLMode,
		MaxConns:        10, // Optimized: uses Redis cache heavily
		MinConns:        2,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		ConnectTimeout:  5 * time.Second,
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

	// Initialize Redis connection (optional - cache will be disabled if connection fails)
	var redisClient *redis.Client
	redisCfg := &redis.Config{
		Host:          cfg.Redis.Host,
		Port:          cfg.Redis.Port,
		Password:      cfg.Redis.Password,
		DB:            cfg.Redis.DB,
		PoolSize:      cfg.Redis.PoolSize,
		MinIdleConns:  cfg.Redis.MinIdleConns,
		DialTimeout:   cfg.Redis.DialTimeout,
		ReadTimeout:   cfg.Redis.ReadTimeout,
		WriteTimeout:  cfg.Redis.WriteTimeout,
		MaxRetries:    3,
		RetryInterval: time.Second,
		EnableTracing: cfg.OTel.Enabled,
		ServiceName:   "ticket-service",
	}
	redisClient, err = redis.NewClient(ctx, redisCfg)
	if err != nil {
		appLog.Warn(fmt.Sprintf("Redis connection failed (caching disabled): %v", err))
		redisClient = nil
	} else {
		defer redisClient.Close()
		appLog.Info(fmt.Sprintf("Redis connected (%s)", redisCfg.Addr()))
	}

	// Build dependency injection container
	container := di.NewContainer(&di.ContainerConfig{
		DB:    db,
		Redis: redisClient,
	})

	// Setup Gin
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Add OpenTelemetry tracing middleware if enabled
	if cfg.OTel.Enabled {
		router.Use(telemetry.TracingMiddleware("ticket-service"))
		router.Use(telemetry.TraceHeaderMiddleware())
	}

	// Health check endpoints
	router.GET("/health", container.HealthHandler.Health)
	router.GET("/ready", container.HealthHandler.Ready)

	// JWT middleware configuration
	jwtConfig := &middleware.JWTConfig{
		Secret: cfg.JWT.Secret,
		SkipPaths: []string{
			"/health",
			"/ready",
		},
	}

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Events endpoints - public read, authenticated write
		events := v1.Group("/events")
		{
			// Public endpoints (no auth required)
			events.GET("", container.EventHandler.List)
			events.GET("/id/:id", container.EventHandler.GetByID)
			events.GET("/:slug/shows", container.ShowHandler.ListByEvent)

			// Protected endpoints (Organizer/Admin only)
			protected := events.Group("")
			protected.Use(middleware.JWTMiddleware(jwtConfig))
			protected.Use(middleware.RequireRole("admin", "organizer"))
			{
				protected.GET("/my", container.EventHandler.ListMyEvents) // Must be before /:slug
				protected.POST("", container.EventHandler.Create)
				protected.PUT("/:id", container.EventHandler.Update)
				protected.DELETE("/:id", container.EventHandler.Delete)
				protected.POST("/:id/publish", container.EventHandler.Publish)
				protected.POST("/:id/shows", container.ShowHandler.Create)
			}

			// This must be last to avoid catching /id/:id, /my, etc.
			events.GET("/:slug", container.EventHandler.GetBySlug)
		}

		// Shows endpoints - for direct show access
		shows := v1.Group("/shows")
		{
			// Public endpoints
			shows.GET("/:id", container.ShowHandler.GetByID)
			shows.GET("/:id/zones", container.ShowZoneHandler.ListByShow)

			// Protected endpoints (Organizer/Admin only)
			protectedShows := shows.Group("")
			protectedShows.Use(middleware.JWTMiddleware(jwtConfig))
			protectedShows.Use(middleware.RequireRole("admin", "organizer"))
			{
				protectedShows.PUT("/:id", container.ShowHandler.Update)
				protectedShows.DELETE("/:id", container.ShowHandler.Delete)
				protectedShows.POST("/:id/zones", container.ShowZoneHandler.Create)
			}
		}

		// Zones endpoints - for direct zone access
		zones := v1.Group("/zones")
		{
			// Public endpoints (note: /active must come before /:id to avoid route conflict)
			zones.GET("/active", container.ShowZoneHandler.ListActive)
			zones.GET("/:id", container.ShowZoneHandler.GetByID)

			// Protected endpoints (Organizer/Admin only)
			protectedZones := zones.Group("")
			protectedZones.Use(middleware.JWTMiddleware(jwtConfig))
			protectedZones.Use(middleware.RequireRole("admin", "organizer"))
			{
				protectedZones.PUT("/:id", container.ShowZoneHandler.Update)
				protectedZones.DELETE("/:id", container.ShowZoneHandler.Delete)
			}
		}

		// Tickets endpoints (to be implemented)
		_ = v1.Group("/tickets")
		// {
		// 	tickets.GET("/types/:eventId", container.TicketHandler.GetByEvent)
		// 	tickets.GET("/types/:id", container.TicketHandler.GetType)
		// 	tickets.POST("/types", container.TicketHandler.CreateType)
		// 	tickets.PUT("/types/:id", container.TicketHandler.UpdateType)
		// 	tickets.DELETE("/types/:id", container.TicketHandler.DeleteType)
		// 	tickets.POST("/availability", container.TicketHandler.CheckAvailability)
		// }

		// Venues endpoints (to be implemented)
		_ = v1.Group("/venues")
		// {
		// 	venues.GET("", container.VenueHandler.List)
		// 	venues.GET("/:id", container.VenueHandler.Get)
		// 	venues.POST("", container.VenueHandler.Create)
		// 	venues.PUT("/:id", container.VenueHandler.Update)
		// 	venues.DELETE("/:id", container.VenueHandler.Delete)
		// }
	}

	// Create HTTP server
	port := cfg.Server.Port
	if port == 0 {
		port = 8082 // Default port for ticket-service
	}
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	// Start server in goroutine
	go func() {
		appLog.Info(fmt.Sprintf("Ticket Service listening on %s", addr))
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
