package oauth

import (
	"fmt"
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// Config defines the OAuth service configuration
type Config struct {
	// Provider specifies the OAuth provider (google, github, apple, twitter, custom)
	Provider string `env:"OAUTH_PROVIDER,default:google"`

	// ClientID is the OAuth application's client ID
	ClientID string `env:"OAUTH_CLIENT_ID,required"`

	// ClientSecret is the OAuth application's client secret
	ClientSecret string `env:"OAUTH_CLIENT_SECRET,required"`

	// RedirectURL is the callback URL after authentication
	RedirectURL string `env:"OAUTH_REDIRECT_URL,required"`

	// Scopes is a comma-separated list of OAuth scopes
	Scopes string `env:"OAUTH_SCOPES,default:openid,profile,email"`

	// State is the default state parameter for CSRF protection
	State string `env:"OAUTH_STATE"`

	// StateGenerator defines how to generate state tokens (uuid, secure, custom)
	StateGenerator string `env:"OAUTH_STATE_GENERATOR,default:secure"`

	// PKCEEnabled enables PKCE flow for enhanced security
	PKCEEnabled bool `env:"OAUTH_PKCE_ENABLED,default:true"`

	// PKCEMethod is the PKCE challenge method (S256 or plain)
	PKCEMethod string `env:"OAUTH_PKCE_METHOD,default:S256"`

	// TokenCacheDuration is how long to cache tokens
	TokenCacheDuration time.Duration `env:"OAUTH_TOKEN_CACHE_DURATION,default:1h"`

	// StateTimeout is how long state parameters are valid
	StateTimeout time.Duration `env:"OAUTH_STATE_TIMEOUT,default:5m"`

	// HTTPTimeout is the timeout for HTTP requests
	HTTPTimeout time.Duration `env:"OAUTH_HTTP_TIMEOUT,default:30s"`

	// Debug enables debug logging
	Debug bool `env:"OAUTH_DEBUG,default:false"`

	// Custom provider configuration (for generic OAuth2 providers)
	AuthURL     string `env:"OAUTH_AUTH_URL"`
	TokenURL    string `env:"OAUTH_TOKEN_URL"`
	UserInfoURL string `env:"OAUTH_USERINFO_URL"`

	// Provider-specific configurations
	AppleTeamID     string `env:"OAUTH_APPLE_TEAM_ID"`
	AppleKeyID      string `env:"OAUTH_APPLE_KEY_ID"`
	ApplePrivateKey string `env:"OAUTH_APPLE_PRIVATE_KEY"`

	// Twitter API version (1.1 or 2)
	TwitterAPIVersion string `env:"OAUTH_TWITTER_API_VERSION,default:2"`
}

// GetConfig returns config loaded from environment with optional LoadOptions
func GetConfig(opts ...config.LoadOptions) (*Config, error) {
	cfg := &Config{}
	if err := config.Load(cfg, opts...); err != nil {
		return nil, fmt.Errorf("failed to load oauth config: %w", err)
	}
	return cfg, nil
}

// validateConfig checks configuration validity
func validateConfig(cfg Config) error {
	if cfg.ClientID == "" {
		return fmt.Errorf("%w: client_id required", ErrInvalidConfig)
	}
	if cfg.ClientSecret == "" && cfg.Provider != "apple" { // Apple uses JWT instead
		return fmt.Errorf("%w: client_secret required", ErrInvalidConfig)
	}
	if cfg.RedirectURL == "" {
		return fmt.Errorf("%w: redirect_url required", ErrInvalidConfig)
	}

	// Validate provider
	switch cfg.Provider {
	case "google", "github", "apple", "twitter", "custom":
		// Valid providers
	default:
		return fmt.Errorf("%w: unknown provider: %s", ErrInvalidConfig, cfg.Provider)
	}

	// Validate custom provider requirements
	if cfg.Provider == "custom" {
		if cfg.AuthURL == "" || cfg.TokenURL == "" {
			return fmt.Errorf("%w: auth_url and token_url required for custom provider", ErrInvalidConfig)
		}
	}

	// Validate Apple-specific requirements
	if cfg.Provider == "apple" {
		if cfg.AppleTeamID == "" || cfg.AppleKeyID == "" || cfg.ApplePrivateKey == "" {
			return fmt.Errorf("%w: apple provider requires team_id, key_id, and private_key", ErrInvalidConfig)
		}
	}

	// Validate PKCE method
	if cfg.PKCEEnabled && cfg.PKCEMethod != "S256" && cfg.PKCEMethod != "plain" {
		return fmt.Errorf("%w: invalid PKCE method: %s (must be S256 or plain)", ErrInvalidConfig, cfg.PKCEMethod)
	}

	return nil
}

// Builder pattern for custom prefixes
type Builder struct {
	prefix string
}

// WithPrefix creates a new Builder with the specified prefix
func WithPrefix(prefix string) *Builder {
	return &Builder{prefix: prefix}
}

// Init initializes the OAuth service with the builder's prefix
func (b *Builder) Init() error {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return err
	}
	return Init(*cfg)
}

// New creates a new OAuth service instance with the builder's prefix
func (b *Builder) New() (*Service, error) {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return nil, err
	}
	return New(*cfg)
}
