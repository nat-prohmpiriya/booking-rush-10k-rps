package repository

import (
	"context"
)

// JoinQueueResult represents the result of joining a queue
type JoinQueueResult struct {
	Success      bool
	Position     int64
	TotalInQueue int64
	JoinedAt     float64
	ErrorCode    string
	ErrorMessage string
}

// QueuePositionResult represents the result of getting queue position
type QueuePositionResult struct {
	Position     int64
	TotalInQueue int64
	IsInQueue    bool
}

// QueueRepository defines the interface for Redis-based queue operations
type QueueRepository interface {
	// JoinQueue adds a user to the queue using Sorted Set
	JoinQueue(ctx context.Context, params JoinQueueParams) (*JoinQueueResult, error)

	// GetPosition gets the user's current position in queue
	GetPosition(ctx context.Context, eventID, userID string) (*QueuePositionResult, error)

	// LeaveQueue removes a user from the queue
	LeaveQueue(ctx context.Context, eventID, userID, token string) error

	// GetQueueSize gets the total number of users in queue for an event
	GetQueueSize(ctx context.Context, eventID string) (int64, error)

	// GetUserQueueInfo gets the user's queue info (token, joined_at, etc.)
	GetUserQueueInfo(ctx context.Context, eventID, userID string) (map[string]string, error)

	// StoreQueuePass stores the queue pass token in Redis with TTL
	StoreQueuePass(ctx context.Context, eventID, userID, queuePass string, ttl int) error

	// GetQueuePass retrieves the queue pass for a user (if exists)
	GetQueuePass(ctx context.Context, eventID, userID string) (string, error)

	// ValidateQueuePass validates if the queue pass is valid and not expired
	ValidateQueuePass(ctx context.Context, eventID, userID, queuePass string) (bool, error)

	// DeleteQueuePass deletes the queue pass after successful booking
	DeleteQueuePass(ctx context.Context, eventID, userID string) error

	// PopUsersFromQueue pops the first N users from the queue (for batch release)
	PopUsersFromQueue(ctx context.Context, eventID string, count int64) ([]string, error)

	// GetAllQueueEventIDs returns all event IDs that have active queues
	GetAllQueueEventIDs(ctx context.Context) ([]string, error)

	// RemoveUserFromQueue removes a user from the queue without token verification (for worker use)
	RemoveUserFromQueue(ctx context.Context, eventID, userID string) error

	// CountActiveQueuePasses counts the number of active (non-expired) queue passes for an event
	CountActiveQueuePasses(ctx context.Context, eventID string) (int64, error)

	// GetEventQueueConfig gets the queue configuration for an event from Redis cache
	GetEventQueueConfig(ctx context.Context, eventID string) (*EventQueueConfig, error)

	// SetEventQueueConfig sets the queue configuration for an event in Redis cache
	SetEventQueueConfig(ctx context.Context, eventID string, config *EventQueueConfig) error
}

// EventQueueConfig holds queue configuration for an event
type EventQueueConfig struct {
	MaxConcurrentBookings int `json:"max_concurrent_bookings"`
	QueuePassTTLMinutes   int `json:"queue_pass_ttl_minutes"`
}

// JoinQueueParams contains parameters for joining a queue
type JoinQueueParams struct {
	UserID       string
	EventID      string
	Token        string
	TTLSeconds   int
	MaxQueueSize int64
}
