package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
)

// SeatReleaseEvent represents the event received from payment service
type SeatReleaseEvent struct {
	EventType   string `json:"event_type"`
	BookingID   string `json:"booking_id"`
	PaymentID   string `json:"payment_id"`
	UserID      string `json:"user_id,omitempty"`
	Reason      string `json:"reason"`
	FailureCode string `json:"failure_code,omitempty"`
	Message     string `json:"message,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// SeatReleaseWorkerConfig contains configuration for the seat release worker
type SeatReleaseWorkerConfig struct {
	WorkerCount   int
	RetryAttempts int
	RetryDelay    time.Duration
}

// SeatReleaseWorker consumes seat release events and releases seats
type SeatReleaseWorker struct {
	consumer        *kafka.Consumer
	bookingRepo     repository.BookingRepository
	reservationRepo repository.ReservationRepository
	config          *SeatReleaseWorkerConfig
}

// NewSeatReleaseWorker creates a new seat release worker
func NewSeatReleaseWorker(
	consumer *kafka.Consumer,
	bookingRepo repository.BookingRepository,
	reservationRepo repository.ReservationRepository,
	config *SeatReleaseWorkerConfig,
) *SeatReleaseWorker {
	if config == nil {
		config = &SeatReleaseWorkerConfig{
			WorkerCount:   5,
			RetryAttempts: 3,
			RetryDelay:    time.Second,
		}
	}
	return &SeatReleaseWorker{
		consumer:        consumer,
		bookingRepo:     bookingRepo,
		reservationRepo: reservationRepo,
		config:          config,
	}
}

// Start starts the worker and begins consuming messages
func (w *SeatReleaseWorker) Start(ctx context.Context) error {
	log := logger.Get()
	log.Info(fmt.Sprintf("Starting seat release worker with %d workers", w.config.WorkerCount))

	recordsCh := make(chan *kafka.Record, w.config.WorkerCount*10)

	// Start worker goroutines
	for i := 0; i < w.config.WorkerCount; i++ {
		go w.worker(ctx, i, recordsCh)
	}

	// Poll for messages
	return w.poll(ctx, recordsCh)
}

// poll continuously polls for messages from Kafka
func (w *SeatReleaseWorker) poll(ctx context.Context, recordsCh chan<- *kafka.Record) error {
	log := logger.Get()

	for {
		select {
		case <-ctx.Done():
			close(recordsCh)
			return ctx.Err()
		default:
			records, err := w.consumer.Poll(ctx)
			if err != nil {
				log.Error(fmt.Sprintf("Failed to poll messages: %v", err))
				time.Sleep(time.Second)
				continue
			}

			for _, record := range records {
				select {
				case recordsCh <- record:
				case <-ctx.Done():
					close(recordsCh)
					return ctx.Err()
				}
			}
		}
	}
}

// worker processes messages from the channel
func (w *SeatReleaseWorker) worker(ctx context.Context, id int, recordsCh <-chan *kafka.Record) {
	log := logger.Get()
	log.Info(fmt.Sprintf("Worker %d started", id))

	for record := range recordsCh {
		if err := w.processRecord(ctx, record); err != nil {
			log.Error(fmt.Sprintf("Worker %d failed to process record: %v", id, err))
		}
	}

	log.Info(fmt.Sprintf("Worker %d stopped", id))
}

// processRecord processes a single Kafka record
func (w *SeatReleaseWorker) processRecord(ctx context.Context, record *kafka.Record) error {
	log := logger.Get()

	var event SeatReleaseEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		log.Error(fmt.Sprintf("Failed to unmarshal event: %v", err))
		// Commit the record to avoid reprocessing malformed messages
		return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	log.Info(fmt.Sprintf("Processing seat release: booking_id=%s, reason=%s", event.BookingID, event.Reason))

	// Release seats with retry
	var lastErr error
	for attempt := 0; attempt < w.config.RetryAttempts; attempt++ {
		if err := w.releaseSeats(ctx, &event); err != nil {
			lastErr = err
			log.Warn(fmt.Sprintf("Attempt %d failed to release seats for booking %s: %v", attempt+1, event.BookingID, err))
			time.Sleep(w.config.RetryDelay)
			continue
		}
		lastErr = nil
		break
	}

	if lastErr != nil {
		log.Error(fmt.Sprintf("Failed to release seats after %d attempts: booking_id=%s, error=%v", w.config.RetryAttempts, event.BookingID, lastErr))
		// Still commit to avoid infinite loop, but log the failure for manual investigation
	} else {
		log.Info(fmt.Sprintf("Successfully released seats: booking_id=%s", event.BookingID))
	}

	return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
}

// releaseSeats releases the seats for a booking
func (w *SeatReleaseWorker) releaseSeats(ctx context.Context, event *SeatReleaseEvent) error {
	log := logger.Get()

	// Get booking from database
	booking, err := w.bookingRepo.GetByID(ctx, event.BookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking: %w", err)
	}

	if booking == nil {
		log.Warn(fmt.Sprintf("Booking not found: %s", event.BookingID))
		return nil // Not an error, booking might have been already cancelled
	}

	// Check if booking is in a state that requires seat release
	if booking.Status != "pending" && booking.Status != "reserved" {
		log.Info(fmt.Sprintf("Booking %s already in status %s, skipping seat release", event.BookingID, booking.Status))
		return nil
	}

	// Release seats in Redis
	_, err = w.reservationRepo.ReleaseSeats(ctx, booking.ID, booking.UserID)
	if err != nil {
		return fmt.Errorf("failed to release seats in Redis: %w", err)
	}

	// Update booking status in database
	booking.Status = "cancelled"
	booking.UpdatedAt = time.Now()
	if err := w.bookingRepo.Update(ctx, booking); err != nil {
		// Log but don't fail - Redis is the source of truth for availability
		log.Error(fmt.Sprintf("Failed to update booking status in database: %v", err))
	}

	log.Info(fmt.Sprintf("Released %d seats for booking %s (zone=%s, show=%s)",
		booking.Quantity, booking.ID, booking.ZoneID, booking.ShowID))

	return nil
}
