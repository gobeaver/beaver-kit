# OAuth Package - Production-Ready OAuth 2.0 & OpenID Connect

A comprehensive, production-ready OAuth 2.0 and OpenID Connect implementation for Go applications. Part of the Beaver Kit ecosystem.

## Features

### 🔐 Security First
- **Apple JWT Validation**: Full ECDSA/RSA signature verification with public key caching
- **PKCE Support**: RFC 7636 compliant implementation (43-128 character verifiers)
- **State Management**: Replay attack prevention with immediate session deletion
- **Token Encryption**: AES-GCM encryption for sensitive token storage
- **Security Headers**: CORS, HSTS, CSP, XSS protection middleware

### 🌐 Multi-Provider Architecture
- **Built-in Providers**: Google, GitHub, Apple, Twitter
- **Custom Provider Support**: Easy integration with any OAuth 2.0 provider
- **Dynamic Registration**: Add/remove providers at runtime
- **Provider Isolation**: Circuit breakers per provider

### 💾 Advanced Token Management
- **Automatic Refresh**: Tokens refreshed before expiration
- **Encrypted Storage**: Secure token persistence
- **Bulk Operations**: Efficient cleanup of expired tokens
- **User Limits**: Configurable token limits per user
- **Cache Integration**: Built-in caching support

### 🛡️ Production Hardening
- **Rate Limiting**: Token bucket and sliding window algorithms
- **Circuit Breakers**: Protect against cascading failures
- **Health Checks**: Liveness, readiness, and component health endpoints
- **Monitoring**: Comprehensive metrics collection
- **Request Logging**: Structured logging with sensitive data redaction

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

### Google Provider

```go
provider := oauth.NewGoogle(oauth.ProviderConfig{
    ClientID:     "google_client_id",
    ClientSecret: "google_client_secret",
    RedirectURL:  "http://localhost:8080/callback/google",
    Scopes:       []string{"openid", "profile", "email"},
})

// Google-specific features:
// - OpenID Connect support with ID tokens
// - Refresh token support (offline access)
// - PKCE support for enhanced security
// - Perfect for PWA and Flutter apps
```

### Apple Provider

```go
provider, err := oauth.NewApple(oauth.ProviderConfig{
    ClientID:     "com.yourapp.serviceid",
    RedirectURL:  "https://yourapp.com/callback/apple",
    TeamID:       "YOUR_TEAM_ID",
    KeyID:        "YOUR_KEY_ID", 
    PrivateKey:   "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
    Scopes:       []string{"name", "email"},
})

// Apple-specific features:
// - Requires Apple Developer Program enrollment
// - Uses JWT-based client authentication
// - ID token contains user information (no separate userinfo endpoint)
// - PKCE support for enhanced security
// - form_post response mode for better security
```

### Twitter (X) Provider

```go
provider := oauth.NewTwitter(oauth.ProviderConfig{
    ClientID:     "twitter_client_id",
    ClientSecret: "twitter_client_secret", // Optional for public clients
    RedirectURL:  "https://yourapp.com/callback/twitter",
    Scopes:       []string{"tweet.read", "users.read", "offline.access"},
})

// Twitter-specific features:
// - OAuth 2.0 with PKCE (required for public clients)
// - API v2 support with enhanced user fields
// - No email in user profile (separate permission required)
// - Rich user metadata (followers, tweets, etc.)
```

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
| Google | ✅ Complete | ✅ | ✅ | OpenID Connect, perfect for PWA/Flutter |
| Apple | ✅ Complete | ✅ | ✅ | JWT client auth, ID tokens, iOS apps |
| Twitter | ✅ Complete | ✅ | ✅ | OAuth 2.0 API v2, social integration |
| Custom | 📋 Planned | Varies | Varies | Generic OAuth 2.0 |

## PWA and Flutter App Integration

The OAuth package is specifically designed to work well with Progressive Web Apps (PWA) and Flutter applications that need secure OAuth flows:

### Google OAuth for PWA/Flutter

```go
// Perfect configuration for PWA/Flutter apps
provider := oauth.NewGoogle(oauth.ProviderConfig{
    ClientID:     "your_app.googleusercontent.com",
    ClientSecret: "your_client_secret", // For server-side flow
    RedirectURL:  "https://yourapp.com/auth/callback",
    Scopes:       []string{"openid", "profile", "email"},
})

// Always use PKCE for enhanced security in public clients
pkce, err := oauth.GeneratePKCEChallenge("S256")
if err != nil {
    log.Fatal(err)
}

// Generate authorization URL with PKCE
authURL := provider.GetAuthURL(state, pkce)
// Redirect user or open in browser

// After callback, exchange with PKCE verifier
token, err := provider.Exchange(ctx, authCode, pkce)
if err != nil {
    log.Fatal(err)
}

// Parse ID token for additional user claims
if token.IDToken != "" {
    claims, err := provider.ParseIDToken(token.IDToken)
    if err == nil {
        // Access additional user information from ID token
        fmt.Printf("User email from ID token: %v", claims["email"])
    }
}
```

### Key Benefits for PWA/Flutter:

1. **PKCE Security** - Essential for public clients (PWA/mobile apps)
2. **Refresh Tokens** - Google provides refresh tokens for offline access
3. **ID Tokens** - OpenID Connect support for additional user claims
4. **Cross-Platform** - Same backend code works for web and mobile
5. **Secure Flow** - Server-side token exchange prevents token exposure

### Deployment Considerations:

- **PWA**: Use server-side callback URL, handle PKCE on client
- **Flutter**: Integrate with deep links for callback handling  
- **Security**: Always validate state parameter and use HTTPS
- **Tokens**: Store refresh tokens securely, never in client storage

## Integration with Authentication Systems

The OAuth package is designed to integrate seamlessly with authentication systems and auth flows:

### Complete Auth Flow Implementation

```go
package auth

import (
    "context"
    "net/http"
    "time"
    
    "github.com/gobeaver/beaver-kit/oauth"
    "github.com/gobeaver/beaver-kit/krypto"
)

type AuthService struct {
    oauth   *oauth.Service
    users   UserRepository
    tokens  TokenRepository
}

func NewAuthService() *AuthService {
    // Initialize OAuth service
    oauthService := oauth.New()
    config := oauth.Config{
        Providers: map[string]oauth.ProviderConfig{
            "google": {
                ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
                ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
                RedirectURL:  "https://yourapp.com/auth/google/callback",
                Scopes:       []string{"openid", "profile", "email"},
            },
            "github": {
                ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
                ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
                RedirectURL:  "https://yourapp.com/auth/github/callback",
                Scopes:       []string{"read:user", "user:email"},
            },
        },
    }
    oauthService.Init(config)
    
    return &AuthService{
        oauth: oauthService,
        users: NewUserRepository(),
        tokens: NewTokenRepository(),
    }
}

// StartOAuthFlow initiates OAuth authentication
func (a *AuthService) StartOAuthFlow(provider string, w http.ResponseWriter, r *http.Request) error {
    // Generate auth URL with PKCE
    authURL, state, err := a.oauth.GenerateAuthURL(provider, true)
    if err != nil {
        return err
    }
    
    // Store state and PKCE in session for callback verification
    session := &AuthSession{
        State:     state,
        Provider:  provider,
        ExpiresAt: time.Now().Add(10 * time.Minute),
    }
    
    // Store session (in production, use Redis/database)
    a.storeSession(r, session)
    
    // Redirect to OAuth provider
    http.Redirect(w, r, authURL, http.StatusFound)
    return nil
}

// HandleOAuthCallback processes OAuth callback and completes authentication
func (a *AuthService) HandleOAuthCallback(provider string, w http.ResponseWriter, r *http.Request) (*User, error) {
    // Retrieve and validate session
    session, err := a.getSession(r)
    if err != nil {
        return nil, fmt.Errorf("invalid session: %w", err)
    }
    
    if session.Provider != provider {
        return nil, fmt.Errorf("provider mismatch")
    }
    
    // Handle OAuth callback
    resp, err := a.oauth.HandleCallback(r, provider)
    if err != nil {
        return nil, fmt.Errorf("oauth callback failed: %w", err)
    }
    
    // Verify state (CSRF protection)
    if resp.State != session.State {
        return nil, fmt.Errorf("state mismatch - possible CSRF attack")
    }
    
    // Get user information from OAuth provider
    userInfo, err := a.oauth.GetUserInfo(r.Context(), provider, resp.Token.AccessToken)
    if err != nil {
        return nil, fmt.Errorf("failed to get user info: %w", err)
    }
    
    // Find or create user in your system
    user, err := a.findOrCreateUser(userInfo, provider)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    // Cache OAuth token for API calls
    err = a.oauth.CacheToken(user.ID, provider, resp.Token)
    if err != nil {
        // Log error but don't fail auth
        log.Printf("Failed to cache OAuth token: %v", err)
    }
    
    // Generate your application's session token
    sessionToken, err := a.generateSessionToken(user)
    if err != nil {
        return nil, fmt.Errorf("failed to generate session: %w", err)
    }
    
    // Set session cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session",
        Value:    sessionToken,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
    })
    
    return user, nil
}

// findOrCreateUser handles user registration/login
func (a *AuthService) findOrCreateUser(userInfo *oauth.UserInfo, provider string) (*User, error) {
    // Check if user exists by OAuth provider ID
    user, err := a.users.FindByOAuthID(provider, userInfo.ID)
    if err == nil {
        // Existing user - update their info
        user.Email = userInfo.Email
        user.Name = userInfo.Name
        user.Picture = userInfo.Picture
        user.LastLoginAt = time.Now()
        return a.users.Update(user)
    }
    
    // Check if user exists by email
    if userInfo.Email != "" {
        user, err = a.users.FindByEmail(userInfo.Email)
        if err == nil {
            // Link OAuth account to existing user
            oauthAccount := &OAuthAccount{
                UserID:     user.ID,
                Provider:   provider,
                ProviderID: userInfo.ID,
                Email:      userInfo.Email,
            }
            a.users.AddOAuthAccount(oauthAccount)
            return user, nil
        }
    }
    
    // Create new user
    user = &User{
        Email:       userInfo.Email,
        Name:        userInfo.Name,
        Picture:     userInfo.Picture,
        EmailVerified: userInfo.EmailVerified,
        CreatedAt:   time.Now(),
        LastLoginAt: time.Now(),
    }
    
    user, err = a.users.Create(user)
    if err != nil {
        return nil, err
    }
    
    // Create OAuth account link
    oauthAccount := &OAuthAccount{
        UserID:     user.ID,
        Provider:   provider,
        ProviderID: userInfo.ID,
        Email:      userInfo.Email,
    }
    
    err = a.users.AddOAuthAccount(oauthAccount)
    if err != nil {
        return nil, err
    }
    
    return user, nil
}

// generateSessionToken creates a secure session token
func (a *AuthService) generateSessionToken(user *User) (string, error) {
    claims := map[string]interface{}{
        "user_id": user.ID,
        "email":   user.Email,
        "exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
        "iat":     time.Now().Unix(),
    }
    
    // Use krypto package to generate JWT
    return krypto.GenerateJWT(claims)
}

// Middleware for protecting routes
func (a *AuthService) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Get session token from cookie
        cookie, err := r.Cookie("session")
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Validate session token
        claims, err := krypto.ValidateJWT(cookie.Value)
        if err != nil {
            http.Error(w, "Invalid session", http.StatusUnauthorized)
            return
        }
        
        // Get user from claims
        userID, ok := claims["user_id"].(string)
        if !ok {
            http.Error(w, "Invalid session", http.StatusUnauthorized)
            return
        }
        
        user, err := a.users.FindByID(userID)
        if err != nil {
            http.Error(w, "User not found", http.StatusUnauthorized)
            return
        }
        
        // Add user to request context
        ctx := context.WithValue(r.Context(), "user", user)
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}

// Helper types
type User struct {
    ID            string    `json:"id"`
    Email         string    `json:"email"`
    Name          string    `json:"name"`
    Picture       string    `json:"picture"`
    EmailVerified bool      `json:"email_verified"`
    CreatedAt     time.Time `json:"created_at"`
    LastLoginAt   time.Time `json:"last_login_at"`
}

type OAuthAccount struct {
    UserID     string `json:"user_id"`
    Provider   string `json:"provider"`
    ProviderID string `json:"provider_id"`
    Email      string `json:"email"`
}

type AuthSession struct {
    State     string    `json:"state"`
    Provider  string    `json:"provider"`
    ExpiresAt time.Time `json:"expires_at"`
}
```

### HTTP Routes Setup

```go
func SetupAuthRoutes(auth *AuthService) *http.ServeMux {
    mux := http.NewServeMux()
    
    // OAuth initiation routes
    mux.HandleFunc("/auth/google", func(w http.ResponseWriter, r *http.Request) {
        auth.StartOAuthFlow("google", w, r)
    })
    
    mux.HandleFunc("/auth/github", func(w http.ResponseWriter, r *http.Request) {
        auth.StartOAuthFlow("github", w, r)
    })
    
    mux.HandleFunc("/auth/apple", func(w http.ResponseWriter, r *http.Request) {
        auth.StartOAuthFlow("apple", w, r)
    })
    
    mux.HandleFunc("/auth/twitter", func(w http.ResponseWriter, r *http.Request) {
        auth.StartOAuthFlow("twitter", w, r)
    })
    
    // OAuth callback routes
    mux.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
        user, err := auth.HandleOAuthCallback("google", w, r)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Redirect to app dashboard
        http.Redirect(w, r, "/dashboard", http.StatusFound)
    })
    
    mux.HandleFunc("/auth/github/callback", func(w http.ResponseWriter, r *http.Request) {
        user, err := auth.HandleOAuthCallback("github", w, r)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        http.Redirect(w, r, "/dashboard", http.StatusFound)
    })
    
    // Protected routes
    mux.HandleFunc("/dashboard", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
        user := r.Context().Value("user").(*User)
        fmt.Fprintf(w, "Welcome %s!", user.Name)
    }))
    
    mux.HandleFunc("/api/user", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
        user := r.Context().Value("user").(*User)
        json.NewEncoder(w).Encode(user)
    }))
    
    return mux
}
```

### Database Schema

```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    picture VARCHAR(500),
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP DEFAULT NOW()
);

-- OAuth accounts table for linking multiple providers
CREATE TABLE oauth_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(provider, provider_id)
);

-- Sessions table (optional - for server-side sessions)
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_oauth_accounts_user_id ON oauth_accounts(user_id);
CREATE INDEX idx_oauth_accounts_provider ON oauth_accounts(provider, provider_id);
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

### Environment Configuration

```bash
# OAuth Providers
BEAVER_OAUTH_GOOGLE_CLIENT_ID=your_google_client_id
BEAVER_OAUTH_GOOGLE_CLIENT_SECRET=your_google_client_secret

BEAVER_OAUTH_GITHUB_CLIENT_ID=your_github_client_id  
BEAVER_OAUTH_GITHUB_CLIENT_SECRET=your_github_client_secret

BEAVER_OAUTH_APPLE_CLIENT_ID=com.yourapp.serviceid
BEAVER_OAUTH_APPLE_TEAM_ID=YOUR_TEAM_ID
BEAVER_OAUTH_APPLE_KEY_ID=YOUR_KEY_ID
BEAVER_OAUTH_APPLE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----"

BEAVER_OAUTH_TWITTER_CLIENT_ID=your_twitter_client_id
BEAVER_OAUTH_TWITTER_CLIENT_SECRET=your_twitter_client_secret

# Security
JWT_SECRET=your_jwt_secret_key
COOKIE_SECRET=your_cookie_secret_key

# Database
DATABASE_URL=postgres://user:pass@localhost/dbname
```

### Frontend Integration

```javascript
// Login buttons
<button onclick="loginWith('google')">Sign in with Google</button>
<button onclick="loginWith('github')">Sign in with GitHub</button>
<button onclick="loginWith('apple')">Sign in with Apple</button>
<button onclick="loginWith('twitter')">Sign in with Twitter</button>

<script>
function loginWith(provider) {
    // For PWA/SPA apps, you might handle PKCE on the client side
    window.location.href = `/auth/${provider}`;
}

// Handle post-login redirect
if (window.location.pathname === '/dashboard') {
    // User successfully authenticated
    loadUserDashboard();
}
</script>
```

### Key Security Considerations

1. **CSRF Protection** - Always verify state parameter
2. **PKCE for Public Clients** - Essential for PWA/mobile apps
3. **Secure Cookies** - HttpOnly, Secure, SameSite attributes
4. **Token Expiration** - Implement proper session management
5. **Email Verification** - Verify email ownership when needed
6. **Account Linking** - Handle users with multiple OAuth accounts
7. **Error Handling** - Secure error messages without information leakage

This complete auth implementation provides a production-ready authentication system using the OAuth package! 🔐

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