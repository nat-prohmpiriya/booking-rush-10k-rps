package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	// TopicPaymentSuccess is the Kafka topic for payment success events
	TopicPaymentSuccess = "payment.success"
)

// PaymentSuccessEvent represents a payment success event from payment service
type PaymentSuccessEvent struct {
	EventType             string    `json:"event_type"`
	BookingID             string    `json:"booking_id"`
	PaymentID             string    `json:"payment_id"`
	StripePaymentIntentID string    `json:"stripe_payment_intent_id"`
	UserID                string    `json:"user_id,omitempty"`
	Amount                int64     `json:"amount"`
	Currency              string    `json:"currency"`
	Timestamp             time.Time `json:"timestamp"`
}

// PostPaymentSagaData contains data for the post-payment saga
type PostPaymentSagaData struct {
	BookingID             string    `json:"booking_id"`
	PaymentID             string    `json:"payment_id"`
	StripePaymentIntentID string    `json:"stripe_payment_intent_id"`
	UserID                string    `json:"user_id,omitempty"`
	Amount                int64     `json:"amount"`
	Currency              string    `json:"currency"`
	Timestamp             time.Time `json:"timestamp"`
}

// ToMap converts PostPaymentSagaData to map[string]interface{}
func (d *PostPaymentSagaData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"booking_id":              d.BookingID,
		"payment_id":              d.PaymentID,
		"stripe_payment_intent_id": d.StripePaymentIntentID,
		"user_id":                 d.UserID,
		"amount":                  d.Amount,
		"currency":                d.Currency,
		"timestamp":               d.Timestamp.Format(time.RFC3339),
	}
}

// PaymentSuccessConsumerConfig holds configuration for PaymentSuccessConsumer
type PaymentSuccessConsumerConfig struct {
	Brokers          []string
	GroupID          string
	ClientID         string
	Store            pkgsaga.Store
	Producer         SagaProducer
	Logger           pkgsaga.Logger
	SessionTimeout   time.Duration
	RebalanceTimeout time.Duration
}

// PaymentSuccessConsumer consumes payment.success events and starts post-payment sagas
type PaymentSuccessConsumer struct {
	config   *PaymentSuccessConsumerConfig
	client   *kgo.Client
	store    pkgsaga.Store
	producer SagaProducer
	logger   pkgsaga.Logger
	wg       sync.WaitGroup
	stopCh   chan struct{}
}

// NewPaymentSuccessConsumer creates a new PaymentSuccessConsumer
func NewPaymentSuccessConsumer(ctx context.Context, cfg *PaymentSuccessConsumerConfig) (*PaymentSuccessConsumer, error) {
	if cfg.Logger == nil {
		cfg.Logger = &ZapLogger{}
	}
	if cfg.SessionTimeout == 0 {
		cfg.SessionTimeout = 30 * time.Second
	}
	if cfg.RebalanceTimeout == 0 {
		cfg.RebalanceTimeout = 60 * time.Second
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(TopicPaymentSuccess),
		kgo.ClientID(cfg.ClientID),
		kgo.DisableAutoCommit(),
		kgo.SessionTimeout(cfg.SessionTimeout),
		kgo.RebalanceTimeout(cfg.RebalanceTimeout),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Kafka: %w", err)
	}

	return &PaymentSuccessConsumer{
		config:   cfg,
		client:   client,
		store:    cfg.Store,
		producer: cfg.Producer,
		logger:   cfg.Logger,
		stopCh:   make(chan struct{}),
	}, nil
}

// Start begins consuming payment success events
func (c *PaymentSuccessConsumer) Start(ctx context.Context) error {
	log := logger.Get()
	log.Info(fmt.Sprintf("PaymentSuccessConsumer started, listening to topic: %s", TopicPaymentSuccess))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		default:
		}

		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				c.logger.ErrorContext(ctx, fmt.Sprintf("Fetch error: topic=%s, partition=%d, err=%v",
					err.Topic, err.Partition, err.Err))
			}
			continue
		}

		fetches.EachRecord(func(record *kgo.Record) {
			c.wg.Add(1)
			go func(r *kgo.Record) {
				defer c.wg.Done()
				if err := c.processRecord(ctx, r); err != nil {
					c.logger.ErrorContext(ctx, fmt.Sprintf("Failed to process record: %v", err))
				}
			}(record)
		})

		// Commit after processing
		if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
			c.logger.ErrorContext(ctx, fmt.Sprintf("Failed to commit offsets: %v", err))
		}
	}
}

// Stop stops the consumer
func (c *PaymentSuccessConsumer) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	c.client.Close()
}

// processRecord processes a single payment success event
func (c *PaymentSuccessConsumer) processRecord(ctx context.Context, record *kgo.Record) error {
	log := logger.Get()

	var event PaymentSuccessEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal payment success event: %w", err)
	}

	log.Info(fmt.Sprintf("Received payment.success event: booking_id=%s, payment_id=%s",
		event.BookingID, event.PaymentID))

	// Create post-payment saga data
	sagaData := &PostPaymentSagaData{
		BookingID:             event.BookingID,
		PaymentID:             event.PaymentID,
		StripePaymentIntentID: event.StripePaymentIntentID,
		UserID:                event.UserID,
		Amount:                event.Amount,
		Currency:              event.Currency,
		Timestamp:             event.Timestamp,
	}

	// Start the post-payment saga
	sagaID, err := c.startPostPaymentSaga(ctx, sagaData)
	if err != nil {
		return fmt.Errorf("failed to start post-payment saga: %w", err)
	}

	log.Info(fmt.Sprintf("Started post-payment saga: saga_id=%s, booking_id=%s", sagaID, event.BookingID))
	return nil
}

// startPostPaymentSaga creates and starts a new post-payment saga instance
func (c *PaymentSuccessConsumer) startPostPaymentSaga(ctx context.Context, data *PostPaymentSagaData) (string, error) {
	// Create saga instance
	instance := pkgsaga.NewInstance(PostPaymentSagaName, data.ToMap())

	// Save to store
	if err := c.store.Save(ctx, instance); err != nil {
		return "", fmt.Errorf("failed to save saga instance: %w", err)
	}

	// Update status to running
	instance.Status = pkgsaga.StatusRunning
	instance.CurrentStep = 0 // First step (confirm-booking)
	if err := c.store.Update(ctx, instance); err != nil {
		return "", fmt.Errorf("failed to update saga status: %w", err)
	}

	// Send first command (confirm-booking)
	cmd := NewSagaCommand(
		instance.ID,         // sagaID
		PostPaymentSagaName, // sagaName
		StepConfirmBooking,  // stepName
		0,                   // stepIndex
		data.ToMap(),        // data
		30*time.Second,      // timeout
		3,                   // maxRetries
	)

	if err := c.producer.SendCommand(ctx, cmd); err != nil {
		return "", fmt.Errorf("failed to send confirm-booking command: %w", err)
	}

	return instance.ID, nil
}
