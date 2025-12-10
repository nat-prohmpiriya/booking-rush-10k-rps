package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/domain"
)

// TransactionalBookingRepository implements BookingRepository with outbox support
type TransactionalBookingRepository struct {
	pool       *pgxpool.Pool
	outboxRepo *PostgresOutboxRepository
}

// NewTransactionalBookingRepository creates a new TransactionalBookingRepository
func NewTransactionalBookingRepository(pool *pgxpool.Pool) *TransactionalBookingRepository {
	return &TransactionalBookingRepository{
		pool:       pool,
		outboxRepo: NewPostgresOutboxRepository(pool),
	}
}

// CreateWithOutbox creates a booking and outbox message in a single transaction
func (r *TransactionalBookingRepository) CreateWithOutbox(ctx context.Context, booking *domain.Booking, eventType domain.BookingEventType) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert booking
	if err := r.createBookingTx(ctx, tx, booking); err != nil {
		return err
	}

	// Create outbox message
	eventID := uuid.New().String()
	outboxMsg, err := domain.BookingOutboxEvent(eventType, booking, eventID)
	if err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := r.outboxRepo.CreateTx(ctx, tx, outboxMsg); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateWithOutbox updates a booking and creates outbox message in a single transaction
func (r *TransactionalBookingRepository) UpdateWithOutbox(ctx context.Context, booking *domain.Booking, eventType domain.BookingEventType) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update booking
	if err := r.updateBookingTx(ctx, tx, booking); err != nil {
		return err
	}

	// Create outbox message
	eventID := uuid.New().String()
	outboxMsg, err := domain.BookingOutboxEvent(eventType, booking, eventID)
	if err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := r.outboxRepo.CreateTx(ctx, tx, outboxMsg); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ConfirmWithOutbox confirms a booking and creates outbox message in a single transaction
func (r *TransactionalBookingRepository) ConfirmWithOutbox(ctx context.Context, id, paymentID string, booking *domain.Booking) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Confirm booking
	query := `
		UPDATE bookings SET
			status = $2,
			payment_id = $3,
			confirmed_at = $4,
			updated_at = $5
		WHERE id = $1 AND status = 'reserved'
	`

	now := time.Now()
	result, err := tx.Exec(ctx, query, id, domain.BookingStatusConfirmed.String(), paymentID, now, now)
	if err != nil {
		return fmt.Errorf("failed to confirm booking: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if booking exists
		var exists bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bookings WHERE id = $1)", id).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check booking existence: %w", err)
		}
		if !exists {
			return domain.ErrBookingNotFound
		}
		return domain.ErrAlreadyConfirmed
	}

	// Create outbox message
	eventID := uuid.New().String()
	outboxMsg, err := domain.BookingOutboxEvent(domain.BookingEventConfirmed, booking, eventID)
	if err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := r.outboxRepo.CreateTx(ctx, tx, outboxMsg); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CancelWithOutbox cancels a booking and creates outbox message in a single transaction
func (r *TransactionalBookingRepository) CancelWithOutbox(ctx context.Context, id string, booking *domain.Booking) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Cancel booking
	query := `
		UPDATE bookings SET
			status = $2,
			cancelled_at = $3,
			updated_at = $4
		WHERE id = $1 AND status = 'reserved'
	`

	now := time.Now()
	result, err := tx.Exec(ctx, query, id, domain.BookingStatusCancelled.String(), now, now)
	if err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if booking exists and its status
		var status string
		err := tx.QueryRow(ctx, "SELECT status FROM bookings WHERE id = $1", id).Scan(&status)
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

	// Create outbox message
	eventID := uuid.New().String()
	outboxMsg, err := domain.BookingOutboxEvent(domain.BookingEventCancelled, booking, eventID)
	if err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := r.outboxRepo.CreateTx(ctx, tx, outboxMsg); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// MarkAsExpiredWithOutbox marks a booking as expired with outbox
func (r *TransactionalBookingRepository) MarkAsExpiredWithOutbox(ctx context.Context, id string, booking *domain.Booking) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Mark as expired
	query := `
		UPDATE bookings SET
			status = $2,
			status_reason = $3,
			updated_at = $4
		WHERE id = $1 AND status = 'reserved'
	`

	result, err := tx.Exec(ctx, query,
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

	// Create outbox message
	eventID := uuid.New().String()
	outboxMsg, err := domain.BookingOutboxEvent(domain.BookingEventExpired, booking, eventID)
	if err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := r.outboxRepo.CreateTx(ctx, tx, outboxMsg); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// createBookingTx creates a booking within a transaction
func (r *TransactionalBookingRepository) createBookingTx(ctx context.Context, tx pgx.Tx, booking *domain.Booking) error {
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

	_, err := tx.Exec(ctx, query,
		booking.ID,
		nullStringPtr(booking.TenantID),
		booking.UserID,
		booking.EventID,
		nullStringPtr(booking.ShowID),
		booking.ZoneID,
		booking.Quantity,
		booking.UnitPrice,
		booking.TotalPrice,
		booking.Currency,
		booking.Status.String(),
		nullStringPtr(booking.IdempotencyKey),
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

// updateBookingTx updates a booking within a transaction
func (r *TransactionalBookingRepository) updateBookingTx(ctx context.Context, tx pgx.Tx, booking *domain.Booking) error {
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

	result, err := tx.Exec(ctx, query,
		booking.ID,
		booking.Quantity,
		booking.UnitPrice,
		booking.TotalPrice,
		booking.Status.String(),
		booking.ConfirmedAt,
		nullStringPtr(booking.PaymentID),
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

// GetOutboxRepo returns the outbox repository
func (r *TransactionalBookingRepository) GetOutboxRepo() *PostgresOutboxRepository {
	return r.outboxRepo
}

// nullStringPtr converts string to *string, returning nil for empty strings
func nullStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
