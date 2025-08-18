package oauth

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/config"
	"github.com/gobeaver/beaver-kit/krypto"
)

// Global instance management
var (
	defaultService *Service
	defaultOnce    sync.Once
	defaultErr     error
)

// Service is the main OAuth service
type Service struct {
	config   Config
	provider Provider
	client   HTTPClient
	sessions SessionStore
	tokens   TokenStore
	stateGen StateGenerator
}

// Init initializes the global OAuth service instance
func Init(configs ...Config) error {
	defaultOnce.Do(func() {
		var cfg *Config
		if len(configs) > 0 {
			cfg = &configs[0]
		} else {
			cfg, defaultErr = GetConfig(config.LoadOptions{Prefix: "BEAVER_"})
			if defaultErr != nil {
				return
			}
		}

		defaultService, defaultErr = New(*cfg)
	})

	return defaultErr
}

// New creates a new OAuth service instance
func New(cfg Config) (*Service, error) {
	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: cfg.HTTPTimeout,
	}

	// Create provider config
	providerConfig := ProviderConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       strings.Split(cfg.Scopes, ","),
		HTTPClient:   httpClient,
		Debug:        cfg.Debug,
	}

	// Create provider based on configuration
	var provider Provider
	var err error

	switch cfg.Provider {
	case "google":
		provider, err = NewGoogleProvider(providerConfig)
	case "github":
		provider, err = NewGitHubProvider(providerConfig)
	case "apple":
		providerConfig.TeamID = cfg.AppleTeamID
		providerConfig.KeyID = cfg.AppleKeyID
		providerConfig.PrivateKey = cfg.ApplePrivateKey
		provider, err = NewAppleProvider(providerConfig)
	case "twitter":
		providerConfig.APIVersion = cfg.TwitterAPIVersion
		provider, err = NewTwitterProvider(providerConfig)
	case "custom":
		providerConfig.AuthURL = cfg.AuthURL
		providerConfig.TokenURL = cfg.TokenURL
		providerConfig.UserInfoURL = cfg.UserInfoURL
		provider, err = NewCustomProvider(providerConfig)
	default:
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Create state generator
	var stateGen StateGenerator
	switch cfg.StateGenerator {
	case "uuid":
		stateGen = &UUIDStateGenerator{}
	case "secure", "":
		stateGen = &SecureStateGenerator{}
	default:
		return nil, fmt.Errorf("%w: unknown state generator: %s", ErrInvalidConfig, cfg.StateGenerator)
	}

	// Create session and token stores (in-memory for now)
	sessions := NewMemorySessionStore(5 * time.Minute) // 5 minute session timeout
	tokens := NewMemoryTokenStore(cfg.TokenCacheDuration)

	return &Service{
		config:   cfg,
		provider: provider,
		client:   httpClient,
		sessions: sessions,
		tokens:   tokens,
		stateGen: stateGen,
	}, nil
}

// GetAuthURL generates an authorization URL
func (s *Service) GetAuthURL(ctx context.Context) (string, error) {
	if s == nil {
		return "", ErrNotInitialized
	}

	// Generate state for CSRF protection
	state, err := s.stateGen.Generate()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Generate PKCE challenge if enabled
	var pkce *PKCEChallenge
	if s.config.PKCEEnabled && s.provider.SupportsPKCE() {
		pkce, err = GeneratePKCEChallenge(s.config.PKCEMethod)
		if err != nil {
			return "", fmt.Errorf("failed to generate PKCE challenge: %w", err)
		}
	}

	// Store session data
	sessionData := &SessionData{
		State:         state,
		PKCEChallenge: pkce,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(s.config.StateTimeout),
		Provider:      s.provider.Name(),
	}

	if err := s.sessions.Store(ctx, state, sessionData); err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	// Get authorization URL from provider
	authURL := s.provider.GetAuthURL(state, pkce)
	return authURL, nil
}

// Exchange exchanges an authorization code for tokens
func (s *Service) Exchange(ctx context.Context, code, state string) (*Token, error) {
	if s == nil {
		return nil, ErrNotInitialized
	}

	// Retrieve and immediately delete session to prevent replay attacks
	sessionData, err := s.sessions.RetrieveAndDelete(ctx, state)
	if err != nil {
		// If RetrieveAndDelete is not implemented, fallback to separate operations
		sessionData, err = s.sessions.Retrieve(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidState, err)
		}
		// Immediately delete to prevent replay
		s.sessions.Delete(ctx, state)
	}

	// Validate session hasn't expired
	if sessionData.IsExpired() {
		return nil, fmt.Errorf("%w: session expired", ErrInvalidState)
	}

	// Validate state matches (double-check against timing attacks)
	if sessionData.State != state {
		return nil, fmt.Errorf("%w: state mismatch", ErrInvalidState)
	}

	// Validate provider matches if set
	if sessionData.Provider != "" && sessionData.Provider != s.provider.Name() {
		return nil, fmt.Errorf("%w: provider mismatch", ErrInvalidState)
	}

	// Exchange code for token
	token, err := s.provider.Exchange(ctx, code, sessionData.PKCEChallenge)
	if err != nil {
		return nil, err
	}

	// Calculate expiration time if not set
	if token.ExpiresAt.IsZero() && token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	// Cache token if enabled
	if s.config.TokenCacheDuration > 0 {
		cacheKey := fmt.Sprintf("token:%s:%s", s.provider.Name(), code)
		s.tokens.Store(ctx, cacheKey, token)
	}

	return token, nil
}

// RefreshToken refreshes an access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	if s == nil {
		return nil, ErrNotInitialized
	}

	if !s.provider.SupportsRefresh() {
		return nil, fmt.Errorf("%w: provider %s doesn't support refresh", ErrNoRefreshToken, s.provider.Name())
	}

	token, err := s.provider.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Calculate expiration time if not set
	if token.ExpiresAt.IsZero() && token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return token, nil
}

// GetUserInfo retrieves user information
func (s *Service) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	if s == nil {
		return nil, ErrNotInitialized
	}

	userInfo, err := s.provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	// Set provider name
	userInfo.Provider = s.provider.Name()

	return userInfo, nil
}

// ValidateState validates the state parameter for CSRF protection
func (s *Service) ValidateState(ctx context.Context, state string) error {
	if s == nil {
		return ErrNotInitialized
	}

	sessionData, err := s.sessions.Retrieve(ctx, state)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidState, err)
	}

	if sessionData.IsExpired() {
		return fmt.Errorf("%w: session expired", ErrInvalidState)
	}

	return nil
}

// Provider returns the current provider
func (s *Service) Provider() Provider {
	if s == nil {
		return nil
	}
	return s.provider
}

// Config returns the service configuration
func (s *Service) Config() Config {
	if s == nil {
		return Config{}
	}
	return s.config
}

// Reset clears the global instance (for testing)
func Reset() {
	defaultService = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}

// GetService returns the global OAuth service instance
func GetService() *Service {
	if defaultService == nil {
		Init() // Initialize with defaults if needed
	}
	return defaultService
}

// OAuth is an alias for GetService()
func OAuth() *Service {
	return GetService()
}

// State generator implementations

// SecureStateGenerator generates cryptographically secure state tokens
type SecureStateGenerator struct{}

func (g *SecureStateGenerator) Generate() (string, error) {
	return krypto.GenerateSecureToken(32)
}

// UUIDStateGenerator generates UUID-based state tokens
type UUIDStateGenerator struct{}

func (g *UUIDStateGenerator) Generate() (string, error) {
	// Simple UUID v4 implementation
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// Memory stores for sessions and tokens

// MemorySessionStore implements SessionStore with in-memory storage
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
	ttl      time.Duration
}

func NewMemorySessionStore(ttl time.Duration) *MemorySessionStore {
	store := &MemorySessionStore{
		sessions: make(map[string]*SessionData),
		ttl:      ttl,
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

func (s *MemorySessionStore) Store(ctx context.Context, key string, data *SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[key] = data
	return nil
}

func (s *MemorySessionStore) Retrieve(ctx context.Context, key string) (*SessionData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.sessions[key]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return data, nil
}

func (s *MemorySessionStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, key)
	return nil
}

func (s *MemorySessionStore) RetrieveAndDelete(ctx context.Context, key string) (*SessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	data, ok := s.sessions[key]
	if !ok {
		return nil, ErrSessionNotFound
	}
	
	// Immediately delete to prevent replay
	delete(s.sessions, key)
	
	return data, nil
}

func (s *MemorySessionStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, session := range s.sessions {
			if session.IsExpired() || now.Sub(session.CreatedAt) > s.ttl {
				delete(s.sessions, key)
			}
		}
		s.mu.Unlock()
	}
}

// MemoryTokenStore implements TokenStore with in-memory storage
type MemoryTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*Token
	ttl    time.Duration
}

func NewMemoryTokenStore(ttl time.Duration) *MemoryTokenStore {
	return &MemoryTokenStore{
		tokens: make(map[string]*Token),
		ttl:    ttl,
	}
}

func (s *MemoryTokenStore) Store(ctx context.Context, key string, token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[key] = token
	return nil
}

func (s *MemoryTokenStore) Retrieve(ctx context.Context, key string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.tokens[key]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}
	return token, nil
}

func (s *MemoryTokenStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, key)
	return nil
}
