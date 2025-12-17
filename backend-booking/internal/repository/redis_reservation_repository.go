package repository

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.reservation.reserve_seats")
	defer span.End()

	span.SetAttributes(
		attribute.String("zone_id", params.ZoneID),
		attribute.String("user_id", params.UserID),
		attribute.String("event_id", params.EventID),
		attribute.Int("quantity", params.Quantity),
	)

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
		span.RecordError(result.Err())
		span.SetStatus(codes.Error, result.Err().Error())
		return nil, fmt.Errorf("failed to execute reserve_seats script: %w", result.Err())
	}

	// Parse result
	values, err := result.Slice()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	if len(values) < 3 {
		span.SetStatus(codes.Error, "unexpected result length")
		return nil, fmt.Errorf("unexpected script result length: %d", len(values))
	}

	success, _ := toInt64(values[0])
	if success == 1 {
		availableSeats, _ := toInt64(values[1])
		userReserved, _ := toInt64(values[2])
		span.SetAttributes(
			attribute.String("booking_id", bookingID),
			attribute.Int64("available_seats", availableSeats),
		)
		span.SetStatus(codes.Ok, "")
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
	span.SetAttributes(attribute.String("error_code", errorCode))
	span.SetStatus(codes.Error, errorCode)
	return &ReserveResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// ConfirmBooking confirms a reservation and makes it permanent
func (r *RedisReservationRepository) ConfirmBooking(ctx context.Context, bookingID, userID, paymentID string) (*ConfirmResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.reservation.confirm")
	defer span.End()

	span.SetAttributes(
		attribute.String("booking_id", bookingID),
		attribute.String("user_id", userID),
	)

	reservationKey := fmt.Sprintf("reservation:%s", bookingID)
	keys := []string{reservationKey}
	args := []interface{}{bookingID, userID, paymentID}

	result := r.client.EvalWithFallback(ctx, scriptConfirmBooking, confirmBookingScript, keys, args...)
	if result.Err() != nil {
		span.RecordError(result.Err())
		span.SetStatus(codes.Error, result.Err().Error())
		return nil, fmt.Errorf("failed to execute confirm_booking script: %w", result.Err())
	}

	// Parse result
	values, err := result.Slice()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	if len(values) < 3 {
		span.SetStatus(codes.Error, "unexpected result length")
		return nil, fmt.Errorf("unexpected script result length: %d", len(values))
	}

	success, _ := toInt64(values[0])
	if success == 1 {
		status, _ := values[1].(string)
		confirmedAt, _ := values[2].(string)
		span.SetStatus(codes.Ok, "")
		return &ConfirmResult{
			Success:     true,
			Status:      status,
			ConfirmedAt: confirmedAt,
		}, nil
	}

	// Error case
	errorCode, _ := values[1].(string)
	errorMessage, _ := values[2].(string)
	span.SetAttributes(attribute.String("error_code", errorCode))
	span.SetStatus(codes.Error, errorCode)
	return &ConfirmResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// ReleaseSeats releases reserved seats back to inventory
func (r *RedisReservationRepository) ReleaseSeats(ctx context.Context, bookingID, userID string) (*ReleaseResult, error) {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.reservation.release_seats")
	defer span.End()

	span.SetAttributes(
		attribute.String("booking_id", bookingID),
		attribute.String("user_id", userID),
	)

	// First, get the reservation to find the zone_id and event_id
	reservationKey := fmt.Sprintf("reservation:%s", bookingID)
	reservationData, err := r.client.HGetAll(ctx, reservationKey).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	if len(reservationData) == 0 {
		span.SetStatus(codes.Error, "RESERVATION_NOT_FOUND")
		return &ReleaseResult{
			Success:      false,
			ErrorCode:    "RESERVATION_NOT_FOUND",
			ErrorMessage: "Reservation does not exist or has expired",
		}, nil
	}

	zoneID := reservationData["zone_id"]
	eventID := reservationData["event_id"]

	span.SetAttributes(
		attribute.String("zone_id", zoneID),
		attribute.String("event_id", eventID),
	)

	// Build Redis keys
	zoneAvailabilityKey := fmt.Sprintf("zone:availability:%s", zoneID)
	userReservationsKey := fmt.Sprintf("user:reservations:%s:%s", userID, eventID)

	keys := []string{zoneAvailabilityKey, userReservationsKey, reservationKey}
	args := []interface{}{bookingID, userID}

	result := r.client.EvalWithFallback(ctx, scriptReleaseSeats, releaseSeatsScript, keys, args...)
	if result.Err() != nil {
		span.RecordError(result.Err())
		span.SetStatus(codes.Error, result.Err().Error())
		return nil, fmt.Errorf("failed to execute release_seats script: %w", result.Err())
	}

	// Parse result
	values, err := result.Slice()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	if len(values) < 3 {
		span.SetStatus(codes.Error, "unexpected result length")
		return nil, fmt.Errorf("unexpected script result length: %d", len(values))
	}

	success, _ := toInt64(values[0])
	if success == 1 {
		availableSeats, _ := toInt64(values[1])
		userReserved, _ := toInt64(values[2])
		span.SetAttributes(attribute.Int64("available_seats", availableSeats))
		span.SetStatus(codes.Ok, "")
		return &ReleaseResult{
			Success:        true,
			AvailableSeats: availableSeats,
			UserReserved:   userReserved,
		}, nil
	}

	// Error case
	errorCode, _ := values[1].(string)
	errorMessage, _ := values[2].(string)
	span.SetAttributes(attribute.String("error_code", errorCode))
	span.SetStatus(codes.Error, errorCode)
	return &ReleaseResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// GetZoneAvailability gets the current available seats for a zone
func (r *RedisReservationRepository) GetZoneAvailability(ctx context.Context, zoneID string) (int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.reservation.get_zone_availability")
	defer span.End()

	span.SetAttributes(attribute.String("zone_id", zoneID))

	key := fmt.Sprintf("zone:availability:%s", zoneID)
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			span.SetStatus(codes.Ok, "zone not found")
			return 0, nil // Zone not found, return 0
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, fmt.Errorf("failed to get zone availability: %w", err)
	}

	seats, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, fmt.Errorf("failed to parse availability: %w", err)
	}

	span.SetAttributes(attribute.Int64("available_seats", seats))
	span.SetStatus(codes.Ok, "")
	return seats, nil
}

// SetZoneAvailability sets the available seats for a zone (for initialization)
func (r *RedisReservationRepository) SetZoneAvailability(ctx context.Context, zoneID string, seats int64) error {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.reservation.set_zone_availability")
	defer span.End()

	span.SetAttributes(
		attribute.String("zone_id", zoneID),
		attribute.Int64("seats", seats),
	)

	key := fmt.Sprintf("zone:availability:%s", zoneID)
	err := r.client.Set(ctx, key, seats, 0).Err()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to set zone availability: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// GetReservation gets a reservation by booking ID
func (r *RedisReservationRepository) GetReservation(ctx context.Context, bookingID string) (map[string]string, error) {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.reservation.get")
	defer span.End()

	span.SetAttributes(attribute.String("booking_id", bookingID))

	key := fmt.Sprintf("reservation:%s", bookingID)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// GetUserReservedCount gets the total reserved count for a user on an event
func (r *RedisReservationRepository) GetUserReservedCount(ctx context.Context, userID, eventID string) (int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "repo.redis.reservation.get_user_count")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", eventID),
	)

	key := fmt.Sprintf("user:reservations:%s:%s", userID, eventID)
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			span.SetStatus(codes.Ok, "no reservations")
			return 0, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, fmt.Errorf("failed to get user reserved count: %w", err)
	}

	count, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, fmt.Errorf("failed to parse reserved count: %w", err)
	}

	span.SetAttributes(attribute.Int64("count", count))
	span.SetStatus(codes.Ok, "")
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
