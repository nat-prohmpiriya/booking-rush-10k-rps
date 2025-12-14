package service

import (
	"context"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueueRepository is a mock implementation of QueueRepository
type MockQueueRepository struct {
	mock.Mock
}

func (m *MockQueueRepository) JoinQueue(ctx context.Context, params repository.JoinQueueParams) (*repository.JoinQueueResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.JoinQueueResult), args.Error(1)
}

func (m *MockQueueRepository) GetPosition(ctx context.Context, eventID, userID string) (*repository.QueuePositionResult, error) {
	args := m.Called(ctx, eventID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.QueuePositionResult), args.Error(1)
}

func (m *MockQueueRepository) LeaveQueue(ctx context.Context, eventID, userID, token string) error {
	args := m.Called(ctx, eventID, userID, token)
	return args.Error(0)
}

func (m *MockQueueRepository) GetQueueSize(ctx context.Context, eventID string) (int64, error) {
	args := m.Called(ctx, eventID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueueRepository) GetUserQueueInfo(ctx context.Context, eventID, userID string) (map[string]string, error) {
	args := m.Called(ctx, eventID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockQueueRepository) StoreQueuePass(ctx context.Context, eventID, userID, queuePass string, ttl int) error {
	args := m.Called(ctx, eventID, userID, queuePass, ttl)
	return args.Error(0)
}

func (m *MockQueueRepository) ValidateQueuePass(ctx context.Context, eventID, userID, queuePass string) (bool, error) {
	args := m.Called(ctx, eventID, userID, queuePass)
	return args.Bool(0), args.Error(1)
}

func (m *MockQueueRepository) DeleteQueuePass(ctx context.Context, eventID, userID string) error {
	args := m.Called(ctx, eventID, userID)
	return args.Error(0)
}

func (m *MockQueueRepository) PopUsersFromQueue(ctx context.Context, eventID string, count int64) ([]string, error) {
	args := m.Called(ctx, eventID, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockQueueRepository) GetAllQueueEventIDs(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockQueueRepository) RemoveUserFromQueue(ctx context.Context, eventID, userID string) error {
	args := m.Called(ctx, eventID, userID)
	return args.Error(0)
}

func (m *MockQueueRepository) CountActiveQueuePasses(ctx context.Context, eventID string) (int64, error) {
	args := m.Called(ctx, eventID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueueRepository) GetEventQueueConfig(ctx context.Context, eventID string) (*repository.EventQueueConfig, error) {
	args := m.Called(ctx, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.EventQueueConfig), args.Error(1)
}

func (m *MockQueueRepository) SetEventQueueConfig(ctx context.Context, eventID string, config *repository.EventQueueConfig) error {
	args := m.Called(ctx, eventID, config)
	return args.Error(0)
}

// testJWTSecret is a constant secret used for testing only
const testJWTSecret = "test-jwt-secret-for-unit-tests"

func TestQueueService_JoinQueue_Success(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{
		QueueTTL:             30 * time.Minute,
		MaxQueueSize:         0,
		EstimatedWaitPerUser: 3,
		JWTSecret:            testJWTSecret,
	})

	expectedResult := &repository.JoinQueueResult{
		Success:      true,
		Position:     1,
		TotalInQueue: 1,
		JoinedAt:     float64(time.Now().Unix()),
	}

	mockRepo.On("JoinQueue", mock.Anything, mock.MatchedBy(func(params repository.JoinQueueParams) bool {
		return params.UserID == "user-123" && params.EventID == "event-123"
	})).Return(expectedResult, nil)

	req := &dto.JoinQueueRequest{
		EventID: "event-123",
	}

	result, err := service.JoinQueue(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.Position)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, "Successfully joined the queue", result.Message)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_JoinQueue_AlreadyInQueue(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	expectedResult := &repository.JoinQueueResult{
		Success:      false,
		ErrorCode:    "ALREADY_IN_QUEUE",
		ErrorMessage: "User is already in queue",
	}

	mockRepo.On("JoinQueue", mock.Anything, mock.Anything).Return(expectedResult, nil)

	req := &dto.JoinQueueRequest{
		EventID: "event-123",
	}

	result, err := service.JoinQueue(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrAlreadyInQueue, err)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_JoinQueue_QueueFull(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	expectedResult := &repository.JoinQueueResult{
		Success:      false,
		ErrorCode:    "QUEUE_FULL",
		ErrorMessage: "Queue has reached maximum capacity",
	}

	mockRepo.On("JoinQueue", mock.Anything, mock.Anything).Return(expectedResult, nil)

	req := &dto.JoinQueueRequest{
		EventID: "event-123",
	}

	result, err := service.JoinQueue(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrQueueFull, err)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_JoinQueue_InvalidEventID(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	req := &dto.JoinQueueRequest{
		EventID: "",
	}

	result, err := service.JoinQueue(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrInvalidEventID, err)
}

func TestQueueService_JoinQueue_InvalidUserID(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	req := &dto.JoinQueueRequest{
		EventID: "event-123",
	}

	result, err := service.JoinQueue(context.Background(), "", req)

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrInvalidUserID, err)
}

func TestQueueService_GetPosition_Success(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{
		EstimatedWaitPerUser: 3,
		JWTSecret:            testJWTSecret,
	})

	expectedResult := &repository.QueuePositionResult{
		Position:     5,
		TotalInQueue: 100,
		IsInQueue:    true,
	}

	userInfo := map[string]string{
		"expires_at": "1700000000",
	}

	mockRepo.On("GetPosition", mock.Anything, "event-123", "user-123").Return(expectedResult, nil)
	mockRepo.On("GetUserQueueInfo", mock.Anything, "event-123", "user-123").Return(userInfo, nil)

	result, err := service.GetPosition(context.Background(), "user-123", "event-123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(5), result.Position)
	assert.Equal(t, int64(100), result.TotalInQueue)
	assert.Equal(t, int64(15), result.EstimatedWait) // 5 * 3
	assert.False(t, result.IsReady)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_GetPosition_NotInQueue(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	expectedResult := &repository.QueuePositionResult{
		Position:     0,
		TotalInQueue: 0,
		IsInQueue:    false,
	}

	mockRepo.On("GetPosition", mock.Anything, "event-123", "user-123").Return(expectedResult, nil)

	result, err := service.GetPosition(context.Background(), "user-123", "event-123")

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNotInQueue, err)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_GetPosition_IsReady(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{
		EstimatedWaitPerUser: 3,
		QueuePassTTL:         5 * time.Minute,
		JWTSecret:            "test-secret",
	})

	// Position 1 means user is ready
	expectedResult := &repository.QueuePositionResult{
		Position:     1,
		TotalInQueue: 100,
		IsInQueue:    true,
	}

	userInfo := map[string]string{}

	mockRepo.On("GetPosition", mock.Anything, "event-123", "user-123").Return(expectedResult, nil)
	mockRepo.On("GetUserQueueInfo", mock.Anything, "event-123", "user-123").Return(userInfo, nil)
	mockRepo.On("StoreQueuePass", mock.Anything, "event-123", "user-123", mock.AnythingOfType("string"), 300).Return(nil)

	result, err := service.GetPosition(context.Background(), "user-123", "event-123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsReady)
	assert.NotEmpty(t, result.QueuePass)
	assert.False(t, result.QueuePassExpiresAt.IsZero())

	mockRepo.AssertExpectations(t)
}

func TestQueueService_LeaveQueue_Success(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	mockRepo.On("LeaveQueue", mock.Anything, "event-123", "user-123", "token-123").Return(nil)

	req := &dto.LeaveQueueRequest{
		EventID: "event-123",
		Token:   "token-123",
	}

	result, err := service.LeaveQueue(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "Successfully left the queue", result.Message)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_LeaveQueue_InvalidToken(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	mockRepo.On("LeaveQueue", mock.Anything, "event-123", "user-123", "wrong-token").Return(domain.ErrInvalidQueueToken)

	req := &dto.LeaveQueueRequest{
		EventID: "event-123",
		Token:   "wrong-token",
	}

	result, err := service.LeaveQueue(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrInvalidQueueToken, err)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_LeaveQueue_NotInQueue(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	mockRepo.On("LeaveQueue", mock.Anything, "event-123", "user-123", "token-123").Return(domain.ErrNotInQueue)

	req := &dto.LeaveQueueRequest{
		EventID: "event-123",
		Token:   "token-123",
	}

	result, err := service.LeaveQueue(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNotInQueue, err)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_GetQueueStatus_Success(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	mockRepo.On("GetQueueSize", mock.Anything, "event-123").Return(int64(500), nil)

	result, err := service.GetQueueStatus(context.Background(), "event-123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "event-123", result.EventID)
	assert.Equal(t, int64(500), result.TotalInQueue)
	assert.True(t, result.IsOpen)

	mockRepo.AssertExpectations(t)
}

func TestQueueService_GetQueueStatus_InvalidEventID(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{JWTSecret: testJWTSecret})

	result, err := service.GetQueueStatus(context.Background(), "")

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrInvalidEventID, err)
}

func TestQueueService_EstimatedWait_Calculation(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{
		EstimatedWaitPerUser: 5, // 5 seconds per user
		JWTSecret:            testJWTSecret,
	})

	expectedResult := &repository.JoinQueueResult{
		Success:      true,
		Position:     10,
		TotalInQueue: 10,
		JoinedAt:     float64(time.Now().Unix()),
	}

	mockRepo.On("JoinQueue", mock.Anything, mock.Anything).Return(expectedResult, nil)

	req := &dto.JoinQueueRequest{
		EventID: "event-123",
	}

	result, err := service.JoinQueue(context.Background(), "user-123", req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(50), result.EstimatedWait) // 10 * 5

	mockRepo.AssertExpectations(t)
}

func TestQueueService_GetPosition_QueuePassGeneration(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{
		EstimatedWaitPerUser: 3,
		QueuePassTTL:         5 * time.Minute,
		JWTSecret:            "test-secret-key",
	})

	// Position 1 means user is ready and should receive a queue pass
	expectedResult := &repository.QueuePositionResult{
		Position:     1,
		TotalInQueue: 50,
		IsInQueue:    true,
	}

	userInfo := map[string]string{
		"expires_at": "1700000000",
	}

	mockRepo.On("GetPosition", mock.Anything, "event-456", "user-789").Return(expectedResult, nil)
	mockRepo.On("GetUserQueueInfo", mock.Anything, "event-456", "user-789").Return(userInfo, nil)
	mockRepo.On("StoreQueuePass", mock.Anything, "event-456", "user-789", mock.AnythingOfType("string"), 300).Return(nil)

	result, err := service.GetPosition(context.Background(), "user-789", "event-456")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsReady)
	assert.NotEmpty(t, result.QueuePass)
	assert.False(t, result.QueuePassExpiresAt.IsZero())
	// Verify queue pass expires in approximately 5 minutes from now
	assert.True(t, result.QueuePassExpiresAt.After(time.Now()))
	assert.True(t, result.QueuePassExpiresAt.Before(time.Now().Add(6*time.Minute)))

	mockRepo.AssertExpectations(t)
}

func TestQueueService_GetPosition_NoQueuePassWhenNotReady(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{
		EstimatedWaitPerUser: 3,
		QueuePassTTL:         5 * time.Minute,
		JWTSecret:            "test-secret-key",
	})

	// Position 5 means user is not ready yet
	expectedResult := &repository.QueuePositionResult{
		Position:     5,
		TotalInQueue: 100,
		IsInQueue:    true,
	}

	userInfo := map[string]string{}

	mockRepo.On("GetPosition", mock.Anything, "event-123", "user-123").Return(expectedResult, nil)
	mockRepo.On("GetUserQueueInfo", mock.Anything, "event-123", "user-123").Return(userInfo, nil)

	result, err := service.GetPosition(context.Background(), "user-123", "event-123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsReady)
	assert.Empty(t, result.QueuePass)
	assert.True(t, result.QueuePassExpiresAt.IsZero())

	mockRepo.AssertExpectations(t)
}

func TestQueueService_GetPosition_QueuePassStoreFails(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{
		EstimatedWaitPerUser: 3,
		QueuePassTTL:         5 * time.Minute,
		JWTSecret:            "test-secret-key",
	})

	expectedResult := &repository.QueuePositionResult{
		Position:     1,
		TotalInQueue: 50,
		IsInQueue:    true,
	}

	userInfo := map[string]string{}

	mockRepo.On("GetPosition", mock.Anything, "event-123", "user-123").Return(expectedResult, nil)
	mockRepo.On("GetUserQueueInfo", mock.Anything, "event-123", "user-123").Return(userInfo, nil)
	// Simulate Redis store failure
	mockRepo.On("StoreQueuePass", mock.Anything, "event-123", "user-123", mock.AnythingOfType("string"), 300).Return(assert.AnError)

	result, err := service.GetPosition(context.Background(), "user-123", "event-123")

	// Should still return successfully, just without the queue pass
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsReady)
	assert.Empty(t, result.QueuePass) // Empty because store failed

	mockRepo.AssertExpectations(t)
}

func TestQueueService_QueuePassJWTFormat(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	service := NewQueueService(mockRepo, &QueueServiceConfig{
		EstimatedWaitPerUser: 3,
		QueuePassTTL:         5 * time.Minute,
		JWTSecret:            "test-secret-key",
	})

	expectedResult := &repository.QueuePositionResult{
		Position:     1,
		TotalInQueue: 50,
		IsInQueue:    true,
	}

	userInfo := map[string]string{}

	mockRepo.On("GetPosition", mock.Anything, "event-123", "user-123").Return(expectedResult, nil)
	mockRepo.On("GetUserQueueInfo", mock.Anything, "event-123", "user-123").Return(userInfo, nil)
	mockRepo.On("StoreQueuePass", mock.Anything, "event-123", "user-123", mock.AnythingOfType("string"), 300).Return(nil)

	result, err := service.GetPosition(context.Background(), "user-123", "event-123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// JWT should have 3 parts separated by dots
	parts := len(result.QueuePass)
	assert.Greater(t, parts, 50) // JWT tokens are typically longer than 50 chars

	mockRepo.AssertExpectations(t)
}
