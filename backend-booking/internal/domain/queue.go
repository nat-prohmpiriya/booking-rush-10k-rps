package domain

import "time"

// QueueEntry represents a user's position in the virtual queue
type QueueEntry struct {
	UserID    string    `json:"user_id"`
	EventID   string    `json:"event_id"`
	Position  int64     `json:"position"`
	JoinedAt  time.Time `json:"joined_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Token     string    `json:"token"`
}

// QueueStatus represents the current state of a queue
type QueueStatus struct {
	EventID       string `json:"event_id"`
	TotalInQueue  int64  `json:"total_in_queue"`
	IsOpen        bool   `json:"is_open"`
	EstimatedWait int64  `json:"estimated_wait_seconds"`
}

// NewQueueEntry creates a new queue entry
func NewQueueEntry(userID, eventID, token string, ttlSeconds int64) *QueueEntry {
	now := time.Now()
	return &QueueEntry{
		UserID:    userID,
		EventID:   eventID,
		JoinedAt:  now,
		ExpiresAt: now.Add(time.Duration(ttlSeconds) * time.Second),
		Token:     token,
	}
}

// IsExpired checks if the queue entry has expired
func (q *QueueEntry) IsExpired() bool {
	return time.Now().After(q.ExpiresAt)
}

// Validate validates the queue entry
func (q *QueueEntry) Validate() error {
	if q.UserID == "" {
		return ErrInvalidUserID
	}
	if q.EventID == "" {
		return ErrInvalidEventID
	}
	return nil
}
