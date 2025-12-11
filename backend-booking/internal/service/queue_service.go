package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
)

// QueueService defines the interface for queue business logic
type QueueService interface {
	// JoinQueue adds a user to the virtual queue for an event
	JoinQueue(ctx context.Context, userID string, req *dto.JoinQueueRequest) (*dto.JoinQueueResponse, error)

	// GetPosition gets the user's current position in queue
	GetPosition(ctx context.Context, userID, eventID string) (*dto.QueuePositionResponse, error)

	// LeaveQueue removes a user from the queue
	LeaveQueue(ctx context.Context, userID string, req *dto.LeaveQueueRequest) (*dto.LeaveQueueResponse, error)

	// GetQueueStatus gets the queue status for an event
	GetQueueStatus(ctx context.Context, eventID string) (*dto.QueueStatusResponse, error)
}

// queueService implements QueueService
type queueService struct {
	queueRepo            repository.QueueRepository
	queueTTL             time.Duration
	maxQueueSize         int64
	estimatedWaitPerUser int64 // seconds per user in queue
}

// QueueServiceConfig contains configuration for queue service
type QueueServiceConfig struct {
	QueueTTL             time.Duration
	MaxQueueSize         int64
	EstimatedWaitPerUser int64
}

// NewQueueService creates a new queue service
func NewQueueService(
	queueRepo repository.QueueRepository,
	cfg *QueueServiceConfig,
) QueueService {
	ttl := 30 * time.Minute
	maxSize := int64(0) // 0 = unlimited
	estimatedWait := int64(3) // 3 seconds per user

	if cfg != nil {
		if cfg.QueueTTL > 0 {
			ttl = cfg.QueueTTL
		}
		if cfg.MaxQueueSize > 0 {
			maxSize = cfg.MaxQueueSize
		}
		if cfg.EstimatedWaitPerUser > 0 {
			estimatedWait = cfg.EstimatedWaitPerUser
		}
	}

	return &queueService{
		queueRepo:            queueRepo,
		queueTTL:             ttl,
		maxQueueSize:         maxSize,
		estimatedWaitPerUser: estimatedWait,
	}
}

// JoinQueue adds a user to the virtual queue for an event
func (s *queueService) JoinQueue(ctx context.Context, userID string, req *dto.JoinQueueRequest) (*dto.JoinQueueResponse, error) {
	// Validate request
	if req == nil {
		return nil, domain.ErrInvalidEventID
	}
	if req.EventID == "" {
		return nil, domain.ErrInvalidEventID
	}
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	// Generate unique queue token
	token := generateQueueToken()

	// Join queue in Redis
	params := repository.JoinQueueParams{
		UserID:       userID,
		EventID:      req.EventID,
		Token:        token,
		TTLSeconds:   int(s.queueTTL.Seconds()),
		MaxQueueSize: s.maxQueueSize,
	}

	result, err := s.queueRepo.JoinQueue(ctx, params)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		switch result.ErrorCode {
		case "ALREADY_IN_QUEUE":
			return nil, domain.ErrAlreadyInQueue
		case "QUEUE_FULL":
			return nil, domain.ErrQueueFull
		default:
			return nil, domain.ErrQueueNotOpen
		}
	}

	// Calculate estimated wait time
	estimatedWait := result.Position * s.estimatedWaitPerUser

	now := time.Now()
	return &dto.JoinQueueResponse{
		Position:      result.Position,
		Token:         token,
		EstimatedWait: estimatedWait,
		JoinedAt:      now,
		ExpiresAt:     now.Add(s.queueTTL),
		Message:       "Successfully joined the queue",
	}, nil
}

// GetPosition gets the user's current position in queue
func (s *queueService) GetPosition(ctx context.Context, userID, eventID string) (*dto.QueuePositionResponse, error) {
	// Validate inputs
	if eventID == "" {
		return nil, domain.ErrInvalidEventID
	}
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	result, err := s.queueRepo.GetPosition(ctx, eventID, userID)
	if err != nil {
		return nil, err
	}

	if !result.IsInQueue {
		return nil, domain.ErrNotInQueue
	}

	// Calculate estimated wait time
	estimatedWait := result.Position * s.estimatedWaitPerUser

	// Check if user is ready (position <= some threshold, e.g., position 1)
	isReady := result.Position <= 1

	// Get expiry info
	userInfo, _ := s.queueRepo.GetUserQueueInfo(ctx, eventID, userID)
	var expiresAt time.Time
	if expires, ok := userInfo["expires_at"]; ok {
		if ts, err := parseTimestamp(expires); err == nil {
			expiresAt = time.Unix(ts, 0)
		}
	}

	return &dto.QueuePositionResponse{
		Position:      result.Position,
		TotalInQueue:  result.TotalInQueue,
		EstimatedWait: estimatedWait,
		IsReady:       isReady,
		ExpiresAt:     expiresAt,
	}, nil
}

// LeaveQueue removes a user from the queue
func (s *queueService) LeaveQueue(ctx context.Context, userID string, req *dto.LeaveQueueRequest) (*dto.LeaveQueueResponse, error) {
	// Validate inputs
	if req == nil {
		return nil, domain.ErrInvalidEventID
	}
	if req.EventID == "" {
		return nil, domain.ErrInvalidEventID
	}
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}
	if req.Token == "" {
		return nil, domain.ErrInvalidQueueToken
	}

	err := s.queueRepo.LeaveQueue(ctx, req.EventID, userID, req.Token)
	if err != nil {
		return nil, err
	}

	return &dto.LeaveQueueResponse{
		Success: true,
		Message: "Successfully left the queue",
	}, nil
}

// GetQueueStatus gets the queue status for an event
func (s *queueService) GetQueueStatus(ctx context.Context, eventID string) (*dto.QueueStatusResponse, error) {
	// Validate input
	if eventID == "" {
		return nil, domain.ErrInvalidEventID
	}

	size, err := s.queueRepo.GetQueueSize(ctx, eventID)
	if err != nil {
		return nil, err
	}

	return &dto.QueueStatusResponse{
		EventID:      eventID,
		TotalInQueue: size,
		IsOpen:       true, // TODO: Check event status from event service
	}, nil
}

// generateQueueToken generates a random queue token
func generateQueueToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:32]
	}
	return hex.EncodeToString(bytes)
}

// parseTimestamp parses a string timestamp to int64
func parseTimestamp(s string) (int64, error) {
	// Try parsing as RFC3339
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t.Unix(), nil
	}
	// Try parsing as unix timestamp
	i, err2 := strconv.ParseInt(s, 10, 64)
	if err2 == nil {
		return i, nil
	}
	return 0, fmt.Errorf("cannot parse timestamp: %s", s)
}
