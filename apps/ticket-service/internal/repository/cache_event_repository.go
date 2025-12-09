package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/ticket-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

const (
	// Cache key prefixes
	eventDetailKeyPrefix = "event:detail:"
	eventSlugKeyPrefix   = "event:slug:"
	eventListKeyPrefix   = "event:list:"

	// Default TTL for event caches
	eventCacheTTL = 5 * time.Minute
)

// CachedEventRepository wraps EventRepository with Redis caching
type CachedEventRepository struct {
	repo  EventRepository
	cache *redis.Client
}

// NewCachedEventRepository creates a new CachedEventRepository
func NewCachedEventRepository(repo EventRepository, cache *redis.Client) *CachedEventRepository {
	return &CachedEventRepository{
		repo:  repo,
		cache: cache,
	}
}

// Create creates a new event and invalidates list cache
func (r *CachedEventRepository) Create(ctx context.Context, event *domain.Event) error {
	if err := r.repo.Create(ctx, event); err != nil {
		return err
	}
	// Invalidate list caches
	r.invalidateListCaches(ctx)
	return nil
}

// GetByID retrieves an event by ID with caching
func (r *CachedEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	// Try cache first
	cacheKey := eventDetailKeyPrefix + id
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var event domain.Event
		if err := json.Unmarshal([]byte(cached), &event); err == nil {
			return &event, nil
		}
	}

	// Cache miss - get from database
	event, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, nil
	}

	// Store in cache
	r.cacheEvent(ctx, cacheKey, event)

	return event, nil
}

// GetBySlug retrieves an event by slug with caching
func (r *CachedEventRepository) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	// Try cache first
	cacheKey := eventSlugKeyPrefix + slug
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var event domain.Event
		if err := json.Unmarshal([]byte(cached), &event); err == nil {
			return &event, nil
		}
	}

	// Cache miss - get from database
	event, err := r.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, nil
	}

	// Store in cache (both by slug and by ID)
	r.cacheEvent(ctx, cacheKey, event)
	r.cacheEvent(ctx, eventDetailKeyPrefix+event.ID, event)

	return event, nil
}

// GetByTenantID retrieves events by tenant ID (no caching for tenant-specific queries)
func (r *CachedEventRepository) GetByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, error) {
	return r.repo.GetByTenantID(ctx, tenantID, limit, offset)
}

// Update updates an event and invalidates caches
func (r *CachedEventRepository) Update(ctx context.Context, event *domain.Event) error {
	if err := r.repo.Update(ctx, event); err != nil {
		return err
	}

	// Invalidate event caches
	r.invalidateEventCaches(ctx, event.ID, event.Slug)

	return nil
}

// Delete soft deletes an event and invalidates caches
func (r *CachedEventRepository) Delete(ctx context.Context, id string) error {
	// Get event first to know the slug for cache invalidation
	event, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := r.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate caches
	if event != nil {
		r.invalidateEventCaches(ctx, id, event.Slug)
	}

	return nil
}

// ListPublished lists all published events with caching
func (r *CachedEventRepository) ListPublished(ctx context.Context, limit, offset int) ([]*domain.Event, int, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("%spublished:%d:%d", eventListKeyPrefix, limit, offset)
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var result cachedEventList
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result.Events, result.Total, nil
		}
	}

	// Cache miss - get from database
	events, total, err := r.repo.ListPublished(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Store in cache
	r.cacheEventList(ctx, cacheKey, events, total)

	return events, total, nil
}

// List lists events with filters and pagination (cached only for simple queries)
func (r *CachedEventRepository) List(ctx context.Context, filter *EventFilter, limit, offset int) ([]*domain.Event, int, error) {
	// Only cache simple queries without filters or with only status filter
	if filter == nil || (filter.TenantID == "" && filter.VenueID == "" && filter.Search == "") {
		status := ""
		if filter != nil {
			status = filter.Status
		}
		cacheKey := fmt.Sprintf("%sall:%s:%d:%d", eventListKeyPrefix, status, limit, offset)
		cached, err := r.cache.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var result cachedEventList
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
				return result.Events, result.Total, nil
			}
		}

		// Cache miss - get from database
		events, total, err := r.repo.List(ctx, filter, limit, offset)
		if err != nil {
			return nil, 0, err
		}

		// Store in cache
		r.cacheEventList(ctx, cacheKey, events, total)

		return events, total, nil
	}

	// Complex queries bypass cache
	return r.repo.List(ctx, filter, limit, offset)
}

// SlugExists checks if a slug already exists (bypass cache)
func (r *CachedEventRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	return r.repo.SlugExists(ctx, slug)
}

// --- Helper functions ---

type cachedEventList struct {
	Events []*domain.Event `json:"events"`
	Total  int             `json:"total"`
}

func (r *CachedEventRepository) cacheEvent(ctx context.Context, key string, event *domain.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	r.cache.Set(ctx, key, string(data), eventCacheTTL)
}

func (r *CachedEventRepository) cacheEventList(ctx context.Context, key string, events []*domain.Event, total int) {
	data, err := json.Marshal(cachedEventList{Events: events, Total: total})
	if err != nil {
		return
	}
	r.cache.Set(ctx, key, string(data), eventCacheTTL)
}

func (r *CachedEventRepository) invalidateEventCaches(ctx context.Context, id, slug string) {
	// Delete detail cache
	r.cache.Del(ctx, eventDetailKeyPrefix+id)
	// Delete slug cache
	if slug != "" {
		r.cache.Del(ctx, eventSlugKeyPrefix+slug)
	}
	// Invalidate list caches
	r.invalidateListCaches(ctx)
}

func (r *CachedEventRepository) invalidateListCaches(ctx context.Context) {
	// Delete all list caches using pattern
	// Since we can't use KEYS in production, we use specific key deletion
	// In production, you might want to use SCAN or track cached keys
	patterns := []string{
		eventListKeyPrefix + "published:*",
		eventListKeyPrefix + "all:*",
	}

	for _, pattern := range patterns {
		iter := r.cache.Client().Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			r.cache.Del(ctx, iter.Val())
		}
	}
}

// InvalidateAll invalidates all event caches (useful for admin operations)
func (r *CachedEventRepository) InvalidateAll(ctx context.Context) error {
	patterns := []string{
		eventDetailKeyPrefix + "*",
		eventSlugKeyPrefix + "*",
		eventListKeyPrefix + "*",
	}

	for _, pattern := range patterns {
		iter := r.cache.Client().Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			r.cache.Del(ctx, iter.Val())
		}
	}

	return nil
}
