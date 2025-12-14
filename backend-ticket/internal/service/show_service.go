package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/repository"
)

// ShowService errors
var (
	ErrShowNotFound = errors.New("show not found")
)

// showService implements the ShowService interface
type showService struct {
	showRepo   repository.ShowRepository
	eventRepo  repository.EventRepository
	zoneSyncer ZoneSyncer
}

// NewShowService creates a new ShowService
func NewShowService(showRepo repository.ShowRepository, eventRepo repository.EventRepository, zoneSyncer ZoneSyncer) ShowService {
	return &showService{
		showRepo:   showRepo,
		eventRepo:  eventRepo,
		zoneSyncer: zoneSyncer,
	}
}

// parseTime parses time string supporting multiple formats (ISO 8601 and time-only)
func parseTime(s string) (time.Time, error) {
	// Try ISO 8601 full datetime formats first
	formats := []string{
		time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05Z", // UTC
		"2006-01-02T15:04:05",  // No timezone
		"15:04:05Z07:00",       // Time with timezone
		"15:04:05",             // Time only
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("invalid time format, expected ISO 8601 (e.g., 2006-01-02T15:04:05+07:00) or time-only (e.g., 15:04:05)")
}

// CreateShow creates a new show for an event
func (s *showService) CreateShow(ctx context.Context, req *dto.CreateShowRequest) (*domain.Show, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Verify event exists
	event, err := s.eventRepo.GetByID(ctx, req.EventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}

	// Parse date and times from string
	showDate, err := time.Parse("2006-01-02", req.ShowDate)
	if err != nil {
		return nil, errors.New("invalid show_date format, expected YYYY-MM-DD")
	}

	startTime, err := parseTime(req.StartTime)
	if err != nil {
		return nil, errors.New("invalid start_time: " + err.Error())
	}

	var endTime time.Time
	if req.EndTime != "" {
		endTime, err = parseTime(req.EndTime)
		if err != nil {
			return nil, errors.New("invalid end_time: " + err.Error())
		}
	}

	var doorsOpenAt *time.Time
	if req.DoorsOpenAt != "" {
		t, err := parseTime(req.DoorsOpenAt)
		if err != nil {
			return nil, errors.New("invalid doors_open_at: " + err.Error())
		}
		doorsOpenAt = &t
	}

	// Create show
	now := time.Now()
	show := &domain.Show{
		ID:          uuid.New().String(),
		EventID:     req.EventID,
		Name:        req.Name,
		ShowDate:    showDate,
		StartTime:   startTime,
		EndTime:     endTime,
		DoorsOpenAt: doorsOpenAt,
		Status:      domain.ShowStatusScheduled,
		SaleStartAt: req.SaleStartAt,
		SaleEndAt:   req.SaleEndAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.showRepo.Create(ctx, show); err != nil {
		return nil, err
	}

	return show, nil
}

// GetShowByID retrieves a show by ID
func (s *showService) GetShowByID(ctx context.Context, id string) (*domain.Show, error) {
	show, err := s.showRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if show == nil {
		return nil, ErrShowNotFound
	}
	return show, nil
}

// ListShowsByEvent lists shows for an event
func (s *showService) ListShowsByEvent(ctx context.Context, eventID string, filter *dto.ShowListFilter) ([]*domain.Show, int, error) {
	if filter == nil {
		filter = &dto.ShowListFilter{}
	}
	filter.SetDefaults()

	// Verify event exists
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, 0, err
	}
	if event == nil {
		return nil, 0, ErrEventNotFound
	}

	return s.showRepo.GetByEventID(ctx, eventID, filter.Limit, filter.Offset)
}

// UpdateShow updates a show
func (s *showService) UpdateShow(ctx context.Context, id string, req *dto.UpdateShowRequest) (*domain.Show, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Get existing show
	show, err := s.showRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if show == nil {
		return nil, ErrShowNotFound
	}

	// Track status change for zone sync
	oldStatus := show.Status

	// Update fields
	if req.Name != "" {
		show.Name = req.Name
	}
	if req.ShowDate != "" {
		showDate, err := time.Parse("2006-01-02", req.ShowDate)
		if err != nil {
			return nil, errors.New("invalid show_date format, expected YYYY-MM-DD")
		}
		show.ShowDate = showDate
	}
	if req.StartTime != "" {
		startTime, err := parseTime(req.StartTime)
		if err != nil {
			return nil, errors.New("invalid start_time: " + err.Error())
		}
		show.StartTime = startTime
	}
	if req.EndTime != "" {
		endTime, err := parseTime(req.EndTime)
		if err != nil {
			return nil, errors.New("invalid end_time: " + err.Error())
		}
		show.EndTime = endTime
	}
	if req.DoorsOpenAt != "" {
		t, err := parseTime(req.DoorsOpenAt)
		if err != nil {
			return nil, errors.New("invalid doors_open_at: " + err.Error())
		}
		show.DoorsOpenAt = &t
	}
	if req.Status != "" {
		show.Status = req.Status
	}
	if req.SaleStartAt != nil {
		show.SaleStartAt = req.SaleStartAt
	}
	if req.SaleEndAt != nil {
		show.SaleEndAt = req.SaleEndAt
	}

	if err := s.showRepo.Update(ctx, show); err != nil {
		return nil, err
	}

	// Sync zones to Redis when status changes to on_sale
	if s.zoneSyncer != nil {
		if oldStatus != domain.ShowStatusOnSale && show.Status == domain.ShowStatusOnSale {
			// Show just went on_sale - sync all zones to Redis
			_ = s.zoneSyncer.SyncByShowID(ctx, show.ID)
		} else if oldStatus == domain.ShowStatusOnSale && show.Status != domain.ShowStatusOnSale {
			// Show is no longer on_sale - remove zones from Redis
			_ = s.zoneSyncer.RemoveByShowID(ctx, show.ID)
		}
	}

	return show, nil
}

// DeleteShow soft deletes a show
func (s *showService) DeleteShow(ctx context.Context, id string) error {
	// Check if show exists
	show, err := s.showRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if show == nil {
		return ErrShowNotFound
	}

	return s.showRepo.Delete(ctx, id)
}
