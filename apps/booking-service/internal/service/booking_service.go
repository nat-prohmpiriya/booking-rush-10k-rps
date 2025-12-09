package service

import (
	"context"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/repository"
)

// BookingService defines the interface for booking business logic
type BookingService interface {
	// ReserveSeats reserves seats for a user
	ReserveSeats(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error)

	// ConfirmBooking confirms a reservation
	ConfirmBooking(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error)

	// ReleaseBooking releases a reservation
	ReleaseBooking(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error)

	// GetBooking retrieves a booking by ID
	GetBooking(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error)

	// GetUserBookings retrieves all bookings for a user
	GetUserBookings(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error)
}

// bookingService implements BookingService
type bookingService struct {
	bookingRepo     repository.BookingRepository
	reservationRepo repository.ReservationRepository
	reservationTTL  time.Duration
	maxPerUser      int
}

// BookingServiceConfig contains configuration for booking service
type BookingServiceConfig struct {
	ReservationTTL time.Duration
	MaxPerUser     int
}

// NewBookingService creates a new booking service
func NewBookingService(
	bookingRepo repository.BookingRepository,
	reservationRepo repository.ReservationRepository,
	cfg *BookingServiceConfig,
) BookingService {
	ttl := 10 * time.Minute
	maxPerUser := 10
	if cfg != nil {
		if cfg.ReservationTTL > 0 {
			ttl = cfg.ReservationTTL
		}
		if cfg.MaxPerUser > 0 {
			maxPerUser = cfg.MaxPerUser
		}
	}
	return &bookingService{
		bookingRepo:     bookingRepo,
		reservationRepo: reservationRepo,
		reservationTTL:  ttl,
		maxPerUser:      maxPerUser,
	}
}

// ReserveSeats reserves seats for a user
func (s *bookingService) ReserveSeats(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error) {
	params := repository.ReserveParams{
		ZoneID:     req.ZoneID,
		UserID:     userID,
		EventID:    req.EventID,
		Quantity:   req.Quantity,
		MaxPerUser: s.maxPerUser,
		TTLSeconds: int(s.reservationTTL.Seconds()),
		Price:      0, // TODO: Get from zone/event service
	}

	result, err := s.reservationRepo.ReserveSeats(ctx, params)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		switch result.ErrorCode {
		case "INSUFFICIENT_SEATS":
			return nil, domain.ErrInsufficientSeats
		case "MAX_TICKETS_EXCEEDED":
			return nil, domain.ErrMaxTicketsExceeded
		default:
			return nil, domain.ErrInvalidBookingStatus
		}
	}

	return &dto.ReserveSeatsResponse{
		BookingID:  result.BookingID,
		Status:     "reserved",
		ExpiresAt:  time.Now().Add(s.reservationTTL),
		TotalPrice: 0, // TODO: Calculate from zone price
	}, nil
}

// ConfirmBooking confirms a reservation
func (s *bookingService) ConfirmBooking(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error) {
	paymentID := ""
	if req != nil {
		paymentID = req.PaymentID
	}

	result, err := s.reservationRepo.ConfirmBooking(ctx, bookingID, userID, paymentID)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		switch result.ErrorCode {
		case "RESERVATION_NOT_FOUND":
			return nil, domain.ErrReservationNotFound
		case "INVALID_BOOKING_ID":
			return nil, domain.ErrInvalidBookingID
		case "INVALID_USER_ID":
			return nil, domain.ErrInvalidUserID
		case "ALREADY_CONFIRMED":
			return nil, domain.ErrAlreadyConfirmed
		case "INVALID_STATUS":
			return nil, domain.ErrInvalidBookingStatus
		default:
			return nil, domain.ErrInvalidBookingStatus
		}
	}

	return &dto.ConfirmBookingResponse{
		BookingID:   bookingID,
		Status:      "confirmed",
		ConfirmedAt: time.Now(),
	}, nil
}

// ReleaseBooking releases a reservation
func (s *bookingService) ReleaseBooking(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
	result, err := s.reservationRepo.ReleaseSeats(ctx, bookingID, userID)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		switch result.ErrorCode {
		case "RESERVATION_NOT_FOUND":
			return nil, domain.ErrReservationNotFound
		case "INVALID_BOOKING_ID":
			return nil, domain.ErrInvalidBookingID
		case "INVALID_USER_ID":
			return nil, domain.ErrInvalidUserID
		case "ALREADY_RELEASED":
			return nil, domain.ErrAlreadyReleased
		default:
			return nil, domain.ErrInvalidBookingStatus
		}
	}

	return &dto.ReleaseBookingResponse{
		BookingID: bookingID,
		Status:    "released",
		Message:   "Reservation released successfully",
	}, nil
}

// GetBooking retrieves a booking by ID
func (s *bookingService) GetBooking(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error) {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, domain.ErrBookingNotFound
	}
	if booking.UserID != userID {
		return nil, domain.ErrInvalidUserID
	}
	return dto.FromDomain(booking), nil
}

// GetUserBookings retrieves all bookings for a user
func (s *bookingService) GetUserBookings(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	bookings, err := s.bookingRepo.GetByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.BookingResponse, len(bookings))
	for i, b := range bookings {
		responses[i] = dto.FromDomain(b)
	}

	return &dto.PaginatedResponse{
		Data:     responses,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
