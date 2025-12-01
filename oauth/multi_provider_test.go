package oauth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gobeaver/beaver-kit/oauth"
)

func TestMultiProviderService_RegisterProvider(t *testing.T) {
	config := oauth.MultiProviderConfig{
		PKCEEnabled:        true,
		SessionTimeout:     5 * time.Minute,
		TokenCacheDuration: 1 * time.Hour,
	}

	service, err := oauth.NewMultiProviderService(config)
	if err != nil {
		t.Fatalf("Failed to create multi-provider service: %v", err)
	}

	// Create test providers
	googleProvider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test-google-id",
		ClientSecret: "test-google-secret",
		RedirectURL:  "http://localhost/callback",
	})

	githubProvider := oauth.NewGitHub(oauth.ProviderConfig{
		ClientID:     "test-github-id",
		ClientSecret: "test-github-secret",
		RedirectURL:  "http://localhost/callback",
	})

	// Test registering providers
	if err := service.RegisterProvider("google", googleProvider); err != nil {
		t.Errorf("Failed to register Google provider: %v", err)
	}

	if err := service.RegisterProvider("github", githubProvider); err != nil {
		t.Errorf("Failed to register GitHub provider: %v", err)
	}

	// Test duplicate registration
	if err := service.RegisterProvider("google", googleProvider); err == nil {
		t.Error("Expected error when registering duplicate provider")
	}

	// Test empty name
	if err := service.RegisterProvider("", googleProvider); err == nil {
		t.Error("Expected error when registering provider with empty name")
	}

	// Test nil provider
	if err := service.RegisterProvider("nil", nil); err == nil {
		t.Error("Expected error when registering nil provider")
	}
}

func TestMultiProviderService_GetProvider(t *testing.T) {
	config := oauth.MultiProviderConfig{
		PKCEEnabled:        true,
		SessionTimeout:     5 * time.Minute,
		TokenCacheDuration: 1 * time.Hour,
	}

	service, err := oauth.NewMultiProviderService(config)
	if err != nil {
		t.Fatalf("Failed to create multi-provider service: %v", err)
	}

	// Register a provider
	googleProvider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test-google-id",
		ClientSecret: "test-google-secret",
		RedirectURL:  "http://localhost/callback",
	})

	if err := service.RegisterProvider("google", googleProvider); err != nil {
		t.Fatalf("Failed to register Google provider: %v", err)
	}

	// Test getting existing provider
	provider, err := service.GetProvider("google")
	if err != nil {
		t.Errorf("Failed to get Google provider: %v", err)
	}
	if provider == nil {
		t.Error("Expected non-nil provider")
	}
	if provider.Name() != "google" {
		t.Errorf("Expected provider name 'google', got '%s'", provider.Name())
	}

	// Test getting non-existent provider
	_, err = service.GetProvider("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent provider")
	}
}

func TestMultiProviderService_ListProviders(t *testing.T) {
	config := oauth.MultiProviderConfig{
		PKCEEnabled:        true,
		SessionTimeout:     5 * time.Minute,
		TokenCacheDuration: 1 * time.Hour,
	}

	service, err := oauth.NewMultiProviderService(config)
	if err != nil {
		t.Fatalf("Failed to create multi-provider service: %v", err)
	}

	// Initially should be empty
	providers := service.ListProviders()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}

	// Register providers
	googleProvider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test-google-id",
		ClientSecret: "test-google-secret",
		RedirectURL:  "http://localhost/callback",
	})
	githubProvider := oauth.NewGitHub(oauth.ProviderConfig{
		ClientID:     "test-github-id",
		ClientSecret: "test-github-secret",
		RedirectURL:  "http://localhost/callback",
	})

	_ = service.RegisterProvider("google", googleProvider)
	_ = service.RegisterProvider("github", githubProvider)

	// Should list both providers
	providers = service.ListProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Check that both providers are in the list
	hasGoogle := false
	hasGitHub := false
	for _, name := range providers {
		if name == "google" {
			hasGoogle = true
		}
		if name == "github" {
			hasGitHub = true
		}
	}

	if !hasGoogle {
		t.Error("Google provider not found in list")
	}
	if !hasGitHub {
		t.Error("GitHub provider not found in list")
	}
}

func TestMultiProviderService_GetAuthURL(t *testing.T) {
	config := oauth.MultiProviderConfig{
		PKCEEnabled:        true,
		PKCEMethod:         "S256",
		SessionTimeout:     5 * time.Minute,
		TokenCacheDuration: 1 * time.Hour,
	}

	service, err := oauth.NewMultiProviderService(config)
	if err != nil {
		t.Fatalf("Failed to create multi-provider service: %v", err)
	}

	// Register a provider
	googleProvider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test-google-id",
		ClientSecret: "test-google-secret",
		RedirectURL:  "http://localhost/callback",
	})
	_ = service.RegisterProvider("google", googleProvider)

	ctx := context.Background()

	// Test getting auth URL
	authURL, state, err := service.GetAuthURL(ctx, "google")
	if err != nil {
		t.Errorf("Failed to get auth URL: %v", err)
	}
	if authURL == "" {
		t.Error("Expected non-empty auth URL")
	}
	if state == "" {
		t.Error("Expected non-empty state")
	}

	// Test with PKCE disabled
	authURL2, state2, err := service.GetAuthURL(ctx, "google", oauth.WithPKCE(false))
	if err != nil {
		t.Errorf("Failed to get auth URL with PKCE disabled: %v", err)
	}
	if authURL2 == "" {
		t.Error("Expected non-empty auth URL")
	}
	if state2 == "" {
		t.Error("Expected non-empty state")
	}

	// Test with metadata
	metadata := map[string]interface{}{
		"user_id": "123",
		"origin":  "mobile",
	}
	authURL3, state3, err := service.GetAuthURL(ctx, "google", oauth.WithMetadata(metadata))
	if err != nil {
		t.Errorf("Failed to get auth URL with metadata: %v", err)
	}
	if authURL3 == "" {
		t.Error("Expected non-empty auth URL")
	}
	if state3 == "" {
		t.Error("Expected non-empty state")
	}

	// Validate the state
	sessionData, err := service.ValidateState(ctx, state3)
	if err != nil {
		t.Errorf("Failed to validate state: %v", err)
	}
	if sessionData.Metadata == nil {
		t.Error("Expected metadata in session data")
	}
	if sessionData.Metadata["user_id"] != "123" {
		t.Errorf("Expected user_id '123', got %v", sessionData.Metadata["user_id"])
	}

	// Test with non-existent provider
	_, _, err = service.GetAuthURL(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error when getting auth URL for non-existent provider")
	}
}

func TestMultiProviderService_UnregisterProvider(t *testing.T) {
	config := oauth.MultiProviderConfig{
		PKCEEnabled:        true,
		SessionTimeout:     5 * time.Minute,
		TokenCacheDuration: 1 * time.Hour,
	}

	service, err := oauth.NewMultiProviderService(config)
	if err != nil {
		t.Fatalf("Failed to create multi-provider service: %v", err)
	}

	// Register a provider
	googleProvider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test-google-id",
		ClientSecret: "test-google-secret",
		RedirectURL:  "http://localhost/callback",
	})
	_ = service.RegisterProvider("google", googleProvider)

	// Verify it exists
	_, err = service.GetProvider("google")
	if err != nil {
		t.Error("Provider should exist before unregistering")
	}

	// Unregister the provider
	err = service.UnregisterProvider("google")
	if err != nil {
		t.Errorf("Failed to unregister provider: %v", err)
	}

	// Verify it's gone
	_, err = service.GetProvider("google")
	if err == nil {
		t.Error("Provider should not exist after unregistering")
	}

	// Test unregistering non-existent provider
	err = service.UnregisterProvider("nonexistent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent provider")
	}
}

func TestCustomProvider(t *testing.T) {
	config := oauth.ProviderConfig{
		Type:         "custom_oauth",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		AuthURL:      "https://auth.example.com/authorize",
		TokenURL:     "https://auth.example.com/token",
		UserInfoURL:  "https://auth.example.com/userinfo",
		RevokeURL:    "https://auth.example.com/revoke",
		Scopes:       []string{"read", "write"},
	}

	provider, err := oauth.NewCustom(config)
	if err != nil {
		t.Fatalf("Failed to create custom provider: %v", err)
	}

	// Test provider name
	if provider.Name() != "custom_oauth" {
		t.Errorf("Expected provider name 'custom_oauth', got '%s'", provider.Name())
	}

	// Test GetAuthURL
	authURL := provider.GetAuthURL("test-state", nil)
	if authURL == "" {
		t.Error("Expected non-empty auth URL")
	}
	if !strings.Contains(authURL, "https://auth.example.com/authorize") {
		t.Error("Auth URL should contain the configured auth endpoint")
	}
	if !strings.Contains(authURL, "client_id=test-client-id") {
		t.Error("Auth URL should contain client ID")
	}
	if !strings.Contains(authURL, "state=test-state") {
		t.Error("Auth URL should contain state")
	}

	// Test with PKCE
	pkce, _ := oauth.GeneratePKCEChallenge("S256")
	authURLWithPKCE := provider.GetAuthURL("test-state", pkce)
	if !strings.Contains(authURLWithPKCE, "code_challenge=") {
		t.Error("Auth URL should contain PKCE challenge")
	}
	if !strings.Contains(authURLWithPKCE, "code_challenge_method=S256") {
		t.Error("Auth URL should contain PKCE method")
	}

	// Test capabilities
	if !provider.SupportsRefresh() {
		t.Error("Custom provider should support refresh")
	}
	if !provider.SupportsPKCE() {
		t.Error("Custom provider should support PKCE")
	}

	// Test validation
	if err := provider.ValidateConfig(); err != nil {
		t.Errorf("Valid config should pass validation: %v", err)
	}
}

func TestCustomProvider_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    oauth.ProviderConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: oauth.ProviderConfig{
				ClientID:    "test-id",
				RedirectURL: "http://localhost/callback",
				AuthURL:     "https://auth.example.com/authorize",
				TokenURL:    "https://auth.example.com/token",
			},
			wantError: false,
		},
		{
			name: "missing client ID",
			config: oauth.ProviderConfig{
				RedirectURL: "http://localhost/callback",
				AuthURL:     "https://auth.example.com/authorize",
				TokenURL:    "https://auth.example.com/token",
			},
			wantError: true,
		},
		{
			name: "missing redirect URL",
			config: oauth.ProviderConfig{
				ClientID: "test-id",
				AuthURL:  "https://auth.example.com/authorize",
				TokenURL: "https://auth.example.com/token",
			},
			wantError: true,
		},
		{
			name: "missing auth URL",
			config: oauth.ProviderConfig{
				ClientID:    "test-id",
				RedirectURL: "http://localhost/callback",
				TokenURL:    "https://auth.example.com/token",
			},
			wantError: true,
		},
		{
			name: "missing token URL",
			config: oauth.ProviderConfig{
				ClientID:    "test-id",
				RedirectURL: "http://localhost/callback",
				AuthURL:     "https://auth.example.com/authorize",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := oauth.NewCustom(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("NewCustom() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
