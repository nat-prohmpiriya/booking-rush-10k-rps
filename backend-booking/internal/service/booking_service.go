package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
)

// BookingService defines the interface for booking business logic
type BookingService interface {
	// ReserveSeats reserves seats for a user with idempotency support
	ReserveSeats(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error)

	// ConfirmBooking confirms a reservation with payment
	ConfirmBooking(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error)

	// CancelBooking cancels a reservation
	CancelBooking(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error)

	// ReleaseBooking releases a reservation (alias for CancelBooking)
	ReleaseBooking(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error)

	// GetBooking retrieves a booking by ID
	GetBooking(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error)

	// GetUserBookings retrieves all bookings for a user
	GetUserBookings(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error)

	// GetPendingBookings retrieves pending reservations that are about to expire
	GetPendingBookings(ctx context.Context, limit int) ([]*dto.BookingResponse, error)

	// ExpireReservations marks expired reservations as expired
	ExpireReservations(ctx context.Context, limit int) (int, error)
}

// bookingService implements BookingService
type bookingService struct {
	bookingRepo     repository.BookingRepository
	reservationRepo repository.ReservationRepository
	eventPublisher  EventPublisher
	zoneSyncer      ZoneSyncer
	reservationTTL  time.Duration
	maxPerUser      int
	defaultCurrency string
}

// BookingServiceConfig contains configuration for booking service
type BookingServiceConfig struct {
	ReservationTTL  time.Duration
	MaxPerUser      int
	DefaultCurrency string
}

// NewBookingService creates a new booking service
func NewBookingService(
	bookingRepo repository.BookingRepository,
	reservationRepo repository.ReservationRepository,
	eventPublisher EventPublisher,
	zoneSyncer ZoneSyncer,
	cfg *BookingServiceConfig,
) BookingService {
	ttl := 10 * time.Minute
	maxPerUser := 10
	currency := "THB"
	if cfg != nil {
		if cfg.ReservationTTL > 0 {
			ttl = cfg.ReservationTTL
		}
		if cfg.MaxPerUser > 0 {
			maxPerUser = cfg.MaxPerUser
		}
		if cfg.DefaultCurrency != "" {
			currency = cfg.DefaultCurrency
		}
	}
	// Use NoOpEventPublisher if none provided
	if eventPublisher == nil {
		eventPublisher = NewNoOpEventPublisher()
	}
	return &bookingService{
		bookingRepo:     bookingRepo,
		reservationRepo: reservationRepo,
		eventPublisher:  eventPublisher,
		zoneSyncer:      zoneSyncer,
		reservationTTL:  ttl,
		maxPerUser:      maxPerUser,
		defaultCurrency: currency,
	}
}

// ReserveSeats reserves seats for a user with idempotency support
func (s *bookingService) ReserveSeats(ctx context.Context, userID string, req *dto.ReserveSeatsRequest) (*dto.ReserveSeatsResponse, error) {
	// Validate request
	if req == nil {
		return nil, domain.ErrInvalidQuantity
	}
	if req.Quantity <= 0 {
		return nil, domain.ErrInvalidQuantity
	}
	if req.EventID == "" {
		return nil, domain.ErrInvalidEventID
	}
	if req.ZoneID == "" {
		return nil, domain.ErrInvalidZoneID
	}
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}
	if req.ShowID == "" {
		return nil, domain.ErrInvalidShowID
	}

	// Get tenant_id from show if not provided in request
	tenantID := req.TenantID
	if tenantID == "" {
		var err error
		tenantID, err = s.bookingRepo.GetTenantIDByShowID(ctx, req.ShowID)
		if err != nil {
			return nil, err
		}
	}

	// Check idempotency key if provided
	if req.IdempotencyKey != "" {
		existingBooking, err := s.bookingRepo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil && existingBooking != nil {
			// Return existing booking for idempotent request
			return &dto.ReserveSeatsResponse{
				BookingID:  existingBooking.ID,
				Status:     string(existingBooking.Status),
				ExpiresAt:  existingBooking.ExpiresAt,
				TotalPrice: existingBooking.TotalPrice,
			}, nil
		}
		// If error is not ErrBookingNotFound, it's a real error
		if err != nil && err != domain.ErrBookingNotFound {
			return nil, err
		}
	}

	// Get unit price from zone (TODO: integrate with zone service)
	unitPrice := req.UnitPrice
	if unitPrice <= 0 {
		unitPrice = 100.00 // Default price for testing
	}
	totalPrice := unitPrice * float64(req.Quantity)

	// Reserve seats in Redis atomically
	params := repository.ReserveParams{
		ZoneID:     req.ZoneID,
		UserID:     userID,
		EventID:    req.EventID,
		Quantity:   req.Quantity,
		MaxPerUser: s.maxPerUser,
		TTLSeconds: int(s.reservationTTL.Seconds()),
		Price:      unitPrice,
	}

	result, err := s.reservationRepo.ReserveSeats(ctx, params)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		switch result.ErrorCode {
		case "INSUFFICIENT_STOCK":
			return nil, domain.ErrInsufficientSeats
		case "USER_LIMIT_EXCEEDED":
			return nil, domain.ErrMaxTicketsExceeded
		case "ZONE_NOT_FOUND":
			// Auto-sync zone from ticket service and retry once
			if s.zoneSyncer != nil {
				if syncErr := s.zoneSyncer.SyncZone(ctx, req.ZoneID); syncErr == nil {
					// Retry the reservation after sync
					retryResult, retryErr := s.reservationRepo.ReserveSeats(ctx, params)
					if retryErr != nil {
						return nil, retryErr
					}
					if retryResult.Success {
						result = retryResult
						// Continue to create booking record below
						goto createBooking
					}
					// Retry failed, return the error
					switch retryResult.ErrorCode {
					case "INSUFFICIENT_STOCK":
						return nil, domain.ErrInsufficientSeats
					case "USER_LIMIT_EXCEEDED":
						return nil, domain.ErrMaxTicketsExceeded
					default:
						return nil, domain.ErrZoneNotFound
					}
				}
			}
			return nil, domain.ErrZoneNotFound
		case "INVALID_QUANTITY":
			return nil, domain.ErrInvalidQuantity
		default:
			return nil, domain.ErrInvalidBookingStatus
		}
	}

createBooking:

	// Create booking record in PostgreSQL
	now := time.Now()
	booking := &domain.Booking{
		ID:             result.BookingID,
		TenantID:       tenantID,
		UserID:         userID,
		EventID:        req.EventID,
		ShowID:         req.ShowID,
		ZoneID:         req.ZoneID,
		Quantity:       req.Quantity,
		UnitPrice:      unitPrice,
		TotalPrice:     totalPrice,
		Currency:       s.defaultCurrency,
		Status:         domain.BookingStatusReserved,
		IdempotencyKey: req.IdempotencyKey,
		ReservedAt:     now,
		ExpiresAt:      now.Add(s.reservationTTL),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.bookingRepo.Create(ctx, booking); err != nil {
		// If PostgreSQL insert fails, we should release Redis reservation
		// But for now, let Redis TTL handle cleanup
		return nil, err
	}

	// Publish booking created event (async, don't block on failure)
	go func() {
		if pubErr := s.eventPublisher.PublishBookingCreated(context.Background(), booking); pubErr != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
		}
	}()

	return &dto.ReserveSeatsResponse{
		BookingID:  booking.ID,
		Status:     string(booking.Status),
		ExpiresAt:  booking.ExpiresAt,
		TotalPrice: booking.TotalPrice,
	}, nil
}

// ConfirmBooking confirms a reservation with payment
func (s *bookingService) ConfirmBooking(ctx context.Context, bookingID, userID string, req *dto.ConfirmBookingRequest) (*dto.ConfirmBookingResponse, error) {
	// Validate inputs
	if bookingID == "" {
		return nil, domain.ErrInvalidBookingID
	}
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	// Get booking from PostgreSQL
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !booking.BelongsToUser(userID) {
		return nil, domain.ErrInvalidUserID
	}

	// Check if booking can be confirmed
	if booking.IsConfirmed() {
		return nil, domain.ErrAlreadyConfirmed
	}
	if booking.IsCancelled() {
		return nil, domain.ErrAlreadyReleased
	}
	if booking.IsExpired() {
		return nil, domain.ErrBookingExpired
	}

	paymentID := ""
	if req != nil {
		paymentID = req.PaymentID
	}

	// Confirm in Redis first
	redisResult, err := s.reservationRepo.ConfirmBooking(ctx, bookingID, userID, paymentID)
	if err != nil {
		return nil, err
	}

	if !redisResult.Success {
		switch redisResult.ErrorCode {
		case "RESERVATION_NOT_FOUND":
			return nil, domain.ErrReservationNotFound
		case "INVALID_USER":
			return nil, domain.ErrInvalidUserID
		case "ALREADY_CONFIRMED":
			return nil, domain.ErrAlreadyConfirmed
		case "RESERVATION_EXPIRED":
			return nil, domain.ErrReservationExpired
		default:
			return nil, domain.ErrInvalidBookingStatus
		}
	}

	// Update booking in PostgreSQL
	if err := s.bookingRepo.Confirm(ctx, bookingID, paymentID); err != nil {
		return nil, err
	}

	// Generate confirmation code
	confirmationCode := generateConfirmationCode()

	// Update booking object for event publishing
	booking.Status = domain.BookingStatusConfirmed
	booking.PaymentID = paymentID
	booking.ConfirmationCode = confirmationCode
	now := time.Now()
	booking.ConfirmedAt = &now

	// Publish booking confirmed event (async, don't block on failure)
	go func() {
		if pubErr := s.eventPublisher.PublishBookingConfirmed(context.Background(), booking); pubErr != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
		}
	}()

	return &dto.ConfirmBookingResponse{
		BookingID:        bookingID,
		Status:           "confirmed",
		ConfirmedAt:      now,
		ConfirmationCode: confirmationCode,
	}, nil
}

// CancelBooking cancels a reservation
func (s *bookingService) CancelBooking(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
	// Validate inputs
	if bookingID == "" {
		return nil, domain.ErrInvalidBookingID
	}
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	// Get booking from PostgreSQL
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !booking.BelongsToUser(userID) {
		return nil, domain.ErrInvalidUserID
	}

	// Check if booking can be cancelled
	if booking.IsConfirmed() {
		return nil, domain.ErrAlreadyConfirmed
	}
	if booking.IsCancelled() {
		return nil, domain.ErrAlreadyReleased
	}

	// Release seats in Redis
	releaseResult, err := s.reservationRepo.ReleaseSeats(ctx, bookingID, userID)
	if err != nil {
		return nil, err
	}

	if !releaseResult.Success {
		switch releaseResult.ErrorCode {
		case "RESERVATION_NOT_FOUND":
			// If not found in Redis, it might have expired
			// Still proceed to cancel in PostgreSQL
		case "INVALID_USER":
			return nil, domain.ErrInvalidUserID
		case "ALREADY_RELEASED":
			return nil, domain.ErrAlreadyReleased
		}
	}

	// Cancel in PostgreSQL
	if err := s.bookingRepo.Cancel(ctx, bookingID); err != nil {
		return nil, err
	}

	// Update booking object for event publishing
	booking.Status = domain.BookingStatusCancelled
	now := time.Now()
	booking.CancelledAt = &now

	// Publish booking cancelled event (async, don't block on failure)
	go func() {
		if pubErr := s.eventPublisher.PublishBookingCancelled(context.Background(), booking); pubErr != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
		}
	}()

	return &dto.ReleaseBookingResponse{
		BookingID: bookingID,
		Status:    "cancelled",
		Message:   "Booking cancelled successfully",
	}, nil
}

// ReleaseBooking releases a reservation (alias for CancelBooking)
func (s *bookingService) ReleaseBooking(ctx context.Context, bookingID, userID string) (*dto.ReleaseBookingResponse, error) {
	return s.CancelBooking(ctx, bookingID, userID)
}

// GetBooking retrieves a booking by ID
func (s *bookingService) GetBooking(ctx context.Context, bookingID, userID string) (*dto.BookingResponse, error) {
	// Validate inputs
	if bookingID == "" {
		return nil, domain.ErrInvalidBookingID
	}
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !booking.BelongsToUser(userID) {
		return nil, domain.ErrInvalidUserID
	}

	return dto.FromDomain(booking), nil
}

// GetUserBookings retrieves all bookings for a user
func (s *bookingService) GetUserBookings(ctx context.Context, userID string, page, pageSize int) (*dto.PaginatedResponse, error) {
	// Validate input
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	bookings, err := s.bookingRepo.GetByUserID(ctx, userID, pageSize+1, offset) // Fetch one extra to check if there are more
	if err != nil {
		return nil, err
	}

	hasMore := len(bookings) > pageSize
	if hasMore {
		bookings = bookings[:pageSize]
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

// GetPendingBookings retrieves pending reservations (reserved status)
func (s *bookingService) GetPendingBookings(ctx context.Context, limit int) ([]*dto.BookingResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	bookings, err := s.bookingRepo.GetExpiredReservations(ctx, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.BookingResponse, len(bookings))
	for i, b := range bookings {
		responses[i] = dto.FromDomain(b)
	}

	return responses, nil
}

// ExpireReservations marks expired reservations as expired
func (s *bookingService) ExpireReservations(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}

	// Get expired reservations
	bookings, err := s.bookingRepo.GetExpiredReservations(ctx, limit)
	if err != nil {
		return 0, err
	}

	expiredCount := 0
	for _, booking := range bookings {
		// Mark as expired in PostgreSQL
		if err := s.bookingRepo.MarkAsExpired(ctx, booking.ID); err != nil {
			continue // Log error but continue processing
		}

		// Update booking object for event publishing
		booking.Status = domain.BookingStatusExpired

		// Publish booking expired event (async, don't block on failure)
		go func(b *domain.Booking) {
			if pubErr := s.eventPublisher.PublishBookingExpired(context.Background(), b); pubErr != nil {
				// Log error but don't fail the request
				// TODO: Add proper logging
			}
		}(booking)

		expiredCount++
	}

	return expiredCount, nil
}

// generateConfirmationCode generates a random confirmation code
func generateConfirmationCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return uuid.New().String()[:8]
	}
	return hex.EncodeToString(bytes)
}
