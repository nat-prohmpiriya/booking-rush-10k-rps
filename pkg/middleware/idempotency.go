package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
	"github.com/redis/go-redis/v9"
)

const (
	// IdempotencyKeyHeader is the header name for idempotency key
	IdempotencyKeyHeader = "X-Idempotency-Key"
	// ContextKeyIdempotencyKey is the context key for idempotency key
	ContextKeyIdempotencyKey = "idempotency_key"
	// Default TTL for idempotency records (5 minutes - short-lived for network retries)
	DefaultIdempotencyTTL = 5 * time.Minute
	// Redis key prefix for idempotency
	IdempotencyKeyPrefix = "idempotency:"
)

var (
	ErrMissingIdempotencyKey = errors.New("missing idempotency key")
	ErrDuplicateRequest      = errors.New("duplicate request")
	ErrRequestInProgress     = errors.New("request in progress")
)

// IdempotencyStatus represents the status of an idempotency record
type IdempotencyStatus string

const (
	StatusProcessing IdempotencyStatus = "processing"
	StatusCompleted  IdempotencyStatus = "completed"
)

// IdempotencyRecord stores the state of an idempotent request
type IdempotencyRecord struct {
	Key          string            `json:"key"`
	Status       IdempotencyStatus `json:"status"`
	RequestHash  string            `json:"request_hash"`
	ResponseCode int               `json:"response_code"`
	ResponseBody string            `json:"response_body"`
	CreatedAt    time.Time         `json:"created_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// RedisClient interface for Redis operations
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// IdempotencyConfig holds configuration for idempotency middleware
type IdempotencyConfig struct {
	// Redis client for storing idempotency records
	Redis RedisClient
	// TTL for COMPLETED idempotency records (default: 24 hours)
	TTL time.Duration
	// TTL for PROCESSING idempotency records (default: 60 seconds)
	ProcessingTTL time.Duration
	// KeyExtractor extracts idempotency key from request (default: from header)
	KeyExtractor func(*gin.Context) string
	// SkipPaths is a list of paths that should skip idempotency check
	SkipPaths []string
	// Methods that require idempotency (default: POST, PUT, PATCH, DELETE)
	RequiredMethods []string
	// IncludeBodyInHash includes request body in the hash (default: true)
	IncludeBodyInHash bool
	// IncludePathInHash includes request path in the hash (default: true)
	IncludePathInHash bool
	// IncludeUserInHash includes user ID in the hash (default: true)
	IncludeUserInHash bool
}

// DefaultIdempotencyConfig returns default configuration
func DefaultIdempotencyConfig(redis RedisClient) *IdempotencyConfig {
	return &IdempotencyConfig{
		Redis:             redis,
		TTL:               DefaultIdempotencyTTL, // 24h
		ProcessingTTL:     60 * time.Second,      // 60s (Dual-TTL Strategy)
		KeyExtractor:      defaultKeyExtractor,
		SkipPaths:         []string{},
		RequiredMethods:   []string{"POST", "PUT", "PATCH", "DELETE"},
		IncludeBodyInHash: true,
		IncludePathInHash: true,
		IncludeUserInHash: true,
	}
}

// defaultKeyExtractor extracts idempotency key from header
func defaultKeyExtractor(c *gin.Context) string {
	return c.GetHeader(IdempotencyKeyHeader)
}

// IdempotencyMiddleware creates a new idempotency middleware
func IdempotencyMiddleware(config *IdempotencyConfig) gin.HandlerFunc {
	// Set default ProcessingTTL if not set
	if config.ProcessingTTL == 0 {
		config.ProcessingTTL = 60 * time.Second
	}

	return func(c *gin.Context) {
		// Check if path should skip idempotency check
		for _, path := range config.SkipPaths {
			if matchPath(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// Check if method requires idempotency
		if !isMethodRequired(c.Request.Method, config.RequiredMethods) {
			c.Next()
			return
		}

		// Extract idempotency key
		idempotencyKey := config.KeyExtractor(c)
		if idempotencyKey == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, response.Error("MISSING_IDEMPOTENCY_KEY", "X-Idempotency-Key header is required"))
			return
		}

		// Store key in context
		c.Set(ContextKeyIdempotencyKey, idempotencyKey)

		// Read request body for hashing
		var bodyBytes []byte
		if c.Request.Body != nil && config.IncludeBodyInHash {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			// Restore body for downstream handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Generate request hash
		requestHash := generateRequestHash(c, bodyBytes, config)

		// Build Redis key
		redisKey := IdempotencyKeyPrefix + idempotencyKey

		ctx := c.Request.Context()

		// Try to get existing record
		existingRecord, err := getIdempotencyRecord(ctx, config.Redis, redisKey)
		if err != nil && !errors.Is(err, redis.Nil) {
			// Redis error - continue without idempotency (fail open)
			c.Next()
			return
		}

		if existingRecord != nil {
			// Check if request hash matches
			if existingRecord.RequestHash != requestHash {
				c.AbortWithStatusJSON(http.StatusUnprocessableEntity, response.Error("IDEMPOTENCY_KEY_REUSED", "Idempotency key already used with different request"))
				return
			}

			// Check status
			if existingRecord.Status == StatusProcessing {
				c.AbortWithStatusJSON(http.StatusConflict, response.Error("REQUEST_IN_PROGRESS", "A request with this idempotency key is already being processed"))
				return
			}

			// Return cached response
			c.Data(existingRecord.ResponseCode, "application/json", []byte(existingRecord.ResponseBody))
			c.Abort()
			return
		}

		// Create new processing record
		record := &IdempotencyRecord{
			Key:         idempotencyKey,
			Status:      StatusProcessing,
			RequestHash: requestHash,
			CreatedAt:   time.Now(),
		}

		// Try to set record (atomic) with SHORT ProcessingTTL
		if !trySetIdempotencyRecord(ctx, config.Redis, redisKey, record, config.ProcessingTTL) {
			// Another request beat us - retry get
			existingRecord, _ = getIdempotencyRecord(ctx, config.Redis, redisKey)
			if existingRecord != nil {
				if existingRecord.Status == StatusProcessing {
					c.AbortWithStatusJSON(http.StatusConflict, response.Error("REQUEST_IN_PROGRESS", "A request with this idempotency key is already being processed"))
					return
				}
				// Return cached response
				c.Data(existingRecord.ResponseCode, "application/json", []byte(existingRecord.ResponseBody))
				c.Abort()
				return
			}
		}

		// Create response writer to capture response
		rw := &idempotencyResponseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = rw

		// Process request
		c.Next()

		// Save completed record with LONG TTL
		now := time.Now()
		record.Status = StatusCompleted
		record.ResponseCode = rw.status
		record.ResponseBody = rw.body.String()
		record.CompletedAt = &now

		saveIdempotencyRecord(ctx, config.Redis, redisKey, record, config.TTL)
	}
}

// RequireIdempotencyKey creates a middleware that enforces idempotency key presence
func RequireIdempotencyKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader(IdempotencyKeyHeader)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, response.Error("MISSING_IDEMPOTENCY_KEY", "X-Idempotency-Key header is required"))
			return
		}
		c.Set(ContextKeyIdempotencyKey, key)
		c.Next()
	}
}

// GetIdempotencyKey extracts idempotency key from gin context
func GetIdempotencyKey(c *gin.Context) (string, bool) {
	key, exists := c.Get(ContextKeyIdempotencyKey)
	if !exists {
		return "", false
	}
	k, ok := key.(string)
	return k, ok
}

// idempotencyResponseWriter captures response for caching
type idempotencyResponseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *idempotencyResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *idempotencyResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Helper functions

func isMethodRequired(method string, requiredMethods []string) bool {
	for _, m := range requiredMethods {
		if method == m {
			return true
		}
	}
	return false
}

func matchPath(path, pattern string) bool {
	// Simple prefix matching, supports wildcards at end
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return path == pattern
}

func generateRequestHash(c *gin.Context, body []byte, config *IdempotencyConfig) string {
	h := sha256.New()

	if config.IncludePathInHash {
		h.Write([]byte(c.Request.Method))
		h.Write([]byte(c.Request.URL.Path))
	}

	if config.IncludeUserInHash {
		if userID, ok := GetUserID(c); ok {
			h.Write([]byte(userID))
		}
	}

	if config.IncludeBodyInHash && len(body) > 0 {
		h.Write(body)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func getIdempotencyRecord(ctx context.Context, redis RedisClient, key string) (*IdempotencyRecord, error) {
	result, err := redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var record IdempotencyRecord
	if err := json.Unmarshal([]byte(result), &record); err != nil {
		return nil, err
	}

	return &record, nil
}

func trySetIdempotencyRecord(ctx context.Context, redis RedisClient, key string, record *IdempotencyRecord, ttl time.Duration) bool {
	data, err := json.Marshal(record)
	if err != nil {
		return false
	}

	result, err := redis.SetNX(ctx, key, string(data), ttl).Result()
	if err != nil {
		return false
	}

	return result
}

func saveIdempotencyRecord(ctx context.Context, redis RedisClient, key string, record *IdempotencyRecord, ttl time.Duration) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return redis.Set(ctx, key, string(data), ttl).Err()
}

// DeleteIdempotencyRecord deletes an idempotency record (for testing or cleanup)
func DeleteIdempotencyRecord(ctx context.Context, redis RedisClient, idempotencyKey string) error {
	redisKey := IdempotencyKeyPrefix + idempotencyKey
	return redis.Del(ctx, redisKey).Err()
}

// CheckIdempotency checks if a request with the given key exists and returns its status
func CheckIdempotency(ctx context.Context, redis RedisClient, idempotencyKey string) (*IdempotencyRecord, error) {
	redisKey := IdempotencyKeyPrefix + idempotencyKey
	return getIdempotencyRecord(ctx, redis, redisKey)
}
