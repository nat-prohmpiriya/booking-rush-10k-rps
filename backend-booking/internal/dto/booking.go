package dto

import (
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
)

// ReserveSeatsRequest represents request to reserve seats
type ReserveSeatsRequest struct {
	EventID        string  `json:"event_id" binding:"required"`
	ZoneID         string  `json:"zone_id" binding:"required"`
	ShowID         string  `json:"show_id,omitempty"`
	TenantID       string  `json:"tenant_id,omitempty"`
	Quantity       int     `json:"quantity" binding:"required,min=1,max=10"`
	UnitPrice      float64 `json:"unit_price,omitempty"`
	IdempotencyKey string  `json:"idempotency_key,omitempty"`
}

// ReserveSeatsResponse represents response after reserving seats
type ReserveSeatsResponse struct {
	BookingID  string    `json:"booking_id"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
	TotalPrice float64   `json:"total_price"`
}

// ConfirmBookingRequest represents request to confirm a booking
type ConfirmBookingRequest struct {
	PaymentID string `json:"payment_id,omitempty"`
}

// ConfirmBookingResponse represents response after confirming a booking
type ConfirmBookingResponse struct {
	BookingID        string    `json:"booking_id"`
	Status           string    `json:"status"`
	ConfirmedAt      time.Time `json:"confirmed_at"`
	ConfirmationCode string    `json:"confirmation_code,omitempty"`
}

// ReleaseBookingResponse represents response after releasing a booking
type ReleaseBookingResponse struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// BookingResponse represents a booking in API response
type BookingResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	EventID     string     `json:"event_id"`
	ZoneID      string     `json:"zone_id"`
	Quantity    int        `json:"quantity"`
	Status      string     `json:"status"`
	TotalPrice  float64    `json:"total_price"`
	PaymentID   string     `json:"payment_id,omitempty"`
	ReservedAt  time.Time  `json:"reserved_at"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
}

// UserBookingSummaryResponse represents user's booking summary for an event
type UserBookingSummaryResponse struct {
	UserID       string `json:"user_id"`
	EventID      string `json:"event_id"`
	BookedCount  int    `json:"booked_count"`   // Total tickets booked (confirmed + reserved)
	MaxAllowed   int    `json:"max_allowed"`    // Maximum allowed per user
	RemainingSlots int  `json:"remaining_slots"` // How many more can be booked
}

// FromDomain converts domain Booking to BookingResponse
func FromDomain(b *domain.Booking) *BookingResponse {
	return &BookingResponse{
		ID:          b.ID,
		UserID:      b.UserID,
		EventID:     b.EventID,
		ZoneID:      b.ZoneID,
		Quantity:    b.Quantity,
		Status:      string(b.Status),
		TotalPrice:  b.TotalPrice,
		PaymentID:   b.PaymentID,
		ReservedAt:  b.ReservedAt,
		ConfirmedAt: b.ConfirmedAt,
		ExpiresAt:   b.ExpiresAt,
	}
}
