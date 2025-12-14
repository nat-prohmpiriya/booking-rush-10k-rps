package di

import (
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// Container holds all dependencies for the ticket service
type Container struct {
	// Infrastructure
	DB    *database.PostgresDB
	Redis *redis.Client

	// Repositories
	EventRepo    repository.EventRepository
	VenueRepo    repository.VenueRepository
	ShowRepo     repository.ShowRepository
	ShowZoneRepo repository.ShowZoneRepository
	// SeatRepo       repository.SeatRepository
	// TicketTypeRepo repository.TicketTypeRepository

	// Services
	ZoneSyncer      service.ZoneSyncer
	EventService    service.EventService
	ShowService     service.ShowService
	ShowZoneService service.ShowZoneService
	// TicketService service.TicketService
	// VenueService  service.VenueService

	// Handlers
	HealthHandler   *handler.HealthHandler
	EventHandler    *handler.EventHandler
	ShowHandler     *handler.ShowHandler
	ShowZoneHandler *handler.ShowZoneHandler
	// TicketHandler *handler.TicketHandler
	// VenueHandler  *handler.VenueHandler
}

// ContainerConfig contains configuration for building the container
type ContainerConfig struct {
	DB    *database.PostgresDB
	Redis *redis.Client
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *ContainerConfig) *Container {
	c := &Container{
		DB:    cfg.DB,
		Redis: cfg.Redis,
	}

	// Initialize repositories
	pgEventRepo := repository.NewPostgresEventRepository(c.DB.Pool())

	// Wrap with cache if Redis is available
	if c.Redis != nil {
		c.EventRepo = repository.NewCachedEventRepository(pgEventRepo, c.Redis)
	} else {
		c.EventRepo = pgEventRepo
	}
	c.VenueRepo = repository.NewPostgresVenueRepository(c.DB.Pool())
	c.ShowRepo = repository.NewPostgresShowRepository(c.DB.Pool())
	c.ShowZoneRepo = repository.NewPostgresShowZoneRepository(c.DB.Pool())
	// c.SeatRepo = repository.NewPostgresSeatRepository(c.DB.Pool())
	// c.TicketTypeRepo = repository.NewPostgresTicketTypeRepository(c.DB.Pool())

	// Initialize services
	c.ZoneSyncer = service.NewZoneSyncer(c.ShowZoneRepo, c.ShowRepo, c.Redis)
	c.EventService = service.NewEventService(c.EventRepo)
	c.ShowService = service.NewShowService(c.ShowRepo, c.EventRepo, c.ZoneSyncer)
	c.ShowZoneService = service.NewShowZoneService(c.ShowZoneRepo, c.ShowRepo, c.ZoneSyncer)
	// c.TicketService = service.NewTicketService(c.TicketTypeRepo, c.EventRepo)
	// c.VenueService = service.NewVenueService(c.VenueRepo, c.ZoneRepo, c.SeatRepo)

	// Initialize handlers
	c.HealthHandler = handler.NewHealthHandler(c.DB)
	c.EventHandler = handler.NewEventHandler(c.EventService, c.ShowService)
	c.ShowHandler = handler.NewShowHandler(c.ShowService, c.EventService)
	c.ShowZoneHandler = handler.NewShowZoneHandler(c.ShowZoneService, c.ShowService)
	// c.TicketHandler = handler.NewTicketHandler(c.TicketService)
	// c.VenueHandler = handler.NewVenueHandler(c.VenueService)

	return c
}
