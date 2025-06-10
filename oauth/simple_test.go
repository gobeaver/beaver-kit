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