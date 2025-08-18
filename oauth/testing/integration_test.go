package testing_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gobeaver/beaver-kit/oauth"
	oauthtest "github.com/gobeaver/beaver-kit/oauth/testing"
)

func TestCompleteOAuthFlow(t *testing.T) {
	// Setup mock OAuth server
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName:    "test",
		ClientID:        "test-client",
		ClientSecret:    "test-secret",
		SupportsPKCE:    true,
		SupportsRefresh: true,
		TokenExpiry:     1 * time.Hour,
	})
	defer mockServer.Close()
	
	// Set user info
	mockServer.SetUserInfo("test_user", &oauth.UserInfo{
		ID:            "test_user",
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          "Test User",
		Provider:      "test",
	})
	
	// Create OAuth service with mock provider
	provider := mockServer.CreateMockProvider()
	
	// Override provider in service (would normally use multi-provider service)
	multiService, _ := oauth.NewMultiProviderService(oauth.MultiProviderConfig{
		PKCEEnabled:    true,
		SessionTimeout: 5 * time.Minute,
	})
	multiService.RegisterProvider("test", provider)
	
	ctx := context.Background()
	
	// Step 1: Get authorization URL
	authURL, state, err := multiService.GetAuthURL(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get auth URL: %v", err)
	}
	
	if authURL == "" {
		t.Error("Auth URL should not be empty")
	}
	
	if state == "" {
		t.Error("State should not be empty")
	}
	
	// Step 2: Simulate user authorization
	code := mockServer.IssueAuthorizationCode("test_user", state, "http://localhost:8080/callback", "")
	
	// Step 3: Exchange code for token
	token, err := multiService.Exchange(ctx, "test", code, state)
	if err != nil {
		t.Fatalf("Failed to exchange code: %v", err)
	}
	
	if token.AccessToken == "" {
		t.Error("Access token should not be empty")
	}
	
	if token.RefreshToken == "" {
		t.Error("Refresh token should not be empty when refresh is supported")
	}
	
	// Step 4: Get user info
	userInfo, err := provider.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to get user info: %v", err)
	}
	
	if userInfo.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", userInfo.Email)
	}
	
	// Step 5: Refresh token (if supported)
	if token.RefreshToken != "" {
		newToken, err := provider.RefreshToken(ctx, token.RefreshToken)
		if err != nil {
			t.Fatalf("Failed to refresh token: %v", err)
		}
		
		if newToken.AccessToken == "" {
			t.Error("New access token should not be empty")
		}
		
		if newToken.AccessToken == token.AccessToken {
			t.Error("New access token should be different from old token")
		}
	}
	
	// Step 6: Revoke token
	err = provider.RevokeToken(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}
}

func TestOAuthFlowWithPKCE(t *testing.T) {
	// Setup mock OAuth server with PKCE
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName: "test",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		SupportsPKCE: true,
	})
	defer mockServer.Close()
	
	provider := mockServer.CreateMockProvider()
	
	// Create PKCE challenge
	pkce, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("Failed to generate PKCE: %v", err)
	}
	
	// Get auth URL with PKCE
	state := "test-state"
	authURL := provider.GetAuthURL(state, pkce)
	
	if authURL == "" {
		t.Error("Auth URL should not be empty")
	}
	
	// Issue authorization code with PKCE verifier
	code := mockServer.IssueAuthorizationCode("test_user", state, "http://localhost:8080/callback", pkce.Verifier)
	
	// Exchange code with PKCE
	ctx := context.Background()
	token, err := provider.Exchange(ctx, code, pkce)
	if err != nil {
		t.Fatalf("Failed to exchange code with PKCE: %v", err)
	}
	
	if token.AccessToken == "" {
		t.Error("Access token should not be empty")
	}
}

func TestOAuthErrorHandling(t *testing.T) {
	// Setup mock OAuth server
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName: "test",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
	defer mockServer.Close()
	
	provider := mockServer.CreateMockProvider()
	ctx := context.Background()
	
	// Test invalid code
	_, err := provider.Exchange(ctx, "invalid_code", nil)
	if err == nil {
		t.Error("Expected error for invalid code")
	}
	
	// Test server error
	mockServer.SetFailureScenario("token", true)
	_, err = provider.Exchange(ctx, "some_code", nil)
	if err == nil {
		t.Error("Expected error when server fails")
	}
	mockServer.SetFailureScenario("token", false)
	
	// Test invalid access token for user info
	_, err = provider.GetUserInfo(ctx, "invalid_token")
	if err == nil {
		t.Error("Expected error for invalid access token")
	}
}

func TestOAuthWithLatency(t *testing.T) {
	// Setup mock OAuth server with latency
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName: "test",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
	defer mockServer.Close()
	
	// Set 100ms latency for token endpoint
	mockServer.SetLatency("token", 100*time.Millisecond)
	
	provider := mockServer.CreateMockProvider()
	
	// Issue a code
	code := mockServer.IssueAuthorizationCode("test_user", "state", "http://localhost:8080/callback", "")
	
	// Measure exchange time
	start := time.Now()
	ctx := context.Background()
	token, err := provider.Exchange(ctx, code, nil)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Failed to exchange code: %v", err)
	}
	
	if token.AccessToken == "" {
		t.Error("Access token should not be empty")
	}
	
	// Check that latency was applied
	if duration < 100*time.Millisecond {
		t.Errorf("Expected latency of at least 100ms, got %v", duration)
	}
}

func TestConcurrentOAuthFlows(t *testing.T) {
	// Setup mock OAuth server
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName:    "test",
		ClientID:        "test-client",
		ClientSecret:    "test-secret",
		SupportsRefresh: true,
	})
	defer mockServer.Close()
	
	provider := mockServer.CreateMockProvider()
	ctx := context.Background()
	
	// Run multiple OAuth flows concurrently
	numFlows := 10
	errors := make(chan error, numFlows)
	
	for i := 0; i < numFlows; i++ {
		go func(id int) {
			// Issue unique code for this flow
			state := fmt.Sprintf("state_%d", id)
			code := mockServer.IssueAuthorizationCode(
				fmt.Sprintf("user_%d", id),
				state,
				"http://localhost:8080/callback",
				"",
			)
			
			// Exchange code
			token, err := provider.Exchange(ctx, code, nil)
			if err != nil {
				errors <- fmt.Errorf("flow %d: exchange failed: %w", id, err)
				return
			}
			
			// Get user info
			_, err = provider.GetUserInfo(ctx, token.AccessToken)
			if err != nil {
				errors <- fmt.Errorf("flow %d: user info failed: %w", id, err)
				return
			}
			
			// Refresh token
			if token.RefreshToken != "" {
				_, err = provider.RefreshToken(ctx, token.RefreshToken)
				if err != nil {
					errors <- fmt.Errorf("flow %d: refresh failed: %w", id, err)
					return
				}
			}
			
			errors <- nil
		}(i)
	}
	
	// Wait for all flows to complete
	for i := 0; i < numFlows; i++ {
		if err := <-errors; err != nil {
			t.Error(err)
		}
	}
}

func TestTokenExpiration(t *testing.T) {
	// Setup mock OAuth server with short token expiry
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName: "test",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenExpiry:  100 * time.Millisecond, // Very short expiry for testing
	})
	defer mockServer.Close()
	
	provider := mockServer.CreateMockProvider()
	ctx := context.Background()
	
	// Issue and exchange code
	code := mockServer.IssueAuthorizationCode("test_user", "state", "http://localhost:8080/callback", "")
	token, err := provider.Exchange(ctx, code, nil)
	if err != nil {
		t.Fatalf("Failed to exchange code: %v", err)
	}
	
	// Token should be valid immediately
	if token.IsExpired() {
		t.Error("Token should not be expired immediately after issuance")
	}
	
	// Wait for token to expire
	time.Sleep(150 * time.Millisecond)
	
	// Token should be expired now
	if !token.IsExpired() {
		t.Error("Token should be expired after waiting")
	}
	
	// Attempting to use expired token should fail
	_, err = provider.GetUserInfo(ctx, token.AccessToken)
	if err == nil {
		t.Error("Expected error when using expired token")
	}
}