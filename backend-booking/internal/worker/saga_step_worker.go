package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
)

// SagaStepWorkerConfig contains configuration for the saga step worker
type SagaStepWorkerConfig struct {
	WorkerCount   int
	RetryAttempts int
	RetryDelay    time.Duration
}

// SagaStepWorker consumes saga commands and executes steps
type SagaStepWorker struct {
	consumer        *kafka.Consumer
	producer        saga.SagaProducer
	bookingRepo     repository.BookingRepository
	reservationRepo repository.ReservationRepository
	config          *SagaStepWorkerConfig
}

// NewSagaStepWorker creates a new saga step worker
func NewSagaStepWorker(
	consumer *kafka.Consumer,
	producer saga.SagaProducer,
	bookingRepo repository.BookingRepository,
	reservationRepo repository.ReservationRepository,
	config *SagaStepWorkerConfig,
) *SagaStepWorker {
	if config == nil {
		config = &SagaStepWorkerConfig{
			WorkerCount:   5,
			RetryAttempts: 3,
			RetryDelay:    time.Second,
		}
	}
	return &SagaStepWorker{
		consumer:        consumer,
		producer:        producer,
		bookingRepo:     bookingRepo,
		reservationRepo: reservationRepo,
		config:          config,
	}
}

// Start starts the worker
func (w *SagaStepWorker) Start(ctx context.Context) error {
	log := logger.Get()
	log.Info(fmt.Sprintf("Starting saga step worker with %d workers", w.config.WorkerCount))

	recordsCh := make(chan *kafka.Record, w.config.WorkerCount*10)

	// Start worker goroutines
	for i := 0; i < w.config.WorkerCount; i++ {
		go w.worker(ctx, i, recordsCh)
	}

	// Poll for messages
	return w.poll(ctx, recordsCh)
}

func (w *SagaStepWorker) poll(ctx context.Context, recordsCh chan<- *kafka.Record) error {
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

func (w *SagaStepWorker) worker(ctx context.Context, id int, recordsCh <-chan *kafka.Record) {
	log := logger.Get()
	log.Info(fmt.Sprintf("Worker %d started", id))

	for record := range recordsCh {
		if err := w.processRecord(ctx, record); err != nil {
			log.Error(fmt.Sprintf("Worker %d failed to process record: %v", id, err))
		}
	}

	log.Info(fmt.Sprintf("Worker %d stopped", id))
}

func (w *SagaStepWorker) processRecord(ctx context.Context, record *kafka.Record) error {
	log := logger.Get()

	// Determine message type from topic
	topic := record.Topic

	switch topic {
	case saga.TopicSagaReserveSeatsCommand:
		return w.handleReserveSeats(ctx, record)
	case saga.TopicSagaReleaseSeatsCommand:
		return w.handleReleaseSeats(ctx, record)
	case saga.TopicSagaConfirmBookingCommand:
		return w.handleConfirmBooking(ctx, record)
	default:
		log.Warn(fmt.Sprintf("Unknown topic: %s", topic))
		return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}
}

// handleReserveSeats handles the reserve-seats step
func (w *SagaStepWorker) handleReserveSeats(ctx context.Context, record *kafka.Record) error {
	log := logger.Get()
	startTime := time.Now()

	var command saga.SagaCommand
	if err := json.Unmarshal(record.Value, &command); err != nil {
		log.Error(fmt.Sprintf("Failed to unmarshal command: %v", err))
		return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	log.Info(fmt.Sprintf("Processing reserve-seats: saga_id=%s", command.SagaID))

	// Extract data
	data := &saga.BookingSagaData{}
	data.FromMap(command.Data)

	// Execute reservation
	var resultData map[string]interface{}
	var execErr error

	params := repository.ReserveParams{
		ZoneID:     data.ZoneID,
		UserID:     data.UserID,
		EventID:    data.EventID,
		Quantity:   data.Quantity,
		MaxPerUser: 10,
		TTLSeconds: 600, // 10 minutes
		Price:      data.TotalPrice / float64(data.Quantity),
	}

	for attempt := 0; attempt < w.config.RetryAttempts; attempt++ {
		result, err := w.reservationRepo.ReserveSeats(ctx, params)
		if err != nil {
			execErr = err
			time.Sleep(w.config.RetryDelay)
			continue
		}

		resultData = map[string]interface{}{
			"reservation_id": result.BookingID,
			"reserved_at":    time.Now().Format(time.RFC3339),
		}
		execErr = nil
		break
	}

	finishTime := time.Now()

	// Send result event
	if execErr != nil {
		event := saga.NewSagaFailureEvent(
			command.SagaID,
			command.SagaName,
			command.StepName,
			command.StepIndex,
			execErr.Error(),
			"RESERVATION_FAILED",
			startTime,
			finishTime,
		)
		if err := w.producer.SendStepFailureEvent(ctx, event); err != nil {
			log.Error(fmt.Sprintf("Failed to send failure event: %v", err))
		}
	} else {
		event := saga.NewSagaSuccessEvent(
			command.SagaID,
			command.SagaName,
			command.StepName,
			command.StepIndex,
			resultData,
			startTime,
			finishTime,
		)
		if err := w.producer.SendStepSuccessEvent(ctx, event); err != nil {
			log.Error(fmt.Sprintf("Failed to send success event: %v", err))
		}
	}

	return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
}

// handleReleaseSeats handles the release-seats compensation step
func (w *SagaStepWorker) handleReleaseSeats(ctx context.Context, record *kafka.Record) error {
	log := logger.Get()

	var command saga.CompensationCommand
	if err := json.Unmarshal(record.Value, &command); err != nil {
		log.Error(fmt.Sprintf("Failed to unmarshal compensation command: %v", err))
		return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	log.Info(fmt.Sprintf("Processing release-seats compensation: saga_id=%s", command.SagaID))

	// Extract data
	data := &saga.BookingSagaData{}
	data.FromMap(command.OriginalStepData)

	// Execute release
	_, err := w.reservationRepo.ReleaseSeats(ctx, data.BookingID, data.UserID)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to release seats: %v", err))
	} else {
		log.Info(fmt.Sprintf("Released seats: booking_id=%s", data.BookingID))
	}

	return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
}

// handleConfirmBooking handles the confirm-booking step
func (w *SagaStepWorker) handleConfirmBooking(ctx context.Context, record *kafka.Record) error {
	log := logger.Get()
	startTime := time.Now()

	var command saga.SagaCommand
	if err := json.Unmarshal(record.Value, &command); err != nil {
		log.Error(fmt.Sprintf("Failed to unmarshal command: %v", err))
		return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	log.Info(fmt.Sprintf("Processing confirm-booking: saga_id=%s", command.SagaID))

	// Extract data
	data := &saga.BookingSagaData{}
	data.FromMap(command.Data)

	// Get booking and confirm
	var resultData map[string]interface{}
	var execErr error

	booking, err := w.bookingRepo.GetByID(ctx, data.BookingID)
	if err != nil {
		execErr = err
	} else if booking == nil {
		execErr = fmt.Errorf("booking not found: %s", data.BookingID)
	} else {
		// Confirm booking
		booking.Status = "confirmed"
		booking.PaymentID = data.PaymentID
		booking.UpdatedAt = time.Now()

		if err := w.bookingRepo.Update(ctx, booking); err != nil {
			execErr = err
		} else {
			resultData = map[string]interface{}{
				"confirmation_code": booking.ID[:8], // Use first 8 chars as confirmation code
				"confirmed_at":      time.Now().Format(time.RFC3339),
			}
		}
	}

	finishTime := time.Now()

	// Send result event
	if execErr != nil {
		event := saga.NewSagaFailureEvent(
			command.SagaID,
			command.SagaName,
			command.StepName,
			command.StepIndex,
			execErr.Error(),
			"CONFIRMATION_FAILED",
			startTime,
			finishTime,
		)
		if err := w.producer.SendStepFailureEvent(ctx, event); err != nil {
			log.Error(fmt.Sprintf("Failed to send failure event: %v", err))
		}
	} else {
		event := saga.NewSagaSuccessEvent(
			command.SagaID,
			command.SagaName,
			command.StepName,
			command.StepIndex,
			resultData,
			startTime,
			finishTime,
		)
		if err := w.producer.SendStepSuccessEvent(ctx, event); err != nil {
			log.Error(fmt.Sprintf("Failed to send success event: %v", err))
		}
	}

	return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
}
