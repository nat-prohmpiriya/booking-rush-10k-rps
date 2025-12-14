package saga

import (
	"context"
	"fmt"
	"time"

	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

// OrchestratorEventHandler handles saga events and advances the saga
type OrchestratorEventHandler struct {
	orchestrator *pkgsaga.Orchestrator
	producer     SagaProducer
	store        pkgsaga.Store
	logger       Logger
}

// NewOrchestratorEventHandler creates a new orchestrator event handler
func NewOrchestratorEventHandler(
	orchestrator *pkgsaga.Orchestrator,
	producer SagaProducer,
	store pkgsaga.Store,
) *OrchestratorEventHandler {
	return &OrchestratorEventHandler{
		orchestrator: orchestrator,
		producer:     producer,
		store:        store,
		logger:       &ZapLogger{},
	}
}

// HandleStepSuccess handles a successful step completion
func (h *OrchestratorEventHandler) HandleStepSuccess(ctx context.Context, event *SagaEvent) error {
	h.logger.Info("Handling step success",
		"saga_id", event.SagaID,
		"step_name", event.StepName,
		"step_index", event.StepIndex)

	// Get saga instance
	instance, err := h.store.Get(ctx, event.SagaID)
	if err != nil {
		return fmt.Errorf("failed to get saga instance: %w", err)
	}

	if instance == nil {
		h.logger.Warn("Saga instance not found", "saga_id", event.SagaID)
		return nil
	}

	// Update saga data with step result
	if event.Data != nil {
		instance.UpdateData(event.Data)
	}

	// Add step result
	instance.AddStepResult(&pkgsaga.StepResult{
		StepName:   event.StepName,
		Status:     pkgsaga.StepStatusCompleted,
		Data:       event.Data,
		StartedAt:  event.StartedAt,
		FinishedAt: event.FinishedAt,
		Duration:   event.Duration,
	})

	// Determine next step
	nextStepName := h.getNextStep(event.StepName)
	if nextStepName == "" {
		// Saga completed
		return h.completeSaga(ctx, instance)
	}

	// Update current step
	instance.CurrentStep = event.StepIndex + 1

	// Save updated instance
	if err := h.store.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update saga instance: %w", err)
	}

	// Send next step command
	command := NewSagaCommand(
		event.SagaID,
		event.SagaName,
		nextStepName,
		event.StepIndex+1,
		instance.GetData(),
		30*time.Second,
		2,
	)

	if err := h.producer.SendCommand(ctx, command); err != nil {
		return fmt.Errorf("failed to send next step command: %w", err)
	}

	h.logger.Info("Sent next step command",
		"saga_id", event.SagaID,
		"next_step", nextStepName)

	return nil
}

// HandleStepFailure handles a failed step
func (h *OrchestratorEventHandler) HandleStepFailure(ctx context.Context, event *SagaEvent) error {
	h.logger.Error("Handling step failure",
		"saga_id", event.SagaID,
		"step_name", event.StepName,
		"error", event.ErrorMessage)

	// Get saga instance
	instance, err := h.store.Get(ctx, event.SagaID)
	if err != nil {
		return fmt.Errorf("failed to get saga instance: %w", err)
	}

	if instance == nil {
		h.logger.Warn("Saga instance not found", "saga_id", event.SagaID)
		return nil
	}

	// Add failed step result
	instance.AddStepResult(&pkgsaga.StepResult{
		StepName:   event.StepName,
		Status:     pkgsaga.StepStatusFailed,
		Error:      event.ErrorMessage,
		StartedAt:  event.StartedAt,
		FinishedAt: event.FinishedAt,
		Duration:   event.Duration,
	})

	// Set error and start compensation
	instance.SetError(fmt.Errorf("%s", event.ErrorMessage))
	instance.SetStatus(pkgsaga.StatusCompensating)

	if err := h.store.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update saga instance: %w", err)
	}

	// Start compensation from previous steps
	return h.startCompensation(ctx, instance, event.StepIndex)
}

// HandleTimeout handles a step timeout
func (h *OrchestratorEventHandler) HandleTimeout(ctx context.Context, check *TimeoutCheck) error {
	h.logger.Warn("Handling step timeout",
		"saga_id", check.SagaID,
		"step_name", check.StepName)

	// Get saga instance
	instance, err := h.store.Get(ctx, check.SagaID)
	if err != nil {
		return fmt.Errorf("failed to get saga instance: %w", err)
	}

	if instance == nil || instance.Status != pkgsaga.StatusRunning {
		// Saga already completed or doesn't exist
		return nil
	}

	// Check if step is still pending
	for _, result := range instance.StepResults {
		if result.StepName == check.StepName && result.Status == pkgsaga.StepStatusCompleted {
			// Step already completed
			return nil
		}
	}

	// Timeout occurred, start compensation
	instance.SetError(fmt.Errorf("step %s timed out", check.StepName))
	instance.SetStatus(pkgsaga.StatusCompensating)

	if err := h.store.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update saga instance: %w", err)
	}

	return h.startCompensation(ctx, instance, check.StepIndex)
}

// startCompensation starts compensating from the given step index
func (h *OrchestratorEventHandler) startCompensation(ctx context.Context, instance *pkgsaga.Instance, fromStep int) error {
	// Get completed steps that need compensation (reverse order)
	for i := fromStep - 1; i >= 0; i-- {
		stepName := h.getStepByIndex(i)
		if stepName == "" {
			continue
		}

		compensationTopic := StepToCompensationTopic(stepName)
		if compensationTopic == "" {
			// No compensation for this step
			continue
		}

		// Check if step was completed
		completed := false
		for _, result := range instance.StepResults {
			if result.StepName == stepName && result.Status == pkgsaga.StepStatusCompleted {
				completed = true
				break
			}
		}

		if !completed {
			continue
		}

		// Send compensation command
		command := NewCompensationCommand(
			instance.ID,
			instance.DefinitionID,
			stepName,
			i,
			instance.GetData(),
			"Step failed, compensating",
		)

		if err := h.producer.SendCompensationCommand(ctx, command); err != nil {
			h.logger.Error("Failed to send compensation command",
				"saga_id", instance.ID,
				"step_name", stepName,
				"error", err)
		} else {
			h.logger.Info("Sent compensation command",
				"saga_id", instance.ID,
				"step_name", stepName)
		}
	}

	// Mark saga as compensated
	instance.SetStatus(pkgsaga.StatusCompensated)
	now := time.Now()
	instance.CompletedAt = &now

	if err := h.store.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update compensated saga: %w", err)
	}

	// Send compensated event
	compensatedEvent := NewSagaCompensatedEvent(
		instance.ID,
		instance.DefinitionID,
		instance.Error,
		instance.CreatedAt,
	)
	if err := h.producer.SendSagaCompensatedEvent(ctx, compensatedEvent); err != nil {
		h.logger.Warn("Failed to send saga compensated event", "error", err)
	}

	return nil
}

// completeSaga marks the saga as completed
func (h *OrchestratorEventHandler) completeSaga(ctx context.Context, instance *pkgsaga.Instance) error {
	instance.Complete()

	if err := h.store.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update completed saga: %w", err)
	}

	// Send completed event
	completedEvent := NewSagaCompletedEvent(
		instance.ID,
		instance.DefinitionID,
		instance.GetData(),
		instance.CreatedAt,
	)
	if err := h.producer.SendSagaCompletedEvent(ctx, completedEvent); err != nil {
		h.logger.Warn("Failed to send saga completed event", "error", err)
	}

	h.logger.Info("Saga completed successfully", "saga_id", instance.ID)

	return nil
}

// getNextStep returns the next step name after the given step
func (h *OrchestratorEventHandler) getNextStep(currentStep string) string {
	switch currentStep {
	case StepReserveSeats:
		return StepProcessPayment
	case StepProcessPayment:
		return StepConfirmBooking
	case StepConfirmBooking:
		return StepSendNotification
	case StepSendNotification:
		return "" // Last step
	default:
		return ""
	}
}

// getStepByIndex returns the step name for the given index
func (h *OrchestratorEventHandler) getStepByIndex(index int) string {
	steps := []string{
		StepReserveSeats,
		StepProcessPayment,
		StepConfirmBooking,
		StepSendNotification,
	}
	if index < 0 || index >= len(steps) {
		return ""
	}
	return steps[index]
}

// ZapLogger implements saga.Logger using zap
type ZapLogger struct{}

func (l *ZapLogger) Info(msg string, fields ...interface{}) {
	log := getLogger()
	if log != nil {
		log.Info(formatLogMessage(msg, fields...))
	}
}

func (l *ZapLogger) Warn(msg string, fields ...interface{}) {
	log := getLogger()
	if log != nil {
		log.Warn(formatLogMessage(msg, fields...))
	}
}

func (l *ZapLogger) Error(msg string, fields ...interface{}) {
	log := getLogger()
	if log != nil {
		log.Error(formatLogMessage(msg, fields...))
	}
}

func getLogger() interface{ Info(string); Warn(string); Error(string) } {
	// Import cycle prevention - use simple fmt for now
	return &simpleLogger{}
}

type simpleLogger struct{}

func (l *simpleLogger) Info(msg string)  { fmt.Printf("[INFO] %s\n", msg) }
func (l *simpleLogger) Warn(msg string)  { fmt.Printf("[WARN] %s\n", msg) }
func (l *simpleLogger) Error(msg string) { fmt.Printf("[ERROR] %s\n", msg) }

func formatLogMessage(msg string, fields ...interface{}) string {
	if len(fields) == 0 {
		return msg
	}
	return fmt.Sprintf("%s %v", msg, fields)
}
