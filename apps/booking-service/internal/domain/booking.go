package domain

import (
	"time"
)

// BookingStatus represents the status of a booking
type BookingStatus string

const (
	BookingStatusReserved  BookingStatus = "reserved"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusCancelled BookingStatus = "cancelled"
	BookingStatusExpired   BookingStatus = "expired"
)

// Booking represents a booking entity
type Booking struct {
	ID          string        `json:"id"`
	UserID      string        `json:"user_id"`
	EventID     string        `json:"event_id"`
	ZoneID      string        `json:"zone_id"`
	Quantity    int           `json:"quantity"`
	Status      BookingStatus `json:"status"`
	TotalPrice  float64       `json:"total_price"`
	PaymentID   string        `json:"payment_id,omitempty"`
	ReservedAt  time.Time     `json:"reserved_at"`
	ConfirmedAt *time.Time    `json:"confirmed_at,omitempty"`
	ExpiresAt   time.Time     `json:"expires_at"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// IsExpired checks if the booking has expired
func (b *Booking) IsExpired() bool {
	return time.Now().After(b.ExpiresAt)
}

// CanConfirm checks if the booking can be confirmed
func (b *Booking) CanConfirm() bool {
	return b.Status == BookingStatusReserved && !b.IsExpired()
}

// CanCancel checks if the booking can be cancelled
func (b *Booking) CanCancel() bool {
	return b.Status == BookingStatusReserved
}
