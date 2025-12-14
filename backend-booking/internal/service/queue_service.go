package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	queuePassTTL         time.Duration
	jwtSecret            string
}

// QueueServiceConfig contains configuration for queue service
type QueueServiceConfig struct {
	QueueTTL             time.Duration
	MaxQueueSize         int64
	EstimatedWaitPerUser int64
	QueuePassTTL         time.Duration // TTL for queue pass token (default: 5 minutes)
	JWTSecret            string        // Secret for signing queue pass JWT
}

// NewQueueService creates a new queue service
func NewQueueService(
	queueRepo repository.QueueRepository,
	cfg *QueueServiceConfig,
) QueueService {
	ttl := 30 * time.Minute
	maxSize := int64(0)       // 0 = unlimited
	estimatedWait := int64(3) // 3 seconds per user
	queuePassTTL := 5 * time.Minute
	jwtSecret := "" // Must be provided via config

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
		if cfg.QueuePassTTL > 0 {
			queuePassTTL = cfg.QueuePassTTL
		}
		jwtSecret = cfg.JWTSecret
	}

	if jwtSecret == "" {
		panic("QueueServiceConfig.JWTSecret is required")
	}

	return &queueService{
		queueRepo:            queueRepo,
		queueTTL:             ttl,
		maxQueueSize:         maxSize,
		estimatedWaitPerUser: estimatedWait,
		queuePassTTL:         queuePassTTL,
		jwtSecret:            jwtSecret,
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

	response := &dto.QueuePositionResponse{
		Position:      result.Position,
		TotalInQueue:  result.TotalInQueue,
		EstimatedWait: estimatedWait,
		IsReady:       isReady,
		ExpiresAt:     expiresAt,
	}

	// Generate queue pass when user is ready (position = 1)
	if isReady {
		queuePass, queuePassExpiresAt, err := s.generateQueuePass(userID, eventID)
		if err != nil {
			// Log error but don't fail the request
			// The user can still see their position
			return response, nil
		}

		// Store queue pass in Redis for validation
		ttlSeconds := int(s.queuePassTTL.Seconds())
		if err := s.queueRepo.StoreQueuePass(ctx, eventID, userID, queuePass, ttlSeconds); err != nil {
			// Log error but don't fail the request
			return response, nil
		}

		response.QueuePass = queuePass
		response.QueuePassExpiresAt = queuePassExpiresAt
	}

	return response, nil
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

// QueuePassClaims represents the claims for a queue pass JWT
type QueuePassClaims struct {
	UserID  string `json:"user_id"`
	EventID string `json:"event_id"`
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

// generateQueuePass generates a signed JWT queue pass token
func (s *queueService) generateQueuePass(userID, eventID string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.queuePassTTL)

	claims := QueuePassClaims{
		UserID:  userID,
		EventID: eventID,
		Purpose: "queue_pass",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "booking-service",
			Subject:   userID,
			ID:        generateQueueToken(), // Unique JWT ID
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign queue pass: %w", err)
	}

	return signedToken, expiresAt, nil
}
