package cache

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrLockAcquisitionFailed is returned when a lock can't be acquired
	ErrLockAcquisitionFailed = errors.New("failed to acquire lock")

	// ErrLockReleaseUnauthorized is returned when trying to release a lock that isn't owned by the caller
	ErrLockReleaseUnauthorized = errors.New("lock is not owned by this instance")
)

// DistributedLock represents a Redis-based distributed lock
type DistributedLock struct {
	redis  *redis.Client
	key    string
	token  string
	expiry time.Duration
}

// NewDistributedLock creates a new distributed lock
func (r *RedisCache) NewDistributedLock(key string, expiry time.Duration) *DistributedLock {
	return &DistributedLock{
		redis:  r.client,
		key:    "lock:" + key,
		token:  uuid.New().String(), // Unique token to identify lock owner
		expiry: expiry,
	}
}

// Acquire attempts to acquire the lock
func (dl *DistributedLock) Acquire(ctx context.Context) error {
	// Use SET NX to set the lock key only if it doesn't exist
	ok, err := dl.redis.SetNX(ctx, dl.key, dl.token, dl.expiry).Result()
	if err != nil {
		return err
	}

	if !ok {
		return ErrLockAcquisitionFailed
	}

	return nil
}

// Release releases the lock if it's owned by this instance
func (dl *DistributedLock) Release(ctx context.Context) error {
	// Use Lua script to ensure we only delete our own lock
	const script = `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	res, err := dl.redis.Eval(ctx, script, []string{dl.key}, dl.token).Result()
	if err != nil {
		return err
	}

	if res.(int64) == 0 {
		return ErrLockReleaseUnauthorized
	}

	return nil
}

// Extend extends the lock's expiry time if it's owned by this instance
func (dl *DistributedLock) Extend(ctx context.Context, extension time.Duration) error {
	// Use Lua script to ensure we only extend our own lock
	const script = `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	res, err := dl.redis.Eval(
		ctx,
		script,
		[]string{dl.key},
		dl.token,
		extension.Milliseconds(),
	).Result()

	if err != nil {
		return err
	}

	if res.(int64) == 0 {
		return ErrLockReleaseUnauthorized
	}

	dl.expiry = extension
	return nil
}
