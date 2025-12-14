package worker

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	return args.Get(0).(bool), args.Error(1)
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

// Ensure MockQueueRepository implements QueueRepository
var _ repository.QueueRepository = (*MockQueueRepository)(nil)

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

// testWorkerJWTSecret is a constant secret used for testing only
const testWorkerJWTSecret = "test-jwt-secret-for-worker-tests"

func TestNewQueueReleaseWorker(t *testing.T) {
	mockRepo := new(MockQueueRepository)

	t.Run("creates worker with custom config", func(t *testing.T) {
		cfg := &QueueReleaseWorkerConfig{
			DefaultMaxConcurrent: 1000,
			ReleaseInterval:      5 * time.Second,
			DefaultQueuePassTTL:  10 * time.Minute,
			JWTSecret:            "custom-secret",
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)
		assert.NotNil(t, worker)
		assert.Equal(t, 1000, worker.GetDefaultMaxConcurrent())
	})

	t.Run("uses defaults for invalid config values except JWTSecret", func(t *testing.T) {
		cfg := &QueueReleaseWorkerConfig{
			DefaultMaxConcurrent: -1,
			ReleaseInterval:      0,
			DefaultQueuePassTTL:  0,
			JWTSecret:            testWorkerJWTSecret, // JWTSecret is now required
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)
		assert.NotNil(t, worker)
		assert.Equal(t, 500, worker.GetDefaultMaxConcurrent())
	})

	t.Run("panics when JWTSecret is empty", func(t *testing.T) {
		cfg := &QueueReleaseWorkerConfig{
			JWTSecret: "",
		}
		assert.Panics(t, func() {
			NewQueueReleaseWorker(cfg, mockRepo, nil)
		})
	})
}

func TestQueueReleaseWorker_ReleaseFromQueueOnce(t *testing.T) {
	t.Run("releases users based on dynamic capacity", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := &QueueReleaseWorkerConfig{
			DefaultMaxConcurrent: 500,
			DefaultQueuePassTTL:  5 * time.Minute,
			JWTSecret:            "test-secret",
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"
		userIDs := []string{"user-1", "user-2", "user-3"}

		// Config not found, use defaults (500 max)
		mockRepo.On("GetEventQueueConfig", ctx, eventID).Return(nil, nil)
		// 100 active, so release 400 (but only 3 in queue)
		mockRepo.On("CountActiveQueuePasses", ctx, eventID).Return(int64(100), nil)
		mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(400)).Return(userIDs, nil)
		mockRepo.On("StoreQueuePass", ctx, eventID, mock.AnythingOfType("string"), mock.AnythingOfType("string"), 300).Return(nil)

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.NoError(t, err)
		assert.Len(t, releasedUsers, 3)

		for i, user := range releasedUsers {
			assert.Equal(t, userIDs[i], user.UserID)
			assert.Equal(t, eventID, user.EventID)
			assert.NotEmpty(t, user.QueuePass)
			assert.False(t, user.QueuePassExpires.IsZero())
		}

		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty when at capacity", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := &QueueReleaseWorkerConfig{
			DefaultMaxConcurrent: 500,
			DefaultQueuePassTTL:  5 * time.Minute,
			JWTSecret:            "test-secret",
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"

		mockRepo.On("GetEventQueueConfig", ctx, eventID).Return(nil, nil)
		// At capacity (500 active, 500 max)
		mockRepo.On("CountActiveQueuePasses", ctx, eventID).Return(int64(500), nil)

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.NoError(t, err)
		assert.Len(t, releasedUsers, 0)

		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty when no users in queue", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := &QueueReleaseWorkerConfig{
			DefaultMaxConcurrent: 500,
			DefaultQueuePassTTL:  5 * time.Minute,
			JWTSecret:            testWorkerJWTSecret,
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"

		mockRepo.On("GetEventQueueConfig", ctx, eventID).Return(nil, nil)
		mockRepo.On("CountActiveQueuePasses", ctx, eventID).Return(int64(0), nil)
		mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(500)).Return([]string{}, nil)

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.NoError(t, err)
		assert.Len(t, releasedUsers, 0)

		mockRepo.AssertExpectations(t)
	})

	t.Run("handles count error gracefully", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := &QueueReleaseWorkerConfig{
			DefaultMaxConcurrent: 500,
			DefaultQueuePassTTL:  5 * time.Minute,
			JWTSecret:            testWorkerJWTSecret,
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"

		mockRepo.On("GetEventQueueConfig", ctx, eventID).Return(nil, nil)
		mockRepo.On("CountActiveQueuePasses", ctx, eventID).Return(int64(0), assert.AnError)

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.Error(t, err)
		assert.Nil(t, releasedUsers)

		mockRepo.AssertExpectations(t)
	})

	t.Run("uses custom event config", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := &QueueReleaseWorkerConfig{
			DefaultMaxConcurrent: 500,
			DefaultQueuePassTTL:  5 * time.Minute,
			JWTSecret:            testWorkerJWTSecret,
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"

		// Custom config: 100 max, 10 min TTL
		customConfig := &repository.EventQueueConfig{
			MaxConcurrentBookings: 100,
			QueuePassTTLMinutes:   10,
		}
		mockRepo.On("GetEventQueueConfig", ctx, eventID).Return(customConfig, nil)
		// 50 active, so release 50
		mockRepo.On("CountActiveQueuePasses", ctx, eventID).Return(int64(50), nil)
		mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(50)).Return([]string{"user-1"}, nil)
		// TTL should be 10 min = 600 seconds
		mockRepo.On("StoreQueuePass", ctx, eventID, mock.AnythingOfType("string"), mock.AnythingOfType("string"), 600).Return(nil)

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.NoError(t, err)
		assert.Len(t, releasedUsers, 1)

		mockRepo.AssertExpectations(t)
	})
}

func TestQueueReleaseWorker_GenerateQueuePass(t *testing.T) {
	t.Run("generates valid JWT", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		secret := "test-secret-key"
		cfg := &QueueReleaseWorkerConfig{
			DefaultMaxConcurrent: 500,
			DefaultQueuePassTTL:  5 * time.Minute,
			JWTSecret:            secret,
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		queuePass, expiresAt, err := worker.generateQueuePass("user-123", "event-456")

		assert.NoError(t, err)
		assert.NotEmpty(t, queuePass)
		assert.WithinDuration(t, time.Now().Add(5*time.Minute), expiresAt, time.Second)

		// Verify JWT can be parsed
		token, err := jwt.ParseWithClaims(queuePass, &QueuePassClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		assert.NoError(t, err)
		assert.True(t, token.Valid)

		claims, ok := token.Claims.(*QueuePassClaims)
		assert.True(t, ok)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "event-456", claims.EventID)
		assert.Equal(t, "queue_pass", claims.Purpose)
		assert.Equal(t, "queue-release-worker", claims.Issuer)
	})

	t.Run("generates JWT with custom TTL", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		secret := "test-secret-key"
		cfg := &QueueReleaseWorkerConfig{
			JWTSecret: secret,
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		queuePass, expiresAt, err := worker.generateQueuePassWithTTL("user-123", "event-456", 10*time.Minute)

		assert.NoError(t, err)
		assert.NotEmpty(t, queuePass)
		assert.WithinDuration(t, time.Now().Add(10*time.Minute), expiresAt, time.Second)
	})
}

func TestQueueReleaseWorker_GetMetrics(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	cfg := &QueueReleaseWorkerConfig{
		DefaultMaxConcurrent: 500,
		DefaultQueuePassTTL:  5 * time.Minute,
		JWTSecret:            "test-secret",
	}
	worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

	// Initial metrics should be zero
	total, lastTime, lastCount := worker.GetMetrics()
	assert.Equal(t, int64(0), total)
	assert.True(t, lastTime.IsZero())
	assert.Equal(t, 0, lastCount)

	// Release some users
	ctx := context.Background()
	eventID := "event-123"
	userIDs := []string{"user-1", "user-2"}

	mockRepo.On("GetEventQueueConfig", ctx, eventID).Return(nil, nil)
	mockRepo.On("CountActiveQueuePasses", ctx, eventID).Return(int64(0), nil)
	mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(500)).Return(userIDs, nil)
	mockRepo.On("StoreQueuePass", ctx, eventID, mock.AnythingOfType("string"), mock.AnythingOfType("string"), 300).Return(nil)

	_, _ = worker.ReleaseFromQueueOnce(ctx, eventID)

	total, lastTime, lastCount = worker.GetMetrics()
	assert.Equal(t, int64(2), total)
	assert.False(t, lastTime.IsZero())
	assert.Equal(t, 2, lastCount)
}

func TestDefaultQueueReleaseWorkerConfig(t *testing.T) {
	cfg := DefaultQueueReleaseWorkerConfig()

	assert.Equal(t, 500, cfg.DefaultMaxConcurrent)
	assert.Equal(t, 1*time.Second, cfg.ReleaseInterval)
	assert.Equal(t, 5*time.Minute, cfg.DefaultQueuePassTTL)
	// JWTSecret is now empty by default and must be provided
	assert.Empty(t, cfg.JWTSecret)
}

func TestGenerateUniqueID(t *testing.T) {
	id1 := generateUniqueID()
	id2 := generateUniqueID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // Should be unique
	assert.Len(t, id1, 32)       // 16 bytes = 32 hex chars
}
