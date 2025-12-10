package worker

import (
	"testing"
	"time"
)

func TestDefaultOutboxWorkerConfig(t *testing.T) {
	config := DefaultOutboxWorkerConfig()

	if config.PollInterval != 100*time.Millisecond {
		t.Errorf("PollInterval = %v, want %v", config.PollInterval, 100*time.Millisecond)
	}

	if config.BatchSize != 100 {
		t.Errorf("BatchSize = %v, want %v", config.BatchSize, 100)
	}

	if config.RetryInterval != 5*time.Second {
		t.Errorf("RetryInterval = %v, want %v", config.RetryInterval, 5*time.Second)
	}

	if config.CleanupInterval != 1*time.Hour {
		t.Errorf("CleanupInterval = %v, want %v", config.CleanupInterval, 1*time.Hour)
	}

	if config.CleanupRetentionDays != 7 {
		t.Errorf("CleanupRetentionDays = %v, want %v", config.CleanupRetentionDays, 7)
	}
}

func TestOutboxWorkerConfig_Custom(t *testing.T) {
	config := &OutboxWorkerConfig{
		PollInterval:         50 * time.Millisecond,
		BatchSize:            50,
		RetryInterval:        10 * time.Second,
		CleanupInterval:      2 * time.Hour,
		CleanupRetentionDays: 14,
	}

	if config.PollInterval != 50*time.Millisecond {
		t.Errorf("PollInterval = %v, want %v", config.PollInterval, 50*time.Millisecond)
	}

	if config.BatchSize != 50 {
		t.Errorf("BatchSize = %v, want %v", config.BatchSize, 50)
	}

	if config.RetryInterval != 10*time.Second {
		t.Errorf("RetryInterval = %v, want %v", config.RetryInterval, 10*time.Second)
	}

	if config.CleanupInterval != 2*time.Hour {
		t.Errorf("CleanupInterval = %v, want %v", config.CleanupInterval, 2*time.Hour)
	}

	if config.CleanupRetentionDays != 14 {
		t.Errorf("CleanupRetentionDays = %v, want %v", config.CleanupRetentionDays, 14)
	}
}

func TestNewOutboxWorker_WithDefaultConfig(t *testing.T) {
	worker := NewOutboxWorker(nil, nil, nil)

	if worker == nil {
		t.Fatal("NewOutboxWorker() returned nil")
	}

	if worker.config == nil {
		t.Fatal("Worker config should not be nil")
	}

	if worker.config.PollInterval != 100*time.Millisecond {
		t.Errorf("Default PollInterval = %v, want %v", worker.config.PollInterval, 100*time.Millisecond)
	}

	if worker.running {
		t.Error("Worker should not be running initially")
	}
}

func TestNewOutboxWorker_WithCustomConfig(t *testing.T) {
	customConfig := &OutboxWorkerConfig{
		PollInterval:         200 * time.Millisecond,
		BatchSize:            200,
		RetryInterval:        15 * time.Second,
		CleanupInterval:      30 * time.Minute,
		CleanupRetentionDays: 3,
	}

	worker := NewOutboxWorker(nil, nil, customConfig)

	if worker == nil {
		t.Fatal("NewOutboxWorker() returned nil")
	}

	if worker.config.PollInterval != 200*time.Millisecond {
		t.Errorf("PollInterval = %v, want %v", worker.config.PollInterval, 200*time.Millisecond)
	}

	if worker.config.BatchSize != 200 {
		t.Errorf("BatchSize = %v, want %v", worker.config.BatchSize, 200)
	}
}

func TestOutboxWorkerStats(t *testing.T) {
	stats := &OutboxWorkerStats{
		IsRunning:       true,
		PendingMessages: true,
		FailedMessages:  false,
	}

	if !stats.IsRunning {
		t.Error("IsRunning should be true")
	}

	if !stats.PendingMessages {
		t.Error("PendingMessages should be true")
	}

	if stats.FailedMessages {
		t.Error("FailedMessages should be false")
	}
}
