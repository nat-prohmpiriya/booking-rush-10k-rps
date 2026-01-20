package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueueService is a mock implementation of QueueService
type MockQueueService struct {
	mock.Mock
}

func (m *MockQueueService) JoinQueue(ctx context.Context, userID string, req *dto.JoinQueueRequest) (*dto.JoinQueueResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.JoinQueueResponse), args.Error(1)
}

func (m *MockQueueService) GetPosition(ctx context.Context, userID, eventID string) (*dto.QueuePositionResponse, error) {
	args := m.Called(ctx, userID, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.QueuePositionResponse), args.Error(1)
}

func (m *MockQueueService) LeaveQueue(ctx context.Context, userID string, req *dto.LeaveQueueRequest) (*dto.LeaveQueueResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LeaveQueueResponse), args.Error(1)
}

func (m *MockQueueService) GetQueueStatus(ctx context.Context, eventID string) (*dto.QueueStatusResponse, error) {
	args := m.Called(ctx, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.QueueStatusResponse), args.Error(1)
}

func (m *MockQueueService) ValidateQueuePass(ctx context.Context, userID, eventID, queuePass string) error {
	args := m.Called(ctx, userID, eventID, queuePass)
	return args.Error(0)
}

func (m *MockQueueService) DeleteQueuePass(ctx context.Context, userID, eventID string) error {
	args := m.Called(ctx, userID, eventID)
	return args.Error(0)
}

// newTestQueueHandler creates a QueueHandler for testing
func newTestQueueHandler(queueService *MockQueueService) *QueueHandler {
	return &QueueHandler{
		queueService: queueService,
		redisClient:  nil, // redis.Client can be nil for tests
	}
}

func setupQueueTestRouter(handler *QueueHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware to set user_id
	router.Use(func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	})

	queue := router.Group("/api/v1/queue")
	{
		queue.POST("/join", handler.JoinQueue)
		queue.GET("/position/:event_id", handler.GetPosition)
		queue.DELETE("/leave", handler.LeaveQueue)
		queue.GET("/status/:event_id", handler.GetQueueStatus)
	}

	return router
}

func TestQueueHandler_JoinQueue_Success(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	now := time.Now()
	expectedResponse := &dto.JoinQueueResponse{
		Position:      1,
		Token:         "test-token-123",
		EstimatedWait: 3,
		JoinedAt:      now,
		ExpiresAt:     now.Add(30 * time.Minute),
		Message:       "Successfully joined the queue",
	}

	mockService.On("JoinQueue", mock.Anything, "user-123", mock.AnythingOfType("*dto.JoinQueueRequest")).Return(expectedResponse, nil)

	reqBody := dto.JoinQueueRequest{
		EventID: "event-123",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/queue/join", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response dto.JoinQueueResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), response.Position)
	assert.Equal(t, "test-token-123", response.Token)

	mockService.AssertExpectations(t)
}

func TestQueueHandler_JoinQueue_Unauthorized(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	reqBody := dto.JoinQueueRequest{
		EventID: "event-123",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/queue/join", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-User-ID header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestQueueHandler_JoinQueue_InvalidRequest(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	// Missing required field
	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/queue/join", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQueueHandler_JoinQueue_AlreadyInQueue(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	mockService.On("JoinQueue", mock.Anything, "user-123", mock.AnythingOfType("*dto.JoinQueueRequest")).Return(nil, domain.ErrAlreadyInQueue)

	reqBody := dto.JoinQueueRequest{
		EventID: "event-123",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/queue/join", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response dto.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ALREADY_IN_QUEUE", response.Code)

	mockService.AssertExpectations(t)
}

func TestQueueHandler_GetPosition_Success(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	expectedResponse := &dto.QueuePositionResponse{
		Position:      5,
		TotalInQueue:  100,
		EstimatedWait: 15,
		IsReady:       false,
	}

	mockService.On("GetPosition", mock.Anything, "user-123", "event-123").Return(expectedResponse, nil)

	req, _ := http.NewRequest("GET", "/api/v1/queue/position/event-123", nil)
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.QueuePositionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), response.Position)
	assert.Equal(t, int64(100), response.TotalInQueue)
	assert.False(t, response.IsReady)

	mockService.AssertExpectations(t)
}

func TestQueueHandler_GetPosition_NotInQueue(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	mockService.On("GetPosition", mock.Anything, "user-123", "event-123").Return(nil, domain.ErrNotInQueue)

	req, _ := http.NewRequest("GET", "/api/v1/queue/position/event-123", nil)
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response dto.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "NOT_IN_QUEUE", response.Code)

	mockService.AssertExpectations(t)
}

func TestQueueHandler_LeaveQueue_Success(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	expectedResponse := &dto.LeaveQueueResponse{
		Success: true,
		Message: "Successfully left the queue",
	}

	mockService.On("LeaveQueue", mock.Anything, "user-123", mock.AnythingOfType("*dto.LeaveQueueRequest")).Return(expectedResponse, nil)

	reqBody := dto.LeaveQueueRequest{
		EventID: "event-123",
		Token:   "test-token-123",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("DELETE", "/api/v1/queue/leave", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.LeaveQueueResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockService.AssertExpectations(t)
}

func TestQueueHandler_LeaveQueue_InvalidToken(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	mockService.On("LeaveQueue", mock.Anything, "user-123", mock.AnythingOfType("*dto.LeaveQueueRequest")).Return(nil, domain.ErrInvalidQueueToken)

	reqBody := dto.LeaveQueueRequest{
		EventID: "event-123",
		Token:   "wrong-token",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("DELETE", "/api/v1/queue/leave", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response dto.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_TOKEN", response.Code)

	mockService.AssertExpectations(t)
}

func TestQueueHandler_GetQueueStatus_Success(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	expectedResponse := &dto.QueueStatusResponse{
		EventID:      "event-123",
		TotalInQueue: 500,
		IsOpen:       true,
	}

	mockService.On("GetQueueStatus", mock.Anything, "event-123").Return(expectedResponse, nil)

	req, _ := http.NewRequest("GET", "/api/v1/queue/status/event-123", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.QueueStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "event-123", response.EventID)
	assert.Equal(t, int64(500), response.TotalInQueue)
	assert.True(t, response.IsOpen)

	mockService.AssertExpectations(t)
}

func TestQueueHandler_QueueFull(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	mockService.On("JoinQueue", mock.Anything, "user-123", mock.AnythingOfType("*dto.JoinQueueRequest")).Return(nil, domain.ErrQueueFull)

	reqBody := dto.JoinQueueRequest{
		EventID: "event-123",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/queue/join", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response dto.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "QUEUE_FULL", response.Code)

	mockService.AssertExpectations(t)
}

func TestQueueHandler_GetPosition_WithQueuePass(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	now := time.Now()
	expectedResponse := &dto.QueuePositionResponse{
		Position:           1,
		TotalInQueue:       100,
		EstimatedWait:      0,
		IsReady:            true,
		QueuePass:          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature",
		QueuePassExpiresAt: now.Add(5 * time.Minute),
	}

	mockService.On("GetPosition", mock.Anything, "user-123", "event-123").Return(expectedResponse, nil)

	req, _ := http.NewRequest("GET", "/api/v1/queue/position/event-123", nil)
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.QueuePositionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), response.Position)
	assert.True(t, response.IsReady)
	assert.NotEmpty(t, response.QueuePass)
	assert.False(t, response.QueuePassExpiresAt.IsZero())

	mockService.AssertExpectations(t)
}

func TestQueueHandler_GetPosition_NoQueuePassWhenNotReady(t *testing.T) {
	mockService := new(MockQueueService)
	handler := newTestQueueHandler(mockService)
	router := setupQueueTestRouter(handler)

	expectedResponse := &dto.QueuePositionResponse{
		Position:      10,
		TotalInQueue:  100,
		EstimatedWait: 30,
		IsReady:       false,
		QueuePass:     "", // Empty when not ready
	}

	mockService.On("GetPosition", mock.Anything, "user-123", "event-123").Return(expectedResponse, nil)

	req, _ := http.NewRequest("GET", "/api/v1/queue/position/event-123", nil)
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.QueuePositionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), response.Position)
	assert.False(t, response.IsReady)
	assert.Empty(t, response.QueuePass)

	mockService.AssertExpectations(t)
}
