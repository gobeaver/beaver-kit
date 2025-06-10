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
	State         string         `json:"state"`
	PKCEChallenge *PKCEChallenge `json:"pkce_challenge,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ExpiresAt     time.Time      `json:"expires_at"`
	Provider      string         `json:"provider"`
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
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	HTTPClient   HTTPClient
	Debug        bool
	
	// Apple-specific
	TeamID     string
	KeyID      string
	PrivateKey string
	
	// Twitter-specific
	APIVersion string
}