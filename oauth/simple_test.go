package oauth_test

import (
	"context"
	"testing"

	"github.com/gobeaver/beaver-kit/oauth"
)

func TestBasicOAuth(t *testing.T) {
	// Test basic OAuth service creation
	config := oauth.Config{
		Provider:     "github",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
	}
	service, err := oauth.New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if service == nil {
		t.Error("New() should return a non-nil service")
	}

	// Test GitHub provider creation
	provider := oauth.NewGitHub(oauth.ProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
	})

	if provider == nil {
		t.Error("NewGitHub() should return a non-nil provider")
	}

	if provider.Name() != "github" {
		t.Errorf("Name() = %v, want github", provider.Name())
	}

	if provider.SupportsRefresh() {
		t.Error("GitHub should not support refresh tokens")
	}

	if !provider.SupportsPKCE() {
		t.Error("GitHub should support PKCE")
	}

	// Test auth URL generation
	authURL := provider.GetAuthURL("test_state", nil)
	if authURL == "" {
		t.Error("GetAuthURL() should return a non-empty URL")
	}

	// Test validation
	err = provider.ValidateConfig()
	if err != nil {
		t.Errorf("ValidateConfig() error = %v", err)
	}
}

func TestGitHubProviderWithPKCE(t *testing.T) {
	provider := oauth.NewGitHub(oauth.ProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
	})

	// Generate PKCE challenge
	pkce, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("GeneratePKCE() error = %v", err)
	}

	// Test auth URL with PKCE
	authURL := provider.GetAuthURL("test_state", pkce)
	if authURL == "" {
		t.Error("GetAuthURL() with PKCE should return a non-empty URL")
	}

	// Verify PKCE parameters are included
	if !containsString(authURL, "code_challenge") {
		t.Error("Auth URL should contain code_challenge parameter")
	}
	if !containsString(authURL, "code_challenge_method") {
		t.Error("Auth URL should contain code_challenge_method parameter")
	}
}

func TestPKCEGeneration(t *testing.T) {
	// Test S256 method
	pkce, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Errorf("GeneratePKCEChallenge(S256) error = %v", err)
	}
	if pkce.ChallengeMethod != "S256" {
		t.Errorf("GeneratePKCEChallenge(S256) method = %v, want S256", pkce.ChallengeMethod)
	}
	if len(pkce.Verifier) == 0 {
		t.Error("GeneratePKCEChallenge(S256) should generate non-empty verifier")
	}
	if len(pkce.Challenge) == 0 {
		t.Error("GeneratePKCEChallenge(S256) should generate non-empty challenge")
	}

	// Test plain method
	pkce, err = oauth.GeneratePKCEChallenge("plain")
	if err != nil {
		t.Errorf("GeneratePKCEChallenge(plain) error = %v", err)
	}
	if pkce.ChallengeMethod != "plain" {
		t.Errorf("GeneratePKCEChallenge(plain) method = %v, want plain", pkce.ChallengeMethod)
	}
	if pkce.Challenge != pkce.Verifier {
		t.Error("GeneratePKCEChallenge(plain) challenge should equal verifier")
	}

	// Test invalid method
	_, err = oauth.GeneratePKCEChallenge("invalid")
	if err == nil {
		t.Error("GeneratePKCEChallenge(invalid) should return an error")
	}
}

func TestProviderValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  oauth.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: oauth.ProviderConfig{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				RedirectURL:  "http://localhost:8080/callback",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: oauth.ProviderConfig{
				ClientSecret: "test_client_secret",
				RedirectURL:  "http://localhost:8080/callback",
			},
			wantErr: true,
		},
		{
			name: "missing client secret",
			config: oauth.ProviderConfig{
				ClientID:    "test_client_id",
				RedirectURL: "http://localhost:8080/callback",
			},
			wantErr: true,
		},
		{
			name: "missing redirect URL",
			config: oauth.ProviderConfig{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := oauth.NewGitHub(tt.config)
			err := provider.ValidateConfig()

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitHubProviderRefreshToken(t *testing.T) {
	provider := oauth.NewGitHub(oauth.ProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
	})

	// GitHub doesn't support refresh tokens
	_, err := provider.RefreshToken(context.Background(), "any_token")
	if err == nil {
		t.Error("RefreshToken() should return an error for GitHub")
	}

	if err.Error() != oauth.ErrNoRefreshToken.Error() {
		t.Errorf("RefreshToken() error = %v, want %v", err, oauth.ErrNoRefreshToken)
	}
}

func TestGoogleProvider(t *testing.T) {
	// Test Google provider creation
	provider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "google_client_id",
		ClientSecret: "google_client_secret",
		RedirectURL:  "http://localhost:8080/callback/google",
	})

	if provider == nil {
		t.Error("NewGoogle() should return a non-nil provider")
	}

	if provider.Name() != "google" {
		t.Errorf("Name() = %v, want google", provider.Name())
	}

	if !provider.SupportsRefresh() {
		t.Error("Google should support refresh tokens")
	}

	if !provider.SupportsPKCE() {
		t.Error("Google should support PKCE")
	}

	// Test auth URL generation with PKCE
	pkce, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("GeneratePKCEChallenge() error = %v", err)
	}

	authURL := provider.GetAuthURL("test_state", pkce)
	if authURL == "" {
		t.Error("GetAuthURL() should return a non-empty URL")
	}

	// Should contain Google-specific parameters
	if !containsString(authURL, "accounts.google.com") {
		t.Error("Auth URL should use Google's authorization endpoint")
	}
	if !containsString(authURL, "access_type=offline") {
		t.Error("Auth URL should request offline access for refresh tokens")
	}
	if !containsString(authURL, "prompt=consent") {
		t.Error("Auth URL should force consent for refresh tokens")
	}

	// Test validation
	err = provider.ValidateConfig()
	if err != nil {
		t.Errorf("ValidateConfig() error = %v", err)
	}
}

func TestAppleProvider(t *testing.T) {
	// Test Apple provider creation
	provider, err := oauth.NewApple(oauth.ProviderConfig{
		ClientID:    "com.example.app",
		RedirectURL: "https://example.com/callback",
		TeamID:      "TEAM123",
		KeyID:       "KEY123",
		PrivateKey:  "", // No private key for basic test
	})

	if err != nil {
		t.Fatalf("NewApple() error = %v", err)
	}

	if provider == nil {
		t.Error("NewApple() should return a non-nil provider")
	}

	if provider.Name() != "apple" {
		t.Errorf("Name() = %v, want apple", provider.Name())
	}

	if !provider.SupportsRefresh() {
		t.Error("Apple should support refresh tokens")
	}

	if !provider.SupportsPKCE() {
		t.Error("Apple should support PKCE")
	}

	// Test auth URL generation with PKCE
	pkce, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("GeneratePKCEChallenge() error = %v", err)
	}

	authURL := provider.GetAuthURL("test_state", pkce)
	if authURL == "" {
		t.Error("GetAuthURL() should return a non-empty URL")
	}

	// Should contain Apple-specific parameters
	if !containsString(authURL, "appleid.apple.com") {
		t.Error("Auth URL should use Apple's authorization endpoint")
	}
	if !containsString(authURL, "response_mode=form_post") {
		t.Error("Auth URL should use form_post response mode")
	}
}

func TestTwitterProvider(t *testing.T) {
	// Test Twitter provider creation
	provider := oauth.NewTwitter(oauth.ProviderConfig{
		ClientID:    "twitter_client_id",
		RedirectURL: "https://example.com/callback",
	})

	if provider == nil {
		t.Error("NewTwitter() should return a non-nil provider")
	}

	if provider.Name() != "twitter" {
		t.Errorf("Name() = %v, want twitter", provider.Name())
	}

	if !provider.SupportsRefresh() {
		t.Error("Twitter should support refresh tokens")
	}

	if !provider.SupportsPKCE() {
		t.Error("Twitter should support PKCE")
	}

	// Test auth URL generation with PKCE
	pkce, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("GeneratePKCEChallenge() error = %v", err)
	}

	authURL := provider.GetAuthURL("test_state", pkce)
	if authURL == "" {
		t.Error("GetAuthURL() should return a non-empty URL")
	}

	// Should contain Twitter-specific parameters
	if !containsString(authURL, "twitter.com/i/oauth2/authorize") {
		t.Error("Auth URL should use Twitter's OAuth 2.0 authorization endpoint")
	}
	if !containsString(authURL, "code_challenge_method=S256") {
		t.Error("Auth URL should include PKCE challenge method")
	}

	// Test validation
	err = provider.ValidateConfig()
	if err != nil {
		t.Errorf("ValidateConfig() error = %v", err)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}