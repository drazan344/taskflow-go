package database

import (
	"context"
	"fmt"
	"time"

	"github.com/drazan344/taskflow-go/internal/config"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *redis.Client
}

// ConnectRedis establishes a connection to Redis
func ConnectRedis(cfg *config.Config) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	fmt.Println("Redis connection established successfully")

	return &Redis{Client: rdb}, nil
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.Client.Close()
}

// Health checks Redis connection health
func (r *Redis) Health(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Set stores a value in Redis with expiration
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value from Redis
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Delete removes a key from Redis
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in Redis
func (r *Redis) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.Client.Exists(ctx, keys...).Result()
}

// SetWithTenant stores a value with tenant prefix
func (r *Redis) SetWithTenant(ctx context.Context, tenantID, key string, value interface{}, expiration time.Duration) error {
	prefixedKey := fmt.Sprintf("tenant:%s:%s", tenantID, key)
	return r.Set(ctx, prefixedKey, value, expiration)
}

// GetWithTenant retrieves a value with tenant prefix
func (r *Redis) GetWithTenant(ctx context.Context, tenantID, key string) (string, error) {
	prefixedKey := fmt.Sprintf("tenant:%s:%s", tenantID, key)
	return r.Get(ctx, prefixedKey)
}

// DeleteWithTenant removes a key with tenant prefix
func (r *Redis) DeleteWithTenant(ctx context.Context, tenantID string, keys ...string) error {
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = fmt.Sprintf("tenant:%s:%s", tenantID, key)
	}
	return r.Delete(ctx, prefixedKeys...)
}

// IncrWithTenant increments a counter with tenant prefix
func (r *Redis) IncrWithTenant(ctx context.Context, tenantID, key string) (int64, error) {
	prefixedKey := fmt.Sprintf("tenant:%s:%s", tenantID, key)
	return r.Client.Incr(ctx, prefixedKey).Result()
}

// ExpireWithTenant sets expiration for a key with tenant prefix
func (r *Redis) ExpireWithTenant(ctx context.Context, tenantID, key string, expiration time.Duration) error {
	prefixedKey := fmt.Sprintf("tenant:%s:%s", tenantID, key)
	return r.Client.Expire(ctx, prefixedKey, expiration).Err()
}

// Publish publishes a message to a Redis channel
func (r *Redis) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.Client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to Redis channels
func (r *Redis) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.Client.Subscribe(ctx, channels...)
}