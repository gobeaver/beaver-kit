// Package oauth provides a comprehensive OAuth 2.0 client implementation with support for multiple providers.
//
// This package offers production-ready OAuth 2.0 authentication with security hardening features including:
//   - PKCE (Proof Key for Code Exchange) support per RFC 7636
//   - JWT signature validation for providers that support it
//   - Multi-provider architecture with dynamic registration
//   - Advanced token management with encryption and auto-refresh
//   - Rate limiting, circuit breakers, and monitoring
//   - Comprehensive security measures against replay attacks
//
// # Quick Start
//
// Initialize the OAuth service using environment variables:
//
//	import "github.com/gobeaver/beaver-kit/oauth"
//
//	// Initialize with default configuration (uses BEAVER_OAUTH_* env vars)
//	err := oauth.Init()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get authorization URL
//	authURL, state := oauth.OAuth().GetAuthURL(context.Background())
//	fmt.Printf("Visit: %s\n", authURL)
//
//	// Exchange authorization code for token
//	token, err := oauth.OAuth().Exchange(context.Background(), code, state)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Multi-Provider Usage
//
// For applications requiring multiple OAuth providers:
//
//	// Create multi-provider service
//	config := oauth.MultiProviderConfig{
//	    Providers: map[string]oauth.ProviderConfig{
//	        "google": {
//	            ClientID:     "your-google-client-id",
//	            ClientSecret: "your-google-client-secret",
//	            RedirectURL:  "https://yourapp.com/callback",
//	        },
//	        "github": {
//	            ClientID:     "your-github-client-id",
//	            ClientSecret: "your-github-client-secret",
//	            RedirectURL:  "https://yourapp.com/callback",
//	        },
//	    },
//	}
//
//	service, err := oauth.NewMultiProviderService(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use specific provider
//	authURL, state, err := service.GetAuthURL(ctx, "google")
//	token, err := service.Exchange(ctx, "google", code, state)
//
// # Environment Variables
//
// The package supports configuration via environment variables with the BEAVER_OAUTH_ prefix:
//   - BEAVER_OAUTH_PROVIDER: OAuth provider (google, github, apple, twitter, custom)
//   - BEAVER_OAUTH_CLIENT_ID: OAuth client ID
//   - BEAVER_OAUTH_CLIENT_SECRET: OAuth client secret
//   - BEAVER_OAUTH_REDIRECT_URL: OAuth redirect URL
//   - BEAVER_OAUTH_SCOPES: Comma-separated OAuth scopes
//   - BEAVER_OAUTH_USE_PKCE: Enable PKCE (default: true for public clients)
//   - BEAVER_OAUTH_RATE_LIMIT: Requests per second (default: 10)
//   - BEAVER_OAUTH_CIRCUIT_THRESHOLD: Circuit breaker threshold (default: 5)
//   - BEAVER_OAUTH_ENABLE_METRICS: Enable metrics collection (default: false)
//
// Custom prefixes can be used with the config.LoadOptions{Prefix: "CUSTOM_"} pattern.
//
// # Supported Providers
//
// Built-in support for popular OAuth providers:
//   - Google OAuth 2.0
//   - GitHub OAuth 2.0
//   - Apple Sign In (with JWT validation)
//   - Twitter OAuth 2.0
//   - Generic OAuth 2.0 (CustomProvider)
//
// # Security Features
//
//   - PKCE implementation prevents authorization code interception
//   - JWT signature validation for Apple and other providers
//   - State parameter validation prevents CSRF attacks
//   - Session replay protection with immediate session deletion
//   - Token encryption for storage (AES-GCM)
//   - Rate limiting to prevent API abuse
//   - Circuit breakers for resilient external API calls
//
// # Advanced Features
//
//   - Automatic token refresh before expiration
//   - Token caching with configurable TTL
//   - Bulk token operations for cleanup
//   - Comprehensive metrics and monitoring
//   - Health checks for service validation
//   - Request/response logging with sensitive data redaction
//
// # Error Handling
//
// The package defines specific error types for different scenarios:
//   - ErrInvalidConfig: Configuration validation errors
//   - ErrNotInitialized: Service not properly initialized
//   - ErrInvalidState: State parameter validation failures
//   - ErrTokenExpired: Token expiration errors
//   - ErrProviderNotFound: Unknown provider errors
//
// # Production Considerations
//
// For production deployments, consider:
//   - Enabling rate limiting to prevent API quota exhaustion
//   - Using circuit breakers for resilient external API calls
//   - Enabling metrics collection for monitoring
//   - Implementing proper token storage with encryption
//   - Setting appropriate token refresh thresholds
//   - Configuring logging levels appropriately
//
// For complete examples and advanced usage patterns, see the project documentation
// and example applications in the examples directory.
package oauth
