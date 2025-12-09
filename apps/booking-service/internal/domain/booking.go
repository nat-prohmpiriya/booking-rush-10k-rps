package domain

import (
	"strings"
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

// IsValid checks if the status is a valid BookingStatus
func (s BookingStatus) IsValid() bool {
	switch s {
	case BookingStatusReserved, BookingStatusConfirmed, BookingStatusCancelled, BookingStatusExpired:
		return true
	}
	return false
}

// String returns the string representation of BookingStatus
func (s BookingStatus) String() string {
	return string(s)
}

// Booking represents a booking entity
type Booking struct {
	ID               string        `json:"id"`
	TenantID         string        `json:"tenant_id"`
	UserID           string        `json:"user_id"`
	EventID          string        `json:"event_id"`
	ShowID           string        `json:"show_id"`
	ZoneID           string        `json:"zone_id"`
	Quantity         int           `json:"quantity"`
	UnitPrice        float64       `json:"unit_price"`
	TotalPrice       float64       `json:"total_price"`
	Currency         string        `json:"currency"`
	Status           BookingStatus `json:"status"`
	StatusReason     string        `json:"status_reason,omitempty"`
	IdempotencyKey   string        `json:"idempotency_key,omitempty"`
	PaymentID        string        `json:"payment_id,omitempty"`
	ConfirmationCode string        `json:"confirmation_code,omitempty"`
	ReservedAt       time.Time     `json:"reserved_at"`
	ConfirmedAt      *time.Time    `json:"confirmed_at,omitempty"`
	CancelledAt      *time.Time    `json:"cancelled_at,omitempty"`
	ExpiresAt        time.Time     `json:"expires_at"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// Validate validates all booking fields
func (b *Booking) Validate() error {
	if err := b.ValidateID(); err != nil {
		return err
	}
	if err := b.ValidateUserID(); err != nil {
		return err
	}
	if err := b.ValidateEventID(); err != nil {
		return err
	}
	if err := b.ValidateZoneID(); err != nil {
		return err
	}
	if err := b.ValidateQuantity(); err != nil {
		return err
	}
	if err := b.ValidateStatus(); err != nil {
		return err
	}
	if err := b.ValidateTotalPrice(); err != nil {
		return err
	}
	return nil
}

// ValidateID validates the booking ID
func (b *Booking) ValidateID() error {
	if strings.TrimSpace(b.ID) == "" {
		return ErrInvalidBookingID
	}
	return nil
}

// ValidateUserID validates the user ID
func (b *Booking) ValidateUserID() error {
	if strings.TrimSpace(b.UserID) == "" {
		return ErrInvalidUserID
	}
	return nil
}

// ValidateEventID validates the event ID
func (b *Booking) ValidateEventID() error {
	if strings.TrimSpace(b.EventID) == "" {
		return ErrInvalidEventID
	}
	return nil
}

// ValidateZoneID validates the zone ID
func (b *Booking) ValidateZoneID() error {
	if strings.TrimSpace(b.ZoneID) == "" {
		return ErrInvalidZoneID
	}
	return nil
}

// ValidateQuantity validates the booking quantity
func (b *Booking) ValidateQuantity() error {
	if b.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	return nil
}

// ValidateStatus validates the booking status
func (b *Booking) ValidateStatus() error {
	if !b.Status.IsValid() {
		return ErrInvalidBookingStatus
	}
	return nil
}

// ValidateTotalPrice validates the total price
func (b *Booking) ValidateTotalPrice() error {
	if b.TotalPrice < 0 {
		return ErrInvalidTotalPrice
	}
	return nil
}

// IsExpired checks if the booking has expired
func (b *Booking) IsExpired() bool {
	return time.Now().After(b.ExpiresAt)
}

// IsExpiredAt checks if the booking is expired at a specific time
func (b *Booking) IsExpiredAt(t time.Time) bool {
	return t.After(b.ExpiresAt)
}

// CanConfirm checks if the booking can be confirmed
func (b *Booking) CanConfirm() bool {
	return b.Status == BookingStatusReserved && !b.IsExpired()
}

// CanCancel checks if the booking can be cancelled
func (b *Booking) CanCancel() bool {
	return b.Status == BookingStatusReserved
}

// IsReserved checks if the booking is in reserved status
func (b *Booking) IsReserved() bool {
	return b.Status == BookingStatusReserved
}

// IsConfirmed checks if the booking is in confirmed status
func (b *Booking) IsConfirmed() bool {
	return b.Status == BookingStatusConfirmed
}

// IsCancelled checks if the booking is in cancelled status
func (b *Booking) IsCancelled() bool {
	return b.Status == BookingStatusCancelled
}

// Confirm marks the booking as confirmed
func (b *Booking) Confirm(paymentID string) error {
	if !b.CanConfirm() {
		if b.IsExpired() {
			return ErrBookingExpired
		}
		return ErrAlreadyConfirmed
	}
	now := time.Now()
	b.Status = BookingStatusConfirmed
	b.PaymentID = paymentID
	b.ConfirmedAt = &now
	b.UpdatedAt = now
	return nil
}

// Cancel marks the booking as cancelled
func (b *Booking) Cancel() error {
	if !b.CanCancel() {
		if b.Status == BookingStatusConfirmed {
			return ErrAlreadyConfirmed
		}
		if b.Status == BookingStatusCancelled {
			return ErrAlreadyReleased
		}
		return ErrInvalidBookingStatus
	}
	b.Status = BookingStatusCancelled
	b.UpdatedAt = time.Now()
	return nil
}

// Expire marks the booking as expired
func (b *Booking) Expire() error {
	if b.Status != BookingStatusReserved {
		return ErrInvalidBookingStatus
	}
	b.Status = BookingStatusExpired
	b.UpdatedAt = time.Now()
	return nil
}

// TimeUntilExpiry returns the duration until the booking expires
func (b *Booking) TimeUntilExpiry() time.Duration {
	return time.Until(b.ExpiresAt)
}

// BelongsToUser checks if the booking belongs to the specified user
func (b *Booking) BelongsToUser(userID string) bool {
	return b.UserID == userID
}
