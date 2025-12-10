package retry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DLQMessage represents a message in the dead letter queue
type DLQMessage struct {
	// ID is the unique identifier for the message
	ID string `json:"id"`
	// OriginalTopic is the topic the message was originally sent to
	OriginalTopic string `json:"original_topic"`
	// OriginalKey is the original message key
	OriginalKey string `json:"original_key"`
	// Payload is the original message payload
	Payload json.RawMessage `json:"payload"`
	// Headers are the original message headers
	Headers map[string]string `json:"headers,omitempty"`
	// Error is the error message that caused the failure
	Error string `json:"error"`
	// ErrorCode is an optional error code
	ErrorCode string `json:"error_code,omitempty"`
	// Attempts is the number of attempts made before moving to DLQ
	Attempts int `json:"attempts"`
	// FirstAttemptAt is when the first attempt was made
	FirstAttemptAt time.Time `json:"first_attempt_at"`
	// LastAttemptAt is when the last attempt was made
	LastAttemptAt time.Time `json:"last_attempt_at"`
	// MovedToDLQAt is when the message was moved to DLQ
	MovedToDLQAt time.Time `json:"moved_to_dlq_at"`
	// Source is the service that moved the message to DLQ
	Source string `json:"source"`
	// Metadata contains additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DLQPublisher publishes failed messages to a dead letter queue
type DLQPublisher interface {
	// PublishToDLQ publishes a message to the dead letter queue
	PublishToDLQ(ctx context.Context, msg *DLQMessage) error
	// GetDLQTopic returns the DLQ topic name for a given original topic
	GetDLQTopic(originalTopic string) string
}

// DLQConfig contains configuration for DLQ publishing
type DLQConfig struct {
	// TopicPrefix is the prefix for DLQ topics (default: "dlq.")
	TopicPrefix string
	// TopicSuffix is the suffix for DLQ topics (default: ".dlq")
	TopicSuffix string
	// UsePrefix determines if prefix or suffix is used (default: false = suffix)
	UsePrefix bool
	// Source is the service name for logging
	Source string
}

// DefaultDLQConfig returns default DLQ configuration
func DefaultDLQConfig() *DLQConfig {
	return &DLQConfig{
		TopicPrefix: "dlq.",
		TopicSuffix: ".dlq",
		UsePrefix:   false,
		Source:      "unknown",
	}
}

// KafkaPublisher interface for publishing to Kafka
type KafkaPublisher interface {
	PublishJSON(ctx context.Context, topic string, key string, data interface{}, headers map[string]string) error
}

// KafkaDLQPublisher publishes failed messages to Kafka DLQ topics
type KafkaDLQPublisher struct {
	producer KafkaPublisher
	config   *DLQConfig
}

// PublishJSON interface wrapper for kafka producer
type PublishJSON interface {
	ProduceJSON(ctx context.Context, topic string, key string, data interface{}, headers map[string]string) error
}

// KafkaProducerAdapter adapts a Kafka producer to the KafkaPublisher interface
type KafkaProducerAdapter struct {
	Producer PublishJSON
}

// PublishJSON publishes JSON data to Kafka
func (a *KafkaProducerAdapter) PublishJSON(ctx context.Context, topic string, key string, data interface{}, headers map[string]string) error {
	return a.Producer.ProduceJSON(ctx, topic, key, data, headers)
}

// NewKafkaDLQPublisher creates a new Kafka DLQ publisher
func NewKafkaDLQPublisher(producer KafkaPublisher, config *DLQConfig) *KafkaDLQPublisher {
	if config == nil {
		config = DefaultDLQConfig()
	}
	return &KafkaDLQPublisher{
		producer: producer,
		config:   config,
	}
}

// PublishToDLQ publishes a message to the dead letter queue
func (p *KafkaDLQPublisher) PublishToDLQ(ctx context.Context, msg *DLQMessage) error {
	if msg == nil {
		return fmt.Errorf("DLQ message cannot be nil")
	}

	dlqTopic := p.GetDLQTopic(msg.OriginalTopic)
	msg.MovedToDLQAt = time.Now()
	msg.Source = p.config.Source

	headers := map[string]string{
		"content_type":    "application/json",
		"original_topic":  msg.OriginalTopic,
		"error":           msg.Error,
		"attempts":        fmt.Sprintf("%d", msg.Attempts),
		"moved_to_dlq_at": msg.MovedToDLQAt.Format(time.RFC3339),
		"source":          msg.Source,
	}

	if msg.ErrorCode != "" {
		headers["error_code"] = msg.ErrorCode
	}

	// Merge original headers
	for k, v := range msg.Headers {
		if _, exists := headers[k]; !exists {
			headers["original_"+k] = v
		}
	}

	return p.producer.PublishJSON(ctx, dlqTopic, msg.OriginalKey, msg, headers)
}

// GetDLQTopic returns the DLQ topic name for a given original topic
func (p *KafkaDLQPublisher) GetDLQTopic(originalTopic string) string {
	if p.config.UsePrefix {
		return p.config.TopicPrefix + originalTopic
	}
	return originalTopic + p.config.TopicSuffix
}

// DLQHandler handles failed operations with DLQ publishing
type DLQHandler struct {
	retrier   *Retrier
	publisher DLQPublisher
	config    *DLQHandlerConfig
}

// DLQHandlerConfig contains configuration for DLQ handler
type DLQHandlerConfig struct {
	// RetryConfig is the retry configuration
	RetryConfig *Config
	// Source is the service name
	Source string
	// OnDLQ is called when a message is moved to DLQ
	OnDLQ func(msg *DLQMessage)
}

// DefaultDLQHandlerConfig returns default DLQ handler configuration
func DefaultDLQHandlerConfig() *DLQHandlerConfig {
	return &DLQHandlerConfig{
		RetryConfig: DefaultConfig(),
		Source:      "unknown",
	}
}

// NewDLQHandler creates a new DLQ handler
func NewDLQHandler(publisher DLQPublisher, config *DLQHandlerConfig) *DLQHandler {
	if config == nil {
		config = DefaultDLQHandlerConfig()
	}
	return &DLQHandler{
		retrier:   New(config.RetryConfig),
		publisher: publisher,
		config:    config,
	}
}

// MessageContext contains context for message processing
type MessageContext struct {
	// ID is the message ID
	ID string
	// Topic is the original topic
	Topic string
	// Key is the message key
	Key string
	// Payload is the message payload
	Payload json.RawMessage
	// Headers are the message headers
	Headers map[string]string
	// FirstAttemptAt is when processing started
	FirstAttemptAt time.Time
	// Metadata contains additional context
	Metadata map[string]interface{}
}

// ProcessWithDLQ processes a message with retry and DLQ support
func (h *DLQHandler) ProcessWithDLQ(ctx context.Context, msgCtx *MessageContext, op Operation) error {
	if msgCtx.FirstAttemptAt.IsZero() {
		msgCtx.FirstAttemptAt = time.Now()
	}

	var lastErr error
	result := h.retrier.DoWithCallback(ctx, op, func(attempt int, err error, nextInterval time.Duration) {
		lastErr = err
	})

	if result.Err == nil {
		return nil
	}

	// All retries failed, move to DLQ
	errMsg := result.Err.Error()
	if result.LastError != nil {
		errMsg = result.LastError.Error()
	}

	dlqMsg := &DLQMessage{
		ID:             msgCtx.ID,
		OriginalTopic:  msgCtx.Topic,
		OriginalKey:    msgCtx.Key,
		Payload:        msgCtx.Payload,
		Headers:        msgCtx.Headers,
		Error:          errMsg,
		Attempts:       result.Attempts,
		FirstAttemptAt: msgCtx.FirstAttemptAt,
		LastAttemptAt:  time.Now(),
		Source:         h.config.Source,
		Metadata:       msgCtx.Metadata,
	}

	// Invoke callback if set
	if h.config.OnDLQ != nil {
		h.config.OnDLQ(dlqMsg)
	}

	// Publish to DLQ
	if publishErr := h.publisher.PublishToDLQ(ctx, dlqMsg); publishErr != nil {
		return fmt.Errorf("failed to publish to DLQ: %w (original error: %v)", publishErr, lastErr)
	}

	return result.Err
}

// NoOpDLQPublisher is a DLQ publisher that does nothing (for testing or disabled DLQ)
type NoOpDLQPublisher struct {
	config *DLQConfig
}

// NewNoOpDLQPublisher creates a new no-op DLQ publisher
func NewNoOpDLQPublisher() *NoOpDLQPublisher {
	return &NoOpDLQPublisher{
		config: DefaultDLQConfig(),
	}
}

// PublishToDLQ does nothing
func (p *NoOpDLQPublisher) PublishToDLQ(ctx context.Context, msg *DLQMessage) error {
	return nil
}

// GetDLQTopic returns the DLQ topic name
func (p *NoOpDLQPublisher) GetDLQTopic(originalTopic string) string {
	return originalTopic + p.config.TopicSuffix
}
