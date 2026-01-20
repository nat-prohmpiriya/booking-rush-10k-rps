package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// getEnvInt reads an integer from environment variable with a default value
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Rate limit per second per IP (0 = unlimited)
	RequestsPerSecond int
	// Burst size (token bucket capacity)
	BurstSize int
	// Whether to use Redis for distributed rate limiting
	UseRedis bool
	// Redis client (required if UseRedis is true)
	RedisClient *pkgredis.Client
	// Key prefix for Redis
	KeyPrefix string
	// Cleanup interval for local rate limiter
	CleanupInterval time.Duration
	// Entry TTL for local rate limiter
	EntryTTL time.Duration
}

// EndpointRateLimitConfig holds per-endpoint rate limiting configuration
type EndpointRateLimitConfig struct {
	// Path pattern (supports wildcards: /api/v1/*, /api/v1/events/:id)
	PathPattern string
	// HTTP methods this config applies to (empty = all methods)
	Methods []string
	// Rate limit per second per IP
	RequestsPerSecond int
	// Burst size (token bucket capacity)
	BurstSize int
}

// PerEndpointRateLimitConfig holds configuration for per-endpoint rate limiting
type PerEndpointRateLimitConfig struct {
	// Default rate limit for endpoints not in the list
	Default RateLimitConfig
	// Per-endpoint configurations (checked in order, first match wins)
	Endpoints []EndpointRateLimitConfig
	// Whether to use Redis for distributed rate limiting
	UseRedis bool
	// Redis client (required if UseRedis is true)
	RedisClient *pkgredis.Client
	// Key prefix for Redis
	KeyPrefix string
	// Cleanup interval for local rate limiter
	CleanupInterval time.Duration
	// Entry TTL for local rate limiter
	EntryTTL time.Duration
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 1000,         // 1000 req/s per IP
		BurstSize:         100,          // Allow burst of 100
		UseRedis:          false,        // Local by default for speed
		KeyPrefix:         "ratelimit:", // Redis key prefix
		CleanupInterval:   time.Minute,  // Cleanup stale entries every minute
		EntryTTL:          time.Minute,  // Entries expire after 1 minute of inactivity
	}
}

// rateLimitEntry tracks rate limit state for an IP
type rateLimitEntry struct {
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// LocalRateLimiter implements in-memory token bucket rate limiting
type LocalRateLimiter struct {
	config  RateLimitConfig
	entries sync.Map
	stop    chan struct{}

	// Metrics
	totalAllowed  uint64
	totalRejected uint64
}

// NewLocalRateLimiter creates a new local rate limiter
func NewLocalRateLimiter(config RateLimitConfig) *LocalRateLimiter {
	rl := &LocalRateLimiter{
		config: config,
		stop:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed
func (rl *LocalRateLimiter) Allow(key string) bool {
	allowed, _ := rl.AllowWithRemaining(key)
	return allowed
}

// AllowWithRemaining checks if a request should be allowed and returns remaining tokens
func (rl *LocalRateLimiter) AllowWithRemaining(key string) (bool, float64) {
	now := time.Now()

	// Get or create entry
	entry, _ := rl.entries.LoadOrStore(key, &rateLimitEntry{
		tokens:     float64(rl.config.BurstSize),
		lastUpdate: now,
	})
	e := entry.(*rateLimitEntry)

	e.mu.Lock()
	defer e.mu.Unlock()

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(e.lastUpdate).Seconds()
	tokensToAdd := elapsed * float64(rl.config.RequestsPerSecond)
	e.tokens = min(float64(rl.config.BurstSize), e.tokens+tokensToAdd)
	e.lastUpdate = now

	// Check if we have tokens available
	if e.tokens >= 1 {
		e.tokens--
		atomic.AddUint64(&rl.totalAllowed, 1)
		return true, e.tokens
	}

	atomic.AddUint64(&rl.totalRejected, 1)
	return false, e.tokens
}

// GetStats returns rate limiter statistics
func (rl *LocalRateLimiter) GetStats() (allowed, rejected uint64) {
	return atomic.LoadUint64(&rl.totalAllowed), atomic.LoadUint64(&rl.totalRejected)
}

// cleanup periodically removes stale entries
func (rl *LocalRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-rl.config.EntryTTL)
			rl.entries.Range(func(key, value interface{}) bool {
				e := value.(*rateLimitEntry)
				e.mu.Lock()
				if e.lastUpdate.Before(cutoff) {
					rl.entries.Delete(key)
				}
				e.mu.Unlock()
				return true
			})
		case <-rl.stop:
			return
		}
	}
}

// Stop stops the cleanup goroutine
func (rl *LocalRateLimiter) Stop() {
	close(rl.stop)
}

// RedisRateLimiter implements Redis-based distributed rate limiting
type RedisRateLimiter struct {
	config RateLimitConfig
	script string
}

// NewRedisRateLimiter creates a new Redis rate limiter
func NewRedisRateLimiter(config RateLimitConfig) *RedisRateLimiter {
	// Lua script for atomic token bucket rate limiting
	script := `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = 1

local data = redis.call("HMGET", key, "tokens", "last_update")
local tokens = tonumber(data[1]) or burst
local last_update = tonumber(data[2]) or now

-- Calculate tokens to add
local elapsed = now - last_update
local tokens_to_add = elapsed * rate
tokens = math.min(burst, tokens + tokens_to_add)

-- Check if request is allowed
if tokens >= requested then
    tokens = tokens - requested
    redis.call("HMSET", key, "tokens", tokens, "last_update", now)
    redis.call("EXPIRE", key, 60)
    return {1, tokens}
else
    redis.call("HMSET", key, "tokens", tokens, "last_update", now)
    redis.call("EXPIRE", key, 60)
    return {0, tokens}
end
`
	return &RedisRateLimiter{
		config: config,
		script: script,
	}
}

// Allow checks if a request should be allowed using Redis
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	allowed, _, err := rl.AllowWithRemaining(ctx, key, rl.config.RequestsPerSecond, rl.config.BurstSize)
	return allowed, err
}

// AllowWithRemaining checks if a request should be allowed and returns remaining tokens
func (rl *RedisRateLimiter) AllowWithRemaining(ctx context.Context, key string, rps, burst int) (bool, float64, error) {
	now := float64(time.Now().UnixNano()) / 1e9

	result := rl.config.RedisClient.Eval(ctx, rl.script,
		[]string{rl.config.KeyPrefix + key},
		float64(rps),
		float64(burst),
		now,
	)

	if result.Err() != nil {
		return false, 0, result.Err()
	}

	values, err := result.Slice()
	if err != nil {
		return false, 0, err
	}

	if len(values) < 2 {
		return false, 0, fmt.Errorf("unexpected result length: %d", len(values))
	}

	// Safe type conversion for allowed flag - handle multiple types Redis may return
	allowed := int64(0)
	switch v := values[0].(type) {
	case int64:
		allowed = v
	case int:
		allowed = int64(v)
	case float64:
		allowed = int64(v)
	case string:
		// Try parsing as integer first, then float
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			allowed = i
		} else if f, err := strconv.ParseFloat(v, 64); err == nil {
			allowed = int64(f)
		}
	case nil:
		// Redis returned null, default to 0 (not allowed)
		allowed = 0
	}

	// Safe type conversion for remaining tokens
	remaining := float64(0)
	switch v := values[1].(type) {
	case int64:
		remaining = float64(v)
	case int:
		remaining = float64(v)
	case float64:
		remaining = v
	case string:
		remaining, _ = strconv.ParseFloat(v, 64)
	case nil:
		// Redis returned null, default to 0
		remaining = 0
	}

	return allowed == 1, remaining, nil
}

// RateLimiter creates a rate limiting middleware
func RateLimiter(config RateLimitConfig) gin.HandlerFunc {
	var localLimiter *LocalRateLimiter
	var redisLimiter *RedisRateLimiter

	if config.UseRedis && config.RedisClient != nil {
		redisLimiter = NewRedisRateLimiter(config)
	} else {
		localLimiter = NewLocalRateLimiter(config)
	}

	return func(c *gin.Context) {
		ctx, span := telemetry.StartSpan(c.Request.Context(), "middleware.rate_limiter")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)

		// Get client IP as rate limit key
		clientIP := c.ClientIP()
		span.SetAttributes(attribute.String("client_ip", clientIP))

		var allowed bool
		var remaining int
		var err error

		startTime := time.Now()

		if redisLimiter != nil {
			allowed, err = redisLimiter.Allow(ctx, clientIP)
			if err != nil {
				// Fallback to allowing on Redis errors (fail open)
				allowed = true
			}
		} else {
			allowed = localLimiter.Allow(clientIP)
		}

		span.SetAttributes(attribute.Bool("allowed", allowed))

		// Calculate remaining tokens (approximation for headers)
		remaining = config.BurstSize - 1
		if !allowed {
			remaining = 0
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerSecond))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))

		if !allowed {
			span.SetStatus(codes.Error, "rate limit exceeded")

			// Calculate retry after (1 second default)
			retryAfter := 1
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			// Track rejection latency
			latency := time.Since(startTime)
			c.Header("X-RateLimit-Latency", latency.String())

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOO_MANY_REQUESTS",
					"message": "Rate limit exceeded. Please retry after " + strconv.Itoa(retryAfter) + " second(s).",
				},
			})
			return
		}

		span.SetStatus(codes.Ok, "")
		c.Next()
	}
}

// RateLimiterWithDefault creates a rate limiting middleware with default config
func RateLimiterWithDefault() gin.HandlerFunc {
	return RateLimiter(DefaultRateLimitConfig())
}

// GlobalRateLimiter implements global (non-per-IP) rate limiting for spike protection
type GlobalRateLimiter struct {
	maxConcurrent int64
	currentCount  int64
	mu            sync.Mutex
}

// NewGlobalRateLimiter creates a new global rate limiter
func NewGlobalRateLimiter(maxConcurrent int64) *GlobalRateLimiter {
	return &GlobalRateLimiter{
		maxConcurrent: maxConcurrent,
	}
}

// Acquire tries to acquire a slot
func (g *GlobalRateLimiter) Acquire() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.currentCount >= g.maxConcurrent {
		return false
	}
	g.currentCount++
	return true
}

// Release releases a slot
func (g *GlobalRateLimiter) Release() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.currentCount > 0 {
		g.currentCount--
	}
}

// CurrentCount returns the current concurrent request count
func (g *GlobalRateLimiter) CurrentCount() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.currentCount
}

// ConcurrencyLimiter creates a middleware that limits concurrent requests
func ConcurrencyLimiter(maxConcurrent int64) gin.HandlerFunc {
	limiter := NewGlobalRateLimiter(maxConcurrent)

	return func(c *gin.Context) {
		if !limiter.Acquire() {
			c.Header("Retry-After", "1")
			c.Header("X-Concurrency-Limit", strconv.FormatInt(maxConcurrent, 10))
			c.Header("X-Concurrency-Current", strconv.FormatInt(limiter.CurrentCount(), 10))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOO_MANY_REQUESTS",
					"message": "Server is at capacity. Please retry in a moment.",
				},
			})
			return
		}

		defer limiter.Release()
		c.Next()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// DefaultPerEndpointConfig returns sensible defaults for per-endpoint rate limiting
// Reads from environment variables:
// - RATE_LIMIT_REQUESTS_PER_MINUTE: default requests per minute (converted to per second)
// - RATE_LIMIT_BURST: default burst size
// - BOOKING_RATE_LIMIT_REQUESTS_PER_MINUTE: booking endpoint requests per minute
// - BOOKING_RATE_LIMIT_BURST: booking endpoint burst size
func DefaultPerEndpointConfig() PerEndpointRateLimitConfig {
	// Read from ENV with defaults (convert per-minute to per-second)
	defaultRPS := getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 60000) / 60     // default 1000/s
	defaultBurst := getEnvInt("RATE_LIMIT_BURST", 100)
	bookingRPS := getEnvInt("BOOKING_RATE_LIMIT_REQUESTS_PER_MINUTE", 6000) / 60  // default 100/s
	bookingBurst := getEnvInt("BOOKING_RATE_LIMIT_BURST", 20)

	return PerEndpointRateLimitConfig{
		Default: RateLimitConfig{
			RequestsPerSecond: defaultRPS,
			BurstSize:         defaultBurst,
		},
		Endpoints: []EndpointRateLimitConfig{
			// Critical booking endpoints - configurable via ENV
			{
				PathPattern:       "/api/v1/bookings",
				Methods:           []string{"POST"},
				RequestsPerSecond: bookingRPS,
				BurstSize:         bookingBurst,
			},
			{
				PathPattern:       "/api/v1/bookings/*/confirm",
				Methods:           []string{"POST"},
				RequestsPerSecond: bookingRPS / 2, // half of booking rate
				BurstSize:         bookingBurst / 2,
			},
			// Read-heavy endpoints - more generous limits
			{
				PathPattern:       "/api/v1/events",
				Methods:           []string{"GET"},
				RequestsPerSecond: defaultRPS * 2,
				BurstSize:         defaultBurst * 2,
			},
			{
				PathPattern:       "/api/v1/events/*",
				Methods:           []string{"GET"},
				RequestsPerSecond: defaultRPS * 2,
				BurstSize:         defaultBurst * 2,
			},
			// Auth endpoints - moderate limits
			{
				PathPattern:       "/api/v1/auth/*",
				Methods:           []string{"POST"},
				RequestsPerSecond: 20,
				BurstSize:         5,
			},
		},
		KeyPrefix:       "ratelimit:",
		CleanupInterval: time.Minute,
		EntryTTL:        time.Minute,
	}
}

// matchPath checks if a request path matches a pattern
// Supports wildcards: * matches any segment, ** matches any number of segments
func matchPath(pattern, path string) bool {
	// Exact match
	if pattern == path {
		return true
	}

	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	pi := 0 // pattern index
	for i := 0; i < len(pathParts); i++ {
		if pi >= len(patternParts) {
			return false
		}

		patternPart := patternParts[pi]

		// ** matches any remaining path
		if patternPart == "**" {
			return true
		}

		// * matches any single segment
		if patternPart == "*" {
			pi++
			continue
		}

		// :param matches any single segment (Gin-style parameter)
		if strings.HasPrefix(patternPart, ":") {
			pi++
			continue
		}

		// Exact segment match
		if patternPart != pathParts[i] {
			return false
		}
		pi++
	}

	// Check if we've matched the entire pattern
	return pi == len(patternParts)
}

// containsMethod checks if a method is in the list (empty list matches all)
func containsMethod(methods []string, method string) bool {
	if len(methods) == 0 {
		return true
	}
	for _, m := range methods {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

// findEndpointConfig finds the matching endpoint configuration
func (c *PerEndpointRateLimitConfig) findEndpointConfig(method, path string) (int, int) {
	for _, endpoint := range c.Endpoints {
		if matchPath(endpoint.PathPattern, path) && containsMethod(endpoint.Methods, method) {
			return endpoint.RequestsPerSecond, endpoint.BurstSize
		}
	}
	return c.Default.RequestsPerSecond, c.Default.BurstSize
}

// PerEndpointRateLimiter creates a middleware with per-endpoint rate limiting
func PerEndpointRateLimiter(config PerEndpointRateLimitConfig) gin.HandlerFunc {
	var localLimiters sync.Map  // map[string]*LocalRateLimiter for different rate configs
	var redisLimiter *RedisRateLimiter

	if config.UseRedis && config.RedisClient != nil {
		// For Redis, we use a single limiter but adjust the key to include rate info
		redisLimiter = NewRedisRateLimiter(RateLimitConfig{
			RedisClient: config.RedisClient,
			KeyPrefix:   config.KeyPrefix,
		})
	}

	// getLimiter returns or creates a local rate limiter for the given rate config
	getLimiter := func(rps, burst int) *LocalRateLimiter {
		key := fmt.Sprintf("%d:%d", rps, burst)
		if limiter, ok := localLimiters.Load(key); ok {
			return limiter.(*LocalRateLimiter)
		}
		limiter := NewLocalRateLimiter(RateLimitConfig{
			RequestsPerSecond: rps,
			BurstSize:         burst,
			CleanupInterval:   config.CleanupInterval,
			EntryTTL:          config.EntryTTL,
		})
		actual, _ := localLimiters.LoadOrStore(key, limiter)
		return actual.(*LocalRateLimiter)
	}

	return func(c *gin.Context) {
		// Add panic recovery for rate limiter
		defer func() {
			if r := recover(); r != nil {
				// Log the panic and allow request to proceed (fail open)
				span := telemetry.SpanFromContext(c.Request.Context())
				if span != nil {
					span.SetStatus(codes.Error, fmt.Sprintf("rate limiter panic: %v", r))
					span.RecordError(fmt.Errorf("panic: %v", r))
				}
				// Allow request to proceed on panic
				c.Next()
			}
		}()

		ctx, span := telemetry.StartSpan(c.Request.Context(), "middleware.per_endpoint_rate_limiter")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)

		path := c.FullPath() // Use registered path pattern instead of actual path
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		// Get client IP as rate limit key
		clientIP := c.ClientIP()

		// Get rate limit config for this endpoint
		rps, burst := config.findEndpointConfig(method, path)

		span.SetAttributes(
			attribute.String("client_ip", clientIP),
			attribute.String("path", path),
			attribute.Int("rps", rps),
			attribute.Int("burst", burst),
		)

		// Skip rate limiting if unlimited
		if rps <= 0 {
			span.SetStatus(codes.Ok, "")
			c.Next()
			return
		}

		var allowed bool
		var remainingTokens float64

		if redisLimiter != nil {
			// For Redis, include the rate config in the key for per-endpoint limits
			redisKey := fmt.Sprintf("%s:%d:%d", clientIP, rps, burst)
			var err error
			allowed, remainingTokens, err = redisLimiter.AllowWithRemaining(ctx, redisKey, rps, burst)
			if err != nil {
				// Fallback to allowing on Redis errors (fail open)
				allowed = true
				remainingTokens = float64(burst)
			}
		} else {
			limiter := getLimiter(rps, burst)
			allowed, remainingTokens = limiter.AllowWithRemaining(clientIP)
		}

		span.SetAttributes(attribute.Bool("allowed", allowed))

		// Calculate remaining (at least 0)
		remaining := int(remainingTokens)
		if remaining < 0 {
			remaining = 0
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(rps))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))
		c.Header("X-RateLimit-Burst", strconv.Itoa(burst))

		if !allowed {
			span.SetStatus(codes.Error, "rate limit exceeded")

			// Calculate retry after based on how many tokens we need and refill rate
			retryAfterSeconds := 1.0
			if rps > 0 {
				tokensNeeded := 1.0 - remainingTokens
				if tokensNeeded > 0 {
					retryAfterSeconds = tokensNeeded / float64(rps)
				}
			}
			retryAfter := int(retryAfterSeconds)
			if retryAfter < 1 {
				retryAfter = 1
			}

			c.Header("Retry-After", strconv.Itoa(retryAfter))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOO_MANY_REQUESTS",
					"message": "Rate limit exceeded. Please retry after " + strconv.Itoa(retryAfter) + " second(s).",
				},
			})
			return
		}

		span.SetStatus(codes.Ok, "")
		c.Next()
	}
}
