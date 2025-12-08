package scripts

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/reserve_seats.lua
var ReserveSeatsScript string

// ReserveSeatsParams holds parameters for seat reservation
type ReserveSeatsParams struct {
	ZoneID     string
	UserID     string
	EventID    string
	ShowID     string
	BookingID  string
	Quantity   int
	MaxPerUser int
	UnitPrice  float64
	TTLSeconds int // Default: 600 (10 minutes)
}

// ReserveSeatsResult holds the result of a seat reservation
type ReserveSeatsResult struct {
	Success           bool
	RemainingSeats    int64
	UserTotalReserved int64
	ErrorCode         string
	ErrorMessage      string
}

// ReserveSeats executes the seat reservation Lua script
func ReserveSeats(ctx context.Context, client *redis.Client, params ReserveSeatsParams) (*ReserveSeatsResult, error) {
	// Set defaults
	if params.TTLSeconds <= 0 {
		params.TTLSeconds = 600 // 10 minutes
	}
	if params.MaxPerUser <= 0 {
		params.MaxPerUser = 10 // Default max per user
	}

	// Build keys
	zoneAvailabilityKey := fmt.Sprintf("zone:availability:%s", params.ZoneID)
	userReservationsKey := fmt.Sprintf("user:reservations:%s:%s", params.UserID, params.EventID)
	reservationKey := fmt.Sprintf("reservation:%s", params.BookingID)

	keys := []string{
		zoneAvailabilityKey,
		userReservationsKey,
		reservationKey,
	}

	args := []interface{}{
		params.Quantity,
		params.MaxPerUser,
		params.UserID,
		params.BookingID,
		params.ZoneID,
		params.EventID,
		params.ShowID,
		fmt.Sprintf("%.2f", params.UnitPrice),
		params.TTLSeconds,
	}

	// Execute Lua script
	result, err := client.Eval(ctx, ReserveSeatsScript, keys, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to execute reserve_seats script: %w", err)
	}

	// Parse result
	return parseReserveSeatsResult(result)
}

func parseReserveSeatsResult(result interface{}) (*ReserveSeatsResult, error) {
	arr, ok := result.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, fmt.Errorf("unexpected result format: %v", result)
	}

	success, _ := arr[0].(int64)

	if success == 1 {
		// Success: {1, remaining_seats, user_total_reserved}
		remaining, _ := arr[1].(int64)
		userReserved, _ := arr[2].(int64)
		return &ReserveSeatsResult{
			Success:           true,
			RemainingSeats:    remaining,
			UserTotalReserved: userReserved,
		}, nil
	}

	// Error: {0, error_code, error_message}
	errorCode, _ := arr[1].(string)
	errorMessage, _ := arr[2].(string)
	return &ReserveSeatsResult{
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}, nil
}

// Error codes returned by the Lua script
const (
	ErrInsufficientStock = "INSUFFICIENT_STOCK"
	ErrUserLimitExceeded = "USER_LIMIT_EXCEEDED"
	ErrInvalidQuantity   = "INVALID_QUANTITY"
	ErrZoneNotFound      = "ZONE_NOT_FOUND"
)
