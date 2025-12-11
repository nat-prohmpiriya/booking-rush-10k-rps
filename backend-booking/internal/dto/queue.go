package dto

import "time"

// JoinQueueRequest represents request to join the queue
type JoinQueueRequest struct {
	EventID string `json:"event_id" binding:"required"`
}

// JoinQueueResponse represents response after joining the queue
type JoinQueueResponse struct {
	Position      int64     `json:"position"`
	Token         string    `json:"token"`
	EstimatedWait int64     `json:"estimated_wait_seconds"`
	JoinedAt      time.Time `json:"joined_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Message       string    `json:"message,omitempty"`
}

// QueuePositionResponse represents current queue position
type QueuePositionResponse struct {
	Position      int64     `json:"position"`
	TotalInQueue  int64     `json:"total_in_queue"`
	EstimatedWait int64     `json:"estimated_wait_seconds"`
	IsReady       bool      `json:"is_ready"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}

// QueueStatusResponse represents queue status for an event
type QueueStatusResponse struct {
	EventID      string `json:"event_id"`
	TotalInQueue int64  `json:"total_in_queue"`
	IsOpen       bool   `json:"is_open"`
}

// LeaveQueueRequest represents request to leave the queue
type LeaveQueueRequest struct {
	EventID string `json:"event_id" binding:"required"`
	Token   string `json:"token" binding:"required"`
}

// LeaveQueueResponse represents response after leaving the queue
type LeaveQueueResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
