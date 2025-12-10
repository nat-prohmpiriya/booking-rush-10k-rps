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
	showRepo  repository.ShowRepository
	eventRepo repository.EventRepository
}

// NewShowService creates a new ShowService
func NewShowService(showRepo repository.ShowRepository, eventRepo repository.EventRepository) ShowService {
	return &showService{
		showRepo:  showRepo,
		eventRepo: eventRepo,
	}
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

	// Create show
	now := time.Now()
	show := &domain.Show{
		ID:        uuid.New().String(),
		EventID:   req.EventID,
		Name:      req.Name,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Status:    domain.ShowStatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
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

	// Update fields
	if req.Name != "" {
		show.Name = req.Name
	}
	if !req.StartTime.IsZero() {
		show.StartTime = req.StartTime
	}
	if !req.EndTime.IsZero() {
		show.EndTime = req.EndTime
	}
	if req.Status != "" {
		show.Status = req.Status
	}

	if err := s.showRepo.Update(ctx, show); err != nil {
		return nil, err
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
