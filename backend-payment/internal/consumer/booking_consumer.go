package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
)

// BookingConsumer consumes booking events from Kafka
type BookingConsumer struct {
	consumer       *kafka.Consumer
	producer       *kafka.Producer
	paymentService service.PaymentService
	logger         *logger.Logger
	config         *BookingConsumerConfig
	wg             sync.WaitGroup
	stopCh         chan struct{}
	mu             sync.RWMutex
	running        bool
}

// BookingConsumerConfig contains configuration for the booking consumer
type BookingConsumerConfig struct {
	Brokers        []string
	GroupID        string
	Topic          string
	PaymentTopic   string
	MaxRetries     int
	RetryInterval  time.Duration
	ProcessTimeout time.Duration
	WorkerCount    int
}

// DefaultBookingConsumerConfig returns default configuration
func DefaultBookingConsumerConfig() *BookingConsumerConfig {
	return &BookingConsumerConfig{
		Brokers:        []string{"localhost:9092"},
		GroupID:        "payment-service",
		Topic:          "booking-events",
		PaymentTopic:   "payment-events",
		MaxRetries:     3,
		RetryInterval:  2 * time.Second,
		ProcessTimeout: 30 * time.Second,
		WorkerCount:    10,
	}
}

// NewBookingConsumer creates a new booking consumer
func NewBookingConsumer(
	ctx context.Context,
	cfg *BookingConsumerConfig,
	paymentService service.PaymentService,
	log *logger.Logger,
) (*BookingConsumer, error) {
	if cfg == nil {
		cfg = DefaultBookingConsumerConfig()
	}

	// Create Kafka consumer
	consumerCfg := &kafka.ConsumerConfig{
		Brokers:       cfg.Brokers,
		GroupID:       cfg.GroupID,
		Topics:        []string{cfg.Topic},
		ClientID:      "payment-service-consumer",
		MaxRetries:    cfg.MaxRetries,
		RetryInterval: cfg.RetryInterval,
	}

	consumer, err := kafka.NewConsumer(ctx, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	// Create Kafka producer for payment events
	producerCfg := &kafka.ProducerConfig{
		Brokers:       cfg.Brokers,
		ClientID:      "payment-service-producer",
		MaxRetries:    cfg.MaxRetries,
		RetryInterval: cfg.RetryInterval,
	}

	producer, err := kafka.NewProducer(ctx, producerCfg)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return &BookingConsumer{
		consumer:       consumer,
		producer:       producer,
		paymentService: paymentService,
		logger:         log,
		config:         cfg,
		stopCh:         make(chan struct{}),
	}, nil
}

// Start starts the consumer
func (c *BookingConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.running = true
	c.mu.Unlock()

	c.logger.Info("Starting booking consumer...")

	// Start worker goroutines
	recordsCh := make(chan *kafka.Record, c.config.WorkerCount*10)

	// Start workers
	for i := 0; i < c.config.WorkerCount; i++ {
		c.wg.Add(1)
		go c.worker(ctx, i, recordsCh)
	}

	// Start polling goroutine
	c.wg.Add(1)
	go c.poll(ctx, recordsCh)

	return nil
}

// poll continuously polls for new records
func (c *BookingConsumer) poll(ctx context.Context, recordsCh chan<- *kafka.Record) {
	defer c.wg.Done()
	defer close(recordsCh)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer context cancelled, stopping poll...")
			return
		case <-c.stopCh:
			c.logger.Info("Consumer stop signal received, stopping poll...")
			return
		default:
			records, err := c.consumer.Poll(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error(fmt.Sprintf("Failed to poll records: %v", err))
				time.Sleep(time.Second)
				continue
			}

			for _, record := range records {
				select {
				case recordsCh <- record:
				case <-ctx.Done():
					return
				case <-c.stopCh:
					return
				}
			}
		}
	}
}

// worker processes records from the channel
func (c *BookingConsumer) worker(ctx context.Context, id int, recordsCh <-chan *kafka.Record) {
	defer c.wg.Done()

	c.logger.Info(fmt.Sprintf("Worker %d started", id))

	for record := range recordsCh {
		if err := c.processRecord(ctx, record); err != nil {
			c.logger.Error(fmt.Sprintf("Worker %d failed to process record: %v", id, err))
		}
	}

	c.logger.Info(fmt.Sprintf("Worker %d stopped", id))
}

// processRecord processes a single Kafka record
func (c *BookingConsumer) processRecord(ctx context.Context, record *kafka.Record) error {
	// Parse booking event
	var event BookingEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to unmarshal booking event: %v", err))
		// Commit the record anyway to avoid reprocessing invalid messages
		return c.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	c.logger.Info(fmt.Sprintf("Received booking event: type=%s, booking_id=%s",
		event.EventType, event.BookingData.BookingID))

	// Only process booking.created events
	if event.EventType != BookingEventCreated {
		c.logger.Info(fmt.Sprintf("Skipping event type: %s", event.EventType))
		return c.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	// Process the booking event
	if err := c.handleBookingCreated(ctx, &event); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to handle booking.created event: %v", err))
		// Don't commit on error - let it be reprocessed
		return err
	}

	// Commit the record
	return c.consumer.CommitRecords(ctx, []*kafka.Record{record})
}

// handleBookingCreated handles a booking.created event
func (c *BookingConsumer) handleBookingCreated(ctx context.Context, event *BookingEvent) error {
	data := event.BookingData
	if data == nil {
		return fmt.Errorf("booking data is nil")
	}

	c.logger.Info(fmt.Sprintf("Processing booking.created: booking_id=%s, user_id=%s, amount=%.2f %s",
		data.BookingID, data.UserID, data.TotalPrice, data.Currency))

	// Create payment request
	paymentReq := &service.CreatePaymentRequest{
		BookingID: data.BookingID,
		UserID:    data.UserID,
		Amount:    data.TotalPrice,
		Currency:  data.Currency,
		Method:    domain.PaymentMethodCreditCard, // Default to credit card
		Metadata: map[string]string{
			"event_id":   data.EventID,
			"zone_id":    data.ZoneID,
			"quantity":   fmt.Sprintf("%d", data.Quantity),
			"unit_price": fmt.Sprintf("%.2f", data.UnitPrice),
		},
	}

	// Create and process payment
	payment, err := c.paymentService.CreatePayment(ctx, paymentReq)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to create payment: %v", err))
		// Publish payment.failed event
		return c.publishPaymentEvent(ctx, PaymentEventFailed, nil, data.BookingID, data.UserID, err.Error())
	}

	// Process the payment
	processedPayment, err := c.paymentService.ProcessPayment(ctx, payment.ID)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to process payment: %v", err))
		// Publish payment.failed event
		return c.publishPaymentEvent(ctx, PaymentEventFailed, payment, data.BookingID, data.UserID, err.Error())
	}

	// Publish payment result event
	if processedPayment.Status == domain.PaymentStatusSucceeded {
		c.logger.Info(fmt.Sprintf("Payment successful: payment_id=%s, gateway_payment_id=%s",
			processedPayment.ID, processedPayment.GatewayPaymentID))
		return c.publishPaymentEvent(ctx, PaymentEventSuccess, processedPayment, data.BookingID, data.UserID, "")
	} else {
		c.logger.Info(fmt.Sprintf("Payment failed: payment_id=%s, reason=%s",
			processedPayment.ID, processedPayment.ErrorMessage))
		return c.publishPaymentEvent(ctx, PaymentEventFailed, processedPayment, data.BookingID, data.UserID, processedPayment.ErrorMessage)
	}
}

// publishPaymentEvent publishes a payment event to Kafka
func (c *BookingConsumer) publishPaymentEvent(
	ctx context.Context,
	eventType PaymentEventType,
	payment *domain.Payment,
	bookingID, userID, errorMessage string,
) error {
	eventData := &PaymentEventData{
		BookingID:    bookingID,
		UserID:       userID,
		ProcessedAt:  time.Now(),
		ErrorMessage: errorMessage,
	}

	if payment != nil {
		eventData.PaymentID = payment.ID
		eventData.Amount = payment.Amount
		eventData.Currency = payment.Currency
		eventData.Status = string(payment.Status)
		eventData.Method = string(payment.Method)
		eventData.GatewayPaymentID = payment.GatewayPaymentID
		eventData.ErrorCode = payment.ErrorCode
		if payment.ErrorMessage != "" {
			eventData.ErrorMessage = payment.ErrorMessage
		}
	}

	event := &PaymentEvent{
		EventID:     uuid.New().String(),
		EventType:   eventType,
		OccurredAt:  time.Now(),
		Version:     1,
		PaymentData: eventData,
	}

	headers := map[string]string{
		"event_type": string(eventType),
		"source":     "payment-service",
	}

	if err := c.producer.ProduceJSON(ctx, c.config.PaymentTopic, event.Key(), event, headers); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to publish payment event: %v", err))
		return err
	}

	c.logger.Info(fmt.Sprintf("Published payment event: type=%s, booking_id=%s", eventType, bookingID))
	return nil
}

// Stop stops the consumer
func (c *BookingConsumer) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	c.mu.Unlock()

	c.logger.Info("Stopping booking consumer...")

	// Signal stop
	close(c.stopCh)

	// Wait for goroutines to finish
	c.wg.Wait()

	// Close connections
	c.consumer.Close()
	c.producer.Close()

	c.logger.Info("Booking consumer stopped")
	return nil
}

// IsRunning returns whether the consumer is running
func (c *BookingConsumer) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}
