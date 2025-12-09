package domain

import "errors"

// Domain errors
var (
	// Booking errors
	ErrBookingNotFound      = errors.New("booking not found")
	ErrBookingExpired       = errors.New("booking has expired")
	ErrBookingAlreadyExists = errors.New("booking already exists")
	ErrInvalidBookingStatus = errors.New("invalid booking status")

	// Reservation errors
	ErrReservationNotFound     = errors.New("reservation not found")
	ErrReservationExpired      = errors.New("reservation has expired")
	ErrAlreadyConfirmed        = errors.New("reservation already confirmed")
	ErrAlreadyReleased         = errors.New("reservation already released")
	ErrInvalidUserID           = errors.New("invalid user id")
	ErrInvalidBookingID        = errors.New("invalid booking id")

	// Availability errors
	ErrInsufficientSeats = errors.New("insufficient seats available")
	ErrMaxTicketsExceeded = errors.New("maximum tickets per user exceeded")

	// Zone errors
	ErrZoneNotFound = errors.New("zone not found")

	// Event errors
	ErrEventNotFound = errors.New("event not found")
)
