package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/gateway"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
)

const (
	TopicProcessPaymentCommand = "saga.booking.process-payment.command"
	TopicRefundPaymentCommand  = "saga.booking.refund-payment.command"
	TopicPaymentProcessedEvent = "saga.booking.payment-processed.event"
	TopicPaymentFailedEvent    = "saga.booking.payment-failed.event"
	TopicPaymentRefundedEvent  = "saga.booking.payment-refunded.event"
)

// SagaCommand represents a saga command message
type SagaCommand struct {
	MessageID      string                 `json:"message_id"`
	SagaID         string                 `json:"saga_id"`
	SagaName       string                 `json:"saga_name"`
	StepName       string                 `json:"step_name"`
	StepIndex      int                    `json:"step_index"`
	IdempotencyKey string                 `json:"idempotency_key"`
	Data           map[string]interface{} `json:"data"`
}

// CompensationCommand represents a compensation command
type CompensationCommand struct {
	MessageID        string                 `json:"message_id"`
	SagaID           string                 `json:"saga_id"`
	SagaName         string                 `json:"saga_name"`
	StepName         string                 `json:"step_name"`
	StepIndex        int                    `json:"step_index"`
	OriginalStepData map[string]interface{} `json:"original_step_data"`
	Reason           string                 `json:"reason"`
}

// SagaEvent represents a saga event message
type SagaEvent struct {
	MessageID    string                 `json:"message_id"`
	SagaID       string                 `json:"saga_id"`
	SagaName     string                 `json:"saga_name"`
	StepName     string                 `json:"step_name"`
	StepIndex    int                    `json:"step_index"`
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	FinishedAt   time.Time              `json:"finished_at"`
	Duration     time.Duration          `json:"duration_ms"`
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logCfg := &logger.Config{
		Level:       cfg.App.Environment,
		ServiceName: "saga-payment-worker",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Saga Payment Worker...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	dbCfg := &database.PostgresConfig{
		Host:          cfg.PaymentDatabase.Host,
		Port:          cfg.PaymentDatabase.Port,
		User:          cfg.PaymentDatabase.User,
		Password:      cfg.PaymentDatabase.Password,
		Database:      cfg.PaymentDatabase.DBName,
		SSLMode:       cfg.PaymentDatabase.SSLMode,
		MaxConns:      10,
		MinConns:      2,
		MaxRetries:    3,
		RetryInterval: 2 * time.Second,
	}
	db, err := database.NewPostgres(ctx, dbCfg)
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer db.Close()
	appLog.Info("Database connected")

	// Initialize payment gateway
	var paymentGateway gateway.PaymentGateway
	gatewayType := os.Getenv("PAYMENT_GATEWAY")
	if gatewayType == "stripe" {
		stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
		if stripeSecretKey != "" {
			paymentGateway, _ = gateway.NewPaymentGateway("stripe", &gateway.GatewayConfig{
				SecretKey:   stripeSecretKey,
				Environment: os.Getenv("STRIPE_ENVIRONMENT"),
			})
		}
	}
	if paymentGateway == nil {
		paymentGateway = gateway.NewMockGatewayWithConfig(0.95, 100)
		appLog.Info("Using mock payment gateway")
	}

	// Initialize payment repository and service
	paymentRepo := repository.NewPostgresPaymentRepository(db)
	paymentService := service.NewPaymentService(paymentRepo, paymentGateway, &service.PaymentServiceConfig{
		Currency: "THB",
	})

	// Initialize Kafka consumer
	consumerCfg := &kafka.ConsumerConfig{
		Brokers: cfg.Kafka.Brokers,
		GroupID: "saga-payment-worker",
		Topics: []string{
			TopicProcessPaymentCommand,
			TopicRefundPaymentCommand,
		},
		ClientID:       "saga-payment-worker",
		MaxRetries:     3,
		RetryInterval:  2 * time.Second,
		SessionTimeout: 30 * time.Second,
	}
	consumer, err := kafka.NewConsumer(ctx, consumerCfg)
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to create Kafka consumer: %v", err))
	}
	defer consumer.Close()
	appLog.Info("Kafka consumer connected")

	// Initialize Kafka producer
	producerCfg := &kafka.ProducerConfig{
		Brokers:  cfg.Kafka.Brokers,
		ClientID: "saga-payment-worker-producer",
	}
	producer, err := kafka.NewProducer(ctx, producerCfg)
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to create Kafka producer: %v", err))
	}
	defer producer.Close()
	appLog.Info("Kafka producer connected")

	// Start worker
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				records, err := consumer.Poll(ctx)
				if err != nil {
					appLog.Error(fmt.Sprintf("Failed to poll: %v", err))
					time.Sleep(time.Second)
					continue
				}

				for _, record := range records {
					processRecord(ctx, record, paymentService, producer, consumer, appLog)
				}
			}
		}
	}()

	appLog.Info("Saga Payment Worker started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLog.Info("Shutting down worker...")
	cancel()

	time.Sleep(2 * time.Second)
	appLog.Info("Worker exited gracefully")
}

func processRecord(ctx context.Context, record *kafka.Record, paymentService service.PaymentService, producer *kafka.Producer, consumer *kafka.Consumer, appLog *logger.Logger) {
	switch record.Topic {
	case TopicProcessPaymentCommand:
		handleProcessPayment(ctx, record, paymentService, producer, consumer, appLog)
	case TopicRefundPaymentCommand:
		handleRefundPayment(ctx, record, paymentService, producer, consumer, appLog)
	default:
		appLog.Warn(fmt.Sprintf("Unknown topic: %s", record.Topic))
		consumer.CommitRecords(ctx, []*kafka.Record{record})
	}
}

func handleProcessPayment(ctx context.Context, record *kafka.Record, paymentService service.PaymentService, producer *kafka.Producer, consumer *kafka.Consumer, appLog *logger.Logger) {
	startTime := time.Now()

	var command SagaCommand
	if err := json.Unmarshal(record.Value, &command); err != nil {
		appLog.Error(fmt.Sprintf("Failed to unmarshal command: %v", err))
		consumer.CommitRecords(ctx, []*kafka.Record{record})
		return
	}

	appLog.Info(fmt.Sprintf("Processing payment: saga_id=%s", command.SagaID))

	// Extract data
	bookingID := getString(command.Data, "booking_id")
	if bookingID == "" {
		// Fallback to reservation_id (returned by reserve-seats step)
		bookingID = getString(command.Data, "reservation_id")
	}
	userID := getString(command.Data, "user_id")
	tenantID := getString(command.Data, "tenant_id")
	totalPrice := getFloat(command.Data, "total_price")
	currency := getString(command.Data, "currency")
	if currency == "" {
		currency = "THB"
	}

	// Create payment
	payment, err := paymentService.CreatePayment(ctx, &service.CreatePaymentRequest{
		TenantID:  tenantID,
		BookingID: bookingID,
		UserID:    userID,
		Amount:    totalPrice,
		Currency:  currency,
		Method:    "credit_card",
	})

	var resultData map[string]interface{}
	var execErr error

	if err != nil {
		execErr = err
	} else {
		// Process payment
		processedPayment, err := paymentService.ProcessPayment(ctx, payment.ID)
		if err != nil {
			execErr = err
		} else if processedPayment.Status != "succeeded" {
			execErr = fmt.Errorf("payment failed: %s", processedPayment.ErrorMessage)
		} else {
			resultData = map[string]interface{}{
				"payment_id":   processedPayment.ID,
				"processed_at": time.Now().Format(time.RFC3339),
			}
		}
	}

	finishTime := time.Now()

	// Send result event
	var event SagaEvent
	var topic string

	if execErr != nil {
		topic = TopicPaymentFailedEvent
		event = SagaEvent{
			MessageID:    fmt.Sprintf("%d", time.Now().UnixNano()),
			SagaID:       command.SagaID,
			SagaName:     command.SagaName,
			StepName:     command.StepName,
			StepIndex:    command.StepIndex,
			Success:      false,
			ErrorMessage: execErr.Error(),
			ErrorCode:    "PAYMENT_FAILED",
			StartedAt:    startTime,
			FinishedAt:   finishTime,
			Duration:     finishTime.Sub(startTime),
		}
	} else {
		topic = TopicPaymentProcessedEvent
		event = SagaEvent{
			MessageID:  fmt.Sprintf("%d", time.Now().UnixNano()),
			SagaID:     command.SagaID,
			SagaName:   command.SagaName,
			StepName:   command.StepName,
			StepIndex:  command.StepIndex,
			Success:    true,
			Data:       resultData,
			StartedAt:  startTime,
			FinishedAt: finishTime,
			Duration:   finishTime.Sub(startTime),
		}
	}

	if err := producer.ProduceJSON(ctx, topic, command.SagaID, event, nil); err != nil {
		appLog.Error(fmt.Sprintf("Failed to send event: %v", err))
	}

	consumer.CommitRecords(ctx, []*kafka.Record{record})
}

func handleRefundPayment(ctx context.Context, record *kafka.Record, paymentService service.PaymentService, producer *kafka.Producer, consumer *kafka.Consumer, appLog *logger.Logger) {
	var command CompensationCommand
	if err := json.Unmarshal(record.Value, &command); err != nil {
		appLog.Error(fmt.Sprintf("Failed to unmarshal command: %v", err))
		consumer.CommitRecords(ctx, []*kafka.Record{record})
		return
	}

	appLog.Info(fmt.Sprintf("Processing refund: saga_id=%s", command.SagaID))

	paymentID := getString(command.OriginalStepData, "payment_id")
	if paymentID != "" {
		_, err := paymentService.RefundPayment(ctx, paymentID, command.Reason)
		if err != nil {
			appLog.Error(fmt.Sprintf("Failed to refund payment: %v", err))
		} else {
			appLog.Info(fmt.Sprintf("Payment refunded: payment_id=%s", paymentID))
		}
	}

	consumer.CommitRecords(ctx, []*kafka.Record{record})
}

func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(data map[string]interface{}, key string) float64 {
	if v, ok := data[key].(float64); ok {
		return v
	}
	return 0
}
