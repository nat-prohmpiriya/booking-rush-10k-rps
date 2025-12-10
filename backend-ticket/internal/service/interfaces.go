package service

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
)

// EventService defines the interface for event business logic
type EventService interface {
	// CreateEvent creates a new event
	CreateEvent(ctx context.Context, req *dto.CreateEventRequest) (*domain.Event, error)
	// GetEventByID retrieves an event by ID
	GetEventByID(ctx context.Context, id string) (*domain.Event, error)
	// GetEventBySlug retrieves an event by slug
	GetEventBySlug(ctx context.Context, slug string) (*domain.Event, error)
	// ListEvents lists events with filters and pagination
	ListEvents(ctx context.Context, filter *dto.EventListFilter) ([]*domain.Event, int, error)
	// UpdateEvent updates an event
	UpdateEvent(ctx context.Context, id string, req *dto.UpdateEventRequest) (*domain.Event, error)
	// DeleteEvent soft deletes an event
	DeleteEvent(ctx context.Context, id string) error
	// PublishEvent publishes an event
	PublishEvent(ctx context.Context, id string) (*domain.Event, error)
}

// TicketService defines the interface for ticket business logic
type TicketService interface {
	// CreateTicketType creates a new ticket type for an event
	CreateTicketType(ctx context.Context, req *dto.CreateTicketTypeRequest) (*domain.TicketType, error)
	// GetTicketType retrieves a ticket type by ID
	GetTicketType(ctx context.Context, id string) (*domain.TicketType, error)
	// GetTicketTypesByEvent retrieves ticket types by event ID
	GetTicketTypesByEvent(ctx context.Context, eventID string) ([]*domain.TicketType, error)
	// GetAvailableTicketTypes retrieves available ticket types by event ID
	GetAvailableTicketTypes(ctx context.Context, eventID string) ([]*domain.TicketType, error)
	// UpdateTicketType updates a ticket type
	UpdateTicketType(ctx context.Context, id string, req *dto.UpdateTicketTypeRequest) (*domain.TicketType, error)
	// DeleteTicketType deletes a ticket type
	DeleteTicketType(ctx context.Context, id string) error
	// CheckAvailability checks ticket availability for an event
	CheckAvailability(ctx context.Context, eventID string, ticketTypeID string, quantity int) (*dto.AvailabilityResponse, error)
}

// VenueService defines the interface for venue business logic
type VenueService interface {
	// CreateVenue creates a new venue
	CreateVenue(ctx context.Context, req *dto.CreateVenueRequest) (*domain.Venue, error)
	// GetVenue retrieves a venue by ID
	GetVenue(ctx context.Context, id string) (*domain.Venue, error)
	// GetVenuesByTenant retrieves venues by tenant ID
	GetVenuesByTenant(ctx context.Context, tenantID string) ([]*domain.Venue, error)
	// UpdateVenue updates a venue
	UpdateVenue(ctx context.Context, id string, req *dto.UpdateVenueRequest) (*domain.Venue, error)
	// DeleteVenue deletes a venue
	DeleteVenue(ctx context.Context, id string) error
}

// ShowService defines the interface for show business logic
type ShowService interface {
	// CreateShow creates a new show for an event
	CreateShow(ctx context.Context, req *dto.CreateShowRequest) (*domain.Show, error)
	// GetShowByID retrieves a show by ID
	GetShowByID(ctx context.Context, id string) (*domain.Show, error)
	// ListShowsByEvent lists shows for an event
	ListShowsByEvent(ctx context.Context, eventID string, filter *dto.ShowListFilter) ([]*domain.Show, int, error)
	// UpdateShow updates a show
	UpdateShow(ctx context.Context, id string, req *dto.UpdateShowRequest) (*domain.Show, error)
	// DeleteShow soft deletes a show
	DeleteShow(ctx context.Context, id string) error
}

// ShowZoneService defines the interface for show zone business logic
type ShowZoneService interface {
	// CreateShowZone creates a new zone for a show
	CreateShowZone(ctx context.Context, req *dto.CreateShowZoneRequest) (*domain.ShowZone, error)
	// GetShowZoneByID retrieves a show zone by ID
	GetShowZoneByID(ctx context.Context, id string) (*domain.ShowZone, error)
	// ListZonesByShow lists zones for a show
	ListZonesByShow(ctx context.Context, showID string, filter *dto.ShowZoneListFilter) ([]*domain.ShowZone, int, error)
	// UpdateShowZone updates a show zone
	UpdateShowZone(ctx context.Context, id string, req *dto.UpdateShowZoneRequest) (*domain.ShowZone, error)
	// DeleteShowZone soft deletes a show zone
	DeleteShowZone(ctx context.Context, id string) error
}
