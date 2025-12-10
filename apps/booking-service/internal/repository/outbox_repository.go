package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/domain"
)

// OutboxRepository defines the interface for outbox data access
type OutboxRepository interface {
	// Create creates a new outbox message
	Create(ctx context.Context, msg *domain.OutboxMessage) error

	// CreateTx creates a new outbox message within a transaction
	CreateTx(ctx context.Context, tx pgx.Tx, msg *domain.OutboxMessage) error

	// GetPendingMessages gets pending messages to be published
	GetPendingMessages(ctx context.Context, limit int) ([]*domain.OutboxMessage, error)

	// GetFailedMessages gets failed messages that can be retried
	GetFailedMessages(ctx context.Context, limit int) ([]*domain.OutboxMessage, error)

	// MarkAsPublished marks a message as successfully published
	MarkAsPublished(ctx context.Context, id string) error

	// MarkAsFailed marks a message as failed
	MarkAsFailed(ctx context.Context, id string, err string) error

	// DeletePublished deletes old published messages for cleanup
	DeletePublished(ctx context.Context, olderThan int) (int64, error)

	// BeginTx starts a new transaction
	BeginTx(ctx context.Context) (pgx.Tx, error)
}
