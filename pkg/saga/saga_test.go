package saga

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition("test-saga", "A test saga")

	if def.Name != "test-saga" {
		t.Errorf("expected name 'test-saga', got '%s'", def.Name)
	}
	if def.Description != "A test saga" {
		t.Errorf("expected description 'A test saga', got '%s'", def.Description)
	}
	if len(def.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(def.Steps))
	}
	if def.Timeout != 5*time.Minute {
		t.Errorf("expected default timeout of 5 minutes, got %v", def.Timeout)
	}
}

func TestAddStep(t *testing.T) {
	def := NewDefinition("test-saga", "A test saga")

	step := &Step{
		Name:        "step1",
		Description: "First step",
		Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
			return data, nil
		},
		Compensate: func(ctx context.Context, data map[string]interface{}) error {
			return nil
		},
	}

	def.AddStep(step)

	if len(def.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(def.Steps))
	}
	if def.Steps[0].Name != "step1" {
		t.Errorf("expected step name 'step1', got '%s'", def.Steps[0].Name)
	}
	// Check that default timeout was set
	if def.Steps[0].Timeout != 30*time.Second {
		t.Errorf("expected default step timeout of 30 seconds, got %v", def.Steps[0].Timeout)
	}
}

func TestWithTimeout(t *testing.T) {
	def := NewDefinition("test-saga", "A test saga").
		WithTimeout(10 * time.Minute)

	if def.Timeout != 10*time.Minute {
		t.Errorf("expected timeout of 10 minutes, got %v", def.Timeout)
	}
}

func TestNewInstance(t *testing.T) {
	initialData := map[string]interface{}{
		"user_id": "123",
	}
	instance := NewInstance("test-saga", initialData)

	if instance.ID == "" {
		t.Error("expected non-empty ID")
	}
	if instance.DefinitionID != "test-saga" {
		t.Errorf("expected definition ID 'test-saga', got '%s'", instance.DefinitionID)
	}
	if instance.Status != StatusPending {
		t.Errorf("expected status 'pending', got '%s'", instance.Status)
	}
	if instance.Data["user_id"] != "123" {
		t.Errorf("expected user_id '123', got '%v'", instance.Data["user_id"])
	}
}

func TestInstanceSetStatus(t *testing.T) {
	instance := NewInstance("test-saga", nil)

	instance.SetStatus(StatusRunning)
	if instance.GetStatus() != StatusRunning {
		t.Errorf("expected status 'running', got '%s'", instance.GetStatus())
	}
}

func TestInstanceUpdateData(t *testing.T) {
	instance := NewInstance("test-saga", map[string]interface{}{
		"key1": "value1",
	})

	instance.UpdateData(map[string]interface{}{
		"key2": "value2",
	})

	data := instance.GetData()
	if data["key1"] != "value1" {
		t.Errorf("expected key1 'value1', got '%v'", data["key1"])
	}
	if data["key2"] != "value2" {
		t.Errorf("expected key2 'value2', got '%v'", data["key2"])
	}
}

func TestInstanceToJSONAndFromJSON(t *testing.T) {
	instance := NewInstance("test-saga", map[string]interface{}{
		"user_id": "123",
	})
	instance.SetStatus(StatusRunning)

	jsonData, err := instance.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	restored, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	if restored.ID != instance.ID {
		t.Errorf("expected ID '%s', got '%s'", instance.ID, restored.ID)
	}
	if restored.Status != StatusRunning {
		t.Errorf("expected status 'running', got '%s'", restored.Status)
	}
	if restored.Data["user_id"] != "123" {
		t.Errorf("expected user_id '123', got '%v'", restored.Data["user_id"])
	}
}

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	// Test Save
	instance := NewInstance("test-saga", nil)
	if err := store.Save(ctx, instance); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Test duplicate save
	if err := store.Save(ctx, instance); !errors.Is(err, ErrSagaAlreadyExists) {
		t.Errorf("expected ErrSagaAlreadyExists, got %v", err)
	}

	// Test Get
	retrieved, err := store.Get(ctx, instance.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if retrieved.ID != instance.ID {
		t.Errorf("expected ID '%s', got '%s'", instance.ID, retrieved.ID)
	}

	// Test Get not found
	_, err = store.Get(ctx, "nonexistent")
	if !errors.Is(err, ErrSagaNotFound) {
		t.Errorf("expected ErrSagaNotFound, got %v", err)
	}

	// Test Update
	instance.SetStatus(StatusRunning)
	if err := store.Update(ctx, instance); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	retrieved, _ = store.Get(ctx, instance.ID)
	if retrieved.Status != StatusRunning {
		t.Errorf("expected status 'running', got '%s'", retrieved.Status)
	}

	// Test Update not found
	notExists := NewInstance("test", nil)
	if err := store.Update(ctx, notExists); !errors.Is(err, ErrSagaNotFound) {
		t.Errorf("expected ErrSagaNotFound, got %v", err)
	}

	// Test Delete
	if err := store.Delete(ctx, instance.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = store.Get(ctx, instance.ID)
	if !errors.Is(err, ErrSagaNotFound) {
		t.Errorf("expected ErrSagaNotFound after delete, got %v", err)
	}
}

func TestMemoryStoreGetByStatus(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	// Create instances with different statuses
	running1 := NewInstance("test-saga", nil)
	running1.Status = StatusRunning
	store.Save(ctx, running1)

	running2 := NewInstance("test-saga", nil)
	running2.Status = StatusRunning
	store.Save(ctx, running2)

	completed := NewInstance("test-saga", nil)
	completed.Status = StatusCompleted
	store.Save(ctx, completed)

	// Get running instances
	runningInstances, err := store.GetByStatus(ctx, StatusRunning, 0)
	if err != nil {
		t.Fatalf("failed to get by status: %v", err)
	}
	if len(runningInstances) != 2 {
		t.Errorf("expected 2 running instances, got %d", len(runningInstances))
	}

	// Get with limit
	limitedInstances, err := store.GetByStatus(ctx, StatusRunning, 1)
	if err != nil {
		t.Fatalf("failed to get by status with limit: %v", err)
	}
	if len(limitedInstances) != 1 {
		t.Errorf("expected 1 instance with limit, got %d", len(limitedInstances))
	}
}

func TestOrchestratorRegisterDefinition(t *testing.T) {
	orch := NewOrchestrator(&OrchestratorConfig{})

	def := NewDefinition("test-saga", "A test saga")
	if err := orch.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Try to register again
	if err := orch.RegisterDefinition(def); err == nil {
		t.Error("expected error when registering duplicate")
	}

	// Get definition
	retrieved, err := orch.GetDefinition("test-saga")
	if err != nil {
		t.Fatalf("failed to get definition: %v", err)
	}
	if retrieved.Name != "test-saga" {
		t.Errorf("expected name 'test-saga', got '%s'", retrieved.Name)
	}
}

func TestOrchestratorExecuteSuccess(t *testing.T) {
	ctx := context.Background()
	orch := NewOrchestrator(&OrchestratorConfig{})

	var step1Executed, step2Executed bool

	def := NewDefinition("booking-saga", "Booking saga").
		AddStep(&Step{
			Name: "reserve-seats",
			Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
				step1Executed = true
				return map[string]interface{}{"reservation_id": "res-123"}, nil
			},
			Compensate: func(ctx context.Context, data map[string]interface{}) error {
				return nil
			},
		}).
		AddStep(&Step{
			Name: "process-payment",
			Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
				step2Executed = true
				return map[string]interface{}{"payment_id": "pay-456"}, nil
			},
			Compensate: func(ctx context.Context, data map[string]interface{}) error {
				return nil
			},
		})

	orch.RegisterDefinition(def)

	instance, err := orch.Execute(ctx, "booking-saga", map[string]interface{}{
		"booking_id": "book-789",
	})

	if err != nil {
		t.Fatalf("saga execution failed: %v", err)
	}

	if !step1Executed {
		t.Error("step1 was not executed")
	}
	if !step2Executed {
		t.Error("step2 was not executed")
	}

	if instance.Status != StatusCompleted {
		t.Errorf("expected status 'completed', got '%s'", instance.Status)
	}

	// Check that data was merged
	data := instance.GetData()
	if data["reservation_id"] != "res-123" {
		t.Errorf("expected reservation_id 'res-123', got '%v'", data["reservation_id"])
	}
	if data["payment_id"] != "pay-456" {
		t.Errorf("expected payment_id 'pay-456', got '%v'", data["payment_id"])
	}
}

func TestOrchestratorExecuteWithCompensation(t *testing.T) {
	ctx := context.Background()
	orch := NewOrchestrator(&OrchestratorConfig{})

	var step1Compensated, step2Executed bool

	def := NewDefinition("booking-saga", "Booking saga").
		AddStep(&Step{
			Name: "reserve-seats",
			Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"reservation_id": "res-123"}, nil
			},
			Compensate: func(ctx context.Context, data map[string]interface{}) error {
				step1Compensated = true
				return nil
			},
		}).
		AddStep(&Step{
			Name: "process-payment",
			Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
				step2Executed = true
				return nil, errors.New("payment failed")
			},
			Compensate: func(ctx context.Context, data map[string]interface{}) error {
				return nil
			},
		})

	orch.RegisterDefinition(def)

	instance, err := orch.Execute(ctx, "booking-saga", nil)

	if err == nil {
		t.Error("expected error due to step failure")
	}

	if !step2Executed {
		t.Error("step2 should have been executed")
	}

	if !step1Compensated {
		t.Error("step1 should have been compensated")
	}

	if instance.Status != StatusCompensated {
		t.Errorf("expected status 'compensated', got '%s'", instance.Status)
	}
}

func TestOrchestratorExecuteWithRetry(t *testing.T) {
	ctx := context.Background()
	orch := NewOrchestrator(&OrchestratorConfig{})

	var attempts int32

	def := NewDefinition("retry-saga", "Saga with retry").
		AddStep(&Step{
			Name:    "flaky-step",
			Retries: 2,
			Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
				count := atomic.AddInt32(&attempts, 1)
				if count < 3 {
					return nil, errors.New("temporary failure")
				}
				return map[string]interface{}{"success": true}, nil
			},
			Compensate: func(ctx context.Context, data map[string]interface{}) error {
				return nil
			},
		})

	orch.RegisterDefinition(def)

	instance, err := orch.Execute(ctx, "retry-saga", nil)

	if err != nil {
		t.Fatalf("saga should have succeeded after retries: %v", err)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}

	if instance.Status != StatusCompleted {
		t.Errorf("expected status 'completed', got '%s'", instance.Status)
	}
}

func TestOrchestratorExecuteWithTimeout(t *testing.T) {
	ctx := context.Background()
	orch := NewOrchestrator(&OrchestratorConfig{})

	def := NewDefinition("timeout-saga", "Saga with timeout").
		WithTimeout(100 * time.Millisecond).
		AddStep(&Step{
			Name:    "slow-step",
			Timeout: 50 * time.Millisecond,
			Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(200 * time.Millisecond):
					return nil, nil
				}
			},
			Compensate: func(ctx context.Context, data map[string]interface{}) error {
				return nil
			},
		})

	orch.RegisterDefinition(def)

	instance, err := orch.Execute(ctx, "timeout-saga", nil)

	if err == nil {
		t.Error("expected timeout error")
	}

	if instance.Status != StatusCompensated {
		t.Errorf("expected status 'compensated', got '%s'", instance.Status)
	}
}

func TestOrchestratorGetInstance(t *testing.T) {
	ctx := context.Background()
	orch := NewOrchestrator(&OrchestratorConfig{})

	def := NewDefinition("simple-saga", "Simple saga").
		AddStep(&Step{
			Name: "step1",
			Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		})

	orch.RegisterDefinition(def)

	instance, _ := orch.Execute(ctx, "simple-saga", nil)

	retrieved, err := orch.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}

	if retrieved.ID != instance.ID {
		t.Errorf("expected ID '%s', got '%s'", instance.ID, retrieved.ID)
	}
}

func TestStepResultFields(t *testing.T) {
	ctx := context.Background()
	orch := NewOrchestrator(&OrchestratorConfig{})

	def := NewDefinition("result-saga", "Saga for testing results").
		AddStep(&Step{
			Name: "test-step",
			Execute: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return map[string]interface{}{"result": "success"}, nil
			},
		})

	orch.RegisterDefinition(def)

	instance, _ := orch.Execute(ctx, "result-saga", nil)

	if len(instance.StepResults) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(instance.StepResults))
	}

	result := instance.StepResults[0]

	if result.StepName != "test-step" {
		t.Errorf("expected step name 'test-step', got '%s'", result.StepName)
	}

	if result.Status != StepStatusCompleted {
		t.Errorf("expected status 'completed', got '%s'", result.Status)
	}

	if result.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}

	if result.FinishedAt.IsZero() {
		t.Error("expected FinishedAt to be set")
	}

	if result.Duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", result.Duration)
	}
}
