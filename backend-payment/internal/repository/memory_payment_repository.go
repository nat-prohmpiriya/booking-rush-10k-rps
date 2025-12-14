package repository

import (
	"context"
	"sync"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/domain"
)

// MemoryPaymentRepository implements PaymentRepository using in-memory storage
// This is useful for testing and development
type MemoryPaymentRepository struct {
	payments         map[string]*domain.Payment
	byBooking        map[string]string   // bookingID -> paymentID
	byUser           map[string][]string // userID -> []paymentID
	byGatewayPayment map[string]string   // gatewayPaymentID -> paymentID
	byIdempotency    map[string]string   // idempotencyKey -> paymentID
	mu               sync.RWMutex
}

// NewMemoryPaymentRepository creates a new in-memory payment repository
func NewMemoryPaymentRepository() *MemoryPaymentRepository {
	return &MemoryPaymentRepository{
		payments:         make(map[string]*domain.Payment),
		byBooking:        make(map[string]string),
		byUser:           make(map[string][]string),
		byGatewayPayment: make(map[string]string),
		byIdempotency:    make(map[string]string),
	}
}

// Create creates a new payment record
func (r *MemoryPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.payments[payment.ID]; exists {
		return domain.ErrPaymentAlreadyExists
	}

	if _, exists := r.byBooking[payment.BookingID]; exists {
		return domain.ErrPaymentAlreadyExists
	}

	// Clone payment to avoid external modifications
	p := *payment
	r.payments[payment.ID] = &p
	r.byBooking[payment.BookingID] = payment.ID
	r.byUser[payment.UserID] = append(r.byUser[payment.UserID], payment.ID)

	return nil
}

// GetByID retrieves a payment by its ID
func (r *MemoryPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	payment, exists := r.payments[id]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}

	// Return a copy
	p := *payment
	return &p, nil
}

// GetByBookingID retrieves a payment by booking ID
func (r *MemoryPaymentRepository) GetByBookingID(ctx context.Context, bookingID string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paymentID, exists := r.byBooking[bookingID]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}

	payment, exists := r.payments[paymentID]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}

	p := *payment
	return &p, nil
}

// GetByUserID retrieves all payments for a user
func (r *MemoryPaymentRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paymentIDs, exists := r.byUser[userID]
	if !exists {
		return []*domain.Payment{}, nil
	}

	// Apply pagination
	start := offset
	if start >= len(paymentIDs) {
		return []*domain.Payment{}, nil
	}

	end := start + limit
	if end > len(paymentIDs) {
		end = len(paymentIDs)
	}

	result := make([]*domain.Payment, 0, end-start)
	for _, id := range paymentIDs[start:end] {
		if payment, exists := r.payments[id]; exists {
			p := *payment
			result = append(result, &p)
		}
	}

	return result, nil
}

// Update updates an existing payment
func (r *MemoryPaymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.payments[payment.ID]; !exists {
		return domain.ErrPaymentNotFound
	}

	// Clone and update
	p := *payment
	r.payments[payment.ID] = &p

	// Update gateway payment index if set
	if payment.GatewayPaymentID != "" {
		r.byGatewayPayment[payment.GatewayPaymentID] = payment.ID
	}

	// Update idempotency key index if set
	if payment.IdempotencyKey != "" {
		r.byIdempotency[payment.IdempotencyKey] = payment.ID
	}

	return nil
}

// GetByGatewayPaymentID retrieves a payment by gateway payment ID
func (r *MemoryPaymentRepository) GetByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paymentID, exists := r.byGatewayPayment[gatewayPaymentID]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}

	payment, exists := r.payments[paymentID]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}

	p := *payment
	return &p, nil
}

// GetByIdempotencyKey retrieves a payment by idempotency key
func (r *MemoryPaymentRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paymentID, exists := r.byIdempotency[idempotencyKey]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}

	payment, exists := r.payments[paymentID]
	if !exists {
		return nil, domain.ErrPaymentNotFound
	}

	p := *payment
	return &p, nil
}

// Clear clears all data (for testing)
func (r *MemoryPaymentRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.payments = make(map[string]*domain.Payment)
	r.byBooking = make(map[string]string)
	r.byUser = make(map[string][]string)
	r.byGatewayPayment = make(map[string]string)
	r.byIdempotency = make(map[string]string)
}

// Count returns the total number of payments (for testing)
func (r *MemoryPaymentRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.payments)
}
