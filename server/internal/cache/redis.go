package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache wraps a go-redis client for application-level caching.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new RedisCache and pings the server.
// Returns nil if Redis is unavailable — the app degrades gracefully without cache.
func NewRedisCache(addr, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		fmt.Printf("redis: unavailable at %s (%v) — caching disabled\n", addr, err)
		return nil
	}

	fmt.Printf("redis: connected to %s\n", addr)
	return &RedisCache{client: client}
}

// IsAvailable returns true if Redis is connected.
func (r *RedisCache) IsAvailable() bool {
	return r != nil && r.client != nil
}

// Get retrieves a cached value and unmarshals it into dest.
// Returns false if key not found or on error.
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) bool {
	if !r.IsAvailable() {
		return false
	}
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false
	}
	return true
}

// Set stores a value with TTL. Returns silently on error.
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if !r.IsAvailable() {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	r.client.Set(ctx, key, data, ttl)
}

// Delete removes a key. Returns silently on error.
func (r *RedisCache) Delete(ctx context.Context, key string) {
	if !r.IsAvailable() {
		return
	}
	r.client.Del(ctx, key)
}

// Close shuts down the Redis client.
func (r *RedisCache) Close() error {
	if !r.IsAvailable() {
		return nil
	}
	return r.client.Close()
}
