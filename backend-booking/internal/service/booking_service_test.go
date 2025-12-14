package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
)

// MockBookingRepository is a mock implementation of BookingRepository
type MockBookingRepository struct {
	CreateFunc                 func(ctx context.Context, booking *domain.Booking) error
	GetByIDFunc                func(ctx context.Context, id string) (*domain.Booking, error)
	GetByUserIDFunc            func(ctx context.Context, userID string, limit, offset int) ([]*domain.Booking, error)
	UpdateFunc                 func(ctx context.Context, booking *domain.Booking) error
	UpdateStatusFunc           func(ctx context.Context, id string, status domain.BookingStatus) error
	DeleteFunc                 func(ctx context.Context, id string) error
	ConfirmFunc                func(ctx context.Context, id, paymentID string) error
	CancelFunc                 func(ctx context.Context, id string) error
	GetExpiredReservationsFunc func(ctx context.Context, limit int) ([]*domain.Booking, error)
	MarkAsExpiredFunc          func(ctx context.Context, id string) error
	GetByIdempotencyKeyFunc    func(ctx context.Context, key string) (*domain.Booking, error)
	CountByUserAndEventFunc    func(ctx context.Context, userID, eventID string) (int, error)
	GetTenantIDByShowIDFunc    func(ctx context.Context, showID string) (string, error)
}

func (m *MockBookingRepository) Create(ctx context.Context, booking *domain.Booking) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, booking)
	}
	return nil
}

func (m *MockBookingRepository) GetByID(ctx context.Context, id string) (*domain.Booking, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, domain.ErrBookingNotFound
}

func (m *MockBookingRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Booking, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID, limit, offset)
	}
	return []*domain.Booking{}, nil
}

func (m *MockBookingRepository) Update(ctx context.Context, booking *domain.Booking) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, booking)
	}
	return nil
}

func (m *MockBookingRepository) UpdateStatus(ctx context.Context, id string, status domain.BookingStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
}

func (m *MockBookingRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockBookingRepository) Confirm(ctx context.Context, id, paymentID string) error {
	if m.ConfirmFunc != nil {
		return m.ConfirmFunc(ctx, id, paymentID)
	}
	return nil
}

func (m *MockBookingRepository) Cancel(ctx context.Context, id string) error {
	if m.CancelFunc != nil {
		return m.CancelFunc(ctx, id)
	}
	return nil
}

func (m *MockBookingRepository) GetExpiredReservations(ctx context.Context, limit int) ([]*domain.Booking, error) {
	if m.GetExpiredReservationsFunc != nil {
		return m.GetExpiredReservationsFunc(ctx, limit)
	}
	return []*domain.Booking{}, nil
}

func (m *MockBookingRepository) MarkAsExpired(ctx context.Context, id string) error {
	if m.MarkAsExpiredFunc != nil {
		return m.MarkAsExpiredFunc(ctx, id)
	}
	return nil
}

func (m *MockBookingRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error) {
	if m.GetByIdempotencyKeyFunc != nil {
		return m.GetByIdempotencyKeyFunc(ctx, key)
	}
	return nil, domain.ErrBookingNotFound
}

func (m *MockBookingRepository) CountByUserAndEvent(ctx context.Context, userID, eventID string) (int, error) {
	if m.CountByUserAndEventFunc != nil {
		return m.CountByUserAndEventFunc(ctx, userID, eventID)
	}
	return 0, nil
}

func (m *MockBookingRepository) GetTenantIDByShowID(ctx context.Context, showID string) (string, error) {
	if m.GetTenantIDByShowIDFunc != nil {
		return m.GetTenantIDByShowIDFunc(ctx, showID)
	}
	return "test-tenant-id", nil
}

// MockReservationRepository is a mock implementation of ReservationRepository
type MockReservationRepository struct {
	ReserveSeatsFunc        func(ctx context.Context, params repository.ReserveParams) (*repository.ReserveResult, error)
	ConfirmBookingFunc      func(ctx context.Context, bookingID, userID, paymentID string) (*repository.ConfirmResult, error)
	ReleaseSeatsFunc        func(ctx context.Context, bookingID, userID string) (*repository.ReleaseResult, error)
	GetZoneAvailabilityFunc func(ctx context.Context, zoneID string) (int64, error)
	SetZoneAvailabilityFunc func(ctx context.Context, zoneID string, seats int64) error
}

func (m *MockReservationRepository) ReserveSeats(ctx context.Context, params repository.ReserveParams) (*repository.ReserveResult, error) {
	if m.ReserveSeatsFunc != nil {
		return m.ReserveSeatsFunc(ctx, params)
	}
	return &repository.ReserveResult{
		Success:   true,
		BookingID: "test-booking-id",
	}, nil
}

func (m *MockReservationRepository) ConfirmBooking(ctx context.Context, bookingID, userID, paymentID string) (*repository.ConfirmResult, error) {
	if m.ConfirmBookingFunc != nil {
		return m.ConfirmBookingFunc(ctx, bookingID, userID, paymentID)
	}
	return &repository.ConfirmResult{
		Success: true,
		Status:  "CONFIRMED",
	}, nil
}

func (m *MockReservationRepository) ReleaseSeats(ctx context.Context, bookingID, userID string) (*repository.ReleaseResult, error) {
	if m.ReleaseSeatsFunc != nil {
		return m.ReleaseSeatsFunc(ctx, bookingID, userID)
	}
	return &repository.ReleaseResult{
		Success: true,
	}, nil
}

func (m *MockReservationRepository) GetZoneAvailability(ctx context.Context, zoneID string) (int64, error) {
	if m.GetZoneAvailabilityFunc != nil {
		return m.GetZoneAvailabilityFunc(ctx, zoneID)
	}
	return 100, nil
}

func (m *MockReservationRepository) SetZoneAvailability(ctx context.Context, zoneID string, seats int64) error {
	if m.SetZoneAvailabilityFunc != nil {
		return m.SetZoneAvailabilityFunc(ctx, zoneID, seats)
	}
	return nil
}

func TestBookingService_ReserveSeats(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		req           *dto.ReserveSeatsRequest
		setupMocks    func(*MockBookingRepository, *MockReservationRepository)
		wantErr       error
		wantBookingID bool
	}{
		{
			name:   "successful reservation",
			userID: "user-001",
			req: &dto.ReserveSeatsRequest{
				EventID:   "event-001",
				ZoneID:    "zone-001",
				ShowID:    "show-001",
				Quantity:  2,
				UnitPrice: 100.00,
			},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				rr.ReserveSeatsFunc = func(ctx context.Context, params repository.ReserveParams) (*repository.ReserveResult, error) {
					return &repository.ReserveResult{
						Success:        true,
						BookingID:      "booking-123",
						AvailableSeats: 98,
						UserReserved:   2,
					}, nil
				}
				br.CreateFunc = func(ctx context.Context, booking *domain.Booking) error {
					return nil
				}
			},
			wantErr:       nil,
			wantBookingID: true,
		},
		{
			name:   "idempotent request returns existing booking",
			userID: "user-001",
			req: &dto.ReserveSeatsRequest{
				EventID:        "event-001",
				ZoneID:         "zone-001",
				ShowID:         "show-001",
				Quantity:       2,
				IdempotencyKey: "idempotency-key-123",
			},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIdempotencyKeyFunc = func(ctx context.Context, key string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:         "existing-booking-id",
						Status:     domain.BookingStatusReserved,
						TotalPrice: 200.00,
						ExpiresAt:  time.Now().Add(10 * time.Minute),
					}, nil
				}
			},
			wantErr:       nil,
			wantBookingID: true,
		},
		{
			name:   "insufficient seats",
			userID: "user-001",
			req: &dto.ReserveSeatsRequest{
				EventID:  "event-001",
				ZoneID:   "zone-001",
				ShowID:   "show-001",
				Quantity: 2,
			},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				rr.ReserveSeatsFunc = func(ctx context.Context, params repository.ReserveParams) (*repository.ReserveResult, error) {
					return &repository.ReserveResult{
						Success:   false,
						ErrorCode: "INSUFFICIENT_STOCK",
					}, nil
				}
			},
			wantErr: domain.ErrInsufficientSeats,
		},
		{
			name:   "user limit exceeded",
			userID: "user-001",
			req: &dto.ReserveSeatsRequest{
				EventID:  "event-001",
				ZoneID:   "zone-001",
				ShowID:   "show-001",
				Quantity: 5,
			},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				rr.ReserveSeatsFunc = func(ctx context.Context, params repository.ReserveParams) (*repository.ReserveResult, error) {
					return &repository.ReserveResult{
						Success:   false,
						ErrorCode: "USER_LIMIT_EXCEEDED",
					}, nil
				}
			},
			wantErr: domain.ErrMaxTicketsExceeded,
		},
		{
			name:   "invalid quantity",
			userID: "user-001",
			req: &dto.ReserveSeatsRequest{
				EventID:  "event-001",
				ZoneID:   "zone-001",
				Quantity: 0,
			},
			wantErr: domain.ErrInvalidQuantity,
		},
		{
			name:   "missing event ID",
			userID: "user-001",
			req: &dto.ReserveSeatsRequest{
				ZoneID:   "zone-001",
				Quantity: 2,
			},
			wantErr: domain.ErrInvalidEventID,
		},
		{
			name:   "missing zone ID",
			userID: "user-001",
			req: &dto.ReserveSeatsRequest{
				EventID:  "event-001",
				Quantity: 2,
			},
			wantErr: domain.ErrInvalidZoneID,
		},
		{
			name:   "missing user ID",
			userID: "",
			req: &dto.ReserveSeatsRequest{
				EventID:  "event-001",
				ZoneID:   "zone-001",
				Quantity: 2,
			},
			wantErr: domain.ErrInvalidUserID,
		},
		{
			name:    "nil request",
			userID:  "user-001",
			req:     nil,
			wantErr: domain.ErrInvalidQuantity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookingRepo := &MockBookingRepository{}
			reservationRepo := &MockReservationRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(bookingRepo, reservationRepo)
			}

			svc := NewBookingService(bookingRepo, reservationRepo, nil, nil, &BookingServiceConfig{
				ReservationTTL: 10 * time.Minute,
				MaxPerUser:     10,
			})

			resp, err := svc.ReserveSeats(context.Background(), tt.userID, tt.req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ReserveSeats() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ReserveSeats() unexpected error = %v", err)
				return
			}

			if tt.wantBookingID && resp.BookingID == "" {
				t.Error("ReserveSeats() expected booking ID, got empty")
			}
		})
	}
}

func TestBookingService_ConfirmBooking(t *testing.T) {
	tests := []struct {
		name       string
		bookingID  string
		userID     string
		req        *dto.ConfirmBookingRequest
		setupMocks func(*MockBookingRepository, *MockReservationRepository)
		wantErr    error
	}{
		{
			name:      "successful confirmation",
			bookingID: "booking-123",
			userID:    "user-001",
			req:       &dto.ConfirmBookingRequest{PaymentID: "payment-123"},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:        id,
						UserID:    "user-001",
						Status:    domain.BookingStatusReserved,
						ExpiresAt: time.Now().Add(10 * time.Minute),
					}, nil
				}
				rr.ConfirmBookingFunc = func(ctx context.Context, bookingID, userID, paymentID string) (*repository.ConfirmResult, error) {
					return &repository.ConfirmResult{
						Success: true,
						Status:  "CONFIRMED",
					}, nil
				}
				br.ConfirmFunc = func(ctx context.Context, id, paymentID string) error {
					return nil
				}
			},
			wantErr: nil,
		},
		{
			name:      "booking not found",
			bookingID: "nonexistent",
			userID:    "user-001",
			req:       &dto.ConfirmBookingRequest{PaymentID: "payment-123"},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return nil, domain.ErrBookingNotFound
				}
			},
			wantErr: domain.ErrBookingNotFound,
		},
		{
			name:      "wrong user",
			bookingID: "booking-123",
			userID:    "user-002",
			req:       &dto.ConfirmBookingRequest{PaymentID: "payment-123"},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:        id,
						UserID:    "user-001", // Different user
						Status:    domain.BookingStatusReserved,
						ExpiresAt: time.Now().Add(10 * time.Minute),
					}, nil
				}
			},
			wantErr: domain.ErrInvalidUserID,
		},
		{
			name:      "already confirmed",
			bookingID: "booking-123",
			userID:    "user-001",
			req:       &dto.ConfirmBookingRequest{PaymentID: "payment-123"},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:     id,
						UserID: "user-001",
						Status: domain.BookingStatusConfirmed,
					}, nil
				}
			},
			wantErr: domain.ErrAlreadyConfirmed,
		},
		{
			name:      "booking expired",
			bookingID: "booking-123",
			userID:    "user-001",
			req:       &dto.ConfirmBookingRequest{PaymentID: "payment-123"},
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:        id,
						UserID:    "user-001",
						Status:    domain.BookingStatusReserved,
						ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
					}, nil
				}
			},
			wantErr: domain.ErrBookingExpired,
		},
		{
			name:      "missing booking ID",
			bookingID: "",
			userID:    "user-001",
			wantErr:   domain.ErrInvalidBookingID,
		},
		{
			name:      "missing user ID",
			bookingID: "booking-123",
			userID:    "",
			wantErr:   domain.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookingRepo := &MockBookingRepository{}
			reservationRepo := &MockReservationRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(bookingRepo, reservationRepo)
			}

			svc := NewBookingService(bookingRepo, reservationRepo, nil, nil, nil)

			resp, err := svc.ConfirmBooking(context.Background(), tt.bookingID, tt.userID, tt.req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ConfirmBooking() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ConfirmBooking() unexpected error = %v", err)
				return
			}

			if resp.Status != "confirmed" {
				t.Errorf("ConfirmBooking() status = %v, want confirmed", resp.Status)
			}
		})
	}
}

func TestBookingService_CancelBooking(t *testing.T) {
	tests := []struct {
		name       string
		bookingID  string
		userID     string
		setupMocks func(*MockBookingRepository, *MockReservationRepository)
		wantErr    error
	}{
		{
			name:      "successful cancellation",
			bookingID: "booking-123",
			userID:    "user-001",
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:     id,
						UserID: "user-001",
						Status: domain.BookingStatusReserved,
					}, nil
				}
				rr.ReleaseSeatsFunc = func(ctx context.Context, bookingID, userID string) (*repository.ReleaseResult, error) {
					return &repository.ReleaseResult{
						Success: true,
					}, nil
				}
				br.CancelFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			wantErr: nil,
		},
		{
			name:      "booking not found",
			bookingID: "nonexistent",
			userID:    "user-001",
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return nil, domain.ErrBookingNotFound
				}
			},
			wantErr: domain.ErrBookingNotFound,
		},
		{
			name:      "cannot cancel confirmed booking",
			bookingID: "booking-123",
			userID:    "user-001",
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:     id,
						UserID: "user-001",
						Status: domain.BookingStatusConfirmed,
					}, nil
				}
			},
			wantErr: domain.ErrAlreadyConfirmed,
		},
		{
			name:      "already cancelled",
			bookingID: "booking-123",
			userID:    "user-001",
			setupMocks: func(br *MockBookingRepository, rr *MockReservationRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:     id,
						UserID: "user-001",
						Status: domain.BookingStatusCancelled,
					}, nil
				}
			},
			wantErr: domain.ErrAlreadyReleased,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookingRepo := &MockBookingRepository{}
			reservationRepo := &MockReservationRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(bookingRepo, reservationRepo)
			}

			svc := NewBookingService(bookingRepo, reservationRepo, nil, nil, nil)

			resp, err := svc.CancelBooking(context.Background(), tt.bookingID, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CancelBooking() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("CancelBooking() unexpected error = %v", err)
				return
			}

			if resp.Status != "cancelled" {
				t.Errorf("CancelBooking() status = %v, want cancelled", resp.Status)
			}
		})
	}
}

func TestBookingService_GetBooking(t *testing.T) {
	tests := []struct {
		name       string
		bookingID  string
		userID     string
		setupMocks func(*MockBookingRepository)
		wantErr    error
	}{
		{
			name:      "successful get",
			bookingID: "booking-123",
			userID:    "user-001",
			setupMocks: func(br *MockBookingRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:         id,
						UserID:     "user-001",
						EventID:    "event-001",
						ZoneID:     "zone-001",
						Quantity:   2,
						TotalPrice: 200.00,
						Status:     domain.BookingStatusReserved,
					}, nil
				}
			},
			wantErr: nil,
		},
		{
			name:      "booking not found",
			bookingID: "nonexistent",
			userID:    "user-001",
			setupMocks: func(br *MockBookingRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return nil, domain.ErrBookingNotFound
				}
			},
			wantErr: domain.ErrBookingNotFound,
		},
		{
			name:      "wrong user",
			bookingID: "booking-123",
			userID:    "user-002",
			setupMocks: func(br *MockBookingRepository) {
				br.GetByIDFunc = func(ctx context.Context, id string) (*domain.Booking, error) {
					return &domain.Booking{
						ID:     id,
						UserID: "user-001",
					}, nil
				}
			},
			wantErr: domain.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookingRepo := &MockBookingRepository{}
			reservationRepo := &MockReservationRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(bookingRepo)
			}

			svc := NewBookingService(bookingRepo, reservationRepo, nil, nil, nil)

			resp, err := svc.GetBooking(context.Background(), tt.bookingID, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetBooking() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetBooking() unexpected error = %v", err)
				return
			}

			if resp.ID != tt.bookingID {
				t.Errorf("GetBooking() ID = %v, want %v", resp.ID, tt.bookingID)
			}
		})
	}
}

func TestBookingService_GetUserBookings(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		page       int
		pageSize   int
		setupMocks func(*MockBookingRepository)
		wantErr    error
		wantCount  int
	}{
		{
			name:     "successful get with results",
			userID:   "user-001",
			page:     1,
			pageSize: 10,
			setupMocks: func(br *MockBookingRepository) {
				br.GetByUserIDFunc = func(ctx context.Context, userID string, limit, offset int) ([]*domain.Booking, error) {
					return []*domain.Booking{
						{ID: "booking-1", UserID: userID},
						{ID: "booking-2", UserID: userID},
					}, nil
				}
			},
			wantErr:   nil,
			wantCount: 2,
		},
		{
			name:     "empty results",
			userID:   "user-001",
			page:     1,
			pageSize: 10,
			setupMocks: func(br *MockBookingRepository) {
				br.GetByUserIDFunc = func(ctx context.Context, userID string, limit, offset int) ([]*domain.Booking, error) {
					return []*domain.Booking{}, nil
				}
			},
			wantErr:   nil,
			wantCount: 0,
		},
		{
			name:     "missing user ID",
			userID:   "",
			page:     1,
			pageSize: 10,
			wantErr:  domain.ErrInvalidUserID,
		},
		{
			name:     "default pagination values",
			userID:   "user-001",
			page:     0,
			pageSize: 0,
			setupMocks: func(br *MockBookingRepository) {
				br.GetByUserIDFunc = func(ctx context.Context, userID string, limit, offset int) ([]*domain.Booking, error) {
					if limit != 21 || offset != 0 { // 20 + 1 for checking more
						t.Errorf("Expected limit=21, offset=0, got limit=%d, offset=%d", limit, offset)
					}
					return []*domain.Booking{}, nil
				}
			},
			wantErr:   nil,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookingRepo := &MockBookingRepository{}
			reservationRepo := &MockReservationRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(bookingRepo)
			}

			svc := NewBookingService(bookingRepo, reservationRepo, nil, nil, nil)

			resp, err := svc.GetUserBookings(context.Background(), tt.userID, tt.page, tt.pageSize)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetUserBookings() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetUserBookings() unexpected error = %v", err)
				return
			}

			data, ok := resp.Data.([]*dto.BookingResponse)
			if !ok {
				t.Error("GetUserBookings() data is not []*BookingResponse")
				return
			}

			if len(data) != tt.wantCount {
				t.Errorf("GetUserBookings() count = %d, want %d", len(data), tt.wantCount)
			}
		})
	}
}

func TestBookingService_ExpireReservations(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		setupMocks func(*MockBookingRepository)
		wantCount  int
		wantErr    error
	}{
		{
			name:  "expire multiple reservations",
			limit: 100,
			setupMocks: func(br *MockBookingRepository) {
				br.GetExpiredReservationsFunc = func(ctx context.Context, limit int) ([]*domain.Booking, error) {
					return []*domain.Booking{
						{ID: "booking-1"},
						{ID: "booking-2"},
						{ID: "booking-3"},
					}, nil
				}
				br.MarkAsExpiredFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			wantCount: 3,
			wantErr:   nil,
		},
		{
			name:  "no expired reservations",
			limit: 100,
			setupMocks: func(br *MockBookingRepository) {
				br.GetExpiredReservationsFunc = func(ctx context.Context, limit int) ([]*domain.Booking, error) {
					return []*domain.Booking{}, nil
				}
			},
			wantCount: 0,
			wantErr:   nil,
		},
		{
			name:  "partial failure",
			limit: 100,
			setupMocks: func(br *MockBookingRepository) {
				br.GetExpiredReservationsFunc = func(ctx context.Context, limit int) ([]*domain.Booking, error) {
					return []*domain.Booking{
						{ID: "booking-1"},
						{ID: "booking-2"},
					}, nil
				}
				callCount := 0
				br.MarkAsExpiredFunc = func(ctx context.Context, id string) error {
					callCount++
					if callCount == 1 {
						return errors.New("db error")
					}
					return nil
				}
			},
			wantCount: 1, // Only one succeeds
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookingRepo := &MockBookingRepository{}
			reservationRepo := &MockReservationRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(bookingRepo)
			}

			svc := NewBookingService(bookingRepo, reservationRepo, nil, nil, nil)

			count, err := svc.ExpireReservations(context.Background(), tt.limit)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ExpireReservations() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ExpireReservations() unexpected error = %v", err)
				return
			}

			if count != tt.wantCount {
				t.Errorf("ExpireReservations() count = %d, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestBookingServiceConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		svc := NewBookingService(nil, nil, nil, nil, nil)
		impl := svc.(*bookingService)

		if impl.reservationTTL != 10*time.Minute {
			t.Errorf("Default TTL = %v, want 10 minutes", impl.reservationTTL)
		}
		if impl.maxPerUser != 10 {
			t.Errorf("Default maxPerUser = %d, want 10", impl.maxPerUser)
		}
		if impl.defaultCurrency != "THB" {
			t.Errorf("Default currency = %s, want THB", impl.defaultCurrency)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		svc := NewBookingService(nil, nil, nil, nil, &BookingServiceConfig{
			ReservationTTL:  5 * time.Minute,
			MaxPerUser:      4,
			DefaultCurrency: "USD",
		})
		impl := svc.(*bookingService)

		if impl.reservationTTL != 5*time.Minute {
			t.Errorf("Custom TTL = %v, want 5 minutes", impl.reservationTTL)
		}
		if impl.maxPerUser != 4 {
			t.Errorf("Custom maxPerUser = %d, want 4", impl.maxPerUser)
		}
		if impl.defaultCurrency != "USD" {
			t.Errorf("Custom currency = %s, want USD", impl.defaultCurrency)
		}
	})
}
