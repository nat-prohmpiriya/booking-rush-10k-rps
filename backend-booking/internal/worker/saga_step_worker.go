package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
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
	dlqHandler      *saga.DLQHandler
	config          *SagaStepWorkerConfig
}

// NewSagaStepWorker creates a new saga step worker
func NewSagaStepWorker(
	consumer *kafka.Consumer,
	producer saga.SagaProducer,
	bookingRepo repository.BookingRepository,
	reservationRepo repository.ReservationRepository,
	dlqHandler *saga.DLQHandler,
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
		dlqHandler:      dlqHandler,
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
	case saga.TopicSagaSendNotificationCommand:
		return w.handleSendNotification(ctx, record)
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

		// Check if reservation was successful (lua script may return success=0 with nil error)
		if !result.Success {
			execErr = fmt.Errorf("%s: %s", result.ErrorCode, result.ErrorMessage)
			time.Sleep(w.config.RetryDelay)
			continue
		}

		// Create booking record in PostgreSQL (status = reserved)
		now := time.Now()
		bookingID := result.BookingID
		if bookingID == "" {
			bookingID = uuid.New().String()
		}

		booking := &domain.Booking{
			ID:         bookingID,
			TenantID:   data.TenantID,
			UserID:     data.UserID,
			EventID:    data.EventID,
			ShowID:     data.ShowID,
			ZoneID:     data.ZoneID,
			Quantity:   data.Quantity,
			UnitPrice:  data.TotalPrice / float64(data.Quantity),
			TotalPrice: data.TotalPrice,
			Currency:   data.Currency,
			Status:     domain.BookingStatusReserved,
			ReservedAt: now,
			ExpiresAt:  now.Add(10 * time.Minute),
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := w.bookingRepo.Create(ctx, booking); err != nil {
			log.Error(fmt.Sprintf("Failed to create booking in PostgreSQL: %v", err))
			// Continue anyway - Redis reservation is the source of truth for availability
		} else {
			log.Info(fmt.Sprintf("Created booking in PostgreSQL: booking_id=%s", bookingID))
		}

		resultData = map[string]interface{}{
			"reservation_id": bookingID,
			"booking_id":     bookingID,
			"reserved_at":    now.Format(time.RFC3339),
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
// This is triggered by the post-payment saga after payment success
func (w *SagaStepWorker) handleConfirmBooking(ctx context.Context, record *kafka.Record) error {
	log := logger.Get()
	startTime := time.Now()

	var command saga.SagaCommand
	if err := json.Unmarshal(record.Value, &command); err != nil {
		log.Error(fmt.Sprintf("Failed to unmarshal command: %v", err))
		return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	log.Info(fmt.Sprintf("Processing confirm-booking: saga_id=%s, saga_name=%s", command.SagaID, command.SagaName))

	// Extract booking_id and payment_id from command data
	// Data comes from PaymentSuccessEvent or BookingSagaData
	bookingID, _ := command.Data["booking_id"].(string)
	paymentID, _ := command.Data["payment_id"].(string)
	userID, _ := command.Data["user_id"].(string)

	// Fallback to legacy data extraction
	if bookingID == "" {
		data := &saga.BookingSagaData{}
		data.FromMap(command.Data)
		bookingID = data.BookingID
		if bookingID == "" {
			bookingID = data.ReservationID
		}
		paymentID = data.PaymentID
		userID = data.UserID
	}

	if bookingID == "" {
		log.Error("booking_id is empty in confirm-booking command")
		return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	log.Info(fmt.Sprintf("Confirming booking: booking_id=%s, payment_id=%s", bookingID, paymentID))

	var resultData map[string]interface{}
	var execErr error

	// Step 1: Get booking from PostgreSQL
	booking, err := w.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		execErr = fmt.Errorf("failed to get booking: %w", err)
	} else if booking == nil {
		execErr = fmt.Errorf("booking not found: %s", bookingID)
	} else {
		// Use userID from booking if not provided in command
		if userID == "" {
			userID = booking.UserID
		}

		// Step 2: Confirm in Redis (remove TTL - make reservation permanent)
		if userID != "" {
			redisResult, redisErr := w.reservationRepo.ConfirmBooking(ctx, bookingID, userID, paymentID)
			if redisErr != nil {
				log.Warn(fmt.Sprintf("Failed to confirm in Redis (may have expired): %v", redisErr))
				// Continue anyway - PostgreSQL is the final source of truth
			} else if !redisResult.Success {
				log.Warn(fmt.Sprintf("Redis confirmation returned error: %s - %s", redisResult.ErrorCode, redisResult.ErrorMessage))
				// Continue anyway - booking might have expired in Redis but we still confirm in PostgreSQL
			} else {
				log.Info(fmt.Sprintf("Confirmed booking in Redis (TTL removed): booking_id=%s", bookingID))
			}
		}

		// Step 3: Update PostgreSQL status to confirmed
		now := time.Now()
		booking.Status = domain.BookingStatusConfirmed
		booking.PaymentID = paymentID
		booking.ConfirmedAt = &now
		booking.UpdatedAt = now

		// Generate confirmation code
		confirmationCode := bookingID[:8]
		if len(bookingID) >= 8 {
			confirmationCode = bookingID[:8]
		} else {
			confirmationCode = bookingID
		}
		booking.ConfirmationCode = confirmationCode

		if err := w.bookingRepo.Update(ctx, booking); err != nil {
			execErr = fmt.Errorf("failed to update booking status: %w", err)
		} else {
			log.Info(fmt.Sprintf("Confirmed booking in PostgreSQL: booking_id=%s, confirmation_code=%s", bookingID, confirmationCode))
			resultData = map[string]interface{}{
				"booking_id":        bookingID,
				"confirmation_code": confirmationCode,
				"confirmed_at":      now.Format(time.RFC3339),
				"payment_id":        paymentID,
			}
		}
	}

	finishTime := time.Now()

	// Send result event
	if execErr != nil {
		log.Error(fmt.Sprintf("Confirm booking failed: %v", execErr))
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

// handleSendNotification handles the send-notification step (NON-CRITICAL)
// This step is NON-CRITICAL: if it fails after retries, it goes to DLQ
// It does NOT trigger compensation (no refund, no seat release)
func (w *SagaStepWorker) handleSendNotification(ctx context.Context, record *kafka.Record) error {
	log := logger.Get()
	startTime := time.Now()

	var command saga.SagaCommand
	if err := json.Unmarshal(record.Value, &command); err != nil {
		log.Error(fmt.Sprintf("Failed to unmarshal notification command: %v", err))
		return w.consumer.CommitRecords(ctx, []*kafka.Record{record})
	}

	log.Info(fmt.Sprintf("Processing send-notification (NON-CRITICAL): saga_id=%s", command.SagaID))

	// Extract data from command
	bookingID, _ := command.Data["booking_id"].(string)
	userID, _ := command.Data["user_id"].(string)
	confirmationCode, _ := command.Data["confirmation_code"].(string)

	var resultData map[string]interface{}
	var execErr error

	// Mock notification implementation
	// In production, this would call email service (SendGrid, AWS SES, etc.)
	for attempt := 0; attempt < w.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			log.Info(fmt.Sprintf("Retrying notification: saga_id=%s, attempt=%d", command.SagaID, attempt+1))
			time.Sleep(w.config.RetryDelay * time.Duration(attempt+1)) // Exponential backoff
		}

		// MOCK: Simulate sending notification
		// TODO: Replace with real notification service
		notificationID := fmt.Sprintf("notif-%s", uuid.New().String()[:8])

		log.Info(fmt.Sprintf("[MOCK] Sending booking confirmation email: booking_id=%s, user_id=%s, confirmation_code=%s",
			bookingID, userID, confirmationCode))

		// Simulate success (in production, check email service response)
		resultData = map[string]interface{}{
			"notification_id":   notificationID,
			"notification_type": "email",
			"booking_id":        bookingID,
			"user_id":           userID,
			"sent_at":           time.Now().Format(time.RFC3339),
		}
		execErr = nil
		break
	}

	finishTime := time.Now()

	// Handle result
	if execErr != nil {
		log.Warn(fmt.Sprintf("Notification failed after %d retries: saga_id=%s, error=%v",
			w.config.RetryAttempts, command.SagaID, execErr))

		// NON-CRITICAL: Send to DLQ instead of triggering compensation
		if w.dlqHandler != nil {
			dlqErr := w.dlqHandler.HandleFailedMessage(
				ctx,
				saga.TopicSagaSendNotificationCommand,
				command.SagaID,
				record.Value,
				execErr,
				w.config.RetryAttempts,
			)
			if dlqErr != nil {
				log.Error(fmt.Sprintf("Failed to send to DLQ: %v", dlqErr))
			}
		}

		// Still send success event to complete the saga
		// Because notification is NON-CRITICAL - the booking is already confirmed
		log.Info(fmt.Sprintf("NON-CRITICAL step failed, completing saga anyway: saga_id=%s", command.SagaID))

		event := saga.NewSagaSuccessEvent(
			command.SagaID,
			command.SagaName,
			command.StepName,
			command.StepIndex,
			map[string]interface{}{
				"notification_status": "failed_to_dlq",
				"error":               execErr.Error(),
			},
			startTime,
			finishTime,
		)
		if err := w.producer.SendStepSuccessEvent(ctx, event); err != nil {
			log.Error(fmt.Sprintf("Failed to send success event: %v", err))
		}
	} else {
		log.Info(fmt.Sprintf("Notification sent successfully: saga_id=%s, booking_id=%s", command.SagaID, bookingID))

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
