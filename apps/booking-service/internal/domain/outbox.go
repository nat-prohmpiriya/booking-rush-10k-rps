package domain

import (
	"encoding/json"
	"time"
)

// OutboxStatus represents the status of an outbox message
type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "pending"
	OutboxStatusPublished OutboxStatus = "published"
	OutboxStatusFailed    OutboxStatus = "failed"
)

// IsValid checks if the status is a valid OutboxStatus
func (s OutboxStatus) IsValid() bool {
	switch s {
	case OutboxStatusPending, OutboxStatusPublished, OutboxStatusFailed:
		return true
	}
	return false
}

// String returns the string representation of OutboxStatus
func (s OutboxStatus) String() string {
	return string(s)
}

// OutboxMessage represents a message in the outbox table
type OutboxMessage struct {
	ID            string       `json:"id"`
	AggregateType string       `json:"aggregate_type"` // e.g., "booking"
	AggregateID   string       `json:"aggregate_id"`   // ID of the related entity
	EventType     string       `json:"event_type"`     // e.g., "booking.created"
	Payload       []byte       `json:"payload"`        // JSON payload
	Topic         string       `json:"topic"`          // Kafka topic
	PartitionKey  string       `json:"partition_key"`  // For Kafka partitioning
	Status        OutboxStatus `json:"status"`
	RetryCount    int          `json:"retry_count"`
	MaxRetries    int          `json:"max_retries"`
	LastError     string       `json:"last_error,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	ProcessedAt   *time.Time   `json:"processed_at,omitempty"`
	PublishedAt   *time.Time   `json:"published_at,omitempty"`
}

// NewOutboxMessage creates a new outbox message
func NewOutboxMessage(aggregateType, aggregateID, eventType, topic string, payload interface{}) (*OutboxMessage, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &OutboxMessage{
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payloadBytes,
		Topic:         topic,
		PartitionKey:  aggregateID, // Default: use aggregate ID for partitioning
		Status:        OutboxStatusPending,
		RetryCount:    0,
		MaxRetries:    5,
		CreatedAt:     time.Now(),
	}, nil
}

// CanRetry checks if the message can be retried
func (m *OutboxMessage) CanRetry() bool {
	return m.Status == OutboxStatusFailed && m.RetryCount < m.MaxRetries
}

// MarkAsPublished marks the message as successfully published
func (m *OutboxMessage) MarkAsPublished() {
	now := time.Now()
	m.Status = OutboxStatusPublished
	m.PublishedAt = &now
	m.ProcessedAt = &now
}

// MarkAsFailed marks the message as failed
func (m *OutboxMessage) MarkAsFailed(err string) {
	now := time.Now()
	m.Status = OutboxStatusFailed
	m.LastError = err
	m.RetryCount++
	m.ProcessedAt = &now
}

// ResetForRetry resets the message for retry
func (m *OutboxMessage) ResetForRetry() {
	m.Status = OutboxStatusPending
	m.ProcessedAt = nil
}

// GetPayload unmarshals the payload into the given interface
func (m *OutboxMessage) GetPayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// BookingOutboxEvent creates an outbox message for a booking event
func BookingOutboxEvent(eventType BookingEventType, booking *Booking, eventID string) (*OutboxMessage, error) {
	event := NewBookingEvent(eventType, booking, eventID)
	return NewOutboxMessage(
		"booking",
		booking.ID,
		string(eventType),
		"booking-events",
		event,
	)
}
