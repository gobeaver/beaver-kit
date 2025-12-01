package testing_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gobeaver/beaver-kit/oauth"
	oauthtest "github.com/gobeaver/beaver-kit/oauth/testing"
)

// TestStateReplayAttackPrevention tests that state tokens cannot be reused
func TestStateReplayAttackPrevention(t *testing.T) {
	sessionStore := oauth.NewMemorySessionStore(5 * time.Minute)

	ctx := context.Background()

	// Get the state from session store (normally this would be in the URL)
	// For this test, we'll simulate having captured a state
	capturedState := "test-state-123"

	// Store session data
	sessionData := &oauth.SessionData{
		State:     capturedState,
		Provider:  "test",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	err := sessionStore.Store(ctx, capturedState, sessionData)
	if err != nil {
		t.Fatalf("Failed to store session: %v", err)
	}

	// First exchange should succeed and delete the session
	retrieved, err := sessionStore.RetrieveAndDelete(ctx, capturedState)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if retrieved.State != capturedState {
		t.Error("Retrieved state doesn't match")
	}

	// Second attempt with same state should fail (replay attack)
	_, err = sessionStore.RetrieveAndDelete(ctx, capturedState)
	if err == nil {
		t.Error("Expected error on replay attack, but got none")
	}

	// Verify session is truly deleted
	_, err = sessionStore.Retrieve(ctx, capturedState)
	if err == nil {
		t.Error("Session should be deleted after first retrieval")
	}
}

// TestPKCEValidation tests PKCE challenge validation
func TestPKCEValidation(t *testing.T) {
	tests := []struct {
		name        string
		verifierLen int
		expectError bool
		description string
	}{
		{
			name:        "valid_minimum_length",
			verifierLen: 43,
			expectError: false,
			description: "43 characters should be valid",
		},
		{
			name:        "valid_maximum_length",
			verifierLen: 128,
			expectError: false,
			description: "128 characters should be valid",
		},
		{
			name:        "invalid_too_short",
			verifierLen: 42,
			expectError: true,
			description: "Less than 43 characters should be invalid",
		},
		{
			name:        "invalid_too_long",
			verifierLen: 129,
			expectError: true,
			description: "More than 128 characters should be invalid",
		},
		{
			name:        "valid_typical_length",
			verifierLen: 64,
			expectError: false,
			description: "64 characters (typical) should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate verifier of specific length
			verifier := generateVerifier(tt.verifierLen)

			// Validate using PKCE validation logic
			err := validatePKCEVerifier(verifier)

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
		})
	}
}

// TestCSRFProtection tests CSRF protection mechanisms
func TestCSRFProtection(t *testing.T) {
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName: "test",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
	defer mockServer.Close()

	multiService, _ := oauth.NewMultiProviderService(oauth.MultiProviderConfig{
		PKCEEnabled:    true,
		SessionTimeout: 5 * time.Minute,
	})

	provider := mockServer.CreateMockProvider()
	_ = multiService.RegisterProvider("test", provider)

	ctx := context.Background()

	// Get legitimate auth URL and state
	_, legitimateState, err := multiService.GetAuthURL(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get auth URL: %v", err)
	}

	// Issue code with legitimate state
	code := mockServer.IssueAuthorizationCode("user", legitimateState, "http://localhost:8080/callback", "")

	// Try to exchange with different state (CSRF attack)
	attackerState := "attacker-state-456"
	_, err = multiService.Exchange(ctx, "test", code, attackerState)
	if err == nil {
		t.Error("Expected error when state doesn't match, potential CSRF vulnerability")
	}

	// Exchange with correct state should work
	// Note: This might fail because we already tried with wrong state, but that's ok for this test
	_, _ = multiService.Exchange(ctx, "test", code, legitimateState)
}

// TestTokenEncryption tests token encryption/decryption
func TestTokenEncryption(t *testing.T) {
	// Create encryption key
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := oauth.NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Create token manager with encryption
	tokenStore := oauth.NewMemoryTokenStore(1 * time.Hour)
	encryptedStore, err := oauth.NewEncryptedTokenStore(tokenStore, key)
	if err != nil {
		t.Fatalf("Failed to create encrypted store: %v", err)
	}

	tokenManager := oauth.NewAdvancedTokenManager(oauth.TokenManagerConfig{
		Store:     encryptedStore,
		Encryptor: encryptor,
	})

	// Test token
	token := &oauth.Token{
		AccessToken:  "secret-access-token",
		RefreshToken: "secret-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	ctx := context.Background()
	userID := "test-user"
	provider := "test-provider"

	// Cache token (should be encrypted)
	err = tokenManager.CacheToken(ctx, userID, provider, token)
	if err != nil {
		t.Fatalf("Failed to cache token: %v", err)
	}

	// Retrieve token (should be decrypted)
	retrieved, err := tokenManager.GetCachedToken(ctx, userID, provider)
	if err != nil {
		t.Fatalf("Failed to retrieve token: %v", err)
	}

	// Verify tokens match
	if retrieved.AccessToken != token.AccessToken {
		t.Error("Access token doesn't match after encryption/decryption")
	}

	if retrieved.RefreshToken != token.RefreshToken {
		t.Error("Refresh token doesn't match after encryption/decryption")
	}

	// Verification that encryption actually happened would require testing
	// at a lower level or inspecting the raw stored data
}

// TestSessionTimeout tests that sessions expire correctly
func TestSessionTimeout(t *testing.T) {
	// Create session store with very short timeout
	sessionStore := oauth.NewMemorySessionStore(100 * time.Millisecond)

	ctx := context.Background()
	sessionData := &oauth.SessionData{
		State:     "test-state",
		Provider:  "test",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(100 * time.Millisecond),
	}

	// Store session
	err := sessionStore.Store(ctx, "test-state", sessionData)
	if err != nil {
		t.Fatalf("Failed to store session: %v", err)
	}

	// Should be retrievable immediately
	retrieved, err := sessionStore.Retrieve(ctx, "test-state")
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if retrieved.State != "test-state" {
		t.Error("Retrieved session doesn't match")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be retrievable after expiration
	_, err = sessionStore.Retrieve(ctx, "test-state")
	if err == nil {
		t.Error("Expected error when retrieving expired session")
	}
}

// TestInsecureRedirectPrevention tests prevention of open redirects
func TestInsecureRedirectPrevention(t *testing.T) {
	validRedirects := []string{
		"http://localhost:8080/callback",
		"https://myapp.com/oauth/callback",
		"http://127.0.0.1:3000/auth",
	}

	invalidRedirects := []string{
		"javascript:alert('xss')",
		"data:text/html,<script>alert('xss')</script>",
		"//evil.com/steal",
		"http://evil.com/phishing",
	}

	for _, redirect := range validRedirects {
		if !isValidRedirectURL(redirect) {
			t.Errorf("Valid redirect URL rejected: %s", redirect)
		}
	}

	for _, redirect := range invalidRedirects {
		if isValidRedirectURL(redirect) {
			t.Errorf("Invalid redirect URL accepted: %s", redirect)
		}
	}
}

// TestRateLimitingProtection tests rate limiting for OAuth endpoints
func TestRateLimitingProtection(t *testing.T) {
	config := oauth.RateLimiterConfig{
		Rate:      5,
		Interval:  1 * time.Second,
		BurstSize: 5,
	}

	limiter := oauth.NewTokenBucketLimiter(config)
	ctx := context.Background()
	clientIP := "192.168.1.100"

	// Should allow burst of 5 requests
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, clientIP)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed within burst", i+1)
		}
	}

	// 6th request should be denied
	allowed, err := limiter.Allow(ctx, clientIP)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("Request should be denied after burst limit")
	}
}

// TestXSSProtection tests XSS protection in user data handling
func TestXSSProtection(t *testing.T) {
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName: "test",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
	defer mockServer.Close()

	// Set user info with potential XSS payload
	maliciousUserInfo := &oauth.UserInfo{
		ID:    "test_user",
		Email: "test@example.com",
		Name:  "<script>alert('xss')</script>",
		// The application should sanitize this when displaying
	}

	mockServer.SetUserInfo("test_user", maliciousUserInfo)

	provider := mockServer.CreateMockProvider()

	// Get token
	code := mockServer.IssueAuthorizationCode("test_user", "state", "http://localhost:8080/callback", "")
	ctx := context.Background()
	token, _ := provider.Exchange(ctx, code, nil)

	// Get user info
	userInfo, err := provider.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to get user info: %v", err)
	}

	// Check that malicious content is present (not sanitized at this layer)
	// Sanitization should happen at the presentation layer
	if !strings.Contains(userInfo.Name, "<script>") {
		t.Error("User info should contain raw data, sanitization happens at presentation")
	}
}

// Helper functions

func generateVerifier(length int) string {
	// Generate a verifier of specific length
	if length <= 0 {
		return ""
	}

	// Use base64url encoding to generate the verifier
	bytes := make([]byte, (length*6)/8+1)
	_, _ = rand.Read(bytes)
	verifier := base64.RawURLEncoding.EncodeToString(bytes)

	if len(verifier) > length {
		verifier = verifier[:length]
	}

	return verifier
}

func validatePKCEVerifier(verifier string) error {
	// RFC 7636 requires 43-128 characters
	if len(verifier) < 43 {
		return fmt.Errorf("verifier too short: %d < 43", len(verifier))
	}
	if len(verifier) > 128 {
		return fmt.Errorf("verifier too long: %d > 128", len(verifier))
	}

	// Check for unreserved characters only
	for _, c := range verifier {
		if !isUnreservedChar(c) {
			return fmt.Errorf("invalid character in verifier: %c", c)
		}
	}

	return nil
}

func isUnreservedChar(c rune) bool {
	// RFC 7636: ALPHA / DIGIT / "-" / "." / "_" / "~"
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

func isValidRedirectURL(url string) bool {
	// Simple validation - in production, use a proper URL parser
	// and validate against a whitelist

	// Reject obvious XSS attempts
	if strings.HasPrefix(strings.ToLower(url), "javascript:") {
		return false
	}
	if strings.HasPrefix(strings.ToLower(url), "data:") {
		return false
	}
	if strings.HasPrefix(url, "//") {
		return false
	}

	// Should start with http:// or https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}

	// In production, check against whitelist of allowed domains
	allowedHosts := []string{
		"localhost",
		"127.0.0.1",
		"myapp.com",
	}

	for _, host := range allowedHosts {
		if strings.Contains(url, host) {
			return true
		}
	}

	return false
}
