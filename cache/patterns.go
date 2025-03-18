package cache

import (
	"context"
	"errors"
	"time"
)

// LoaderFunc is a function that loads data when cache misses
type LoaderFunc func(ctx context.Context, key string) (interface{}, error)

// CacheAside implements the cache-aside pattern
func (r *RedisCache) CacheAside(ctx context.Context, key string, dest interface{}, expiry time.Duration, loader LoaderFunc) error {
	// Try to get from cache first
	err := r.Get(ctx, key, dest)
	if err == nil {
		// Cache hit
		return nil
	}

	if err != ErrKeyNotFound {
		// Real error
		return err
	}

	// Cache miss - load from source
	data, err := loader(ctx, key)
	if err != nil {
		return err
	}

	// Store in cache for future requests
	if err := r.Set(ctx, key, data, expiry); err != nil {
		return err
	}

	// Copy to destination
	// Since dest is a pointer, we need to set the loaded data into it
	switch v := dest.(type) {
	case *interface{}:
		*v = data
	default:
		// For complex types, we need to set again to load into the destination
		return r.Get(ctx, key, dest)
	}

	return nil
}

// RateLimiter implements a Redis-based distributed rate limiter
type RateLimiter struct {
	cache       *RedisCache
	window      time.Duration
	maxRequests int64
}

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// NewRateLimiter creates a new rate limiter
func (r *RedisCache) NewRateLimiter(window time.Duration, maxRequests int64) *RateLimiter {
	return &RateLimiter{
		cache:       r,
		window:      window,
		maxRequests: maxRequests,
	}
}

// Allow checks if a request is allowed under rate limits
func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// Use a sliding window for rate limiting
	limitKey := "ratelimit:" + key

	// Use Lua script for atomic operations
	const script = `
		local key = KEYS[1]
		local max = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		-- Remove expired entries
		redis.call("ZREMRANGEBYSCORE", key, 0, now - window)
		
		-- Count requests in current window
		local count = redis.call("ZCARD", key)
		
		-- If under the limit, add the new request
		if count < max then
			redis.call("ZADD", key, now, now .. ":" .. math.random())
			redis.call("EXPIRE", key, window)
			return 1
		end
		
		return 0
	`

	// Execute the script
	res, err := rl.cache.client.Eval(
		ctx,
		script,
		[]string{limitKey},
		rl.maxRequests,
		rl.window.Seconds(),
		time.Now().Unix(),
	).Result()

	if err != nil {
		return false, err
	}

	// Check if allowed
	allowed := res.(int64) == 1
	if !allowed {
		return false, ErrRateLimitExceeded
	}

	return true, nil
}

// RemainingQuota returns the number of remaining requests allowed
func (rl *RateLimiter) RemainingQuota(ctx context.Context, key string) (int64, error) {
	limitKey := "ratelimit:" + key
	now := time.Now().Unix()

	// Remove expired entries
	err := rl.cache.client.ZRemRangeByScore(
		ctx,
		limitKey,
		"0",
		string(now-int64(rl.window.Seconds())),
	).Err()

	if err != nil {
		return 0, err
	}

	// Count current entries
	count, err := rl.cache.client.ZCard(ctx, limitKey).Result()
	if err != nil {
		return 0, err
	}

	// Calculate remaining
	remaining := rl.maxRequests - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}
