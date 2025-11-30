package cache

import (
	"context"
	"time"
)

// Cache defines the interface for cache implementations
type Cache interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with optional TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// Clear removes all keys
	Clear(ctx context.Context) error

	// Close closes the cache connection
	Close() error

	// Ping checks if cache is reachable
	Ping(ctx context.Context) error
}
