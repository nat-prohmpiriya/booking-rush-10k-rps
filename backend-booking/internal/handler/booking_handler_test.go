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
)

// MockBookingService is a mock implementation of BookingService for testing
type MockBookingService struct {
	ReserveSeatsFunc           func(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error)
	ConfirmBookingFunc         func(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error)
	CancelBookingFunc          func(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error)
	ReleaseBookingFunc         func(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error)
	GetBookingFunc             func(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error)
	GetUserBookingsFunc        func(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error)
	GetUserBookingSummaryFunc  func(ctx context.Context, userID, eventID string) (*dto.UserBookingSummaryResponse, error)
	GetPendingBookingsFunc     func(ctx context.Context, limit int) ([]*dto.BookingResponse, error)
	ExpireReservationsFunc     func(ctx context.Context, limit int) (int, error)
}

func (m *MockBookingService) ReserveSeats(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error) {
	if m.ReserveSeatsFunc != nil {
		return m.ReserveSeatsFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *MockBookingService) ConfirmBooking(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error) {
	if m.ConfirmBookingFunc != nil {
		return m.ConfirmBookingFunc(ctx, bookingID, userID, req)
	}
	return nil, nil
}

func (m *MockBookingService) CancelBooking(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
	if m.CancelBookingFunc != nil {
		return m.CancelBookingFunc(ctx, bookingID, userID)
	}
	return nil, nil
}

func (m *MockBookingService) ReleaseBooking(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
	if m.ReleaseBookingFunc != nil {
		return m.ReleaseBookingFunc(ctx, bookingID, userID)
	}
	return nil, nil
}

func (m *MockBookingService) GetBooking(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error) {
	if m.GetBookingFunc != nil {
		return m.GetBookingFunc(ctx, bookingID, userID)
	}
	return nil, nil
}

func (m *MockBookingService) GetUserBookings(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error) {
	if m.GetUserBookingsFunc != nil {
		return m.GetUserBookingsFunc(ctx, userID, page, pageSize)
	}
	return nil, nil
}

func (m *MockBookingService) GetUserBookingSummary(ctx context.Context, userID, eventID string) (*dto.UserBookingSummaryResponse, error) {
	if m.GetUserBookingSummaryFunc != nil {
		return m.GetUserBookingSummaryFunc(ctx, userID, eventID)
	}
	return nil, nil
}

func (m *MockBookingService) GetPendingBookings(ctx context.Context, limit int) ([]*dto.BookingResponse, error) {
	if m.GetPendingBookingsFunc != nil {
		return m.GetPendingBookingsFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockBookingService) ExpireReservations(ctx context.Context, limit int) (int, error) {
	if m.ExpireReservationsFunc != nil {
		return m.ExpireReservationsFunc(ctx, limit)
	}
	return 0, nil
}

// newTestBookingHandler creates a BookingHandler for testing with mock services
func newTestBookingHandler(bookingService *MockBookingService) *BookingHandler {
	return &BookingHandler{
		bookingService:   bookingService,
		queueService:     &MockQueueService{},
		requireQueuePass: false,
	}
}

func setupTestRouter(handler *BookingHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	bookings := router.Group("/bookings")
	{
		bookings.POST("/reserve", handler.ReserveSeats)
		bookings.GET("", handler.GetUserBookings)
		bookings.GET("/pending", handler.GetPendingBookings)
		bookings.GET("/:id", handler.GetBooking)
		bookings.POST("/:id/confirm", handler.ConfirmBooking)
		bookings.POST("/:id/cancel", handler.CancelBooking)
		bookings.DELETE("/:id", handler.ReleaseBooking)
	}

	return router
}

func setupTestRouterWithAuth(handler *BookingHandler, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware to set user_id
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})

	bookings := router.Group("/bookings")
	{
		bookings.POST("/reserve", handler.ReserveSeats)
		bookings.GET("", handler.GetUserBookings)
		bookings.GET("/pending", handler.GetPendingBookings)
		bookings.GET("/:id", handler.GetBooking)
		bookings.POST("/:id/confirm", handler.ConfirmBooking)
		bookings.POST("/:id/cancel", handler.CancelBooking)
		bookings.DELETE("/:id", handler.ReleaseBooking)
	}

	return router
}

func TestBookingHandler_ReserveSeats(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		request        *dto.ReserveSeatsRequest
		mockFunc       func(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:   "successful reservation",
			userID: "user-123",
			request: &dto.ReserveSeatsRequest{
				EventID:  "event-123",
				ZoneID:   "zone-123",
				Quantity: 2,
			},
			mockFunc: func(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error) {
				return &dto.ReserveSeatsResponse{
					BookingID:  "booking-123",
					Status:     "reserved",
					ExpiresAt:  time.Now().Add(10 * time.Minute),
					TotalPrice: 200.00,
				}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "unauthorized - no user_id",
			userID:         "",
			request:        &dto.ReserveSeatsRequest{},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:   "insufficient seats",
			userID: "user-123",
			request: &dto.ReserveSeatsRequest{
				EventID:  "event-123",
				ZoneID:   "zone-123",
				Quantity: 5,
			},
			mockFunc: func(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error) {
				return nil, domain.ErrInsufficientSeats
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "INSUFFICIENT_SEATS",
		},
		{
			name:   "max tickets exceeded",
			userID: "user-123",
			request: &dto.ReserveSeatsRequest{
				EventID:  "event-123",
				ZoneID:   "zone-123",
				Quantity: 5,
			},
			mockFunc: func(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error) {
				return nil, domain.ErrMaxTicketsExceeded
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "MAX_TICKETS_EXCEEDED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBookingService{
				ReserveSeatsFunc: tt.mockFunc,
			}
			handler := newTestBookingHandler(mockService)

			var router *gin.Engine
			if tt.userID != "" {
				router = setupTestRouterWithAuth(handler, tt.userID)
			} else {
				router = setupTestRouter(handler)
			}

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/bookings/reserve", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedCode != "" {
				var response dto.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if response.Code != tt.expectedCode {
						t.Errorf("expected code %s, got %s", tt.expectedCode, response.Code)
					}
				}
			}
		})
	}
}

func TestBookingHandler_ConfirmBooking(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		bookingID      string
		request        *dto.ConfirmBookingRequest
		mockFunc       func(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful confirmation",
			userID:    "user-123",
			bookingID: "booking-123",
			request:   &dto.ConfirmBookingRequest{PaymentID: "payment-123"},
			mockFunc: func(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error) {
				return &dto.ConfirmBookingResponse{
					BookingID:        bookingID,
					Status:           "confirmed",
					ConfirmedAt:      time.Now(),
					ConfirmationCode: "ABC12345",
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no user_id",
			userID:         "",
			bookingID:      "booking-123",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:      "booking not found",
			userID:    "user-123",
			bookingID: "non-existent",
			mockFunc: func(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error) {
				return nil, domain.ErrBookingNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:      "already confirmed",
			userID:    "user-123",
			bookingID: "booking-123",
			mockFunc: func(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error) {
				return nil, domain.ErrAlreadyConfirmed
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "ALREADY_CONFIRMED",
		},
		{
			name:      "booking expired",
			userID:    "user-123",
			bookingID: "booking-123",
			mockFunc: func(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error) {
				return nil, domain.ErrBookingExpired
			},
			expectedStatus: http.StatusGone,
			expectedCode:   "EXPIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBookingService{
				ConfirmBookingFunc: tt.mockFunc,
			}
			handler := newTestBookingHandler(mockService)

			var router *gin.Engine
			if tt.userID != "" {
				router = setupTestRouterWithAuth(handler, tt.userID)
			} else {
				router = setupTestRouter(handler)
			}

			var body []byte
			if tt.request != nil {
				body, _ = json.Marshal(tt.request)
			}
			req := httptest.NewRequest(http.MethodPost, "/bookings/"+tt.bookingID+"/confirm", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedCode != "" {
				var response dto.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if response.Code != tt.expectedCode {
						t.Errorf("expected code %s, got %s", tt.expectedCode, response.Code)
					}
				}
			}
		})
	}
}

func TestBookingHandler_CancelBooking(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		bookingID      string
		mockFunc       func(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful cancellation",
			userID:    "user-123",
			bookingID: "booking-123",
			mockFunc: func(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
				return &dto.ReleaseBookingResponse{
					BookingID: bookingID,
					Status:    "cancelled",
					Message:   "Booking cancelled successfully",
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no user_id",
			userID:         "",
			bookingID:      "booking-123",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:      "booking not found",
			userID:    "user-123",
			bookingID: "non-existent",
			mockFunc: func(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
				return nil, domain.ErrBookingNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:      "already confirmed - cannot cancel",
			userID:    "user-123",
			bookingID: "booking-123",
			mockFunc: func(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
				return nil, domain.ErrAlreadyConfirmed
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "ALREADY_CONFIRMED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBookingService{
				CancelBookingFunc: tt.mockFunc,
			}
			handler := newTestBookingHandler(mockService)

			var router *gin.Engine
			if tt.userID != "" {
				router = setupTestRouterWithAuth(handler, tt.userID)
			} else {
				router = setupTestRouter(handler)
			}

			req := httptest.NewRequest(http.MethodPost, "/bookings/"+tt.bookingID+"/cancel", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedCode != "" {
				var response dto.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if response.Code != tt.expectedCode {
						t.Errorf("expected code %s, got %s", tt.expectedCode, response.Code)
					}
				}
			}
		})
	}
}

func TestBookingHandler_ReleaseBooking(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		bookingID      string
		mockFunc       func(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful release",
			userID:    "user-123",
			bookingID: "booking-123",
			mockFunc: func(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
				return &dto.ReleaseBookingResponse{
					BookingID: bookingID,
					Status:    "cancelled",
					Message:   "Booking released successfully",
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no user_id",
			userID:         "",
			bookingID:      "booking-123",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBookingService{
				ReleaseBookingFunc: tt.mockFunc,
			}
			handler := newTestBookingHandler(mockService)

			var router *gin.Engine
			if tt.userID != "" {
				router = setupTestRouterWithAuth(handler, tt.userID)
			} else {
				router = setupTestRouter(handler)
			}

			req := httptest.NewRequest(http.MethodDelete, "/bookings/"+tt.bookingID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestBookingHandler_GetBooking(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		bookingID      string
		mockFunc       func(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful get",
			userID:    "user-123",
			bookingID: "booking-123",
			mockFunc: func(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error) {
				return &dto.BookingResponse{
					ID:         bookingID,
					UserID:     userID,
					EventID:    "event-123",
					ZoneID:     "zone-123",
					Quantity:   2,
					Status:     "reserved",
					TotalPrice: 200.00,
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no user_id",
			userID:         "",
			bookingID:      "booking-123",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:      "booking not found",
			userID:    "user-123",
			bookingID: "non-existent",
			mockFunc: func(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error) {
				return nil, domain.ErrBookingNotFound
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:      "wrong user",
			userID:    "user-123",
			bookingID: "booking-456",
			mockFunc: func(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error) {
				return nil, domain.ErrInvalidUserID
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBookingService{
				GetBookingFunc: tt.mockFunc,
			}
			handler := newTestBookingHandler(mockService)

			var router *gin.Engine
			if tt.userID != "" {
				router = setupTestRouterWithAuth(handler, tt.userID)
			} else {
				router = setupTestRouter(handler)
			}

			req := httptest.NewRequest(http.MethodGet, "/bookings/"+tt.bookingID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedCode != "" {
				var response dto.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if response.Code != tt.expectedCode {
						t.Errorf("expected code %s, got %s", tt.expectedCode, response.Code)
					}
				}
			}
		})
	}
}

func TestBookingHandler_GetUserBookings(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		query           string
		mockFunc        func(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error)
		expectedStatus  int
		expectedCode    string
		checkPagination func(t *testing.T, ctx context.Context, userID string, page, pageSize int)
	}{
		{
			name:   "successful list with defaults",
			userID: "user-123",
			query:  "",
			mockFunc: func(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error) {
				return &dto.PaginatedResponse{
					Data:     []*dto.BookingResponse{},
					Page:     page,
					PageSize: pageSize,
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "successful list with pagination",
			userID: "user-123",
			query:  "?page=2&page_size=50",
			mockFunc: func(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error) {
				if page != 2 {
					t.Errorf("expected page 2, got %d", page)
				}
				if pageSize != 50 {
					t.Errorf("expected pageSize 50, got %d", pageSize)
				}
				return &dto.PaginatedResponse{
					Data:     []*dto.BookingResponse{},
					Page:     page,
					PageSize: pageSize,
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no user_id",
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBookingService{
				GetUserBookingsFunc: tt.mockFunc,
			}
			handler := newTestBookingHandler(mockService)

			var router *gin.Engine
			if tt.userID != "" {
				router = setupTestRouterWithAuth(handler, tt.userID)
			} else {
				router = setupTestRouter(handler)
			}

			req := httptest.NewRequest(http.MethodGet, "/bookings"+tt.query, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestBookingHandler_GetPendingBookings(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		mockFunc       func(ctx context.Context, limit int) ([]*dto.BookingResponse, error)
		expectedStatus int
	}{
		{
			name:  "successful list with default limit",
			query: "",
			mockFunc: func(ctx context.Context, limit int) ([]*dto.BookingResponse, error) {
				if limit != 100 {
					t.Errorf("expected default limit 100, got %d", limit)
				}
				return []*dto.BookingResponse{}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "successful list with custom limit",
			query: "?limit=50",
			mockFunc: func(ctx context.Context, limit int) ([]*dto.BookingResponse, error) {
				if limit != 50 {
					t.Errorf("expected limit 50, got %d", limit)
				}
				return []*dto.BookingResponse{
					{ID: "booking-1", Status: "reserved"},
					{ID: "booking-2", Status: "reserved"},
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "service error",
			query: "",
			mockFunc: func(ctx context.Context, limit int) ([]*dto.BookingResponse, error) {
				return nil, domain.ErrBookingNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBookingService{
				GetPendingBookingsFunc: tt.mockFunc,
			}
			handler := newTestBookingHandler(mockService)
			router := setupTestRouterWithAuth(handler, "admin")

			req := httptest.NewRequest(http.MethodGet, "/bookings/pending"+tt.query, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestBookingHandler_HandleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "booking not found",
			err:            domain.ErrBookingNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "reservation not found",
			err:            domain.ErrReservationNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "invalid user id",
			err:            domain.ErrInvalidUserID,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "insufficient seats",
			err:            domain.ErrInsufficientSeats,
			expectedStatus: http.StatusConflict,
			expectedCode:   "INSUFFICIENT_SEATS",
		},
		{
			name:           "max tickets exceeded",
			err:            domain.ErrMaxTicketsExceeded,
			expectedStatus: http.StatusConflict,
			expectedCode:   "MAX_TICKETS_EXCEEDED",
		},
		{
			name:           "already confirmed",
			err:            domain.ErrAlreadyConfirmed,
			expectedStatus: http.StatusConflict,
			expectedCode:   "ALREADY_CONFIRMED",
		},
		{
			name:           "already released",
			err:            domain.ErrAlreadyReleased,
			expectedStatus: http.StatusConflict,
			expectedCode:   "ALREADY_RELEASED",
		},
		{
			name:           "booking expired",
			err:            domain.ErrBookingExpired,
			expectedStatus: http.StatusGone,
			expectedCode:   "EXPIRED",
		},
		{
			name:           "reservation expired",
			err:            domain.ErrReservationExpired,
			expectedStatus: http.StatusGone,
			expectedCode:   "EXPIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBookingService{
				GetBookingFunc: func(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error) {
					return nil, tt.err
				},
			}
			handler := newTestBookingHandler(mockService)
			router := setupTestRouterWithAuth(handler, "user-123")

			req := httptest.NewRequest(http.MethodGet, "/bookings/test-id", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response dto.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if response.Code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, response.Code)
			}
		})
	}
}

func TestBookingHandler_InvalidRequestBody(t *testing.T) {
	mockService := &MockBookingService{}
	handler := newTestBookingHandler(mockService)
	router := setupTestRouterWithAuth(handler, "user-123")

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/bookings/reserve", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response dto.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response.Code != "INVALID_REQUEST" {
		t.Errorf("expected code INVALID_REQUEST, got %s", response.Code)
	}
}
