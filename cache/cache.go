package cache

import (
	"context"
	"time"
)

// Cache defines the interface for cache operations
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string, dest interface{}) error

	// Set stores a value in the cache with optional expiration
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// Close closes the cache connection
	Close() error
}
