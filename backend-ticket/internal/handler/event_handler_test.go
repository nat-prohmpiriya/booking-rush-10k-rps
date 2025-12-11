package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/middleware"
)

// MockEventService is a mock implementation of EventService
type MockEventService struct {
	events map[string]*domain.Event
}

func NewMockEventService() *MockEventService {
	return &MockEventService{
		events: make(map[string]*domain.Event),
	}
}

func (m *MockEventService) CreateEvent(ctx context.Context, req *dto.CreateEventRequest) (*domain.Event, error) {
	now := time.Now()
	event := &domain.Event{
		ID:             "event-123",
		Name:           req.Name,
		Slug:           "test-slug",
		Description:    req.Description,
		VenueName:      req.VenueName,
		VenueAddress:   req.VenueAddress,
		BookingStartAt: req.BookingStartAt,
		BookingEndAt:   req.BookingEndAt,
		Status:         domain.EventStatusDraft,
		TenantID:       req.TenantID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	m.events[event.ID] = event
	return event, nil
}

func (m *MockEventService) GetEventByID(ctx context.Context, id string) (*domain.Event, error) {
	event, ok := m.events[id]
	if !ok {
		return nil, service.ErrEventNotFound
	}
	return event, nil
}

func (m *MockEventService) GetEventBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	for _, e := range m.events {
		if e.Slug == slug {
			return e, nil
		}
	}
	return nil, service.ErrEventNotFound
}

func (m *MockEventService) ListEvents(ctx context.Context, filter *dto.EventListFilter) ([]*domain.Event, int, error) {
	var events []*domain.Event
	for _, e := range m.events {
		events = append(events, e)
	}
	return events, len(events), nil
}

func (m *MockEventService) UpdateEvent(ctx context.Context, id string, req *dto.UpdateEventRequest) (*domain.Event, error) {
	event, ok := m.events[id]
	if !ok {
		return nil, service.ErrEventNotFound
	}
	if req.Name != "" {
		event.Name = req.Name
	}
	if req.Description != "" {
		event.Description = req.Description
	}
	return event, nil
}

func (m *MockEventService) DeleteEvent(ctx context.Context, id string) error {
	if _, ok := m.events[id]; !ok {
		return service.ErrEventNotFound
	}
	delete(m.events, id)
	return nil
}

func (m *MockEventService) PublishEvent(ctx context.Context, id string) (*domain.Event, error) {
	event, ok := m.events[id]
	if !ok {
		return nil, service.ErrEventNotFound
	}
	if event.Status != domain.EventStatusDraft {
		return nil, service.ErrInvalidEventStatus
	}
	event.Status = domain.EventStatusPublished
	return event, nil
}

func (m *MockEventService) ListPublishedEvents(ctx context.Context, limit, offset int) ([]*domain.Event, int, error) {
	var events []*domain.Event
	for _, e := range m.events {
		if e.Status == domain.EventStatusPublished {
			events = append(events, e)
		}
	}
	return events, len(events), nil
}

// AddEvent adds an event to the mock service
func (m *MockEventService) AddEvent(event *domain.Event) {
	m.events[event.ID] = event
}

func setupRouter(h *EventHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	events := router.Group("/events")
	{
		events.GET("", h.List)
		events.GET("/:slug", h.GetBySlug)
		events.GET("/id/:id", h.GetByID)
		events.POST("", h.Create)
		events.PUT("/:id", h.Update)
		events.DELETE("/:id", h.Delete)
		events.POST("/:id/publish", h.Publish)
	}

	return router
}

func TestEventHandler_List(t *testing.T) {
	mockSvc := NewMockEventService()
	handler := NewEventHandler(mockSvc)
	router := setupRouter(handler)

	// Add test event
	now := time.Now()
	mockSvc.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	})

	req, _ := http.NewRequest(http.MethodGet, "/events", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
}

func TestEventHandler_GetBySlug(t *testing.T) {
	mockSvc := NewMockEventService()
	handler := NewEventHandler(mockSvc)
	router := setupRouter(handler)

	// Add test event
	now := time.Now()
	mockSvc.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name       string
		slug       string
		wantStatus int
	}{
		{
			name:       "existing event",
			slug:       "test-event",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent event",
			slug:       "non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/events/"+tt.slug, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestEventHandler_GetByID(t *testing.T) {
	mockSvc := NewMockEventService()
	handler := NewEventHandler(mockSvc)
	router := setupRouter(handler)

	// Add test event
	now := time.Now()
	mockSvc.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "existing event",
			id:         "event-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent event",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/events/id/"+tt.id, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestEventHandler_Create(t *testing.T) {
	mockSvc := NewMockEventService()
	handler := NewEventHandler(mockSvc)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	events := router.Group("/events")
	{
		// Simulate JWT middleware setting tenant_id
		events.POST("", func(c *gin.Context) {
			c.Set(middleware.ContextKeyTenantID, "tenant-1")
			c.Next()
		}, handler.Create)
	}

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "valid request",
			body: map[string]interface{}{
				"name":             "New Event",
				"venue_name":       "Test Venue",
				"booking_start_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				"booking_end_at":   time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing name",
			body: map[string]interface{}{
				"venue_name":       "Test Venue",
				"booking_start_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				"booking_end_at":   time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/events", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestEventHandler_Update(t *testing.T) {
	mockSvc := NewMockEventService()
	handler := NewEventHandler(mockSvc)
	router := setupRouter(handler)

	// Add test event
	now := time.Now()
	mockSvc.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name       string
		id         string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "valid update",
			id:   "event-1",
			body: map[string]interface{}{
				"name": "Updated Event",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty update",
			id:         "event-1",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "non-existent event",
			id:   "non-existent",
			body: map[string]interface{}{
				"name": "Updated",
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPut, "/events/"+tt.id, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestEventHandler_Delete(t *testing.T) {
	mockSvc := NewMockEventService()
	handler := NewEventHandler(mockSvc)
	router := setupRouter(handler)

	// Add test event
	now := time.Now()
	mockSvc.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "delete existing event",
			id:         "event-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete non-existent event",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodDelete, "/events/"+tt.id, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestEventHandler_Publish(t *testing.T) {
	mockSvc := NewMockEventService()
	handler := NewEventHandler(mockSvc)
	router := setupRouter(handler)

	// Add test events
	now := time.Now()
	mockSvc.AddEvent(&domain.Event{
		ID:        "event-draft",
		Name:      "Draft Event",
		Slug:      "draft-event",
		Status:    domain.EventStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	})
	mockSvc.AddEvent(&domain.Event{
		ID:        "event-published",
		Name:      "Published Event",
		Slug:      "published-event",
		Status:    domain.EventStatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "publish draft event",
			id:         "event-draft",
			wantStatus: http.StatusOK,
		},
		{
			name:       "publish already published event",
			id:         "event-published",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "publish non-existent event",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/events/"+tt.id+"/publish", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}
