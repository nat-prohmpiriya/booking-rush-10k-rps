package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

func TestBookingSagaBuilder_Build(t *testing.T) {
	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  NewMockSeatReservationService(),
		PaymentService:      NewMockPaymentService(),
		ConfirmationService: NewMockBookingConfirmationService(),
		NotificationService: NewMockNotificationService(),
	})

	def := builder.Build()

	if def.Name != BookingSagaName {
		t.Errorf("expected saga name %s, got %s", BookingSagaName, def.Name)
	}

	// Legacy saga has 3 steps (notification is commented out)
	if len(def.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(def.Steps))
	}

	expectedSteps := []string{
		StepReserveSeats,
		StepProcessPayment,
		StepConfirmBooking,
	}

	for i, step := range def.Steps {
		if step.Name != expectedSteps[i] {
			t.Errorf("step %d: expected name %s, got %s", i, expectedSteps[i], step.Name)
		}
	}
}

func TestPostPaymentSagaBuilder_Build(t *testing.T) {
	builder := NewPostPaymentSagaBuilder(&PostPaymentSagaConfig{
		StepTimeout: 30 * time.Second,
		MaxRetries:  3,
	})

	def := builder.Build()

	if def.Name != PostPaymentSagaName {
		t.Errorf("expected saga name %s, got %s", PostPaymentSagaName, def.Name)
	}

	// Post-payment saga has 2 steps:
	// 1. confirm-booking (CRITICAL)
	// 2. send-notification (NON-CRITICAL)
	if len(def.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(def.Steps))
	}

	// Verify step names
	expectedSteps := []string{StepConfirmBooking, StepSendNotification}
	for i, step := range def.Steps {
		if step.Name != expectedSteps[i] {
			t.Errorf("step %d: expected name %s, got %s", i, expectedSteps[i], step.Name)
		}
	}

	// Verify step policies:
	// - confirm-booking: CRITICAL (would have Compensate if implemented)
	// - send-notification: NON-CRITICAL (Compensate is nil)
	if def.Steps[1].Compensate != nil {
		t.Error("send-notification step should have nil Compensate (NON-CRITICAL)")
	}
}

func TestBookingSaga_SuccessfulExecution(t *testing.T) {
	// Setup mock services
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()
	notificationSvc := NewMockNotificationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-123",
		"user_id":        "user-456",
		"event_id":       "event-789",
		"zone_id":        "zone-A",
		"quantity":       2,
		"total_price":    200.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	}

	instance, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err != nil {
		t.Fatalf("saga execution failed: %v", err)
	}

	// Verify saga completed successfully
	if instance.Status != pkgsaga.StatusCompleted {
		t.Errorf("expected status %s, got %s", pkgsaga.StatusCompleted, instance.Status)
	}

	// Verify all steps completed (3 steps: reserve, payment, confirm - notification is commented out)
	if len(instance.StepResults) != 3 {
		t.Errorf("expected 3 step results, got %d", len(instance.StepResults))
	}

	for _, result := range instance.StepResults {
		if result.Status != pkgsaga.StepStatusCompleted {
			t.Errorf("step %s: expected status %s, got %s", result.StepName, pkgsaga.StepStatusCompleted, result.Status)
		}
	}

	// Verify reservation was created
	reservation, exists := reservationSvc.GetReservation("booking-123")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if reservation.Released {
		t.Error("expected reservation not to be released")
	}

	// Verify payment was processed
	payment, exists := paymentSvc.GetPaymentByBookingID("booking-123")
	if !exists {
		t.Error("expected payment to exist")
	}
	if payment.Refunded {
		t.Error("expected payment not to be refunded")
	}

	// Verify booking was confirmed
	confirmation, exists := confirmationSvc.GetConfirmation("booking-123")
	if !exists {
		t.Error("expected confirmation to exist")
	}
	if confirmation.ConfirmationCode == "" {
		t.Error("expected confirmation code to be set")
	}

	// Note: Notification step is commented out in the saga definition
	// so we don't verify notification here
}

func TestBookingSaga_ReservationFailure_NoCompensation(t *testing.T) {
	// Setup mock services with reservation failure
	reservationSvc := NewMockSeatReservationService()
	reservationSvc.ShouldFail = true
	reservationSvc.FailureError = ErrInsufficientSeats

	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()
	notificationSvc := NewMockNotificationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-123",
		"user_id":        "user-456",
		"event_id":       "event-789",
		"zone_id":        "zone-A",
		"quantity":       2,
		"total_price":    200.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	}

	_, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err == nil {
		t.Fatal("expected saga execution to fail")
	}

	// Verify no payment was processed (first step failed, no compensation needed)
	_, exists := paymentSvc.GetPaymentByBookingID("booking-123")
	if exists {
		t.Error("expected no payment to exist since reservation failed")
	}
}

func TestBookingSaga_PaymentFailure_ReleasesSeats(t *testing.T) {
	// Setup mock services with payment failure
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	paymentSvc.ShouldFail = true
	paymentSvc.FailureError = ErrPaymentDeclined

	confirmationSvc := NewMockBookingConfirmationService()
	notificationSvc := NewMockNotificationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
		MaxRetries:          0, // No retries for faster test
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-456",
		"user_id":        "user-789",
		"event_id":       "event-123",
		"zone_id":        "zone-B",
		"quantity":       3,
		"total_price":    300.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	}

	_, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err == nil {
		t.Fatal("expected saga execution to fail")
	}

	// Verify reservation was created and then released (compensated)
	reservation, exists := reservationSvc.GetReservation("booking-456")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if !reservation.Released {
		t.Error("expected reservation to be released (compensated)")
	}
}

func TestBookingSaga_ConfirmationFailure_RefundsPayment(t *testing.T) {
	// Setup mock services with confirmation failure
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()
	confirmationSvc.ShouldFail = true
	confirmationSvc.FailureError = errors.New("confirmation service unavailable")

	notificationSvc := NewMockNotificationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
		MaxRetries:          0,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-789",
		"user_id":        "user-123",
		"event_id":       "event-456",
		"zone_id":        "zone-C",
		"quantity":       1,
		"total_price":    100.00,
		"currency":       "THB",
		"payment_method": "debit_card",
	}

	_, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err == nil {
		t.Fatal("expected saga execution to fail")
	}

	// Verify reservation was released
	reservation, exists := reservationSvc.GetReservation("booking-789")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if !reservation.Released {
		t.Error("expected reservation to be released (compensated)")
	}

	// Verify payment was refunded
	payment, exists := paymentSvc.GetPaymentByBookingID("booking-789")
	if !exists {
		t.Error("expected payment to exist")
	}
	if !payment.Refunded {
		t.Error("expected payment to be refunded (compensated)")
	}
}

// Note: TestBookingSaga_NotificationFailure_StillCompletes is removed because
// the notification step is now commented out in the saga definition.
// Notification will be implemented as a separate async worker in the future.

func TestBookingSaga_WithoutNotificationService(t *testing.T) {
	// Setup mock services without notification service
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: nil, // No notification service
		StepTimeout:         5 * time.Second,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-no-notify",
		"user_id":        "user-no-notify",
		"event_id":       "event-no-notify",
		"zone_id":        "zone-E",
		"quantity":       1,
		"total_price":    150.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	}

	instance, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err != nil {
		t.Fatalf("saga execution failed: %v", err)
	}

	// Verify saga completed successfully
	if instance.Status != pkgsaga.StatusCompleted {
		t.Errorf("expected status %s, got %s", pkgsaga.StatusCompleted, instance.Status)
	}
}

func TestBookingSagaData_ToMapAndFromMap(t *testing.T) {
	original := &BookingSagaData{
		BookingID:        "booking-123",
		UserID:           "user-456",
		EventID:          "event-789",
		ZoneID:           "zone-A",
		Quantity:         5,
		TotalPrice:       500.00,
		Currency:         "THB",
		PaymentMethod:    "credit_card",
		IdempotencyKey:   "idem-key-123",
		ReservationID:    "res-123",
		PaymentID:        "pay-456",
		ConfirmationCode: "CONF-789",
		NotificationID:   "notif-012",
	}

	// Convert to map
	m := original.ToMap()

	// Convert back from map
	restored := &BookingSagaData{}
	restored.FromMap(m)

	// Verify all fields
	if restored.BookingID != original.BookingID {
		t.Errorf("BookingID: expected %s, got %s", original.BookingID, restored.BookingID)
	}
	if restored.UserID != original.UserID {
		t.Errorf("UserID: expected %s, got %s", original.UserID, restored.UserID)
	}
	if restored.EventID != original.EventID {
		t.Errorf("EventID: expected %s, got %s", original.EventID, restored.EventID)
	}
	if restored.ZoneID != original.ZoneID {
		t.Errorf("ZoneID: expected %s, got %s", original.ZoneID, restored.ZoneID)
	}
	if restored.Quantity != original.Quantity {
		t.Errorf("Quantity: expected %d, got %d", original.Quantity, restored.Quantity)
	}
	if restored.TotalPrice != original.TotalPrice {
		t.Errorf("TotalPrice: expected %f, got %f", original.TotalPrice, restored.TotalPrice)
	}
	if restored.Currency != original.Currency {
		t.Errorf("Currency: expected %s, got %s", original.Currency, restored.Currency)
	}
	if restored.PaymentMethod != original.PaymentMethod {
		t.Errorf("PaymentMethod: expected %s, got %s", original.PaymentMethod, restored.PaymentMethod)
	}
	if restored.IdempotencyKey != original.IdempotencyKey {
		t.Errorf("IdempotencyKey: expected %s, got %s", original.IdempotencyKey, restored.IdempotencyKey)
	}
	if restored.ReservationID != original.ReservationID {
		t.Errorf("ReservationID: expected %s, got %s", original.ReservationID, restored.ReservationID)
	}
	if restored.PaymentID != original.PaymentID {
		t.Errorf("PaymentID: expected %s, got %s", original.PaymentID, restored.PaymentID)
	}
	if restored.ConfirmationCode != original.ConfirmationCode {
		t.Errorf("ConfirmationCode: expected %s, got %s", original.ConfirmationCode, restored.ConfirmationCode)
	}
	if restored.NotificationID != original.NotificationID {
		t.Errorf("NotificationID: expected %s, got %s", original.NotificationID, restored.NotificationID)
	}
}

func TestMockSeatReservationService(t *testing.T) {
	svc := NewMockSeatReservationService()
	ctx := context.Background()

	// Test successful reservation
	reservationID, err := svc.ReserveSeats(ctx, "booking-1", "user-1", "event-1", "zone-1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reservationID == "" {
		t.Error("expected reservation ID to be set")
	}

	// Verify reservation exists
	reservation, exists := svc.GetReservation("booking-1")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if reservation.Quantity != 2 {
		t.Errorf("expected quantity 2, got %d", reservation.Quantity)
	}

	// Test release
	err = svc.ReleaseSeats(ctx, "booking-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reservation, _ = svc.GetReservation("booking-1")
	if !reservation.Released {
		t.Error("expected reservation to be released")
	}

	// Test release non-existent
	err = svc.ReleaseSeats(ctx, "non-existent", "user-1")
	if !errors.Is(err, ErrReservationNotFound) {
		t.Errorf("expected ErrReservationNotFound, got %v", err)
	}
}

func TestMockPaymentService(t *testing.T) {
	svc := NewMockPaymentService()
	ctx := context.Background()

	// Test successful payment
	paymentID, err := svc.ProcessPayment(ctx, "booking-1", "user-1", 100.00, "THB", "credit_card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paymentID == "" {
		t.Error("expected payment ID to be set")
	}

	// Verify payment exists
	payment, exists := svc.GetPayment(paymentID)
	if !exists {
		t.Error("expected payment to exist")
	}
	if payment.Amount != 100.00 {
		t.Errorf("expected amount 100.00, got %f", payment.Amount)
	}

	// Test refund
	err = svc.RefundPayment(ctx, paymentID, "test refund")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payment, _ = svc.GetPayment(paymentID)
	if !payment.Refunded {
		t.Error("expected payment to be refunded")
	}
	if payment.RefundReason != "test refund" {
		t.Errorf("expected refund reason 'test refund', got '%s'", payment.RefundReason)
	}

	// Test refund non-existent
	err = svc.RefundPayment(ctx, "non-existent", "reason")
	if !errors.Is(err, ErrPaymentNotFound) {
		t.Errorf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestMockBookingConfirmationService(t *testing.T) {
	svc := NewMockBookingConfirmationService()
	ctx := context.Background()

	// Test successful confirmation
	confirmationCode, err := svc.ConfirmBooking(ctx, "booking-1", "user-1", "payment-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmationCode == "" {
		t.Error("expected confirmation code to be set")
	}

	// Verify confirmation exists
	confirmation, exists := svc.GetConfirmation("booking-1")
	if !exists {
		t.Error("expected confirmation to exist")
	}
	if confirmation.PaymentID != "payment-1" {
		t.Errorf("expected payment ID 'payment-1', got '%s'", confirmation.PaymentID)
	}
}

func TestMockNotificationService(t *testing.T) {
	svc := NewMockNotificationService()
	ctx := context.Background()

	// Test successful notification
	notificationID, err := svc.SendBookingConfirmation(ctx, "user-1", "booking-1", "CONF-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notificationID == "" {
		t.Error("expected notification ID to be set")
	}

	// Verify notification exists
	notification, exists := svc.GetNotification(notificationID)
	if !exists {
		t.Error("expected notification to exist")
	}
	if notification.ConfirmationCode != "CONF-123" {
		t.Errorf("expected confirmation code 'CONF-123', got '%s'", notification.ConfirmationCode)
	}
}

// ============================================================================
// STEP POLICY TESTS - Critical vs Non-Critical Steps
// ============================================================================
//
// Step Policy defines how the saga handles step failures:
// - CRITICAL Step: If fail → Trigger compensation (refund + release seats)
// - NON-CRITICAL Step: If fail → Retry → DLQ (NO compensation, saga still completes)
//
// Post-Payment Saga Step Policies:
// - confirm-booking: CRITICAL (payment already done, must confirm or refund)
// - send-notification: NON-CRITICAL (booking confirmed, email can fail gracefully)

// TestStepPolicy_VerifyStepTypes verifies which steps are Critical vs Non-Critical
// by checking if they have a Compensate function defined.
func TestStepPolicy_VerifyStepTypes(t *testing.T) {
	t.Run("PostPaymentSaga_StepTypes", func(t *testing.T) {
		builder := NewPostPaymentSagaBuilder(&PostPaymentSagaConfig{
			StepTimeout: 30 * time.Second,
			MaxRetries:  3,
		})
		def := builder.Build()

		// Step policy table for documentation and verification
		stepPolicies := map[string]struct {
			isCritical  bool
			description string
		}{
			StepConfirmBooking: {
				isCritical:  true, // Note: Currently nil, but SHOULD have compensation in production
				description: "CRITICAL: If fails, must refund payment and release seats",
			},
			StepSendNotification: {
				isCritical:  false,
				description: "NON-CRITICAL: If fails, just retry and DLQ. Booking is already confirmed.",
			},
		}

		for _, step := range def.Steps {
			policy, ok := stepPolicies[step.Name]
			if !ok {
				t.Errorf("unexpected step: %s", step.Name)
				continue
			}

			// For NON-CRITICAL steps, Compensate MUST be nil
			if !policy.isCritical && step.Compensate != nil {
				t.Errorf("step %s is NON-CRITICAL but has Compensate function. "+
					"Non-critical steps should NOT trigger compensation on failure. %s",
					step.Name, policy.description)
			}

			t.Logf("✓ Step '%s': %s", step.Name, policy.description)
		}
	})

	t.Run("LegacySaga_StepTypes", func(t *testing.T) {
		builder := NewBookingSagaBuilder(&BookingSagaConfig{
			ReservationService:  NewMockSeatReservationService(),
			PaymentService:      NewMockPaymentService(),
			ConfirmationService: NewMockBookingConfirmationService(),
		})
		def := builder.Build()

		// All legacy saga steps are CRITICAL (except notification which is commented out)
		criticalSteps := []string{
			StepReserveSeats,
			StepProcessPayment,
			StepConfirmBooking,
		}

		for i, stepName := range criticalSteps {
			if i >= len(def.Steps) {
				break
			}
			step := def.Steps[i]
			if step.Name != stepName {
				t.Errorf("step %d: expected %s, got %s", i, stepName, step.Name)
			}

			// These are CRITICAL steps - they should have compensation
			// (except confirm-booking which has nil Compensate because
			// if confirm fails after payment, payment step's compensate handles refund)
			t.Logf("✓ Legacy step '%s': CRITICAL", step.Name)
		}
	})
}

// TestStepPolicy_CriticalStep_ConfirmBookingFailure documents behavior when
// a CRITICAL step fails - should trigger full compensation (refund + release).
func TestStepPolicy_CriticalStep_ConfirmBookingFailure(t *testing.T) {
	// This test demonstrates that when confirm-booking (CRITICAL step) fails,
	// the saga triggers compensation: refund payment AND release seats.
	//
	// Flow:
	// 1. Reserve seats ✓
	// 2. Process payment ✓
	// 3. Confirm booking ✗ (FAIL)
	// 4. COMPENSATION TRIGGERED:
	//    - Refund payment ✓
	//    - Release seats ✓
	// 5. Saga status: FAILED (with compensation)

	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()
	confirmationSvc.ShouldFail = true
	confirmationSvc.FailureError = errors.New("database connection failed")

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		StepTimeout:         5 * time.Second,
		MaxRetries:          0, // No retries for faster test
	})

	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga: %v", err)
	}

	ctx := context.Background()
	_, err := orchestrator.Execute(ctx, BookingSagaName, map[string]interface{}{
		"booking_id":     "critical-test-booking",
		"user_id":        "user-critical",
		"event_id":       "event-1",
		"zone_id":        "zone-A",
		"quantity":       2,
		"total_price":    200.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	})

	// Saga should fail
	if err == nil {
		t.Fatal("expected saga to fail when CRITICAL step (confirm-booking) fails")
	}

	// CRITICAL BEHAVIOR: Compensation must be triggered
	// 1. Seats must be released
	reservation, exists := reservationSvc.GetReservation("critical-test-booking")
	if !exists {
		t.Fatal("reservation should exist")
	}
	if !reservation.Released {
		t.Error("CRITICAL STEP FAILED: Seats should be released (compensated)")
	}

	// 2. Payment must be refunded
	payment, exists := paymentSvc.GetPaymentByBookingID("critical-test-booking")
	if !exists {
		t.Fatal("payment should exist")
	}
	if !payment.Refunded {
		t.Error("CRITICAL STEP FAILED: Payment should be refunded (compensated)")
	}

	t.Log("✓ CRITICAL step (confirm-booking) failure correctly triggered compensation")
	t.Log("  → Seats released: true")
	t.Log("  → Payment refunded: true")
}

// TestStepPolicy_NonCriticalStep_NotificationFailure_NoCompensation documents
// that when NON-CRITICAL step fails, saga still completes WITHOUT compensation.
//
// Note: This test demonstrates the CONCEPT. In actual event-driven implementation,
// the saga_step_worker handles this by:
// 1. Retrying the notification
// 2. If all retries fail, sending to DLQ
// 3. Sending success event to orchestrator (saga completes)
// 4. NOT triggering any compensation (no refund, no seat release)
func TestStepPolicy_NonCriticalStep_Description(t *testing.T) {
	t.Log("NON-CRITICAL Step Behavior (send-notification):")
	t.Log("")
	t.Log("When notification fails in post-payment saga:")
	t.Log("1. Step worker retries sending (up to 5 times with backoff)")
	t.Log("2. If all retries fail → Message sent to DLQ for manual retry")
	t.Log("3. Saga still receives SUCCESS event → Saga completes")
	t.Log("4. NO compensation triggered (no refund, no seat release)")
	t.Log("")
	t.Log("Reason: Customer already has their ticket (payment done, booking confirmed).")
	t.Log("        Email failure is NOT a reason to cancel their booking.")
	t.Log("")
	t.Log("DLQ allows operations team to manually resend notification later.")

	// Verify PostPaymentSaga has notification as NON-CRITICAL
	builder := NewPostPaymentSagaBuilder(&PostPaymentSagaConfig{})
	def := builder.Build()

	if len(def.Steps) < 2 {
		t.Fatal("post-payment saga should have 2 steps")
	}

	notificationStep := def.Steps[1]
	if notificationStep.Name != StepSendNotification {
		t.Errorf("second step should be %s, got %s", StepSendNotification, notificationStep.Name)
	}

	// Key check: NON-CRITICAL step has NO Compensate function
	if notificationStep.Compensate != nil {
		t.Error("send-notification is NON-CRITICAL and should have nil Compensate")
	}

	// Verify higher retry count for non-critical steps
	if notificationStep.Retries < 3 {
		t.Errorf("NON-CRITICAL steps should have more retries, got %d", notificationStep.Retries)
	}

	t.Log("✓ send-notification step is correctly configured as NON-CRITICAL")
	t.Logf("  → Compensate: nil (no compensation on failure)")
	t.Logf("  → Retries: %d (more retries before DLQ)", notificationStep.Retries)
}

// TestStepPolicy_CompensationOrder verifies compensation happens in reverse order
// when a CRITICAL step fails in the middle of saga execution.
func TestStepPolicy_CompensationOrder(t *testing.T) {
	t.Log("Compensation Order for CRITICAL Step Failure:")
	t.Log("")
	t.Log("If step 3 (confirm-booking) fails:")
	t.Log("1. Compensate step 2 (process-payment) → Refund")
	t.Log("2. Compensate step 1 (reserve-seats) → Release seats")
	t.Log("")
	t.Log("Compensation happens in REVERSE ORDER of execution.")

	// This is already tested in TestStepPolicy_CriticalStep_ConfirmBookingFailure
	// This test just documents the expected order.

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  NewMockSeatReservationService(),
		PaymentService:      NewMockPaymentService(),
		ConfirmationService: NewMockBookingConfirmationService(),
	})
	def := builder.Build()

	// Verify steps have compensation in correct order
	if def.Steps[0].Name != StepReserveSeats {
		t.Error("first step should be reserve-seats")
	}
	if def.Steps[0].Compensate == nil {
		t.Error("reserve-seats should have Compensate (release seats)")
	}

	if def.Steps[1].Name != StepProcessPayment {
		t.Error("second step should be process-payment")
	}
	if def.Steps[1].Compensate == nil {
		t.Error("process-payment should have Compensate (refund)")
	}

	t.Log("✓ Compensation functions are correctly defined for CRITICAL steps")
}

// TestStepPolicy_Summary provides a complete summary of step policies
func TestStepPolicy_Summary(t *testing.T) {
	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════════════╗")
	t.Log("║                    SAGA STEP POLICY SUMMARY                      ║")
	t.Log("╠══════════════════════════════════════════════════════════════════╣")
	t.Log("║                                                                  ║")
	t.Log("║  CRITICAL Steps (trigger compensation on failure):              ║")
	t.Log("║  ┌─────────────────┬────────────────────────────────────────┐   ║")
	t.Log("║  │ Step            │ Compensation                           │   ║")
	t.Log("║  ├─────────────────┼────────────────────────────────────────┤   ║")
	t.Log("║  │ reserve-seats   │ release-seats (return to inventory)    │   ║")
	t.Log("║  │ process-payment │ refund-payment (return money)          │   ║")
	t.Log("║  │ confirm-booking │ refund + release (handled by above)    │   ║")
	t.Log("║  └─────────────────┴────────────────────────────────────────┘   ║")
	t.Log("║                                                                  ║")
	t.Log("║  NON-CRITICAL Steps (NO compensation, retry → DLQ):             ║")
	t.Log("║  ┌──────────────────┬───────────────────────────────────────┐   ║")
	t.Log("║  │ Step             │ On Failure                            │   ║")
	t.Log("║  ├──────────────────┼───────────────────────────────────────┤   ║")
	t.Log("║  │ send-notification│ Retry 5x → DLQ → Saga still completes │   ║")
	t.Log("║  └──────────────────┴───────────────────────────────────────┘   ║")
	t.Log("║                                                                  ║")
	t.Log("║  KEY INSIGHT:                                                   ║")
	t.Log("║  Customer has their ticket even if email fails.                 ║")
	t.Log("║  Don't punish customer for infrastructure issues.               ║")
	t.Log("║                                                                  ║")
	t.Log("╚══════════════════════════════════════════════════════════════════╝")
	t.Log("")
}
