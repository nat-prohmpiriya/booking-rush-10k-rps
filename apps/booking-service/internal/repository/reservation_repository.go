package repository

import (
	"context"
)

// ReserveResult represents the result of a seat reservation
type ReserveResult struct {
	Success          bool
	BookingID        string
	AvailableSeats   int64
	UserReserved     int64
	ErrorCode        string
	ErrorMessage     string
}

// ConfirmResult represents the result of confirming a booking
type ConfirmResult struct {
	Success      bool
	Status       string
	ConfirmedAt  string
	ErrorCode    string
	ErrorMessage string
}

// ReleaseResult represents the result of releasing a reservation
type ReleaseResult struct {
	Success         bool
	AvailableSeats  int64
	UserReserved    int64
	ErrorCode       string
	ErrorMessage    string
}

// ReservationRepository defines the interface for Redis-based reservation operations
type ReservationRepository interface {
	// ReserveSeats atomically reserves seats using Lua script
	ReserveSeats(ctx context.Context, params ReserveParams) (*ReserveResult, error)

	// ConfirmBooking confirms a reservation and makes it permanent
	ConfirmBooking(ctx context.Context, bookingID, userID, paymentID string) (*ConfirmResult, error)

	// ReleaseSeats releases reserved seats back to inventory
	ReleaseSeats(ctx context.Context, bookingID, userID string) (*ReleaseResult, error)

	// GetZoneAvailability gets the current available seats for a zone
	GetZoneAvailability(ctx context.Context, zoneID string) (int64, error)

	// SetZoneAvailability sets the available seats for a zone (for initialization)
	SetZoneAvailability(ctx context.Context, zoneID string, seats int64) error
}

// ReserveParams contains parameters for seat reservation
type ReserveParams struct {
	ZoneID      string
	UserID      string
	EventID     string
	Quantity    int
	MaxPerUser  int
	TTLSeconds  int
	Price       float64
}
