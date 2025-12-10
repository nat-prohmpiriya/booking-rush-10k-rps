package saga

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestKafkaTopics(t *testing.T) {
	t.Run("GetAllCommandTopics", func(t *testing.T) {
		topics := GetAllCommandTopics()
		if len(topics) != 6 {
			t.Errorf("expected 6 command topics, got %d", len(topics))
		}

		expectedTopics := []string{
			TopicSagaReserveSeatsCommand,
			TopicSagaProcessPaymentCommand,
			TopicSagaConfirmBookingCommand,
			TopicSagaSendNotificationCommand,
			TopicSagaReleaseSeatsCommand,
			TopicSagaRefundPaymentCommand,
		}

		for i, expected := range expectedTopics {
			if topics[i] != expected {
				t.Errorf("expected topic %s at index %d, got %s", expected, i, topics[i])
			}
		}
	})

	t.Run("GetAllEventTopics", func(t *testing.T) {
		topics := GetAllEventTopics()
		if len(topics) != 14 {
			t.Errorf("expected 14 event topics, got %d", len(topics))
		}
	})

	t.Run("StepToCommandTopic", func(t *testing.T) {
		tests := []struct {
			stepName string
			expected string
		}{
			{StepReserveSeats, TopicSagaReserveSeatsCommand},
			{StepProcessPayment, TopicSagaProcessPaymentCommand},
			{StepConfirmBooking, TopicSagaConfirmBookingCommand},
			{StepSendNotification, TopicSagaSendNotificationCommand},
			{"unknown", ""},
		}

		for _, tt := range tests {
			result := StepToCommandTopic(tt.stepName)
			if result != tt.expected {
				t.Errorf("StepToCommandTopic(%s) = %s, expected %s", tt.stepName, result, tt.expected)
			}
		}
	})

	t.Run("StepToCompensationTopic", func(t *testing.T) {
		tests := []struct {
			stepName string
			expected string
		}{
			{StepReserveSeats, TopicSagaReleaseSeatsCommand},
			{StepProcessPayment, TopicSagaRefundPaymentCommand},
			{StepConfirmBooking, ""},
			{StepSendNotification, ""},
		}

		for _, tt := range tests {
			result := StepToCompensationTopic(tt.stepName)
			if result != tt.expected {
				t.Errorf("StepToCompensationTopic(%s) = %s, expected %s", tt.stepName, result, tt.expected)
			}
		}
	})

	t.Run("StepToSuccessEventTopic", func(t *testing.T) {
		tests := []struct {
			stepName string
			expected string
		}{
			{StepReserveSeats, TopicSagaSeatsReservedEvent},
			{StepProcessPayment, TopicSagaPaymentProcessedEvent},
			{StepConfirmBooking, TopicSagaBookingConfirmedEvent},
			{StepSendNotification, TopicSagaNotificationSentEvent},
		}

		for _, tt := range tests {
			result := StepToSuccessEventTopic(tt.stepName)
			if result != tt.expected {
				t.Errorf("StepToSuccessEventTopic(%s) = %s, expected %s", tt.stepName, result, tt.expected)
			}
		}
	})

	t.Run("StepToFailureEventTopic", func(t *testing.T) {
		tests := []struct {
			stepName string
			expected string
		}{
			{StepReserveSeats, TopicSagaSeatsReservationFailedEvent},
			{StepProcessPayment, TopicSagaPaymentFailedEvent},
			{StepConfirmBooking, TopicSagaBookingConfirmationFailedEvent},
			{StepSendNotification, TopicSagaNotificationFailedEvent},
		}

		for _, tt := range tests {
			result := StepToFailureEventTopic(tt.stepName)
			if result != tt.expected {
				t.Errorf("StepToFailureEventTopic(%s) = %s, expected %s", tt.stepName, result, tt.expected)
			}
		}
	})
}

func TestSagaMessages(t *testing.T) {
	t.Run("NewSagaCommand", func(t *testing.T) {
		data := map[string]interface{}{
			"booking_id": "booking-123",
			"user_id":    "user-456",
		}

		command := NewSagaCommand(
			"saga-123",
			"booking-saga",
			StepReserveSeats,
			0,
			data,
			30*time.Second,
			3,
		)

		if command.SagaID != "saga-123" {
			t.Errorf("expected SagaID saga-123, got %s", command.SagaID)
		}
		if command.SagaName != "booking-saga" {
			t.Errorf("expected SagaName booking-saga, got %s", command.SagaName)
		}
		if command.StepName != StepReserveSeats {
			t.Errorf("expected StepName %s, got %s", StepReserveSeats, command.StepName)
		}
		if command.StepIndex != 0 {
			t.Errorf("expected StepIndex 0, got %d", command.StepIndex)
		}
		if command.MaxRetries != 3 {
			t.Errorf("expected MaxRetries 3, got %d", command.MaxRetries)
		}
		if command.MessageType != MessageTypeCommand {
			t.Errorf("expected MessageType command, got %s", command.MessageType)
		}
		if command.IdempotencyKey == "" {
			t.Error("expected IdempotencyKey to be set")
		}
		if command.TimeoutAt.Before(time.Now()) {
			t.Error("expected TimeoutAt to be in the future")
		}
	})

	t.Run("NewSagaSuccessEvent", func(t *testing.T) {
		data := map[string]interface{}{
			"reservation_id": "res-123",
		}
		startedAt := time.Now().Add(-1 * time.Second)
		finishedAt := time.Now()

		event := NewSagaSuccessEvent(
			"saga-123",
			"booking-saga",
			StepReserveSeats,
			0,
			data,
			startedAt,
			finishedAt,
		)

		if !event.Success {
			t.Error("expected Success to be true")
		}
		if event.SagaID != "saga-123" {
			t.Errorf("expected SagaID saga-123, got %s", event.SagaID)
		}
		if event.Duration <= 0 {
			t.Error("expected Duration to be positive")
		}
	})

	t.Run("NewSagaFailureEvent", func(t *testing.T) {
		startedAt := time.Now().Add(-1 * time.Second)
		finishedAt := time.Now()

		event := NewSagaFailureEvent(
			"saga-123",
			"booking-saga",
			StepProcessPayment,
			1,
			"payment declined",
			"PAYMENT_DECLINED",
			startedAt,
			finishedAt,
		)

		if event.Success {
			t.Error("expected Success to be false")
		}
		if event.ErrorMessage != "payment declined" {
			t.Errorf("expected ErrorMessage 'payment declined', got %s", event.ErrorMessage)
		}
		if event.ErrorCode != "PAYMENT_DECLINED" {
			t.Errorf("expected ErrorCode PAYMENT_DECLINED, got %s", event.ErrorCode)
		}
	})

	t.Run("SagaLifecycleEvents", func(t *testing.T) {
		data := map[string]interface{}{
			"booking_id": "booking-123",
		}

		// Started event
		startedEvent := NewSagaStartedEvent("saga-123", "booking-saga", data)
		if startedEvent.Status != "started" {
			t.Errorf("expected Status started, got %s", startedEvent.Status)
		}

		// Completed event
		completedEvent := NewSagaCompletedEvent("saga-123", "booking-saga", data, startedEvent.StartedAt)
		if completedEvent.Status != "completed" {
			t.Errorf("expected Status completed, got %s", completedEvent.Status)
		}
		if completedEvent.Duration <= 0 {
			t.Error("expected Duration to be positive")
		}

		// Failed event
		failedEvent := NewSagaFailedEvent("saga-123", "booking-saga", "step failed", startedEvent.StartedAt)
		if failedEvent.Status != "failed" {
			t.Errorf("expected Status failed, got %s", failedEvent.Status)
		}
		if failedEvent.ErrorMessage != "step failed" {
			t.Errorf("expected ErrorMessage 'step failed', got %s", failedEvent.ErrorMessage)
		}

		// Compensated event
		compensatedEvent := NewSagaCompensatedEvent("saga-123", "booking-saga", "compensation reason", startedEvent.StartedAt)
		if compensatedEvent.Status != "compensated" {
			t.Errorf("expected Status compensated, got %s", compensatedEvent.Status)
		}
	})

	t.Run("NewCompensationCommand", func(t *testing.T) {
		data := map[string]interface{}{
			"reservation_id": "res-123",
		}

		command := NewCompensationCommand(
			"saga-123",
			"booking-saga",
			StepReserveSeats,
			0,
			data,
			"payment failed",
		)

		if command.SagaID != "saga-123" {
			t.Errorf("expected SagaID saga-123, got %s", command.SagaID)
		}
		if command.Reason != "payment failed" {
			t.Errorf("expected Reason 'payment failed', got %s", command.Reason)
		}
		if command.OriginalStepData == nil {
			t.Error("expected OriginalStepData to be set")
		}
	})

	t.Run("TimeoutCheck", func(t *testing.T) {
		check := NewTimeoutCheck(
			"saga-123",
			"booking-saga",
			StepReserveSeats,
			0,
			time.Now().Add(30*time.Second),
			3,
		)

		if check.IsTimedOut() {
			t.Error("expected IsTimedOut to be false for future timeout")
		}

		// Create expired timeout
		expiredCheck := NewTimeoutCheck(
			"saga-456",
			"booking-saga",
			StepReserveSeats,
			0,
			time.Now().Add(-1*time.Second),
			3,
		)

		if !expiredCheck.IsTimedOut() {
			t.Error("expected IsTimedOut to be true for past timeout")
		}
	})
}

func TestMockSagaProducer(t *testing.T) {
	ctx := context.Background()
	producer := NewMockSagaProducer()

	t.Run("SendCommand", func(t *testing.T) {
		command := NewSagaCommand("saga-1", "booking-saga", StepReserveSeats, 0, nil, 30*time.Second, 3)
		err := producer.SendCommand(ctx, command)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(producer.Commands) != 1 {
			t.Errorf("expected 1 command, got %d", len(producer.Commands))
		}
	})

	t.Run("SendCompensationCommand", func(t *testing.T) {
		command := NewCompensationCommand("saga-1", "booking-saga", StepReserveSeats, 0, nil, "test")
		err := producer.SendCompensationCommand(ctx, command)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(producer.CompensationCommands) != 1 {
			t.Errorf("expected 1 compensation command, got %d", len(producer.CompensationCommands))
		}
	})

	t.Run("SendStepSuccessEvent", func(t *testing.T) {
		event := NewSagaSuccessEvent("saga-1", "booking-saga", StepReserveSeats, 0, nil, time.Now(), time.Now())
		err := producer.SendStepSuccessEvent(ctx, event)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(producer.SuccessEvents) != 1 {
			t.Errorf("expected 1 success event, got %d", len(producer.SuccessEvents))
		}
	})

	t.Run("SendStepFailureEvent", func(t *testing.T) {
		event := NewSagaFailureEvent("saga-1", "booking-saga", StepReserveSeats, 0, "error", "ERROR", time.Now(), time.Now())
		err := producer.SendStepFailureEvent(ctx, event)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(producer.FailureEvents) != 1 {
			t.Errorf("expected 1 failure event, got %d", len(producer.FailureEvents))
		}
	})

	t.Run("SendLifecycleEvents", func(t *testing.T) {
		producer.Clear()

		startedEvent := NewSagaStartedEvent("saga-1", "booking-saga", nil)
		_ = producer.SendSagaStartedEvent(ctx, startedEvent)

		completedEvent := NewSagaCompletedEvent("saga-1", "booking-saga", nil, time.Now())
		_ = producer.SendSagaCompletedEvent(ctx, completedEvent)

		failedEvent := NewSagaFailedEvent("saga-1", "booking-saga", "error", time.Now())
		_ = producer.SendSagaFailedEvent(ctx, failedEvent)

		compensatedEvent := NewSagaCompensatedEvent("saga-1", "booking-saga", "reason", time.Now())
		_ = producer.SendSagaCompensatedEvent(ctx, compensatedEvent)

		if len(producer.LifecycleEvents) != 4 {
			t.Errorf("expected 4 lifecycle events, got %d", len(producer.LifecycleEvents))
		}

		startedEvents := producer.GetLifecycleEventsByStatus("started")
		if len(startedEvents) != 1 {
			t.Errorf("expected 1 started event, got %d", len(startedEvents))
		}
	})

	t.Run("ScheduleTimeoutCheck", func(t *testing.T) {
		producer.Clear()

		check := NewTimeoutCheck("saga-1", "booking-saga", StepReserveSeats, 0, time.Now().Add(30*time.Second), 3)
		err := producer.ScheduleTimeoutCheck(ctx, check)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(producer.TimeoutChecks) != 1 {
			t.Errorf("expected 1 timeout check, got %d", len(producer.TimeoutChecks))
		}
	})

	t.Run("ShouldFail", func(t *testing.T) {
		producer.Clear()
		producer.ShouldFail = true

		command := NewSagaCommand("saga-1", "booking-saga", StepReserveSeats, 0, nil, 30*time.Second, 3)
		err := producer.SendCommand(ctx, command)
		if err == nil {
			t.Error("expected error when ShouldFail is true")
		}

		producer.ShouldFail = false
	})
}

func TestMockSagaConsumer(t *testing.T) {
	ctx := context.Background()
	consumer := NewMockSagaConsumer()

	t.Run("StartStop", func(t *testing.T) {
		err := consumer.Start(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		err = consumer.Stop()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("SimulateEvents", func(t *testing.T) {
		successEvent := NewSagaSuccessEvent("saga-1", "booking-saga", StepReserveSeats, 0, nil, time.Now(), time.Now())
		_ = consumer.SimulateSuccessEvent(ctx, successEvent)

		failureEvent := NewSagaFailureEvent("saga-1", "booking-saga", StepProcessPayment, 1, "error", "ERROR", time.Now(), time.Now())
		_ = consumer.SimulateFailureEvent(ctx, failureEvent)

		timeoutCheck := NewTimeoutCheck("saga-1", "booking-saga", StepReserveSeats, 0, time.Now(), 3)
		_ = consumer.SimulateTimeoutCheck(ctx, timeoutCheck)

		if len(consumer.GetSuccessEvents()) != 1 {
			t.Errorf("expected 1 success event, got %d", len(consumer.GetSuccessEvents()))
		}
		if len(consumer.GetFailureEvents()) != 1 {
			t.Errorf("expected 1 failure event, got %d", len(consumer.GetFailureEvents()))
		}
		if len(consumer.GetTimeoutChecks()) != 1 {
			t.Errorf("expected 1 timeout check, got %d", len(consumer.GetTimeoutChecks()))
		}

		consumer.Clear()
		if len(consumer.GetSuccessEvents()) != 0 {
			t.Error("expected 0 events after Clear")
		}
	})

	t.Run("WithHandler", func(t *testing.T) {
		handlerCalled := false
		handler := &mockEventHandler{
			onSuccess: func(ctx context.Context, event *SagaEvent) error {
				handlerCalled = true
				return nil
			},
		}

		consumer.SetHandler(handler)

		successEvent := NewSagaSuccessEvent("saga-1", "booking-saga", StepReserveSeats, 0, nil, time.Now(), time.Now())
		_ = consumer.SimulateSuccessEvent(ctx, successEvent)

		if !handlerCalled {
			t.Error("expected handler to be called")
		}
	})
}

func TestMockTimeoutHandler(t *testing.T) {
	ctx := context.Background()
	handler := NewMockTimeoutHandler()

	t.Run("StartStop", func(t *testing.T) {
		err := handler.Start(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		err = handler.Stop()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("RegisterAndCancelTimeout", func(t *testing.T) {
		check := NewTimeoutCheck("saga-1", "booking-saga", StepReserveSeats, 0, time.Now().Add(30*time.Second), 3)
		handler.RegisterTimeout(check)

		checks := handler.GetRegisteredChecks()
		if len(checks) != 1 {
			t.Errorf("expected 1 registered check, got %d", len(checks))
		}

		handler.CancelTimeout("saga-1", StepReserveSeats)
		if !handler.IsCancelled("saga-1", StepReserveSeats) {
			t.Error("expected timeout to be cancelled")
		}

		handler.Clear()
		if len(handler.GetRegisteredChecks()) != 0 {
			t.Error("expected 0 checks after Clear")
		}
	})
}

func TestMessageParsing(t *testing.T) {
	t.Run("ParseSagaCommand", func(t *testing.T) {
		original := NewSagaCommand("saga-123", "booking-saga", StepReserveSeats, 0, map[string]interface{}{"key": "value"}, 30*time.Second, 3)

		// Serialize
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Parse
		parsed, err := ParseSagaCommand(data)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		if parsed.SagaID != original.SagaID {
			t.Errorf("expected SagaID %s, got %s", original.SagaID, parsed.SagaID)
		}
		if parsed.StepName != original.StepName {
			t.Errorf("expected StepName %s, got %s", original.StepName, parsed.StepName)
		}
	})

	t.Run("ParseSagaEvent", func(t *testing.T) {
		original := NewSagaSuccessEvent("saga-123", "booking-saga", StepReserveSeats, 0, map[string]interface{}{"key": "value"}, time.Now(), time.Now())

		// Serialize
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Parse
		parsed, err := ParseSagaEvent(data)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		if parsed.SagaID != original.SagaID {
			t.Errorf("expected SagaID %s, got %s", original.SagaID, parsed.SagaID)
		}
		if parsed.Success != original.Success {
			t.Errorf("expected Success %t, got %t", original.Success, parsed.Success)
		}
	})

	t.Run("ParseCompensationCommand", func(t *testing.T) {
		original := NewCompensationCommand("saga-123", "booking-saga", StepReserveSeats, 0, map[string]interface{}{"key": "value"}, "reason")

		// Serialize
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Parse
		parsed, err := ParseCompensationCommand(data)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		if parsed.SagaID != original.SagaID {
			t.Errorf("expected SagaID %s, got %s", original.SagaID, parsed.SagaID)
		}
		if parsed.Reason != original.Reason {
			t.Errorf("expected Reason %s, got %s", original.Reason, parsed.Reason)
		}
	})

	t.Run("ParseInvalidJSON", func(t *testing.T) {
		invalidData := []byte("{invalid json}")

		_, err := ParseSagaCommand(invalidData)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}

		_, err = ParseSagaEvent(invalidData)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}

		_, err = ParseCompensationCommand(invalidData)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

// mockEventHandler is a helper for testing
type mockEventHandler struct {
	onSuccess func(ctx context.Context, event *SagaEvent) error
	onFailure func(ctx context.Context, event *SagaEvent) error
	onTimeout func(ctx context.Context, check *TimeoutCheck) error
}

func (h *mockEventHandler) HandleStepSuccess(ctx context.Context, event *SagaEvent) error {
	if h.onSuccess != nil {
		return h.onSuccess(ctx, event)
	}
	return nil
}

func (h *mockEventHandler) HandleStepFailure(ctx context.Context, event *SagaEvent) error {
	if h.onFailure != nil {
		return h.onFailure(ctx, event)
	}
	return nil
}

func (h *mockEventHandler) HandleTimeout(ctx context.Context, check *TimeoutCheck) error {
	if h.onTimeout != nil {
		return h.onTimeout(ctx, check)
	}
	return nil
}

// Verify mockEventHandler implements SagaEventHandler
var _ SagaEventHandler = (*mockEventHandler)(nil)
