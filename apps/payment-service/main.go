package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/di"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
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
		ServiceName: "payment-service",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Payment Service...")

	ctx := context.Background()

	// Initialize database connection
	var db *database.PostgresDB
	dbCfg := &database.PostgresConfig{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxConns:        100,
		MinConns:        20,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		ConnectTimeout:  5 * time.Second,
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
	}
	db, err = database.NewPostgres(ctx, dbCfg)
	if err != nil {
		appLog.Warn(fmt.Sprintf("Database connection failed: %v", err))
	} else {
		defer db.Close()
		appLog.Info(fmt.Sprintf("Database connected (pool: min=%d, max=%d)", dbCfg.MinConns, dbCfg.MaxConns))
	}

	// Initialize Redis connection
	var redisClient *pkgredis.Client
	redisCfg := &pkgredis.Config{
		Host:          cfg.Redis.Host,
		Port:          cfg.Redis.Port,
		Password:      cfg.Redis.Password,
		DB:            cfg.Redis.DB,
		PoolSize:      100,
		MinIdleConns:  20,
		MaxRetries:    3,
		RetryInterval: 100 * time.Millisecond,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		PoolTimeout:   4 * time.Second,
	}
	redisClient, err = pkgredis.NewClient(ctx, redisCfg)
	if err != nil {
		appLog.Warn(fmt.Sprintf("Redis connection failed: %v", err))
	} else {
		defer redisClient.Close()
		appLog.Info(fmt.Sprintf("Redis connected (pool: %d, minIdle: %d)", redisCfg.PoolSize, redisCfg.MinIdleConns))
	}

	// Build dependency injection container
	container := di.NewContainer(&di.ContainerConfig{
		DB:    db,
		Redis: redisClient,
		ServiceConfig: &service.PaymentServiceConfig{
			Currency: "THB",
		},
	})

	// Setup Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Apply middlewares
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
				"service": "payment-service",
			})
		})

		// Payment routes (placeholder for future implementation)
		payments := v1.Group("/payments")
		{
			payments.GET("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "payment list endpoint",
				})
			})
		}
	}

	// Create HTTP server
	port := getEnvInt("PORT", 8084)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	// Start server in goroutine
	go func() {
		appLog.Info(fmt.Sprintf("Payment Service listening on %s", addr))
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

// getEnvInt returns environment variable as int or default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
