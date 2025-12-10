package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/payment-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
)

// PostgreSQL error code for unique violation
const pgUniqueViolationCode = "23505"

// PostgresPaymentRepository implements PaymentRepository using PostgreSQL
type PostgresPaymentRepository struct {
	db *database.PostgresDB
}

// NewPostgresPaymentRepository creates a new PostgreSQL payment repository
func NewPostgresPaymentRepository(db *database.PostgresDB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

// Create creates a new payment record
func (r *PostgresPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (
			id, booking_id, user_id, amount, currency, status, method,
			transaction_id, failure_reason, metadata, created_at, updated_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	metadataJSON, err := json.Marshal(payment.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Pool().Exec(ctx, query,
		payment.ID,
		payment.BookingID,
		payment.UserID,
		payment.Amount,
		payment.Currency,
		string(payment.Status),
		string(payment.Method),
		payment.TransactionID,
		payment.FailureReason,
		metadataJSON,
		payment.CreatedAt,
		payment.UpdatedAt,
		payment.CompletedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode {
			return domain.ErrPaymentAlreadyExists
		}
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

// GetByID retrieves a payment by its ID
func (r *PostgresPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `
		SELECT id, booking_id, user_id, amount, currency, status, method,
		       transaction_id, failure_reason, metadata, created_at, updated_at, completed_at
		FROM payments
		WHERE id = $1`

	return r.scanPayment(r.db.Pool().QueryRow(ctx, query, id))
}

// GetByBookingID retrieves a payment by booking ID
func (r *PostgresPaymentRepository) GetByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error) {
	query := `
		SELECT id, booking_id, user_id, amount, currency, status, method,
		       transaction_id, failure_reason, metadata, created_at, updated_at, completed_at
		FROM payments
		WHERE booking_id = $1`

	return r.scanPayment(r.db.Pool().QueryRow(ctx, query, bookingID))
}

// GetByUserID retrieves all payments for a user
func (r *PostgresPaymentRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	query := `
		SELECT id, booking_id, user_id, amount, currency, status, method,
		       transaction_id, failure_reason, metadata, created_at, updated_at, completed_at
		FROM payments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Pool().Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query payments: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		payment, err := r.scanPaymentFromRows(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate payments: %w", err)
	}

	return payments, nil
}

// Update updates an existing payment
func (r *PostgresPaymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	query := `
		UPDATE payments
		SET status = $2,
		    transaction_id = $3,
		    failure_reason = $4,
		    metadata = $5,
		    updated_at = $6,
		    completed_at = $7
		WHERE id = $1`

	metadataJSON, err := json.Marshal(payment.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.db.Pool().Exec(ctx, query,
		payment.ID,
		string(payment.Status),
		payment.TransactionID,
		payment.FailureReason,
		metadataJSON,
		payment.UpdatedAt,
		payment.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}

// GetByTransactionID retrieves a payment by transaction ID
func (r *PostgresPaymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*domain.Payment, error) {
	query := `
		SELECT id, booking_id, user_id, amount, currency, status, method,
		       transaction_id, failure_reason, metadata, created_at, updated_at, completed_at
		FROM payments
		WHERE transaction_id = $1`

	return r.scanPayment(r.db.Pool().QueryRow(ctx, query, transactionID))
}

// scanPayment scans a single payment from a row
func (r *PostgresPaymentRepository) scanPayment(row pgx.Row) (*domain.Payment, error) {
	var payment domain.Payment
	var status, method string
	var metadataJSON []byte

	err := row.Scan(
		&payment.ID,
		&payment.BookingID,
		&payment.UserID,
		&payment.Amount,
		&payment.Currency,
		&status,
		&method,
		&payment.TransactionID,
		&payment.FailureReason,
		&metadataJSON,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.CompletedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to scan payment: %w", err)
	}

	payment.Status = domain.PaymentStatus(status)
	payment.Method = domain.PaymentMethod(method)

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &payment.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &payment, nil
}

// scanPaymentFromRows scans a single payment from rows
func (r *PostgresPaymentRepository) scanPaymentFromRows(rows pgx.Rows) (*domain.Payment, error) {
	var payment domain.Payment
	var status, method string
	var metadataJSON []byte

	err := rows.Scan(
		&payment.ID,
		&payment.BookingID,
		&payment.UserID,
		&payment.Amount,
		&payment.Currency,
		&status,
		&method,
		&payment.TransactionID,
		&payment.FailureReason,
		&metadataJSON,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.CompletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan payment: %w", err)
	}

	payment.Status = domain.PaymentStatus(status)
	payment.Method = domain.PaymentMethod(method)

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &payment.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &payment, nil
}
