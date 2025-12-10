package retry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestDefaultDLQConfig(t *testing.T) {
	config := DefaultDLQConfig()

	if config.TopicPrefix != "dlq." {
		t.Errorf("TopicPrefix = %s, want dlq.", config.TopicPrefix)
	}

	if config.TopicSuffix != ".dlq" {
		t.Errorf("TopicSuffix = %s, want .dlq", config.TopicSuffix)
	}

	if config.UsePrefix {
		t.Error("UsePrefix should be false by default")
	}

	if config.Source != "unknown" {
		t.Errorf("Source = %s, want unknown", config.Source)
	}
}

func TestDLQMessage_JSON(t *testing.T) {
	now := time.Now()
	msg := &DLQMessage{
		ID:            "msg-123",
		OriginalTopic: "booking-events",
		OriginalKey:   "book-456",
		Payload:       json.RawMessage(`{"test": "data"}`),
		Headers: map[string]string{
			"event_type": "booking.created",
		},
		Error:          "kafka connection failed",
		ErrorCode:      "KAFKA_ERR",
		Attempts:       3,
		FirstAttemptAt: now.Add(-5 * time.Minute),
		LastAttemptAt:  now,
		MovedToDLQAt:   now,
		Source:         "booking-service",
		Metadata: map[string]interface{}{
			"user_id": "user-789",
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal DLQMessage: %v", err)
	}

	// Test JSON unmarshaling
	var decoded DLQMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal DLQMessage: %v", err)
	}

	if decoded.ID != msg.ID {
		t.Errorf("ID = %s, want %s", decoded.ID, msg.ID)
	}

	if decoded.OriginalTopic != msg.OriginalTopic {
		t.Errorf("OriginalTopic = %s, want %s", decoded.OriginalTopic, msg.OriginalTopic)
	}

	if decoded.Error != msg.Error {
		t.Errorf("Error = %s, want %s", decoded.Error, msg.Error)
	}

	if decoded.Attempts != msg.Attempts {
		t.Errorf("Attempts = %d, want %d", decoded.Attempts, msg.Attempts)
	}
}

// MockKafkaPublisher is a mock Kafka publisher for testing
type MockKafkaPublisher struct {
	PublishedMessages []struct {
		Topic   string
		Key     string
		Data    interface{}
		Headers map[string]string
	}
	ShouldFail bool
}

func (m *MockKafkaPublisher) PublishJSON(ctx context.Context, topic string, key string, data interface{}, headers map[string]string) error {
	if m.ShouldFail {
		return errors.New("mock publish failed")
	}

	m.PublishedMessages = append(m.PublishedMessages, struct {
		Topic   string
		Key     string
		Data    interface{}
		Headers map[string]string
	}{
		Topic:   topic,
		Key:     key,
		Data:    data,
		Headers: headers,
	})

	return nil
}

func TestKafkaDLQPublisher_GetDLQTopic(t *testing.T) {
	tests := []struct {
		name          string
		originalTopic string
		usePrefix     bool
		prefix        string
		suffix        string
		expected      string
	}{
		{
			name:          "suffix mode",
			originalTopic: "booking-events",
			usePrefix:     false,
			suffix:        ".dlq",
			expected:      "booking-events.dlq",
		},
		{
			name:          "prefix mode",
			originalTopic: "booking-events",
			usePrefix:     true,
			prefix:        "dlq.",
			expected:      "dlq.booking-events",
		},
		{
			name:          "custom suffix",
			originalTopic: "payment-events",
			usePrefix:     false,
			suffix:        "-dead-letter",
			expected:      "payment-events-dead-letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DLQConfig{
				TopicPrefix: tt.prefix,
				TopicSuffix: tt.suffix,
				UsePrefix:   tt.usePrefix,
			}

			publisher := NewKafkaDLQPublisher(&MockKafkaPublisher{}, config)
			got := publisher.GetDLQTopic(tt.originalTopic)

			if got != tt.expected {
				t.Errorf("GetDLQTopic(%s) = %s, want %s", tt.originalTopic, got, tt.expected)
			}
		})
	}
}

func TestKafkaDLQPublisher_PublishToDLQ(t *testing.T) {
	mock := &MockKafkaPublisher{}
	config := &DLQConfig{
		TopicSuffix: ".dlq",
		UsePrefix:   false,
		Source:      "test-service",
	}

	publisher := NewKafkaDLQPublisher(mock, config)

	msg := &DLQMessage{
		ID:            "msg-123",
		OriginalTopic: "booking-events",
		OriginalKey:   "book-456",
		Payload:       json.RawMessage(`{"booking_id": "book-456"}`),
		Headers: map[string]string{
			"event_type": "booking.created",
		},
		Error:          "kafka connection failed",
		Attempts:       3,
		FirstAttemptAt: time.Now().Add(-1 * time.Minute),
		LastAttemptAt:  time.Now(),
	}

	err := publisher.PublishToDLQ(context.Background(), msg)
	if err != nil {
		t.Fatalf("PublishToDLQ failed: %v", err)
	}

	if len(mock.PublishedMessages) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mock.PublishedMessages))
	}

	published := mock.PublishedMessages[0]

	if published.Topic != "booking-events.dlq" {
		t.Errorf("Topic = %s, want booking-events.dlq", published.Topic)
	}

	if published.Key != "book-456" {
		t.Errorf("Key = %s, want book-456", published.Key)
	}

	// Check headers
	if published.Headers["original_topic"] != "booking-events" {
		t.Errorf("Header original_topic = %s, want booking-events", published.Headers["original_topic"])
	}

	if published.Headers["error"] != "kafka connection failed" {
		t.Errorf("Header error = %s, want 'kafka connection failed'", published.Headers["error"])
	}

	if published.Headers["attempts"] != "3" {
		t.Errorf("Header attempts = %s, want 3", published.Headers["attempts"])
	}

	if published.Headers["source"] != "test-service" {
		t.Errorf("Header source = %s, want test-service", published.Headers["source"])
	}

	// Check that MovedToDLQAt was set
	publishedMsg, ok := published.Data.(*DLQMessage)
	if !ok {
		t.Fatal("Published data is not a DLQMessage")
	}

	if publishedMsg.MovedToDLQAt.IsZero() {
		t.Error("MovedToDLQAt should be set")
	}

	if publishedMsg.Source != "test-service" {
		t.Errorf("Source = %s, want test-service", publishedMsg.Source)
	}
}

func TestKafkaDLQPublisher_PublishToDLQ_NilMessage(t *testing.T) {
	mock := &MockKafkaPublisher{}
	publisher := NewKafkaDLQPublisher(mock, nil)

	err := publisher.PublishToDLQ(context.Background(), nil)
	if err == nil {
		t.Error("Expected error for nil message")
	}
}

func TestKafkaDLQPublisher_PublishToDLQ_PublishFails(t *testing.T) {
	mock := &MockKafkaPublisher{ShouldFail: true}
	publisher := NewKafkaDLQPublisher(mock, nil)

	msg := &DLQMessage{
		ID:            "msg-123",
		OriginalTopic: "booking-events",
		OriginalKey:   "book-456",
		Error:         "test error",
	}

	err := publisher.PublishToDLQ(context.Background(), msg)
	if err == nil {
		t.Error("Expected error when publish fails")
	}
}

func TestNewKafkaDLQPublisher_WithNilConfig(t *testing.T) {
	mock := &MockKafkaPublisher{}
	publisher := NewKafkaDLQPublisher(mock, nil)

	if publisher.config == nil {
		t.Fatal("Config should not be nil")
	}

	if publisher.config.TopicSuffix != ".dlq" {
		t.Errorf("TopicSuffix = %s, want .dlq", publisher.config.TopicSuffix)
	}
}

func TestNoOpDLQPublisher(t *testing.T) {
	publisher := NewNoOpDLQPublisher()

	msg := &DLQMessage{
		ID:            "msg-123",
		OriginalTopic: "test-topic",
	}

	err := publisher.PublishToDLQ(context.Background(), msg)
	if err != nil {
		t.Errorf("NoOpDLQPublisher.PublishToDLQ should not return error, got %v", err)
	}

	topic := publisher.GetDLQTopic("test-topic")
	if topic != "test-topic.dlq" {
		t.Errorf("GetDLQTopic = %s, want test-topic.dlq", topic)
	}
}

func TestDLQHandler_ProcessWithDLQ_Success(t *testing.T) {
	mock := &MockKafkaPublisher{}
	dlqPublisher := NewKafkaDLQPublisher(mock, nil)

	handlerConfig := &DLQHandlerConfig{
		RetryConfig: &Config{
			MaxRetries:      3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			JitterFactor:    0,
		},
		Source: "test-service",
	}

	handler := NewDLQHandler(dlqPublisher, handlerConfig)

	msgCtx := &MessageContext{
		ID:      "msg-123",
		Topic:   "booking-events",
		Key:     "book-456",
		Payload: json.RawMessage(`{"test": "data"}`),
	}

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return nil
	}

	err := handler.ProcessWithDLQ(context.Background(), msgCtx, op)
	if err != nil {
		t.Errorf("ProcessWithDLQ failed: %v", err)
	}

	if attempts != 1 {
		t.Errorf("Operation called %d times, want 1", attempts)
	}

	// No DLQ message should be published
	if len(mock.PublishedMessages) != 0 {
		t.Errorf("Expected 0 DLQ messages, got %d", len(mock.PublishedMessages))
	}
}

func TestDLQHandler_ProcessWithDLQ_AllRetriesFail(t *testing.T) {
	mock := &MockKafkaPublisher{}
	dlqPublisher := NewKafkaDLQPublisher(mock, &DLQConfig{
		TopicSuffix: ".dlq",
		Source:      "test-service",
	})

	handlerConfig := &DLQHandlerConfig{
		RetryConfig: &Config{
			MaxRetries:      2,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			JitterFactor:    0,
		},
		Source: "test-service",
	}

	var dlqCallback *DLQMessage
	handlerConfig.OnDLQ = func(msg *DLQMessage) {
		dlqCallback = msg
	}

	handler := NewDLQHandler(dlqPublisher, handlerConfig)

	msgCtx := &MessageContext{
		ID:      "msg-123",
		Topic:   "booking-events",
		Key:     "book-456",
		Payload: json.RawMessage(`{"test": "data"}`),
		Headers: map[string]string{
			"event_type": "booking.created",
		},
	}

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return errors.New("persistent error")
	}

	err := handler.ProcessWithDLQ(context.Background(), msgCtx, op)
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("Expected ErrMaxRetriesExceeded, got %v", err)
	}

	// Initial + 2 retries = 3 total
	if attempts != 3 {
		t.Errorf("Operation called %d times, want 3", attempts)
	}

	// DLQ message should be published
	if len(mock.PublishedMessages) != 1 {
		t.Fatalf("Expected 1 DLQ message, got %d", len(mock.PublishedMessages))
	}

	published := mock.PublishedMessages[0]
	if published.Topic != "booking-events.dlq" {
		t.Errorf("DLQ topic = %s, want booking-events.dlq", published.Topic)
	}

	// Check callback was invoked
	if dlqCallback == nil {
		t.Error("OnDLQ callback was not invoked")
	} else {
		if dlqCallback.Attempts != 3 {
			t.Errorf("DLQ callback attempts = %d, want 3", dlqCallback.Attempts)
		}
	}
}

func TestDLQHandler_ProcessWithDLQ_PermanentError(t *testing.T) {
	mock := &MockKafkaPublisher{}
	dlqPublisher := NewKafkaDLQPublisher(mock, nil)

	handlerConfig := &DLQHandlerConfig{
		RetryConfig: &Config{
			MaxRetries:      5,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			JitterFactor:    0,
		},
		Source: "test-service",
	}

	handler := NewDLQHandler(dlqPublisher, handlerConfig)

	msgCtx := &MessageContext{
		ID:    "msg-123",
		Topic: "booking-events",
		Key:   "book-456",
	}

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return Permanent(errors.New("permanent error"))
	}

	err := handler.ProcessWithDLQ(context.Background(), msgCtx, op)
	if err == nil {
		t.Error("Expected error for permanent error")
	}

	// Only 1 attempt for permanent errors
	if attempts != 1 {
		t.Errorf("Operation called %d times, want 1", attempts)
	}

	// DLQ message should still be published for permanent errors
	if len(mock.PublishedMessages) != 1 {
		t.Errorf("Expected 1 DLQ message for permanent error, got %d", len(mock.PublishedMessages))
	}
}

func TestDefaultDLQHandlerConfig(t *testing.T) {
	config := DefaultDLQHandlerConfig()

	if config.RetryConfig == nil {
		t.Error("RetryConfig should not be nil")
	}

	if config.Source != "unknown" {
		t.Errorf("Source = %s, want unknown", config.Source)
	}
}

func TestNewDLQHandler_WithNilConfig(t *testing.T) {
	mock := &MockKafkaPublisher{}
	dlqPublisher := NewKafkaDLQPublisher(mock, nil)

	handler := NewDLQHandler(dlqPublisher, nil)
	if handler.config == nil {
		t.Error("Config should not be nil")
	}
}

func TestKafkaProducerAdapter(t *testing.T) {
	// Create a mock that implements PublishJSON
	mock := &mockPublishJSON{}

	adapter := &KafkaProducerAdapter{Producer: mock}

	err := adapter.PublishJSON(context.Background(), "test-topic", "key", map[string]string{"test": "data"}, nil)
	if err != nil {
		t.Errorf("PublishJSON failed: %v", err)
	}

	if mock.callCount != 1 {
		t.Errorf("Expected 1 call, got %d", mock.callCount)
	}
}

type mockPublishJSON struct {
	callCount int
}

func (m *mockPublishJSON) ProduceJSON(ctx context.Context, topic string, key string, data interface{}, headers map[string]string) error {
	m.callCount++
	return nil
}
