package telemetry

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the name of the Gin tracer
	TracerName = "gin-server"

	// TraceIDHeader is the header key for trace ID
	TraceIDHeader = "X-Trace-ID"

	// SpanIDHeader is the header key for span ID
	SpanIDHeader = "X-Span-ID"
)

// TracingMiddleware returns a Gin middleware for automatic tracing
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(TracerName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// Extract trace context from incoming request
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Create span name from route
		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.URL.Path
		}
		spanName = fmt.Sprintf("%s %s", c.Request.Method, spanName)

		// Start span
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Request.Method),
				semconv.HTTPURL(c.Request.URL.String()),
				semconv.HTTPRoute(c.FullPath()),
				semconv.NetHostName(c.Request.Host),
				semconv.UserAgentOriginal(c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
			),
		)
		defer span.End()

		// Add trace ID to response header
		if span.SpanContext().HasTraceID() {
			traceID := span.SpanContext().TraceID().String()
			c.Header(TraceIDHeader, traceID)
			c.Set("trace_id", traceID)
		}
		if span.SpanContext().HasSpanID() {
			spanID := span.SpanContext().SpanID().String()
			c.Header(SpanIDHeader, spanID)
			c.Set("span_id", spanID)
		}

		// Set request context with span
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Set response attributes
		status := c.Writer.Status()
		span.SetAttributes(
			semconv.HTTPStatusCode(status),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Record error if any
		if len(c.Errors) > 0 {
			span.RecordError(c.Errors.Last())
			span.SetAttributes(attribute.String("error.message", c.Errors.String()))
		}

		// Mark span as error if 5xx
		if status >= 500 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}

// InjectTraceContext injects trace context into outgoing HTTP headers
func InjectTraceContext(ctx *gin.Context) map[string]string {
	headers := make(map[string]string)
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx.Request.Context(), propagation.MapCarrier(headers))
	return headers
}
