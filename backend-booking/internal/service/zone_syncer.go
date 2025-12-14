package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"golang.org/x/sync/singleflight"
)

// ZoneInfo represents zone data fetched from ticket service
type ZoneInfo struct {
	ID             string  `json:"id"`
	ShowID         string  `json:"show_id"`
	Name           string  `json:"name"`
	Price          float64 `json:"price"`
	TotalSeats     int64   `json:"total_seats"`
	AvailableSeats int64   `json:"available_seats"`
	IsActive       bool    `json:"is_active"`
}

// ZoneFetcher fetches zone data from ticket service
type ZoneFetcher interface {
	// FetchZone fetches zone data by ID from ticket service
	FetchZone(ctx context.Context, zoneID string) (*ZoneInfo, error)
}

// ZoneSyncer handles syncing zone data to Redis with single-flight pattern
type ZoneSyncer interface {
	// SyncZone syncs zone availability to Redis (uses single-flight)
	SyncZone(ctx context.Context, zoneID string) error
	// SyncZoneIfNotExists syncs zone only if it doesn't exist in Redis
	SyncZoneIfNotExists(ctx context.Context, zoneID string) error
}

// HTTPZoneFetcher fetches zone data via HTTP from ticket service
type HTTPZoneFetcher struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPZoneFetcher creates a new HTTP zone fetcher
func NewHTTPZoneFetcher(ticketServiceURL string) *HTTPZoneFetcher {
	return &HTTPZoneFetcher{
		baseURL: ticketServiceURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// FetchZone fetches zone data from ticket service via HTTP
func (f *HTTPZoneFetcher) FetchZone(ctx context.Context, zoneID string) (*ZoneInfo, error) {
	url := fmt.Sprintf("%s/api/v1/zones/%s", f.baseURL, zoneID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch zone: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("zone not found: %s", zoneID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response - backend returns { success: true, data: ZoneInfo }
	var response struct {
		Success bool     `json:"success"`
		Data    ZoneInfo `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned unsuccessful response")
	}

	return &response.Data, nil
}

// DefaultZoneSyncer implements ZoneSyncer with single-flight pattern
type DefaultZoneSyncer struct {
	fetcher         ZoneFetcher
	reservationRepo repository.ReservationRepository
	sfGroup         singleflight.Group
	syncingZones    sync.Map // Track zones currently being synced
}

// NewZoneSyncer creates a new zone syncer
func NewZoneSyncer(fetcher ZoneFetcher, reservationRepo repository.ReservationRepository) *DefaultZoneSyncer {
	return &DefaultZoneSyncer{
		fetcher:         fetcher,
		reservationRepo: reservationRepo,
	}
}

// SyncZone syncs zone availability to Redis using single-flight pattern
// Multiple concurrent calls for the same zoneID will share the same result
func (s *DefaultZoneSyncer) SyncZone(ctx context.Context, zoneID string) error {
	// Use single-flight to prevent multiple concurrent syncs for the same zone
	_, err, _ := s.sfGroup.Do(zoneID, func() (interface{}, error) {
		return nil, s.doSync(ctx, zoneID)
	})

	return err
}

// SyncZoneIfNotExists syncs zone only if it doesn't exist in Redis
func (s *DefaultZoneSyncer) SyncZoneIfNotExists(ctx context.Context, zoneID string) error {
	// Check if zone exists in Redis
	_, err := s.reservationRepo.GetZoneAvailability(ctx, zoneID)
	if err == nil {
		// Zone exists, no need to sync
		return nil
	}

	// Zone doesn't exist, sync it
	return s.SyncZone(ctx, zoneID)
}

// doSync performs the actual sync operation
func (s *DefaultZoneSyncer) doSync(ctx context.Context, zoneID string) error {
	// Fetch zone data from ticket service
	zone, err := s.fetcher.FetchZone(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("failed to fetch zone %s: %w", zoneID, err)
	}

	// Check if zone is active
	if !zone.IsActive {
		return fmt.Errorf("zone %s is not active", zoneID)
	}

	// Sync to Redis
	if err := s.reservationRepo.SetZoneAvailability(ctx, zoneID, zone.AvailableSeats); err != nil {
		return fmt.Errorf("failed to sync zone %s to Redis: %w", zoneID, err)
	}

	return nil
}
