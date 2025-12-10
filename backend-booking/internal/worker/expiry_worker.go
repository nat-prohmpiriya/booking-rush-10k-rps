package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
)

// ExpiryWorkerConfig contains configuration for the expiry worker
type ExpiryWorkerConfig struct {
	// ScanInterval is the interval between scanning for expired reservations
	ScanInterval time.Duration
	// BatchSize is the number of reservations to process in each scan
	BatchSize int
}

// DefaultExpiryWorkerConfig returns default configuration
func DefaultExpiryWorkerConfig() *ExpiryWorkerConfig {
	return &ExpiryWorkerConfig{
		ScanInterval: 5 * time.Second, // Scan every 5 seconds
		BatchSize:    100,
	}
}

// ExpiryWorker scans and expires stale reservations
type ExpiryWorker struct {
	bookingRepo       *repository.PostgresBookingRepository
	transactionalRepo *repository.TransactionalBookingRepository
	reservationRepo   *repository.RedisReservationRepository
	config            *ExpiryWorkerConfig
	log               *logger.Logger
	stopCh            chan struct{}
	wg                sync.WaitGroup
	mu                sync.Mutex
	running           bool

	// Stats
	totalExpired     int64
	totalReleased    int64
	lastScanTime     time.Time
	lastExpiredCount int
}

// NewExpiryWorker creates a new expiry worker
func NewExpiryWorker(
	bookingRepo *repository.PostgresBookingRepository,
	transactionalRepo *repository.TransactionalBookingRepository,
	reservationRepo *repository.RedisReservationRepository,
	config *ExpiryWorkerConfig,
) *ExpiryWorker {
	if config == nil {
		config = DefaultExpiryWorkerConfig()
	}

	return &ExpiryWorker{
		bookingRepo:       bookingRepo,
		transactionalRepo: transactionalRepo,
		reservationRepo:   reservationRepo,
		config:            config,
		log:               logger.Get(),
		stopCh:            make(chan struct{}),
	}
}

// Start starts the expiry worker
func (w *ExpiryWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("expiry worker already running")
	}
	w.running = true
	w.mu.Unlock()

	w.log.Info("Starting expiry worker")

	// Start scanner goroutine
	w.wg.Add(1)
	go w.scanExpiredReservations(ctx)

	return nil
}

// Stop stops the expiry worker
func (w *ExpiryWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	w.log.Info("Stopping expiry worker")
	close(w.stopCh)
	w.wg.Wait()
	w.log.Info("Expiry worker stopped")
}

// scanExpiredReservations periodically scans for expired reservations
func (w *ExpiryWorker) scanExpiredReservations(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.ScanInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processExpiredReservations(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processExpiredReservations(ctx)
		}
	}
}

// processExpiredReservations fetches and processes expired reservations
func (w *ExpiryWorker) processExpiredReservations(ctx context.Context) {
	w.lastScanTime = time.Now()

	// Fetch expired reservations from PostgreSQL
	expired, err := w.bookingRepo.GetExpiredReservations(ctx, w.config.BatchSize)
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to get expired reservations: %v", err))
		return
	}

	if len(expired) == 0 {
		return
	}

	w.log.Info(fmt.Sprintf("Found %d expired reservations to process", len(expired)))
	w.lastExpiredCount = len(expired)

	for _, booking := range expired {
		if err := w.expireBooking(ctx, booking); err != nil {
			w.log.Error(fmt.Sprintf("Failed to expire booking %s: %v", booking.ID, err))
			continue
		}
		w.totalExpired++
	}
}

// expireBooking expires a single booking
func (w *ExpiryWorker) expireBooking(ctx context.Context, booking *domain.Booking) error {
	// 1. Release seats back to Redis inventory
	releaseResult, err := w.reservationRepo.ReleaseSeats(ctx, booking.ID, booking.UserID)
	if err != nil {
		// Log error but continue - Redis reservation might have already expired
		w.log.Warn(fmt.Sprintf("Failed to release seats from Redis for booking %s: %v", booking.ID, err))
	} else if releaseResult.Success {
		w.totalReleased++
		w.log.Info(fmt.Sprintf("Released %d seats for booking %s, new availability: %d",
			booking.Quantity, booking.ID, releaseResult.AvailableSeats))
	} else if releaseResult.ErrorCode == "RESERVATION_NOT_FOUND" {
		// Redis reservation already expired via TTL - this is expected
		w.log.Debug(fmt.Sprintf("Redis reservation for booking %s already expired (TTL)", booking.ID))
	} else {
		w.log.Warn(fmt.Sprintf("Could not release seats for booking %s: %s - %s",
			booking.ID, releaseResult.ErrorCode, releaseResult.ErrorMessage))
	}

	// 2. Update booking status in PostgreSQL and create outbox event
	// Update booking status for outbox event
	booking.Status = domain.BookingStatusExpired
	booking.StatusReason = "Reservation TTL expired"
	booking.UpdatedAt = time.Now()

	if err := w.transactionalRepo.MarkAsExpiredWithOutbox(ctx, booking.ID, booking); err != nil {
		return fmt.Errorf("failed to mark booking as expired in DB: %w", err)
	}

	w.log.Info(fmt.Sprintf("Successfully expired booking %s (user: %s, event: %s, zone: %s, qty: %d)",
		booking.ID, booking.UserID, booking.EventID, booking.ZoneID, booking.Quantity))

	return nil
}

// GetStats returns worker statistics
func (w *ExpiryWorker) GetStats() *ExpiryWorkerStats {
	w.mu.Lock()
	defer w.mu.Unlock()

	return &ExpiryWorkerStats{
		IsRunning:        w.running,
		TotalExpired:     w.totalExpired,
		TotalReleased:    w.totalReleased,
		LastScanTime:     w.lastScanTime,
		LastExpiredCount: w.lastExpiredCount,
	}
}

// ExpiryWorkerStats contains worker statistics
type ExpiryWorkerStats struct {
	IsRunning        bool      `json:"is_running"`
	TotalExpired     int64     `json:"total_expired"`
	TotalReleased    int64     `json:"total_released"`
	LastScanTime     time.Time `json:"last_scan_time"`
	LastExpiredCount int       `json:"last_expired_count"`
}
