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

	// Initialize database connection for saga store
	dbCfg := &database.PostgresConfig{
		Host:          cfg.BookingDatabase.Host,
		Port:          cfg.BookingDatabase.Port,
		User:          cfg.BookingDatabase.User,
		Password:      cfg.BookingDatabase.Password,
		Database:      cfg.BookingDatabase.DBName,
		SSLMode:       cfg.BookingDatabase.SSLMode,
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

	// Initialize saga store (using memory store for now, can switch to postgres later)
	store := pkgsaga.NewMemoryStore()
	appLog.Info("Saga store initialized")

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

	// Register booking saga definition
	sagaBuilder := saga.NewBookingSagaBuilder(&saga.BookingSagaConfig{
		StepTimeout: 30 * time.Second,
		MaxRetries:  2,
	})
	if err := orchestrator.RegisterDefinition(sagaBuilder.Build()); err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to register saga definition: %v", err))
	}
	appLog.Info("Booking saga definition registered")

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

	// Start consumer
	go func() {
		if err := consumer.Start(ctx); err != nil {
			appLog.Error(fmt.Sprintf("Consumer error: %v", err))
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
