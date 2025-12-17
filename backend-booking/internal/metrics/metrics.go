package metrics

import (
	"context"
	"sync"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

var (
	// Booking counters
	BookingsReserved  *telemetry.Counter
	BookingsConfirmed *telemetry.Counter
	BookingsExpired   *telemetry.Counter
	BookingsFailed    *telemetry.Counter
	BookingsCancelled *telemetry.Counter

	// Queue counters
	QueueJoined *telemetry.Counter
	QueueLeft   *telemetry.Counter

	// Error tracking counters
	ErrorsTotal      *telemetry.Counter
	SlowRequestsTotal *telemetry.Counter

	// Histograms
	ReservationDuration *telemetry.Histogram
	QueueWaitTime       *telemetry.Histogram
	RequestDuration     *telemetry.Histogram

	// Gauges
	ActiveReservations *telemetry.UpDownCounter
	QueueDepth         *telemetry.UpDownCounter

	initOnce sync.Once
	initErr  error
)

// Init initializes all booking metrics
func Init() error {
	initOnce.Do(func() {
		initErr = initMetrics()
	})
	return initErr
}

func initMetrics() error {
	var err error

	// Booking counters
	BookingsReserved, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "booking_reservations_total",
		Description: "Total number of seat reservations created",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	BookingsConfirmed, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "booking_confirmations_total",
		Description: "Total number of bookings confirmed",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	BookingsExpired, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "booking_expirations_total",
		Description: "Total number of expired reservations",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	BookingsFailed, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "booking_failures_total",
		Description: "Total number of failed bookings",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	BookingsCancelled, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "booking_cancellations_total",
		Description: "Total number of cancelled bookings",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	// Queue counters
	QueueJoined, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "queue_joins_total",
		Description: "Total number of users joined queue",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	QueueLeft, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "queue_leaves_total",
		Description: "Total number of users left queue",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	// Histograms with custom buckets for latency
	ReservationDuration, err = telemetry.NewHistogramWithBuckets(telemetry.MetricOpts{
		Name:        "booking_reservation_duration_seconds",
		Description: "Duration from reservation to confirmation",
		Unit:        "s",
	}, []float64{1, 5, 10, 30, 60, 120, 300, 600, 900}) // 1s to 15min
	if err != nil {
		return err
	}

	QueueWaitTime, err = telemetry.NewHistogramWithBuckets(telemetry.MetricOpts{
		Name:        "queue_wait_time_seconds",
		Description: "Time spent waiting in queue",
		Unit:        "s",
	}, []float64{1, 5, 10, 30, 60, 120, 300, 600}) // 1s to 10min
	if err != nil {
		return err
	}

	// Request duration histogram for latency tracking (p50, p90, p99)
	RequestDuration, err = telemetry.NewHistogramWithBuckets(telemetry.MetricOpts{
		Name:        "booking_request_duration_seconds",
		Description: "HTTP request duration in seconds",
		Unit:        "s",
	}, []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}) // 5ms to 10s
	if err != nil {
		return err
	}

	// Error tracking
	ErrorsTotal, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "booking_errors_total",
		Description: "Total number of errors by type",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	SlowRequestsTotal, err = telemetry.NewCounter(telemetry.MetricOpts{
		Name:        "booking_slow_requests_total",
		Description: "Total number of slow requests (>1s)",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	// Up-down counters for current state
	ActiveReservations, err = telemetry.NewUpDownCounter(telemetry.MetricOpts{
		Name:        "booking_active_reservations",
		Description: "Current number of active (pending) reservations",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	QueueDepth, err = telemetry.NewUpDownCounter(telemetry.MetricOpts{
		Name:        "queue_depth",
		Description: "Current number of users in queue",
		Unit:        "1",
	})
	if err != nil {
		return err
	}

	return nil
}

// RecordReservation records a reservation metric
func RecordReservation(ctx context.Context, eventID, userID, zoneID string, quantity int) {
	if BookingsReserved != nil {
		BookingsReserved.Inc(ctx,
			attribute.String("event_id", eventID),
			attribute.String("zone_id", zoneID),
			attribute.Int("quantity", quantity),
		)
	}
	if ActiveReservations != nil {
		ActiveReservations.Inc(ctx)
	}
}

// RecordConfirmation records a booking confirmation metric
func RecordConfirmation(ctx context.Context, eventID, userID string, durationSeconds float64) {
	if BookingsConfirmed != nil {
		BookingsConfirmed.Inc(ctx,
			attribute.String("event_id", eventID),
		)
	}
	if ReservationDuration != nil {
		ReservationDuration.Record(ctx, durationSeconds,
			attribute.String("event_id", eventID),
		)
	}
	if ActiveReservations != nil {
		ActiveReservations.Dec(ctx)
	}
}

// RecordExpiration records a reservation expiration metric
func RecordExpiration(ctx context.Context, eventID string, count int64) {
	if BookingsExpired != nil {
		BookingsExpired.Add(ctx, count,
			attribute.String("event_id", eventID),
		)
	}
	if ActiveReservations != nil {
		ActiveReservations.Add(ctx, -count)
	}
}

// RecordFailure records a booking failure metric
func RecordFailure(ctx context.Context, eventID, reason string) {
	if BookingsFailed != nil {
		BookingsFailed.Inc(ctx,
			attribute.String("event_id", eventID),
			attribute.String("reason", reason),
		)
	}
}

// RecordCancellation records a booking cancellation metric
func RecordCancellation(ctx context.Context, eventID string) {
	if BookingsCancelled != nil {
		BookingsCancelled.Inc(ctx,
			attribute.String("event_id", eventID),
		)
	}
	if ActiveReservations != nil {
		ActiveReservations.Dec(ctx)
	}
}

// RecordQueueJoin records a queue join metric
func RecordQueueJoin(ctx context.Context, eventID string) {
	if QueueJoined != nil {
		QueueJoined.Inc(ctx,
			attribute.String("event_id", eventID),
		)
	}
	if QueueDepth != nil {
		QueueDepth.Inc(ctx)
	}
}

// RecordQueueLeave records a queue leave metric
func RecordQueueLeave(ctx context.Context, eventID string, waitTimeSeconds float64) {
	if QueueLeft != nil {
		QueueLeft.Inc(ctx,
			attribute.String("event_id", eventID),
		)
	}
	if QueueWaitTime != nil {
		QueueWaitTime.Record(ctx, waitTimeSeconds,
			attribute.String("event_id", eventID),
		)
	}
	if QueueDepth != nil {
		QueueDepth.Dec(ctx)
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
