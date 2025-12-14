package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
)

// QueueReleaseWorkerConfig holds configuration for the queue release worker
type QueueReleaseWorkerConfig struct {
	// ReleaseInterval is the time between release batches (default: 1 second)
	ReleaseInterval time.Duration
	// JWTSecret is the secret for signing queue pass JWTs
	JWTSecret string
	// DefaultMaxConcurrent is used when event config is not set (default: 500)
	DefaultMaxConcurrent int
	// DefaultQueuePassTTL is used when event config is not set (default: 5 minutes)
	DefaultQueuePassTTL time.Duration
}

// DefaultQueueReleaseWorkerConfig returns default configuration
// Note: JWTSecret must be set before use
func DefaultQueueReleaseWorkerConfig() *QueueReleaseWorkerConfig {
	return &QueueReleaseWorkerConfig{
		ReleaseInterval:      1 * time.Second,
		JWTSecret:            "", // Must be provided via environment or config
		DefaultMaxConcurrent: domain.DefaultMaxConcurrentBookings,
		DefaultQueuePassTTL:  time.Duration(domain.DefaultQueuePassTTLMinutes) * time.Minute,
	}
}

// ReleasedUser represents a user that has been released from the queue
type ReleasedUser struct {
	UserID           string
	EventID          string
	QueuePass        string
	QueuePassExpires time.Time
}

// QueueReleaseWorker releases users from the virtual queue in batches
type QueueReleaseWorker struct {
	config    *QueueReleaseWorkerConfig
	queueRepo repository.QueueRepository
	log       *logger.Logger

	// Metrics
	mu               sync.Mutex
	totalReleased    int64
	lastReleaseTime  time.Time
	lastReleaseCount int

	// Cache for event configs (to reduce Redis calls)
	configCache     map[string]*repository.EventQueueConfig
	configCacheMu   sync.RWMutex
	configCacheTTL  time.Duration
	configCacheTime map[string]time.Time
}

// NewQueueReleaseWorker creates a new queue release worker
func NewQueueReleaseWorker(
	cfg *QueueReleaseWorkerConfig,
	queueRepo repository.QueueRepository,
	log *logger.Logger,
) *QueueReleaseWorker {
	if cfg == nil {
		cfg = DefaultQueueReleaseWorkerConfig()
	}
	if cfg.ReleaseInterval <= 0 {
		cfg.ReleaseInterval = 1 * time.Second
	}
	if cfg.JWTSecret == "" {
		panic("QueueReleaseWorkerConfig.JWTSecret is required")
	}
	if cfg.DefaultMaxConcurrent <= 0 {
		cfg.DefaultMaxConcurrent = domain.DefaultMaxConcurrentBookings
	}
	if cfg.DefaultQueuePassTTL <= 0 {
		cfg.DefaultQueuePassTTL = time.Duration(domain.DefaultQueuePassTTLMinutes) * time.Minute
	}

	return &QueueReleaseWorker{
		config:          cfg,
		queueRepo:       queueRepo,
		log:             log,
		configCache:     make(map[string]*repository.EventQueueConfig),
		configCacheTTL:  30 * time.Second, // Cache config for 30 seconds
		configCacheTime: make(map[string]time.Time),
	}
}

// Start begins the continuous queue release process
func (w *QueueReleaseWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.config.ReleaseInterval)
	defer ticker.Stop()

	w.log.Info(fmt.Sprintf("Queue release worker started (default max concurrent: %d, interval: %v)",
		w.config.DefaultMaxConcurrent, w.config.ReleaseInterval))

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Queue release worker stopping...")
			return
		case <-ticker.C:
			w.processAllQueues(ctx)
		}
	}
}

// processAllQueues processes all active event queues
func (w *QueueReleaseWorker) processAllQueues(ctx context.Context) {
	// Get all event IDs with active queues
	eventIDs, err := w.queueRepo.GetAllQueueEventIDs(ctx)
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to get queue event IDs: %v", err))
		return
	}

	if len(eventIDs) == 0 {
		return
	}

	// Process each queue
	for _, eventID := range eventIDs {
		select {
		case <-ctx.Done():
			return
		default:
			w.releaseFromQueue(ctx, eventID)
		}
	}
}

// releaseFromQueue releases users from a specific event queue using dynamic capacity
func (w *QueueReleaseWorker) releaseFromQueue(ctx context.Context, eventID string) {
	// Get event queue config (cached)
	config := w.getEventConfig(ctx, eventID)
	maxConcurrent := config.MaxConcurrentBookings
	queuePassTTL := time.Duration(config.QueuePassTTLMinutes) * time.Minute

	// Count current active queue passes
	activeCount, err := w.queueRepo.CountActiveQueuePasses(ctx, eventID)
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to count active queue passes for %s: %v", eventID, err))
		return
	}

	// Calculate how many users to release
	releaseCount := int64(maxConcurrent) - activeCount
	if releaseCount <= 0 {
		// At capacity, no need to release
		return
	}

	// Pop users from queue
	userIDs, err := w.queueRepo.PopUsersFromQueue(ctx, eventID, releaseCount)
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to pop users from queue %s: %v", eventID, err))
		return
	}

	if len(userIDs) == 0 {
		return
	}

	w.log.Info(fmt.Sprintf("Releasing %d users from queue %s (active: %d, max: %d)",
		len(userIDs), eventID, activeCount, maxConcurrent))

	// Generate and store queue passes for each user
	releasedCount := 0
	ttlSeconds := int(queuePassTTL.Seconds())
	for _, userID := range userIDs {
		queuePass, expiresAt, err := w.generateQueuePassWithTTL(userID, eventID, queuePassTTL)
		if err != nil {
			w.log.Error(fmt.Sprintf("Failed to generate queue pass for user %s: %v", userID, err))
			continue
		}

		// Store queue pass in Redis
		if err := w.queueRepo.StoreQueuePass(ctx, eventID, userID, queuePass, ttlSeconds); err != nil {
			w.log.Error(fmt.Sprintf("Failed to store queue pass for user %s: %v", userID, err))
			continue
		}

		releasedCount++
		w.log.Debug(fmt.Sprintf("Released user %s from queue %s with pass expiring at %v",
			userID, eventID, expiresAt))
	}

	// Update metrics
	w.mu.Lock()
	w.totalReleased += int64(releasedCount)
	w.lastReleaseTime = time.Now()
	w.lastReleaseCount = releasedCount
	w.mu.Unlock()

	if releasedCount > 0 {
		w.log.Info(fmt.Sprintf("Successfully released %d/%d users from queue %s",
			releasedCount, len(userIDs), eventID))
	}
}

// getEventConfig gets event queue config with caching
func (w *QueueReleaseWorker) getEventConfig(ctx context.Context, eventID string) *repository.EventQueueConfig {
	// Check cache first
	w.configCacheMu.RLock()
	if cached, ok := w.configCache[eventID]; ok {
		if cacheTime, ok := w.configCacheTime[eventID]; ok && time.Since(cacheTime) < w.configCacheTTL {
			w.configCacheMu.RUnlock()
			return cached
		}
	}
	w.configCacheMu.RUnlock()

	// Fetch from Redis
	config, err := w.queueRepo.GetEventQueueConfig(ctx, eventID)
	if err != nil || config == nil {
		// Use defaults
		config = &repository.EventQueueConfig{
			MaxConcurrentBookings: w.config.DefaultMaxConcurrent,
			QueuePassTTLMinutes:   int(w.config.DefaultQueuePassTTL.Minutes()),
		}
	}

	// Apply defaults if values are zero
	if config.MaxConcurrentBookings <= 0 {
		config.MaxConcurrentBookings = w.config.DefaultMaxConcurrent
	}
	if config.QueuePassTTLMinutes <= 0 {
		config.QueuePassTTLMinutes = int(w.config.DefaultQueuePassTTL.Minutes())
	}

	// Update cache
	w.configCacheMu.Lock()
	w.configCache[eventID] = config
	w.configCacheTime[eventID] = time.Now()
	w.configCacheMu.Unlock()

	return config
}

// QueuePassClaims represents the claims for a queue pass JWT
type QueuePassClaims struct {
	UserID  string `json:"user_id"`
	EventID string `json:"event_id"`
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

// generateQueuePassWithTTL generates a signed JWT queue pass token with custom TTL
func (w *QueueReleaseWorker) generateQueuePassWithTTL(userID, eventID string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := QueuePassClaims{
		UserID:  userID,
		EventID: eventID,
		Purpose: "queue_pass",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "queue-release-worker",
			Subject:   userID,
			ID:        generateUniqueID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(w.config.JWTSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign queue pass: %w", err)
	}

	return signedToken, expiresAt, nil
}

// generateQueuePass generates a signed JWT queue pass token with default TTL
func (w *QueueReleaseWorker) generateQueuePass(userID, eventID string) (string, time.Time, error) {
	return w.generateQueuePassWithTTL(userID, eventID, w.config.DefaultQueuePassTTL)
}

// generateUniqueID generates a unique ID for JWT
func generateUniqueID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:32]
	}
	return hex.EncodeToString(bytes)
}

// GetMetrics returns current worker metrics
func (w *QueueReleaseWorker) GetMetrics() (totalReleased int64, lastReleaseTime time.Time, lastReleaseCount int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.totalReleased, w.lastReleaseTime, w.lastReleaseCount
}

// ReleaseFromQueueOnce releases users from a specific queue using dynamic capacity (for testing)
func (w *QueueReleaseWorker) ReleaseFromQueueOnce(ctx context.Context, eventID string) ([]ReleasedUser, error) {
	// Get event queue config (cached)
	config := w.getEventConfig(ctx, eventID)
	maxConcurrent := config.MaxConcurrentBookings
	queuePassTTL := time.Duration(config.QueuePassTTLMinutes) * time.Minute

	// Count current active queue passes
	activeCount, err := w.queueRepo.CountActiveQueuePasses(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to count active queue passes: %w", err)
	}

	// Calculate how many users to release
	releaseCount := int64(maxConcurrent) - activeCount
	if releaseCount <= 0 {
		return []ReleasedUser{}, nil // At capacity
	}

	// Pop users from queue
	userIDs, err := w.queueRepo.PopUsersFromQueue(ctx, eventID, releaseCount)
	if err != nil {
		return nil, fmt.Errorf("failed to pop users from queue: %w", err)
	}

	if len(userIDs) == 0 {
		return []ReleasedUser{}, nil
	}

	var releasedUsers []ReleasedUser
	ttlSeconds := int(queuePassTTL.Seconds())
	for _, userID := range userIDs {
		queuePass, expiresAt, err := w.generateQueuePassWithTTL(userID, eventID, queuePassTTL)
		if err != nil {
			continue
		}

		if err := w.queueRepo.StoreQueuePass(ctx, eventID, userID, queuePass, ttlSeconds); err != nil {
			continue
		}

		releasedUsers = append(releasedUsers, ReleasedUser{
			UserID:           userID,
			EventID:          eventID,
			QueuePass:        queuePass,
			QueuePassExpires: expiresAt,
		})
	}

	// Update metrics
	w.mu.Lock()
	w.totalReleased += int64(len(releasedUsers))
	w.lastReleaseTime = time.Now()
	w.lastReleaseCount = len(releasedUsers)
	w.mu.Unlock()

	return releasedUsers, nil
}

// SetEventConfig sets the queue configuration for an event (for testing or admin API)
func (w *QueueReleaseWorker) SetEventConfig(ctx context.Context, eventID string, maxConcurrent, queuePassTTLMinutes int) error {
	config := &repository.EventQueueConfig{
		MaxConcurrentBookings: maxConcurrent,
		QueuePassTTLMinutes:   queuePassTTLMinutes,
	}
	return w.queueRepo.SetEventQueueConfig(ctx, eventID, config)
}

// GetDefaultMaxConcurrent returns the default max concurrent bookings
func (w *QueueReleaseWorker) GetDefaultMaxConcurrent() int {
	return w.config.DefaultMaxConcurrent
}
