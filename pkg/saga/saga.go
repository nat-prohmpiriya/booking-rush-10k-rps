package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status represents the current status of a saga
type Status string

const (
	StatusPending      Status = "pending"
	StatusRunning      Status = "running"
	StatusCompleted    Status = "completed"
	StatusFailed       Status = "failed"
	StatusCompensating Status = "compensating"
	StatusCompensated  Status = "compensated"
)

// StepStatus represents the status of a saga step
type StepStatus string

const (
	StepStatusPending      StepStatus = "pending"
	StepStatusRunning      StepStatus = "running"
	StepStatusCompleted    StepStatus = "completed"
	StepStatusFailed       StepStatus = "failed"
	StepStatusCompensating StepStatus = "compensating"
	StepStatusCompensated  StepStatus = "compensated"
	StepStatusSkipped      StepStatus = "skipped"
)

// ExecuteFunc is the function signature for step execution
type ExecuteFunc func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)

// CompensateFunc is the function signature for step compensation
type CompensateFunc func(ctx context.Context, data map[string]interface{}) error

// Step represents a single step in a saga
type Step struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Execute     ExecuteFunc    `json:"-"`
	Compensate  CompensateFunc `json:"-"`
	Timeout     time.Duration  `json:"timeout"`
	Retries     int            `json:"retries"`
}

// StepResult represents the result of executing a step
type StepResult struct {
	StepName   string                 `json:"step_name"`
	Status     StepStatus             `json:"status"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Error      string                 `json:"error,omitempty"`
	StartedAt  time.Time              `json:"started_at"`
	FinishedAt time.Time              `json:"finished_at,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
}

// Definition defines a saga with its steps
type Definition struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Steps       []*Step       `json:"steps"`
	Timeout     time.Duration `json:"timeout"`
}

// NewDefinition creates a new saga definition
func NewDefinition(name, description string) *Definition {
	return &Definition{
		Name:        name,
		Description: description,
		Steps:       make([]*Step, 0),
		Timeout:     5 * time.Minute, // Default timeout
	}
}

// AddStep adds a step to the saga definition
func (d *Definition) AddStep(step *Step) *Definition {
	if step.Timeout == 0 {
		step.Timeout = 30 * time.Second // Default step timeout
	}
	d.Steps = append(d.Steps, step)
	return d
}

// WithTimeout sets the overall saga timeout
func (d *Definition) WithTimeout(timeout time.Duration) *Definition {
	d.Timeout = timeout
	return d
}

// Instance represents a running or completed saga instance
type Instance struct {
	ID           string                 `json:"id"`
	DefinitionID string                 `json:"definition_id"`
	Status       Status                 `json:"status"`
	Data         map[string]interface{} `json:"data"`
	StepResults  []*StepResult          `json:"step_results"`
	CurrentStep  int                    `json:"current_step"`
	Error        string                 `json:"error,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`

	mu sync.RWMutex
}

// NewInstance creates a new saga instance
func NewInstance(definitionID string, initialData map[string]interface{}) *Instance {
	now := time.Now()
	if initialData == nil {
		initialData = make(map[string]interface{})
	}
	return &Instance{
		ID:           uuid.New().String(),
		DefinitionID: definitionID,
		Status:       StatusPending,
		Data:         initialData,
		StepResults:  make([]*StepResult, 0),
		CurrentStep:  0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetStatus updates the saga status
func (i *Instance) SetStatus(status Status) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Status = status
	i.UpdatedAt = time.Now()
}

// GetStatus returns the current saga status
func (i *Instance) GetStatus() Status {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.Status
}

// AddStepResult adds a step result to the saga
func (i *Instance) AddStepResult(result *StepResult) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.StepResults = append(i.StepResults, result)
	i.UpdatedAt = time.Now()
}

// UpdateData merges new data into the saga data
func (i *Instance) UpdateData(data map[string]interface{}) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for k, v := range data {
		i.Data[k] = v
	}
	i.UpdatedAt = time.Now()
}

// GetData returns a copy of the saga data
func (i *Instance) GetData() map[string]interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range i.Data {
		result[k] = v
	}
	return result
}

// SetError sets the saga error
func (i *Instance) SetError(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if err != nil {
		i.Error = err.Error()
	}
	i.UpdatedAt = time.Now()
}

// Complete marks the saga as completed
func (i *Instance) Complete() {
	i.mu.Lock()
	defer i.mu.Unlock()
	now := time.Now()
	i.Status = StatusCompleted
	i.CompletedAt = &now
	i.UpdatedAt = now
}

// Fail marks the saga as failed
func (i *Instance) Fail(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	now := time.Now()
	i.Status = StatusFailed
	if err != nil {
		i.Error = err.Error()
	}
	i.CompletedAt = &now
	i.UpdatedAt = now
}

// ToJSON serializes the saga instance to JSON
func (i *Instance) ToJSON() ([]byte, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return json.Marshal(i)
}

// FromJSON deserializes the saga instance from JSON
func FromJSON(data []byte) (*Instance, error) {
	var instance Instance
	if err := json.Unmarshal(data, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal saga instance: %w", err)
	}
	return &instance, nil
}
