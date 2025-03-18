package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"securedesign/cache"
)

type User struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	LastSeen time.Time `json:"last_seen"`
}

// This is a simulation of an external database
var mockDB = map[string]User{
	"user123": {
		ID:       "user123",
		Name:     "John Doe",
		Email:    "john@example.com",
		LastSeen: time.Now().Add(-24 * time.Hour),
	},
	"user456": {
		ID:       "user456",
		Name:     "Jane Smith",
		Email:    "jane@example.com",
		LastSeen: time.Now().Add(-12 * time.Hour),
	},
}

// Simulate database query latency
func simulateDBLatency() {
	time.Sleep(500 * time.Millisecond)
}

// Mock database lookup function
func lookupUserFromDB(id string) (User, error) {
	simulateDBLatency()
	user, exists := mockDB[id]
	if !exists {
		return User{}, sql.ErrNoRows
	}
	return user, nil
}

// Example of using cache-aside pattern
func cacheAsideExample(ctx context.Context, redisCache *cache.RedisCache) {
	fmt.Println("\n=== Cache-Aside Pattern Example ===")

	// Create a loader function that will be called on cache miss
	loader := func(ctx context.Context, key string) (interface{}, error) {
		fmt.Println("Cache miss! Loading from database...")
		userID := key[5:] // Remove "user:" prefix
		user, err := lookupUserFromDB(userID)
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	// First lookup - should be a cache miss and load from DB
	var user1 User
	err := redisCache.CacheAside(ctx, "user:user123", &user1, time.Minute, loader)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("First lookup (cache miss): %+v\n", user1)

	// Second lookup - should be a cache hit
	var user2 User
	start := time.Now()
	err = redisCache.CacheAside(ctx, "user:user123", &user2, time.Minute, loader)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("Second lookup (cache hit): %+v (took %v)\n", user2, time.Since(start))
}

// Example of using distributed locks
func distributedLockExample(ctx context.Context, redisCache *cache.RedisCache) {
	fmt.Println("\n=== Distributed Lock Example ===")

	// Create a distributed lock with 10 second expiry
	lock := redisCache.NewDistributedLock("user_update", 10*time.Second)

	// Try to acquire the lock
	err := lock.Acquire(ctx)
	if err != nil {
		if err == cache.ErrLockAcquisitionFailed {
			fmt.Println("Could not acquire lock - another process has it")
			return
		}
		log.Fatalf("Failed to acquire lock: %v", err)
	}

	fmt.Println("Lock acquired successfully")

	// Simulate some work with the lock held
	fmt.Println("Performing critical section work...")
	time.Sleep(2 * time.Second)

	// Extend the lock if the operation takes longer than expected
	err = lock.Extend(ctx, 10*time.Second)
	if err != nil {
		log.Fatalf("Failed to extend lock: %v", err)
	}
	fmt.Println("Lock extended successfully")

	// More work
	time.Sleep(1 * time.Second)

	// Release the lock when done
	err = lock.Release(ctx)
	if err != nil {
		log.Fatalf("Failed to release lock: %v", err)
	}
	fmt.Println("Lock released successfully")
}

// Example of using rate limiting
func rateLimitExample(ctx context.Context, redisCache *cache.RedisCache) {
	fmt.Println("\n=== Rate Limiting Example ===")

	// Create a rate limiter: 5 requests per 10 seconds
	limiter := redisCache.NewRateLimiter(10*time.Second, 5)

	// Try multiple requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Try to make a request
			allowed, err := limiter.Allow(ctx, "api:endpoint1")
			if err != nil {
				if err == cache.ErrRateLimitExceeded {
					fmt.Printf("Request %d: Rate limit exceeded\n", i)
					return
				}
				log.Printf("Request %d: Rate limit check failed: %v\n", i, err)
				return
			}

			if allowed {
				fmt.Printf("Request %d: Allowed\n", i)
				// Simulate processing the request
				time.Sleep(100 * time.Millisecond)
			}

			// Check remaining quota
			remaining, err := limiter.RemainingQuota(ctx, "api:endpoint1")
			if err != nil {
				log.Printf("Request %d: Failed to check remaining quota: %v\n", i, err)
				return
			}

			fmt.Printf("Request %d: Remaining quota: %d\n", i, remaining)
		}(i)
	}

	wg.Wait()
}

func main() {
	// Initialize the Redis cache
	redisCache, err := cache.NewRedisCache(cache.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	ctx := context.Background()

	// Run examples
	cacheAsideExample(ctx, redisCache)
	distributedLockExample(ctx, redisCache)
	rateLimitExample(ctx, redisCache)
}
