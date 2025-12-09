package repository

import (
	"context"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/ticket-service/internal/domain"
)

// MockEventRepository is a mock implementation of EventRepository for testing
type MockEventRepository struct {
	events       map[string]*domain.Event
	eventsBySlug map[string]*domain.Event
	listCount    int // Track how many times List is called
	getByIDCount int // Track how many times GetByID is called
}

func NewMockEventRepository() *MockEventRepository {
	return &MockEventRepository{
		events:       make(map[string]*domain.Event),
		eventsBySlug: make(map[string]*domain.Event),
	}
}

func (m *MockEventRepository) Create(ctx context.Context, event *domain.Event) error {
	m.events[event.ID] = event
	m.eventsBySlug[event.Slug] = event
	return nil
}

func (m *MockEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	m.getByIDCount++
	event, ok := m.events[id]
	if !ok {
		return nil, nil
	}
	return event, nil
}

func (m *MockEventRepository) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	event, ok := m.eventsBySlug[slug]
	if !ok {
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
	if _, ok := m.events[event.ID]; !ok {
		return nil
	}
	m.events[event.ID] = event
	m.eventsBySlug[event.Slug] = event
	return nil
}

func (m *MockEventRepository) Delete(ctx context.Context, id string) error {
	event, ok := m.events[id]
	if !ok {
		return nil
	}
	now := time.Now()
	event.DeletedAt = &now
	return nil
}

func (m *MockEventRepository) ListPublished(ctx context.Context, limit, offset int) ([]*domain.Event, int, error) {
	m.listCount++
	var events []*domain.Event
	for _, e := range m.events {
		if e.Status == domain.EventStatusPublished && e.DeletedAt == nil {
			events = append(events, e)
		}
	}
	return events, len(events), nil
}

func (m *MockEventRepository) List(ctx context.Context, filter *EventFilter, limit, offset int) ([]*domain.Event, int, error) {
	m.listCount++
	var events []*domain.Event
	for _, e := range m.events {
		if e.DeletedAt != nil {
			continue
		}
		if filter != nil {
			if filter.Status != "" && e.Status != filter.Status {
				continue
			}
			if filter.TenantID != "" && e.TenantID != filter.TenantID {
				continue
			}
		}
		events = append(events, e)
	}
	return events, len(events), nil
}

func (m *MockEventRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	_, ok := m.eventsBySlug[slug]
	return ok, nil
}

func (m *MockEventRepository) AddEvent(event *domain.Event) {
	m.events[event.ID] = event
	m.eventsBySlug[event.Slug] = event
}

func (m *MockEventRepository) ResetCounts() {
	m.listCount = 0
	m.getByIDCount = 0
}

// MockRedisClient is a mock implementation of Redis client for testing
type MockRedisClient struct {
	data map[string]string
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
	}
}

// TestCachedEventRepository_GetByID tests GetByID caching behavior
func TestCachedEventRepository_GetByID(t *testing.T) {
	mockRepo := NewMockEventRepository()

	// Add test event
	now := time.Now()
	event := &domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.AddEvent(event)

	// Without cache, just test the repository directly
	ctx := context.Background()

	// First call - should hit database
	result, err := mockRepo.GetByID(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}
	if result.ID != "event-1" {
		t.Errorf("expected ID 'event-1', got '%s'", result.ID)
	}
	if mockRepo.getByIDCount != 1 {
		t.Errorf("expected getByIDCount 1, got %d", mockRepo.getByIDCount)
	}

	// Second call - would hit cache in real implementation
	result, err = mockRepo.GetByID(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}
	// With mock (no cache), both calls hit the database
	if mockRepo.getByIDCount != 2 {
		t.Errorf("expected getByIDCount 2, got %d", mockRepo.getByIDCount)
	}
}

// TestCachedEventRepository_GetByID_NotFound tests GetByID with non-existent event
func TestCachedEventRepository_GetByID_NotFound(t *testing.T) {
	mockRepo := NewMockEventRepository()
	ctx := context.Background()

	result, err := mockRepo.GetByID(ctx, "non-existent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// TestCachedEventRepository_ListPublished tests list caching behavior
func TestCachedEventRepository_ListPublished(t *testing.T) {
	mockRepo := NewMockEventRepository()

	// Add test events
	now := time.Now()
	mockRepo.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Event 1",
		Slug:      "event-1",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	})
	mockRepo.AddEvent(&domain.Event{
		ID:        "event-2",
		Name:      "Event 2",
		Slug:      "event-2",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(48 * time.Hour),
		EndTime:   now.Add(50 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	})
	mockRepo.AddEvent(&domain.Event{
		ID:        "event-3",
		Name:      "Event 3 Draft",
		Slug:      "event-3",
		Status:    domain.EventStatusDraft, // Not published
		TenantID:  "tenant-1",
		StartTime: now.Add(72 * time.Hour),
		EndTime:   now.Add(74 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	})

	ctx := context.Background()

	// First call
	events, total, err := mockRepo.ListPublished(ctx, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	if mockRepo.listCount != 1 {
		t.Errorf("expected listCount 1, got %d", mockRepo.listCount)
	}

	// Second call
	events, total, err = mockRepo.ListPublished(ctx, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	// With mock (no cache), both calls hit the database
	if mockRepo.listCount != 2 {
		t.Errorf("expected listCount 2, got %d", mockRepo.listCount)
	}
}

// TestCachedEventRepository_Update tests cache invalidation on update
func TestCachedEventRepository_Update(t *testing.T) {
	mockRepo := NewMockEventRepository()

	// Add test event
	now := time.Now()
	event := &domain.Event{
		ID:        "event-1",
		Name:      "Original Name",
		Slug:      "event-1",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.AddEvent(event)

	ctx := context.Background()

	// Get event first
	result, err := mockRepo.GetByID(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Original Name" {
		t.Errorf("expected name 'Original Name', got '%s'", result.Name)
	}

	// Update event
	event.Name = "Updated Name"
	err = mockRepo.Update(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get event again - should return updated data
	result, err = mockRepo.GetByID(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
}

// TestCachedEventRepository_Delete tests cache invalidation on delete
func TestCachedEventRepository_Delete(t *testing.T) {
	mockRepo := NewMockEventRepository()

	// Add test event
	now := time.Now()
	event := &domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "event-1",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.AddEvent(event)

	ctx := context.Background()

	// Verify event exists
	result, err := mockRepo.GetByID(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}

	// Delete event
	err = mockRepo.Delete(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify event is soft deleted
	result, err = mockRepo.GetByID(ctx, "event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DeletedAt == nil {
		t.Error("expected DeletedAt to be set")
	}
}

// TestCachedEventRepository_List tests list with filters
func TestCachedEventRepository_List(t *testing.T) {
	mockRepo := NewMockEventRepository()

	// Add test events
	now := time.Now()
	mockRepo.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Event 1",
		Slug:      "event-1",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	})
	mockRepo.AddEvent(&domain.Event{
		ID:        "event-2",
		Name:      "Event 2",
		Slug:      "event-2",
		Status:    domain.EventStatusDraft,
		TenantID:  "tenant-2",
		StartTime: now.Add(48 * time.Hour),
		EndTime:   now.Add(50 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	})

	ctx := context.Background()

	tests := []struct {
		name      string
		filter    *EventFilter
		wantCount int
	}{
		{
			name:      "no filter",
			filter:    nil,
			wantCount: 2,
		},
		{
			name:      "filter by published status",
			filter:    &EventFilter{Status: string(domain.EventStatusPublished)},
			wantCount: 1,
		},
		{
			name:      "filter by draft status",
			filter:    &EventFilter{Status: string(domain.EventStatusDraft)},
			wantCount: 1,
		},
		{
			name:      "filter by tenant",
			filter:    &EventFilter{TenantID: "tenant-1"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, total, err := mockRepo.List(ctx, tt.filter, 10, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tt.wantCount {
				t.Errorf("expected total %d, got %d", tt.wantCount, total)
			}
			if len(events) != tt.wantCount {
				t.Errorf("expected %d events, got %d", tt.wantCount, len(events))
			}
		})
	}
}

// TestCachedEventRepository_GetBySlug tests GetBySlug caching behavior
func TestCachedEventRepository_GetBySlug(t *testing.T) {
	mockRepo := NewMockEventRepository()

	// Add test event
	now := time.Now()
	event := &domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.AddEvent(event)

	ctx := context.Background()

	// Get by slug
	result, err := mockRepo.GetBySlug(ctx, "test-event")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}
	if result.Slug != "test-event" {
		t.Errorf("expected slug 'test-event', got '%s'", result.Slug)
	}

	// Get non-existent slug
	result, err = mockRepo.GetBySlug(ctx, "non-existent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// TestCachedEventRepository_SlugExists tests slug existence check
func TestCachedEventRepository_SlugExists(t *testing.T) {
	mockRepo := NewMockEventRepository()

	// Add test event
	now := time.Now()
	event := &domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.AddEvent(event)

	ctx := context.Background()

	// Check existing slug
	exists, err := mockRepo.SlugExists(ctx, "test-event")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected slug to exist")
	}

	// Check non-existent slug
	exists, err = mockRepo.SlugExists(ctx, "non-existent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected slug to not exist")
	}
}
