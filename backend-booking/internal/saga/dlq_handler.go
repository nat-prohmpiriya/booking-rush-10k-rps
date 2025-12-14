package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

const (
	// DLQTopic is the dead letter queue topic name
	DLQTopic = "saga.booking.dlq"

	// MaxRetryAttempts is the maximum number of retry attempts before sending to DLQ
	MaxRetryAttempts = 3
)

// DLQMessage represents a message in the dead letter queue
type DLQMessage struct {
	ID             string                 `json:"id"`
	OriginalTopic  string                 `json:"original_topic"`
	SagaID         string                 `json:"saga_id,omitempty"`
	MessageKey     string                 `json:"message_key,omitempty"`
	MessageValue   map[string]interface{} `json:"message_value"`
	ErrorMessage   string                 `json:"error_message"`
	ErrorCode      string                 `json:"error_code,omitempty"`
	RetryCount     int                    `json:"retry_count"`
	FirstFailedAt  time.Time              `json:"first_failed_at"`
	LastFailedAt   time.Time              `json:"last_failed_at"`
	Headers        map[string]string      `json:"headers,omitempty"`
}

// DLQHandler handles dead letter queue operations
type DLQHandler struct {
	producer SagaProducer
	store    *pkgsaga.PostgresStore // For DB-based DLQ storage
	logger   Logger
}

// NewDLQHandler creates a new DLQ handler
func NewDLQHandler(producer SagaProducer, store *pkgsaga.PostgresStore, logger Logger) *DLQHandler {
	if logger == nil {
		logger = &NoOpLogger{}
	}
	return &DLQHandler{
		producer: producer,
		store:    store,
		logger:   logger,
	}
}

// HandleFailedMessage sends a failed message to the dead letter queue
func (h *DLQHandler) HandleFailedMessage(ctx context.Context, originalTopic string, messageKey string, messageValue []byte, err error, retryCount int) error {
	h.logger.Error(fmt.Sprintf("[ALERT] Message failed after %d retries, sending to DLQ", retryCount),
		"topic", originalTopic,
		"message_key", messageKey,
		"error", err.Error())

	// Parse message value
	var parsedValue map[string]interface{}
	if jsonErr := json.Unmarshal(messageValue, &parsedValue); jsonErr != nil {
		parsedValue = map[string]interface{}{
			"raw_value": string(messageValue),
		}
	}

	// Extract saga_id if present
	sagaID := ""
	if id, ok := parsedValue["saga_id"].(string); ok {
		sagaID = id
	}

	dlqMsg := &DLQMessage{
		ID:            fmt.Sprintf("%d", time.Now().UnixNano()),
		OriginalTopic: originalTopic,
		SagaID:        sagaID,
		MessageKey:    messageKey,
		MessageValue:  parsedValue,
		ErrorMessage:  err.Error(),
		RetryCount:    retryCount,
		FirstFailedAt: time.Now(),
		LastFailedAt:  time.Now(),
	}

	// Store in PostgreSQL DLQ table if store is available
	if h.store != nil {
		deadLetter := &pkgsaga.DeadLetter{
			SagaID:       sagaID,
			Topic:        originalTopic,
			MessageKey:   messageKey,
			MessageValue: parsedValue,
			ErrorMessage: err.Error(),
			RetryCount:   retryCount,
		}
		if storeErr := h.store.SaveDeadLetter(ctx, deadLetter); storeErr != nil {
			h.logger.Error("Failed to save dead letter to database",
				"error", storeErr.Error())
		}
	}

	// Also publish to DLQ Kafka topic for alerting/monitoring
	if h.producer != nil {
		dlqMsgBytes, marshalErr := json.Marshal(dlqMsg)
		if marshalErr != nil {
			h.logger.Error("Failed to marshal DLQ message",
				"error", marshalErr.Error())
			return marshalErr
		}

		if publishErr := h.producer.Publish(ctx, DLQTopic, messageKey, dlqMsgBytes); publishErr != nil {
			h.logger.Error("Failed to publish to DLQ topic",
				"error", publishErr.Error())
			return publishErr
		}
	}

	h.logger.Info("Message sent to DLQ",
		"saga_id", sagaID,
		"original_topic", originalTopic)

	return nil
}

// ShouldRetry determines if a message should be retried
func (h *DLQHandler) ShouldRetry(retryCount int, err error) bool {
	// Check if we've exceeded max retries
	if retryCount >= MaxRetryAttempts {
		return false
	}

	// Check for non-retryable errors
	if isNonRetryableError(err) {
		return false
	}

	return true
}

// isNonRetryableError checks if an error should not be retried
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// List of error patterns that should not be retried
	nonRetryablePatterns := []string{
		"invalid request",
		"validation failed",
		"not found",
		"unauthorized",
		"forbidden",
		"duplicate",
		"already exists",
		"schema",
		"json",
		"unmarshal",
	}

	for _, pattern := range nonRetryablePatterns {
		if containsIgnoreCase(errMsg, pattern) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, substr, i) {
			return true
		}
	}
	return false
}

// equalFoldAt checks if s[start:start+len(substr)] equals substr (case-insensitive)
func equalFoldAt(s, substr string, start int) bool {
	if start+len(substr) > len(s) {
		return false
	}
	for i := 0; i < len(substr); i++ {
		c1 := s[start+i]
		c2 := substr[i]
		if c1 != c2 {
			// Check case-insensitive match for ASCII letters
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 'a' - 'A'
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 'a' - 'A'
			}
			if c1 != c2 {
				return false
			}
		}
	}
	return true
}

// RetryMessage attempts to retry a message from the DLQ
func (h *DLQHandler) RetryMessage(ctx context.Context, dlqMsg *DLQMessage) error {
	h.logger.Info("Retrying message from DLQ",
		"saga_id", dlqMsg.SagaID,
		"original_topic", dlqMsg.OriginalTopic)

	// Re-serialize the message value
	msgBytes, err := json.Marshal(dlqMsg.MessageValue)
	if err != nil {
		return fmt.Errorf("failed to marshal message value: %w", err)
	}

	// Republish to the original topic
	if h.producer != nil {
		if err := h.producer.Publish(ctx, dlqMsg.OriginalTopic, dlqMsg.MessageKey, msgBytes); err != nil {
			return fmt.Errorf("failed to republish message: %w", err)
		}
	}

	return nil
}

// GetStats returns DLQ statistics
type DLQStats struct {
	TotalMessages      int64     `json:"total_messages"`
	UnprocessedCount   int64     `json:"unprocessed_count"`
	ProcessedCount     int64     `json:"processed_count"`
	OldestMessageTime  time.Time `json:"oldest_message_time,omitempty"`
	ByTopic            map[string]int64 `json:"by_topic"`
}

// GetDLQStats returns statistics about the dead letter queue
func (h *DLQHandler) GetDLQStats(ctx context.Context) (*DLQStats, error) {
	if h.store == nil {
		return nil, fmt.Errorf("store not configured")
	}

	// Get unprocessed dead letters
	deadLetters, err := h.store.GetUnprocessedDeadLetters(ctx, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get dead letters: %w", err)
	}

	stats := &DLQStats{
		UnprocessedCount: int64(len(deadLetters)),
		ByTopic:          make(map[string]int64),
	}

	for _, dl := range deadLetters {
		stats.ByTopic[dl.Topic]++
		if stats.OldestMessageTime.IsZero() || dl.CreatedAt.Before(stats.OldestMessageTime) {
			stats.OldestMessageTime = dl.CreatedAt
		}
	}

	return stats, nil
}
