package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		ServiceName: "inventory-worker",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Inventory Sync Worker...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	dbCfg := &database.PostgresConfig{
		Host:          cfg.Database.Host,
		Port:          cfg.Database.Port,
		User:          cfg.Database.User,
		Password:      cfg.Database.Password,
		Database:      cfg.Database.DBName,
		SSLMode:       cfg.Database.SSLMode,
		MaxConns:      int32(cfg.Database.MaxOpenConns),
		MinConns:      int32(cfg.Database.MaxIdleConns),
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
		PoolSize:      cfg.Redis.PoolSize,
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
		GroupID:        "inventory-sync-worker",
		Topics:         []string{"booking-events"},
		ClientID:       "inventory-worker",
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

	// Create worker configuration
	workerCfg := &worker.InventoryWorkerConfig{
		BatchInterval:    5 * time.Second,
		MaxBatchSize:     1000,
		RebuildOnStartup: true,
	}

	// Create and start inventory worker
	inventoryWorker := worker.NewInventoryWorker(workerCfg, consumer, db, redis, appLog)

	// Rebuild Redis from DB on startup if enabled
	if workerCfg.RebuildOnStartup {
		appLog.Info("Rebuilding Redis inventory from database...")
		if err := inventoryWorker.RebuildRedisFromDB(ctx); err != nil {
			appLog.Error(fmt.Sprintf("Failed to rebuild Redis from DB: %v", err))
			// Continue anyway, sync will catch up
		} else {
			appLog.Info("Redis inventory rebuilt successfully")
		}
	}

	// Start worker
	go inventoryWorker.Start(ctx)
	appLog.Info("Inventory worker started")

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLog.Info("Shutting down inventory worker...")
	cancel()

	// Give worker time to finish
	time.Sleep(2 * time.Second)
	appLog.Info("Inventory worker stopped")
}
