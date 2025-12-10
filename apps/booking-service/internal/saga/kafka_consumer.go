package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

// SagaEventHandler defines the interface for handling saga events
type SagaEventHandler interface {
	HandleStepSuccess(ctx context.Context, event *SagaEvent) error
	HandleStepFailure(ctx context.Context, event *SagaEvent) error
	HandleTimeout(ctx context.Context, check *TimeoutCheck) error
}

// SagaConsumer consumes saga events from Kafka and advances the saga
type SagaConsumer struct {
	consumer     *kafka.Consumer
	orchestrator *pkgsaga.Orchestrator
	store        pkgsaga.Store
	producer     SagaProducer
	logger       Logger
	handler      SagaEventHandler
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
}

// SagaConsumerConfig holds configuration for the saga consumer
type SagaConsumerConfig struct {
	Brokers          []string
	GroupID          string
	Topics           []string
	ClientID         string
	Orchestrator     *pkgsaga.Orchestrator
	Store            pkgsaga.Store
	Producer         SagaProducer
	Logger           Logger
	Handler          SagaEventHandler
	SessionTimeout   time.Duration
	RebalanceTimeout time.Duration
}

// NewSagaConsumer creates a new saga consumer
func NewSagaConsumer(ctx context.Context, cfg *SagaConsumerConfig) (*SagaConsumer, error) {
	topics := cfg.Topics
	if len(topics) == 0 {
		// Subscribe to all saga event topics by default
		topics = GetAllEventTopics()
		topics = append(topics, "saga.booking.timeout-check")
	}

	consumer, err := kafka.NewConsumer(ctx, &kafka.ConsumerConfig{
		Brokers:          cfg.Brokers,
		GroupID:          cfg.GroupID,
		Topics:           topics,
		ClientID:         cfg.ClientID,
		SessionTimeout:   cfg.SessionTimeout,
		RebalanceTimeout: cfg.RebalanceTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = &NoOpLogger{}
	}

	return &SagaConsumer{
		consumer:     consumer,
		orchestrator: cfg.Orchestrator,
		store:        cfg.Store,
		producer:     cfg.Producer,
		logger:       logger,
		handler:      cfg.Handler,
		stopCh:       make(chan struct{}),
	}, nil
}

// Start starts consuming saga events
func (c *SagaConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.running = true
	c.mu.Unlock()

	c.wg.Add(1)
	go c.consumeLoop(ctx)

	c.logger.Info("Saga consumer started")
	return nil
}

// Stop stops the consumer
func (c *SagaConsumer) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	c.mu.Unlock()

	close(c.stopCh)
	c.wg.Wait()
	c.consumer.Close()

	c.logger.Info("Saga consumer stopped")
	return nil
}

func (c *SagaConsumer) consumeLoop(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		records, err := c.consumer.Poll(ctx)
		if err != nil {
			c.logger.Error("Failed to poll records", "error", err)
			continue
		}

		for _, record := range records {
			if err := c.handleRecord(ctx, record); err != nil {
				c.logger.Error("Failed to handle record",
					"topic", record.Topic,
					"error", err)
			}
		}

		if len(records) > 0 {
			if err := c.consumer.CommitRecords(ctx, records); err != nil {
				c.logger.Error("Failed to commit records", "error", err)
			}
		}
	}
}

func (c *SagaConsumer) handleRecord(ctx context.Context, record *kafka.Record) error {
	topic := record.Topic

	switch topic {
	case TopicSagaSeatsReservedEvent,
		TopicSagaPaymentProcessedEvent,
		TopicSagaBookingConfirmedEvent,
		TopicSagaNotificationSentEvent:
		return c.handleSuccessEvent(ctx, record)

	case TopicSagaSeatsReservationFailedEvent,
		TopicSagaPaymentFailedEvent,
		TopicSagaBookingConfirmationFailedEvent,
		TopicSagaNotificationFailedEvent:
		return c.handleFailureEvent(ctx, record)

	case "saga.booking.timeout-check":
		return c.handleTimeoutCheck(ctx, record)

	default:
		c.logger.Warn("Unknown topic", "topic", topic)
		return nil
	}
}

func (c *SagaConsumer) handleSuccessEvent(ctx context.Context, record *kafka.Record) error {
	event, err := ParseSagaEvent(record.Value)
	if err != nil {
		return fmt.Errorf("failed to parse success event: %w", err)
	}

	c.logger.Info("Handling success event",
		"saga_id", event.SagaID,
		"step_name", event.StepName)

	// Use custom handler if provided
	if c.handler != nil {
		return c.handler.HandleStepSuccess(ctx, event)
	}

	// Default handling: advance saga to next step
	return c.advanceSaga(ctx, event)
}

func (c *SagaConsumer) handleFailureEvent(ctx context.Context, record *kafka.Record) error {
	event, err := ParseSagaEvent(record.Value)
	if err != nil {
		return fmt.Errorf("failed to parse failure event: %w", err)
	}

	c.logger.Info("Handling failure event",
		"saga_id", event.SagaID,
		"step_name", event.StepName,
		"error", event.ErrorMessage)

	// Use custom handler if provided
	if c.handler != nil {
		return c.handler.HandleStepFailure(ctx, event)
	}

	// Default handling: trigger compensation
	return c.triggerCompensation(ctx, event)
}

func (c *SagaConsumer) handleTimeoutCheck(ctx context.Context, record *kafka.Record) error {
	var check TimeoutCheck
	if err := json.Unmarshal(record.Value, &check); err != nil {
		return fmt.Errorf("failed to parse timeout check: %w", err)
	}

	// Use custom handler if provided
	if c.handler != nil {
		return c.handler.HandleTimeout(ctx, &check)
	}

	// Check if timeout has occurred
	if !check.IsTimedOut() {
		// Re-schedule for later
		check.CheckCount++
		if check.CheckCount < check.MaxChecks {
			return c.producer.ScheduleTimeoutCheck(ctx, &check)
		}
	}

	c.logger.Warn("Step timeout detected",
		"saga_id", check.SagaID,
		"step_name", check.StepName)

	// Get saga instance and check status
	instance, err := c.store.Get(ctx, check.SagaID)
	if err != nil {
		return fmt.Errorf("failed to get saga instance: %w", err)
	}

	// If saga is still running, trigger timeout handling
	if instance.Status == pkgsaga.StatusRunning {
		return c.handleStepTimeout(ctx, &check, instance)
	}

	return nil
}

func (c *SagaConsumer) advanceSaga(ctx context.Context, event *SagaEvent) error {
	instance, err := c.store.Get(ctx, event.SagaID)
	if err != nil {
		return fmt.Errorf("failed to get saga instance: %w", err)
	}

	// Update instance with step result data
	if event.Data != nil {
		instance.UpdateData(event.Data)
	}

	// Check if this was the last step
	if c.orchestrator != nil {
		def, err := c.orchestrator.GetDefinition(instance.DefinitionID)
		if err == nil {
			nextStepIndex := event.StepIndex + 1
			if nextStepIndex < len(def.Steps) {
				// Send command for next step
				nextStep := def.Steps[nextStepIndex]
				command := NewSagaCommand(
					instance.ID,
					instance.DefinitionID,
					nextStep.Name,
					nextStepIndex,
					instance.GetData(),
					nextStep.Timeout,
					nextStep.Retries,
				)
				if err := c.producer.SendCommand(ctx, command); err != nil {
					return fmt.Errorf("failed to send next step command: %w", err)
				}

				// Schedule timeout check
				timeoutCheck := NewTimeoutCheck(
					instance.ID,
					instance.DefinitionID,
					nextStep.Name,
					nextStepIndex,
					time.Now().Add(nextStep.Timeout),
					3,
				)
				if err := c.producer.ScheduleTimeoutCheck(ctx, timeoutCheck); err != nil {
					c.logger.Warn("Failed to schedule timeout check", "error", err)
				}
			} else {
				// Saga completed
				instance.Complete()
				if err := c.store.Update(ctx, instance); err != nil {
					return fmt.Errorf("failed to update completed saga: %w", err)
				}

				// Send completed lifecycle event
				completedEvent := NewSagaCompletedEvent(
					instance.ID,
					instance.DefinitionID,
					instance.GetData(),
					instance.CreatedAt,
				)
				if err := c.producer.SendSagaCompletedEvent(ctx, completedEvent); err != nil {
					c.logger.Warn("Failed to send saga completed event", "error", err)
				}
			}
		}
	}

	return c.store.Update(ctx, instance)
}

func (c *SagaConsumer) triggerCompensation(ctx context.Context, event *SagaEvent) error {
	instance, err := c.store.Get(ctx, event.SagaID)
	if err != nil {
		return fmt.Errorf("failed to get saga instance: %w", err)
	}

	instance.SetStatus(pkgsaga.StatusCompensating)
	instance.SetError(fmt.Errorf("%s", event.ErrorMessage))

	if err := c.store.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update saga status: %w", err)
	}

	// Send failed lifecycle event
	failedEvent := NewSagaFailedEvent(
		instance.ID,
		instance.DefinitionID,
		event.ErrorMessage,
		instance.CreatedAt,
	)
	if err := c.producer.SendSagaFailedEvent(ctx, failedEvent); err != nil {
		c.logger.Warn("Failed to send saga failed event", "error", err)
	}

	// Send compensation commands for completed steps in reverse order
	for i := event.StepIndex - 1; i >= 0; i-- {
		stepName := c.getStepNameByIndex(instance.DefinitionID, i)
		if stepName == "" {
			continue
		}

		// Only compensate steps that have compensation defined
		compensationTopic := StepToCompensationTopic(stepName)
		if compensationTopic == "" {
			continue
		}

		command := NewCompensationCommand(
			instance.ID,
			instance.DefinitionID,
			stepName,
			i,
			instance.GetData(),
			event.ErrorMessage,
		)

		if err := c.producer.SendCompensationCommand(ctx, command); err != nil {
			c.logger.Error("Failed to send compensation command",
				"saga_id", instance.ID,
				"step_name", stepName,
				"error", err)
		}
	}

	return nil
}

func (c *SagaConsumer) handleStepTimeout(ctx context.Context, check *TimeoutCheck, instance *pkgsaga.Instance) error {
	c.logger.Error("Step timed out, triggering compensation",
		"saga_id", check.SagaID,
		"step_name", check.StepName)

	// Create a failure event for the timeout
	event := NewSagaFailureEvent(
		check.SagaID,
		check.SagaName,
		check.StepName,
		check.StepIndex,
		"step timed out",
		"TIMEOUT",
		check.TimeoutAt.Add(-instance.UpdatedAt.Sub(instance.CreatedAt)),
		time.Now(),
	)

	return c.triggerCompensation(ctx, event)
}

func (c *SagaConsumer) getStepNameByIndex(definitionID string, index int) string {
	if c.orchestrator == nil {
		return ""
	}

	def, err := c.orchestrator.GetDefinition(definitionID)
	if err != nil {
		return ""
	}

	if index < 0 || index >= len(def.Steps) {
		return ""
	}

	return def.Steps[index].Name
}

// MockSagaConsumer is a mock implementation for testing
type MockSagaConsumer struct {
	mu              sync.RWMutex
	successEvents   []*SagaEvent
	failureEvents   []*SagaEvent
	timeoutChecks   []*TimeoutCheck
	handler         SagaEventHandler
	running         bool
}

// NewMockSagaConsumer creates a new mock saga consumer
func NewMockSagaConsumer() *MockSagaConsumer {
	return &MockSagaConsumer{
		successEvents: make([]*SagaEvent, 0),
		failureEvents: make([]*SagaEvent, 0),
		timeoutChecks: make([]*TimeoutCheck, 0),
	}
}

// Start mock starts the consumer (no-op)
func (m *MockSagaConsumer) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = true
	return nil
}

// Stop mock stops the consumer (no-op)
func (m *MockSagaConsumer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	return nil
}

// SimulateSuccessEvent simulates receiving a success event
func (m *MockSagaConsumer) SimulateSuccessEvent(ctx context.Context, event *SagaEvent) error {
	m.mu.Lock()
	m.successEvents = append(m.successEvents, event)
	m.mu.Unlock()

	if m.handler != nil {
		return m.handler.HandleStepSuccess(ctx, event)
	}
	return nil
}

// SimulateFailureEvent simulates receiving a failure event
func (m *MockSagaConsumer) SimulateFailureEvent(ctx context.Context, event *SagaEvent) error {
	m.mu.Lock()
	m.failureEvents = append(m.failureEvents, event)
	m.mu.Unlock()

	if m.handler != nil {
		return m.handler.HandleStepFailure(ctx, event)
	}
	return nil
}

// SimulateTimeoutCheck simulates receiving a timeout check
func (m *MockSagaConsumer) SimulateTimeoutCheck(ctx context.Context, check *TimeoutCheck) error {
	m.mu.Lock()
	m.timeoutChecks = append(m.timeoutChecks, check)
	m.mu.Unlock()

	if m.handler != nil {
		return m.handler.HandleTimeout(ctx, check)
	}
	return nil
}

// SetHandler sets the event handler
func (m *MockSagaConsumer) SetHandler(handler SagaEventHandler) {
	m.handler = handler
}

// GetSuccessEvents returns received success events
func (m *MockSagaConsumer) GetSuccessEvents() []*SagaEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.successEvents
}

// GetFailureEvents returns received failure events
func (m *MockSagaConsumer) GetFailureEvents() []*SagaEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failureEvents
}

// GetTimeoutChecks returns received timeout checks
func (m *MockSagaConsumer) GetTimeoutChecks() []*TimeoutCheck {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.timeoutChecks
}

// Clear clears all recorded events
func (m *MockSagaConsumer) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successEvents = make([]*SagaEvent, 0)
	m.failureEvents = make([]*SagaEvent, 0)
	m.timeoutChecks = make([]*TimeoutCheck, 0)
}
