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
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/di"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
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
		ServiceName: "auth-service",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Auth Service...")

	ctx := context.Background()

	// Initialize OpenTelemetry
	telemetryCfg := &telemetry.Config{
		Enabled:        cfg.OTel.Enabled,
		ServiceName:    "auth-service",
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

	// Initialize database connection (uses AuthDatabase config)
	var db *database.PostgresDB
	dbCfg := &database.PostgresConfig{
		Host:            cfg.AuthDatabase.Host,
		Port:            cfg.AuthDatabase.Port,
		User:            cfg.AuthDatabase.User,
		Password:        cfg.AuthDatabase.Password,
		Database:        cfg.AuthDatabase.DBName,
		SSLMode:         cfg.AuthDatabase.SSLMode,
		MaxConns:        10, // Optimized: auth has low DB usage
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

	// Initialize repositories
	userRepo := repository.NewPostgresUserRepository(db.Pool())
	sessionRepo := repository.NewPostgresSessionRepository(db.Pool())
	tenantRepo := repository.NewPostgresTenantRepository(db.Pool())

	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if cfg.IsDevelopment() {
			jwtSecret = "dev-only-secret-key-do-not-use-in-production"
			appLog.Warn("JWT_SECRET not set, using dev-only default (NEVER use in production)")
		} else {
			appLog.Fatal("JWT_SECRET environment variable is required in production")
		}
	}

	// Build dependency injection container
	container := di.NewContainer(&di.ContainerConfig{
		DB:          db,
		UserRepo:    userRepo,
		SessionRepo: sessionRepo,
		TenantRepo:  tenantRepo,
		ServiceConfig: &service.AuthServiceConfig{
			JWTSecret:          jwtSecret,
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
			BcryptCost:         12, // Per P3-02 requirement
		},
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
		router.Use(telemetry.TracingMiddleware("auth-service"))
		router.Use(telemetry.TraceHeaderMiddleware())
	}

	// Health check endpoints
	router.GET("/health", container.HealthHandler.Health)
	router.GET("/ready", container.HealthHandler.Ready)

	// API routes
	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			// Public endpoints
			auth.POST("/register", container.AuthHandler.Register)
			auth.POST("/login", container.AuthHandler.Login)
			auth.POST("/refresh", container.AuthHandler.RefreshToken)
			auth.POST("/logout", container.AuthHandler.Logout)

			// Internal endpoint for token validation (used by other services)
			auth.POST("/validate", container.AuthHandler.ValidateToken)

			// Protected endpoints (require authentication)
			protected := auth.Group("")
			protected.Use(authMiddleware(container.AuthService))
			{
				protected.GET("/me", container.AuthHandler.Me)
				protected.PUT("/me", container.AuthHandler.UpdateMe)
				protected.POST("/logout-all", container.AuthHandler.LogoutAll)
			}

			// Internal endpoints for service-to-service communication
			// These endpoints are used by payment-service to manage Stripe Customer IDs
			internal := auth.Group("/users")
			{
				internal.GET("/:id/stripe-customer", container.AuthHandler.GetStripeCustomerID)
				internal.PUT("/:id/stripe-customer", container.AuthHandler.UpdateStripeCustomerID)
			}
		}

		// Tenant management routes (Admin/Super Admin only)
		tenants := v1.Group("/tenants")
		tenants.Use(authMiddleware(container.AuthService))
		tenants.Use(adminOnlyMiddleware())
		{
			tenants.POST("", container.TenantHandler.Create)
			tenants.GET("", container.TenantHandler.List)
			tenants.GET("/:id", container.TenantHandler.GetByID)
			tenants.GET("/slug/:slug", container.TenantHandler.GetBySlug)
			tenants.PUT("/:id", container.TenantHandler.Update)
			tenants.DELETE("/:id", container.TenantHandler.Delete)
		}
	}

	// Create HTTP server
	port := cfg.Server.Port
	if port == 0 {
		port = 8081 // Default port for auth-service
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
		appLog.Info(fmt.Sprintf("Auth Service listening on %s", addr))
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

// authMiddleware validates JWT token and sets user claims in context
func authMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_TOKEN",
					"message": "Authorization header is required",
				},
			})
			return
		}

		// Extract token from "Bearer <token>"
		const bearerPrefix = "Bearer "
		if len(authHeader) <= len(bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid authorization header format",
				},
			})
			return
		}
		token := authHeader[len(bearerPrefix):]

		claims, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid or expired token",
				},
			})
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", string(claims.Role))
		c.Next()
	}
}

// adminOnlyMiddleware restricts access to admin and super_admin roles only
func adminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User role not found in context",
				},
			})
			return
		}

		roleStr := role.(string)
		if roleStr != "admin" && roleStr != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Only admin or super_admin can access this resource",
				},
			})
			return
		}

		c.Next()
	}
}
