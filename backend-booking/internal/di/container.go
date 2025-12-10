package di

import (
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// Container holds all dependencies for the booking service
type Container struct {
	// Infrastructure
	DB    *database.PostgresDB
	Redis *redis.Client

	// Repositories
	BookingRepo     repository.BookingRepository
	ReservationRepo repository.ReservationRepository

	// Publishers
	EventPublisher service.EventPublisher

	// Services
	BookingService service.BookingService

	// Handlers
	HealthHandler  *handler.HealthHandler
	BookingHandler *handler.BookingHandler
}

// ContainerConfig contains configuration for building the container
type ContainerConfig struct {
	DB              *database.PostgresDB
	Redis           *redis.Client
	BookingRepo     repository.BookingRepository
	ReservationRepo repository.ReservationRepository
	EventPublisher  service.EventPublisher
	ServiceConfig   *service.BookingServiceConfig
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *ContainerConfig) *Container {
	c := &Container{
		DB:              cfg.DB,
		Redis:           cfg.Redis,
		BookingRepo:     cfg.BookingRepo,
		ReservationRepo: cfg.ReservationRepo,
		EventPublisher:  cfg.EventPublisher,
	}

	// Initialize services
	c.BookingService = service.NewBookingService(
		c.BookingRepo,
		c.ReservationRepo,
		c.EventPublisher,
		cfg.ServiceConfig,
	)

	// Initialize handlers
	c.HealthHandler = handler.NewHealthHandler(c.DB, c.Redis)
	c.BookingHandler = handler.NewBookingHandler(c.BookingService)

	return c
}
