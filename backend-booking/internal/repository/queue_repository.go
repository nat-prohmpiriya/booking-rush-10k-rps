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
}

// JoinQueueParams contains parameters for joining a queue
type JoinQueueParams struct {
	UserID       string
	EventID      string
	Token        string
	TTLSeconds   int
	MaxQueueSize int64
}
