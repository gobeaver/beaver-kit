package slack

import (
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// Config defines slack configuration
type Config struct {
	WebhookURL string        `env:"WEBHOOK_URL"`
	Channel    string        `env:"CHANNEL"`
	Username   string        `env:"USERNAME,default:Beaver"`
	IconEmoji  string        `env:"ICON_EMOJI"`
	IconURL    string        `env:"ICON_URL"`
	Timeout    time.Duration `env:"TIMEOUT,default:10s"`

	// Retry configuration
	MaxRetries    int           `env:"MAX_RETRIES,default:3"`
	RetryDelay    time.Duration `env:"RETRY_DELAY,default:1s"`
	RetryMaxDelay time.Duration `env:"RETRY_MAX_DELAY,default:30s"`
	RetryJitter   bool          `env:"RETRY_JITTER,default:true"`

	// Rate limiting
	RateLimit int `env:"RATE_LIMIT,default:1"`  // requests per second
	RateBurst int `env:"RATE_BURST,default:10"` // burst size

	// Circuit breaker
	CircuitThreshold   int           `env:"CIRCUIT_THRESHOLD,default:5"`    // failures before opening
	CircuitTimeout     time.Duration `env:"CIRCUIT_TIMEOUT,default:60s"`    // time before half-open
	CircuitMaxRequests int           `env:"CIRCUIT_MAX_REQUESTS,default:1"` // requests in half-open state

	// Security
	MaxMessageSize int  `env:"MAX_MESSAGE_SIZE,default:40000"` // Slack's limit is 40KB
	SanitizeInput  bool `env:"SANITIZE_INPUT,default:true"`
	RedactErrors   bool `env:"REDACT_ERRORS,default:true"`

	// Monitoring
	EnableMetrics bool   `env:"ENABLE_METRICS,default:false"`
	EnableLogging bool   `env:"ENABLE_LOGGING,default:false"`
	LogLevel      string `env:"LOG_LEVEL,default:info"`

	// Debug configuration
	Debug bool `env:"DEBUG,default:false"`
}

// GetConfig returns config loaded from environment
func GetConfig(opts ...config.LoadOptions) (*Config, error) {
	cfg := &Config{}
	// Apply default prefix if not specified
	if len(opts) == 0 {
		opts = append(opts, config.LoadOptions{Prefix: "BEAVER_SLACK_"})
	}
	if err := config.Load(cfg, opts...); err != nil {
		return nil, err
	}
	return cfg, nil
}
