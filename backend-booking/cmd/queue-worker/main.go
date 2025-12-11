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
		ServiceName: "queue-release-worker",
		Development: cfg.IsDevelopment(),
	}
	if err := logger.Init(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	appLog := logger.Get()
	appLog.Info("Starting Queue Release Worker...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Create queue repository
	queueRepo := repository.NewRedisQueueRepository(redis)

	// Load queue Lua scripts
	if err := queueRepo.LoadScripts(ctx); err != nil {
		appLog.Warn(fmt.Sprintf("Failed to load queue scripts: %v", err))
	}

	// Get worker configuration from environment or use defaults
	defaultMaxConcurrent := getEnvInt("QUEUE_DEFAULT_MAX_CONCURRENT", 500)
	releaseInterval := getEnvDuration("QUEUE_RELEASE_INTERVAL", 1*time.Second)
	defaultQueuePassTTL := getEnvDuration("QUEUE_DEFAULT_PASS_TTL", 5*time.Minute)
	jwtSecret := getEnvString("QUEUE_JWT_SECRET", cfg.JWT.Secret)

	workerCfg := &worker.QueueReleaseWorkerConfig{
		DefaultMaxConcurrent: defaultMaxConcurrent,
		ReleaseInterval:      releaseInterval,
		DefaultQueuePassTTL:  defaultQueuePassTTL,
		JWTSecret:            jwtSecret,
	}

	appLog.Info(fmt.Sprintf("Worker configuration: DefaultMaxConcurrent=%d, ReleaseInterval=%v, DefaultQueuePassTTL=%v",
		workerCfg.DefaultMaxConcurrent, workerCfg.ReleaseInterval, workerCfg.DefaultQueuePassTTL))

	// Create and start queue release worker
	queueWorker := worker.NewQueueReleaseWorker(workerCfg, queueRepo, appLog)

	// Start worker in background
	go queueWorker.Start(ctx)
	appLog.Info("Queue release worker started")

	// Start metrics reporter in background
	go reportMetrics(ctx, queueWorker, appLog)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLog.Info("Shutting down queue release worker...")
	cancel()

	// Give worker time to finish
	time.Sleep(2 * time.Second)
	appLog.Info("Queue release worker stopped")
}

// reportMetrics periodically logs worker metrics
func reportMetrics(ctx context.Context, w *worker.QueueReleaseWorker, log *logger.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			totalReleased, lastReleaseTime, lastReleaseCount := w.GetMetrics()
			if totalReleased > 0 {
				log.Info(fmt.Sprintf("Metrics: Total released=%d, Last release=%d users at %v",
					totalReleased, lastReleaseCount, lastReleaseTime.Format(time.RFC3339)))
			}
		}
	}
}

// getEnvString gets a string environment variable with a default
func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt gets an integer environment variable with a default
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var i int
		if _, err := fmt.Sscanf(val, "%d", &i); err == nil {
			return i
		}
	}
	return defaultVal
}

// getEnvDuration gets a duration environment variable with a default
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
