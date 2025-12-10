package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Common errors
var (
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrContextCanceled    = errors.New("context canceled during retry")
)

// Config contains retry configuration
type Config struct {
	// MaxRetries is the maximum number of retry attempts (0 = no retries, just initial attempt)
	MaxRetries int
	// InitialInterval is the initial backoff interval (default: 1s)
	InitialInterval time.Duration
	// MaxInterval is the maximum backoff interval (default: 30s)
	MaxInterval time.Duration
	// Multiplier is the factor to multiply the interval by after each retry (default: 2.0)
	Multiplier float64
	// JitterFactor is the random jitter factor (0-1) to add/subtract from interval (default: 0.1)
	// e.g., 0.1 means ±10% jitter
	JitterFactor float64
}

// DefaultConfig returns default retry configuration
// Uses exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (capped)
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:      5,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.1, // ±10% jitter
	}
}

// Operation is the function to be retried
type Operation func(ctx context.Context) error

// RetryableError wraps an error indicating it should be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// Retryable marks an error as retryable
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// PermanentError wraps an error indicating it should NOT be retried
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// Permanent marks an error as permanent (not retryable)
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// Result contains the result of a retry operation
type Result struct {
	// Err is the final error (nil if successful)
	Err error
	// Attempts is the total number of attempts made (including initial)
	Attempts int
	// TotalDuration is the total time spent including waits
	TotalDuration time.Duration
	// LastError is the error from the last attempt
	LastError error
}

// Retrier handles retry logic with exponential backoff
type Retrier struct {
	config *Config
}

// New creates a new Retrier with the given configuration
func New(config *Config) *Retrier {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply defaults for zero values
	if config.InitialInterval <= 0 {
		config.InitialInterval = 1 * time.Second
	}
	if config.MaxInterval <= 0 {
		config.MaxInterval = 30 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}
	if config.JitterFactor < 0 {
		config.JitterFactor = 0
	}
	if config.JitterFactor > 1 {
		config.JitterFactor = 1
	}

	return &Retrier{
		config: config,
	}
}

// Do executes the operation with retry logic
func (r *Retrier) Do(ctx context.Context, op Operation) *Result {
	return r.DoWithCallback(ctx, op, nil)
}

// RetryCallback is called before each retry attempt
type RetryCallback func(attempt int, err error, nextInterval time.Duration)

// DoWithCallback executes the operation with retry logic and a callback
func (r *Retrier) DoWithCallback(ctx context.Context, op Operation, callback RetryCallback) *Result {
	startTime := time.Now()
	result := &Result{}
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Check context before attempting
		if ctx.Err() != nil {
			result.Err = ErrContextCanceled
			result.LastError = lastErr
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Execute operation
		err := op(ctx)
		if err == nil {
			// Success
			result.TotalDuration = time.Since(startTime)
			return result
		}

		lastErr = err

		// Check if error is permanent (not retryable)
		var permErr *PermanentError
		if errors.As(err, &permErr) {
			result.Err = permErr.Err
			result.LastError = permErr.Err
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Last attempt, no more retries
		if attempt == r.config.MaxRetries {
			break
		}

		// Calculate backoff interval
		interval := r.calculateInterval(attempt)

		// Invoke callback before waiting
		if callback != nil {
			callback(attempt+1, err, interval)
		}

		// Wait for backoff interval
		select {
		case <-ctx.Done():
			result.Err = ErrContextCanceled
			result.LastError = lastErr
			result.TotalDuration = time.Since(startTime)
			return result
		case <-time.After(interval):
			// Continue to next retry
		}
	}

	result.Err = ErrMaxRetriesExceeded
	result.LastError = lastErr
	result.TotalDuration = time.Since(startTime)
	return result
}

// calculateInterval calculates the backoff interval for a given attempt
func (r *Retrier) calculateInterval(attempt int) time.Duration {
	// Calculate exponential backoff: initial * multiplier^attempt
	interval := float64(r.config.InitialInterval) * math.Pow(r.config.Multiplier, float64(attempt))

	// Apply jitter to prevent thundering herd
	if r.config.JitterFactor > 0 {
		jitter := interval * r.config.JitterFactor
		// Add random value between -jitter and +jitter
		interval = interval + (rand.Float64()*2-1)*jitter
	}

	// Cap at max interval
	if interval > float64(r.config.MaxInterval) {
		interval = float64(r.config.MaxInterval)
	}

	// Ensure positive
	if interval < 0 {
		interval = float64(r.config.InitialInterval)
	}

	return time.Duration(interval)
}

// Do is a convenience function that creates a retrier and executes the operation
func Do(ctx context.Context, config *Config, op Operation) *Result {
	return New(config).Do(ctx, op)
}

// DoWithCallback is a convenience function with callback support
func DoWithCallback(ctx context.Context, config *Config, op Operation, callback RetryCallback) *Result {
	return New(config).DoWithCallback(ctx, op, callback)
}

// WithRetry wraps an operation to be retried with default config
func WithRetry(op Operation) Operation {
	return func(ctx context.Context) error {
		result := Do(ctx, DefaultConfig(), op)
		if result.Err != nil {
			return result.Err
		}
		return nil
	}
}

// WithRetryConfig wraps an operation to be retried with custom config
func WithRetryConfig(config *Config, op Operation) Operation {
	return func(ctx context.Context) error {
		result := Do(ctx, config, op)
		if result.Err != nil {
			return result.Err
		}
		return nil
	}
}
