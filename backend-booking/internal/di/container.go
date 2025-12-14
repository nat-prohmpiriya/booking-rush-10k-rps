package di

import (
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

// Container holds all dependencies for the booking service
type Container struct {
	// Infrastructure
	DB    *database.PostgresDB
	Redis *redis.Client

	// Repositories
	BookingRepo     repository.BookingRepository
	ReservationRepo repository.ReservationRepository
	QueueRepo       repository.QueueRepository

	// Publishers
	EventPublisher service.EventPublisher

	// Services
	BookingService service.BookingService
	QueueService   service.QueueService
	SagaService    service.SagaService

	// Handlers
	HealthHandler  *handler.HealthHandler
	BookingHandler *handler.BookingHandler
	QueueHandler   *handler.QueueHandler
	AdminHandler   *handler.AdminHandler
	SagaHandler    *handler.SagaHandler
}

// ContainerConfig contains configuration for building the container
type ContainerConfig struct {
	DB                 *database.PostgresDB
	Redis              *redis.Client
	BookingRepo        repository.BookingRepository
	ReservationRepo    repository.ReservationRepository
	QueueRepo          repository.QueueRepository
	EventPublisher     service.EventPublisher
	ServiceConfig      *service.BookingServiceConfig
	QueueServiceConfig *service.QueueServiceConfig
	TicketServiceURL   string // URL of ticket service for zone sync
	SagaProducer       saga.SagaProducer
	SagaStore          pkgsaga.Store
	SagaServiceConfig  *service.SagaServiceConfig
	UseSagaForBooking  bool // Enable saga-based booking (default: false)
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *ContainerConfig) *Container {
	c := &Container{
		DB:              cfg.DB,
		Redis:           cfg.Redis,
		BookingRepo:     cfg.BookingRepo,
		ReservationRepo: cfg.ReservationRepo,
		QueueRepo:       cfg.QueueRepo,
		EventPublisher:  cfg.EventPublisher,
	}

	// Initialize zone syncer for auto-sync on ZONE_NOT_FOUND
	var zoneSyncer service.ZoneSyncer
	if cfg.TicketServiceURL != "" {
		zoneFetcher := service.NewHTTPZoneFetcher(cfg.TicketServiceURL)
		zoneSyncer = service.NewZoneSyncer(zoneFetcher, c.ReservationRepo)
	}

	// Initialize services
	c.BookingService = service.NewBookingService(
		c.BookingRepo,
		c.ReservationRepo,
		c.EventPublisher,
		zoneSyncer,
		cfg.ServiceConfig,
	)

	c.QueueService = service.NewQueueService(
		c.QueueRepo,
		cfg.QueueServiceConfig,
	)

	// Initialize saga service (optional - depends on Kafka availability)
	if cfg.SagaProducer != nil && cfg.SagaStore != nil {
		c.SagaService = service.NewKafkaSagaService(cfg.SagaProducer, cfg.SagaStore, cfg.SagaServiceConfig)
	} else {
		c.SagaService = service.NewNoOpSagaService()
	}

	// Initialize handlers
	c.HealthHandler = handler.NewHealthHandler(c.DB, c.Redis)

	// Use saga-based booking handler if enabled
	if cfg.UseSagaForBooking && c.SagaService != nil {
		c.BookingHandler = handler.NewBookingHandlerWithSaga(c.BookingService, c.SagaService, &handler.BookingHandlerConfig{
			UseSaga:     true,
			SagaTimeout: 30 * time.Second,
		})
	} else {
		c.BookingHandler = handler.NewBookingHandler(c.BookingService)
	}

	c.QueueHandler = handler.NewQueueHandler(c.QueueService)
	c.AdminHandler = handler.NewAdminHandler(c.Redis)
	c.SagaHandler = handler.NewSagaHandler(c.SagaService)

	return c
}
