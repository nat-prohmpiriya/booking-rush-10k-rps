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

// ShowZoneService errors
var (
	ErrShowZoneNotFound = errors.New("show zone not found")
)

// showZoneService implements the ShowZoneService interface
type showZoneService struct {
	showZoneRepo repository.ShowZoneRepository
	showRepo     repository.ShowRepository
	zoneSyncer   ZoneSyncer
}

// NewShowZoneService creates a new ShowZoneService
func NewShowZoneService(showZoneRepo repository.ShowZoneRepository, showRepo repository.ShowRepository, zoneSyncer ZoneSyncer) ShowZoneService {
	return &showZoneService{
		showZoneRepo: showZoneRepo,
		showRepo:     showRepo,
		zoneSyncer:   zoneSyncer,
	}
}

// CreateShowZone creates a new zone for a show
func (s *showZoneService) CreateShowZone(ctx context.Context, req *dto.CreateShowZoneRequest) (*domain.ShowZone, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Verify show exists
	show, err := s.showRepo.GetByID(ctx, req.ShowID)
	if err != nil {
		return nil, err
	}
	if show == nil {
		return nil, ErrShowNotFound
	}

	// Create show zone
	now := time.Now()
	zone := &domain.ShowZone{
		ID:             uuid.New().String(),
		ShowID:         req.ShowID,
		Name:           req.Name,
		Price:          req.Price,
		TotalSeats:     req.TotalSeats,
		AvailableSeats: req.TotalSeats, // Initially all seats are available
		Description:    req.Description,
		SortOrder:      req.SortOrder,
		IsActive:       true, // Default to active
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.showZoneRepo.Create(ctx, zone); err != nil {
		return nil, err
	}

	// Only sync to Redis if show is on_sale
	if show.Status == domain.ShowStatusOnSale && zone.IsActive {
		_ = s.zoneSyncer.SyncZone(ctx, zone)
	}

	return zone, nil
}

// GetShowZoneByID retrieves a show zone by ID
func (s *showZoneService) GetShowZoneByID(ctx context.Context, id string) (*domain.ShowZone, error) {
	zone, err := s.showZoneRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if zone == nil {
		return nil, ErrShowZoneNotFound
	}
	return zone, nil
}

// ListZonesByShow lists zones for a show
func (s *showZoneService) ListZonesByShow(ctx context.Context, showID string, filter *dto.ShowZoneListFilter) ([]*domain.ShowZone, int, error) {
	filter.SetDefaults()

	// Verify show exists
	show, err := s.showRepo.GetByID(ctx, showID)
	if err != nil {
		return nil, 0, err
	}
	if show == nil {
		return nil, 0, ErrShowNotFound
	}

	return s.showZoneRepo.GetByShowID(ctx, showID, filter.IsActive, filter.Limit, filter.Offset)
}

// UpdateShowZone updates a show zone
func (s *showZoneService) UpdateShowZone(ctx context.Context, id string, req *dto.UpdateShowZoneRequest) (*domain.ShowZone, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Get existing zone
	zone, err := s.showZoneRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if zone == nil {
		return nil, ErrShowZoneNotFound
	}

	// Update fields
	if req.Name != "" {
		zone.Name = req.Name
	}
	if req.Price != nil {
		zone.Price = *req.Price
	}
	if req.TotalSeats != nil {
		// Calculate the difference and adjust available seats
		diff := *req.TotalSeats - zone.TotalSeats
		zone.TotalSeats = *req.TotalSeats
		zone.AvailableSeats += diff
		if zone.AvailableSeats < 0 {
			zone.AvailableSeats = 0
		}
	}
	if req.Description != "" {
		zone.Description = req.Description
	}
	if req.SortOrder != nil {
		zone.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		zone.IsActive = *req.IsActive
	}

	if err := s.showZoneRepo.Update(ctx, zone); err != nil {
		return nil, err
	}

	// Only sync to Redis if show is on_sale
	show, err := s.showRepo.GetByID(ctx, zone.ShowID)
	if err == nil && show != nil && show.Status == domain.ShowStatusOnSale {
		if zone.IsActive {
			_ = s.zoneSyncer.SyncZone(ctx, zone)
		} else {
			// Remove from Redis if zone is deactivated
			_ = s.zoneSyncer.RemoveZone(ctx, zone.ID)
		}
	}

	return zone, nil
}

// DeleteShowZone soft deletes a show zone
func (s *showZoneService) DeleteShowZone(ctx context.Context, id string) error {
	// Check if zone exists
	zone, err := s.showZoneRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if zone == nil {
		return ErrShowZoneNotFound
	}

	return s.showZoneRepo.Delete(ctx, id)
}

// ListActiveZones lists all active zones for inventory sync
func (s *showZoneService) ListActiveZones(ctx context.Context) ([]*domain.ShowZone, error) {
	return s.showZoneRepo.ListActive(ctx)
}
