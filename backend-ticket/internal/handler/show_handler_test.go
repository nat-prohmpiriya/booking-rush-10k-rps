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
)

// MockShowService is a mock implementation of ShowService
type MockShowService struct {
	shows map[string]*domain.Show
}

func NewMockShowService() *MockShowService {
	return &MockShowService{
		shows: make(map[string]*domain.Show),
	}
}

func (m *MockShowService) CreateShow(ctx context.Context, req *dto.CreateShowRequest) (*domain.Show, error) {
	now := time.Now()
	show := &domain.Show{
		ID:        "show-123",
		EventID:   req.EventID,
		Name:      req.Name,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Status:    domain.ShowStatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.shows[show.ID] = show
	return show, nil
}

func (m *MockShowService) GetShowByID(ctx context.Context, id string) (*domain.Show, error) {
	show, ok := m.shows[id]
	if !ok {
		return nil, service.ErrShowNotFound
	}
	return show, nil
}

func (m *MockShowService) ListShowsByEvent(ctx context.Context, eventID string, filter *dto.ShowListFilter) ([]*domain.Show, int, error) {
	var shows []*domain.Show
	for _, s := range m.shows {
		if s.EventID == eventID && s.DeletedAt == nil {
			shows = append(shows, s)
		}
	}
	return shows, len(shows), nil
}

func (m *MockShowService) UpdateShow(ctx context.Context, id string, req *dto.UpdateShowRequest) (*domain.Show, error) {
	show, ok := m.shows[id]
	if !ok {
		return nil, service.ErrShowNotFound
	}
	if req.Name != "" {
		show.Name = req.Name
	}
	if req.Status != "" {
		show.Status = req.Status
	}
	return show, nil
}

func (m *MockShowService) DeleteShow(ctx context.Context, id string) error {
	if _, ok := m.shows[id]; !ok {
		return service.ErrShowNotFound
	}
	delete(m.shows, id)
	return nil
}

func (m *MockShowService) AddShow(show *domain.Show) {
	m.shows[show.ID] = show
}

// MockEventServiceForShow is a mock implementation of EventService for show handler tests
type MockEventServiceForShow struct {
	events map[string]*domain.Event
}

func NewMockEventServiceForShow() *MockEventServiceForShow {
	return &MockEventServiceForShow{
		events: make(map[string]*domain.Event),
	}
}

func (m *MockEventServiceForShow) CreateEvent(ctx context.Context, req *dto.CreateEventRequest) (*domain.Event, error) {
	return nil, nil
}

func (m *MockEventServiceForShow) GetEventByID(ctx context.Context, id string) (*domain.Event, error) {
	event, ok := m.events[id]
	if !ok {
		return nil, service.ErrEventNotFound
	}
	return event, nil
}

func (m *MockEventServiceForShow) GetEventBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	for _, e := range m.events {
		if e.Slug == slug {
			return e, nil
		}
	}
	return nil, service.ErrEventNotFound
}

func (m *MockEventServiceForShow) ListEvents(ctx context.Context, filter *dto.EventListFilter) ([]*domain.Event, int, error) {
	return nil, 0, nil
}

func (m *MockEventServiceForShow) UpdateEvent(ctx context.Context, id string, req *dto.UpdateEventRequest) (*domain.Event, error) {
	return nil, nil
}

func (m *MockEventServiceForShow) DeleteEvent(ctx context.Context, id string) error {
	return nil
}

func (m *MockEventServiceForShow) PublishEvent(ctx context.Context, id string) (*domain.Event, error) {
	return nil, nil
}

func (m *MockEventServiceForShow) AddEvent(event *domain.Event) {
	m.events[event.ID] = event
}

func setupShowRouter(h *ShowHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	events := router.Group("/events")
	{
		events.GET("/:slug/shows", h.ListByEvent)
		events.POST("/:id/shows", h.Create)
	}

	shows := router.Group("/shows")
	{
		shows.GET("/:id", h.GetByID)
		shows.PUT("/:id", h.Update)
		shows.DELETE("/:id", h.Delete)
	}

	return router
}

func TestShowHandler_ListByEvent(t *testing.T) {
	mockShowSvc := NewMockShowService()
	mockEventSvc := NewMockEventServiceForShow()
	handler := NewShowHandler(mockShowSvc, mockEventSvc)
	router := setupShowRouter(handler)

	// Add test event
	now := time.Now()
	mockEventSvc.AddEvent(&domain.Event{
		ID:        "event-1",
		Name:      "Test Event",
		Slug:      "test-event",
		Status:    domain.EventStatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Add test shows
	mockShowSvc.AddShow(&domain.Show{
		ID:        "show-1",
		EventID:   "event-1",
		Name:      "Morning Show",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		Status:    domain.ShowStatusScheduled,
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
			req, _ := http.NewRequest(http.MethodGet, "/events/"+tt.slug+"/shows", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestShowHandler_Create(t *testing.T) {
	mockShowSvc := NewMockShowService()
	mockEventSvc := NewMockEventServiceForShow()
	handler := NewShowHandler(mockShowSvc, mockEventSvc)
	router := setupShowRouter(handler)

	now := time.Now()
	tests := []struct {
		name       string
		eventID    string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:    "valid request",
			eventID: "event-1",
			body: map[string]interface{}{
				"name":       "Evening Show",
				"start_time": now.Add(24 * time.Hour).Format(time.RFC3339),
				"end_time":   now.Add(26 * time.Hour).Format(time.RFC3339),
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:    "missing name",
			eventID: "event-1",
			body: map[string]interface{}{
				"start_time": now.Add(24 * time.Hour).Format(time.RFC3339),
				"end_time":   now.Add(26 * time.Hour).Format(time.RFC3339),
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/events/"+tt.eventID+"/shows", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestShowHandler_GetByID(t *testing.T) {
	mockShowSvc := NewMockShowService()
	mockEventSvc := NewMockEventServiceForShow()
	handler := NewShowHandler(mockShowSvc, mockEventSvc)
	router := setupShowRouter(handler)

	// Add test show
	now := time.Now()
	mockShowSvc.AddShow(&domain.Show{
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
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "existing show",
			id:         "show-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent show",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/shows/"+tt.id, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestShowHandler_Update(t *testing.T) {
	mockShowSvc := NewMockShowService()
	mockEventSvc := NewMockEventServiceForShow()
	handler := NewShowHandler(mockShowSvc, mockEventSvc)
	router := setupShowRouter(handler)

	// Add test show
	now := time.Now()
	mockShowSvc.AddShow(&domain.Show{
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
		name       string
		id         string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "valid update",
			id:   "show-1",
			body: map[string]interface{}{
				"name": "Updated Name",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty update",
			id:         "show-1",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "non-existent show",
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
			req, _ := http.NewRequest(http.MethodPut, "/shows/"+tt.id, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestShowHandler_Delete(t *testing.T) {
	mockShowSvc := NewMockShowService()
	mockEventSvc := NewMockEventServiceForShow()
	handler := NewShowHandler(mockShowSvc, mockEventSvc)
	router := setupShowRouter(handler)

	// Add test show
	now := time.Now()
	mockShowSvc.AddShow(&domain.Show{
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
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "delete existing show",
			id:         "show-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete non-existent show",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodDelete, "/shows/"+tt.id, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}
