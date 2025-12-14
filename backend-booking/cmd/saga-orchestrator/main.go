package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logCfg := &logger.Config{
		Level:       cfg.App.Environment,
		ServiceName: "saga-orchestrator",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Saga Orchestrator Worker...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenTelemetry tracing
	if cfg.OTel.Enabled {
		_, err := telemetry.Init(ctx, &telemetry.Config{
			Enabled:       true,
			ServiceName:   "saga-orchestrator",
			CollectorAddr: cfg.OTel.CollectorAddr,
			SampleRatio:   cfg.OTel.SampleRatio,
			Environment:   cfg.App.Environment,
		})
		if err != nil {
			appLog.Warn(fmt.Sprintf("Failed to initialize tracer (continuing without tracing): %v", err))
		} else {
			defer telemetry.Shutdown(ctx)
			appLog.Info("OpenTelemetry tracing initialized")
		}
	}

	// Initialize PostgreSQL connection for saga store (primary source of truth)
	dbCfg := &database.PostgresConfig{
		Host:          cfg.BookingDatabase.Host,
		Port:          cfg.BookingDatabase.Port,
		User:          cfg.BookingDatabase.User,
		Password:      cfg.BookingDatabase.Password,
		Database:      cfg.BookingDatabase.DBName,
		SSLMode:       cfg.BookingDatabase.SSLMode,
		MaxConns:      20,
		MinConns:      5,
		MaxRetries:    3,
		RetryInterval: 2 * time.Second,
	}
	db, err := database.NewPostgres(ctx, dbCfg)
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to connect to PostgreSQL: %v", err))
	}
	defer db.Close()
	appLog.Info("PostgreSQL connected")

	// Initialize saga store using PostgreSQL (durable state persistence)
	store := pkgsaga.NewPostgresStore(db.Pool())
	appLog.Info("Saga store initialized (PostgreSQL)")

	// Initialize Kafka producer
	producer, err := saga.NewKafkaSagaProducer(ctx, &saga.KafkaSagaProducerConfig{
		Brokers:       cfg.Kafka.Brokers,
		ClientID:      "saga-orchestrator-producer",
		MaxRetries:    3,
		RetryInterval: time.Second,
		Logger:        &saga.ZapLogger{},
	})
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to create Kafka producer: %v", err))
	}
	defer producer.Close()
	appLog.Info("Kafka producer connected")

	// Create saga orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store:  store,
		Logger: &saga.ZapLogger{},
	})

	// Register booking saga definition (legacy - for backward compatibility)
	sagaBuilder := saga.NewBookingSagaBuilder(&saga.BookingSagaConfig{
		StepTimeout: 30 * time.Second,
		MaxRetries:  2,
	})
	if err := orchestrator.RegisterDefinition(sagaBuilder.Build()); err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to register booking saga definition: %v", err))
	}
	appLog.Info("Booking saga definition registered (legacy)")

	// Register post-payment saga definition (new - triggered after payment success)
	postPaymentSagaBuilder := saga.NewPostPaymentSagaBuilder(&saga.PostPaymentSagaConfig{
		StepTimeout: 30 * time.Second,
		MaxRetries:  3,
	})
	if err := orchestrator.RegisterDefinition(postPaymentSagaBuilder.Build()); err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to register post-payment saga definition: %v", err))
	}
	appLog.Info("Post-payment saga definition registered")

	// Create event handler
	eventHandler := saga.NewOrchestratorEventHandler(orchestrator, producer, store)

	// Initialize Kafka consumer
	consumer, err := saga.NewSagaConsumer(ctx, &saga.SagaConsumerConfig{
		Brokers:          cfg.Kafka.Brokers,
		GroupID:          "saga-orchestrator",
		ClientID:         "saga-orchestrator-consumer",
		Orchestrator:     orchestrator,
		Store:            store,
		Producer:         producer,
		Logger:           &saga.ZapLogger{},
		Handler:          eventHandler,
		SessionTimeout:   30 * time.Second,
		RebalanceTimeout: 60 * time.Second,
	})
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to create Kafka consumer: %v", err))
	}
	defer consumer.Stop()
	appLog.Info("Kafka consumer connected")

	// Start saga event consumer
	go func() {
		if err := consumer.Start(ctx); err != nil {
			appLog.Error(fmt.Sprintf("Saga event consumer error: %v", err))
		}
	}()

	// Initialize payment success consumer (triggers post-payment saga)
	paymentConsumer, err := saga.NewPaymentSuccessConsumer(ctx, &saga.PaymentSuccessConsumerConfig{
		Brokers:          cfg.Kafka.Brokers,
		GroupID:          "saga-orchestrator-payment",
		ClientID:         "saga-orchestrator-payment-consumer",
		Store:            store,
		Producer:         producer,
		Logger:           &saga.ZapLogger{},
		SessionTimeout:   30 * time.Second,
		RebalanceTimeout: 60 * time.Second,
	})
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to create payment success consumer: %v", err))
	}
	defer paymentConsumer.Stop()
	appLog.Info("Payment success consumer connected (topic: payment.success)")

	// Start payment success consumer
	go func() {
		if err := paymentConsumer.Start(ctx); err != nil {
			appLog.Error(fmt.Sprintf("Payment success consumer error: %v", err))
		}
	}()

	appLog.Info("Saga Orchestrator Worker started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLog.Info("Shutting down worker...")
	cancel()

	time.Sleep(2 * time.Second)
	appLog.Info("Worker exited gracefully")
}
