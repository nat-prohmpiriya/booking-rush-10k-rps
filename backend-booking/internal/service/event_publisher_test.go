package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
)

// MockEventPublisher is a mock implementation of EventPublisher for testing
type MockEventPublisher struct {
	mu                    sync.Mutex
	createdEvents         []*domain.Booking
	confirmedEvents       []*domain.Booking
	cancelledEvents       []*domain.Booking
	expiredEvents         []*domain.Booking
	publishCreatedError   error
	publishConfirmedError error
	publishCancelledError error
	publishExpiredError   error
}

func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		createdEvents:   make([]*domain.Booking, 0),
		confirmedEvents: make([]*domain.Booking, 0),
		cancelledEvents: make([]*domain.Booking, 0),
		expiredEvents:   make([]*domain.Booking, 0),
	}
}

func (m *MockEventPublisher) PublishBookingCreated(ctx context.Context, booking *domain.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishCreatedError != nil {
		return m.publishCreatedError
	}
	m.createdEvents = append(m.createdEvents, booking)
	return nil
}

func (m *MockEventPublisher) PublishBookingConfirmed(ctx context.Context, booking *domain.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishConfirmedError != nil {
		return m.publishConfirmedError
	}
	m.confirmedEvents = append(m.confirmedEvents, booking)
	return nil
}

func (m *MockEventPublisher) PublishBookingCancelled(ctx context.Context, booking *domain.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishCancelledError != nil {
		return m.publishCancelledError
	}
	m.cancelledEvents = append(m.cancelledEvents, booking)
	return nil
}

func (m *MockEventPublisher) PublishBookingExpired(ctx context.Context, booking *domain.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishExpiredError != nil {
		return m.publishExpiredError
	}
	m.expiredEvents = append(m.expiredEvents, booking)
	return nil
}

func (m *MockEventPublisher) Close() error {
	return nil
}

func (m *MockEventPublisher) GetCreatedEvents() []*domain.Booking {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createdEvents
}

func (m *MockEventPublisher) GetConfirmedEvents() []*domain.Booking {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.confirmedEvents
}

func (m *MockEventPublisher) GetCancelledEvents() []*domain.Booking {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancelledEvents
}

func (m *MockEventPublisher) GetExpiredEvents() []*domain.Booking {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.expiredEvents
}

func TestNoOpEventPublisher(t *testing.T) {
	publisher := NewNoOpEventPublisher()
	ctx := context.Background()
	booking := &domain.Booking{
		ID:       "test-booking-123",
		UserID:   "user-123",
		EventID:  "event-123",
		ZoneID:   "zone-123",
		Quantity: 2,
		Status:   domain.BookingStatusReserved,
	}

	t.Run("PublishBookingCreated returns nil", func(t *testing.T) {
		err := publisher.PublishBookingCreated(ctx, booking)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("PublishBookingConfirmed returns nil", func(t *testing.T) {
		err := publisher.PublishBookingConfirmed(ctx, booking)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("PublishBookingCancelled returns nil", func(t *testing.T) {
		err := publisher.PublishBookingCancelled(ctx, booking)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("PublishBookingExpired returns nil", func(t *testing.T) {
		err := publisher.PublishBookingExpired(ctx, booking)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("Close returns nil", func(t *testing.T) {
		err := publisher.Close()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

func TestMockEventPublisher(t *testing.T) {
	ctx := context.Background()
	booking := &domain.Booking{
		ID:       "test-booking-123",
		UserID:   "user-123",
		EventID:  "event-123",
		ZoneID:   "zone-123",
		Quantity: 2,
		Status:   domain.BookingStatusReserved,
	}

	t.Run("PublishBookingCreated captures event", func(t *testing.T) {
		publisher := NewMockEventPublisher()
		err := publisher.PublishBookingCreated(ctx, booking)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		events := publisher.GetCreatedEvents()
		if len(events) != 1 {
			t.Errorf("expected 1 event, got %d", len(events))
		}
		if events[0].ID != booking.ID {
			t.Errorf("expected booking ID %s, got %s", booking.ID, events[0].ID)
		}
	})

	t.Run("PublishBookingConfirmed captures event", func(t *testing.T) {
		publisher := NewMockEventPublisher()
		err := publisher.PublishBookingConfirmed(ctx, booking)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		events := publisher.GetConfirmedEvents()
		if len(events) != 1 {
			t.Errorf("expected 1 event, got %d", len(events))
		}
	})

	t.Run("PublishBookingCancelled captures event", func(t *testing.T) {
		publisher := NewMockEventPublisher()
		err := publisher.PublishBookingCancelled(ctx, booking)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		events := publisher.GetCancelledEvents()
		if len(events) != 1 {
			t.Errorf("expected 1 event, got %d", len(events))
		}
	})

	t.Run("PublishBookingExpired captures event", func(t *testing.T) {
		publisher := NewMockEventPublisher()
		err := publisher.PublishBookingExpired(ctx, booking)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		events := publisher.GetExpiredEvents()
		if len(events) != 1 {
			t.Errorf("expected 1 event, got %d", len(events))
		}
	})
}

func TestBookingEvent(t *testing.T) {
	now := time.Now()
	booking := &domain.Booking{
		ID:         "booking-123",
		TenantID:   "tenant-123",
		UserID:     "user-123",
		EventID:    "event-123",
		ShowID:     "show-123",
		ZoneID:     "zone-123",
		Quantity:   2,
		UnitPrice:  500.00,
		TotalPrice: 1000.00,
		Currency:   "THB",
		Status:     domain.BookingStatusReserved,
		ReservedAt: now,
		ExpiresAt:  now.Add(10 * time.Minute),
	}

	t.Run("NewBookingEvent creates event with correct data", func(t *testing.T) {
		event := domain.NewBookingEvent(domain.BookingEventCreated, booking, "event-id-123")

		if event.EventID != "event-id-123" {
			t.Errorf("expected event ID 'event-id-123', got %s", event.EventID)
		}
		if event.EventType != domain.BookingEventCreated {
			t.Errorf("expected event type %s, got %s", domain.BookingEventCreated, event.EventType)
		}
		if event.Version != 1 {
			t.Errorf("expected version 1, got %d", event.Version)
		}
		if event.BookingData == nil {
			t.Error("expected booking data to be set")
		}
		if event.BookingData.BookingID != booking.ID {
			t.Errorf("expected booking ID %s, got %s", booking.ID, event.BookingData.BookingID)
		}
		if event.BookingData.UserID != booking.UserID {
			t.Errorf("expected user ID %s, got %s", booking.UserID, event.BookingData.UserID)
		}
		if event.BookingData.Quantity != booking.Quantity {
			t.Errorf("expected quantity %d, got %d", booking.Quantity, event.BookingData.Quantity)
		}
		if event.BookingData.TotalPrice != booking.TotalPrice {
			t.Errorf("expected total price %f, got %f", booking.TotalPrice, event.BookingData.TotalPrice)
		}
	})

	t.Run("Event Topic returns correct topic", func(t *testing.T) {
		event := domain.NewBookingEvent(domain.BookingEventCreated, booking, "event-id-123")
		if event.Topic() != "booking-events" {
			t.Errorf("expected topic 'booking-events', got %s", event.Topic())
		}
	})

	t.Run("Event Key returns booking ID", func(t *testing.T) {
		event := domain.NewBookingEvent(domain.BookingEventCreated, booking, "event-id-123")
		if event.Key() != booking.ID {
			t.Errorf("expected key %s, got %s", booking.ID, event.Key())
		}
	})

	t.Run("Event types are correct", func(t *testing.T) {
		if string(domain.BookingEventCreated) != "booking.created" {
			t.Errorf("expected 'booking.created', got %s", domain.BookingEventCreated)
		}
		if string(domain.BookingEventConfirmed) != "booking.confirmed" {
			t.Errorf("expected 'booking.confirmed', got %s", domain.BookingEventConfirmed)
		}
		if string(domain.BookingEventCancelled) != "booking.cancelled" {
			t.Errorf("expected 'booking.cancelled', got %s", domain.BookingEventCancelled)
		}
		if string(domain.BookingEventExpired) != "booking.expired" {
			t.Errorf("expected 'booking.expired', got %s", domain.BookingEventExpired)
		}
	})
}
