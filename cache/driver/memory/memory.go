package memory

import (
	"context"
	"errors"
	"sync"
	"time"
)

// item represents a cached item with expiration
type item struct {
	value      []byte
	expiration int64
	size       int64
}

// MemoryCache implements an in-memory cache
type MemoryCache struct {
	mu              sync.RWMutex
	items           map[string]*item
	maxSize         int64
	currentSize     int64
	maxKeys         int
	defaultTTL      time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	keyPrefix       string
}

// Config holds memory cache specific configuration
type Config struct {
	MaxSize         int64
	MaxKeys         int
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
	KeyPrefix       string
	Namespace       string
}

// New creates a new memory cache instance
func New(cfg Config) (*MemoryCache, error) {
	// Set defaults
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 1 * time.Minute
	}

	// Combine prefix and namespace
	prefix := cfg.KeyPrefix
	if cfg.Namespace != "" {
		if prefix != "" {
			prefix = cfg.Namespace + ":" + prefix
		} else {
			prefix = cfg.Namespace + ":"
		}
	}

	mc := &MemoryCache{
		items:           make(map[string]*item),
		maxSize:         cfg.MaxSize,
		maxKeys:         cfg.MaxKeys,
		defaultTTL:      cfg.DefaultTTL,
		cleanupInterval: cfg.CleanupInterval,
		stopCleanup:     make(chan struct{}),
		keyPrefix:       prefix,
	}

	// Start cleanup goroutine
	go mc.cleanupExpired()

	return mc, nil
}

// Get retrieves a value by key
func (mc *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	fullKey := mc.keyPrefix + key
	item, exists := mc.items[fullKey]
	if !exists {
		return nil, errors.New("key not found")
	}

	// Check expiration
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		return nil, errors.New("key not found")
	}

	return item.value, nil
}

// Set stores a value with optional TTL
func (mc *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	fullKey := mc.keyPrefix + key
	size := int64(len(value))

	// Check max keys limit
	if mc.maxKeys > 0 && len(mc.items) >= mc.maxKeys {
		if _, exists := mc.items[fullKey]; !exists {
			return errors.New("max keys limit reached")
		}
	}

	// Check size limit
	if mc.maxSize > 0 {
		// If updating existing key, subtract old size
		if old, exists := mc.items[fullKey]; exists {
			mc.currentSize -= old.size
		}

		if mc.currentSize+size > mc.maxSize {
			return errors.New("max size limit reached")
		}
	}

	// Use default TTL if not specified
	if ttl == 0 {
		ttl = mc.defaultTTL
	}

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	mc.items[fullKey] = &item{
		value:      value,
		expiration: expiration,
		size:       size,
	}

	mc.currentSize += size

	return nil
}

// Delete removes a key
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	fullKey := mc.keyPrefix + key
	if item, exists := mc.items[fullKey]; exists {
		mc.currentSize -= item.size
		delete(mc.items, fullKey)
	}

	return nil
}

// Exists checks if a key exists
func (mc *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	fullKey := mc.keyPrefix + key
	item, exists := mc.items[fullKey]
	if !exists {
		return false, nil
	}

	// Check expiration
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		return false, nil
	}

	return true, nil
}

// Clear removes all keys
func (mc *MemoryCache) Clear(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Only clear items with our prefix
	if mc.keyPrefix != "" {
		for key := range mc.items {
			if len(key) >= len(mc.keyPrefix) && key[:len(mc.keyPrefix)] == mc.keyPrefix {
				delete(mc.items, key)
			}
		}
	} else {
		mc.items = make(map[string]*item)
	}

	mc.currentSize = 0

	return nil
}

// Close closes the cache
func (mc *MemoryCache) Close() error {
	close(mc.stopCleanup)
	return nil
}

// Ping checks if cache is operational
func (mc *MemoryCache) Ping(ctx context.Context) error {
	// Memory cache is always available
	return nil
}

// cleanupExpired removes expired items periodically
func (mc *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(mc.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.removeExpired()
		case <-mc.stopCleanup:
			return
		}
	}
}

// removeExpired removes all expired items
func (mc *MemoryCache) removeExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now().UnixNano()
	for key, item := range mc.items {
		if item.expiration > 0 && now > item.expiration {
			mc.currentSize -= item.size
			delete(mc.items, key)
		}
	}
}

// Stats returns cache statistics
func (mc *MemoryCache) Stats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return map[string]interface{}{
		"keys":       len(mc.items),
		"size":       mc.currentSize,
		"max_size":   mc.maxSize,
		"max_keys":   mc.maxKeys,
		"key_prefix": mc.keyPrefix,
	}
}
