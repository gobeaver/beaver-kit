package oauth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MultiProviderService manages multiple OAuth providers
type MultiProviderService struct {
	providers   map[string]Provider
	sessions    SessionStore
	tokens      TokenStore
	config      MultiProviderConfig
	stateGen    StateGenerator
	mu          sync.RWMutex
	httpTimeout time.Duration
}

// MultiProviderConfig defines configuration for multi-provider service
type MultiProviderConfig struct {
	// Provider configurations mapped by name
	Providers map[string]ProviderConfig `env:"OAUTH_PROVIDERS"`
	
	// Global settings
	PKCEEnabled        bool          `env:"OAUTH_PKCE_ENABLED,default:true"`
	PKCEMethod         string        `env:"OAUTH_PKCE_METHOD,default:S256"`
	SessionTimeout     time.Duration `env:"OAUTH_SESSION_TIMEOUT,default:5m"`
	TokenCacheDuration time.Duration `env:"OAUTH_TOKEN_CACHE_DURATION,default:1h"`
	StateGenerator     string        `env:"OAUTH_STATE_GENERATOR,default:secure"`
	HTTPTimeout        time.Duration `env:"OAUTH_HTTP_TIMEOUT,default:30s"`
	
	// Security settings
	EncryptSessions bool   `env:"OAUTH_ENCRYPT_SESSIONS,default:false"`
	SecretKey       string `env:"OAUTH_SECRET_KEY"`
	
	// Debug mode
	Debug bool `env:"OAUTH_DEBUG,default:false"`
}

// NewMultiProviderService creates a new multi-provider OAuth service
func NewMultiProviderService(config MultiProviderConfig) (*MultiProviderService, error) {
	// Initialize session store
	sessionStore := NewMemorySessionStore(config.SessionTimeout)
	
	// Initialize token store
	tokenStore := NewMemoryTokenStore(config.TokenCacheDuration)
	
	// Initialize state generator
	var stateGen StateGenerator
	switch config.StateGenerator {
	case "uuid":
		stateGen = &UUIDStateGenerator{}
	case "secure", "":
		stateGen = &SecureStateGenerator{}
	default:
		return nil, fmt.Errorf("unknown state generator: %s", config.StateGenerator)
	}
	
	service := &MultiProviderService{
		providers:   make(map[string]Provider),
		sessions:    sessionStore,
		tokens:      tokenStore,
		config:      config,
		stateGen:    stateGen,
		httpTimeout: config.HTTPTimeout,
	}
	
	// Initialize providers from config
	if config.Providers != nil {
		for name, providerConfig := range config.Providers {
			provider, err := createProvider(name, providerConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create provider %s: %w", name, err)
			}
			service.providers[name] = provider
		}
	}
	
	return service, nil
}

// RegisterProvider registers a new OAuth provider
func (s *MultiProviderService) RegisterProvider(name string, provider Provider) error {
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}
	
	s.providers[name] = provider
	return nil
}

// UnregisterProvider removes a provider
func (s *MultiProviderService) UnregisterProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}
	
	delete(s.providers, name)
	return nil
}

// GetProvider retrieves a registered provider by name
func (s *MultiProviderService) GetProvider(name string) (Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	provider, exists := s.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	
	return provider, nil
}

// ListProviders returns a list of registered provider names
func (s *MultiProviderService) ListProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	return names
}

// GetAuthURL generates an authorization URL for the specified provider
func (s *MultiProviderService) GetAuthURL(ctx context.Context, providerName string, opts ...AuthOption) (string, string, error) {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return "", "", err
	}
	
	// Apply options
	options := &authOptions{
		pkceEnabled: s.config.PKCEEnabled,
		pkceMethod:  s.config.PKCEMethod,
	}
	for _, opt := range opts {
		opt(options)
	}
	
	// Generate state
	state, err := s.stateGen.Generate()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}
	
	// Generate PKCE challenge if enabled
	var pkce *PKCEChallenge
	if options.pkceEnabled && provider.SupportsPKCE() {
		pkce, err = GeneratePKCEChallenge(options.pkceMethod)
		if err != nil {
			return "", "", fmt.Errorf("failed to generate PKCE challenge: %w", err)
		}
	}
	
	// Store session data
	sessionData := &SessionData{
		State:         state,
		PKCEChallenge: pkce,
		Provider:      providerName,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(s.config.SessionTimeout),
		Metadata:      options.metadata,
	}
	
	if err := s.sessions.Store(ctx, state, sessionData); err != nil {
		return "", "", fmt.Errorf("failed to store session: %w", err)
	}
	
	// Get authorization URL from provider
	authURL := provider.GetAuthURL(state, pkce)
	return authURL, state, nil
}

// Exchange exchanges an authorization code for tokens
func (s *MultiProviderService) Exchange(ctx context.Context, providerName, code, state string) (*Token, error) {
	// Retrieve and immediately delete session to prevent replay attacks
	sessionData, err := s.sessions.RetrieveAndDelete(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	
	// Validate session hasn't expired
	if sessionData.IsExpired() {
		return nil, fmt.Errorf("%w: session expired", ErrInvalidState)
	}
	
	// Validate state matches
	if sessionData.State != state {
		return nil, fmt.Errorf("%w: state mismatch", ErrInvalidState)
	}
	
	// Validate provider matches
	if sessionData.Provider != providerName {
		return nil, fmt.Errorf("%w: provider mismatch (expected %s, got %s)", 
			ErrInvalidState, sessionData.Provider, providerName)
	}
	
	// Get the provider
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	
	// Exchange code for token
	token, err := provider.Exchange(ctx, code, sessionData.PKCEChallenge)
	if err != nil {
		return nil, err
	}
	
	// Calculate expiration time if not set
	if token.ExpiresAt.IsZero() && token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	
	// Cache token if enabled
	if s.config.TokenCacheDuration > 0 {
		cacheKey := fmt.Sprintf("token:%s:%s", providerName, code)
		s.tokens.Store(ctx, cacheKey, token)
	}
	
	return token, nil
}

// RefreshToken refreshes an access token for the specified provider
func (s *MultiProviderService) RefreshToken(ctx context.Context, providerName, refreshToken string) (*Token, error) {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	
	if !provider.SupportsRefresh() {
		return nil, fmt.Errorf("%w: provider %s doesn't support refresh", ErrNoRefreshToken, providerName)
	}
	
	token, err := provider.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	
	// Calculate expiration time if not set
	if token.ExpiresAt.IsZero() && token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	
	return token, nil
}

// GetUserInfo retrieves user information from the specified provider
func (s *MultiProviderService) GetUserInfo(ctx context.Context, providerName, accessToken string) (*UserInfo, error) {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	
	userInfo, err := provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	
	// Set provider name
	userInfo.Provider = providerName
	
	return userInfo, nil
}

// RevokeToken revokes a token for the specified provider
func (s *MultiProviderService) RevokeToken(ctx context.Context, providerName, token string) error {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return err
	}
	
	return provider.RevokeToken(ctx, token)
}

// ValidateState validates the state parameter for CSRF protection
func (s *MultiProviderService) ValidateState(ctx context.Context, state string) (*SessionData, error) {
	sessionData, err := s.sessions.Retrieve(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
	}
	
	if sessionData.IsExpired() {
		return nil, fmt.Errorf("%w: session expired", ErrInvalidState)
	}
	
	return sessionData, nil
}

// AuthOption defines options for GetAuthURL
type AuthOption func(*authOptions)

type authOptions struct {
	pkceEnabled bool
	pkceMethod  string
	metadata    map[string]interface{}
}

// WithPKCE enables or disables PKCE for this auth request
func WithPKCE(enabled bool) AuthOption {
	return func(o *authOptions) {
		o.pkceEnabled = enabled
	}
}

// WithPKCEMethod sets the PKCE method for this auth request
func WithPKCEMethod(method string) AuthOption {
	return func(o *authOptions) {
		o.pkceMethod = method
	}
}

// WithMetadata adds metadata to the session
func WithMetadata(metadata map[string]interface{}) AuthOption {
	return func(o *authOptions) {
		o.metadata = metadata
	}
}

// createProvider creates a provider instance based on name and config
func createProvider(name string, config ProviderConfig) (Provider, error) {
	// Normalize provider name
	providerType := strings.ToLower(name)
	if config.Type != "" {
		providerType = strings.ToLower(config.Type)
	}
	
	switch providerType {
	case "google":
		provider := NewGoogle(config)
		return provider, nil
	case "github":
		provider := NewGitHub(config)
		return provider, nil
	case "apple":
		provider, err := NewApple(config)
		if err != nil {
			return nil, err
		}
		return provider, nil
	case "twitter":
		provider := NewTwitter(config)
		return provider, nil
	case "custom":
		provider, err := NewCustom(config)
		if err != nil {
			return nil, err
		}
		return provider, nil
	default:
		// Try to create as custom provider
		if config.AuthURL != "" && config.TokenURL != "" {
			provider, err := NewCustom(config)
			if err != nil {
				return nil, err
			}
			return provider, nil
		}
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}