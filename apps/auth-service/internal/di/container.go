package di

import (
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
)

// Container holds all dependencies for the auth service
type Container struct {
	// Infrastructure
	DB *database.PostgresDB

	// Repositories
	UserRepo    repository.UserRepository
	SessionRepo repository.SessionRepository

	// Services
	AuthService service.AuthService

	// Handlers
	HealthHandler *handler.HealthHandler
	AuthHandler   *handler.AuthHandler
}

// ContainerConfig contains configuration for building the container
type ContainerConfig struct {
	DB            *database.PostgresDB
	UserRepo      repository.UserRepository
	SessionRepo   repository.SessionRepository
	ServiceConfig *service.AuthServiceConfig
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *ContainerConfig) *Container {
	c := &Container{
		DB:          cfg.DB,
		UserRepo:    cfg.UserRepo,
		SessionRepo: cfg.SessionRepo,
	}

	// Initialize services
	c.AuthService = service.NewAuthService(
		c.UserRepo,
		c.SessionRepo,
		cfg.ServiceConfig,
	)

	// Initialize handlers
	c.HealthHandler = handler.NewHealthHandler(c.DB)
	c.AuthHandler = handler.NewAuthHandler(c.AuthService)

	return c
}
