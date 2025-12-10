package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOutboxStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status OutboxStatus
		want   bool
	}{
		{"pending is valid", OutboxStatusPending, true},
		{"published is valid", OutboxStatusPublished, true},
		{"failed is valid", OutboxStatusFailed, true},
		{"unknown is invalid", OutboxStatus("unknown"), false},
		{"empty is invalid", OutboxStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("OutboxStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutboxStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status OutboxStatus
		want   string
	}{
		{"pending", OutboxStatusPending, "pending"},
		{"published", OutboxStatusPublished, "published"},
		{"failed", OutboxStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("OutboxStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOutboxMessage(t *testing.T) {
	payload := map[string]interface{}{
		"booking_id": "book-123",
		"user_id":    "user-456",
	}

	msg, err := NewOutboxMessage("booking", "book-123", "booking.created", "booking-events", payload)
	if err != nil {
		t.Fatalf("NewOutboxMessage() error = %v", err)
	}

	if msg.AggregateType != "booking" {
		t.Errorf("AggregateType = %v, want %v", msg.AggregateType, "booking")
	}

	if msg.AggregateID != "book-123" {
		t.Errorf("AggregateID = %v, want %v", msg.AggregateID, "book-123")
	}

	if msg.EventType != "booking.created" {
		t.Errorf("EventType = %v, want %v", msg.EventType, "booking.created")
	}

	if msg.Topic != "booking-events" {
		t.Errorf("Topic = %v, want %v", msg.Topic, "booking-events")
	}

	if msg.PartitionKey != "book-123" {
		t.Errorf("PartitionKey = %v, want %v", msg.PartitionKey, "book-123")
	}

	if msg.Status != OutboxStatusPending {
		t.Errorf("Status = %v, want %v", msg.Status, OutboxStatusPending)
	}

	if msg.RetryCount != 0 {
		t.Errorf("RetryCount = %v, want %v", msg.RetryCount, 0)
	}

	if msg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %v, want %v", msg.MaxRetries, 5)
	}

	// Verify payload can be unmarshaled
	var decoded map[string]interface{}
	if err := msg.GetPayload(&decoded); err != nil {
		t.Errorf("GetPayload() error = %v", err)
	}

	if decoded["booking_id"] != "book-123" {
		t.Errorf("Payload booking_id = %v, want %v", decoded["booking_id"], "book-123")
	}
}

func TestOutboxMessage_CanRetry(t *testing.T) {
	tests := []struct {
		name       string
		status     OutboxStatus
		retryCount int
		maxRetries int
		want       bool
	}{
		{"failed with retries left", OutboxStatusFailed, 2, 5, true},
		{"failed at max retries", OutboxStatusFailed, 5, 5, false},
		{"failed over max retries", OutboxStatusFailed, 6, 5, false},
		{"pending cannot retry", OutboxStatusPending, 0, 5, false},
		{"published cannot retry", OutboxStatusPublished, 0, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &OutboxMessage{
				Status:     tt.status,
				RetryCount: tt.retryCount,
				MaxRetries: tt.maxRetries,
			}

			if got := msg.CanRetry(); got != tt.want {
				t.Errorf("CanRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutboxMessage_MarkAsPublished(t *testing.T) {
	msg := &OutboxMessage{
		ID:     "msg-123",
		Status: OutboxStatusPending,
	}

	msg.MarkAsPublished()

	if msg.Status != OutboxStatusPublished {
		t.Errorf("Status = %v, want %v", msg.Status, OutboxStatusPublished)
	}

	if msg.PublishedAt == nil {
		t.Error("PublishedAt should not be nil")
	}

	if msg.ProcessedAt == nil {
		t.Error("ProcessedAt should not be nil")
	}
}

func TestOutboxMessage_MarkAsFailed(t *testing.T) {
	msg := &OutboxMessage{
		ID:         "msg-123",
		Status:     OutboxStatusPending,
		RetryCount: 1,
	}

	errMsg := "kafka connection failed"
	msg.MarkAsFailed(errMsg)

	if msg.Status != OutboxStatusFailed {
		t.Errorf("Status = %v, want %v", msg.Status, OutboxStatusFailed)
	}

	if msg.LastError != errMsg {
		t.Errorf("LastError = %v, want %v", msg.LastError, errMsg)
	}

	if msg.RetryCount != 2 {
		t.Errorf("RetryCount = %v, want %v", msg.RetryCount, 2)
	}

	if msg.ProcessedAt == nil {
		t.Error("ProcessedAt should not be nil")
	}
}

func TestOutboxMessage_ResetForRetry(t *testing.T) {
	now := time.Now()
	msg := &OutboxMessage{
		ID:          "msg-123",
		Status:      OutboxStatusFailed,
		ProcessedAt: &now,
	}

	msg.ResetForRetry()

	if msg.Status != OutboxStatusPending {
		t.Errorf("Status = %v, want %v", msg.Status, OutboxStatusPending)
	}

	if msg.ProcessedAt != nil {
		t.Error("ProcessedAt should be nil after reset")
	}
}

func TestOutboxMessage_GetPayload(t *testing.T) {
	type payload struct {
		BookingID string `json:"booking_id"`
		UserID    string `json:"user_id"`
		Amount    int    `json:"amount"`
	}

	original := payload{
		BookingID: "book-123",
		UserID:    "user-456",
		Amount:    1000,
	}

	payloadBytes, _ := json.Marshal(original)
	msg := &OutboxMessage{
		Payload: payloadBytes,
	}

	var decoded payload
	if err := msg.GetPayload(&decoded); err != nil {
		t.Fatalf("GetPayload() error = %v", err)
	}

	if decoded.BookingID != original.BookingID {
		t.Errorf("BookingID = %v, want %v", decoded.BookingID, original.BookingID)
	}

	if decoded.UserID != original.UserID {
		t.Errorf("UserID = %v, want %v", decoded.UserID, original.UserID)
	}

	if decoded.Amount != original.Amount {
		t.Errorf("Amount = %v, want %v", decoded.Amount, original.Amount)
	}
}

func TestBookingOutboxEvent(t *testing.T) {
	booking := &Booking{
		ID:         "book-123",
		UserID:     "user-456",
		EventID:    "event-789",
		ZoneID:     "zone-A",
		Quantity:   2,
		TotalPrice: 2000,
		Currency:   "THB",
		Status:     BookingStatusReserved,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	msg, err := BookingOutboxEvent(BookingEventCreated, booking, "evt-123")
	if err != nil {
		t.Fatalf("BookingOutboxEvent() error = %v", err)
	}

	if msg.AggregateType != "booking" {
		t.Errorf("AggregateType = %v, want %v", msg.AggregateType, "booking")
	}

	if msg.AggregateID != booking.ID {
		t.Errorf("AggregateID = %v, want %v", msg.AggregateID, booking.ID)
	}

	if msg.EventType != string(BookingEventCreated) {
		t.Errorf("EventType = %v, want %v", msg.EventType, string(BookingEventCreated))
	}

	if msg.Topic != "booking-events" {
		t.Errorf("Topic = %v, want %v", msg.Topic, "booking-events")
	}

	// Verify payload contains booking event
	var event BookingEvent
	if err := msg.GetPayload(&event); err != nil {
		t.Errorf("GetPayload() error = %v", err)
	}

	if event.EventType != BookingEventCreated {
		t.Errorf("Event.EventType = %v, want %v", event.EventType, BookingEventCreated)
	}

	if event.BookingData.BookingID != booking.ID {
		t.Errorf("Event.BookingData.BookingID = %v, want %v", event.BookingData.BookingID, booking.ID)
	}
}
