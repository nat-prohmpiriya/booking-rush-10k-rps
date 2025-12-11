package domain

import (
	"time"
)

// Event represents an event entity
type Event struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	VenueID     string    `json:"venue_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Queue settings
	MaxConcurrentBookings int `json:"max_concurrent_bookings"` // Max users with active queue pass (default: 500)
	QueuePassTTLMinutes   int `json:"queue_pass_ttl_minutes"`  // Queue pass TTL in minutes (default: 5)
}

// Default queue settings
const (
	DefaultMaxConcurrentBookings = 500
	DefaultQueuePassTTLMinutes   = 5
)

// GetMaxConcurrentBookings returns max concurrent bookings with default fallback
func (e *Event) GetMaxConcurrentBookings() int {
	if e.MaxConcurrentBookings <= 0 {
		return DefaultMaxConcurrentBookings
	}
	return e.MaxConcurrentBookings
}

// GetQueuePassTTLMinutes returns queue pass TTL with default fallback
func (e *Event) GetQueuePassTTLMinutes() int {
	if e.QueuePassTTLMinutes <= 0 {
		return DefaultQueuePassTTLMinutes
	}
	return e.QueuePassTTLMinutes
}

// Zone represents a seating zone in an event
type Zone struct {
	ID            string  `json:"id"`
	EventID       string  `json:"event_id"`
	Name          string  `json:"name"`
	TotalSeats    int     `json:"total_seats"`
	AvailableSeats int    `json:"available_seats"`
	Price         float64 `json:"price"`
}
