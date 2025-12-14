package consumer

import (
	"time"
)

// BookingEventType represents the type of booking event
type BookingEventType string

const (
	BookingEventCreated   BookingEventType = "booking.created"
	BookingEventConfirmed BookingEventType = "booking.confirmed"
	BookingEventCancelled BookingEventType = "booking.cancelled"
	BookingEventExpired   BookingEventType = "booking.expired"
)

// BookingEvent represents a booking domain event received from Kafka
type BookingEvent struct {
	EventID     string            `json:"event_id"`
	EventType   BookingEventType  `json:"event_type"`
	OccurredAt  time.Time         `json:"occurred_at"`
	Version     int               `json:"version"`
	BookingData *BookingEventData `json:"data"`
}

// BookingEventData contains the booking data in the event
type BookingEventData struct {
	BookingID        string     `json:"booking_id"`
	TenantID         string     `json:"tenant_id,omitempty"`
	UserID           string     `json:"user_id"`
	EventID          string     `json:"event_id"`
	ShowID           string     `json:"show_id,omitempty"`
	ZoneID           string     `json:"zone_id"`
	Quantity         int        `json:"quantity"`
	UnitPrice        float64    `json:"unit_price"`
	TotalPrice       float64    `json:"total_price"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"`
	PaymentID        string     `json:"payment_id,omitempty"`
	ConfirmationCode string     `json:"confirmation_code,omitempty"`
	ReservedAt       time.Time  `json:"reserved_at"`
	ConfirmedAt      *time.Time `json:"confirmed_at,omitempty"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
}

// PaymentEventType represents the type of payment event
type PaymentEventType string

const (
	PaymentEventCreated   PaymentEventType = "payment.created"
	PaymentEventProcessing PaymentEventType = "payment.processing"
	PaymentEventSuccess   PaymentEventType = "payment.success"
	PaymentEventFailed    PaymentEventType = "payment.failed"
	PaymentEventRefunded  PaymentEventType = "payment.refunded"
)

// PaymentEvent represents a payment domain event to publish to Kafka
type PaymentEvent struct {
	EventID     string            `json:"event_id"`
	EventType   PaymentEventType  `json:"event_type"`
	OccurredAt  time.Time         `json:"occurred_at"`
	Version     int               `json:"version"`
	PaymentData *PaymentEventData `json:"data"`
}

// PaymentEventData contains the payment data in the event
type PaymentEventData struct {
	PaymentID        string    `json:"payment_id"`
	BookingID        string    `json:"booking_id"`
	UserID           string    `json:"user_id"`
	Amount           float64   `json:"amount"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	Method           string    `json:"method"`
	GatewayPaymentID string    `json:"gateway_payment_id,omitempty"`
	ErrorCode        string    `json:"error_code,omitempty"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	ProcessedAt      time.Time `json:"processed_at"`
}

// Topic returns the Kafka topic for payment events
func (e *PaymentEvent) Topic() string {
	return "payment-events"
}

// Key returns the partition key for this event
func (e *PaymentEvent) Key() string {
	if e.PaymentData != nil {
		return e.PaymentData.BookingID
	}
	return e.EventID
}
