package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// getTestConfig returns config for testing
func getTestConfig() *Config {
	cfg := DefaultConfig()

	if host := os.Getenv("TEST_REDIS_HOST"); host != "" {
		cfg.Host = host
	}
	if password := os.Getenv("TEST_REDIS_PASSWORD"); password != "" {
		cfg.Password = password
	}

	return cfg
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", cfg.Host)
	}
	if cfg.Port != 6379 {
		t.Errorf("Expected port 6379, got %d", cfg.Port)
	}
	if cfg.PoolSize != 100 {
		t.Errorf("Expected pool size 100, got %d", cfg.PoolSize)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", cfg.MaxRetries)
	}
}

func TestConfig_Addr(t *testing.T) {
	cfg := &Config{
		Host: "redis.example.com",
		Port: 6380,
	}

	expected := "redis.example.com:6380"
	if cfg.Addr() != expected {
		t.Errorf("Expected addr '%s', got '%s'", expected, cfg.Addr())
	}
}

func TestNewClient_InvalidConfig(t *testing.T) {
	cfg := &Config{
		Host:          "invalid-host-that-does-not-exist",
		Port:          9999,
		MaxRetries:    0,
		RetryInterval: 100 * time.Millisecond,
		DialTimeout:   500 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewClient(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid config, got nil")
	}
}

func TestComputeSHA1(t *testing.T) {
	script := "return 1"
	sha := computeSHA1(script)

	// SHA1 should be 40 hex characters
	if len(sha) != 40 {
		t.Errorf("Expected SHA1 length 40, got %d", len(sha))
	}

	// Same script should produce same SHA
	sha2 := computeSHA1(script)
	if sha != sha2 {
		t.Error("Same script should produce same SHA")
	}

	// Different script should produce different SHA
	sha3 := computeSHA1("return 2")
	if sha == sha3 {
		t.Error("Different scripts should produce different SHAs")
	}
}

func TestIsNoScriptError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{fmt.Errorf("some error"), false},
		{fmt.Errorf("NOSCRIPT No matching script. Please use EVAL."), true},
		{fmt.Errorf("NOSCRIPT some other message"), true},
	}

	for _, tt := range tests {
		result := isNoScriptError(tt.err)
		if result != tt.expected {
			t.Errorf("isNoScriptError(%v) = %v, want %v", tt.err, result, tt.expected)
		}
	}
}

// Integration tests - require Redis to be running

func TestNewClient_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to redis: %v", err)
	}
	defer client.Close()

	// Test Ping
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Test IsConnected
	if !client.IsConnected(ctx) {
		t.Error("Expected IsConnected to return true")
	}

	// Test underlying client not nil
	if client.Client() == nil {
		t.Error("Expected Client() to return non-nil")
	}
}

func TestClient_HealthCheck_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to redis: %v", err)
	}
	defer client.Close()

	if err := client.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestClient_BasicOperations_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to redis: %v", err)
	}
	defer client.Close()

	testKey := "test:key:" + time.Now().Format("20060102150405")

	// Test Set
	err = client.Set(ctx, testKey, "test_value", time.Minute).Err()
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// Test Get
	val, err := client.Get(ctx, testKey).Result()
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if val != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", val)
	}

	// Test Exists
	exists, err := client.Exists(ctx, testKey).Result()
	if err != nil {
		t.Errorf("Exists failed: %v", err)
	}
	if exists != 1 {
		t.Errorf("Expected exists=1, got %d", exists)
	}

	// Test Del
	deleted, err := client.Del(ctx, testKey).Result()
	if err != nil {
		t.Errorf("Del failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected deleted=1, got %d", deleted)
	}

	// Verify deleted
	exists, _ = client.Exists(ctx, testKey).Result()
	if exists != 0 {
		t.Error("Key should not exist after deletion")
	}
}

func TestClient_LuaScript_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to redis: %v", err)
	}
	defer client.Close()

	// Simple script that returns the sum of two args
	script := `return tonumber(ARGV[1]) + tonumber(ARGV[2])`
	scriptName := "test_add"

	// Load script
	info, err := client.LoadScript(ctx, scriptName, script)
	if err != nil {
		t.Fatalf("LoadScript failed: %v", err)
	}

	if info.Name != scriptName {
		t.Errorf("Expected name '%s', got '%s'", scriptName, info.Name)
	}
	if info.SHA == "" {
		t.Error("Expected non-empty SHA")
	}

	// Get cached SHA
	sha, ok := client.GetScriptSHA(scriptName)
	if !ok {
		t.Error("Expected script SHA to be cached")
	}
	if sha != info.SHA {
		t.Error("Cached SHA should match loaded SHA")
	}

	// Execute by SHA
	result, err := client.EvalSha(ctx, info.SHA, nil, 5, 3).Int()
	if err != nil {
		t.Errorf("EvalSha failed: %v", err)
	}
	if result != 8 {
		t.Errorf("Expected result 8, got %d", result)
	}

	// Execute by name
	result, err = client.EvalShaByName(ctx, scriptName, nil, 10, 20).Int()
	if err != nil {
		t.Errorf("EvalShaByName failed: %v", err)
	}
	if result != 30 {
		t.Errorf("Expected result 30, got %d", result)
	}
}

func TestClient_EvalWithFallback_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to redis: %v", err)
	}
	defer client.Close()

	script := `return tonumber(ARGV[1]) * 2`
	scriptName := "test_double"

	// First call - script not cached, should load and execute
	result, err := client.EvalWithFallback(ctx, scriptName, script, nil, 7).Int()
	if err != nil {
		t.Errorf("EvalWithFallback failed: %v", err)
	}
	if result != 14 {
		t.Errorf("Expected result 14, got %d", result)
	}

	// Second call - should use cached SHA
	result, err = client.EvalWithFallback(ctx, scriptName, script, nil, 10).Int()
	if err != nil {
		t.Errorf("Second EvalWithFallback failed: %v", err)
	}
	if result != 20 {
		t.Errorf("Expected result 20, got %d", result)
	}
}

func TestClient_HashOperations_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to redis: %v", err)
	}
	defer client.Close()

	testKey := "test:hash:" + time.Now().Format("20060102150405")
	defer client.Del(ctx, testKey)

	// Test HSet
	err = client.HSet(ctx, testKey, "field1", "value1", "field2", "value2").Err()
	if err != nil {
		t.Errorf("HSet failed: %v", err)
	}

	// Test HGet
	val, err := client.HGet(ctx, testKey, "field1").Result()
	if err != nil {
		t.Errorf("HGet failed: %v", err)
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got '%s'", val)
	}

	// Test HGetAll
	all, err := client.HGetAll(ctx, testKey).Result()
	if err != nil {
		t.Errorf("HGetAll failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(all))
	}
}
