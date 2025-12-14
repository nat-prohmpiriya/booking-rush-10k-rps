package service

import (
	"context"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
)

// MockShowZoneRepository is a mock implementation of ShowZoneRepository
type MockShowZoneRepository struct {
	zones map[string]*domain.ShowZone
}

func NewMockShowZoneRepository() *MockShowZoneRepository {
	return &MockShowZoneRepository{
		zones: make(map[string]*domain.ShowZone),
	}
}

func (m *MockShowZoneRepository) Create(ctx context.Context, zone *domain.ShowZone) error {
	m.zones[zone.ID] = zone
	return nil
}

func (m *MockShowZoneRepository) GetByID(ctx context.Context, id string) (*domain.ShowZone, error) {
	zone, ok := m.zones[id]
	if !ok {
		return nil, nil
	}
	return zone, nil
}

func (m *MockShowZoneRepository) GetByShowID(ctx context.Context, showID string, limit, offset int) ([]*domain.ShowZone, int, error) {
	var zones []*domain.ShowZone
	for _, z := range m.zones {
		if z.ShowID == showID && z.DeletedAt == nil {
			zones = append(zones, z)
		}
	}
	return zones, len(zones), nil
}

func (m *MockShowZoneRepository) Update(ctx context.Context, zone *domain.ShowZone) error {
	if _, ok := m.zones[zone.ID]; !ok {
		return ErrShowZoneNotFound
	}
	m.zones[zone.ID] = zone
	return nil
}

func (m *MockShowZoneRepository) Delete(ctx context.Context, id string) error {
	zone, ok := m.zones[id]
	if !ok {
		return ErrShowZoneNotFound
	}
	now := time.Now()
	zone.DeletedAt = &now
	return nil
}

func (m *MockShowZoneRepository) UpdateAvailableSeats(ctx context.Context, id string, availableSeats int) error {
	zone, ok := m.zones[id]
	if !ok {
		return ErrShowZoneNotFound
	}
	zone.AvailableSeats = availableSeats
	return nil
}

func (m *MockShowZoneRepository) ListActive(ctx context.Context) ([]*domain.ShowZone, error) {
	var zones []*domain.ShowZone
	for _, z := range m.zones {
		if z.IsActive && z.DeletedAt == nil {
			zones = append(zones, z)
		}
	}
	return zones, nil
}

func (m *MockShowZoneRepository) AddZone(zone *domain.ShowZone) {
	m.zones[zone.ID] = zone
}

// MockShowRepoForZone is a mock implementation of ShowRepository for zone tests
type MockShowRepoForZone struct {
	shows map[string]*domain.Show
}

func NewMockShowRepoForZone() *MockShowRepoForZone {
	return &MockShowRepoForZone{
		shows: make(map[string]*domain.Show),
	}
}

func (m *MockShowRepoForZone) Create(ctx context.Context, show *domain.Show) error {
	m.shows[show.ID] = show
	return nil
}

func (m *MockShowRepoForZone) GetByID(ctx context.Context, id string) (*domain.Show, error) {
	show, ok := m.shows[id]
	if !ok {
		return nil, nil
	}
	return show, nil
}

func (m *MockShowRepoForZone) GetByEventID(ctx context.Context, eventID string, limit, offset int) ([]*domain.Show, int, error) {
	var shows []*domain.Show
	for _, s := range m.shows {
		if s.EventID == eventID && s.DeletedAt == nil {
			shows = append(shows, s)
		}
	}
	return shows, len(shows), nil
}

func (m *MockShowRepoForZone) Update(ctx context.Context, show *domain.Show) error {
	return nil
}

func (m *MockShowRepoForZone) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockShowRepoForZone) AddShow(show *domain.Show) {
	m.shows[show.ID] = show
}

func TestShowZoneService_CreateShowZone(t *testing.T) {
	mockZoneRepo := NewMockShowZoneRepository()
	mockShowRepo := NewMockShowRepoForZone()
	svc := NewShowZoneService(mockZoneRepo, mockShowRepo, nil)

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
		req     *dto.CreateShowZoneRequest
		wantErr bool
		errType error
	}{
		{
			name: "valid request",
			req: &dto.CreateShowZoneRequest{
				ShowID:      "show-1",
				Name:        "VIP Zone",
				Price:       100.00,
				TotalSeats:  50,
				Description: "VIP seating area",
				SortOrder:   1,
			},
			wantErr: false,
		},
		{
			name: "show not found",
			req: &dto.CreateShowZoneRequest{
				ShowID:     "non-existent",
				Name:       "VIP Zone",
				Price:      100.00,
				TotalSeats: 50,
			},
			wantErr: true,
			errType: ErrShowNotFound,
		},
		{
			name: "missing name",
			req: &dto.CreateShowZoneRequest{
				ShowID:     "show-1",
				Name:       "",
				Price:      100.00,
				TotalSeats: 50,
			},
			wantErr: true,
		},
		{
			name: "invalid total seats",
			req: &dto.CreateShowZoneRequest{
				ShowID:     "show-1",
				Name:       "VIP Zone",
				Price:      100.00,
				TotalSeats: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone, err := svc.CreateShowZone(context.Background(), tt.req)
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
			if zone.Name != tt.req.Name {
				t.Errorf("expected name %s, got %s", tt.req.Name, zone.Name)
			}
			if zone.TotalSeats != tt.req.TotalSeats {
				t.Errorf("expected total seats %d, got %d", tt.req.TotalSeats, zone.TotalSeats)
			}
			if zone.AvailableSeats != tt.req.TotalSeats {
				t.Errorf("expected available seats %d, got %d", tt.req.TotalSeats, zone.AvailableSeats)
			}
		})
	}
}

func TestShowZoneService_GetShowZoneByID(t *testing.T) {
	mockZoneRepo := NewMockShowZoneRepository()
	mockShowRepo := NewMockShowRepoForZone()
	svc := NewShowZoneService(mockZoneRepo, mockShowRepo, nil)

	// Add test zone
	now := time.Now()
	mockZoneRepo.AddZone(&domain.ShowZone{
		ID:             "zone-1",
		ShowID:         "show-1",
		Name:           "VIP Zone",
		Price:          100.00,
		TotalSeats:     50,
		AvailableSeats: 50,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing zone",
			id:      "zone-1",
			wantErr: false,
		},
		{
			name:    "non-existent zone",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone, err := svc.GetShowZoneByID(context.Background(), tt.id)
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
			if zone.ID != tt.id {
				t.Errorf("expected ID %s, got %s", tt.id, zone.ID)
			}
		})
	}
}

func TestShowZoneService_ListZonesByShow(t *testing.T) {
	mockZoneRepo := NewMockShowZoneRepository()
	mockShowRepo := NewMockShowRepoForZone()
	svc := NewShowZoneService(mockZoneRepo, mockShowRepo, nil)

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

	// Add test zones
	mockZoneRepo.AddZone(&domain.ShowZone{
		ID:             "zone-1",
		ShowID:         "show-1",
		Name:           "VIP Zone",
		Price:          100.00,
		TotalSeats:     50,
		AvailableSeats: 50,
		SortOrder:      1,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	mockZoneRepo.AddZone(&domain.ShowZone{
		ID:             "zone-2",
		ShowID:         "show-1",
		Name:           "Standard Zone",
		Price:          50.00,
		TotalSeats:     100,
		AvailableSeats: 100,
		SortOrder:      2,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	tests := []struct {
		name      string
		showID    string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "existing show with zones",
			showID:    "show-1",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "non-existent show",
			showID:    "non-existent",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &dto.ShowZoneListFilter{Limit: 20, Offset: 0}
			zones, total, err := svc.ListZonesByShow(context.Background(), tt.showID, filter)
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
				t.Errorf("expected %d zones, got %d", tt.wantCount, total)
			}
			if len(zones) != tt.wantCount {
				t.Errorf("expected %d zones in slice, got %d", tt.wantCount, len(zones))
			}
		})
	}
}

func TestShowZoneService_UpdateShowZone(t *testing.T) {
	mockZoneRepo := NewMockShowZoneRepository()
	mockShowRepo := NewMockShowRepoForZone()
	svc := NewShowZoneService(mockZoneRepo, mockShowRepo, nil)

	// Add test zone
	now := time.Now()
	mockZoneRepo.AddZone(&domain.ShowZone{
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

	newPrice := 150.00
	newSeats := 60
	newOrder := 2

	tests := []struct {
		name    string
		id      string
		req     *dto.UpdateShowZoneRequest
		wantErr bool
	}{
		{
			name: "valid update",
			id:   "zone-1",
			req: &dto.UpdateShowZoneRequest{
				Name: "Updated Name",
			},
			wantErr: false,
		},
		{
			name: "update price",
			id:   "zone-1",
			req: &dto.UpdateShowZoneRequest{
				Price: &newPrice,
			},
			wantErr: false,
		},
		{
			name: "update total seats",
			id:   "zone-1",
			req: &dto.UpdateShowZoneRequest{
				TotalSeats: &newSeats,
			},
			wantErr: false,
		},
		{
			name: "update sort order",
			id:   "zone-1",
			req: &dto.UpdateShowZoneRequest{
				SortOrder: &newOrder,
			},
			wantErr: false,
		},
		{
			name: "non-existent zone",
			id:   "non-existent",
			req: &dto.UpdateShowZoneRequest{
				Name: "Updated Name",
			},
			wantErr: true,
		},
		{
			name:    "empty update",
			id:      "zone-1",
			req:     &dto.UpdateShowZoneRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone, err := svc.UpdateShowZone(context.Background(), tt.id, tt.req)
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
			if tt.req.Name != "" && zone.Name != tt.req.Name {
				t.Errorf("expected name %s, got %s", tt.req.Name, zone.Name)
			}
		})
	}
}

func TestShowZoneService_DeleteShowZone(t *testing.T) {
	mockZoneRepo := NewMockShowZoneRepository()
	mockShowRepo := NewMockShowRepoForZone()
	svc := NewShowZoneService(mockZoneRepo, mockShowRepo, nil)

	// Add test zone
	now := time.Now()
	mockZoneRepo.AddZone(&domain.ShowZone{
		ID:             "zone-1",
		ShowID:         "show-1",
		Name:           "Test Zone",
		Price:          100.00,
		TotalSeats:     50,
		AvailableSeats: 50,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "delete existing zone",
			id:      "zone-1",
			wantErr: false,
		},
		{
			name:    "delete non-existent zone",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.DeleteShowZone(context.Background(), tt.id)
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
