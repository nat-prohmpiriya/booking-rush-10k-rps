package metrics

import (
	"context"
	"sync"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

var (
	// Payment counters
	PaymentsCreated   *telemetry.Counter
	PaymentsProcessed *telemetry.Counter
	PaymentsFailed    *telemetry.Counter
	PaymentsRefunded  *telemetry.Counter
	PaymentsCancelled *telemetry.Counter

	// Webhook counters
	WebhooksReceived  *telemetry.Counter
	WebhooksProcessed *telemetry.Counter
	WebhooksFailed    *telemetry.Counter

	// Error tracking counters
	ErrorsTotal       *telemetry.Counter
	SlowRequestsTotal *telemetry.Counter

	// Histograms
	PaymentDuration       *telemetry.Histogram
	PaymentAmount         *telemetry.Histogram
	WebhookProcessingTime *telemetry.Histogram
	RequestDuration       *telemetry.Histogram

	// Gauges
	PendingPayments *telemetry.UpDownCounter

	initOnce sync.Once
	initErr  error
)

// Init initializes all payment metrics
func Init() error {
	initOnce.Do(func() {
		initErr = initMetrics()
	})
	return initErr
}

func initMetrics() error {
	var err error

	// Payment counters
	PaymentsCreated, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_created_total",
		Description: "Total number of payments created",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	PaymentsProcessed, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_processed_total",
		Description: "Total number of successfully processed payments",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	PaymentsFailed, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_failed_total",
		Description: "Total number of failed payments",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	PaymentsRefunded, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_refunded_total",
		Description: "Total number of refunded payments",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	PaymentsCancelled, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_cancelled_total",
		Description: "Total number of cancelled payments",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	// Webhook counters
	WebhooksReceived, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_webhooks_received_total",
		Description: "Total number of webhooks received",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	WebhooksProcessed, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_webhooks_processed_total",
		Description: "Total number of webhooks successfully processed",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	WebhooksFailed, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_webhooks_failed_total",
		Description: "Total number of webhooks that failed processing",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	// Histograms
	PaymentDuration, err = telemetry.NewHistogramWithBuckets(telemetry.MetricOpts{
		Name:        "payment_processing_duration_seconds",
		Description: "Duration of payment processing",
		Unit:        "s",
	}, []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}) // 100ms to 30s
	if err != nil {
		return err
	}

	PaymentAmount, err = telemetry.NewHistogramWithBuckets(telemetry.MetricOpts{
		Name:        "payment_amount",
		Description: "Payment amounts distribution",
		Unit:        "THB",
	}, []float64{100, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000}) // 100 to 100k THB
	if err != nil {
		return err
	}

	WebhookProcessingTime, err = telemetry.NewHistogramWithBuckets(telemetry.MetricOpts{
		Name:        "payment_webhook_processing_seconds",
		Description: "Webhook processing duration",
		Unit:        "s",
	}, []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}) // 10ms to 5s
	if err != nil {
		return err
	}

	// Request duration histogram for latency tracking (p50, p90, p99)
	RequestDuration, err = telemetry.NewHistogramWithBuckets(telemetry.MetricOpts{
		Name:        "payment_request_duration_seconds",
		Description: "HTTP request duration in seconds",
		Unit:        "s",
	}, []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}) // 5ms to 10s
	if err != nil {
		return err
	}

	// Error tracking
	ErrorsTotal, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_errors_total",
		Description: "Total number of errors by type",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	SlowRequestsTotal, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "payment_slow_requests_total",
		Description: "Total number of slow requests (>1s)",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	// Up-down counter for current state
	PendingPayments, err = telemetry.NewUpDownCounter(telemetry.MetricOpts{
		Name:        "payment_pending",
		Description: "Current number of pending payments",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	return nil
}

// RecordPaymentCreated records a payment creation metric
func RecordPaymentCreated(ctx context.Context, bookingID, method, currency string, amount float64) {
	if PaymentsCreated != nil {
		PaymentsCreated.Inc(ctx,
			attribute.String("booking_id", bookingID),
			attribute.String("method", method),
			attribute.String("currency", currency),
		)
	}
	if PaymentAmount != nil {
		PaymentAmount.Record(ctx, amount,
			attribute.String("currency", currency),
		)
	}
	if PendingPayments != nil {
		PendingPayments.Inc(ctx)
	}
}

// RecordPaymentProcessed records a successful payment processing metric
func RecordPaymentProcessed(ctx context.Context, bookingID, method, currency string, durationSeconds float64) {
	if PaymentsProcessed != nil {
		PaymentsProcessed.Inc(ctx,
			attribute.String("booking_id", bookingID),
			attribute.String("method", method),
			attribute.String("currency", currency),
		)
	}
	if PaymentDuration != nil {
		PaymentDuration.Record(ctx, durationSeconds,
			attribute.String("method", method),
		)
	}
	if PendingPayments != nil {
		PendingPayments.Dec(ctx)
	}
}

// RecordPaymentFailed records a payment failure metric
func RecordPaymentFailed(ctx context.Context, bookingID, method, reason string) {
	if PaymentsFailed != nil {
		PaymentsFailed.Inc(ctx,
			attribute.String("booking_id", bookingID),
			attribute.String("method", method),
			attribute.String("reason", reason),
		)
	}
	if PendingPayments != nil {
		PendingPayments.Dec(ctx)
	}
}

// RecordPaymentRefunded records a payment refund metric
func RecordPaymentRefunded(ctx context.Context, bookingID, reason string, amount float64) {
	if PaymentsRefunded != nil {
		PaymentsRefunded.Inc(ctx,
			attribute.String("booking_id", bookingID),
			attribute.String("reason", reason),
		)
	}
}

// RecordPaymentCancelled records a payment cancellation metric
func RecordPaymentCancelled(ctx context.Context, bookingID string) {
	if PaymentsCancelled != nil {
		PaymentsCancelled.Inc(ctx,
			attribute.String("booking_id", bookingID),
		)
	}
	if PendingPayments != nil {
		PendingPayments.Dec(ctx)
	}
}

// RecordWebhookReceived records a webhook receipt metric
func RecordWebhookReceived(ctx context.Context, eventType string) {
	if WebhooksReceived != nil {
		WebhooksReceived.Inc(ctx,
			attribute.String("event_type", eventType),
		)
	}
}

// RecordWebhookProcessed records a successful webhook processing metric
func RecordWebhookProcessed(ctx context.Context, eventType string, durationSeconds float64) {
	if WebhooksProcessed != nil {
		WebhooksProcessed.Inc(ctx,
			attribute.String("event_type", eventType),
		)
	}
	if WebhookProcessingTime != nil {
		WebhookProcessingTime.Record(ctx, durationSeconds,
			attribute.String("event_type", eventType),
		)
	}
}

// RecordWebhookFailed records a webhook processing failure metric
func RecordWebhookFailed(ctx context.Context, eventType, reason string) {
	if WebhooksFailed != nil {
		WebhooksFailed.Inc(ctx,
			attribute.String("event_type", eventType),
			attribute.String("reason", reason),
		)
	}
}

// RecordError records an error by type and operation
func RecordError(ctx context.Context, errorType, operation string) {
	if ErrorsTotal != nil {
		ErrorsTotal.Inc(ctx,
			attribute.String("error_type", errorType),
			attribute.String("operation", operation),
		)
	}
}

// RecordRequestDuration records HTTP request duration and tracks slow requests
func RecordRequestDuration(ctx context.Context, operation string, durationSeconds float64) {
	if RequestDuration != nil {
		RequestDuration.Record(ctx, durationSeconds,
			attribute.String("operation", operation),
		)
	}
	// Track slow requests (>1s)
	if durationSeconds > 1.0 && SlowRequestsTotal != nil {
		SlowRequestsTotal.Inc(ctx,
			attribute.String("operation", operation),
		)
	}
}

// RecordSlowRequest explicitly records a slow request
func RecordSlowRequest(ctx context.Context, operation string, durationSeconds float64) {
	if SlowRequestsTotal != nil {
		SlowRequestsTotal.Inc(ctx,
			attribute.String("operation", operation),
			attribute.Float64("duration_seconds", durationSeconds),
		)
	}
}
