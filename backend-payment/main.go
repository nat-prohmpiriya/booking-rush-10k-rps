package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment-service/internal/di"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment-service/internal/gateway"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment-service/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment-service/internal/service"
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

	// Initialize payment gateway based on feature flag
	gatewayType := getEnv("PAYMENT_GATEWAY", "mock")
	var paymentGateway gateway.PaymentGateway
	var gwErr error

	if gatewayType == "stripe" {
		stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
		if stripeSecretKey == "" {
			appLog.Warn("STRIPE_SECRET_KEY not set, falling back to mock gateway")
			gatewayType = "mock"
		} else {
			paymentGateway, gwErr = gateway.NewPaymentGateway("stripe", &gateway.GatewayConfig{
				SecretKey:   stripeSecretKey,
				Environment: getEnv("STRIPE_ENVIRONMENT", "test"),
			})
			if gwErr != nil {
				appLog.Warn(fmt.Sprintf("Failed to create Stripe gateway: %v, falling back to mock", gwErr))
				gatewayType = "mock"
			}
		}
	}

	if gatewayType == "mock" || paymentGateway == nil {
		successRate := getEnvFloat("MOCK_GATEWAY_SUCCESS_RATE", 0.95)
		delayMs := getEnvInt("MOCK_GATEWAY_DELAY_MS", 100)
		paymentGateway = gateway.NewMockGatewayWithConfig(successRate, delayMs)
		appLog.Info(fmt.Sprintf("Using mock payment gateway (success_rate=%.2f, delay_ms=%d)", successRate, delayMs))
	} else {
		appLog.Info("Using Stripe payment gateway")
	}

	// Initialize payment repository
	var paymentRepo repository.PaymentRepository
	if db != nil {
		paymentRepo = repository.NewPostgresPaymentRepository(db)
		appLog.Info("Using PostgreSQL payment repository")
	} else {
		paymentRepo = repository.NewMemoryPaymentRepository()
		appLog.Warn("Using in-memory payment repository (data will not persist)")
	}

	// Build dependency injection container
	container := di.NewContainer(&di.ContainerConfig{
		DB:             db,
		Redis:          redisClient,
		PaymentRepo:    paymentRepo,
		PaymentGateway: paymentGateway,
		ServiceConfig: &service.PaymentServiceConfig{
			Currency:        "THB",
			GatewayType:     gatewayType,
			MockSuccessRate: getEnvFloat("MOCK_GATEWAY_SUCCESS_RATE", 0.95),
			MockDelayMs:     getEnvInt("MOCK_GATEWAY_DELAY_MS", 100),
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

		// Payment routes
		if container.PaymentHandler != nil {
			payments := v1.Group("/payments")

			// Configure idempotency middleware for write operations
			var idempotencyConfig *middleware.IdempotencyConfig
			if redisClient != nil {
				idempotencyConfig = middleware.DefaultIdempotencyConfig(redisClient.Client())
				idempotencyConfig.SkipPaths = []string{"/health", "/ready"}
			}

			{
				// Write operations with idempotency (if Redis available)
				if idempotencyConfig != nil {
					payments.POST("", middleware.IdempotencyMiddleware(idempotencyConfig), container.PaymentHandler.CreatePayment)
					payments.POST("/:id/process", middleware.IdempotencyMiddleware(idempotencyConfig), container.PaymentHandler.ProcessPayment)
					payments.POST("/:id/refund", middleware.IdempotencyMiddleware(idempotencyConfig), container.PaymentHandler.RefundPayment)
					payments.POST("/:id/cancel", middleware.IdempotencyMiddleware(idempotencyConfig), container.PaymentHandler.CancelPayment)
				} else {
					payments.POST("", container.PaymentHandler.CreatePayment)
					payments.POST("/:id/process", container.PaymentHandler.ProcessPayment)
					payments.POST("/:id/refund", container.PaymentHandler.RefundPayment)
					payments.POST("/:id/cancel", container.PaymentHandler.CancelPayment)
				}

				// Read operations without idempotency
				payments.GET("/:id", container.PaymentHandler.GetPayment)
				payments.GET("/booking/:bookingId", container.PaymentHandler.GetPaymentByBookingID)
				payments.GET("/user/:userId", container.PaymentHandler.GetUserPayments)
			}
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

// getEnv returns environment variable or default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.Atoi(value); err == nil {
			return result
		}
	}
	return defaultValue
}

// getEnvFloat returns environment variable as float64 or default value
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.ParseFloat(value, 64); err == nil {
			return result
		}
	}
	return defaultValue
}
