package saga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrSagaNotFound is returned when a saga instance is not found
	ErrSagaNotFound = errors.New("saga instance not found")
	// ErrSagaAlreadyExists is returned when trying to create a duplicate saga
	ErrSagaAlreadyExists = errors.New("saga instance already exists")
)

// Store is the interface for persisting saga state
type Store interface {
	// Save persists a saga instance
	Save(ctx context.Context, instance *Instance) error
	// Get retrieves a saga instance by ID
	Get(ctx context.Context, id string) (*Instance, error)
	// Update updates an existing saga instance
	Update(ctx context.Context, instance *Instance) error
	// Delete removes a saga instance
	Delete(ctx context.Context, id string) error
	// GetByStatus retrieves saga instances by status
	GetByStatus(ctx context.Context, status Status, limit int) ([]*Instance, error)
	// GetPendingCompensations returns sagas that need compensation
	GetPendingCompensations(ctx context.Context, limit int) ([]*Instance, error)
}

// MemoryStore is an in-memory implementation of Store for testing
type MemoryStore struct {
	mu        sync.RWMutex
	instances map[string]*Instance
}

// NewMemoryStore creates a new in-memory saga store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		instances: make(map[string]*Instance),
	}
}

// Save persists a saga instance
func (s *MemoryStore) Save(ctx context.Context, instance *Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.instances[instance.ID]; exists {
		return ErrSagaAlreadyExists
	}

	// Deep copy to prevent external modifications
	copied, err := s.deepCopy(instance)
	if err != nil {
		return err
	}

	s.instances[instance.ID] = copied
	return nil
}

// Get retrieves a saga instance by ID
func (s *MemoryStore) Get(ctx context.Context, id string) (*Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, exists := s.instances[id]
	if !exists {
		return nil, ErrSagaNotFound
	}

	// Return a copy to prevent external modifications
	return s.deepCopy(instance)
}

// Update updates an existing saga instance
func (s *MemoryStore) Update(ctx context.Context, instance *Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.instances[instance.ID]; !exists {
		return ErrSagaNotFound
	}

	// Deep copy to prevent external modifications
	copied, err := s.deepCopy(instance)
	if err != nil {
		return err
	}

	s.instances[instance.ID] = copied
	return nil
}

// Delete removes a saga instance
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.instances[id]; !exists {
		return ErrSagaNotFound
	}

	delete(s.instances, id)
	return nil
}

// GetByStatus retrieves saga instances by status
func (s *MemoryStore) GetByStatus(ctx context.Context, status Status, limit int) ([]*Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Instance
	for _, instance := range s.instances {
		if instance.Status == status {
			copied, err := s.deepCopy(instance)
			if err != nil {
				return nil, err
			}
			result = append(result, copied)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// GetPendingCompensations returns sagas that need compensation
func (s *MemoryStore) GetPendingCompensations(ctx context.Context, limit int) ([]*Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Instance
	for _, instance := range s.instances {
		if instance.Status == StatusFailed || instance.Status == StatusCompensating {
			copied, err := s.deepCopy(instance)
			if err != nil {
				return nil, err
			}
			result = append(result, copied)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// deepCopy creates a deep copy of a saga instance using JSON serialization
func (s *MemoryStore) deepCopy(instance *Instance) (*Instance, error) {
	data, err := json.Marshal(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal instance: %w", err)
	}

	var copied Instance
	if err := json.Unmarshal(data, &copied); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	return &copied, nil
}

// Clear removes all saga instances (for testing)
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances = make(map[string]*Instance)
}

// Count returns the number of stored instances (for testing)
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.instances)
}

// RedisStore is a Redis-based implementation of Store
type RedisStore struct {
	client     RedisClient
	keyPrefix  string
	expiration time.Duration
}

// RedisClient defines the interface for Redis operations needed by the saga store
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
}

// NewRedisStore creates a new Redis-based saga store
func NewRedisStore(client RedisClient, keyPrefix string, expiration time.Duration) *RedisStore {
	if keyPrefix == "" {
		keyPrefix = "saga:"
	}
	if expiration == 0 {
		expiration = 24 * time.Hour
	}
	return &RedisStore{
		client:     client,
		keyPrefix:  keyPrefix,
		expiration: expiration,
	}
}

// key returns the Redis key for a saga instance
func (s *RedisStore) key(id string) string {
	return s.keyPrefix + id
}

// Save persists a saga instance
func (s *RedisStore) Save(ctx context.Context, instance *Instance) error {
	// Check if exists
	_, err := s.client.Get(ctx, s.key(instance.ID))
	if err == nil {
		return ErrSagaAlreadyExists
	}

	data, err := instance.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize saga instance: %w", err)
	}

	return s.client.Set(ctx, s.key(instance.ID), data, s.expiration)
}

// Get retrieves a saga instance by ID
func (s *RedisStore) Get(ctx context.Context, id string) (*Instance, error) {
	data, err := s.client.Get(ctx, s.key(id))
	if err != nil {
		return nil, ErrSagaNotFound
	}

	return FromJSON([]byte(data))
}

// Update updates an existing saga instance
func (s *RedisStore) Update(ctx context.Context, instance *Instance) error {
	// Check if exists
	_, err := s.client.Get(ctx, s.key(instance.ID))
	if err != nil {
		return ErrSagaNotFound
	}

	data, err := instance.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize saga instance: %w", err)
	}

	return s.client.Set(ctx, s.key(instance.ID), data, s.expiration)
}

// Delete removes a saga instance
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	return s.client.Del(ctx, s.key(id))
}

// GetByStatus retrieves saga instances by status
func (s *RedisStore) GetByStatus(ctx context.Context, status Status, limit int) ([]*Instance, error) {
	keys, err := s.client.Keys(ctx, s.keyPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	var result []*Instance
	for _, key := range keys {
		data, err := s.client.Get(ctx, key)
		if err != nil {
			continue
		}

		instance, err := FromJSON([]byte(data))
		if err != nil {
			continue
		}

		if instance.Status == status {
			result = append(result, instance)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// GetPendingCompensations returns sagas that need compensation
func (s *RedisStore) GetPendingCompensations(ctx context.Context, limit int) ([]*Instance, error) {
	keys, err := s.client.Keys(ctx, s.keyPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	var result []*Instance
	for _, key := range keys {
		data, err := s.client.Get(ctx, key)
		if err != nil {
			continue
		}

		instance, err := FromJSON([]byte(data))
		if err != nil {
			continue
		}

		if instance.Status == StatusFailed || instance.Status == StatusCompensating {
			result = append(result, instance)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// RedisClientAdapter adapts go-redis client to RedisClient interface
type RedisClientAdapter struct {
	client *redis.Client
}

// NewRedisClientAdapter creates a new adapter for go-redis client
func NewRedisClientAdapter(client *redis.Client) *RedisClientAdapter {
	return &RedisClientAdapter{client: client}
}

func (a *RedisClientAdapter) Get(ctx context.Context, key string) (string, error) {
	return a.client.Get(ctx, key).Result()
}

func (a *RedisClientAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return a.client.Set(ctx, key, value, expiration).Err()
}

func (a *RedisClientAdapter) Del(ctx context.Context, keys ...string) error {
	return a.client.Del(ctx, keys...).Err()
}

func (a *RedisClientAdapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	return a.client.Keys(ctx, pattern).Result()
}
