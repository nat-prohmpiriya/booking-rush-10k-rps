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

// PostgresOutboxRepository implements OutboxRepository using PostgreSQL
type PostgresOutboxRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresOutboxRepository creates a new PostgresOutboxRepository
func NewPostgresOutboxRepository(pool *pgxpool.Pool) *PostgresOutboxRepository {
	return &PostgresOutboxRepository{pool: pool}
}

// Create creates a new outbox message
func (r *PostgresOutboxRepository) Create(ctx context.Context, msg *domain.OutboxMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	query := `
		INSERT INTO outbox (
			id, aggregate_type, aggregate_id, event_type,
			payload, topic, partition_key, status,
			retry_count, max_retries, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.pool.Exec(ctx, query,
		msg.ID,
		msg.AggregateType,
		msg.AggregateID,
		msg.EventType,
		msg.Payload,
		msg.Topic,
		msg.PartitionKey,
		msg.Status.String(),
		msg.RetryCount,
		msg.MaxRetries,
		msg.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create outbox message: %w", err)
	}

	return nil
}

// CreateTx creates a new outbox message within a transaction
func (r *PostgresOutboxRepository) CreateTx(ctx context.Context, tx pgx.Tx, msg *domain.OutboxMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	query := `
		INSERT INTO outbox (
			id, aggregate_type, aggregate_id, event_type,
			payload, topic, partition_key, status,
			retry_count, max_retries, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := tx.Exec(ctx, query,
		msg.ID,
		msg.AggregateType,
		msg.AggregateID,
		msg.EventType,
		msg.Payload,
		msg.Topic,
		msg.PartitionKey,
		msg.Status.String(),
		msg.RetryCount,
		msg.MaxRetries,
		msg.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create outbox message in transaction: %w", err)
	}

	return nil
}

// GetPendingMessages gets pending messages to be published
func (r *PostgresOutboxRepository) GetPendingMessages(ctx context.Context, limit int) ([]*domain.OutboxMessage, error) {
	query := `
		SELECT
			id, aggregate_type, aggregate_id, event_type,
			payload, topic, partition_key, status,
			retry_count, max_retries, last_error,
			created_at, processed_at, published_at
		FROM outbox
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}
	defer rows.Close()

	return scanOutboxMessages(rows)
}

// GetFailedMessages gets failed messages that can be retried
func (r *PostgresOutboxRepository) GetFailedMessages(ctx context.Context, limit int) ([]*domain.OutboxMessage, error) {
	query := `
		SELECT
			id, aggregate_type, aggregate_id, event_type,
			payload, topic, partition_key, status,
			retry_count, max_retries, last_error,
			created_at, processed_at, published_at
		FROM outbox
		WHERE status = 'failed' AND retry_count < max_retries
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed messages: %w", err)
	}
	defer rows.Close()

	return scanOutboxMessages(rows)
}

// MarkAsPublished marks a message as successfully published
func (r *PostgresOutboxRepository) MarkAsPublished(ctx context.Context, id string) error {
	query := `
		UPDATE outbox SET
			status = 'published',
			processed_at = $2,
			published_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to mark message as published: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("outbox message not found")
	}

	return nil
}

// MarkAsFailed marks a message as failed
func (r *PostgresOutboxRepository) MarkAsFailed(ctx context.Context, id string, errMsg string) error {
	query := `
		UPDATE outbox SET
			status = 'failed',
			last_error = $2,
			retry_count = retry_count + 1,
			processed_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, errMsg, now)
	if err != nil {
		return fmt.Errorf("failed to mark message as failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("outbox message not found")
	}

	return nil
}

// DeletePublished deletes old published messages for cleanup
func (r *PostgresOutboxRepository) DeletePublished(ctx context.Context, olderThanDays int) (int64, error) {
	query := `
		DELETE FROM outbox
		WHERE status = 'published' AND published_at < $1
	`

	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	result, err := r.pool.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete published messages: %w", err)
	}

	return result.RowsAffected(), nil
}

// BeginTx starts a new transaction
func (r *PostgresOutboxRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// scanOutboxMessages scans rows into OutboxMessage slice
func scanOutboxMessages(rows pgx.Rows) ([]*domain.OutboxMessage, error) {
	var messages []*domain.OutboxMessage

	for rows.Next() {
		msg := &domain.OutboxMessage{}
		var (
			status      string
			lastError   *string
			processedAt *time.Time
			publishedAt *time.Time
		)

		err := rows.Scan(
			&msg.ID,
			&msg.AggregateType,
			&msg.AggregateID,
			&msg.EventType,
			&msg.Payload,
			&msg.Topic,
			&msg.PartitionKey,
			&status,
			&msg.RetryCount,
			&msg.MaxRetries,
			&lastError,
			&msg.CreatedAt,
			&processedAt,
			&publishedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox message: %w", err)
		}

		msg.Status = domain.OutboxStatus(status)
		if lastError != nil {
			msg.LastError = *lastError
		}
		msg.ProcessedAt = processedAt
		msg.PublishedAt = publishedAt

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outbox messages: %w", err)
	}

	return messages, nil
}

// Ensure PostgresOutboxRepository implements OutboxRepository
var _ OutboxRepository = (*PostgresOutboxRepository)(nil)
