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
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/di"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/service"
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

	// Initialize database connection
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
		appLog.Warn(fmt.Sprintf("Database connection failed: %v", err))
	} else {
		defer db.Close()
		appLog.Info("Database connected")
	}

	// Initialize Redis connection
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
		appLog.Warn(fmt.Sprintf("Redis connection failed: %v", err))
	} else {
		defer redis.Close()
		appLog.Info("Redis connected")
	}

	// Build dependency injection container
	// Note: Repository implementations will be added in future tasks
	container := di.NewContainer(&di.ContainerConfig{
		DB:              db,
		Redis:           redis,
		BookingRepo:     nil, // TODO: Implement PostgresBookingRepository
		ReservationRepo: nil, // TODO: Implement RedisReservationRepository
		ServiceConfig: &service.BookingServiceConfig{
			ReservationTTL: 10 * time.Minute,
			MaxPerUser:     10,
		},
	})

	// Setup Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Health check endpoints
	router.GET("/health", container.HealthHandler.Health)
	router.GET("/ready", container.HealthHandler.Ready)

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

		// Booking routes (will need auth middleware)
		bookings := v1.Group("/bookings")
		{
			bookings.POST("/reserve", container.BookingHandler.ReserveSeats)
			bookings.GET("", container.BookingHandler.GetUserBookings)
			bookings.GET("/:id", container.BookingHandler.GetBooking)
			bookings.POST("/:id/confirm", container.BookingHandler.ConfirmBooking)
			bookings.DELETE("/:id", container.BookingHandler.ReleaseBooking)
		}
	}

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
