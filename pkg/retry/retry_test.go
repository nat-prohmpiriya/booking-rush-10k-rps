package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", config.MaxRetries)
	}

	if config.InitialInterval != 1*time.Second {
		t.Errorf("InitialInterval = %v, want 1s", config.InitialInterval)
	}

	if config.MaxInterval != 30*time.Second {
		t.Errorf("MaxInterval = %v, want 30s", config.MaxInterval)
	}

	if config.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", config.Multiplier)
	}

	if config.JitterFactor != 0.1 {
		t.Errorf("JitterFactor = %f, want 0.1", config.JitterFactor)
	}
}

func TestNew_WithNilConfig(t *testing.T) {
	retrier := New(nil)
	if retrier == nil {
		t.Fatal("New(nil) returned nil")
	}

	if retrier.config.InitialInterval != 1*time.Second {
		t.Errorf("Default InitialInterval = %v, want 1s", retrier.config.InitialInterval)
	}
}

func TestNew_WithZeroValues(t *testing.T) {
	config := &Config{}
	retrier := New(config)

	if retrier.config.InitialInterval != 1*time.Second {
		t.Errorf("InitialInterval = %v, want 1s (default)", retrier.config.InitialInterval)
	}

	if retrier.config.MaxInterval != 30*time.Second {
		t.Errorf("MaxInterval = %v, want 30s (default)", retrier.config.MaxInterval)
	}

	if retrier.config.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0 (default)", retrier.config.Multiplier)
	}
}

func TestRetrier_Do_Success(t *testing.T) {
	config := &Config{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return nil
	}

	result := retrier.Do(context.Background(), op)

	if result.Err != nil {
		t.Errorf("Err = %v, want nil", result.Err)
	}

	if result.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", result.Attempts)
	}

	if attempts != 1 {
		t.Errorf("Operation called %d times, want 1", attempts)
	}
}

func TestRetrier_Do_SuccessAfterRetries(t *testing.T) {
	config := &Config{
		MaxRetries:      5,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	result := retrier.Do(context.Background(), op)

	if result.Err != nil {
		t.Errorf("Err = %v, want nil", result.Err)
	}

	if result.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3", result.Attempts)
	}

	if attempts != 3 {
		t.Errorf("Operation called %d times, want 3", attempts)
	}
}

func TestRetrier_Do_MaxRetriesExceeded(t *testing.T) {
	config := &Config{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	attempts := 0
	expectedErr := errors.New("persistent error")
	op := func(ctx context.Context) error {
		attempts++
		return expectedErr
	}

	result := retrier.Do(context.Background(), op)

	if !errors.Is(result.Err, ErrMaxRetriesExceeded) {
		t.Errorf("Err = %v, want ErrMaxRetriesExceeded", result.Err)
	}

	if result.LastError == nil || result.LastError.Error() != expectedErr.Error() {
		t.Errorf("LastError = %v, want %v", result.LastError, expectedErr)
	}

	// Initial attempt + 3 retries = 4 total
	if result.Attempts != 4 {
		t.Errorf("Attempts = %d, want 4", result.Attempts)
	}

	if attempts != 4 {
		t.Errorf("Operation called %d times, want 4", attempts)
	}
}

func TestRetrier_Do_PermanentError(t *testing.T) {
	config := &Config{
		MaxRetries:      5,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	attempts := 0
	permErr := errors.New("permanent error")
	op := func(ctx context.Context) error {
		attempts++
		return Permanent(permErr)
	}

	result := retrier.Do(context.Background(), op)

	if result.Err == nil || result.Err.Error() != permErr.Error() {
		t.Errorf("Err = %v, want %v", result.Err, permErr)
	}

	// Should stop immediately, no retries
	if result.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", result.Attempts)
	}

	if attempts != 1 {
		t.Errorf("Operation called %d times, want 1", attempts)
	}
}

func TestRetrier_Do_ContextCanceled(t *testing.T) {
	config := &Config{
		MaxRetries:      10,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return errors.New("error")
	}

	result := retrier.Do(ctx, op)

	if !errors.Is(result.Err, ErrContextCanceled) {
		t.Errorf("Err = %v, want ErrContextCanceled", result.Err)
	}

	// Should have attempted at least twice before context was canceled
	if result.Attempts < 2 {
		t.Errorf("Attempts = %d, want >= 2", result.Attempts)
	}
}

func TestRetrier_Do_ContextTimeout(t *testing.T) {
	config := &Config{
		MaxRetries:      10,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	op := func(ctx context.Context) error {
		return errors.New("error")
	}

	result := retrier.Do(ctx, op)

	if !errors.Is(result.Err, ErrContextCanceled) {
		t.Errorf("Err = %v, want ErrContextCanceled", result.Err)
	}
}

func TestRetrier_DoWithCallback(t *testing.T) {
	config := &Config{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	attempts := 0
	callbackCalls := 0
	op := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("error")
		}
		return nil
	}

	callback := func(attempt int, err error, nextInterval time.Duration) {
		callbackCalls++
	}

	result := retrier.DoWithCallback(context.Background(), op, callback)

	if result.Err != nil {
		t.Errorf("Err = %v, want nil", result.Err)
	}

	// Callback should be called twice (before retry 2 and 3)
	if callbackCalls != 2 {
		t.Errorf("Callback called %d times, want 2", callbackCalls)
	}
}

func TestCalculateInterval_ExponentialBackoff(t *testing.T) {
	config := &Config{
		MaxRetries:      5,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0, // No jitter for predictable testing
	}

	retrier := New(config)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},  // 1 * 2^0 = 1s
		{1, 2 * time.Second},  // 1 * 2^1 = 2s
		{2, 4 * time.Second},  // 1 * 2^2 = 4s
		{3, 8 * time.Second},  // 1 * 2^3 = 8s
		{4, 16 * time.Second}, // 1 * 2^4 = 16s
		{5, 30 * time.Second}, // 1 * 2^5 = 32s, capped at 30s
		{6, 30 * time.Second}, // capped at max
	}

	for _, tt := range tests {
		got := retrier.calculateInterval(tt.attempt)
		if got != tt.expected {
			t.Errorf("calculateInterval(%d) = %v, want %v", tt.attempt, got, tt.expected)
		}
	}
}

func TestCalculateInterval_WithJitter(t *testing.T) {
	config := &Config{
		MaxRetries:      5,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.1, // ±10% jitter
	}

	retrier := New(config)

	// Test that jitter produces varying results
	baseInterval := 1 * time.Second
	minExpected := time.Duration(float64(baseInterval) * 0.9)
	maxExpected := time.Duration(float64(baseInterval) * 1.1)

	results := make(map[time.Duration]bool)
	for i := 0; i < 100; i++ {
		interval := retrier.calculateInterval(0)
		results[interval] = true

		if interval < minExpected || interval > maxExpected {
			t.Errorf("calculateInterval(0) = %v, want between %v and %v", interval, minExpected, maxExpected)
		}
	}

	// With jitter, we should see some variation
	if len(results) < 3 {
		t.Errorf("Expected more variation with jitter, got %d unique values", len(results))
	}
}

func TestRetryable_And_Permanent(t *testing.T) {
	// Test Retryable
	err := errors.New("test error")
	retryableErr := Retryable(err)

	var re *RetryableError
	if !errors.As(retryableErr, &re) {
		t.Error("Retryable error should be RetryableError")
	}

	if re.Error() != err.Error() {
		t.Errorf("RetryableError.Error() = %v, want %v", re.Error(), err.Error())
	}

	if !errors.Is(re.Unwrap(), err) {
		t.Error("RetryableError.Unwrap() should return original error")
	}

	// Test Permanent
	permErr := Permanent(err)

	var pe *PermanentError
	if !errors.As(permErr, &pe) {
		t.Error("Permanent error should be PermanentError")
	}

	if pe.Error() != err.Error() {
		t.Errorf("PermanentError.Error() = %v, want %v", pe.Error(), err.Error())
	}

	if !errors.Is(pe.Unwrap(), err) {
		t.Error("PermanentError.Unwrap() should return original error")
	}

	// Test nil handling
	if Retryable(nil) != nil {
		t.Error("Retryable(nil) should return nil")
	}

	if Permanent(nil) != nil {
		t.Error("Permanent(nil) should return nil")
	}
}

func TestDo_ConvenienceFunction(t *testing.T) {
	config := &Config{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return nil
	}

	result := Do(context.Background(), config, op)

	if result.Err != nil {
		t.Errorf("Err = %v, want nil", result.Err)
	}

	if attempts != 1 {
		t.Errorf("Operation called %d times, want 1", attempts)
	}
}

func TestWithRetry(t *testing.T) {
	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return nil
	}

	wrappedOp := WithRetry(op)
	err := wrappedOp(context.Background())

	if err != nil {
		t.Errorf("Err = %v, want nil", err)
	}

	if attempts != 1 {
		t.Errorf("Operation called %d times, want 1", attempts)
	}
}

func TestWithRetryConfig(t *testing.T) {
	config := &Config{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("error")
		}
		return nil
	}

	wrappedOp := WithRetryConfig(config, op)
	err := wrappedOp(context.Background())

	if err != nil {
		t.Errorf("Err = %v, want nil", err)
	}

	if attempts != 3 {
		t.Errorf("Operation called %d times, want 3", attempts)
	}
}

func TestResult_TotalDuration(t *testing.T) {
	config := &Config{
		MaxRetries:      2,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("error")
		}
		return nil
	}

	result := retrier.Do(context.Background(), op)

	// Should take at least the backoff time (50ms + 100ms = 150ms)
	if result.TotalDuration < 100*time.Millisecond {
		t.Errorf("TotalDuration = %v, want >= 100ms", result.TotalDuration)
	}
}

func TestRetrier_NoRetries(t *testing.T) {
	config := &Config{
		MaxRetries:      0, // No retries
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}

	retrier := New(config)

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return errors.New("error")
	}

	result := retrier.Do(context.Background(), op)

	if !errors.Is(result.Err, ErrMaxRetriesExceeded) {
		t.Errorf("Err = %v, want ErrMaxRetriesExceeded", result.Err)
	}

	// Only initial attempt, no retries
	if result.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", result.Attempts)
	}

	if attempts != 1 {
		t.Errorf("Operation called %d times, want 1", attempts)
	}
}
