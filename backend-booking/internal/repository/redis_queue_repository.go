package repository

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

//go:embed scripts/join_queue.lua
var joinQueueScript string

// Script name for caching
const scriptJoinQueue = "join_queue"

// RedisQueueRepository implements QueueRepository using Redis
type RedisQueueRepository struct {
	client *pkgredis.Client
}

// NewRedisQueueRepository creates a new RedisQueueRepository
func NewRedisQueueRepository(client *pkgredis.Client) *RedisQueueRepository {
	return &RedisQueueRepository{client: client}
}

// LoadScripts loads all queue Lua scripts into Redis
func (r *RedisQueueRepository) LoadScripts(ctx context.Context) error {
	scripts := map[string]string{
		scriptJoinQueue: joinQueueScript,
	}

	for name, script := range scripts {
		if _, err := r.client.LoadScript(ctx, name, script); err != nil {
			return fmt.Errorf("failed to load script %s: %w", name, err)
		}
	}

	return nil
}

// JoinQueue adds a user to the queue using Sorted Set
func (r *RedisQueueRepository) JoinQueue(ctx context.Context, params JoinQueueParams) (*JoinQueueResult, error) {
	// Build Redis keys
	queueKey := fmt.Sprintf("queue:%s", params.EventID)
	userQueueKey := fmt.Sprintf("queue:user:%s:%s", params.EventID, params.UserID)

	keys := []string{queueKey, userQueueKey}
	args := []interface{}{
		params.UserID,       // ARGV[1]: user_id
		params.EventID,      // ARGV[2]: event_id
		params.Token,        // ARGV[3]: token
		params.TTLSeconds,   // ARGV[4]: ttl_seconds
		params.MaxQueueSize, // ARGV[5]: max_queue_size
	}

	result := r.client.EvalWithFallback(ctx, scriptJoinQueue, joinQueueScript, keys, args...)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to execute join_queue script: %w", result.Err())
	}

	// Parse result
	values, err := result.Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	if len(values) < 3 {
		return nil, fmt.Errorf("unexpected script result length: %d", len(values))
	}

	success, _ := toInt64(values[0])
	if success == 1 {
		position, _ := toInt64(values[1])
		totalInQueue, _ := toInt64(values[2])
		joinedAt, _ := toFloat64(values[3])
		return &JoinQueueResult{
			Success:      true,
			Position:     position,
			TotalInQueue: totalInQueue,
			JoinedAt:     joinedAt,
		}, nil
	}

	// Error case
	errorCode, _ := values[1].(string)
	errorMessage, _ := values[2].(string)
	return &JoinQueueResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// GetPosition gets the user's current position in queue
func (r *RedisQueueRepository) GetPosition(ctx context.Context, eventID, userID string) (*QueuePositionResult, error) {
	queueKey := fmt.Sprintf("queue:%s", eventID)

	// Get user's rank in sorted set (0-indexed)
	rank, err := r.client.ZRank(ctx, queueKey, userID).Result()
	if err != nil {
		// User not in queue
		if err.Error() == "redis: nil" {
			return &QueuePositionResult{
				Position:     0,
				TotalInQueue: 0,
				IsInQueue:    false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get queue position: %w", err)
	}

	// Get total count
	total, err := r.client.ZCard(ctx, queueKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue size: %w", err)
	}

	return &QueuePositionResult{
		Position:     rank + 1, // Convert to 1-indexed
		TotalInQueue: total,
		IsInQueue:    true,
	}, nil
}

// LeaveQueue removes a user from the queue
func (r *RedisQueueRepository) LeaveQueue(ctx context.Context, eventID, userID, token string) error {
	// First verify the token
	userQueueKey := fmt.Sprintf("queue:user:%s:%s", eventID, userID)
	storedToken, err := r.client.HGet(ctx, userQueueKey, "token").Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return domain.ErrNotInQueue
		}
		return fmt.Errorf("failed to get user queue info: %w", err)
	}

	if storedToken != token {
		return domain.ErrInvalidQueueToken
	}

	// Remove from sorted set
	queueKey := fmt.Sprintf("queue:%s", eventID)
	removed, err := r.client.ZRem(ctx, queueKey, userID).Result()
	if err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}

	if removed == 0 {
		return domain.ErrNotInQueue
	}

	// Remove user queue info
	r.client.Del(ctx, userQueueKey)

	return nil
}

// GetQueueSize gets the total number of users in queue for an event
func (r *RedisQueueRepository) GetQueueSize(ctx context.Context, eventID string) (int64, error) {
	queueKey := fmt.Sprintf("queue:%s", eventID)
	count, err := r.client.ZCard(ctx, queueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}
	return count, nil
}

// GetUserQueueInfo gets the user's queue info (token, joined_at, etc.)
func (r *RedisQueueRepository) GetUserQueueInfo(ctx context.Context, eventID, userID string) (map[string]string, error) {
	userQueueKey := fmt.Sprintf("queue:user:%s:%s", eventID, userID)
	result, err := r.client.HGetAll(ctx, userQueueKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user queue info: %w", err)
	}
	return result, nil
}

// Helper function to convert interface{} to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int64:
		return float64(val), true
	case int:
		return float64(val), true
	case string:
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// Ensure RedisQueueRepository implements QueueRepository
var _ QueueRepository = (*RedisQueueRepository)(nil)
