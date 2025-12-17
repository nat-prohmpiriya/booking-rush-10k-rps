package saga

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Orchestrator manages saga execution and compensation
type Orchestrator struct {
	definitions map[string]*Definition
	store       Store
	mu          sync.RWMutex
	logger      Logger
}

// Logger interface for saga logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	// Context-aware logging methods
	InfoContext(ctx context.Context, msg string, fields ...interface{})
	WarnContext(ctx context.Context, msg string, fields ...interface{})
	ErrorContext(ctx context.Context, msg string, fields ...interface{})
}

// NoOpLogger is a no-op logger implementation
type NoOpLogger struct{}

func (l *NoOpLogger) Info(msg string, fields ...interface{})                             {}
func (l *NoOpLogger) Warn(msg string, fields ...interface{})                             {}
func (l *NoOpLogger) Error(msg string, fields ...interface{})                            {}
func (l *NoOpLogger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {}
func (l *NoOpLogger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {}
func (l *NoOpLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {}

// OrchestratorConfig holds configuration for the orchestrator
type OrchestratorConfig struct {
	Store  Store
	Logger Logger
}

// NewOrchestrator creates a new saga orchestrator
func NewOrchestrator(cfg *OrchestratorConfig) *Orchestrator {
	store := cfg.Store
	if store == nil {
		store = NewMemoryStore()
	}

	logger := cfg.Logger
	if logger == nil {
		logger = &NoOpLogger{}
	}

	return &Orchestrator{
		definitions: make(map[string]*Definition),
		store:       store,
		logger:      logger,
	}
}

// RegisterDefinition registers a saga definition
func (o *Orchestrator) RegisterDefinition(def *Definition) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.definitions[def.Name]; exists {
		return fmt.Errorf("saga definition %s already registered", def.Name)
	}

	o.definitions[def.Name] = def
	o.logger.Info("Registered saga definition", "name", def.Name, "steps", len(def.Steps))
	return nil
}

// GetDefinition retrieves a saga definition by name
func (o *Orchestrator) GetDefinition(name string) (*Definition, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	def, exists := o.definitions[name]
	if !exists {
		return nil, fmt.Errorf("saga definition %s not found", name)
	}

	return def, nil
}

// Execute starts a new saga instance and runs it to completion
func (o *Orchestrator) Execute(ctx context.Context, definitionName string, initialData map[string]interface{}) (*Instance, error) {
	def, err := o.GetDefinition(definitionName)
	if err != nil {
		return nil, err
	}

	// Create a new saga instance
	instance := NewInstance(def.Name, initialData)
	o.logger.Info("Starting saga execution", "saga_id", instance.ID, "definition", def.Name)

	// Save initial state
	if err := o.store.Save(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to save saga instance: %w", err)
	}

	// Create saga context with timeout
	sagaCtx, cancel := context.WithTimeout(ctx, def.Timeout)
	defer cancel()

	// Execute the saga
	return o.executeSaga(sagaCtx, def, instance)
}

// executeSaga runs through all saga steps
func (o *Orchestrator) executeSaga(ctx context.Context, def *Definition, instance *Instance) (*Instance, error) {
	instance.SetStatus(StatusRunning)
	if err := o.store.Update(ctx, instance); err != nil {
		o.logger.Error("Failed to update saga status", "saga_id", instance.ID, "error", err)
	}

	var lastError error

	for i, step := range def.Steps {
		instance.CurrentStep = i

		// Check for context cancellation
		select {
		case <-ctx.Done():
			lastError = ctx.Err()
			o.logger.Warn("Saga execution cancelled", "saga_id", instance.ID, "step", step.Name)
			break
		default:
		}

		if lastError != nil {
			break
		}

		// Execute step
		result, err := o.executeStep(ctx, step, instance)
		instance.AddStepResult(result)

		if err := o.store.Update(ctx, instance); err != nil {
			o.logger.Error("Failed to update saga after step", "saga_id", instance.ID, "step", step.Name, "error", err)
		}

		if err != nil {
			lastError = err
			o.logger.Error("Step execution failed", "saga_id", instance.ID, "step", step.Name, "error", err)
			break
		}

		// Merge step result data into saga data
		if result.Data != nil {
			instance.UpdateData(result.Data)
		}

		o.logger.Info("Step completed successfully", "saga_id", instance.ID, "step", step.Name)
	}

	// If there was an error, run compensation
	if lastError != nil {
		instance.SetError(lastError)
		return o.compensate(ctx, def, instance)
	}

	// All steps completed successfully
	instance.Complete()
	if err := o.store.Update(ctx, instance); err != nil {
		o.logger.Error("Failed to update completed saga", "saga_id", instance.ID, "error", err)
	}

	o.logger.Info("Saga completed successfully", "saga_id", instance.ID)
	return instance, nil
}

// executeStep executes a single step with timeout and retry logic
func (o *Orchestrator) executeStep(ctx context.Context, step *Step, instance *Instance) (*StepResult, error) {
	result := &StepResult{
		StepName:  step.Name,
		Status:    StepStatusRunning,
		StartedAt: time.Now(),
	}

	// Create step context with timeout
	stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
	defer cancel()

	var lastError error
	maxAttempts := step.Retries + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			o.logger.Info("Retrying step", "saga_id", instance.ID, "step", step.Name, "attempt", attempt+1)
			// Exponential backoff
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		// Get current saga data
		data := instance.GetData()

		// Execute the step
		resultData, err := step.Execute(stepCtx, data)
		if err == nil {
			result.Status = StepStatusCompleted
			result.Data = resultData
			result.FinishedAt = time.Now()
			result.Duration = result.FinishedAt.Sub(result.StartedAt)
			return result, nil
		}

		lastError = err
	}

	// All retries failed
	result.Status = StepStatusFailed
	result.Error = lastError.Error()
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	return result, lastError
}

// compensate runs compensation for all completed steps in reverse order
func (o *Orchestrator) compensate(ctx context.Context, def *Definition, instance *Instance) (*Instance, error) {
	instance.SetStatus(StatusCompensating)
	if err := o.store.Update(ctx, instance); err != nil {
		o.logger.Error("Failed to update saga compensation status", "saga_id", instance.ID, "error", err)
	}

	o.logger.Info("Starting saga compensation", "saga_id", instance.ID, "completed_steps", len(instance.StepResults))

	// Find completed steps that need compensation (in reverse order)
	for i := len(instance.StepResults) - 1; i >= 0; i-- {
		stepResult := instance.StepResults[i]

		// Skip steps that weren't completed
		if stepResult.Status != StepStatusCompleted {
			continue
		}

		// Find the step definition
		var step *Step
		for _, s := range def.Steps {
			if s.Name == stepResult.StepName {
				step = s
				break
			}
		}

		if step == nil || step.Compensate == nil {
			o.logger.Warn("No compensation function for step", "saga_id", instance.ID, "step", stepResult.StepName)
			continue
		}

		// Execute compensation
		compensationResult := o.compensateStep(ctx, step, instance)
		stepResult.Status = compensationResult.Status

		if compensationResult.Status != StepStatusCompensated {
			o.logger.Error("Compensation failed", "saga_id", instance.ID, "step", step.Name, "error", compensationResult.Error)
		} else {
			o.logger.Info("Step compensated", "saga_id", instance.ID, "step", step.Name)
		}
	}

	instance.SetStatus(StatusCompensated)
	now := time.Now()
	instance.CompletedAt = &now
	instance.UpdatedAt = now

	if err := o.store.Update(ctx, instance); err != nil {
		o.logger.Error("Failed to update compensated saga", "saga_id", instance.ID, "error", err)
	}

	o.logger.Info("Saga compensation completed", "saga_id", instance.ID)

	return instance, fmt.Errorf("saga failed and was compensated: %s", instance.Error)
}

// compensateStep executes compensation for a single step
func (o *Orchestrator) compensateStep(ctx context.Context, step *Step, instance *Instance) *StepResult {
	result := &StepResult{
		StepName:  step.Name,
		Status:    StepStatusCompensating,
		StartedAt: time.Now(),
	}

	// Create step context with timeout
	stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
	defer cancel()

	// Get current saga data
	data := instance.GetData()

	// Execute compensation
	err := step.Compensate(stepCtx, data)
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	if err != nil {
		result.Status = StepStatusFailed
		result.Error = err.Error()
	} else {
		result.Status = StepStatusCompensated
	}

	return result
}

// GetInstance retrieves a saga instance by ID
func (o *Orchestrator) GetInstance(ctx context.Context, id string) (*Instance, error) {
	return o.store.Get(ctx, id)
}

// Resume resumes a saga that was interrupted
func (o *Orchestrator) Resume(ctx context.Context, instanceID string) (*Instance, error) {
	instance, err := o.store.Get(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	def, err := o.GetDefinition(instance.DefinitionID)
	if err != nil {
		return nil, err
	}

	switch instance.Status {
	case StatusPending, StatusRunning:
		// Continue execution from where it left off
		return o.resumeExecution(ctx, def, instance)
	case StatusFailed, StatusCompensating:
		// Resume compensation
		return o.compensate(ctx, def, instance)
	case StatusCompleted, StatusCompensated:
		// Already finished
		return instance, nil
	default:
		return nil, fmt.Errorf("unknown saga status: %s", instance.Status)
	}
}

// resumeExecution resumes execution from the current step
func (o *Orchestrator) resumeExecution(ctx context.Context, def *Definition, instance *Instance) (*Instance, error) {
	o.logger.Info("Resuming saga execution", "saga_id", instance.ID, "from_step", instance.CurrentStep)

	instance.SetStatus(StatusRunning)
	if err := o.store.Update(ctx, instance); err != nil {
		o.logger.Error("Failed to update saga status", "saga_id", instance.ID, "error", err)
	}

	var lastError error

	for i := instance.CurrentStep; i < len(def.Steps); i++ {
		step := def.Steps[i]
		instance.CurrentStep = i

		// Check for context cancellation
		select {
		case <-ctx.Done():
			lastError = ctx.Err()
			break
		default:
		}

		if lastError != nil {
			break
		}

		// Check if this step was already completed
		alreadyCompleted := false
		for _, result := range instance.StepResults {
			if result.StepName == step.Name && result.Status == StepStatusCompleted {
				alreadyCompleted = true
				break
			}
		}

		if alreadyCompleted {
			continue
		}

		// Execute step
		result, err := o.executeStep(ctx, step, instance)
		instance.AddStepResult(result)

		if err := o.store.Update(ctx, instance); err != nil {
			o.logger.Error("Failed to update saga after step", "saga_id", instance.ID, "step", step.Name, "error", err)
		}

		if err != nil {
			lastError = err
			break
		}

		// Merge step result data into saga data
		if result.Data != nil {
			instance.UpdateData(result.Data)
		}
	}

	// If there was an error, run compensation
	if lastError != nil {
		instance.SetError(lastError)
		return o.compensate(ctx, def, instance)
	}

	// All steps completed successfully
	instance.Complete()
	if err := o.store.Update(ctx, instance); err != nil {
		o.logger.Error("Failed to update completed saga", "saga_id", instance.ID, "error", err)
	}

	return instance, nil
}
