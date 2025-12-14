package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/worker"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/config"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
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
		ServiceName: "seat-release-worker",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Seat Release Worker...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection (uses BookingDatabase - Microservice pattern)
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

	// Initialize Redis connection
	redisCfg := &pkgredis.Config{
		Host:          cfg.Redis.Host,
		Port:          cfg.Redis.Port,
		Password:      cfg.Redis.Password,
		DB:            cfg.Redis.DB,
		PoolSize:      100,
		MinIdleConns:  20,
		MaxRetries:    3,
		RetryInterval: 2 * time.Second,
	}
	redis, err := pkgredis.NewClient(ctx, redisCfg)
	if err != nil {
		appLog.Fatal(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}
	defer redis.Close()
	appLog.Info("Redis connected")

	// Initialize Kafka consumer
	consumerCfg := &kafka.ConsumerConfig{
		Brokers:        cfg.Kafka.Brokers,
		GroupID:        "seat-release-worker",
		Topics:         []string{"payment.seat-release"},
		ClientID:       "seat-release-worker",
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

	// Initialize repositories
	bookingRepo := repository.NewPostgresBookingRepository(db.Pool())
	reservationRepo := repository.NewRedisReservationRepository(redis)

	// Pre-load Lua scripts into Redis
	if err := reservationRepo.LoadScripts(ctx); err != nil {
		appLog.Warn(fmt.Sprintf("Failed to pre-load Lua scripts: %v", err))
	} else {
		appLog.Info("Lua scripts pre-loaded into Redis")
	}

	// Create worker
	seatReleaseWorker := worker.NewSeatReleaseWorker(
		consumer,
		bookingRepo,
		reservationRepo,
		&worker.SeatReleaseWorkerConfig{
			WorkerCount:   5,
			RetryAttempts: 3,
			RetryDelay:    time.Second,
		},
	)

	// Start worker
	go func() {
		if err := seatReleaseWorker.Start(ctx); err != nil {
			appLog.Error(fmt.Sprintf("Worker error: %v", err))
		}
	}()

	appLog.Info("Seat Release Worker started successfully")

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLog.Info("Shutting down worker...")
	cancel()

	// Give worker time to finish processing
	time.Sleep(2 * time.Second)

	appLog.Info("Worker exited gracefully")
}
