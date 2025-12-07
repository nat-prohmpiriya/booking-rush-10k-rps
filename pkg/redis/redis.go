package redis

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection configuration
type Config struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Retry configuration
	MaxRetries    int
	RetryInterval time.Duration
}

// DefaultConfig returns default Redis configuration
func DefaultConfig() *Config {
	return &Config{
		Host:          "localhost",
		Port:          6379,
		Password:      "",
		DB:            0,
		PoolSize:      100,
		MinIdleConns:  10,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		MaxRetries:    3,
		RetryInterval: time.Second,
	}
}

// Addr returns the Redis address
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Client wraps redis.Client with additional functionality
type Client struct {
	client  *redis.Client
	config  *Config
	scripts sync.Map // map[scriptName]sha
}

// NewClient creates a new Redis client with retry logic
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	opts := &redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	client := redis.NewClient(opts)

	// Connect with retry logic
	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(cfg.RetryInterval)
		}

		if lastErr = client.Ping(ctx).Err(); lastErr == nil {
			return &Client{
				client: client,
				config: cfg,
			}, nil
		}
	}

	client.Close()
	return nil, fmt.Errorf("failed to connect to redis after %d attempts: %w", cfg.MaxRetries+1, lastErr)
}

// Client returns the underlying redis.Client
func (c *Client) Client() *redis.Client {
	return c.client
}

// Ping checks if Redis connection is alive
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// HealthCheck performs a health check on Redis
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	if result != "PONG" {
		return fmt.Errorf("redis health check unexpected response: %s", result)
	}

	return nil
}

// IsConnected returns true if Redis connection is alive
func (c *Client) IsConnected(ctx context.Context) bool {
	return c.Ping(ctx) == nil
}

// --- Lua Script Support ---

// ScriptInfo holds information about a loaded script
type ScriptInfo struct {
	Name   string
	SHA    string
	Script string
}

// computeSHA1 computes SHA1 hash of a script (same as Redis does)
func computeSHA1(script string) string {
	h := sha1.New()
	h.Write([]byte(script))
	return hex.EncodeToString(h.Sum(nil))
}

// LoadScript loads a Lua script into Redis and caches its SHA
func (c *Client) LoadScript(ctx context.Context, name, script string) (*ScriptInfo, error) {
	sha, err := c.client.ScriptLoad(ctx, script).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load script %s: %w", name, err)
	}

	info := &ScriptInfo{
		Name:   name,
		SHA:    sha,
		Script: script,
	}

	c.scripts.Store(name, info)
	return info, nil
}

// GetScriptSHA returns the cached SHA for a script name
func (c *Client) GetScriptSHA(name string) (string, bool) {
	if info, ok := c.scripts.Load(name); ok {
		return info.(*ScriptInfo).SHA, true
	}
	return "", false
}

// EvalSha executes a script by SHA (faster than Eval)
func (c *Client) EvalSha(ctx context.Context, sha string, keys []string, args ...interface{}) *redis.Cmd {
	return c.client.EvalSha(ctx, sha, keys, args...)
}

// EvalShaByName executes a script by name (looks up cached SHA)
func (c *Client) EvalShaByName(ctx context.Context, name string, keys []string, args ...interface{}) *redis.Cmd {
	sha, ok := c.GetScriptSHA(name)
	if !ok {
		cmd := redis.NewCmd(ctx)
		cmd.SetErr(fmt.Errorf("script %s not loaded", name))
		return cmd
	}
	return c.EvalSha(ctx, sha, keys, args...)
}

// EvalWithFallback tries EvalSha, falls back to Eval if script not cached
func (c *Client) EvalWithFallback(ctx context.Context, name, script string, keys []string, args ...interface{}) *redis.Cmd {
	sha, ok := c.GetScriptSHA(name)
	if ok {
		result := c.client.EvalSha(ctx, sha, keys, args...)
		// Check if script exists on server
		if result.Err() != nil && isNoScriptError(result.Err()) {
			// Reload script and retry
			if _, err := c.LoadScript(ctx, name, script); err == nil {
				sha, _ = c.GetScriptSHA(name)
				return c.client.EvalSha(ctx, sha, keys, args...)
			}
		}
		return result
	}

	// Script not cached, load it first
	if _, err := c.LoadScript(ctx, name, script); err != nil {
		cmd := redis.NewCmd(ctx)
		cmd.SetErr(err)
		return cmd
	}

	sha, _ = c.GetScriptSHA(name)
	return c.client.EvalSha(ctx, sha, keys, args...)
}

// isNoScriptError checks if error is NOSCRIPT error
func isNoScriptError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "NOSCRIPT No matching script. Please use EVAL." ||
		len(err.Error()) >= 8 && err.Error()[:8] == "NOSCRIPT"
}

// --- Basic Redis Operations ---

// Get gets a value by key
func (c *Client) Get(ctx context.Context, key string) *redis.StringCmd {
	return c.client.Get(ctx, key)
}

// Set sets a value with optional expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return c.client.Set(ctx, key, value, expiration)
}

// SetNX sets a value only if key doesn't exist
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return c.client.SetNX(ctx, key, value, expiration)
}

// Del deletes keys
func (c *Client) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Del(ctx, keys...)
}

// Exists checks if keys exist
func (c *Client) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.client.Exists(ctx, keys...)
}

// Expire sets TTL on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return c.client.Expire(ctx, key, expiration)
}

// TTL gets TTL of a key
func (c *Client) TTL(ctx context.Context, key string) *redis.DurationCmd {
	return c.client.TTL(ctx, key)
}

// Incr increments a key
func (c *Client) Incr(ctx context.Context, key string) *redis.IntCmd {
	return c.client.Incr(ctx, key)
}

// Decr decrements a key
func (c *Client) Decr(ctx context.Context, key string) *redis.IntCmd {
	return c.client.Decr(ctx, key)
}

// IncrBy increments a key by amount
func (c *Client) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return c.client.IncrBy(ctx, key, value)
}

// DecrBy decrements a key by amount
func (c *Client) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return c.client.DecrBy(ctx, key, value)
}

// --- Hash Operations ---

// HGet gets a hash field
func (c *Client) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return c.client.HGet(ctx, key, field)
}

// HSet sets hash fields
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.HSet(ctx, key, values...)
}

// HGetAll gets all fields in a hash
func (c *Client) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return c.client.HGetAll(ctx, key)
}

// HIncrBy increments a hash field
func (c *Client) HIncrBy(ctx context.Context, key, field string, incr int64) *redis.IntCmd {
	return c.client.HIncrBy(ctx, key, field, incr)
}

// --- List Operations ---

// LPush prepends to a list
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.LPush(ctx, key, values...)
}

// RPush appends to a list
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.client.RPush(ctx, key, values...)
}

// LRange gets a range from a list
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return c.client.LRange(ctx, key, start, stop)
}

// --- Pipeline ---

// Pipeline returns a pipeline for batch operations
func (c *Client) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// TxPipeline returns a transactional pipeline
func (c *Client) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}
