package service

import (
	"context"
	"fmt"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// ZoneSyncer handles syncing zone inventory to Redis
type ZoneSyncer interface {
	// SyncByShowID syncs all zones for a show to Redis (when show goes on_sale)
	SyncByShowID(ctx context.Context, showID string) error
	// RemoveByShowID removes all zones for a show from Redis (when show is no longer on_sale)
	RemoveByShowID(ctx context.Context, showID string) error
	// SyncZone syncs a single zone to Redis
	SyncZone(ctx context.Context, zone *domain.ShowZone) error
	// RemoveZone removes a single zone from Redis
	RemoveZone(ctx context.Context, zoneID string) error
}

// zoneSyncer implements ZoneSyncer
type zoneSyncer struct {
	showZoneRepo repository.ShowZoneRepository
	showRepo     repository.ShowRepository
	redis        *redis.Client
}

// NewZoneSyncer creates a new ZoneSyncer
func NewZoneSyncer(showZoneRepo repository.ShowZoneRepository, showRepo repository.ShowRepository, redisClient *redis.Client) ZoneSyncer {
	return &zoneSyncer{
		showZoneRepo: showZoneRepo,
		showRepo:     showRepo,
		redis:        redisClient,
	}
}

// SyncByShowID syncs all zones for a show to Redis
func (s *zoneSyncer) SyncByShowID(ctx context.Context, showID string) error {
	if s.redis == nil {
		return nil
	}

	// Get only active zones for this show
	isActive := true
	zones, _, err := s.showZoneRepo.GetByShowID(ctx, showID, &isActive, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to get zones for show %s: %w", showID, err)
	}

	// Sync each zone to Redis (already filtered to active only)
	for _, zone := range zones {
		if err := s.SyncZone(ctx, zone); err != nil {
			return err
		}
	}

	return nil
}

// RemoveByShowID removes all zones for a show from Redis
func (s *zoneSyncer) RemoveByShowID(ctx context.Context, showID string) error {
	if s.redis == nil {
		return nil
	}

	// Get all zones for this show (including inactive ones to clean up Redis)
	zones, _, err := s.showZoneRepo.GetByShowID(ctx, showID, nil, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to get zones for show %s: %w", showID, err)
	}

	// Remove each zone from Redis
	for _, zone := range zones {
		if err := s.RemoveZone(ctx, zone.ID); err != nil {
			return err
		}
	}

	return nil
}

// SyncZone syncs a single zone to Redis
func (s *zoneSyncer) SyncZone(ctx context.Context, zone *domain.ShowZone) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("zone:availability:%s", zone.ID)
	return s.redis.Set(ctx, key, zone.AvailableSeats, 0).Err()
}

// RemoveZone removes a single zone from Redis
func (s *zoneSyncer) RemoveZone(ctx context.Context, zoneID string) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("zone:availability:%s", zoneID)
	return s.redis.Del(ctx, key).Err()
}
