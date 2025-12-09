package repository

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/domain"
)

// BookingRepository defines the interface for booking data access
type BookingRepository interface {
	// Create creates a new booking record in the database
	Create(ctx context.Context, booking *domain.Booking) error

	// GetByID retrieves a booking by its ID
	GetByID(ctx context.Context, id string) (*domain.Booking, error)

	// GetByUserID retrieves all bookings for a user
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Booking, error)

	// Update updates an existing booking
	Update(ctx context.Context, booking *domain.Booking) error

	// UpdateStatus updates only the status of a booking
	UpdateStatus(ctx context.Context, id string, status domain.BookingStatus) error

	// Delete deletes a booking by its ID
	Delete(ctx context.Context, id string) error

	// Confirm confirms a booking with payment info
	Confirm(ctx context.Context, id, paymentID string) error

	// Cancel cancels a booking
	Cancel(ctx context.Context, id string) error

	// GetExpiredReservations gets all expired reservations
	GetExpiredReservations(ctx context.Context, limit int) ([]*domain.Booking, error)

	// MarkAsExpired marks a booking as expired
	MarkAsExpired(ctx context.Context, id string) error

	// GetByIdempotencyKey retrieves a booking by idempotency key
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error)

	// CountByUserAndEvent counts bookings for a user on an event
	CountByUserAndEvent(ctx context.Context, userID, eventID string) (int, error)
}
