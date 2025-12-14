package di

import (
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/gateway"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// Container holds all dependencies for the payment service
type Container struct {
	// Infrastructure
	DB    *database.PostgresDB
	Redis *redis.Client

	// Gateways
	PaymentGateway gateway.PaymentGateway

	// Repositories
	PaymentRepo repository.PaymentRepository

	// Services
	PaymentService service.PaymentService

	// Handlers
	HealthHandler  *handler.HealthHandler
	PaymentHandler *handler.PaymentHandler
	WebhookHandler *handler.WebhookHandler
}

// ContainerConfig contains configuration for building the container
type ContainerConfig struct {
	DB                   *database.PostgresDB
	Redis                *redis.Client
	PaymentRepo          repository.PaymentRepository
	PaymentGateway       gateway.PaymentGateway
	KafkaProducer        *kafka.Producer
	ServiceConfig        *service.PaymentServiceConfig
	StripeWebhookSecret  string
	AuthServiceURL       string
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *ContainerConfig) *Container {
	c := &Container{
		DB:             cfg.DB,
		Redis:          cfg.Redis,
		PaymentRepo:    cfg.PaymentRepo,
		PaymentGateway: cfg.PaymentGateway,
	}

	// Initialize handlers
	c.HealthHandler = handler.NewHealthHandler(c.DB, c.Redis)

	// Initialize PaymentService if repository and gateway are provided
	if c.PaymentRepo != nil && c.PaymentGateway != nil {
		c.PaymentService = service.NewPaymentService(c.PaymentRepo, c.PaymentGateway, cfg.ServiceConfig)
		c.PaymentHandler = handler.NewPaymentHandler(c.PaymentService, c.PaymentGateway, cfg.AuthServiceURL)

		// Initialize WebhookHandler if webhook secret is provided
		if cfg.StripeWebhookSecret != "" {
			c.WebhookHandler = handler.NewWebhookHandler(c.PaymentService, cfg.StripeWebhookSecret, cfg.KafkaProducer)
		}
	}

	return c
}
