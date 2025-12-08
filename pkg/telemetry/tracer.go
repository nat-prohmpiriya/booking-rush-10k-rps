package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry configuration
type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	CollectorAddr  string
}

// Telemetry holds the tracer provider and tracer
type Telemetry struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	config   *Config
}

var globalTelemetry *Telemetry

// Init initializes OpenTelemetry with the given configuration
func Init(ctx context.Context, cfg *Config) (*Telemetry, error) {
	if cfg == nil || !cfg.Enabled {
		// Return a no-op tracer if disabled
		globalTelemetry = &Telemetry{
			tracer: otel.Tracer(cfg.ServiceName),
			config: cfg,
		}
		return globalTelemetry, nil
	}

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.CollectorAddr),
		otlptracegrpc.WithInsecure(), // Use insecure for internal network
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create TracerProvider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample all for now
	)

	// Set global provider and propagator
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	globalTelemetry = &Telemetry{
		provider: provider,
		tracer:   provider.Tracer(cfg.ServiceName),
		config:   cfg,
	}

	return globalTelemetry, nil
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	if globalTelemetry != nil && globalTelemetry.provider != nil {
		return globalTelemetry.provider.Shutdown(ctx)
	}
	return nil
}

// Get returns the global telemetry instance
func Get() *Telemetry {
	return globalTelemetry
}

// Tracer returns the tracer
func (t *Telemetry) Tracer() trace.Tracer {
	return t.tracer
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if globalTelemetry == nil || globalTelemetry.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return globalTelemetry.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// GetTraceID returns the trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().HasTraceID() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// GetSpanID returns the span ID from context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().HasSpanID() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanError records an error on the current span
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}
