package slack

import (
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// Config defines slack configuration
type Config struct {
	WebhookURL string        `env:"WEBHOOK_URL"`
	Channel    string        `env:"CHANNEL"`
	Username   string        `env:"USERNAME" envDefault:"Beaver"`
	IconEmoji  string        `env:"ICON_EMOJI"`
	IconURL    string        `env:"ICON_URL"`
	Timeout    time.Duration `env:"TIMEOUT" envDefault:"10s"`

	// Retry configuration
	MaxRetries    int           `env:"MAX_RETRIES" envDefault:"3"`
	RetryDelay    time.Duration `env:"RETRY_DELAY" envDefault:"1s"`
	RetryMaxDelay time.Duration `env:"RETRY_MAX_DELAY" envDefault:"30s"`
	RetryJitter   bool          `env:"RETRY_JITTER" envDefault:"true"`

	// Rate limiting
	RateLimit int `env:"RATE_LIMIT" envDefault:"1"`  // requests per second
	RateBurst int `env:"RATE_BURST" envDefault:"10"` // burst size

	// Circuit breaker
	CircuitThreshold   int           `env:"CIRCUIT_THRESHOLD" envDefault:"5"`    // failures before opening
	CircuitTimeout     time.Duration `env:"CIRCUIT_TIMEOUT" envDefault:"60s"`    // time before half-open
	CircuitMaxRequests int           `env:"CIRCUIT_MAX_REQUESTS" envDefault:"1"` // requests in half-open state

	// Security
	MaxMessageSize int  `env:"MAX_MESSAGE_SIZE" envDefault:"40000"` // Slack's limit is 40KB
	SanitizeInput  bool `env:"SANITIZE_INPUT" envDefault:"true"`
	RedactErrors   bool `env:"REDACT_ERRORS" envDefault:"true"`

	// Monitoring
	EnableMetrics bool   `env:"ENABLE_METRICS" envDefault:"false"`
	EnableLogging bool   `env:"ENABLE_LOGGING" envDefault:"false"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"info"`

	// Debug configuration
	Debug bool `env:"DEBUG" envDefault:"false"`
}

// DefaultConfig returns a Config with all default values applied.
// Use this when creating configs programmatically instead of from environment variables.
func DefaultConfig() Config {
	return Config{
		Username:           "Beaver",
		Timeout:            10 * time.Second,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
		RetryMaxDelay:      30 * time.Second,
		RetryJitter:        true,
		RateLimit:          1,
		RateBurst:          10,
		CircuitThreshold:   5,
		CircuitTimeout:     60 * time.Second,
		CircuitMaxRequests: 1,
		MaxMessageSize:     40000,
		SanitizeInput:      true,
		RedactErrors:       true,
		EnableMetrics:      false,
		EnableLogging:      false,
		LogLevel:           "info",
		Debug:              false,
	}
}

// GetConfig returns config loaded from environment
func GetConfig(opts ...config.Option) (*Config, error) {
	cfg := &Config{}
	// Apply default prefix if not specified
	if len(opts) == 0 {
		opts = append(opts, config.WithPrefix("BEAVER_SLACK_"))
	}
	if err := config.Load(cfg, opts...); err != nil {
		return nil, err
	}
	return cfg, nil
}
