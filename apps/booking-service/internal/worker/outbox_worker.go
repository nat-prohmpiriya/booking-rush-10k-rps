package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/booking-service/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
)

// OutboxWorkerConfig contains configuration for the outbox worker
type OutboxWorkerConfig struct {
	// PollInterval is the interval between polling for pending messages
	PollInterval time.Duration
	// BatchSize is the number of messages to fetch in each poll
	BatchSize int
	// RetryInterval is the interval between retrying failed messages
	RetryInterval time.Duration
	// CleanupInterval is the interval between cleanup of old published messages
	CleanupInterval time.Duration
	// CleanupRetentionDays is the number of days to retain published messages
	CleanupRetentionDays int
}

// DefaultOutboxWorkerConfig returns default configuration
func DefaultOutboxWorkerConfig() *OutboxWorkerConfig {
	return &OutboxWorkerConfig{
		PollInterval:         100 * time.Millisecond, // Poll every 100ms for low latency
		BatchSize:            100,
		RetryInterval:        5 * time.Second,
		CleanupInterval:      1 * time.Hour,
		CleanupRetentionDays: 7,
	}
}

// OutboxWorker polls the outbox table and publishes messages to Kafka
type OutboxWorker struct {
	outboxRepo *repository.PostgresOutboxRepository
	producer   *kafka.Producer
	config     *OutboxWorkerConfig
	log        *logger.Logger
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	running    bool
}

// NewOutboxWorker creates a new outbox worker
func NewOutboxWorker(
	outboxRepo *repository.PostgresOutboxRepository,
	producer *kafka.Producer,
	config *OutboxWorkerConfig,
) *OutboxWorker {
	if config == nil {
		config = DefaultOutboxWorkerConfig()
	}

	return &OutboxWorker{
		outboxRepo: outboxRepo,
		producer:   producer,
		config:     config,
		log:        logger.Get(),
		stopCh:     make(chan struct{}),
	}
}

// Start starts the outbox worker
func (w *OutboxWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("outbox worker already running")
	}
	w.running = true
	w.mu.Unlock()

	w.log.Info("Starting outbox worker")

	// Start pending messages poller
	w.wg.Add(1)
	go w.pollPendingMessages(ctx)

	// Start failed messages retrier
	w.wg.Add(1)
	go w.retryFailedMessages(ctx)

	// Start cleanup worker
	w.wg.Add(1)
	go w.cleanupOldMessages(ctx)

	return nil
}

// Stop stops the outbox worker
func (w *OutboxWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	w.log.Info("Stopping outbox worker")
	close(w.stopCh)
	w.wg.Wait()
	w.log.Info("Outbox worker stopped")
}

// pollPendingMessages polls for pending messages and publishes them
func (w *OutboxWorker) pollPendingMessages(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processPendingMessages(ctx)
		}
	}
}

// processPendingMessages fetches and processes pending messages
func (w *OutboxWorker) processPendingMessages(ctx context.Context) {
	messages, err := w.outboxRepo.GetPendingMessages(ctx, w.config.BatchSize)
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to get pending messages: %v", err))
		return
	}

	for _, msg := range messages {
		if err := w.publishMessage(ctx, msg); err != nil {
			w.log.Error(fmt.Sprintf("Failed to publish message %s: %v", msg.ID, err))
			if markErr := w.outboxRepo.MarkAsFailed(ctx, msg.ID, err.Error()); markErr != nil {
				w.log.Error(fmt.Sprintf("Failed to mark message as failed %s: %v", msg.ID, markErr))
			}
		} else {
			if markErr := w.outboxRepo.MarkAsPublished(ctx, msg.ID); markErr != nil {
				w.log.Error(fmt.Sprintf("Failed to mark message as published %s: %v", msg.ID, markErr))
			}
		}
	}
}

// retryFailedMessages retries failed messages
func (w *OutboxWorker) retryFailedMessages(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.RetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processFailedMessages(ctx)
		}
	}
}

// processFailedMessages fetches and retries failed messages
func (w *OutboxWorker) processFailedMessages(ctx context.Context) {
	messages, err := w.outboxRepo.GetFailedMessages(ctx, w.config.BatchSize)
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to get failed messages: %v", err))
		return
	}

	for _, msg := range messages {
		if err := w.publishMessage(ctx, msg); err != nil {
			w.log.Error(fmt.Sprintf("Failed to retry message %s (attempt %d/%d): %v", msg.ID, msg.RetryCount+1, msg.MaxRetries, err))
			if markErr := w.outboxRepo.MarkAsFailed(ctx, msg.ID, err.Error()); markErr != nil {
				w.log.Error(fmt.Sprintf("Failed to mark message as failed %s: %v", msg.ID, markErr))
			}
		} else {
			w.log.Info(fmt.Sprintf("Successfully retried message %s after %d attempts", msg.ID, msg.RetryCount+1))
			if markErr := w.outboxRepo.MarkAsPublished(ctx, msg.ID); markErr != nil {
				w.log.Error(fmt.Sprintf("Failed to mark message as published %s: %v", msg.ID, markErr))
			}
		}
	}
}

// cleanupOldMessages deletes old published messages
func (w *OutboxWorker) cleanupOldMessages(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			deleted, err := w.outboxRepo.DeletePublished(ctx, w.config.CleanupRetentionDays)
			if err != nil {
				w.log.Error(fmt.Sprintf("Failed to cleanup old messages: %v", err))
			} else if deleted > 0 {
				w.log.Info(fmt.Sprintf("Cleaned up %d old published messages", deleted))
			}
		}
	}
}

// publishMessage publishes a message to Kafka
func (w *OutboxWorker) publishMessage(ctx context.Context, msg *domain.OutboxMessage) error {
	kafkaMsg := &kafka.Message{
		Topic: msg.Topic,
		Key:   []byte(msg.PartitionKey),
		Value: msg.Payload,
		Headers: map[string]string{
			"event_type":     msg.EventType,
			"aggregate_type": msg.AggregateType,
			"aggregate_id":   msg.AggregateID,
			"content_type":   "application/json",
			"source":         "outbox-worker",
		},
		Timestamp: time.Now(),
	}

	return w.producer.Produce(ctx, kafkaMsg)
}

// GetStats returns worker statistics
func (w *OutboxWorker) GetStats(ctx context.Context) (*OutboxWorkerStats, error) {
	pending, err := w.outboxRepo.GetPendingMessages(ctx, 1)
	if err != nil {
		return nil, err
	}

	failed, err := w.outboxRepo.GetFailedMessages(ctx, 1)
	if err != nil {
		return nil, err
	}

	return &OutboxWorkerStats{
		IsRunning:       w.running,
		PendingMessages: len(pending) > 0,
		FailedMessages:  len(failed) > 0,
	}, nil
}

// OutboxWorkerStats contains worker statistics
type OutboxWorkerStats struct {
	IsRunning       bool `json:"is_running"`
	PendingMessages bool `json:"pending_messages"`
	FailedMessages  bool `json:"failed_messages"`
}
