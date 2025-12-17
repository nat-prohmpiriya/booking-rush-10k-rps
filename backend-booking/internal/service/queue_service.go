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
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

	// ValidateQueuePass validates the queue pass JWT and checks Redis
	ValidateQueuePass(ctx context.Context, userID, eventID, queuePass string) error

	// DeleteQueuePass removes the queue pass after successful booking
	DeleteQueuePass(ctx context.Context, userID, eventID string) error
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
	ctx, span := telemetry.StartSpan(ctx, "service.queue.join")
	defer span.End()

	// Validate request
	if req == nil {
		span.SetStatus(codes.Error, "invalid event_id")
		return nil, domain.ErrInvalidEventID
	}
	if req.EventID == "" {
		span.SetStatus(codes.Error, "invalid event_id")
		return nil, domain.ErrInvalidEventID
	}
	if userID == "" {
		span.SetStatus(codes.Error, "invalid user_id")
		return nil, domain.ErrInvalidUserID
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", req.EventID),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if !result.Success {
		switch result.ErrorCode {
		case "ALREADY_IN_QUEUE":
			span.SetStatus(codes.Error, "already in queue")
			return nil, domain.ErrAlreadyInQueue
		case "QUEUE_FULL":
			span.SetStatus(codes.Error, "queue full")
			return nil, domain.ErrQueueFull
		default:
			span.SetStatus(codes.Error, "queue not open")
			return nil, domain.ErrQueueNotOpen
		}
	}

	// Calculate estimated wait time
	estimatedWait := result.Position * s.estimatedWaitPerUser

	now := time.Now()
	span.SetAttributes(attribute.Int64("position", result.Position))
	span.SetStatus(codes.Ok, "")
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
	ctx, span := telemetry.StartSpan(ctx, "service.queue.get_position")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", eventID),
	)

	// Validate inputs
	if eventID == "" {
		span.SetStatus(codes.Error, "invalid event_id")
		return nil, domain.ErrInvalidEventID
	}
	if userID == "" {
		span.SetStatus(codes.Error, "invalid user_id")
		return nil, domain.ErrInvalidUserID
	}

	result, err := s.queueRepo.GetPosition(ctx, eventID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// If user is not in queue, check if they have a valid queue pass
	// (Queue Release Worker may have already released them)
	if !result.IsInQueue {
		// Check if user has a queue pass in Redis
		existingPass, err := s.queueRepo.GetQueuePass(ctx, eventID, userID)
		if err == nil && existingPass != "" {
			// User has a valid queue pass, return it
			// Parse the JWT to get expiry time
			queuePassExpiresAt := time.Now().Add(s.queuePassTTL)

			return &dto.QueuePositionResponse{
				Position:           0, // Already released from queue
				TotalInQueue:       0,
				EstimatedWait:      0,
				IsReady:            true,
				QueuePass:          existingPass,
				QueuePassExpiresAt: queuePassExpiresAt,
			}, nil
		}

		span.SetStatus(codes.Error, "not in queue")
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

	span.SetAttributes(attribute.Int64("position", result.Position))
	span.SetStatus(codes.Ok, "")
	return response, nil
}

// LeaveQueue removes a user from the queue
func (s *queueService) LeaveQueue(ctx context.Context, userID string, req *dto.LeaveQueueRequest) (*dto.LeaveQueueResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.queue.leave")
	defer span.End()

	// Validate inputs
	if req == nil {
		span.SetStatus(codes.Error, "invalid event_id")
		return nil, domain.ErrInvalidEventID
	}
	if req.EventID == "" {
		span.SetStatus(codes.Error, "invalid event_id")
		return nil, domain.ErrInvalidEventID
	}
	if userID == "" {
		span.SetStatus(codes.Error, "invalid user_id")
		return nil, domain.ErrInvalidUserID
	}
	if req.Token == "" {
		span.SetStatus(codes.Error, "invalid queue token")
		return nil, domain.ErrInvalidQueueToken
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", req.EventID),
	)

	err := s.queueRepo.LeaveQueue(ctx, req.EventID, userID, req.Token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &dto.LeaveQueueResponse{
		Success: true,
		Message: "Successfully left the queue",
	}, nil
}

// GetQueueStatus gets the queue status for an event
func (s *queueService) GetQueueStatus(ctx context.Context, eventID string) (*dto.QueueStatusResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.queue.get_status")
	defer span.End()

	span.SetAttributes(attribute.String("event_id", eventID))

	// Validate input
	if eventID == "" {
		span.SetStatus(codes.Error, "invalid event_id")
		return nil, domain.ErrInvalidEventID
	}

	size, err := s.queueRepo.GetQueueSize(ctx, eventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int64("total_in_queue", size))
	span.SetStatus(codes.Ok, "")
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

// ValidateQueuePass validates the queue pass JWT and checks Redis
func (s *queueService) ValidateQueuePass(ctx context.Context, userID, eventID, queuePass string) error {
	ctx, span := telemetry.StartSpan(ctx, "service.queue.validate_pass")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", eventID),
	)

	if queuePass == "" {
		span.SetStatus(codes.Error, "queue pass required")
		return domain.ErrQueuePassRequired
	}

	// Parse and validate JWT
	token, err := jwt.ParseWithClaims(queuePass, &QueuePassClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid queue pass")
		return domain.ErrInvalidQueuePass
	}

	claims, ok := token.Claims.(*QueuePassClaims)
	if !ok || !token.Valid {
		span.SetStatus(codes.Error, "invalid queue pass claims")
		return domain.ErrInvalidQueuePass
	}

	// Verify claims match
	if claims.UserID != userID {
		span.SetStatus(codes.Error, "queue pass user mismatch")
		return domain.ErrQueuePassUserMismatch
	}

	if claims.EventID != eventID {
		span.SetStatus(codes.Error, "queue pass event mismatch")
		return domain.ErrQueuePassEventMismatch
	}

	if claims.Purpose != "queue_pass" {
		span.SetStatus(codes.Error, "invalid queue pass purpose")
		return domain.ErrInvalidQueuePass
	}

	// Validate against Redis (check if not already used/expired)
	valid, err := s.queueRepo.ValidateQueuePass(ctx, eventID, userID, queuePass)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate queue pass in redis")
		return fmt.Errorf("failed to validate queue pass: %w", err)
	}

	if !valid {
		span.SetStatus(codes.Error, "queue pass not found or expired")
		return domain.ErrQueuePassExpired
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// DeleteQueuePass removes the queue pass after successful booking
func (s *queueService) DeleteQueuePass(ctx context.Context, userID, eventID string) error {
	ctx, span := telemetry.StartSpan(ctx, "service.queue.delete_pass")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", eventID),
	)

	if err := s.queueRepo.DeleteQueuePass(ctx, eventID, userID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
