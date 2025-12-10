package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
)

// PostgresBookingRepository implements BookingRepository using PostgreSQL with pgxpool
type PostgresBookingRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresBookingRepository creates a new PostgresBookingRepository
func NewPostgresBookingRepository(pool *pgxpool.Pool) *PostgresBookingRepository {
	return &PostgresBookingRepository{pool: pool}
}

// Create creates a new booking record in the database
func (r *PostgresBookingRepository) Create(ctx context.Context, booking *domain.Booking) error {
	query := `
		INSERT INTO bookings (
			id, tenant_id, user_id, event_id, show_id, zone_id,
			quantity, unit_price, total_amount, currency, status,
			idempotency_key, reserved_at, reservation_expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16
		)
	`

	_, err := r.pool.Exec(ctx, query,
		booking.ID,
		nullString(booking.TenantID),
		booking.UserID,
		booking.EventID,
		nullString(booking.ShowID),
		booking.ZoneID,
		booking.Quantity,
		booking.UnitPrice,
		booking.TotalPrice,
		booking.Currency,
		booking.Status.String(),
		nullString(booking.IdempotencyKey),
		booking.ReservedAt,
		booking.ExpiresAt,
		booking.CreatedAt,
		booking.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create booking: %w", err)
	}

	return nil
}

// GetByID retrieves a booking by its ID
func (r *PostgresBookingRepository) GetByID(ctx context.Context, id string) (*domain.Booking, error) {
	query := `
		SELECT
			id, tenant_id, user_id, event_id, show_id, zone_id,
			quantity, unit_price, total_amount, currency, status,
			idempotency_key, reserved_at, reservation_expires_at,
			confirmed_at, confirmation_code, payment_id,
			cancelled_at, created_at, updated_at
		FROM bookings
		WHERE id = $1
	`

	booking := &domain.Booking{}
	var (
		status           string
		tenantID         *string
		showID           *string
		idempotencyKey   *string
		reservedAt       *time.Time
		expiresAt        *time.Time
		confirmedAt      *time.Time
		confirmationCode *string
		paymentID        *string
		cancelledAt      *time.Time
	)

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&booking.ID,
		&tenantID,
		&booking.UserID,
		&booking.EventID,
		&showID,
		&booking.ZoneID,
		&booking.Quantity,
		&booking.UnitPrice,
		&booking.TotalPrice,
		&booking.Currency,
		&status,
		&idempotencyKey,
		&reservedAt,
		&expiresAt,
		&confirmedAt,
		&confirmationCode,
		&paymentID,
		&cancelledAt,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBookingNotFound
		}
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	booking.Status = domain.BookingStatus(status)
	if tenantID != nil {
		booking.TenantID = *tenantID
	}
	if showID != nil {
		booking.ShowID = *showID
	}
	if idempotencyKey != nil {
		booking.IdempotencyKey = *idempotencyKey
	}
	if reservedAt != nil {
		booking.ReservedAt = *reservedAt
	}
	if expiresAt != nil {
		booking.ExpiresAt = *expiresAt
	}
	if confirmedAt != nil {
		booking.ConfirmedAt = confirmedAt
	}
	if confirmationCode != nil {
		booking.ConfirmationCode = *confirmationCode
	}
	if paymentID != nil {
		booking.PaymentID = *paymentID
	}
	if cancelledAt != nil {
		booking.CancelledAt = cancelledAt
	}

	return booking, nil
}

// GetByUserID retrieves all bookings for a user
func (r *PostgresBookingRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Booking, error) {
	query := `
		SELECT
			id, tenant_id, user_id, event_id, show_id, zone_id,
			quantity, unit_price, total_amount, currency, status,
			idempotency_key, reserved_at, reservation_expires_at,
			confirmed_at, confirmation_code, payment_id,
			cancelled_at, created_at, updated_at
		FROM bookings
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookings by user ID: %w", err)
	}
	defer rows.Close()

	var bookings []*domain.Booking
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bookings: %w", err)
	}

	return bookings, nil
}

// Update updates an existing booking
func (r *PostgresBookingRepository) Update(ctx context.Context, booking *domain.Booking) error {
	query := `
		UPDATE bookings SET
			quantity = $2,
			unit_price = $3,
			total_amount = $4,
			status = $5,
			confirmed_at = $6,
			payment_id = $7,
			cancelled_at = $8,
			updated_at = $9
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		booking.ID,
		booking.Quantity,
		booking.UnitPrice,
		booking.TotalPrice,
		booking.Status.String(),
		booking.ConfirmedAt,
		nullString(booking.PaymentID),
		booking.CancelledAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update booking: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}

	return nil
}

// UpdateStatus updates only the status of a booking
func (r *PostgresBookingRepository) UpdateStatus(ctx context.Context, id string, status domain.BookingStatus) error {
	query := `
		UPDATE bookings SET
			status = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, status.String(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to update booking status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}

	return nil
}

// Delete deletes a booking by its ID
func (r *PostgresBookingRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM bookings WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete booking: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}

	return nil
}

// Confirm confirms a booking with payment info
func (r *PostgresBookingRepository) Confirm(ctx context.Context, id, paymentID string) error {
	query := `
		UPDATE bookings SET
			status = $2,
			payment_id = $3,
			confirmed_at = $4,
			updated_at = $5
		WHERE id = $1 AND status = 'reserved'
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, domain.BookingStatusConfirmed.String(), paymentID, now, now)
	if err != nil {
		return fmt.Errorf("failed to confirm booking: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if booking exists
		var exists bool
		err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bookings WHERE id = $1)", id).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check booking existence: %w", err)
		}
		if !exists {
			return domain.ErrBookingNotFound
		}
		return domain.ErrAlreadyConfirmed
	}

	return nil
}

// Cancel cancels a booking
func (r *PostgresBookingRepository) Cancel(ctx context.Context, id string) error {
	query := `
		UPDATE bookings SET
			status = $2,
			cancelled_at = $3,
			updated_at = $4
		WHERE id = $1 AND status = 'reserved'
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, domain.BookingStatusCancelled.String(), now, now)
	if err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if booking exists and its status
		var status string
		err := r.pool.QueryRow(ctx, "SELECT status FROM bookings WHERE id = $1", id).Scan(&status)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrBookingNotFound
			}
			return fmt.Errorf("failed to check booking status: %w", err)
		}
		if status == "confirmed" {
			return domain.ErrAlreadyConfirmed
		}
		if status == "cancelled" {
			return domain.ErrAlreadyReleased
		}
		return domain.ErrInvalidBookingStatus
	}

	return nil
}

// GetExpiredReservations gets all expired reservations
func (r *PostgresBookingRepository) GetExpiredReservations(ctx context.Context, limit int) ([]*domain.Booking, error) {
	query := `
		SELECT
			id, tenant_id, user_id, event_id, show_id, zone_id,
			quantity, unit_price, total_amount, currency, status,
			idempotency_key, reserved_at, reservation_expires_at,
			confirmed_at, confirmation_code, payment_id,
			cancelled_at, created_at, updated_at
		FROM bookings
		WHERE status = 'reserved'
			AND reservation_expires_at IS NOT NULL
			AND reservation_expires_at < $1
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired reservations: %w", err)
	}
	defer rows.Close()

	var bookings []*domain.Booking
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bookings: %w", err)
	}

	return bookings, nil
}

// MarkAsExpired marks a booking as expired
func (r *PostgresBookingRepository) MarkAsExpired(ctx context.Context, id string) error {
	query := `
		UPDATE bookings SET
			status = $2,
			status_reason = $3,
			updated_at = $4
		WHERE id = $1 AND status = 'reserved'
	`

	result, err := r.pool.Exec(ctx, query,
		id,
		domain.BookingStatusExpired.String(),
		"Reservation TTL expired",
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to mark booking as expired: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}

	return nil
}

// GetByIdempotencyKey retrieves a booking by idempotency key
func (r *PostgresBookingRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error) {
	query := `
		SELECT
			id, tenant_id, user_id, event_id, show_id, zone_id,
			quantity, unit_price, total_amount, currency, status,
			idempotency_key, reserved_at, reservation_expires_at,
			confirmed_at, confirmation_code, payment_id,
			cancelled_at, created_at, updated_at
		FROM bookings
		WHERE idempotency_key = $1
	`

	booking := &domain.Booking{}
	var (
		status           string
		tenantID         *string
		showID           *string
		idempotencyKey   *string
		reservedAt       *time.Time
		expiresAt        *time.Time
		confirmedAt      *time.Time
		confirmationCode *string
		paymentID        *string
		cancelledAt      *time.Time
	)

	err := r.pool.QueryRow(ctx, query, key).Scan(
		&booking.ID,
		&tenantID,
		&booking.UserID,
		&booking.EventID,
		&showID,
		&booking.ZoneID,
		&booking.Quantity,
		&booking.UnitPrice,
		&booking.TotalPrice,
		&booking.Currency,
		&status,
		&idempotencyKey,
		&reservedAt,
		&expiresAt,
		&confirmedAt,
		&confirmationCode,
		&paymentID,
		&cancelledAt,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found, but not an error
		}
		return nil, fmt.Errorf("failed to get booking by idempotency key: %w", err)
	}

	booking.Status = domain.BookingStatus(status)
	if tenantID != nil {
		booking.TenantID = *tenantID
	}
	if showID != nil {
		booking.ShowID = *showID
	}
	if idempotencyKey != nil {
		booking.IdempotencyKey = *idempotencyKey
	}
	if reservedAt != nil {
		booking.ReservedAt = *reservedAt
	}
	if expiresAt != nil {
		booking.ExpiresAt = *expiresAt
	}
	if confirmedAt != nil {
		booking.ConfirmedAt = confirmedAt
	}
	if confirmationCode != nil {
		booking.ConfirmationCode = *confirmationCode
	}
	if paymentID != nil {
		booking.PaymentID = *paymentID
	}
	if cancelledAt != nil {
		booking.CancelledAt = cancelledAt
	}

	return booking, nil
}

// CountByUserAndEvent counts bookings for a user on an event
func (r *PostgresBookingRepository) CountByUserAndEvent(ctx context.Context, userID, eventID string) (int, error) {
	query := `
		SELECT COUNT(*) FROM bookings
		WHERE user_id = $1 AND event_id = $2 AND status IN ('reserved', 'confirmed')
	`

	var count int
	err := r.pool.QueryRow(ctx, query, userID, eventID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count bookings: %w", err)
	}

	return count, nil
}

// scanBooking scans a row into a Booking struct
func scanBooking(rows pgx.Rows) (*domain.Booking, error) {
	booking := &domain.Booking{}
	var (
		status           string
		tenantID         *string
		showID           *string
		idempotencyKey   *string
		reservedAt       *time.Time
		expiresAt        *time.Time
		confirmedAt      *time.Time
		confirmationCode *string
		paymentID        *string
		cancelledAt      *time.Time
	)

	err := rows.Scan(
		&booking.ID,
		&tenantID,
		&booking.UserID,
		&booking.EventID,
		&showID,
		&booking.ZoneID,
		&booking.Quantity,
		&booking.UnitPrice,
		&booking.TotalPrice,
		&booking.Currency,
		&status,
		&idempotencyKey,
		&reservedAt,
		&expiresAt,
		&confirmedAt,
		&confirmationCode,
		&paymentID,
		&cancelledAt,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan booking: %w", err)
	}

	booking.Status = domain.BookingStatus(status)
	if tenantID != nil {
		booking.TenantID = *tenantID
	}
	if showID != nil {
		booking.ShowID = *showID
	}
	if idempotencyKey != nil {
		booking.IdempotencyKey = *idempotencyKey
	}
	if reservedAt != nil {
		booking.ReservedAt = *reservedAt
	}
	if expiresAt != nil {
		booking.ExpiresAt = *expiresAt
	}
	if confirmedAt != nil {
		booking.ConfirmedAt = confirmedAt
	}
	if confirmationCode != nil {
		booking.ConfirmationCode = *confirmationCode
	}
	if paymentID != nil {
		booking.PaymentID = *paymentID
	}
	if cancelledAt != nil {
		booking.CancelledAt = cancelledAt
	}

	return booking, nil
}

// Helper function to convert empty string to nil pointer
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Ensure PostgresBookingRepository implements BookingRepository
var _ BookingRepository = (*PostgresBookingRepository)(nil)
