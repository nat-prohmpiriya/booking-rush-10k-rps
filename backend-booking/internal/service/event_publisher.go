package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"go.uber.org/zap"
)

// EventPublisher defines the interface for publishing booking events
type EventPublisher interface {
	// PublishBookingCreated publishes a booking created event
	PublishBookingCreated(ctx context.Context, booking *domain.Booking) error

	// PublishBookingConfirmed publishes a booking confirmed event
	PublishBookingConfirmed(ctx context.Context, booking *domain.Booking) error

	// PublishBookingCancelled publishes a booking cancelled event
	PublishBookingCancelled(ctx context.Context, booking *domain.Booking) error

	// PublishBookingExpired publishes a booking expired event
	PublishBookingExpired(ctx context.Context, booking *domain.Booking) error

	// Close closes the event publisher
	Close() error
}

// KafkaEventPublisher implements EventPublisher using Kafka
type KafkaEventPublisher struct {
	producer    *kafka.Producer
	topic       string
	serviceName string
	logger      Logger
}

// Logger interface for event publisher
type Logger interface {
	Error(msg string)
	Warn(msg string)
}

// EventPublisherConfig contains configuration for the event publisher
type EventPublisherConfig struct {
	Brokers     []string
	Topic       string
	ServiceName string
	ClientID    string
	Logger      Logger
}

// NewKafkaEventPublisher creates a new Kafka event publisher
func NewKafkaEventPublisher(ctx context.Context, cfg *EventPublisherConfig) (*KafkaEventPublisher, error) {
	if cfg == nil {
		return nil, fmt.Errorf("event publisher config is required")
	}

	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}

	topic := cfg.Topic
	if topic == "" {
		topic = "booking-events"
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "booking-service"
	}

	clientID := cfg.ClientID
	if clientID == "" {
		clientID = "booking-service-producer"
	}

	producer, err := kafka.NewProducer(ctx, &kafka.ProducerConfig{
		Brokers:       cfg.Brokers,
		ClientID:      clientID,
		MaxRetries:    3,
		RetryInterval: 2 * time.Second,
		BatchSize:     100,
		LingerMs:      10,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return &KafkaEventPublisher{
		producer:    producer,
		topic:       topic,
		serviceName: serviceName,
		logger:      cfg.Logger,
	}, nil
}

// PublishBookingCreated publishes a booking created event
func (p *KafkaEventPublisher) PublishBookingCreated(ctx context.Context, booking *domain.Booking) error {
	return p.publishEvent(ctx, domain.BookingEventCreated, booking)
}

// PublishBookingConfirmed publishes a booking confirmed event
func (p *KafkaEventPublisher) PublishBookingConfirmed(ctx context.Context, booking *domain.Booking) error {
	return p.publishEvent(ctx, domain.BookingEventConfirmed, booking)
}

// PublishBookingCancelled publishes a booking cancelled event
func (p *KafkaEventPublisher) PublishBookingCancelled(ctx context.Context, booking *domain.Booking) error {
	return p.publishEvent(ctx, domain.BookingEventCancelled, booking)
}

// PublishBookingExpired publishes a booking expired event
func (p *KafkaEventPublisher) PublishBookingExpired(ctx context.Context, booking *domain.Booking) error {
	return p.publishEvent(ctx, domain.BookingEventExpired, booking)
}

// Close closes the event publisher
func (p *KafkaEventPublisher) Close() error {
	if p.producer != nil {
		p.producer.Close()
	}
	return nil
}

// publishEvent publishes a booking event to Kafka asynchronously (fire-and-forget with logging)
func (p *KafkaEventPublisher) publishEvent(ctx context.Context, eventType domain.BookingEventType, booking *domain.Booking) error {
	eventID := uuid.New().String()
	event := domain.NewBookingEvent(eventType, booking, eventID)

	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	headers := map[string]string{
		"event_type":   string(eventType),
		"event_id":     eventID,
		"source":       p.serviceName,
		"content_type": "application/json",
	}

	msg := &kafka.Message{
		Topic:     p.topic,
		Key:       []byte(event.Key()),
		Value:     value,
		Headers:   headers,
		Timestamp: time.Now(),
	}

	// Use ProduceAsync with background context to avoid blocking the booking request
	// and prevent "context canceled" errors when HTTP request completes before publish
	// Error handling via callback - log but don't fail the request
	p.producer.ProduceAsync(context.Background(), msg, func(err error) {
		if err != nil && p.logger != nil {
			p.logger.Error(fmt.Sprintf("failed to publish %s event for booking %s: %v", eventType, booking.ID, err))
		}
	})

	return nil
}

// ZapLogger is the interface that pkg/logger.Logger implements
type ZapLogger interface {
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
}

// ZapLoggerAdapter adapts zap logger to Logger interface
type ZapLoggerAdapter struct {
	logger ZapLogger
}

// NewZapLoggerAdapter creates a new ZapLoggerAdapter
func NewZapLoggerAdapter(l ZapLogger) *ZapLoggerAdapter {
	return &ZapLoggerAdapter{logger: l}
}

// Error logs an error message
func (a *ZapLoggerAdapter) Error(msg string) {
	if a.logger != nil {
		a.logger.Error(msg)
	}
}

// Warn logs a warning message
func (a *ZapLoggerAdapter) Warn(msg string) {
	if a.logger != nil {
		a.logger.Warn(msg)
	}
}

// NoOpEventPublisher is a no-op implementation of EventPublisher for testing
type NoOpEventPublisher struct{}

// NewNoOpEventPublisher creates a new no-op event publisher
func NewNoOpEventPublisher() *NoOpEventPublisher {
	return &NoOpEventPublisher{}
}

// PublishBookingCreated is a no-op
func (p *NoOpEventPublisher) PublishBookingCreated(ctx context.Context, booking *domain.Booking) error {
	return nil
}

// PublishBookingConfirmed is a no-op
func (p *NoOpEventPublisher) PublishBookingConfirmed(ctx context.Context, booking *domain.Booking) error {
	return nil
}

// PublishBookingCancelled is a no-op
func (p *NoOpEventPublisher) PublishBookingCancelled(ctx context.Context, booking *domain.Booking) error {
	return nil
}

// PublishBookingExpired is a no-op
func (p *NoOpEventPublisher) PublishBookingExpired(ctx context.Context, booking *domain.Booking) error {
	return nil
}

// Close is a no-op
func (p *NoOpEventPublisher) Close() error {
	return nil
}
