package oauth

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// MultiProviderConfigFromEnv loads multi-provider configuration from environment
type MultiProviderConfigFromEnv struct {
	// Provider configurations as JSON string
	ProvidersJSON string `env:"OAUTH_PROVIDERS"`

	// Global settings
	PKCEEnabled        bool          `env:"OAUTH_PKCE_ENABLED" envDefault:"true"`
	PKCEMethod         string        `env:"OAUTH_PKCE_METHOD" envDefault:"S256"`
	SessionTimeout     time.Duration `env:"OAUTH_SESSION_TIMEOUT" envDefault:"5m"`
	TokenCacheDuration time.Duration `env:"OAUTH_TOKEN_CACHE_DURATION" envDefault:"1h"`
	StateGenerator     string        `env:"OAUTH_STATE_GENERATOR" envDefault:"secure"`
	HTTPTimeout        time.Duration `env:"OAUTH_HTTP_TIMEOUT" envDefault:"30s"`

	// Security settings
	EncryptSessions bool   `env:"OAUTH_ENCRYPT_SESSIONS" envDefault:"false"`
	SecretKey       string `env:"OAUTH_SECRET_KEY"`

	// Debug mode
	Debug bool `env:"OAUTH_DEBUG" envDefault:"false"`
}

// GetMultiProviderConfig loads multi-provider config from environment
func GetMultiProviderConfig(opts ...config.Option) (*MultiProviderConfig, error) {
	envConfig := &MultiProviderConfigFromEnv{}
	if err := config.Load(envConfig, opts...); err != nil {
		return nil, fmt.Errorf("failed to load multi-provider config: %w", err)
	}

	mpConfig := &MultiProviderConfig{
		PKCEEnabled:        envConfig.PKCEEnabled,
		PKCEMethod:         envConfig.PKCEMethod,
		SessionTimeout:     envConfig.SessionTimeout,
		TokenCacheDuration: envConfig.TokenCacheDuration,
		StateGenerator:     envConfig.StateGenerator,
		HTTPTimeout:        envConfig.HTTPTimeout,
		EncryptSessions:    envConfig.EncryptSessions,
		SecretKey:          envConfig.SecretKey,
		Debug:              envConfig.Debug,
	}

	// Parse providers JSON if provided
	if envConfig.ProvidersJSON != "" {
		providers := make(map[string]ProviderConfig)
		if err := json.Unmarshal([]byte(envConfig.ProvidersJSON), &providers); err != nil {
			return nil, fmt.Errorf("failed to parse OAUTH_PROVIDERS JSON: %w", err)
		}
		mpConfig.Providers = providers
	} else {
		// Try to load individual provider configs from environment
		// Note: loadProvidersFromEnv uses direct os.Getenv calls, so prefix handling is manual
		mpConfig.Providers = loadProvidersFromEnv()
	}

	return mpConfig, nil
}

// loadProvidersFromEnv loads individual provider configurations from environment variables
func loadProvidersFromEnv() map[string]ProviderConfig {
	providers := make(map[string]ProviderConfig)

	// Check for common provider patterns in environment
	// Format: OAUTH_GOOGLE_CLIENT_ID, OAUTH_GITHUB_CLIENT_ID, etc.
	providerNames := []string{"google", "github", "apple", "twitter", "microsoft", "facebook"}

	for _, name := range providerNames {
		if cfg := loadProviderFromEnv(name); cfg != nil {
			providers[name] = *cfg
		}
	}

	// Also check for numbered custom providers
	// Format: OAUTH_PROVIDER_1_CLIENT_ID, OAUTH_PROVIDER_2_CLIENT_ID, etc.
	for i := 1; i <= 10; i++ {
		customName := fmt.Sprintf("provider_%d", i)
		if cfg := loadProviderFromEnv(customName); cfg != nil {
			// Try to get a better name from TYPE field
			if cfg.Type != "" {
				providers[cfg.Type] = *cfg
			} else {
				providers[customName] = *cfg
			}
		}
	}

	return providers
}

// loadProviderFromEnv loads a single provider configuration from environment
func loadProviderFromEnv(name string) *ProviderConfig {
	// Build prefix for this provider
	prefix := "OAUTH_" + strings.ToUpper(name) + "_"

	// Check if client ID exists (minimum required field)
	clientIDKey := prefix + "CLIENT_ID"

	clientID := os.Getenv(clientIDKey) //nolint:forbidigo // dynamic provider config requires direct env access
	if clientID == "" {
		return nil
	}

	// Load all fields for this provider
	cfg := &ProviderConfig{
		Type:         name,
		ClientID:     clientID,
		ClientSecret: os.Getenv(prefix + "CLIENT_SECRET"),  //nolint:forbidigo
		RedirectURL:  os.Getenv(prefix + "REDIRECT_URL"),   //nolint:forbidigo
		AuthURL:      os.Getenv(prefix + "AUTH_URL"),       //nolint:forbidigo
		TokenURL:     os.Getenv(prefix + "TOKEN_URL"),      //nolint:forbidigo
		UserInfoURL:  os.Getenv(prefix + "USERINFO_URL"),   //nolint:forbidigo
		RevokeURL:    os.Getenv(prefix + "REVOKE_URL"),     //nolint:forbidigo
		TeamID:       os.Getenv(prefix + "TEAM_ID"),        //nolint:forbidigo
		KeyID:        os.Getenv(prefix + "KEY_ID"),         //nolint:forbidigo
		PrivateKey:   os.Getenv(prefix + "PRIVATE_KEY"),    //nolint:forbidigo
		APIVersion:   os.Getenv(prefix + "API_VERSION"),    //nolint:forbidigo
	}

	// Parse scopes
	scopesStr := os.Getenv(prefix + "SCOPES") //nolint:forbidigo
	if scopesStr != "" {
		cfg.Scopes = strings.Split(scopesStr, ",")
		for i := range cfg.Scopes {
			cfg.Scopes[i] = strings.TrimSpace(cfg.Scopes[i])
		}
	}

	// Parse debug flag
	debugStr := os.Getenv(prefix + "DEBUG") //nolint:forbidigo
	cfg.Debug = debugStr == "true" || debugStr == "1"

	return cfg
}

// InitMultiProvider initializes the global multi-provider service
func InitMultiProvider(configs ...MultiProviderConfig) error {
	var cfg MultiProviderConfig

	if len(configs) > 0 {
		cfg = configs[0]
	} else {
		// Load from environment
		loadedConfig, err := GetMultiProviderConfig()
		if err != nil {
			return err
		}
		cfg = *loadedConfig
	}

	// Create multi-provider service
	service, err := NewMultiProviderService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create multi-provider service: %w", err)
	}

	// Set as global instance (we'll need to add this)
	setGlobalMultiProviderService(service)

	return nil
}

// Global multi-provider instance management
var (
	globalMultiProviderService *MultiProviderService
	globalMultiProviderMu      sync.RWMutex
)

// setGlobalMultiProviderService sets the global multi-provider service
func setGlobalMultiProviderService(service *MultiProviderService) {
	globalMultiProviderMu.Lock()
	defer globalMultiProviderMu.Unlock()
	globalMultiProviderService = service
}

// GetMultiProviderService returns the global multi-provider service
func GetMultiProviderService() *MultiProviderService {
	globalMultiProviderMu.RLock()
	defer globalMultiProviderMu.RUnlock()
	return globalMultiProviderService
}

// Multi returns the global multi-provider service (alias)
func Multi() *MultiProviderService {
	return GetMultiProviderService()
}

// WithPrefix creates a builder for multi-provider with custom prefix
type MultiProviderBuilder struct {
	prefix string
}

// WithPrefix creates a new multi-provider builder with custom prefix
func WithMultiProviderPrefix(prefix string) *MultiProviderBuilder {
	return &MultiProviderBuilder{prefix: prefix}
}

// Init initializes multi-provider with the builder's prefix
func (b *MultiProviderBuilder) Init() error {
	cfg, err := GetMultiProviderConfig(config.WithPrefix(b.prefix))
	if err != nil {
		return err
	}
	return InitMultiProvider(*cfg)
}
