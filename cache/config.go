package cache

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds cache configuration
type Config struct {
	// Driver specifies cache backend: "memory" or "redis"
	Driver string `json:"driver"`
	
	// Redis specific settings
	Host     string `json:"host,omitempty"`
	Port     string `json:"port,omitempty"`
	Password string `json:"password,omitempty"`
	Database int    `json:"database,omitempty"`
	
	// Connection URL (overrides host/port/password)
	URL string `json:"url,omitempty"`
	
	// Connection pool settings
	MaxRetries      int `json:"max_retries,omitempty"`
	PoolSize        int `json:"pool_size,omitempty"`
	MinIdleConns    int `json:"min_idle_conns,omitempty"`
	MaxIdleConns    int `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime int `json:"conn_max_lifetime,omitempty"` // seconds
	ConnMaxIdleTime int `json:"conn_max_idle_time,omitempty"` // seconds
	
	// Memory cache specific
	MaxSize       int64         `json:"max_size,omitempty"`        // max memory in bytes
	MaxKeys       int           `json:"max_keys,omitempty"`        // max number of keys
	DefaultTTL    time.Duration `json:"default_ttl,omitempty"`     // default TTL
	CleanupInterval time.Duration `json:"cleanup_interval,omitempty"` // cleanup interval
	
	// TLS settings for Redis
	UseTLS   bool   `json:"use_tls,omitempty"`
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`
	CAFile   string `json:"ca_file,omitempty"`
	
	// Common settings
	KeyPrefix string `json:"key_prefix,omitempty"` // prefix for all keys
	Namespace string `json:"namespace,omitempty"`  // namespace for isolation
}

// GetConfig loads configuration from environment variables
func GetConfig() (*Config, error) {
	cfg := &Config{
		Driver:   getEnv("BEAVER_CACHE_DRIVER", "memory"),
		Host:     getEnv("BEAVER_CACHE_HOST", "localhost"),
		Port:     getEnv("BEAVER_CACHE_PORT", "6379"),
		Password: getEnv("BEAVER_CACHE_PASSWORD", ""),
		URL:      getEnv("BEAVER_CACHE_URL", ""),
		
		KeyPrefix: getEnv("BEAVER_CACHE_KEY_PREFIX", ""),
		Namespace: getEnv("BEAVER_CACHE_NAMESPACE", ""),
	}
	
	// Parse database number
	if dbStr := getEnv("BEAVER_CACHE_DATABASE", "0"); dbStr != "" {
		db, err := strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid database number: %w", err)
		}
		cfg.Database = db
	}
	
	// Parse connection pool settings
	if maxRetries := getEnv("BEAVER_CACHE_MAX_RETRIES", "3"); maxRetries != "" {
		if v, err := strconv.Atoi(maxRetries); err == nil {
			cfg.MaxRetries = v
		}
	}
	
	if poolSize := getEnv("BEAVER_CACHE_POOL_SIZE", "10"); poolSize != "" {
		if v, err := strconv.Atoi(poolSize); err == nil {
			cfg.PoolSize = v
		}
	}
	
	if minIdle := getEnv("BEAVER_CACHE_MIN_IDLE_CONNS", "2"); minIdle != "" {
		if v, err := strconv.Atoi(minIdle); err == nil {
			cfg.MinIdleConns = v
		}
	}
	
	if maxIdle := getEnv("BEAVER_CACHE_MAX_IDLE_CONNS", "5"); maxIdle != "" {
		if v, err := strconv.Atoi(maxIdle); err == nil {
			cfg.MaxIdleConns = v
		}
	}
	
	if lifetime := getEnv("BEAVER_CACHE_CONN_MAX_LIFETIME", "0"); lifetime != "" {
		if v, err := strconv.Atoi(lifetime); err == nil {
			cfg.ConnMaxLifetime = v
		}
	}
	
	if idleTime := getEnv("BEAVER_CACHE_CONN_MAX_IDLE_TIME", "0"); idleTime != "" {
		if v, err := strconv.Atoi(idleTime); err == nil {
			cfg.ConnMaxIdleTime = v
		}
	}
	
	// Memory cache settings
	if maxSize := getEnv("BEAVER_CACHE_MAX_SIZE", "0"); maxSize != "" {
		if v, err := strconv.ParseInt(maxSize, 10, 64); err == nil {
			cfg.MaxSize = v
		}
	}
	
	if maxKeys := getEnv("BEAVER_CACHE_MAX_KEYS", "0"); maxKeys != "" {
		if v, err := strconv.Atoi(maxKeys); err == nil {
			cfg.MaxKeys = v
		}
	}
	
	if defaultTTL := getEnv("BEAVER_CACHE_DEFAULT_TTL", "0"); defaultTTL != "" {
		if v, err := time.ParseDuration(defaultTTL); err == nil {
			cfg.DefaultTTL = v
		}
	}
	
	if cleanupInterval := getEnv("BEAVER_CACHE_CLEANUP_INTERVAL", "1m"); cleanupInterval != "" {
		if v, err := time.ParseDuration(cleanupInterval); err == nil {
			cfg.CleanupInterval = v
		}
	}
	
	// TLS settings
	cfg.UseTLS = getEnv("BEAVER_CACHE_USE_TLS", "false") == "true"
	cfg.CertFile = getEnv("BEAVER_CACHE_CERT_FILE", "")
	cfg.KeyFile = getEnv("BEAVER_CACHE_KEY_FILE", "")
	cfg.CAFile = getEnv("BEAVER_CACHE_CA_FILE", "")
	
	// Normalize driver
	cfg.Driver = strings.ToLower(cfg.Driver)
	
	return cfg, nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// String returns a string representation of the config
func (c Config) String() string {
	// Hide sensitive data
	password := c.Password
	if password != "" {
		password = "***"
	}
	
	return fmt.Sprintf("Cache{driver=%s, host=%s:%s, prefix=%s}",
		c.Driver, c.Host, c.Port, c.KeyPrefix)
}