package saga

import (
	"encoding/json"
	"time"
)

// MessageType represents the type of saga message
type MessageType string

const (
	MessageTypeCommand MessageType = "command"
	MessageTypeEvent   MessageType = "event"
)

// SagaMessage is the base structure for all saga Kafka messages
type SagaMessage struct {
	// Message metadata
	MessageID     string            `json:"message_id"`
	CorrelationID string            `json:"correlation_id"` // Saga instance ID
	MessageType   MessageType       `json:"message_type"`
	Timestamp     time.Time         `json:"timestamp"`
	Headers       map[string]string `json:"headers,omitempty"`

	// Saga context
	SagaID       string `json:"saga_id"`
	SagaName     string `json:"saga_name"`
	StepName     string `json:"step_name"`
	StepIndex    int    `json:"step_index"`

	// Message payload
	Payload json.RawMessage `json:"payload"`
}

// SagaCommand represents a command message sent to trigger a saga step
type SagaCommand struct {
	SagaMessage

	// Command specific fields
	IdempotencyKey string                 `json:"idempotency_key"`
	TimeoutAt      time.Time              `json:"timeout_at"`
	RetryCount     int                    `json:"retry_count"`
	MaxRetries     int                    `json:"max_retries"`
	Data           map[string]interface{} `json:"data"`
}

// NewSagaCommand creates a new saga command
func NewSagaCommand(sagaID, sagaName, stepName string, stepIndex int, data map[string]interface{}, timeout time.Duration, maxRetries int) *SagaCommand {
	payload, _ := json.Marshal(data)

	return &SagaCommand{
		SagaMessage: SagaMessage{
			MessageID:     generateMessageID(),
			CorrelationID: sagaID,
			MessageType:   MessageTypeCommand,
			Timestamp:     time.Now(),
			SagaID:        sagaID,
			SagaName:      sagaName,
			StepName:      stepName,
			StepIndex:     stepIndex,
			Payload:       payload,
		},
		IdempotencyKey: generateIdempotencyKey(sagaID, stepName),
		TimeoutAt:      time.Now().Add(timeout),
		RetryCount:     0,
		MaxRetries:     maxRetries,
		Data:           data,
	}
}

// SagaEvent represents an event message published after step execution
type SagaEvent struct {
	SagaMessage

	// Event specific fields
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`

	// Timing information
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt time.Time     `json:"finished_at"`
	Duration   time.Duration `json:"duration_ms"`
}

// NewSagaSuccessEvent creates a new success event
func NewSagaSuccessEvent(sagaID, sagaName, stepName string, stepIndex int, data map[string]interface{}, startedAt, finishedAt time.Time) *SagaEvent {
	payload, _ := json.Marshal(data)

	return &SagaEvent{
		SagaMessage: SagaMessage{
			MessageID:     generateMessageID(),
			CorrelationID: sagaID,
			MessageType:   MessageTypeEvent,
			Timestamp:     time.Now(),
			SagaID:        sagaID,
			SagaName:      sagaName,
			StepName:      stepName,
			StepIndex:     stepIndex,
			Payload:       payload,
		},
		Success:    true,
		Data:       data,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Duration:   finishedAt.Sub(startedAt),
	}
}

// NewSagaFailureEvent creates a new failure event
func NewSagaFailureEvent(sagaID, sagaName, stepName string, stepIndex int, errorMessage, errorCode string, startedAt, finishedAt time.Time) *SagaEvent {
	return &SagaEvent{
		SagaMessage: SagaMessage{
			MessageID:     generateMessageID(),
			CorrelationID: sagaID,
			MessageType:   MessageTypeEvent,
			Timestamp:     time.Now(),
			SagaID:        sagaID,
			SagaName:      sagaName,
			StepName:      stepName,
			StepIndex:     stepIndex,
		},
		Success:      false,
		ErrorMessage: errorMessage,
		ErrorCode:    errorCode,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		Duration:     finishedAt.Sub(startedAt),
	}
}

// SagaLifecycleEvent represents saga lifecycle events (started, completed, failed, compensated)
type SagaLifecycleEvent struct {
	MessageID     string    `json:"message_id"`
	SagaID        string    `json:"saga_id"`
	SagaName      string    `json:"saga_name"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`

	// For failed/compensated status
	ErrorMessage string `json:"error_message,omitempty"`

	// Saga data
	Data map[string]interface{} `json:"data,omitempty"`

	// Timing
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration_ms,omitempty"`
}

// NewSagaStartedEvent creates a saga started event
func NewSagaStartedEvent(sagaID, sagaName string, data map[string]interface{}) *SagaLifecycleEvent {
	now := time.Now()
	return &SagaLifecycleEvent{
		MessageID: generateMessageID(),
		SagaID:    sagaID,
		SagaName:  sagaName,
		Status:    "started",
		Timestamp: now,
		Data:      data,
		StartedAt: now,
	}
}

// NewSagaCompletedEvent creates a saga completed event
func NewSagaCompletedEvent(sagaID, sagaName string, data map[string]interface{}, startedAt time.Time) *SagaLifecycleEvent {
	now := time.Now()
	return &SagaLifecycleEvent{
		MessageID:   generateMessageID(),
		SagaID:      sagaID,
		SagaName:    sagaName,
		Status:      "completed",
		Timestamp:   now,
		Data:        data,
		StartedAt:   startedAt,
		CompletedAt: now,
		Duration:    now.Sub(startedAt),
	}
}

// NewSagaFailedEvent creates a saga failed event
func NewSagaFailedEvent(sagaID, sagaName, errorMessage string, startedAt time.Time) *SagaLifecycleEvent {
	now := time.Now()
	return &SagaLifecycleEvent{
		MessageID:    generateMessageID(),
		SagaID:       sagaID,
		SagaName:     sagaName,
		Status:       "failed",
		Timestamp:    now,
		ErrorMessage: errorMessage,
		StartedAt:    startedAt,
		CompletedAt:  now,
		Duration:     now.Sub(startedAt),
	}
}

// NewSagaCompensatedEvent creates a saga compensated event
func NewSagaCompensatedEvent(sagaID, sagaName, errorMessage string, startedAt time.Time) *SagaLifecycleEvent {
	now := time.Now()
	return &SagaLifecycleEvent{
		MessageID:    generateMessageID(),
		SagaID:       sagaID,
		SagaName:     sagaName,
		Status:       "compensated",
		Timestamp:    now,
		ErrorMessage: errorMessage,
		StartedAt:    startedAt,
		CompletedAt:  now,
		Duration:     now.Sub(startedAt),
	}
}

// CompensationCommand represents a compensation command message
type CompensationCommand struct {
	SagaMessage

	// Compensation specific fields
	OriginalStepData map[string]interface{} `json:"original_step_data"`
	Reason           string                 `json:"reason"`
}

// NewCompensationCommand creates a new compensation command
func NewCompensationCommand(sagaID, sagaName, stepName string, stepIndex int, originalData map[string]interface{}, reason string) *CompensationCommand {
	payload, _ := json.Marshal(originalData)

	return &CompensationCommand{
		SagaMessage: SagaMessage{
			MessageID:     generateMessageID(),
			CorrelationID: sagaID,
			MessageType:   MessageTypeCommand,
			Timestamp:     time.Now(),
			SagaID:        sagaID,
			SagaName:      sagaName,
			StepName:      stepName,
			StepIndex:     stepIndex,
			Payload:       payload,
		},
		OriginalStepData: originalData,
		Reason:           reason,
	}
}

// TimeoutCheck represents a message for checking step timeout
type TimeoutCheck struct {
	MessageID   string    `json:"message_id"`
	SagaID      string    `json:"saga_id"`
	SagaName    string    `json:"saga_name"`
	StepName    string    `json:"step_name"`
	StepIndex   int       `json:"step_index"`
	TimeoutAt   time.Time `json:"timeout_at"`
	CheckCount  int       `json:"check_count"`
	MaxChecks   int       `json:"max_checks"`
}

// NewTimeoutCheck creates a new timeout check message
func NewTimeoutCheck(sagaID, sagaName, stepName string, stepIndex int, timeoutAt time.Time, maxChecks int) *TimeoutCheck {
	return &TimeoutCheck{
		MessageID:  generateMessageID(),
		SagaID:     sagaID,
		SagaName:   sagaName,
		StepName:   stepName,
		StepIndex:  stepIndex,
		TimeoutAt:  timeoutAt,
		CheckCount: 0,
		MaxChecks:  maxChecks,
	}
}

// IsTimedOut checks if the step has timed out
func (tc *TimeoutCheck) IsTimedOut() bool {
	return time.Now().After(tc.TimeoutAt)
}

// helper functions

func generateMessageID() string {
	return generateConfirmationCode() + "-" + generateConfirmationCode()
}

func generateIdempotencyKey(sagaID, stepName string) string {
	return sagaID + ":" + stepName
}
