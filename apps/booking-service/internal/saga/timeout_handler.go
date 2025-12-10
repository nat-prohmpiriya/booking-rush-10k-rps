package saga

import (
	"context"
	"fmt"
	"sync"
	"time"

	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

// TimeoutHandler manages step timeouts for sagas
type TimeoutHandler struct {
	store         pkgsaga.Store
	producer      SagaProducer
	orchestrator  *pkgsaga.Orchestrator
	logger        Logger
	checkInterval time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	running       bool

	// Track pending timeouts in memory
	pendingTimeouts map[string]*TimeoutCheck
	timeoutMu       sync.RWMutex
}

// TimeoutHandlerConfig holds configuration for the timeout handler
type TimeoutHandlerConfig struct {
	Store         pkgsaga.Store
	Producer      SagaProducer
	Orchestrator  *pkgsaga.Orchestrator
	Logger        Logger
	CheckInterval time.Duration
}

// NewTimeoutHandler creates a new timeout handler
func NewTimeoutHandler(cfg *TimeoutHandlerConfig) *TimeoutHandler {
	checkInterval := cfg.CheckInterval
	if checkInterval == 0 {
		checkInterval = 5 * time.Second
	}

	logger := cfg.Logger
	if logger == nil {
		logger = &NoOpLogger{}
	}

	return &TimeoutHandler{
		store:           cfg.Store,
		producer:        cfg.Producer,
		orchestrator:    cfg.Orchestrator,
		logger:          logger,
		checkInterval:   checkInterval,
		stopCh:          make(chan struct{}),
		pendingTimeouts: make(map[string]*TimeoutCheck),
	}
}

// Start starts the timeout handler
func (h *TimeoutHandler) Start(ctx context.Context) error {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return fmt.Errorf("timeout handler is already running")
	}
	h.running = true
	h.mu.Unlock()

	h.wg.Add(1)
	go h.runLoop(ctx)

	h.logger.Info("Timeout handler started", "check_interval", h.checkInterval)
	return nil
}

// Stop stops the timeout handler
func (h *TimeoutHandler) Stop() error {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return nil
	}
	h.running = false
	h.mu.Unlock()

	close(h.stopCh)
	h.wg.Wait()

	h.logger.Info("Timeout handler stopped")
	return nil
}

// RegisterTimeout registers a timeout check for a step
func (h *TimeoutHandler) RegisterTimeout(check *TimeoutCheck) {
	h.timeoutMu.Lock()
	defer h.timeoutMu.Unlock()

	key := h.timeoutKey(check.SagaID, check.StepName)
	h.pendingTimeouts[key] = check

	h.logger.Info("Timeout registered",
		"saga_id", check.SagaID,
		"step_name", check.StepName,
		"timeout_at", check.TimeoutAt)
}

// CancelTimeout cancels a pending timeout
func (h *TimeoutHandler) CancelTimeout(sagaID, stepName string) {
	h.timeoutMu.Lock()
	defer h.timeoutMu.Unlock()

	key := h.timeoutKey(sagaID, stepName)
	delete(h.pendingTimeouts, key)

	h.logger.Info("Timeout cancelled",
		"saga_id", sagaID,
		"step_name", stepName)
}

func (h *TimeoutHandler) timeoutKey(sagaID, stepName string) string {
	return sagaID + ":" + stepName
}

func (h *TimeoutHandler) runLoop(ctx context.Context) {
	defer h.wg.Done()

	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkTimeouts(ctx)
		}
	}
}

func (h *TimeoutHandler) checkTimeouts(ctx context.Context) {
	h.timeoutMu.Lock()
	now := time.Now()
	var expiredTimeouts []*TimeoutCheck

	for key, check := range h.pendingTimeouts {
		if now.After(check.TimeoutAt) {
			expiredTimeouts = append(expiredTimeouts, check)
			delete(h.pendingTimeouts, key)
		}
	}
	h.timeoutMu.Unlock()

	for _, check := range expiredTimeouts {
		h.handleExpiredTimeout(ctx, check)
	}
}

func (h *TimeoutHandler) handleExpiredTimeout(ctx context.Context, check *TimeoutCheck) {
	h.logger.Warn("Step timeout expired",
		"saga_id", check.SagaID,
		"step_name", check.StepName)

	// Get saga instance to check current status
	instance, err := h.store.Get(ctx, check.SagaID)
	if err != nil {
		h.logger.Error("Failed to get saga instance for timeout",
			"saga_id", check.SagaID,
			"error", err)
		return
	}

	// Only handle timeout if saga is still running
	if instance.Status != pkgsaga.StatusRunning {
		h.logger.Info("Saga is not running, skipping timeout",
			"saga_id", check.SagaID,
			"status", instance.Status)
		return
	}

	// Check if the step was already completed
	for _, result := range instance.StepResults {
		if result.StepName == check.StepName && result.Status == pkgsaga.StepStatusCompleted {
			h.logger.Info("Step already completed, skipping timeout",
				"saga_id", check.SagaID,
				"step_name", check.StepName)
			return
		}
	}

	// Trigger compensation
	h.triggerTimeoutCompensation(ctx, check, instance)
}

func (h *TimeoutHandler) triggerTimeoutCompensation(ctx context.Context, check *TimeoutCheck, instance *pkgsaga.Instance) {
	// Update saga status to compensating
	instance.SetStatus(pkgsaga.StatusCompensating)
	instance.SetError(fmt.Errorf("step %s timed out after %s", check.StepName, time.Since(check.TimeoutAt.Add(-time.Since(check.TimeoutAt)))))

	if err := h.store.Update(ctx, instance); err != nil {
		h.logger.Error("Failed to update saga status",
			"saga_id", instance.ID,
			"error", err)
	}

	// Send timeout failure event
	failureEvent := NewSagaFailureEvent(
		check.SagaID,
		check.SagaName,
		check.StepName,
		check.StepIndex,
		fmt.Sprintf("step timed out after waiting"),
		"STEP_TIMEOUT",
		check.TimeoutAt.Add(-30*time.Second), // approximate start time
		time.Now(),
	)

	if err := h.producer.SendStepFailureEvent(ctx, failureEvent); err != nil {
		h.logger.Error("Failed to send timeout failure event",
			"saga_id", check.SagaID,
			"error", err)
	}

	// Send failed lifecycle event
	sagaFailedEvent := NewSagaFailedEvent(
		instance.ID,
		instance.DefinitionID,
		fmt.Sprintf("step %s timed out", check.StepName),
		instance.CreatedAt,
	)
	if err := h.producer.SendSagaFailedEvent(ctx, sagaFailedEvent); err != nil {
		h.logger.Error("Failed to send saga failed event",
			"saga_id", check.SagaID,
			"error", err)
	}

	// Send compensation commands for completed steps
	h.sendCompensationCommands(ctx, check, instance)
}

func (h *TimeoutHandler) sendCompensationCommands(ctx context.Context, check *TimeoutCheck, instance *pkgsaga.Instance) {
	if h.orchestrator == nil {
		return
	}

	def, err := h.orchestrator.GetDefinition(instance.DefinitionID)
	if err != nil {
		h.logger.Error("Failed to get saga definition",
			"saga_id", instance.ID,
			"error", err)
		return
	}

	// Send compensation commands in reverse order for completed steps
	for i := check.StepIndex - 1; i >= 0; i-- {
		step := def.Steps[i]

		// Check if this step was completed
		stepCompleted := false
		for _, result := range instance.StepResults {
			if result.StepName == step.Name && result.Status == pkgsaga.StepStatusCompleted {
				stepCompleted = true
				break
			}
		}

		if !stepCompleted {
			continue
		}

		// Check if step has compensation
		compensationTopic := StepToCompensationTopic(step.Name)
		if compensationTopic == "" {
			continue
		}

		command := NewCompensationCommand(
			instance.ID,
			instance.DefinitionID,
			step.Name,
			i,
			instance.GetData(),
			fmt.Sprintf("timeout at step %s", check.StepName),
		)

		if err := h.producer.SendCompensationCommand(ctx, command); err != nil {
			h.logger.Error("Failed to send compensation command",
				"saga_id", instance.ID,
				"step_name", step.Name,
				"error", err)
		}
	}

	// Update saga status to compensated
	instance.SetStatus(pkgsaga.StatusCompensated)
	now := time.Now()
	instance.CompletedAt = &now

	if err := h.store.Update(ctx, instance); err != nil {
		h.logger.Error("Failed to update saga to compensated",
			"saga_id", instance.ID,
			"error", err)
	}

	// Send compensated lifecycle event
	compensatedEvent := NewSagaCompensatedEvent(
		instance.ID,
		instance.DefinitionID,
		fmt.Sprintf("step %s timed out", check.StepName),
		instance.CreatedAt,
	)
	if err := h.producer.SendSagaCompensatedEvent(ctx, compensatedEvent); err != nil {
		h.logger.Error("Failed to send saga compensated event",
			"saga_id", check.SagaID,
			"error", err)
	}
}

// GetPendingTimeouts returns all pending timeouts (for testing/monitoring)
func (h *TimeoutHandler) GetPendingTimeouts() []*TimeoutCheck {
	h.timeoutMu.RLock()
	defer h.timeoutMu.RUnlock()

	timeouts := make([]*TimeoutCheck, 0, len(h.pendingTimeouts))
	for _, check := range h.pendingTimeouts {
		timeouts = append(timeouts, check)
	}
	return timeouts
}

// MockTimeoutHandler is a mock implementation for testing
type MockTimeoutHandler struct {
	mu               sync.RWMutex
	registeredChecks []*TimeoutCheck
	cancelledChecks  map[string]bool
	running          bool
}

// NewMockTimeoutHandler creates a new mock timeout handler
func NewMockTimeoutHandler() *MockTimeoutHandler {
	return &MockTimeoutHandler{
		registeredChecks: make([]*TimeoutCheck, 0),
		cancelledChecks:  make(map[string]bool),
	}
}

func (m *MockTimeoutHandler) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = true
	return nil
}

func (m *MockTimeoutHandler) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	return nil
}

func (m *MockTimeoutHandler) RegisterTimeout(check *TimeoutCheck) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registeredChecks = append(m.registeredChecks, check)
}

func (m *MockTimeoutHandler) CancelTimeout(sagaID, stepName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sagaID + ":" + stepName
	m.cancelledChecks[key] = true
}

func (m *MockTimeoutHandler) GetRegisteredChecks() []*TimeoutCheck {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registeredChecks
}

func (m *MockTimeoutHandler) IsCancelled(sagaID, stepName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := sagaID + ":" + stepName
	return m.cancelledChecks[key]
}

func (m *MockTimeoutHandler) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registeredChecks = make([]*TimeoutCheck, 0)
	m.cancelledChecks = make(map[string]bool)
}
