# OAuth Package

A flexible and secure OAuth 2.0 client implementation for Go applications, supporting multiple providers with PKCE (Proof Key for Code Exchange) for enhanced security.

## Features

- 🔐 **Multiple OAuth Providers** - GitHub, Google (coming soon), Apple (coming soon), Twitter (coming soon)
- 🛡️ **PKCE Support** - Enhanced security with Proof Key for Code Exchange
- ⚙️ **Environment Configuration** - Easy setup via environment variables
- 🔄 **Token Management** - Automatic token handling with refresh support
- 📦 **Session Management** - Built-in session store for OAuth state
- 🎯 **Type Safe** - Strongly typed interfaces and configurations
- 🧪 **Well Tested** - Comprehensive test coverage
- 🏗️ **Builder Pattern** - Flexible configuration with custom prefixes

## Installation

```bash
go get github.com/gobeaver/beaver-kit/oauth
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/gobeaver/beaver-kit/oauth"
)

func main() {
    // Create OAuth configuration
    config := oauth.Config{
        Provider:     "github",
        ClientID:     "your_github_client_id",
        ClientSecret: "your_github_client_secret",
        RedirectURL:  "http://localhost:8080/callback",
    }
    
    // Initialize service
    service, err := oauth.New(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use the service for OAuth flows
    // ... see examples below
}
```

### Environment Configuration

Configure via environment variables with the `BEAVER_OAUTH_` prefix:

```bash
# OAuth Provider Configuration
export BEAVER_OAUTH_PROVIDER=github
export BEAVER_OAUTH_CLIENT_ID=your_client_id
export BEAVER_OAUTH_CLIENT_SECRET=your_client_secret
export BEAVER_OAUTH_REDIRECT_URL=http://localhost:8080/callback

# Optional Configuration
export BEAVER_OAUTH_SCOPES=read:user,user:email
export BEAVER_OAUTH_PKCE_ENABLED=true
export BEAVER_OAUTH_DEBUG=false
```

Load from environment:

```go
import (
    "github.com/gobeaver/beaver-kit/config"
    "github.com/gobeaver/beaver-kit/oauth"
)

// Load configuration from environment
cfg := &oauth.Config{}
if err := config.LoadFromEnv(cfg); err != nil {
    log.Fatal(err)
}

// Initialize service
service, err := oauth.New(*cfg)
```

### Custom Environment Prefix

Use a custom prefix for multi-tenant applications:

```go
// Use custom prefix
cfg := &oauth.Config{}
if err := config.LoadFromEnv(cfg, config.WithPrefix("MYAPP_")); err != nil {
    log.Fatal(err)
}
// Now reads from MYAPP_OAUTH_CLIENT_ID, etc.
```

## OAuth Flow Implementation

### 1. Generate Authorization URL

```go
// Start OAuth flow
authURL, state, err := service.GenerateAuthURL("github", true) // true enables PKCE
if err != nil {
    log.Fatal(err)
}

// Store state in session for CSRF protection
// Redirect user to authURL
```

### 2. Handle OAuth Callback

```go
func handleCallback(w http.ResponseWriter, r *http.Request) {
    // Handle the OAuth callback
    resp, err := service.HandleCallback(r, "github")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Verify state matches (CSRF protection)
    if resp.State != storedState {
        http.Error(w, "Invalid state", http.StatusBadRequest)
        return
    }
    
    // Get user information
    userInfo, err := service.GetUserInfo(context.Background(), "github", resp.Token.AccessToken)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // User is authenticated!
    log.Printf("User: %s (%s)", userInfo.Name, userInfo.Email)
}
```

### 3. Direct Provider Usage

```go
// Use provider directly for more control
provider := oauth.NewGitHub(oauth.ProviderConfig{
    ClientID:     "your_client_id",
    ClientSecret: "your_client_secret",
    RedirectURL:  "http://localhost:8080/callback",
    Scopes:       []string{"read:user", "user:email"},
})

// Generate auth URL with PKCE
pkce, err := oauth.GeneratePKCEChallenge("S256")
if err != nil {
    log.Fatal(err)
}

authURL := provider.GetAuthURL("state_token", pkce)

// After callback, exchange code for token
token, err := provider.Exchange(context.Background(), "auth_code", pkce)
if err != nil {
    log.Fatal(err)
}

// Get user info
userInfo, err := provider.GetUserInfo(context.Background(), token.AccessToken)
```

## PKCE (Proof Key for Code Exchange)

PKCE provides additional security for OAuth flows, especially important for public clients:

```go
// Generate PKCE challenge
pkce, err := oauth.GeneratePKCEChallenge("S256") // or "plain"
if err != nil {
    log.Fatal(err)
}

// Include in authorization URL
authURL := provider.GetAuthURL(state, pkce)

// Include verifier when exchanging code
token, err := provider.Exchange(ctx, code, pkce)
```

## Provider-Specific Features

### GitHub Provider

```go
provider := oauth.NewGitHub(oauth.ProviderConfig{
    ClientID:     "github_client_id",
    ClientSecret: "github_client_secret",
    RedirectURL:  "http://localhost:8080/callback/github",
    Scopes:       []string{"read:user", "user:email"},
})

// GitHub-specific features:
// - Automatic private email retrieval
// - No refresh token support (GitHub tokens don't expire)
// - PKCE support for enhanced security
```

## Token Management

### Caching Tokens

```go
// Cache token for later use
err := service.CacheToken(userID, "github", token)

// Retrieve cached token
cachedToken, err := service.GetCachedToken(userID, "github")
if err != nil {
    // Token not found or expired
}

// Clear cached token
err = service.ClearCachedToken(userID, "github")
```

### Token Refresh

For providers that support refresh tokens:

```go
// Refresh access token
newToken, err := provider.RefreshToken(ctx, refreshToken)
if err != nil {
    if err == oauth.ErrNoRefreshToken {
        // Provider doesn't support refresh (e.g., GitHub)
    }
}
```

## Error Handling

The package provides detailed error types for OAuth-specific errors:

```go
// Handle OAuth errors
resp, err := service.HandleCallback(r, "github")
if err != nil {
    var oauthErr *oauth.OAuthError
    if errors.As(err, &oauthErr) {
        log.Printf("OAuth error: %s - %s", oauthErr.Code, oauthErr.Description)
        
        switch oauthErr.Code {
        case "access_denied":
            // User denied access
        case "invalid_grant":
            // Invalid authorization code
        }
    }
}

// Common errors
var (
    ErrInvalidState      // State parameter mismatch (CSRF)
    ErrProviderNotFound  // Unknown provider
    ErrNoRefreshToken    // Provider doesn't support refresh
    ErrSessionNotFound   // Session data not found
    ErrTokenExpired      // Access token expired
)
```

## Configuration Options

### Full Configuration

```go
type Config struct {
    // OAuth provider (github, google, etc.)
    Provider string
    
    // OAuth app credentials
    ClientID     string
    ClientSecret string
    RedirectURL  string
    
    // Comma-separated scopes
    Scopes string
    
    // State generation and validation
    State          string
    StateGenerator string // "uuid", "secure", "custom"
    
    // PKCE settings
    PKCEEnabled bool
    PKCEMethod  string // "S256" or "plain"
    
    // Token caching
    TokenCacheDuration time.Duration
    
    // HTTP settings
    HTTPTimeout time.Duration
    
    // Debug mode
    Debug bool
    
    // Custom provider URLs
    AuthURL     string
    TokenURL    string
    UserInfoURL string
}
```

### Provider Configuration

```go
type ProviderConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
    
    // Optional: Override default URLs
    AuthURL     string
    TokenURL    string
    UserInfoURL string
    
    // Optional: Custom HTTP client
    HTTPClient HTTPClient
    
    // Provider-specific settings
    TeamID     string // Apple
    KeyID      string // Apple
    PrivateKey string // Apple
    APIVersion string // Twitter
}
```

## Testing

The package includes comprehensive tests. To run tests:

```bash
go test ./oauth/...
```

Example test:

```go
func TestOAuthFlow(t *testing.T) {
    // Create test provider
    provider := oauth.NewGitHub(oauth.ProviderConfig{
        ClientID:     "test_id",
        ClientSecret: "test_secret",
        RedirectURL:  "http://test.local/callback",
    })
    
    // Test auth URL generation
    authURL := provider.GetAuthURL("test_state", nil)
    assert.Contains(t, authURL, "client_id=test_id")
    
    // Test PKCE
    pkce, err := oauth.GeneratePKCEChallenge("S256")
    assert.NoError(t, err)
    assert.NotEmpty(t, pkce.Challenge)
}
```

## Security Best Practices

1. **Always use PKCE** when available, especially for public clients
2. **Validate state parameter** to prevent CSRF attacks
3. **Use HTTPS** in production for redirect URLs
4. **Store tokens securely** - never in client-side storage
5. **Implement token rotation** when refresh tokens are available
6. **Validate scopes** returned by the provider
7. **Set appropriate timeouts** for HTTP requests

## Supported Providers

| Provider | Status | Refresh Token | PKCE Support | Notes |
|----------|--------|---------------|--------------|-------|
| GitHub | ✅ Complete | ❌ | ✅ | Tokens don't expire |
| Google | 🚧 Coming Soon | ✅ | ✅ | OpenID Connect |
| Apple | 📋 Planned | ✅ | ✅ | Requires additional config |
| Twitter | 📋 Planned | ✅ | ❌ | OAuth 2.0 support |
| Custom | 📋 Planned | Varies | Varies | Generic OAuth 2.0 |

## Advanced Usage

### Custom State Generator

```go
// Implement custom state generator
type MyStateGenerator struct{}

func (g *MyStateGenerator) Generate() (string, error) {
    // Your custom implementation
    return "custom_state_" + uuid.New().String(), nil
}

// Use in service
service.SetStateGenerator(&MyStateGenerator{})
```

### Custom Session Store

```go
// Implement custom session store (e.g., Redis)
type RedisSessionStore struct {
    client *redis.Client
}

func (s *RedisSessionStore) Store(ctx context.Context, key string, data *oauth.SessionData) error {
    // Store in Redis
}

func (s *RedisSessionStore) Retrieve(ctx context.Context, key string) (*oauth.SessionData, error) {
    // Retrieve from Redis
}

func (s *RedisSessionStore) Delete(ctx context.Context, key string) error {
    // Delete from Redis
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This package is part of the Beaver Kit project and follows its licensing terms.