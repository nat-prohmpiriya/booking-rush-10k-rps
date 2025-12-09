package di

import (
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// Container holds all dependencies for the payment service
type Container struct {
	// Infrastructure
	DB    *database.PostgresDB
	Redis *redis.Client

	// Repositories
	PaymentRepo repository.PaymentRepository

	// Services
	PaymentService service.PaymentService

	// Handlers
	HealthHandler *handler.HealthHandler
}

// ContainerConfig contains configuration for building the container
type ContainerConfig struct {
	DB            *database.PostgresDB
	Redis         *redis.Client
	PaymentRepo   repository.PaymentRepository
	ServiceConfig *service.PaymentServiceConfig
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *ContainerConfig) *Container {
	c := &Container{
		DB:          cfg.DB,
		Redis:       cfg.Redis,
		PaymentRepo: cfg.PaymentRepo,
	}

	// Initialize handlers
	c.HealthHandler = handler.NewHealthHandler(c.DB, c.Redis)

	// Note: PaymentService will be initialized when we implement it
	// c.PaymentService = service.NewPaymentService(...)

	return c
}
