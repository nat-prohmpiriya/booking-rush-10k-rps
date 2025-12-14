package service

import (
	"context"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/repository"
)

// MockShowRepository is a mock implementation of ShowRepository
type MockShowRepository struct {
	shows map[string]*domain.Show
}

func NewMockShowRepository() *MockShowRepository {
	return &MockShowRepository{
		shows: make(map[string]*domain.Show),
	}
}

func (m *MockShowRepository) Create(ctx context.Context, show *domain.Show) error {
	m.shows[show.ID] = show
	return nil
}

func (m *MockShowRepository) GetByID(ctx context.Context, id string) (*domain.Show, error) {
	show, ok := m.shows[id]
	if !ok {
		return nil, nil
	}
	return show, nil
}

func (m *MockShowRepository) GetByEventID(ctx context.Context, eventID string, limit, offset int) ([]*domain.Show, int, error) {
	var shows []*domain.Show
	for _, s := range m.shows {
		if s.EventID == eventID && s.DeletedAt == nil {
			shows = append(shows, s)
		}
	}
	return shows, len(shows), nil
}

func (m *MockShowRepository) Update(ctx context.Context, show *domain.Show) error {
	if _, ok := m.shows[show.ID]; !ok {
		return ErrShowNotFound
	}
	m.shows[show.ID] = show
	return nil
}

func (m *MockShowRepository) Delete(ctx context.Context, id string) error {
	show, ok := m.shows[id]
	if !ok {
		return ErrShowNotFound
	}
	now := time.Now()
	show.DeletedAt = &now
	return nil
}

func (m *MockShowRepository) AddShow(show *domain.Show) {
	m.shows[show.ID] = show
}

// MockEventRepoForShow is a mock implementation of EventRepository for show tests
type MockEventRepoForShow struct {
	events map[string]*domain.Event
}

func NewMockEventRepoForShow() *MockEventRepoForShow {
	return &MockEventRepoForShow{
		events: make(map[string]*domain.Event),
	}
}

func (m *MockEventRepoForShow) Create(ctx context.Context, event *domain.Event) error {
	m.events[event.ID] = event
	return nil
}

func (m *MockEventRepoForShow) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	event, ok := m.events[id]
	if !ok {
		return nil, nil
	}
	return event, nil
}

func (m *MockEventRepoForShow) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	for _, e := range m.events {
		if e.Slug == slug {
			return e, nil
		}
	}
	return nil, nil
}

func (m *MockEventRepoForShow) GetByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, error) {
	return nil, nil
}

func (m *MockEventRepoForShow) Update(ctx context.Context, event *domain.Event) error {
	return nil
}

func (m *MockEventRepoForShow) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockEventRepoForShow) ListPublished(ctx context.Context, limit, offset int) ([]*domain.Event, int, error) {
	return nil, 0, nil
}

func (m *MockEventRepoForShow) List(ctx context.Context, filter *repository.EventFilter, limit, offset int) ([]*domain.Event, int, error) {
	return nil, 0, nil
}

func (m *MockEventRepoForShow) SlugExists(ctx context.Context, slug string) (bool, error) {
	return false, nil
}

func (m *MockEventRepoForShow) AddEvent(event *domain.Event) {
	m.events[event.ID] = event
}

// MockZoneSyncerForShow is a mock implementation of ZoneSyncer
type MockZoneSyncerForShow struct{}

func (m *MockZoneSyncerForShow) SyncByShowID(ctx context.Context, showID string) error {
	return nil
}

func (m *MockZoneSyncerForShow) RemoveByShowID(ctx context.Context, showID string) error {
	return nil
}

func (m *MockZoneSyncerForShow) SyncZone(ctx context.Context, zone *domain.ShowZone) error {
	return nil
}

func (m *MockZoneSyncerForShow) RemoveZone(ctx context.Context, zoneID string) error {
	return nil
}

func TestShowService_CreateShow(t *testing.T) {
	mockShowRepo := NewMockShowRepository()
	mockEventRepo := NewMockEventRepoForShow()
	mockZoneSyncer := &MockZoneSyncerForShow{}
	svc := NewShowService(mockShowRepo, mockEventRepo, mockZoneSyncer)

	// Add test event
	now := time.Now()
	mockEventRepo.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name    string
		req     *dto.CreateShowRequest
		wantErr bool
		errType error
	}{
		{
			name: "valid request",
			req: &dto.CreateShowRequest{
				EventID:   "event-1",
				Name:      "Evening Show",
				ShowDate:  "2025-01-15",
				StartTime: "19:00:00",
				EndTime:   "22:00:00",
			},
			wantErr: false,
		},
		{
			name: "event not found",
			req: &dto.CreateShowRequest{
				EventID:   "non-existent",
				Name:      "Evening Show",
				ShowDate:  "2025-01-15",
				StartTime: "19:00:00",
				EndTime:   "22:00:00",
			},
			wantErr: true,
			errType: ErrEventNotFound,
		},
		{
			name: "missing show date",
			req: &dto.CreateShowRequest{
				EventID:   "event-1",
				Name:      "Evening Show",
				ShowDate:  "",
				StartTime: "19:00:00",
				EndTime:   "22:00:00",
			},
			wantErr: true,
		},
		{
			name: "missing start time",
			req: &dto.CreateShowRequest{
				EventID:   "event-1",
				Name:      "Evening Show",
				ShowDate:  "2025-01-15",
				StartTime: "",
				EndTime:   "22:00:00",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			show, err := svc.CreateShow(context.Background(), tt.req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if show.Name != tt.req.Name {
				t.Errorf("expected name %s, got %s", tt.req.Name, show.Name)
			}
			if show.Status != domain.ShowStatusScheduled {
				t.Errorf("expected status %s, got %s", domain.ShowStatusScheduled, show.Status)
			}
		})
	}
}

func TestShowService_GetShowByID(t *testing.T) {
	mockShowRepo := NewMockShowRepository()
	mockEventRepo := NewMockEventRepoForShow()
	mockZoneSyncer := &MockZoneSyncerForShow{}
	svc := NewShowService(mockShowRepo, mockEventRepo, mockZoneSyncer)

	// Add test show
	now := time.Now()
	mockShowRepo.AddShow(&domain.Show{
		ID:        "show-1",
		EventID:   "event-1",
		Name:      "Evening Show",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		Status:    domain.ShowStatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing show",
			id:      "show-1",
			wantErr: false,
		},
		{
			name:    "non-existent show",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			show, err := svc.GetShowByID(context.Background(), tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if show.ID != tt.id {
				t.Errorf("expected ID %s, got %s", tt.id, show.ID)
			}
		})
	}
}

func TestShowService_ListShowsByEvent(t *testing.T) {
	mockShowRepo := NewMockShowRepository()
	mockEventRepo := NewMockEventRepoForShow()
	mockZoneSyncer := &MockZoneSyncerForShow{}
	svc := NewShowService(mockShowRepo, mockEventRepo, mockZoneSyncer)

	// Add test event
	now := time.Now()
	mockEventRepo.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Add test shows
	mockShowRepo.AddShow(&domain.Show{
		ID:        "show-1",
		EventID:   "event-1",
		Name:      "Morning Show",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		Status:    domain.ShowStatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
	})
	mockShowRepo.AddShow(&domain.Show{
		ID:        "show-2",
		EventID:   "event-1",
		Name:      "Evening Show",
		StartTime: now.Add(30 * time.Hour),
		EndTime:   now.Add(32 * time.Hour),
		Status:    domain.ShowStatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name      string
		eventID   string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "existing event with shows",
			eventID:   "event-1",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "non-existent event",
			eventID:   "non-existent",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &dto.ShowListFilter{Limit: 20, Offset: 0}
			shows, total, err := svc.ListShowsByEvent(context.Background(), tt.eventID, filter)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if total != tt.wantCount {
				t.Errorf("expected %d shows, got %d", tt.wantCount, total)
			}
			if len(shows) != tt.wantCount {
				t.Errorf("expected %d shows in slice, got %d", tt.wantCount, len(shows))
			}
		})
	}
}

func TestShowService_UpdateShow(t *testing.T) {
	mockShowRepo := NewMockShowRepository()
	mockEventRepo := NewMockEventRepoForShow()
	mockZoneSyncer := &MockZoneSyncerForShow{}
	svc := NewShowService(mockShowRepo, mockEventRepo, mockZoneSyncer)

	// Add test show
	now := time.Now()
	mockShowRepo.AddShow(&domain.Show{
		ID:        "show-1",
		EventID:   "event-1",
		Name:      "Original Name",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		Status:    domain.ShowStatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name    string
		id      string
		req     *dto.UpdateShowRequest
		wantErr bool
	}{
		{
			name: "valid update",
			id:   "show-1",
			req: &dto.UpdateShowRequest{
				Name: "Updated Name",
			},
			wantErr: false,
		},
		{
			name: "non-existent show",
			id:   "non-existent",
			req: &dto.UpdateShowRequest{
				Name: "Updated Name",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			show, err := svc.UpdateShow(context.Background(), tt.id, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.req.Name != "" && show.Name != tt.req.Name {
				t.Errorf("expected name %s, got %s", tt.req.Name, show.Name)
			}
		})
	}
}

func TestShowService_DeleteShow(t *testing.T) {
	mockShowRepo := NewMockShowRepository()
	mockEventRepo := NewMockEventRepoForShow()
	mockZoneSyncer := &MockZoneSyncerForShow{}
	svc := NewShowService(mockShowRepo, mockEventRepo, mockZoneSyncer)

	// Add test show
	now := time.Now()
	mockShowRepo.AddShow(&domain.Show{
		ID:        "show-1",
		EventID:   "event-1",
		Name:      "Test Show",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		Status:    domain.ShowStatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "delete existing show",
			id:      "show-1",
			wantErr: false,
		},
		{
			name:    "delete non-existent show",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.DeleteShow(context.Background(), tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
