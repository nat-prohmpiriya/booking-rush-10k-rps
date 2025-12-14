package repository

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
)

// EventRepository defines the interface for event data access
type EventRepository interface {
	// Create creates a new event
	Create(ctx context.Context, event *domain.Event) error
	// GetByID retrieves an event by ID
	GetByID(ctx context.Context, id string) (*domain.Event, error)
	// GetBySlug retrieves an event by slug
	GetBySlug(ctx context.Context, slug string) (*domain.Event, error)
	// GetByTenantID retrieves events by tenant ID
	GetByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, error)
	// Update updates an event
	Update(ctx context.Context, event *domain.Event) error
	// Delete soft deletes an event by ID
	Delete(ctx context.Context, id string) error
	// ListPublished lists all published events with pagination
	ListPublished(ctx context.Context, limit, offset int) ([]*domain.Event, int, error)
	// List lists events with filters and pagination
	List(ctx context.Context, filter *EventFilter, limit, offset int) ([]*domain.Event, int, error)
	// SlugExists checks if a slug already exists
	SlugExists(ctx context.Context, slug string) (bool, error)
}

// EventFilter contains filter options for listing events
type EventFilter struct {
	Status      string
	TenantID    string
	OrganizerID string
	CategoryID  string
	City        string
	Search      string
}

// VenueRepository defines the interface for venue data access
type VenueRepository interface {
	// Create creates a new venue
	Create(ctx context.Context, venue *domain.Venue) error
	// GetByID retrieves a venue by ID
	GetByID(ctx context.Context, id string) (*domain.Venue, error)
	// GetByTenantID retrieves venues by tenant ID
	GetByTenantID(ctx context.Context, tenantID string) ([]*domain.Venue, error)
	// Update updates a venue
	Update(ctx context.Context, venue *domain.Venue) error
	// Delete deletes a venue by ID
	Delete(ctx context.Context, id string) error
}

// ZoneRepository defines the interface for zone data access
type ZoneRepository interface {
	// Create creates a new zone
	Create(ctx context.Context, zone *domain.Zone) error
	// GetByID retrieves a zone by ID
	GetByID(ctx context.Context, id string) (*domain.Zone, error)
	// GetByVenueID retrieves zones by venue ID
	GetByVenueID(ctx context.Context, venueID string) ([]*domain.Zone, error)
	// Update updates a zone
	Update(ctx context.Context, zone *domain.Zone) error
	// Delete deletes a zone by ID
	Delete(ctx context.Context, id string) error
}

// SeatRepository defines the interface for seat data access
type SeatRepository interface {
	// Create creates a new seat
	Create(ctx context.Context, seat *domain.Seat) error
	// CreateBatch creates multiple seats at once
	CreateBatch(ctx context.Context, seats []*domain.Seat) error
	// GetByID retrieves a seat by ID
	GetByID(ctx context.Context, id string) (*domain.Seat, error)
	// GetByZoneID retrieves seats by zone ID
	GetByZoneID(ctx context.Context, zoneID string) ([]*domain.Seat, error)
	// GetAvailableByZoneID retrieves available seats by zone ID
	GetAvailableByZoneID(ctx context.Context, zoneID string) ([]*domain.Seat, error)
	// UpdateStatus updates a seat's status
	UpdateStatus(ctx context.Context, id string, status string) error
	// UpdateStatusBatch updates multiple seats' status
	UpdateStatusBatch(ctx context.Context, ids []string, status string) error
}

// TicketTypeRepository defines the interface for ticket type data access
type TicketTypeRepository interface {
	// Create creates a new ticket type
	Create(ctx context.Context, ticketType *domain.TicketType) error
	// GetByID retrieves a ticket type by ID
	GetByID(ctx context.Context, id string) (*domain.TicketType, error)
	// GetByEventID retrieves ticket types by event ID
	GetByEventID(ctx context.Context, eventID string) ([]*domain.TicketType, error)
	// GetAvailableByEventID retrieves available ticket types by event ID
	GetAvailableByEventID(ctx context.Context, eventID string) ([]*domain.TicketType, error)
	// Update updates a ticket type
	Update(ctx context.Context, ticketType *domain.TicketType) error
	// UpdateSoldQuantity updates the sold quantity for a ticket type
	UpdateSoldQuantity(ctx context.Context, id string, soldQuantity int) error
	// Delete deletes a ticket type by ID
	Delete(ctx context.Context, id string) error
}

// ShowRepository defines the interface for show data access
type ShowRepository interface {
	// Create creates a new show
	Create(ctx context.Context, show *domain.Show) error
	// GetByID retrieves a show by ID
	GetByID(ctx context.Context, id string) (*domain.Show, error)
	// GetByEventID retrieves shows by event ID with pagination
	GetByEventID(ctx context.Context, eventID string, limit, offset int) ([]*domain.Show, int, error)
	// Update updates a show
	Update(ctx context.Context, show *domain.Show) error
	// Delete soft deletes a show by ID
	Delete(ctx context.Context, id string) error
}

// ShowZoneRepository defines the interface for show zone data access
type ShowZoneRepository interface {
	// Create creates a new show zone
	Create(ctx context.Context, zone *domain.ShowZone) error
	// GetByID retrieves a show zone by ID
	GetByID(ctx context.Context, id string) (*domain.ShowZone, error)
	// GetByShowID retrieves all zones for a show with pagination and optional is_active filter
	GetByShowID(ctx context.Context, showID string, isActive *bool, limit, offset int) ([]*domain.ShowZone, int, error)
	// Update updates a show zone
	Update(ctx context.Context, zone *domain.ShowZone) error
	// Delete soft deletes a show zone by ID
	Delete(ctx context.Context, id string) error
	// UpdateAvailableSeats updates the available seats count
	UpdateAvailableSeats(ctx context.Context, id string, availableSeats int) error
	// ListActive retrieves all active zones (for inventory sync)
	ListActive(ctx context.Context) ([]*domain.ShowZone, error)
}
