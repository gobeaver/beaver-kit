package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// Global instances
var (
	defaultCache Cache
	defaultOnce  sync.Once
	defaultErr   error
)

// Common errors
var (
	ErrNotInitialized = errors.New("cache not initialized")
	ErrInvalidDriver  = errors.New("invalid cache driver")
	ErrInvalidConfig  = errors.New("invalid cache configuration")
	ErrKeyNotFound    = errors.New("key not found")
	ErrInvalidTTL     = errors.New("invalid TTL value")
)

// Builder provides a way to create cache instances with custom prefixes
type Builder struct {
	prefix string
}

// WithPrefix creates a new Builder with the specified prefix
func WithPrefix(prefix string) *Builder {
	return &Builder{prefix: prefix}
}

// Init initializes the global cache instance using the builder's prefix
func (b *Builder) Init() error {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return err
	}
	return Init(*cfg)
}

// New creates a new cache instance using the builder's prefix
func (b *Builder) New() (Cache, error) {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return nil, err
	}
	return New(*cfg)
}

// Init initializes the global cache instance with optional config
func Init(configs ...Config) error {
	defaultOnce.Do(func() {
		var cfg *Config
		if len(configs) > 0 {
			cfg = &configs[0]
		} else {
			cfg, defaultErr = GetConfig()
			if defaultErr != nil {
				return
			}
		}

		defaultCache, defaultErr = New(*cfg)
	})

	return defaultErr
}

// New creates a new cache instance with given config
func New(cfg Config) (Cache, error) {
	// Set defaults
	if cfg.Driver == "" {
		cfg.Driver = "memory"
	}

	// Select driver based on config
	switch cfg.Driver {
	case "memory", "builtin":
		return memoryRegister(cfg)
	case "redis":
		return redisRegister(cfg)
	default:
		return nil, ErrInvalidDriver
	}
}

// Default returns the global cache instance
func Default() Cache {
	if defaultCache == nil {
		_ = Init()
	}
	return defaultCache
}

// Get retrieves a value by key from the global cache
func Get(ctx context.Context, key string) ([]byte, error) {
	if defaultCache == nil {
		return nil, ErrNotInitialized
	}
	return defaultCache.Get(ctx, key)
}

// Set stores a value with optional TTL in the global cache
func Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if defaultCache == nil {
		return ErrNotInitialized
	}
	return defaultCache.Set(ctx, key, value, ttl)
}

// Delete removes a key from the global cache
func Delete(ctx context.Context, key string) error {
	if defaultCache == nil {
		return ErrNotInitialized
	}
	return defaultCache.Delete(ctx, key)
}

// Exists checks if a key exists in the global cache
func Exists(ctx context.Context, key string) (bool, error) {
	if defaultCache == nil {
		return false, ErrNotInitialized
	}
	return defaultCache.Exists(ctx, key)
}

// Clear removes all keys from the global cache
func Clear(ctx context.Context) error {
	if defaultCache == nil {
		return ErrNotInitialized
	}
	return defaultCache.Clear(ctx)
}

// Ping checks if the global cache is reachable
func Ping(ctx context.Context) error {
	if defaultCache == nil {
		return ErrNotInitialized
	}
	return defaultCache.Ping(ctx)
}

// Health is an alias for Ping
func Health(ctx context.Context) error {
	return Ping(ctx)
}

// IsHealthy returns true if the cache is reachable
func IsHealthy() bool {
	return Health(context.Background()) == nil
}

// Reset clears the global instance (for testing)
func Reset() {
	if defaultCache != nil {
		defaultCache.Close()
	}
	defaultCache = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}

// Shutdown gracefully closes cache connections
func Shutdown(ctx context.Context) error {
	if defaultCache == nil {
		return nil
	}

	// Close with context timeout
	done := make(chan struct{})
	go func() {
		defaultCache.Close()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// MustInit initializes the cache and panics on error
func MustInit(configs ...Config) {
	if err := Init(configs...); err != nil {
		panic("failed to initialize cache: " + err.Error())
	}
}

// InitFromEnv is an alias for Init with no arguments
func InitFromEnv() error {
	return Init()
}

// NewFromEnv creates cache instance from environment variables
func NewFromEnv() (Cache, error) {
	cfg, err := GetConfig()
	if err != nil {
		return nil, err
	}
	return New(*cfg)
}
