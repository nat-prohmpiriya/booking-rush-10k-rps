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

// MockShowZoneService is a mock implementation of ShowZoneService
type MockShowZoneService struct {
	zones map[string]*domain.ShowZone
}

func NewMockShowZoneService() *MockShowZoneService {
	return &MockShowZoneService{
		zones: make(map[string]*domain.ShowZone),
	}
}

func (m *MockShowZoneService) CreateShowZone(ctx context.Context, req *dto.CreateShowZoneRequest) (*domain.ShowZone, error) {
	now := time.Now()
	zone := &domain.ShowZone{
		ID:             "zone-123",
		ShowID:         req.ShowID,
		Name:           req.Name,
		Price:          req.Price,
		TotalSeats:     req.TotalSeats,
		AvailableSeats: req.TotalSeats,
		Description:    req.Description,
		SortOrder:      req.SortOrder,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	m.zones[zone.ID] = zone
	return zone, nil
}

func (m *MockShowZoneService) GetShowZoneByID(ctx context.Context, id string) (*domain.ShowZone, error) {
	zone, ok := m.zones[id]
	if !ok {
		return nil, service.ErrShowZoneNotFound
	}
	return zone, nil
}

func (m *MockShowZoneService) ListZonesByShow(ctx context.Context, showID string, filter *dto.ShowZoneListFilter) ([]*domain.ShowZone, int, error) {
	var zones []*domain.ShowZone
	for _, z := range m.zones {
		if z.ShowID == showID && z.DeletedAt == nil {
			zones = append(zones, z)
		}
	}
	return zones, len(zones), nil
}

func (m *MockShowZoneService) UpdateShowZone(ctx context.Context, id string, req *dto.UpdateShowZoneRequest) (*domain.ShowZone, error) {
	zone, ok := m.zones[id]
	if !ok {
		return nil, service.ErrShowZoneNotFound
	}
	if req.Name != "" {
		zone.Name = req.Name
	}
	if req.Price != nil {
		zone.Price = *req.Price
	}
	if req.TotalSeats != nil {
		zone.TotalSeats = *req.TotalSeats
	}
	if req.SortOrder != nil {
		zone.SortOrder = *req.SortOrder
	}
	return zone, nil
}

func (m *MockShowZoneService) DeleteShowZone(ctx context.Context, id string) error {
	if _, ok := m.zones[id]; !ok {
		return service.ErrShowZoneNotFound
	}
	delete(m.zones, id)
	return nil
}

func (m *MockShowZoneService) ListActiveZones(ctx context.Context) ([]*domain.ShowZone, error) {
	var zones []*domain.ShowZone
	for _, z := range m.zones {
		if z.IsActive && z.DeletedAt == nil {
			zones = append(zones, z)
		}
	}
	return zones, nil
}

func (m *MockShowZoneService) AddZone(zone *domain.ShowZone) {
	m.zones[zone.ID] = zone
}

// MockShowServiceForZone is a mock implementation of ShowService for zone handler tests
type MockShowServiceForZone struct {
	shows map[string]*domain.Show
}

func NewMockShowServiceForZone() *MockShowServiceForZone {
	return &MockShowServiceForZone{
		shows: make(map[string]*domain.Show),
	}
}

func (m *MockShowServiceForZone) CreateShow(ctx context.Context, req *dto.CreateShowRequest) (*domain.Show, error) {
	return nil, nil
}

func (m *MockShowServiceForZone) GetShowByID(ctx context.Context, id string) (*domain.Show, error) {
	show, ok := m.shows[id]
	if !ok {
		return nil, service.ErrShowNotFound
	}
	return show, nil
}

func (m *MockShowServiceForZone) ListShowsByEvent(ctx context.Context, eventID string, filter *dto.ShowListFilter) ([]*domain.Show, int, error) {
	return nil, 0, nil
}

func (m *MockShowServiceForZone) UpdateShow(ctx context.Context, id string, req *dto.UpdateShowRequest) (*domain.Show, error) {
	return nil, nil
}

func (m *MockShowServiceForZone) DeleteShow(ctx context.Context, id string) error {
	return nil
}

func (m *MockShowServiceForZone) AddShow(show *domain.Show) {
	m.shows[show.ID] = show
}

func setupShowZoneRouter(h *ShowZoneHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	shows := router.Group("/shows")
	{
		shows.GET("/:id/zones", h.ListByShow)
		shows.POST("/:id/zones", h.Create)
	}

	zones := router.Group("/zones")
	{
		zones.GET("/:id", h.GetByID)
		zones.PUT("/:id", h.Update)
		zones.DELETE("/:id", h.Delete)
	}

	return router
}

func TestShowZoneHandler_ListByShow(t *testing.T) {
	mockZoneSvc := NewMockShowZoneService()
	mockShowSvc := NewMockShowServiceForZone()
	handler := NewShowZoneHandler(mockZoneSvc, mockShowSvc)
	router := setupShowZoneRouter(handler)

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

	// Add test zones
	mockZoneSvc.AddZone(&domain.ShowZone{
		ID:             "zone-1",
		ShowID:         "show-1",
		Name:           "VIP Zone",
		Price:          100.00,
		TotalSeats:     50,
		AvailableSeats: 45,
		SortOrder:      1,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	tests := []struct {
		name       string
		showID     string
		wantStatus int
	}{
		{
			name:       "existing show",
			showID:     "show-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent show",
			showID:     "non-existent",
			wantStatus: http.StatusOK, // Returns empty list, not 404
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/shows/"+tt.showID+"/zones", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestShowZoneHandler_Create(t *testing.T) {
	mockZoneSvc := NewMockShowZoneService()
	mockShowSvc := NewMockShowServiceForZone()
	handler := NewShowZoneHandler(mockZoneSvc, mockShowSvc)
	router := setupShowZoneRouter(handler)

	tests := []struct {
		name       string
		showID     string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:   "valid request",
			showID: "show-1",
			body: map[string]interface{}{
				"name":        "VIP Zone",
				"price":       100.00,
				"total_seats": 50,
				"description": "VIP seating area",
				"sort_order":  1,
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:   "missing name",
			showID: "show-1",
			body: map[string]interface{}{
				"price":       100.00,
				"total_seats": 50,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "missing total_seats",
			showID: "show-1",
			body: map[string]interface{}{
				"name":  "VIP Zone",
				"price": 100.00,
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/shows/"+tt.showID+"/zones", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestShowZoneHandler_GetByID(t *testing.T) {
	mockZoneSvc := NewMockShowZoneService()
	mockShowSvc := NewMockShowServiceForZone()
	handler := NewShowZoneHandler(mockZoneSvc, mockShowSvc)
	router := setupShowZoneRouter(handler)

	// Add test zone
	now := time.Now()
	mockZoneSvc.AddZone(&domain.ShowZone{
		ID:             "zone-1",
		ShowID:         "show-1",
		Name:           "VIP Zone",
		Price:          100.00,
		TotalSeats:     50,
		AvailableSeats: 45,
		SortOrder:      1,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "existing zone",
			id:         "zone-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent zone",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/zones/"+tt.id, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}

func TestShowZoneHandler_Update(t *testing.T) {
	mockZoneSvc := NewMockShowZoneService()
	mockShowSvc := NewMockShowServiceForZone()
	handler := NewShowZoneHandler(mockZoneSvc, mockShowSvc)
	router := setupShowZoneRouter(handler)

	// Add test zone
	now := time.Now()
	mockZoneSvc.AddZone(&domain.ShowZone{
		ID:             "zone-1",
		ShowID:         "show-1",
		Name:           "Original Name",
		Price:          100.00,
		TotalSeats:     50,
		AvailableSeats: 50,
		SortOrder:      1,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	tests := []struct {
		name       string
		id         string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "valid update",
			id:   "zone-1",
			body: map[string]interface{}{
				"name": "Updated Name",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "update price",
			id:   "zone-1",
			body: map[string]interface{}{
				"price": 150.00,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty update",
			id:         "zone-1",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "non-existent zone",
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
			req, _ := http.NewRequest(http.MethodPut, "/zones/"+tt.id, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestShowZoneHandler_Delete(t *testing.T) {
	mockZoneSvc := NewMockShowZoneService()
	mockShowSvc := NewMockShowServiceForZone()
	handler := NewShowZoneHandler(mockZoneSvc, mockShowSvc)
	router := setupShowZoneRouter(handler)

	// Add test zone
	now := time.Now()
	mockZoneSvc.AddZone(&domain.ShowZone{
		ID:             "zone-1",
		ShowID:         "show-1",
		Name:           "Test Zone",
		Price:          100.00,
		TotalSeats:     50,
		AvailableSeats: 50,
		SortOrder:      1,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "delete existing zone",
			id:         "zone-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete non-existent zone",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodDelete, "/zones/"+tt.id, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Code)
			}
		})
	}
}
