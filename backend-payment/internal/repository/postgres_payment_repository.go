package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/domain"
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
			id, tenant_id, booking_id, user_id, amount, currency, method, status,
			gateway, gateway_payment_id, gateway_charge_id, gateway_customer_id, gateway_response,
			idempotency_key, card_last_four, card_brand,
			initiated_at, processed_at, refund_amount, refund_reason, refunded_at,
			error_code, error_message, retry_count, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27
		)`

	metadataJSON, err := json.Marshal(payment.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	gatewayResponseJSON, err := json.Marshal(payment.GatewayResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal gateway_response: %w", err)
	}

	// Handle nullable method
	var method *string
	if payment.Method != "" {
		m := string(payment.Method)
		method = &m
	}

	_, err = r.db.Pool().Exec(ctx, query,
		payment.ID,
		payment.TenantID,
		payment.BookingID,
		payment.UserID,
		payment.Amount,
		payment.Currency,
		method,
		string(payment.Status),
		payment.Gateway,
		nullString(payment.GatewayPaymentID),
		nullString(payment.GatewayChargeID),
		nullString(payment.GatewayCustomerID),
		gatewayResponseJSON,
		nullString(payment.IdempotencyKey),
		nullString(payment.CardLastFour),
		nullString(payment.CardBrand),
		payment.InitiatedAt,
		payment.ProcessedAt,
		payment.RefundAmount,
		nullString(payment.RefundReason),
		payment.RefundedAt,
		nullString(payment.ErrorCode),
		nullString(payment.ErrorMessage),
		payment.RetryCount,
		metadataJSON,
		payment.CreatedAt,
		payment.UpdatedAt,
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

// nullString returns nil if string is empty, otherwise returns pointer to string
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// selectColumns defines the columns to select for payment queries
const selectColumns = `
	id, tenant_id, booking_id, user_id, amount, currency, method, status,
	gateway, gateway_payment_id, gateway_charge_id, gateway_customer_id, gateway_response,
	idempotency_key, card_last_four, card_brand,
	initiated_at, processed_at, refund_amount, refund_reason, refunded_at,
	error_code, error_message, retry_count, metadata, created_at, updated_at
`

// GetByID retrieves a payment by its ID
func (r *PostgresPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `SELECT ` + selectColumns + ` FROM payments WHERE id = $1`
	return r.scanPayment(r.db.Pool().QueryRow(ctx, query, id))
}

// GetByBookingID retrieves a payment by booking ID
func (r *PostgresPaymentRepository) GetByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error) {
	query := `SELECT ` + selectColumns + ` FROM payments WHERE booking_id = $1`
	return r.scanPayment(r.db.Pool().QueryRow(ctx, query, bookingID))
}

// GetByUserID retrieves all payments for a user
func (r *PostgresPaymentRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	query := `SELECT ` + selectColumns + ` FROM payments WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

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
		    method = $3,
		    gateway = $4,
		    gateway_payment_id = $5,
		    gateway_charge_id = $6,
		    gateway_customer_id = $7,
		    gateway_response = $8,
		    idempotency_key = $9,
		    card_last_four = $10,
		    card_brand = $11,
		    processed_at = $12,
		    refund_amount = $13,
		    refund_reason = $14,
		    refunded_at = $15,
		    error_code = $16,
		    error_message = $17,
		    retry_count = $18,
		    metadata = $19,
		    updated_at = $20
		WHERE id = $1`

	metadataJSON, err := json.Marshal(payment.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	gatewayResponseJSON, err := json.Marshal(payment.GatewayResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal gateway_response: %w", err)
	}

	// Handle nullable method
	var method *string
	if payment.Method != "" {
		m := string(payment.Method)
		method = &m
	}

	result, err := r.db.Pool().Exec(ctx, query,
		payment.ID,
		string(payment.Status),
		method,
		payment.Gateway,
		nullString(payment.GatewayPaymentID),
		nullString(payment.GatewayChargeID),
		nullString(payment.GatewayCustomerID),
		gatewayResponseJSON,
		nullString(payment.IdempotencyKey),
		nullString(payment.CardLastFour),
		nullString(payment.CardBrand),
		payment.ProcessedAt,
		payment.RefundAmount,
		nullString(payment.RefundReason),
		payment.RefundedAt,
		nullString(payment.ErrorCode),
		nullString(payment.ErrorMessage),
		payment.RetryCount,
		metadataJSON,
		payment.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}

// GetByGatewayPaymentID retrieves a payment by gateway payment ID
func (r *PostgresPaymentRepository) GetByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (*domain.Payment, error) {
	query := `SELECT ` + selectColumns + ` FROM payments WHERE gateway_payment_id = $1`
	return r.scanPayment(r.db.Pool().QueryRow(ctx, query, gatewayPaymentID))
}

// GetByIdempotencyKey retrieves a payment by idempotency key
func (r *PostgresPaymentRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Payment, error) {
	query := `SELECT ` + selectColumns + ` FROM payments WHERE idempotency_key = $1`
	return r.scanPayment(r.db.Pool().QueryRow(ctx, query, idempotencyKey))
}

// scanPayment scans a single payment from a row
func (r *PostgresPaymentRepository) scanPayment(row pgx.Row) (*domain.Payment, error) {
	var payment domain.Payment
	var status string
	var method *string
	var metadataJSON, gatewayResponseJSON []byte
	var gateway, gatewayPaymentID, gatewayChargeID, gatewayCustomerID *string
	var idempotencyKey, cardLastFour, cardBrand *string
	var refundReason, errorCode, errorMessage *string

	err := row.Scan(
		&payment.ID,
		&payment.TenantID,
		&payment.BookingID,
		&payment.UserID,
		&payment.Amount,
		&payment.Currency,
		&method,
		&status,
		&gateway,
		&gatewayPaymentID,
		&gatewayChargeID,
		&gatewayCustomerID,
		&gatewayResponseJSON,
		&idempotencyKey,
		&cardLastFour,
		&cardBrand,
		&payment.InitiatedAt,
		&payment.ProcessedAt,
		&payment.RefundAmount,
		&refundReason,
		&payment.RefundedAt,
		&errorCode,
		&errorMessage,
		&payment.RetryCount,
		&metadataJSON,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to scan payment: %w", err)
	}

	payment.Status = domain.PaymentStatus(status)
	if method != nil {
		payment.Method = domain.PaymentMethod(*method)
	}

	// Handle nullable string fields
	if gateway != nil {
		payment.Gateway = *gateway
	}
	if gatewayPaymentID != nil {
		payment.GatewayPaymentID = *gatewayPaymentID
	}
	if gatewayChargeID != nil {
		payment.GatewayChargeID = *gatewayChargeID
	}
	if gatewayCustomerID != nil {
		payment.GatewayCustomerID = *gatewayCustomerID
	}
	if idempotencyKey != nil {
		payment.IdempotencyKey = *idempotencyKey
	}
	if cardLastFour != nil {
		payment.CardLastFour = *cardLastFour
	}
	if cardBrand != nil {
		payment.CardBrand = *cardBrand
	}
	if refundReason != nil {
		payment.RefundReason = *refundReason
	}
	if errorCode != nil {
		payment.ErrorCode = *errorCode
	}
	if errorMessage != nil {
		payment.ErrorMessage = *errorMessage
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &payment.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	if len(gatewayResponseJSON) > 0 {
		if err := json.Unmarshal(gatewayResponseJSON, &payment.GatewayResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal gateway_response: %w", err)
		}
	}

	return &payment, nil
}

// scanPaymentFromRows scans a single payment from rows
func (r *PostgresPaymentRepository) scanPaymentFromRows(rows pgx.Rows) (*domain.Payment, error) {
	var payment domain.Payment
	var status string
	var method *string
	var metadataJSON, gatewayResponseJSON []byte
	var gateway, gatewayPaymentID, gatewayChargeID, gatewayCustomerID *string
	var idempotencyKey, cardLastFour, cardBrand *string
	var refundReason, errorCode, errorMessage *string

	err := rows.Scan(
		&payment.ID,
		&payment.TenantID,
		&payment.BookingID,
		&payment.UserID,
		&payment.Amount,
		&payment.Currency,
		&method,
		&status,
		&gateway,
		&gatewayPaymentID,
		&gatewayChargeID,
		&gatewayCustomerID,
		&gatewayResponseJSON,
		&idempotencyKey,
		&cardLastFour,
		&cardBrand,
		&payment.InitiatedAt,
		&payment.ProcessedAt,
		&payment.RefundAmount,
		&refundReason,
		&payment.RefundedAt,
		&errorCode,
		&errorMessage,
		&payment.RetryCount,
		&metadataJSON,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan payment: %w", err)
	}

	payment.Status = domain.PaymentStatus(status)
	if method != nil {
		payment.Method = domain.PaymentMethod(*method)
	}

	// Handle nullable string fields
	if gateway != nil {
		payment.Gateway = *gateway
	}
	if gatewayPaymentID != nil {
		payment.GatewayPaymentID = *gatewayPaymentID
	}
	if gatewayChargeID != nil {
		payment.GatewayChargeID = *gatewayChargeID
	}
	if gatewayCustomerID != nil {
		payment.GatewayCustomerID = *gatewayCustomerID
	}
	if idempotencyKey != nil {
		payment.IdempotencyKey = *idempotencyKey
	}
	if cardLastFour != nil {
		payment.CardLastFour = *cardLastFour
	}
	if cardBrand != nil {
		payment.CardBrand = *cardBrand
	}
	if refundReason != nil {
		payment.RefundReason = *refundReason
	}
	if errorCode != nil {
		payment.ErrorCode = *errorCode
	}
	if errorMessage != nil {
		payment.ErrorMessage = *errorMessage
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &payment.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	if len(gatewayResponseJSON) > 0 {
		if err := json.Unmarshal(gatewayResponseJSON, &payment.GatewayResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal gateway_response: %w", err)
		}
	}

	return &payment, nil
}
