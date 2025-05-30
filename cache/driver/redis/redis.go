package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"
	
	"github.com/redis/go-redis/v9"
)

// RedisCache implements cache using Redis
type RedisCache struct {
	client    redis.UniversalClient
	keyPrefix string
}

// Config holds Redis specific configuration
type Config struct {
	// Connection
	Host     string
	Port     string
	Password string
	Database int
	URL      string
	
	// Pool settings
	MaxRetries      int
	PoolSize        int
	MinIdleConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	
	// TLS
	UseTLS   bool
	CertFile string
	KeyFile  string
	CAFile   string
	
	// Common
	KeyPrefix string
	Namespace string
}

// New creates a new Redis cache instance
func New(cfg Config) (*RedisCache, error) {
	// Build options
	opts := &redis.UniversalOptions{
		Addrs:    []string{buildAddr(cfg)},
		Password: cfg.Password,
		DB:       cfg.Database,
	}
	
	// Use URL if provided
	if cfg.URL != "" {
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid redis URL: %w", err)
		}
		opts = &redis.UniversalOptions{
			Addrs:    []string{opt.Addr},
			Password: opt.Password,
			DB:       opt.DB,
		}
		// Apply TLS from URL if present
		if opt.TLSConfig != nil {
			opts.TLSConfig = opt.TLSConfig
		}
	}
	
	// Apply pool settings
	if cfg.MaxRetries > 0 {
		opts.MaxRetries = cfg.MaxRetries
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		opts.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.MaxIdleConns > 0 {
		opts.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.ConnMaxLifetime > 0 {
		opts.ConnMaxLifetime = cfg.ConnMaxLifetime
	}
	if cfg.ConnMaxIdleTime > 0 {
		opts.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	}
	
	// Configure TLS if enabled
	if cfg.UseTLS && opts.TLSConfig == nil {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		
		if cfg.CertFile != "" && cfg.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS cert: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		
		opts.TLSConfig = tlsConfig
	}
	
	// Create client
	client := redis.NewUniversalClient(opts)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
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
	
	return &RedisCache{
		client:    client,
		keyPrefix: prefix,
	}, nil
}

// Get retrieves a value by key
func (rc *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := rc.keyPrefix + key
	val, err := rc.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("key not found")
		}
		return nil, err
	}
	return val, nil
}

// Set stores a value with optional TTL
func (rc *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := rc.keyPrefix + key
	return rc.client.Set(ctx, fullKey, value, ttl).Err()
}

// Delete removes a key
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := rc.keyPrefix + key
	return rc.client.Del(ctx, fullKey).Err()
}

// Exists checks if a key exists
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := rc.keyPrefix + key
	n, err := rc.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Clear removes all keys with the prefix
func (rc *RedisCache) Clear(ctx context.Context) error {
	if rc.keyPrefix == "" {
		// Without prefix, we can't safely clear
		return errors.New("cannot clear all keys without a prefix")
	}
	
	// Use SCAN to find all keys with prefix
	iter := rc.client.Scan(ctx, 0, rc.keyPrefix+"*", 0).Iterator()
	var keys []string
	
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		
		// Delete in batches of 1000
		if len(keys) >= 1000 {
			if err := rc.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}
	
	if err := iter.Err(); err != nil {
		return err
	}
	
	// Delete remaining keys
	if len(keys) > 0 {
		return rc.client.Del(ctx, keys...).Err()
	}
	
	return nil
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// Ping checks if Redis is reachable
func (rc *RedisCache) Ping(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

// buildAddr builds Redis address from config
func buildAddr(cfg Config) string {
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == "" {
		cfg.Port = "6379"
	}
	return fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
}

// Stats returns cache statistics
func (rc *RedisCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := rc.client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}
	
	stats := make(map[string]interface{})
	stats["info"] = info
	stats["key_prefix"] = rc.keyPrefix
	
	// Get pool stats if available
	if poolStats := rc.client.PoolStats(); poolStats != nil {
		stats["pool"] = map[string]interface{}{
			"hits":       poolStats.Hits,
			"misses":     poolStats.Misses,
			"timeouts":   poolStats.Timeouts,
			"total_conns": poolStats.TotalConns,
			"idle_conns": poolStats.IdleConns,
			"stale_conns": poolStats.StaleConns,
		}
	}
	
	return stats, nil
}