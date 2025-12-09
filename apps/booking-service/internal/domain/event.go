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
