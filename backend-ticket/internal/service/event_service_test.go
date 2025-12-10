package service

import (
	"context"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/repository"
)

// MockEventRepository is a mock implementation of EventRepository
type MockEventRepository struct {
	events    map[string]*domain.Event
	slugToID  map[string]string
	createErr error
	updateErr error
	deleteErr error
}

func NewMockEventRepository() *MockEventRepository {
	return &MockEventRepository{
		events:   make(map[string]*domain.Event),
		slugToID: make(map[string]string),
	}
}

func (m *MockEventRepository) Create(ctx context.Context, event *domain.Event) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.events[event.ID] = event
	m.slugToID[event.Slug] = event.ID
	return nil
}

func (m *MockEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	event, ok := m.events[id]
	if !ok || event.DeletedAt != nil {
		return nil, nil
	}
	return event, nil
}

func (m *MockEventRepository) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	id, ok := m.slugToID[slug]
	if !ok {
		return nil, nil
	}
	event := m.events[id]
	if event.DeletedAt != nil {
		return nil, nil
	}
	return event, nil
}

func (m *MockEventRepository) GetByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, error) {
	var events []*domain.Event
	for _, e := range m.events {
		if e.TenantID == tenantID && e.DeletedAt == nil {
			events = append(events, e)
		}
	}
	return events, nil
}

func (m *MockEventRepository) Update(ctx context.Context, event *domain.Event) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.events[event.ID] = event
	m.slugToID[event.Slug] = event.ID
	return nil
}

func (m *MockEventRepository) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if event, ok := m.events[id]; ok {
		now := time.Now()
		event.DeletedAt = &now
	}
	return nil
}

func (m *MockEventRepository) ListPublished(ctx context.Context, limit, offset int) ([]*domain.Event, int, error) {
	var events []*domain.Event
	for _, e := range m.events {
		if e.Status == domain.EventStatusPublished && e.DeletedAt == nil {
			events = append(events, e)
		}
	}
	return events, len(events), nil
}

func (m *MockEventRepository) List(ctx context.Context, filter *repository.EventFilter, limit, offset int) ([]*domain.Event, int, error) {
	var events []*domain.Event
	for _, e := range m.events {
		if e.DeletedAt != nil {
			continue
		}
		if filter != nil && filter.Status != "" && e.Status != filter.Status {
			continue
		}
		if filter != nil && filter.TenantID != "" && e.TenantID != filter.TenantID {
			continue
		}
		events = append(events, e)
	}
	return events, len(events), nil
}

func (m *MockEventRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	_, ok := m.slugToID[slug]
	return ok, nil
}

// MockVenueRepository is a mock implementation of VenueRepository
type MockVenueRepository struct {
	venues map[string]*domain.Venue
}

func NewMockVenueRepository() *MockVenueRepository {
	return &MockVenueRepository{
		venues: make(map[string]*domain.Venue),
	}
}

func (m *MockVenueRepository) Create(ctx context.Context, venue *domain.Venue) error {
	m.venues[venue.ID] = venue
	return nil
}

func (m *MockVenueRepository) GetByID(ctx context.Context, id string) (*domain.Venue, error) {
	venue, ok := m.venues[id]
	if !ok {
		return nil, nil
	}
	return venue, nil
}

func (m *MockVenueRepository) GetByTenantID(ctx context.Context, tenantID string) ([]*domain.Venue, error) {
	var venues []*domain.Venue
	for _, v := range m.venues {
		if v.TenantID == tenantID {
			venues = append(venues, v)
		}
	}
	return venues, nil
}

func (m *MockVenueRepository) Update(ctx context.Context, venue *domain.Venue) error {
	m.venues[venue.ID] = venue
	return nil
}

func (m *MockVenueRepository) Delete(ctx context.Context, id string) error {
	delete(m.venues, id)
	return nil
}

// Add venue helper
func (m *MockVenueRepository) AddVenue(venue *domain.Venue) {
	m.venues[venue.ID] = venue
}

func TestEventService_CreateEvent(t *testing.T) {
	eventRepo := NewMockEventRepository()
	venueRepo := NewMockVenueRepository()
	svc := NewEventService(eventRepo, venueRepo)

	// Add test venue
	venueRepo.AddVenue(&domain.Venue{
		ID:       "venue-1",
		Name:     "Test Venue",
		TenantID: "tenant-1",
	})

	ctx := context.Background()

	tests := []struct {
		name    string
		req     *dto.CreateEventRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: &dto.CreateEventRequest{
				Name:        "Test Concert",
				Description: "A test concert",
				VenueID:     "venue-1",
				StartTime:   time.Now().Add(24 * time.Hour),
				EndTime:     time.Now().Add(48 * time.Hour),
				TenantID:    "tenant-1",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: &dto.CreateEventRequest{
				VenueID:   "venue-1",
				StartTime: time.Now().Add(24 * time.Hour),
				EndTime:   time.Now().Add(48 * time.Hour),
				TenantID:  "tenant-1",
			},
			wantErr: true,
			errMsg:  "Event name is required",
		},
		{
			name: "venue not found",
			req: &dto.CreateEventRequest{
				Name:      "Test Concert",
				VenueID:   "non-existent",
				StartTime: time.Now().Add(24 * time.Hour),
				EndTime:   time.Now().Add(48 * time.Hour),
				TenantID:  "tenant-1",
			},
			wantErr: true,
			errMsg:  "venue not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := svc.CreateEvent(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if event == nil {
					t.Error("expected event but got nil")
				}
				if event != nil && event.Status != domain.EventStatusDraft {
					t.Errorf("expected status %q, got %q", domain.EventStatusDraft, event.Status)
				}
			}
		})
	}
}

func TestEventService_GetEventByID(t *testing.T) {
	eventRepo := NewMockEventRepository()
	venueRepo := NewMockVenueRepository()
	svc := NewEventService(eventRepo, venueRepo)

	ctx := context.Background()

	// Add test event
	now := time.Now()
	testEvent := &domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusDraft,
		TenantID:  "tenant-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	eventRepo.events[testEvent.ID] = testEvent
	eventRepo.slugToID[testEvent.Slug] = testEvent.ID

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing event",
			id:      "event-1",
			wantErr: false,
		},
		{
			name:    "non-existent event",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := svc.GetEventByID(ctx, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if event == nil {
					t.Error("expected event but got nil")
				}
			}
		})
	}
}

func TestEventService_GetEventBySlug(t *testing.T) {
	eventRepo := NewMockEventRepository()
	venueRepo := NewMockVenueRepository()
	svc := NewEventService(eventRepo, venueRepo)

	ctx := context.Background()

	// Add test event
	now := time.Now()
	testEvent := &domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusDraft,
		TenantID:  "tenant-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	eventRepo.events[testEvent.ID] = testEvent
	eventRepo.slugToID[testEvent.Slug] = testEvent.ID

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{
			name:    "existing event",
			slug:    "test-event",
			wantErr: false,
		},
		{
			name:    "non-existent slug",
			slug:    "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := svc.GetEventBySlug(ctx, tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if event == nil {
					t.Error("expected event but got nil")
				}
			}
		})
	}
}

func TestEventService_UpdateEvent(t *testing.T) {
	eventRepo := NewMockEventRepository()
	venueRepo := NewMockVenueRepository()
	svc := NewEventService(eventRepo, venueRepo)

	ctx := context.Background()

	// Add test event
	now := time.Now()
	testEvent := &domain.Event{
		ID:          "event-1",
		Name:        "Test Event",
		Slug:        "test-event",
		Description: "Original description",
		Status:      domain.EventStatusDraft,
		TenantID:    "tenant-1",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	eventRepo.events[testEvent.ID] = testEvent
	eventRepo.slugToID[testEvent.Slug] = testEvent.ID

	tests := []struct {
		name    string
		id      string
		req     *dto.UpdateEventRequest
		wantErr bool
	}{
		{
			name: "update name",
			id:   "event-1",
			req: &dto.UpdateEventRequest{
				Name: "Updated Event",
			},
			wantErr: false,
		},
		{
			name: "update description",
			id:   "event-1",
			req: &dto.UpdateEventRequest{
				Description: "Updated description",
			},
			wantErr: false,
		},
		{
			name:    "empty update",
			id:      "event-1",
			req:     &dto.UpdateEventRequest{},
			wantErr: true,
		},
		{
			name: "non-existent event",
			id:   "non-existent",
			req: &dto.UpdateEventRequest{
				Name: "Updated",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := svc.UpdateEvent(ctx, tt.id, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if event == nil {
					t.Error("expected event but got nil")
				}
			}
		})
	}
}

func TestEventService_DeleteEvent(t *testing.T) {
	eventRepo := NewMockEventRepository()
	venueRepo := NewMockVenueRepository()
	svc := NewEventService(eventRepo, venueRepo)

	ctx := context.Background()

	// Add test event
	now := time.Now()
	testEvent := &domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusDraft,
		TenantID:  "tenant-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	eventRepo.events[testEvent.ID] = testEvent
	eventRepo.slugToID[testEvent.Slug] = testEvent.ID

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "delete existing event",
			id:      "event-1",
			wantErr: false,
		},
		{
			name:    "delete non-existent event",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.DeleteEvent(ctx, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEventService_PublishEvent(t *testing.T) {
	eventRepo := NewMockEventRepository()
	venueRepo := NewMockVenueRepository()
	svc := NewEventService(eventRepo, venueRepo)

	ctx := context.Background()

	// Add test events
	now := time.Now()
	draftEvent := &domain.Event{
		ID:        "event-draft",
		Name:      "Draft Event",
		Slug:      "draft-event",
		Status:    domain.EventStatusDraft,
		TenantID:  "tenant-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	eventRepo.events[draftEvent.ID] = draftEvent
	eventRepo.slugToID[draftEvent.Slug] = draftEvent.ID

	publishedEvent := &domain.Event{
		ID:        "event-published",
		Name:      "Published Event",
		Slug:      "published-event",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	eventRepo.events[publishedEvent.ID] = publishedEvent
	eventRepo.slugToID[publishedEvent.Slug] = publishedEvent.ID

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "publish draft event",
			id:      "event-draft",
			wantErr: false,
		},
		{
			name:    "publish already published event",
			id:      "event-published",
			wantErr: true,
		},
		{
			name:    "publish non-existent event",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := svc.PublishEvent(ctx, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if event == nil {
					t.Error("expected event but got nil")
				}
				if event != nil && event.Status != domain.EventStatusPublished {
					t.Errorf("expected status %q, got %q", domain.EventStatusPublished, event.Status)
				}
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Test Concert",
			expected: "test-concert",
		},
		{
			input:    "Rock & Roll Night",
			expected: "rock-roll-night",
		},
		{
			input:    "  Multiple   Spaces  ",
			expected: "multiple-spaces",
		},
		{
			input:    "Special@#$Characters!",
			expected: "specialcharacters",
		},
		{
			input:    "Concert 2024",
			expected: "concert-2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
