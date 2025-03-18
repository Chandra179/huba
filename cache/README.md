# Distributed Cache System with Redis

This package provides a Go implementation of a distributed cache system using Redis.

## Features

- Basic cache operations (Get, Set, Delete, Exists)
- Cache-aside pattern for transparent loading from a data source
- Distributed locking for coordinating access across services
- Rate limiting with sliding window algorithm

## Requirements

- Go 1.24 or later
- Redis server (v6.0 or later recommended)

## Installation

Add the dependency to your project:

```bash
go get github.com/redis/go-redis/v9
```

## Running Redis with Docker

The project includes a Docker Compose configuration for running Redis:

```bash
# Start Redis container
docker-compose up -d redis

# Or use the management script
./redis-management.sh start
```

The Redis container is configured with:
- Port: 6379 (default Redis port)
- Persistence: AOF (Append Only File) enabled
- Optional password protection (set via REDIS_PASSWORD environment variable)

Management script commands:
```bash
./redis-management.sh start  # Start Redis container
./redis-management.sh stop   # Stop Redis container
./redis-management.sh cli    # Open Redis CLI
./redis-management.sh flush  # Flush all Redis data
./redis-management.sh logs   # View Redis logs
./redis-management.sh info   # Show Redis info
```

## Basic Usage

```go
import (
    "context"
    "time"
    "your/project/cache"
)

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
    
    // Store data in cache (with 1 hour expiration)
    type User struct {
        ID   string
        Name string
    }
    user := User{ID: "123", Name: "John"}
    err = redisCache.Set(ctx, "user:123", user, time.Hour)
    
    // Retrieve data from cache
    var cachedUser User
    err = redisCache.Get(ctx, "user:123", &cachedUser)
    
    // Check if a key exists
    exists, err := redisCache.Exists(ctx, "user:123")
    
    // Delete a key
    err = redisCache.Delete(ctx, "user:123")
}
```

## Advanced Features

### Cache-Aside Pattern

The cache-aside pattern automatically loads data from a source when it's not in the cache:

```go
// Define a loader function
loader := func(ctx context.Context, key string) (interface{}, error) {
    // Load data from database or other source
    userID := key[5:] // Remove "user:" prefix
    return loadUserFromDB(userID)
}

// Use cache-aside to transparently handle cache misses
var user User
err := redisCache.CacheAside(ctx, "user:123", &user, time.Minute, loader)
```

### Distributed Locking

Coordinate access across multiple services:

```go
// Create a lock that expires after 10 seconds
lock := redisCache.NewDistributedLock("resource_name", 10*time.Second)

// Try to acquire the lock
err := lock.Acquire(ctx)
if err == nil {
    // We have the lock, do work...
    defer lock.Release(ctx)
    
    // If operation takes longer, extend the lock
    err = lock.Extend(ctx, 10*time.Second)
}
```

### Rate Limiting

Implement rate limiting across distributed services:

```go
// Create a rate limiter: 100 requests per minute
limiter := redisCache.NewRateLimiter(time.Minute, 100)

// Check if a request is allowed
allowed, err := limiter.Allow(ctx, "api:endpoint1")
if err == nil && allowed {
    // Process the request
}

// Check remaining quota
remaining, err := limiter.RemainingQuota(ctx, "api:endpoint1")
```

## Examples

See the `example` directory for complete working examples:

- `main.go` - Basic cache operations
- `advanced.go` - Advanced features (cache-aside, locking, rate limiting)

## Redis Configuration

The cache can be configured with the following Redis options:

```go
redisCache, err := cache.NewRedisCache(cache.RedisConfig{
    Address:  "redis-host:6379",  // Redis server address
    Password: "secret",           // Redis password (if required)
    DB:       0,                  // Redis database number
})
```

## Performance Considerations

- Use JSON serialization for complex objects
- Consider using compression for large values
- Set appropriate TTL (time-to-live) values to prevent memory issues
- Monitor Redis memory usage in production

## Thread Safety

All operations are safe for concurrent use from multiple goroutines. 