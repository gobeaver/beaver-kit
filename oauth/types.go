package oauth

import (
	"context"
	"net/http"
	"time"
)

// Provider defines the interface that all OAuth providers must implement
type Provider interface {
	// GetAuthURL returns the authorization URL with PKCE parameters if enabled
	GetAuthURL(state string, pkce *PKCEChallenge) string

	// Exchange exchanges an authorization code for tokens
	Exchange(ctx context.Context, code string, pkce *PKCEChallenge) (*Token, error)

	// RefreshToken refreshes an access token using a refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*Token, error)

	// GetUserInfo retrieves user information using an access token
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)

	// Name returns the provider name
	Name() string

	// SupportsRefresh indicates if the provider supports token refresh
	SupportsRefresh() bool

	// SupportsPKCE indicates if the provider supports PKCE
	SupportsPKCE() bool
	
	// RevokeToken revokes an access or refresh token
	RevokeToken(ctx context.Context, token string) error
	
	// ValidateConfig validates the provider configuration
	ValidateConfig() error
}

// Token represents OAuth tokens
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	IDToken      string    `json:"id_token,omitempty"` // For OpenID Connect
	Scope        string    `json:"scope,omitempty"`
}

// IsExpired checks if the token is expired
func (t *Token) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(t.ExpiresAt)
}

// TimeUntilExpiry returns the duration until the token expires
func (t *Token) TimeUntilExpiry() time.Duration {
	if t.ExpiresAt.IsZero() {
		return 0
	}
	return time.Until(t.ExpiresAt)
}

// UserInfo represents user information from OAuth providers
type UserInfo struct {
	ID            string                 `json:"id"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	Name          string                 `json:"name"`
	FirstName     string                 `json:"first_name"`
	LastName      string                 `json:"last_name"`
	Picture       string                 `json:"picture"`
	Locale        string                 `json:"locale"`
	Provider      string                 `json:"provider"`
	Raw           map[string]interface{} `json:"raw"` // Raw response from provider
}

// PKCEChallenge represents PKCE challenge parameters
type PKCEChallenge struct {
	Verifier        string `json:"verifier"`
	Challenge       string `json:"challenge"`
	ChallengeMethod string `json:"challenge_method"`
}

// AuthorizationRequest represents an OAuth authorization request
type AuthorizationRequest struct {
	State         string
	PKCEChallenge *PKCEChallenge
	RedirectURL   string
	Scopes        []string
	ExtraParams   map[string]string
}

// AuthorizationResponse represents the response from an OAuth authorization
type AuthorizationResponse struct {
	Code  string
	State string
	Error string
}

// SessionData represents OAuth session data that can be stored
type SessionData struct {
	State         string                 `json:"state"`
	PKCEChallenge *PKCEChallenge         `json:"pkce_challenge,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
	Provider      string                 `json:"provider"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// IsExpired checks if the session data is expired
func (s *SessionData) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// HTTPClient interface for mocking in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// StateGenerator interface for generating state tokens
type StateGenerator interface {
	Generate() (string, error)
}

// SessionStore interface for storing OAuth session data
type SessionStore interface {
	// Store stores session data with a key
	Store(ctx context.Context, key string, data *SessionData) error

	// Retrieve gets session data by key
	Retrieve(ctx context.Context, key string) (*SessionData, error)

	// Delete removes session data by key
	Delete(ctx context.Context, key string) error

	// RetrieveAndDelete atomically retrieves and deletes session data
	// This prevents replay attacks by ensuring a session can only be used once
	RetrieveAndDelete(ctx context.Context, key string) (*SessionData, error)
}

// TokenStore interface for caching OAuth tokens
type TokenStore interface {
	// Store stores a token with a key
	Store(ctx context.Context, key string, token *Token) error

	// Retrieve gets a token by key
	Retrieve(ctx context.Context, key string) (*Token, error)

	// Delete removes a token by key
	Delete(ctx context.Context, key string) error
}

// ProviderConfig represents configuration for a specific OAuth provider
type ProviderConfig struct {
	// Provider type (google, github, apple, twitter, custom)
	Type         string     `json:"type,omitempty" env:"TYPE"`
	ClientID     string     `json:"client_id" env:"CLIENT_ID"`
	ClientSecret string     `json:"client_secret,omitempty" env:"CLIENT_SECRET"`
	RedirectURL  string     `json:"redirect_url" env:"REDIRECT_URL"`
	Scopes       []string   `json:"scopes,omitempty" env:"SCOPES"`
	AuthURL      string     `json:"auth_url,omitempty" env:"AUTH_URL"`
	TokenURL     string     `json:"token_url,omitempty" env:"TOKEN_URL"`
	UserInfoURL  string     `json:"userinfo_url,omitempty" env:"USERINFO_URL"`
	RevokeURL    string     `json:"revoke_url,omitempty" env:"REVOKE_URL"`
	HTTPClient   HTTPClient `json:"-"`
	Debug        bool       `json:"debug,omitempty" env:"DEBUG"`
	
	// Apple-specific
	TeamID     string `json:"team_id,omitempty" env:"APPLE_TEAM_ID"`
	KeyID      string `json:"key_id,omitempty" env:"APPLE_KEY_ID"`
	PrivateKey string `json:"private_key,omitempty" env:"APPLE_PRIVATE_KEY"`
	
	// Twitter-specific
	APIVersion string `json:"api_version,omitempty" env:"TWITTER_API_VERSION"`
}