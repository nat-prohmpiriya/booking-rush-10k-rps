package repository

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.queue.join")
	defer span.End()

	span.SetAttributes(
		attribute.String("event_id", params.EventID),
		attribute.String("user_id", params.UserID),
	)

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
		span.RecordError(result.Err())
		span.SetStatus(codes.Error, result.Err().Error())
		return nil, fmt.Errorf("failed to execute join_queue script: %w", result.Err())
	}

	// Parse result
	values, err := result.Slice()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	if len(values) < 3 {
		span.SetStatus(codes.Error, "unexpected result length")
		return nil, fmt.Errorf("unexpected script result length: %d", len(values))
	}

	success, _ := toInt64(values[0])
	if success == 1 {
		position, _ := toInt64(values[1])
		totalInQueue, _ := toInt64(values[2])
		joinedAt, _ := toFloat64(values[3])
		span.SetAttributes(
			attribute.Int64("position", position),
			attribute.Int64("total_in_queue", totalInQueue),
		)
		span.SetStatus(codes.Ok, "")
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
	span.SetAttributes(attribute.String("error_code", errorCode))
	span.SetStatus(codes.Error, errorCode)
	return &JoinQueueResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// GetPosition gets the user's current position in queue
func (r *RedisQueueRepository) GetPosition(ctx context.Context, eventID, userID string) (*QueuePositionResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.queue.get_position")
	defer span.End()

	span.SetAttributes(
		attribute.String("event_id", eventID),
		attribute.String("user_id", userID),
	)

	queueKey := fmt.Sprintf("queue:%s", eventID)

	// Get user's rank in sorted set (0-indexed)
	rank, err := r.client.ZRank(ctx, queueKey, userID).Result()
	if err != nil {
		// User not in queue
		if err.Error() == "redis: nil" {
			span.SetStatus(codes.Ok, "not in queue")
			return &QueuePositionResult{
				Position:     0,
				TotalInQueue: 0,
				IsInQueue:    false,
			}, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get queue position: %w", err)
	}

	// Get total count
	total, err := r.client.ZCard(ctx, queueKey).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get queue size: %w", err)
	}

	span.SetAttributes(
		attribute.Int64("position", rank+1),
		attribute.Int64("total_in_queue", total),
	)
	span.SetStatus(codes.Ok, "")
	return &QueuePositionResult{
		Position:     rank + 1, // Convert to 1-indexed
		TotalInQueue: total,
		IsInQueue:    true,
	}, nil
}

// LeaveQueue removes a user from the queue
func (r *RedisQueueRepository) LeaveQueue(ctx context.Context, eventID, userID, token string) error {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.queue.leave")
	defer span.End()

	span.SetAttributes(
		attribute.String("event_id", eventID),
		attribute.String("user_id", userID),
	)

	// First verify the token
	userQueueKey := fmt.Sprintf("queue:user:%s:%s", eventID, userID)
	storedToken, err := r.client.HGet(ctx, userQueueKey, "token").Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			span.SetStatus(codes.Error, "not in queue")
			return domain.ErrNotInQueue
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to get user queue info: %w", err)
	}

	if storedToken != token {
		span.SetStatus(codes.Error, "invalid token")
		return domain.ErrInvalidQueueToken
	}

	// Remove from sorted set
	queueKey := fmt.Sprintf("queue:%s", eventID)
	removed, err := r.client.ZRem(ctx, queueKey, userID).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to remove from queue: %w", err)
	}

	if removed == 0 {
		span.SetStatus(codes.Error, "not in queue")
		return domain.ErrNotInQueue
	}

	// Remove user queue info
	r.client.Del(ctx, userQueueKey)

	span.SetStatus(codes.Ok, "")
	return nil
}

// GetQueueSize gets the total number of users in queue for an event
func (r *RedisQueueRepository) GetQueueSize(ctx context.Context, eventID string) (int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.queue.get_size")
	defer span.End()

	span.SetAttributes(attribute.String("event_id", eventID))

	queueKey := fmt.Sprintf("queue:%s", eventID)
	count, err := r.client.ZCard(ctx, queueKey).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}

	span.SetAttributes(attribute.Int64("count", count))
	span.SetStatus(codes.Ok, "")
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

// GetQueuePass retrieves the queue pass for a user (if exists)
func (r *RedisQueueRepository) GetQueuePass(ctx context.Context, eventID, userID string) (string, error) {
	key := fmt.Sprintf("queue:pass:%s:%s", eventID, userID)
	queuePass, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return "", nil // No queue pass found
		}
		return "", fmt.Errorf("failed to get queue pass: %w", err)
	}
	return queuePass, nil
}

// StoreQueuePass stores the queue pass token in Redis with TTL
func (r *RedisQueueRepository) StoreQueuePass(ctx context.Context, eventID, userID, queuePass string, ttl int) error {
	key := fmt.Sprintf("queue:pass:%s:%s", eventID, userID)
	ttlDuration := time.Duration(ttl) * time.Second
	err := r.client.Set(ctx, key, queuePass, ttlDuration).Err()
	if err != nil {
		return fmt.Errorf("failed to store queue pass: %w", err)
	}

	return nil
}

// ValidateQueuePass validates if the queue pass is valid and not expired
func (r *RedisQueueRepository) ValidateQueuePass(ctx context.Context, eventID, userID, queuePass string) (bool, error) {
	key := fmt.Sprintf("queue:pass:%s:%s", eventID, userID)
	storedPass, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil // No queue pass found or expired
		}
		return false, fmt.Errorf("failed to get queue pass: %w", err)
	}

	return storedPass == queuePass, nil
}

// DeleteQueuePass deletes the queue pass after successful booking
func (r *RedisQueueRepository) DeleteQueuePass(ctx context.Context, eventID, userID string) error {
	key := fmt.Sprintf("queue:pass:%s:%s", eventID, userID)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete queue pass: %w", err)
	}
	return nil
}

// PopUsersFromQueue pops the first N users from the queue (lowest scores = earliest joined)
func (r *RedisQueueRepository) PopUsersFromQueue(ctx context.Context, eventID string, count int64) ([]string, error) {
	queueKey := fmt.Sprintf("queue:%s", eventID)

	// Get users with lowest scores (earliest joined)
	result, err := r.client.ZRange(ctx, queueKey, 0, count-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users from queue: %w", err)
	}

	if len(result) == 0 {
		return []string{}, nil
	}

	// Remove the users from the sorted set
	if _, err := r.client.ZRem(ctx, queueKey, stringSliceToInterface(result)...).Result(); err != nil {
		return nil, fmt.Errorf("failed to remove users from queue: %w", err)
	}

	// Clean up user queue info for each user
	for _, userID := range result {
		userQueueKey := fmt.Sprintf("queue:user:%s:%s", eventID, userID)
		r.client.Del(ctx, userQueueKey)
	}

	return result, nil
}

// GetAllQueueEventIDs returns all event IDs that have active queues
func (r *RedisQueueRepository) GetAllQueueEventIDs(ctx context.Context) ([]string, error) {
	// Scan for all queue keys matching pattern "queue:*"
	// But exclude user-specific keys "queue:user:*" and "queue:pass:*"
	var eventIDs []string
	var cursor uint64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, "queue:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan queue keys: %w", err)
		}

		for _, key := range keys {
			// Skip user-specific keys
			if len(key) > 11 && key[6:10] == "user" {
				continue
			}
			if len(key) > 11 && key[6:10] == "pass" {
				continue
			}
			// Extract event ID from "queue:{eventID}"
			if len(key) > 6 {
				eventID := key[6:] // Remove "queue:" prefix
				eventIDs = append(eventIDs, eventID)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return eventIDs, nil
}

// RemoveUserFromQueue removes a user from the queue without token verification
func (r *RedisQueueRepository) RemoveUserFromQueue(ctx context.Context, eventID, userID string) error {
	queueKey := fmt.Sprintf("queue:%s", eventID)
	userQueueKey := fmt.Sprintf("queue:user:%s:%s", eventID, userID)

	// Remove from sorted set
	if _, err := r.client.ZRem(ctx, queueKey, userID).Result(); err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}

	// Remove user queue info
	r.client.Del(ctx, userQueueKey)

	return nil
}

// Helper function to convert []string to []interface{} for ZRem
func stringSliceToInterface(s []string) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

// CountActiveQueuePasses counts active queue passes for an event using SCAN
func (r *RedisQueueRepository) CountActiveQueuePasses(ctx context.Context, eventID string) (int64, error) {
	pattern := fmt.Sprintf("queue:pass:%s:*", eventID)
	var count int64
	var cursor uint64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to scan queue passes: %w", err)
		}

		count += int64(len(keys))
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	return count, nil
}

// GetEventQueueConfig gets the queue configuration for an event from Redis cache
func (r *RedisQueueRepository) GetEventQueueConfig(ctx context.Context, eventID string) (*EventQueueConfig, error) {
	key := fmt.Sprintf("queue:config:%s", eventID)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get event queue config: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // No config found, use defaults
	}

	config := &EventQueueConfig{}
	if val, ok := result["max_concurrent_bookings"]; ok {
		fmt.Sscanf(val, "%d", &config.MaxConcurrentBookings)
	}
	if val, ok := result["queue_pass_ttl_minutes"]; ok {
		fmt.Sscanf(val, "%d", &config.QueuePassTTLMinutes)
	}

	return config, nil
}

// SetEventQueueConfig sets the queue configuration for an event in Redis cache
func (r *RedisQueueRepository) SetEventQueueConfig(ctx context.Context, eventID string, config *EventQueueConfig) error {
	key := fmt.Sprintf("queue:config:%s", eventID)
	err := r.client.HSet(ctx, key,
		"max_concurrent_bookings", config.MaxConcurrentBookings,
		"queue_pass_ttl_minutes", config.QueuePassTTLMinutes,
	).Err()
	if err != nil {
		return fmt.Errorf("failed to set event queue config: %w", err)
	}
	return nil
}

// Ensure RedisQueueRepository implements QueueRepository
var _ QueueRepository = (*RedisQueueRepository)(nil)
