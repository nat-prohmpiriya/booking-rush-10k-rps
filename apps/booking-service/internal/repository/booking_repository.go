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
}
