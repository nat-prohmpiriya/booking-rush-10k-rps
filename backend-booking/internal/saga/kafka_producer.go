package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
)

// SagaProducer interface for sending saga messages to Kafka
type SagaProducer interface {
	// Commands
	SendCommand(ctx context.Context, command *SagaCommand) error
	SendCompensationCommand(ctx context.Context, command *CompensationCommand) error

	// Events
	SendStepSuccessEvent(ctx context.Context, event *SagaEvent) error
	SendStepFailureEvent(ctx context.Context, event *SagaEvent) error

	// Lifecycle events
	SendSagaStartedEvent(ctx context.Context, event *SagaLifecycleEvent) error
	SendSagaCompletedEvent(ctx context.Context, event *SagaLifecycleEvent) error
	SendSagaFailedEvent(ctx context.Context, event *SagaLifecycleEvent) error
	SendSagaCompensatedEvent(ctx context.Context, event *SagaLifecycleEvent) error

	// Timeout
	ScheduleTimeoutCheck(ctx context.Context, check *TimeoutCheck) error

	// Generic publish (for DLQ and other topics)
	Publish(ctx context.Context, topic string, key string, value []byte) error

	// Close
	Close() error
}

// KafkaSagaProducer implements SagaProducer using Kafka
type KafkaSagaProducer struct {
	producer *kafka.Producer
	logger   Logger
}

// KafkaSagaProducerConfig holds configuration for the Kafka saga producer
type KafkaSagaProducerConfig struct {
	Brokers       []string
	ClientID      string
	MaxRetries    int
	RetryInterval time.Duration
	Logger        Logger
}

// NewKafkaSagaProducer creates a new Kafka saga producer
func NewKafkaSagaProducer(ctx context.Context, cfg *KafkaSagaProducerConfig) (*KafkaSagaProducer, error) {
	producer, err := kafka.NewProducer(ctx, &kafka.ProducerConfig{
		Brokers:       cfg.Brokers,
		ClientID:      cfg.ClientID,
		MaxRetries:    cfg.MaxRetries,
		RetryInterval: cfg.RetryInterval,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = &NoOpLogger{}
	}

	return &KafkaSagaProducer{
		producer: producer,
		logger:   logger,
	}, nil
}

// SendCommand sends a saga command to Kafka
func (p *KafkaSagaProducer) SendCommand(ctx context.Context, command *SagaCommand) error {
	topic := StepToCommandTopic(command.StepName)
	if topic == "" {
		return fmt.Errorf("unknown step name: %s", command.StepName)
	}

	headers := map[string]string{
		"saga_id":        command.SagaID,
		"saga_name":      command.SagaName,
		"step_name":      command.StepName,
		"message_type":   string(MessageTypeCommand),
		"idempotency_key": command.IdempotencyKey,
	}

	if err := p.producer.ProduceJSON(ctx, topic, command.SagaID, command, headers); err != nil {
		p.logger.Error("Failed to send saga command",
			"saga_id", command.SagaID,
			"step_name", command.StepName,
			"topic", topic,
			"error", err)
		return fmt.Errorf("failed to send saga command: %w", err)
	}

	p.logger.Info("Saga command sent",
		"saga_id", command.SagaID,
		"step_name", command.StepName,
		"topic", topic)

	return nil
}

// SendCompensationCommand sends a compensation command to Kafka
func (p *KafkaSagaProducer) SendCompensationCommand(ctx context.Context, command *CompensationCommand) error {
	topic := StepToCompensationTopic(command.StepName)
	if topic == "" {
		return fmt.Errorf("no compensation topic for step: %s", command.StepName)
	}

	headers := map[string]string{
		"saga_id":      command.SagaID,
		"saga_name":    command.SagaName,
		"step_name":    command.StepName,
		"message_type": string(MessageTypeCommand),
		"compensation": "true",
		"reason":       command.Reason,
	}

	if err := p.producer.ProduceJSON(ctx, topic, command.SagaID, command, headers); err != nil {
		p.logger.Error("Failed to send compensation command",
			"saga_id", command.SagaID,
			"step_name", command.StepName,
			"topic", topic,
			"error", err)
		return fmt.Errorf("failed to send compensation command: %w", err)
	}

	p.logger.Info("Compensation command sent",
		"saga_id", command.SagaID,
		"step_name", command.StepName,
		"topic", topic,
		"reason", command.Reason)

	return nil
}

// SendStepSuccessEvent sends a step success event to Kafka
func (p *KafkaSagaProducer) SendStepSuccessEvent(ctx context.Context, event *SagaEvent) error {
	topic := StepToSuccessEventTopic(event.StepName)
	if topic == "" {
		return fmt.Errorf("unknown step name: %s", event.StepName)
	}

	return p.sendEvent(ctx, topic, event)
}

// SendStepFailureEvent sends a step failure event to Kafka
func (p *KafkaSagaProducer) SendStepFailureEvent(ctx context.Context, event *SagaEvent) error {
	topic := StepToFailureEventTopic(event.StepName)
	if topic == "" {
		return fmt.Errorf("unknown step name: %s", event.StepName)
	}

	return p.sendEvent(ctx, topic, event)
}

func (p *KafkaSagaProducer) sendEvent(ctx context.Context, topic string, event *SagaEvent) error {
	headers := map[string]string{
		"saga_id":      event.SagaID,
		"saga_name":    event.SagaName,
		"step_name":    event.StepName,
		"message_type": string(MessageTypeEvent),
		"success":      fmt.Sprintf("%t", event.Success),
	}

	if !event.Success && event.ErrorCode != "" {
		headers["error_code"] = event.ErrorCode
	}

	if err := p.producer.ProduceJSON(ctx, topic, event.SagaID, event, headers); err != nil {
		p.logger.Error("Failed to send saga event",
			"saga_id", event.SagaID,
			"step_name", event.StepName,
			"topic", topic,
			"success", event.Success,
			"error", err)
		return fmt.Errorf("failed to send saga event: %w", err)
	}

	p.logger.Info("Saga event sent",
		"saga_id", event.SagaID,
		"step_name", event.StepName,
		"topic", topic,
		"success", event.Success)

	return nil
}

// SendSagaStartedEvent sends a saga started lifecycle event
func (p *KafkaSagaProducer) SendSagaStartedEvent(ctx context.Context, event *SagaLifecycleEvent) error {
	return p.sendLifecycleEvent(ctx, TopicSagaStartedEvent, event)
}

// SendSagaCompletedEvent sends a saga completed lifecycle event
func (p *KafkaSagaProducer) SendSagaCompletedEvent(ctx context.Context, event *SagaLifecycleEvent) error {
	return p.sendLifecycleEvent(ctx, TopicSagaCompletedEvent, event)
}

// SendSagaFailedEvent sends a saga failed lifecycle event
func (p *KafkaSagaProducer) SendSagaFailedEvent(ctx context.Context, event *SagaLifecycleEvent) error {
	return p.sendLifecycleEvent(ctx, TopicSagaFailedEvent, event)
}

// SendSagaCompensatedEvent sends a saga compensated lifecycle event
func (p *KafkaSagaProducer) SendSagaCompensatedEvent(ctx context.Context, event *SagaLifecycleEvent) error {
	return p.sendLifecycleEvent(ctx, TopicSagaCompensatedEvent, event)
}

func (p *KafkaSagaProducer) sendLifecycleEvent(ctx context.Context, topic string, event *SagaLifecycleEvent) error {
	headers := map[string]string{
		"saga_id":   event.SagaID,
		"saga_name": event.SagaName,
		"status":    event.Status,
	}

	if err := p.producer.ProduceJSON(ctx, topic, event.SagaID, event, headers); err != nil {
		p.logger.Error("Failed to send saga lifecycle event",
			"saga_id", event.SagaID,
			"status", event.Status,
			"topic", topic,
			"error", err)
		return fmt.Errorf("failed to send saga lifecycle event: %w", err)
	}

	p.logger.Info("Saga lifecycle event sent",
		"saga_id", event.SagaID,
		"status", event.Status,
		"topic", topic)

	return nil
}

// ScheduleTimeoutCheck schedules a timeout check (could use delayed message queue or separate scheduler)
func (p *KafkaSagaProducer) ScheduleTimeoutCheck(ctx context.Context, check *TimeoutCheck) error {
	// For now, we'll send to a dedicated timeout topic
	// In production, you might use a delayed message queue or scheduler
	topic := "saga.booking.timeout-check"

	headers := map[string]string{
		"saga_id":    check.SagaID,
		"saga_name":  check.SagaName,
		"step_name":  check.StepName,
		"timeout_at": check.TimeoutAt.Format(time.RFC3339),
	}

	if err := p.producer.ProduceJSON(ctx, topic, check.SagaID, check, headers); err != nil {
		p.logger.Error("Failed to schedule timeout check",
			"saga_id", check.SagaID,
			"step_name", check.StepName,
			"error", err)
		return fmt.Errorf("failed to schedule timeout check: %w", err)
	}

	p.logger.Info("Timeout check scheduled",
		"saga_id", check.SagaID,
		"step_name", check.StepName,
		"timeout_at", check.TimeoutAt)

	return nil
}

// Publish publishes raw bytes to a topic (used for DLQ and other generic publishing)
func (p *KafkaSagaProducer) Publish(ctx context.Context, topic string, key string, value []byte) error {
	msg := &kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}
	if err := p.producer.Produce(ctx, msg); err != nil {
		p.logger.Error("Failed to publish message",
			"topic", topic,
			"key", key,
			"error", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

// Close closes the Kafka producer
func (p *KafkaSagaProducer) Close() error {
	p.producer.Close()
	return nil
}

// MockSagaProducer is a mock implementation for testing
type MockSagaProducer struct {
	Commands             []*SagaCommand
	CompensationCommands []*CompensationCommand
	SuccessEvents        []*SagaEvent
	FailureEvents        []*SagaEvent
	LifecycleEvents      []*SagaLifecycleEvent
	TimeoutChecks        []*TimeoutCheck
	PublishedMessages    []PublishedMessage
	ShouldFail           bool
	FailureError         error
}

// PublishedMessage represents a message published via Publish method
type PublishedMessage struct {
	Topic string
	Key   string
	Value []byte
}

// NewMockSagaProducer creates a new mock saga producer
func NewMockSagaProducer() *MockSagaProducer {
	return &MockSagaProducer{
		Commands:             make([]*SagaCommand, 0),
		CompensationCommands: make([]*CompensationCommand, 0),
		SuccessEvents:        make([]*SagaEvent, 0),
		FailureEvents:        make([]*SagaEvent, 0),
		LifecycleEvents:      make([]*SagaLifecycleEvent, 0),
		TimeoutChecks:        make([]*TimeoutCheck, 0),
		PublishedMessages:    make([]PublishedMessage, 0),
	}
}

func (m *MockSagaProducer) SendCommand(ctx context.Context, command *SagaCommand) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.Commands = append(m.Commands, command)
	return nil
}

func (m *MockSagaProducer) SendCompensationCommand(ctx context.Context, command *CompensationCommand) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.CompensationCommands = append(m.CompensationCommands, command)
	return nil
}

func (m *MockSagaProducer) SendStepSuccessEvent(ctx context.Context, event *SagaEvent) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.SuccessEvents = append(m.SuccessEvents, event)
	return nil
}

func (m *MockSagaProducer) SendStepFailureEvent(ctx context.Context, event *SagaEvent) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.FailureEvents = append(m.FailureEvents, event)
	return nil
}

func (m *MockSagaProducer) SendSagaStartedEvent(ctx context.Context, event *SagaLifecycleEvent) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.LifecycleEvents = append(m.LifecycleEvents, event)
	return nil
}

func (m *MockSagaProducer) SendSagaCompletedEvent(ctx context.Context, event *SagaLifecycleEvent) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.LifecycleEvents = append(m.LifecycleEvents, event)
	return nil
}

func (m *MockSagaProducer) SendSagaFailedEvent(ctx context.Context, event *SagaLifecycleEvent) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.LifecycleEvents = append(m.LifecycleEvents, event)
	return nil
}

func (m *MockSagaProducer) SendSagaCompensatedEvent(ctx context.Context, event *SagaLifecycleEvent) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.LifecycleEvents = append(m.LifecycleEvents, event)
	return nil
}

func (m *MockSagaProducer) ScheduleTimeoutCheck(ctx context.Context, check *TimeoutCheck) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.TimeoutChecks = append(m.TimeoutChecks, check)
	return nil
}

func (m *MockSagaProducer) Publish(ctx context.Context, topic string, key string, value []byte) error {
	if m.ShouldFail {
		if m.FailureError != nil {
			return m.FailureError
		}
		return fmt.Errorf("mock producer failure")
	}
	m.PublishedMessages = append(m.PublishedMessages, PublishedMessage{
		Topic: topic,
		Key:   key,
		Value: value,
	})
	return nil
}

func (m *MockSagaProducer) Close() error {
	return nil
}

// Clear clears all recorded messages
func (m *MockSagaProducer) Clear() {
	m.Commands = make([]*SagaCommand, 0)
	m.CompensationCommands = make([]*CompensationCommand, 0)
	m.SuccessEvents = make([]*SagaEvent, 0)
	m.FailureEvents = make([]*SagaEvent, 0)
	m.LifecycleEvents = make([]*SagaLifecycleEvent, 0)
	m.TimeoutChecks = make([]*TimeoutCheck, 0)
	m.PublishedMessages = make([]PublishedMessage, 0)
}

// GetLifecycleEventsByStatus returns lifecycle events filtered by status
func (m *MockSagaProducer) GetLifecycleEventsByStatus(status string) []*SagaLifecycleEvent {
	var events []*SagaLifecycleEvent
	for _, e := range m.LifecycleEvents {
		if e.Status == status {
			events = append(events, e)
		}
	}
	return events
}

// Ensure MockSagaProducer implements SagaProducer
var _ SagaProducer = (*MockSagaProducer)(nil)

// ParseSagaCommand parses a Kafka message into a SagaCommand
func ParseSagaCommand(data []byte) (*SagaCommand, error) {
	var command SagaCommand
	if err := json.Unmarshal(data, &command); err != nil {
		return nil, fmt.Errorf("failed to parse saga command: %w", err)
	}
	return &command, nil
}

// ParseSagaEvent parses a Kafka message into a SagaEvent
func ParseSagaEvent(data []byte) (*SagaEvent, error) {
	var event SagaEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse saga event: %w", err)
	}
	return &event, nil
}

// ParseCompensationCommand parses a Kafka message into a CompensationCommand
func ParseCompensationCommand(data []byte) (*CompensationCommand, error) {
	var command CompensationCommand
	if err := json.Unmarshal(data, &command); err != nil {
		return nil, fmt.Errorf("failed to parse compensation command: %w", err)
	}
	return &command, nil
}
