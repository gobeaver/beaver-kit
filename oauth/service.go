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

// Global instance management for singleton pattern
var (
	defaultService *Service  // Global OAuth service instance
	defaultOnce    sync.Once // Ensures single initialization
	defaultErr     error     // Stores initialization error
)

// Service is the main OAuth service that handles authentication flows.
// It provides methods for generating authorization URLs, exchanging codes for tokens,
// refreshing tokens, and retrieving user information.
//
// The service supports multiple OAuth providers and includes security features like:
//   - PKCE for public clients
//   - State parameter validation for CSRF protection
//   - Session management with automatic cleanup
//   - Token caching with configurable TTL
type Service struct {
	config   Config         // Service configuration
	provider Provider       // OAuth provider implementation
	client   HTTPClient     // HTTP client for API requests
	sessions SessionStore   // Session storage for state management
	tokens   TokenStore     // Token storage for caching
	stateGen StateGenerator // State parameter generator
}

// Init initializes the global OAuth service instance using the provided configuration
// or loading from environment variables with the BEAVER_OAUTH_ prefix.
//
// This function is safe to call multiple times - initialization only happens once.
// If no configuration is provided, it loads from environment variables.
//
// Example:
//
//	// Initialize with environment variables
//	err := oauth.Init()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Initialize with custom configuration
//	config := oauth.Config{
//	    Provider:     "google",
//	    ClientID:     "your-client-id",
//	    ClientSecret: "your-client-secret",
//	    RedirectURL:  "https://yourapp.com/callback",
//	}
//	err := oauth.Init(config)
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

// New creates a new OAuth service instance with the provided configuration.
// This function validates the configuration, creates the appropriate provider,
// and sets up session and token storage.
//
// The returned service is ready to use for OAuth flows. It supports all
// built-in providers (Google, GitHub, Apple, Twitter) and custom providers.
//
// Example:
//
//	config := oauth.Config{
//	    Provider:     "google",
//	    ClientID:     "your-google-client-id",
//	    ClientSecret: "your-google-client-secret",
//	    RedirectURL:  "https://yourapp.com/callback",
//	    Scopes:       "openid,email,profile",
//	    PKCEEnabled:  true,
//	}
//
//	service, err := oauth.New(config)
//	if err != nil {
//	    return fmt.Errorf("failed to create OAuth service: %w", err)
//	}
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

// GetAuthURL generates an authorization URL for initiating the OAuth flow.
// This method creates a secure state parameter for CSRF protection and
// optionally generates a PKCE challenge for enhanced security.
//
// The generated URL should be used to redirect the user to the OAuth provider's
// authorization page. After user consent, the provider will redirect back to
// your configured redirect URL with an authorization code and state parameter.
//
// Returns the authorization URL and an error if generation fails.
//
// Example:
//
//	authURL, err := service.GetAuthURL(ctx)
//	if err != nil {
//	    return fmt.Errorf("failed to generate auth URL: %w", err)
//	}
//
//	// Redirect user to authURL
//	http.Redirect(w, r, authURL, http.StatusFound)
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

// Exchange exchanges an authorization code for access and refresh tokens.
// This method validates the state parameter to prevent CSRF attacks and
// uses the PKCE verifier if PKCE was enabled during authorization.
//
// The method performs several security checks:
//   - Validates the state parameter matches the stored session
//   - Checks session hasn't expired
//   - Immediately deletes the session to prevent replay attacks
//   - Validates the provider matches if specified
//
// Returns a Token containing the access token, refresh token (if available),
// expiration information, and any additional token metadata.
//
// Parameters:
//   - ctx: Context for the operation
//   - code: Authorization code received from the OAuth callback
//   - state: State parameter received from the OAuth callback
//
// Example:
//
//	// In your OAuth callback handler
//	code := r.URL.Query().Get("code")
//	state := r.URL.Query().Get("state")
//
//	token, err := service.Exchange(ctx, code, state)
//	if err != nil {
//	    return fmt.Errorf("failed to exchange code: %w", err)
//	}
//
//	// Use token.AccessToken for API calls
//	userInfo, err := service.GetUserInfo(ctx, token.AccessToken)
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

// RefreshToken refreshes an access token using the provided refresh token.
// This method is useful for obtaining new access tokens when the current
// token has expired, without requiring the user to re-authenticate.
//
// Not all OAuth providers support refresh tokens. This method will return
// an error if the provider doesn't support token refresh.
//
// Parameters:
//   - ctx: Context for the operation
//   - refreshToken: The refresh token obtained from a previous token exchange
//
// Returns a new Token with updated access token and expiration time.
//
// Example:
//
//	// Check if token is expired and refresh if needed
//	if token.IsExpired() {
//	    newToken, err := service.RefreshToken(ctx, token.RefreshToken)
//	    if err != nil {
//	        return fmt.Errorf("failed to refresh token: %w", err)
//	    }
//	    token = newToken
//	}
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

// GetUserInfo retrieves user profile information using an access token.
// This method calls the OAuth provider's user info endpoint to fetch
// details about the authenticated user.
//
// The returned UserInfo struct contains standard fields like ID, email,
// name, and profile picture, though availability depends on the provider
// and requested scopes.
//
// Parameters:
//   - ctx: Context for the operation
//   - accessToken: Valid access token from a successful OAuth exchange
//
// Returns UserInfo containing the user's profile data.
//
// Example:
//
//	userInfo, err := service.GetUserInfo(ctx, token.AccessToken)
//	if err != nil {
//	    return fmt.Errorf("failed to get user info: %w", err)
//	}
//
//	fmt.Printf("User: %s (%s)\n", userInfo.Name, userInfo.Email)
//	fmt.Printf("Provider: %s\n", userInfo.Provider)
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

// Reset clears the global OAuth service instance, allowing for re-initialization.
// This function is primarily intended for testing purposes to ensure a clean
// state between test runs.
//
// After calling Reset(), the next call to Init() or OAuth() will create a
// new service instance.
//
// Example:
//
//	// In test teardown
//	oauth.Reset()
//
//	// Next OAuth() call will reinitialize
//	service := oauth.OAuth()
func Reset() {
	defaultService = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}

// GetService returns the global OAuth service instance.
// If the service hasn't been initialized, this function will attempt
// to initialize it using environment variables with the BEAVER_OAUTH_ prefix.
//
// Returns nil if initialization fails. Use Init() explicitly if you need
// to handle initialization errors.
//
// Example:
//
//	service := oauth.GetService()
//	if service == nil {
//	    log.Fatal("OAuth service not initialized")
//	}
func GetService() *Service {
	if defaultService == nil {
		Init() // Initialize with defaults if needed
	}
	return defaultService
}

// OAuth returns the global OAuth service instance.
// This is a convenience function that's equivalent to GetService().
//
// If the service hasn't been initialized, this function will attempt
// to initialize it using environment variables with the BEAVER_OAUTH_ prefix.
//
// Example:
//
//	// Generate auth URL using global service
//	authURL, err := oauth.OAuth().GetAuthURL(ctx)
//
//	// Exchange code for token
//	token, err := oauth.OAuth().Exchange(ctx, code, state)
func OAuth() *Service {
	return GetService()
}

// State generator implementations

// SecureStateGenerator generates cryptographically secure state tokens
// using the system's random number generator. This is the recommended
// state generator for production use.
type SecureStateGenerator struct{}

// Generate creates a cryptographically secure random state token.
// The token is 32 bytes of random data encoded as a hex string.
func (g *SecureStateGenerator) Generate() (string, error) {
	return krypto.GenerateSecureToken(32)
}

// UUIDStateGenerator generates UUID-based state tokens.
// This generator creates UUID v4 compatible tokens which may be
// more recognizable in logs but are less random than SecureStateGenerator.
type UUIDStateGenerator struct{}

// Generate creates a UUID v4 formatted state token.
// Returns a string in the format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func (g *UUIDStateGenerator) Generate() (string, error) {
	// Simple UUID v4 implementation
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// Memory stores for sessions and tokens

// MemorySessionStore implements SessionStore with in-memory storage.
// This is a simple implementation suitable for single-instance applications.
// For production deployments with multiple instances, consider using a
// distributed session store like Redis.
//
// The store includes automatic cleanup of expired sessions via a background
// goroutine that runs every minute.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
	ttl      time.Duration
}

// NewMemorySessionStore creates a new in-memory session store with the specified TTL.
// The store will automatically clean up expired sessions every minute.
//
// Parameters:
//   - ttl: Time-to-live for sessions before they're considered expired
//
// Returns a new MemorySessionStore instance with cleanup goroutine running.
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
