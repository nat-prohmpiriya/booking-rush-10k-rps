package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/saga"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

// SagaService manages booking saga execution
type SagaService interface {
	// StartBookingSaga initiates a new booking saga
	StartBookingSaga(ctx context.Context, data *saga.BookingSagaData) (sagaID string, err error)
	// GetSagaStatus retrieves the status of a saga
	GetSagaStatus(ctx context.Context, sagaID string) (*pkgsaga.Instance, error)
}

// KafkaSagaService implements SagaService using Kafka for async saga execution
type KafkaSagaService struct {
	producer    saga.SagaProducer
	store       pkgsaga.Store
	stepTimeout time.Duration
	maxRetries  int
}

// SagaServiceConfig holds configuration for SagaService
type SagaServiceConfig struct {
	StepTimeout time.Duration
	MaxRetries  int
}

// NewKafkaSagaService creates a new Kafka-based saga service
func NewKafkaSagaService(producer saga.SagaProducer, store pkgsaga.Store, cfg *SagaServiceConfig) *KafkaSagaService {
	if cfg == nil {
		cfg = &SagaServiceConfig{
			StepTimeout: 30 * time.Second,
			MaxRetries:  2,
		}
	}
	return &KafkaSagaService{
		producer:    producer,
		store:       store,
		stepTimeout: cfg.StepTimeout,
		maxRetries:  cfg.MaxRetries,
	}
}

// StartBookingSaga initiates a new booking saga by sending the first command
func (s *KafkaSagaService) StartBookingSaga(ctx context.Context, data *saga.BookingSagaData) (string, error) {
	log := logger.Get()

	// Generate saga ID
	sagaID := uuid.New().String()

	// Create saga instance
	instance := pkgsaga.NewInstance(saga.BookingSagaName, data.ToMap())
	instance.ID = sagaID
	instance.SetStatus(pkgsaga.StatusPending)

	// Save to store
	if err := s.store.Save(ctx, instance); err != nil {
		return "", fmt.Errorf("failed to save saga instance: %w", err)
	}

	// Send saga started event
	startedEvent := saga.NewSagaStartedEvent(sagaID, saga.BookingSagaName, data.ToMap())
	if err := s.producer.SendSagaStartedEvent(ctx, startedEvent); err != nil {
		log.Warn(fmt.Sprintf("Failed to send saga started event: %v", err))
	}

	// Send first step command (reserve-seats)
	command := saga.NewSagaCommand(
		sagaID,
		saga.BookingSagaName,
		saga.StepReserveSeats,
		0,
		data.ToMap(),
		s.stepTimeout,
		s.maxRetries,
	)

	if err := s.producer.SendCommand(ctx, command); err != nil {
		// Rollback: delete saga instance
		_ = s.store.Delete(ctx, sagaID)
		return "", fmt.Errorf("failed to send reserve-seats command: %w", err)
	}

	// Update saga status to running
	instance.SetStatus(pkgsaga.StatusRunning)
	if err := s.store.Update(ctx, instance); err != nil {
		log.Warn(fmt.Sprintf("Failed to update saga status: %v", err))
	}

	log.Info(fmt.Sprintf("Started booking saga: saga_id=%s, booking_id=%s", sagaID, data.BookingID))

	return sagaID, nil
}

// GetSagaStatus retrieves the status of a saga
func (s *KafkaSagaService) GetSagaStatus(ctx context.Context, sagaID string) (*pkgsaga.Instance, error) {
	return s.store.Get(ctx, sagaID)
}

// NoOpSagaService is a no-op implementation for when saga is disabled
type NoOpSagaService struct{}

// NewNoOpSagaService creates a no-op saga service
func NewNoOpSagaService() *NoOpSagaService {
	return &NoOpSagaService{}
}

// StartBookingSaga returns an error indicating saga is not enabled
func (s *NoOpSagaService) StartBookingSaga(ctx context.Context, data *saga.BookingSagaData) (string, error) {
	return "", fmt.Errorf("saga service is not enabled")
}

// GetSagaStatus returns an error indicating saga is not enabled
func (s *NoOpSagaService) GetSagaStatus(ctx context.Context, sagaID string) (*pkgsaga.Instance, error) {
	return nil, fmt.Errorf("saga service is not enabled")
}
