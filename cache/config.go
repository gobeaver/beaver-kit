package cache

import (
	"strings"
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// Config holds cache configuration
type Config struct {
	// Driver specifies cache backend: "memory" or "redis"
	Driver string `env:"CACHE_DRIVER" envDefault:"memory"`

	// Redis specific settings
	Host     string `env:"CACHE_HOST" envDefault:"localhost"`
	Port     string `env:"CACHE_PORT" envDefault:"6379"`
	Password string `env:"CACHE_PASSWORD"`
	Database int    `env:"CACHE_DATABASE" envDefault:"0"`

	// Connection URL (overrides host/port/password)
	URL string `env:"CACHE_URL"`

	// Connection pool settings
	MaxRetries      int `env:"CACHE_MAX_RETRIES" envDefault:"3"`
	PoolSize        int `env:"CACHE_POOL_SIZE" envDefault:"10"`
	MinIdleConns    int `env:"CACHE_MIN_IDLE_CONNS" envDefault:"2"`
	MaxIdleConns    int `env:"CACHE_MAX_IDLE_CONNS" envDefault:"5"`
	ConnMaxLifetime int `env:"CACHE_CONN_MAX_LIFETIME" envDefault:"0"`  // seconds
	ConnMaxIdleTime int `env:"CACHE_CONN_MAX_IDLE_TIME" envDefault:"0"` // seconds

	// Memory cache specific
	MaxSize         int64  `env:"CACHE_MAX_SIZE" envDefault:"0"`          // max memory in bytes
	MaxKeys         int    `env:"CACHE_MAX_KEYS" envDefault:"0"`          // max number of keys
	DefaultTTL      string `env:"CACHE_DEFAULT_TTL" envDefault:"0"`       // default TTL as duration string
	CleanupInterval string `env:"CACHE_CLEANUP_INTERVAL" envDefault:"1m"` // cleanup interval as duration string

	// TLS settings for Redis
	UseTLS   bool   `env:"CACHE_USE_TLS" envDefault:"false"`
	CertFile string `env:"CACHE_CERT_FILE"`
	KeyFile  string `env:"CACHE_KEY_FILE"`
	CAFile   string `env:"CACHE_CA_FILE"`

	// Common settings
	KeyPrefix string `env:"CACHE_KEY_PREFIX"` // prefix for all keys
	Namespace string `env:"CACHE_NAMESPACE"`  // namespace for isolation
}

// GetConfig loads configuration from environment variables
func GetConfig(opts ...config.Option) (*Config, error) {
	cfg := &Config{}
	if err := config.Load(cfg, opts...); err != nil {
		return nil, err
	}

	// Normalize driver
	cfg.Driver = strings.ToLower(cfg.Driver)

	return cfg, nil
}

// ParsedDefaultTTL returns the default TTL as a time.Duration
func (c Config) ParsedDefaultTTL() time.Duration {
	if c.DefaultTTL == "" {
		return 0
	}
	if d, err := time.ParseDuration(c.DefaultTTL); err == nil {
		return d
	}
	return 0
}

// ParsedCleanupInterval returns the cleanup interval as a time.Duration
func (c Config) ParsedCleanupInterval() time.Duration {
	if c.CleanupInterval == "" {
		return time.Minute
	}
	if d, err := time.ParseDuration(c.CleanupInterval); err == nil {
		return d
	}
	return time.Minute
}
