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
	ErrReservationNotFound = errors.New("reservation not found")
	ErrReservationExpired  = errors.New("reservation has expired")
	ErrAlreadyConfirmed    = errors.New("reservation already confirmed")
	ErrAlreadyReleased     = errors.New("reservation already released")

	// Validation errors
	ErrInvalidUserID     = errors.New("invalid user id")
	ErrInvalidBookingID  = errors.New("invalid booking id")
	ErrInvalidEventID    = errors.New("invalid event id")
	ErrInvalidShowID     = errors.New("invalid show id")
	ErrInvalidZoneID     = errors.New("invalid zone id")
	ErrInvalidQuantity   = errors.New("quantity must be greater than zero")
	ErrInvalidTotalPrice = errors.New("total price cannot be negative")
	ErrInvalidUnitPrice  = errors.New("unit price cannot be negative")

	// Availability errors
	ErrInsufficientSeats  = errors.New("insufficient seats available")
	ErrMaxTicketsExceeded = errors.New("maximum tickets per user exceeded")

	// Zone errors
	ErrZoneNotFound = errors.New("zone not found")

	// Event errors
	ErrEventNotFound = errors.New("event not found")

	// Queue errors
	ErrQueueNotOpen          = errors.New("queue is not open for this event")
	ErrAlreadyInQueue        = errors.New("user is already in queue")
	ErrNotInQueue            = errors.New("user is not in queue")
	ErrQueueFull             = errors.New("queue is full")
	ErrInvalidQueueToken     = errors.New("invalid queue token")
	ErrQueuePassRequired     = errors.New("queue pass is required")
	ErrInvalidQueuePass      = errors.New("invalid queue pass")
	ErrQueuePassExpired      = errors.New("queue pass has expired or already used")
	ErrQueuePassUserMismatch = errors.New("queue pass does not belong to this user")
	ErrQueuePassEventMismatch = errors.New("queue pass is for a different event")
)

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrBookingNotFound) ||
		errors.Is(err, ErrReservationNotFound) ||
		errors.Is(err, ErrZoneNotFound) ||
		errors.Is(err, ErrEventNotFound)
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidUserID) ||
		errors.Is(err, ErrInvalidBookingID) ||
		errors.Is(err, ErrInvalidEventID) ||
		errors.Is(err, ErrInvalidShowID) ||
		errors.Is(err, ErrInvalidZoneID) ||
		errors.Is(err, ErrInvalidQuantity) ||
		errors.Is(err, ErrInvalidTotalPrice) ||
		errors.Is(err, ErrInvalidUnitPrice) ||
		errors.Is(err, ErrInvalidBookingStatus)
}

// IsConflictError checks if the error is a conflict error
func IsConflictError(err error) bool {
	return errors.Is(err, ErrAlreadyConfirmed) ||
		errors.Is(err, ErrAlreadyReleased) ||
		errors.Is(err, ErrBookingAlreadyExists) ||
		errors.Is(err, ErrInsufficientSeats) ||
		errors.Is(err, ErrMaxTicketsExceeded)
}

// IsExpiredError checks if the error is an expiration error
func IsExpiredError(err error) bool {
	return errors.Is(err, ErrBookingExpired) ||
		errors.Is(err, ErrReservationExpired)
}
