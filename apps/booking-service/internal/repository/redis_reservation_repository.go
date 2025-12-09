package repository

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

//go:embed scripts/reserve_seats.lua
var reserveSeatsScript string

//go:embed scripts/release_seats.lua
var releaseSeatsScript string

//go:embed scripts/confirm_booking.lua
var confirmBookingScript string

// Script names for caching
const (
	scriptReserveSeats   = "reserve_seats"
	scriptReleaseSeats   = "release_seats"
	scriptConfirmBooking = "confirm_booking"
)

// RedisReservationRepository implements ReservationRepository using Redis
type RedisReservationRepository struct {
	client *pkgredis.Client
}

// NewRedisReservationRepository creates a new RedisReservationRepository
func NewRedisReservationRepository(client *pkgredis.Client) *RedisReservationRepository {
	return &RedisReservationRepository{client: client}
}

// LoadScripts loads all Lua scripts into Redis
func (r *RedisReservationRepository) LoadScripts(ctx context.Context) error {
	scripts := map[string]string{
		scriptReserveSeats:   reserveSeatsScript,
		scriptReleaseSeats:   releaseSeatsScript,
		scriptConfirmBooking: confirmBookingScript,
	}

	for name, script := range scripts {
		if _, err := r.client.LoadScript(ctx, name, script); err != nil {
			return fmt.Errorf("failed to load script %s: %w", name, err)
		}
	}

	return nil
}

// ReserveSeats atomically reserves seats using Lua script
func (r *RedisReservationRepository) ReserveSeats(ctx context.Context, params ReserveParams) (*ReserveResult, error) {
	// Generate booking ID if not provided
	bookingID := uuid.New().String()

	// Build Redis keys
	zoneAvailabilityKey := fmt.Sprintf("zone:availability:%s", params.ZoneID)
	userReservationsKey := fmt.Sprintf("user:reservations:%s:%s", params.UserID, params.EventID)
	reservationKey := fmt.Sprintf("reservation:%s", bookingID)

	keys := []string{zoneAvailabilityKey, userReservationsKey, reservationKey}
	args := []interface{}{
		params.Quantity,    // ARGV[1]: quantity
		params.MaxPerUser,  // ARGV[2]: max_per_user
		params.UserID,      // ARGV[3]: user_id
		bookingID,          // ARGV[4]: booking_id
		params.ZoneID,      // ARGV[5]: zone_id
		params.EventID,     // ARGV[6]: event_id
		"",                 // ARGV[7]: show_id (optional)
		params.Price,       // ARGV[8]: unit_price
		params.TTLSeconds,  // ARGV[9]: ttl_seconds
	}

	result := r.client.EvalWithFallback(ctx, scriptReserveSeats, reserveSeatsScript, keys, args...)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to execute reserve_seats script: %w", result.Err())
	}

	// Parse result
	values, err := result.Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	if len(values) < 3 {
		return nil, fmt.Errorf("unexpected script result length: %d", len(values))
	}

	success, _ := toInt64(values[0])
	if success == 1 {
		availableSeats, _ := toInt64(values[1])
		userReserved, _ := toInt64(values[2])
		return &ReserveResult{
			Success:        true,
			BookingID:      bookingID,
			AvailableSeats: availableSeats,
			UserReserved:   userReserved,
		}, nil
	}

	// Error case
	errorCode, _ := values[1].(string)
	errorMessage, _ := values[2].(string)
	return &ReserveResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// ConfirmBooking confirms a reservation and makes it permanent
func (r *RedisReservationRepository) ConfirmBooking(ctx context.Context, bookingID, userID, paymentID string) (*ConfirmResult, error) {
	reservationKey := fmt.Sprintf("reservation:%s", bookingID)
	keys := []string{reservationKey}
	args := []interface{}{bookingID, userID, paymentID}

	result := r.client.EvalWithFallback(ctx, scriptConfirmBooking, confirmBookingScript, keys, args...)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to execute confirm_booking script: %w", result.Err())
	}

	// Parse result
	values, err := result.Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	if len(values) < 3 {
		return nil, fmt.Errorf("unexpected script result length: %d", len(values))
	}

	success, _ := toInt64(values[0])
	if success == 1 {
		status, _ := values[1].(string)
		confirmedAt, _ := values[2].(string)
		return &ConfirmResult{
			Success:     true,
			Status:      status,
			ConfirmedAt: confirmedAt,
		}, nil
	}

	// Error case
	errorCode, _ := values[1].(string)
	errorMessage, _ := values[2].(string)
	return &ConfirmResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// ReleaseSeats releases reserved seats back to inventory
func (r *RedisReservationRepository) ReleaseSeats(ctx context.Context, bookingID, userID string) (*ReleaseResult, error) {
	// First, get the reservation to find the zone_id and event_id
	reservationKey := fmt.Sprintf("reservation:%s", bookingID)
	reservationData, err := r.client.HGetAll(ctx, reservationKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	if len(reservationData) == 0 {
		return &ReleaseResult{
			Success:      false,
			ErrorCode:    "RESERVATION_NOT_FOUND",
			ErrorMessage: "Reservation does not exist or has expired",
		}, nil
	}

	zoneID := reservationData["zone_id"]
	eventID := reservationData["event_id"]

	// Build Redis keys
	zoneAvailabilityKey := fmt.Sprintf("zone:availability:%s", zoneID)
	userReservationsKey := fmt.Sprintf("user:reservations:%s:%s", userID, eventID)

	keys := []string{zoneAvailabilityKey, userReservationsKey, reservationKey}
	args := []interface{}{bookingID, userID}

	result := r.client.EvalWithFallback(ctx, scriptReleaseSeats, releaseSeatsScript, keys, args...)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to execute release_seats script: %w", result.Err())
	}

	// Parse result
	values, err := result.Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	if len(values) < 3 {
		return nil, fmt.Errorf("unexpected script result length: %d", len(values))
	}

	success, _ := toInt64(values[0])
	if success == 1 {
		availableSeats, _ := toInt64(values[1])
		userReserved, _ := toInt64(values[2])
		return &ReleaseResult{
			Success:        true,
			AvailableSeats: availableSeats,
			UserReserved:   userReserved,
		}, nil
	}

	// Error case
	errorCode, _ := values[1].(string)
	errorMessage, _ := values[2].(string)
	return &ReleaseResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// GetZoneAvailability gets the current available seats for a zone
func (r *RedisReservationRepository) GetZoneAvailability(ctx context.Context, zoneID string) (int64, error) {
	key := fmt.Sprintf("zone:availability:%s", zoneID)
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return 0, nil // Zone not found, return 0
		}
		return 0, fmt.Errorf("failed to get zone availability: %w", err)
	}

	seats, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse availability: %w", err)
	}

	return seats, nil
}

// SetZoneAvailability sets the available seats for a zone (for initialization)
func (r *RedisReservationRepository) SetZoneAvailability(ctx context.Context, zoneID string, seats int64) error {
	key := fmt.Sprintf("zone:availability:%s", zoneID)
	err := r.client.Set(ctx, key, seats, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set zone availability: %w", err)
	}
	return nil
}

// GetReservation gets a reservation by booking ID
func (r *RedisReservationRepository) GetReservation(ctx context.Context, bookingID string) (map[string]string, error) {
	key := fmt.Sprintf("reservation:%s", bookingID)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}
	return result, nil
}

// GetUserReservedCount gets the total reserved count for a user on an event
func (r *RedisReservationRepository) GetUserReservedCount(ctx context.Context, userID, eventID string) (int64, error) {
	key := fmt.Sprintf("user:reservations:%s:%s", userID, eventID)
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get user reserved count: %w", err)
	}

	count, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse reserved count: %w", err)
	}

	return count, nil
}

// Helper function to convert interface{} to int64
func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case float64:
		return int64(val), true
	case string:
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

// Ensure RedisReservationRepository implements ReservationRepository
var _ ReservationRepository = (*RedisReservationRepository)(nil)
