// +build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/ticket-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

func skipIfNoRedis(t *testing.T) *redis.Client {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=true to run")
	}

	host := os.Getenv("TEST_REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	password := os.Getenv("TEST_REDIS_PASSWORD")

	ctx := context.Background()
	cfg := &redis.Config{
		Host:          host,
		Port:          6379,
		Password:      password,
		DB:            1, // Use DB 1 for tests
		PoolSize:      10,
		MinIdleConns:  2,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		MaxRetries:    3,
		RetryInterval: time.Second,
	}

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Skipf("Skipping integration test - Redis not available: %v", err)
	}

	return client
}

func TestCachedEventRepository_Integration_GetByID(t *testing.T) {
	redisClient := skipIfNoRedis(t)
	defer redisClient.Close()

	mockRepo := NewMockEventRepository()
	cachedRepo := NewCachedEventRepository(mockRepo, redisClient)

	// Add test event
	now := time.Now()
	event := &domain.Event{
		ID:        "int-test-event-1",
		Name:      "Integration Test Event",
		Slug:      "int-test-event",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.AddEvent(event)

	ctx := context.Background()

	// Clear cache first
	cachedRepo.InvalidateAll(ctx)
	mockRepo.ResetCounts()

	// First call - cache miss, hits database
	result, err := cachedRepo.GetByID(ctx, "int-test-event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}
	if result.ID != "int-test-event-1" {
		t.Errorf("expected ID 'int-test-event-1', got '%s'", result.ID)
	}
	if mockRepo.getByIDCount != 1 {
		t.Errorf("expected database hit count 1, got %d", mockRepo.getByIDCount)
	}

	// Second call - cache hit, should NOT hit database
	result, err = cachedRepo.GetByID(ctx, "int-test-event-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}
	if mockRepo.getByIDCount != 1 {
		t.Errorf("expected database hit count still 1 (cache hit), got %d", mockRepo.getByIDCount)
	}

	// Cleanup
	cachedRepo.InvalidateAll(ctx)
}

func TestCachedEventRepository_Integration_ListPublished(t *testing.T) {
	redisClient := skipIfNoRedis(t)
	defer redisClient.Close()

	mockRepo := NewMockEventRepository()
	cachedRepo := NewCachedEventRepository(mockRepo, redisClient)

	// Add test events
	now := time.Now()
	mockRepo.AddEvent(&domain.Event{
		ID:        "int-test-event-1",
		Name:      "Event 1",
		Slug:      "int-event-1",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	})
	mockRepo.AddEvent(&domain.Event{
		ID:        "int-test-event-2",
		Name:      "Event 2",
		Slug:      "int-event-2",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(48 * time.Hour),
		EndTime:   now.Add(50 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	})

	ctx := context.Background()

	// Clear cache first
	cachedRepo.InvalidateAll(ctx)
	mockRepo.ResetCounts()

	// First call - cache miss
	events, total, err := cachedRepo.ListPublished(ctx, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if mockRepo.listCount != 1 {
		t.Errorf("expected database hit count 1, got %d", mockRepo.listCount)
	}

	// Second call - cache hit
	events, total, err = cachedRepo.ListPublished(ctx, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	if mockRepo.listCount != 1 {
		t.Errorf("expected database hit count still 1 (cache hit), got %d", mockRepo.listCount)
	}

	// Cleanup
	cachedRepo.InvalidateAll(ctx)
}

func TestCachedEventRepository_Integration_CacheInvalidation(t *testing.T) {
	redisClient := skipIfNoRedis(t)
	defer redisClient.Close()

	mockRepo := NewMockEventRepository()
	cachedRepo := NewCachedEventRepository(mockRepo, redisClient)

	// Add test event
	now := time.Now()
	event := &domain.Event{
		ID:        "int-test-event-inv",
		Name:      "Original Name",
		Slug:      "int-test-event-inv",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.AddEvent(event)

	ctx := context.Background()

	// Clear cache first
	cachedRepo.InvalidateAll(ctx)
	mockRepo.ResetCounts()

	// First call - cache miss
	result, err := cachedRepo.GetByID(ctx, "int-test-event-inv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Original Name" {
		t.Errorf("expected 'Original Name', got '%s'", result.Name)
	}

	// Update via cached repo (should invalidate cache)
	event.Name = "Updated Name"
	err = cachedRepo.Update(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get again - should get fresh data (cache was invalidated)
	result, err = cachedRepo.GetByID(ctx, "int-test-event-inv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Name" {
		t.Errorf("expected 'Updated Name', got '%s'", result.Name)
	}

	// Cleanup
	cachedRepo.InvalidateAll(ctx)
}

func TestCachedEventRepository_Integration_GetBySlug(t *testing.T) {
	redisClient := skipIfNoRedis(t)
	defer redisClient.Close()

	mockRepo := NewMockEventRepository()
	cachedRepo := NewCachedEventRepository(mockRepo, redisClient)

	// Add test event
	now := time.Now()
	event := &domain.Event{
		ID:        "int-test-slug-1",
		Name:      "Slug Test Event",
		Slug:      "int-slug-test",
		Status:    domain.EventStatusPublished,
		TenantID:  "tenant-1",
		StartTime: now.Add(24 * time.Hour),
		EndTime:   now.Add(26 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.AddEvent(event)

	ctx := context.Background()

	// Clear cache first
	cachedRepo.InvalidateAll(ctx)

	// First call - cache miss
	result, err := cachedRepo.GetBySlug(ctx, "int-slug-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}
	if result.Slug != "int-slug-test" {
		t.Errorf("expected slug 'int-slug-test', got '%s'", result.Slug)
	}

	// Second call - should hit cache
	result, err = cachedRepo.GetBySlug(ctx, "int-slug-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}

	// Also verify that GetByID can use the cache populated by GetBySlug
	mockRepo.ResetCounts()
	result, err = cachedRepo.GetByID(ctx, "int-test-slug-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected event, got nil")
	}
	// Since GetBySlug also populates the ID-based cache, this should be a cache hit
	if mockRepo.getByIDCount != 0 {
		t.Errorf("expected database hit count 0 (cache hit from slug lookup), got %d", mockRepo.getByIDCount)
	}

	// Cleanup
	cachedRepo.InvalidateAll(ctx)
}
