package oauth_test

import (
	"context"
	"testing"

	"github.com/gobeaver/beaver-kit/oauth"
)

func TestAppleProvider_Basic(t *testing.T) {
	// Test Apple provider creation with minimal config (without private key for basic test)
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
}

func TestAppleProvider_GetAuthURL(t *testing.T) {
	provider, err := oauth.NewApple(oauth.ProviderConfig{
		ClientID:    "com.example.app",
		RedirectURL: "https://example.com/callback",
		TeamID:      "TEAM123",
		KeyID:       "KEY123",
		PrivateKey:  "", // No private key needed for auth URL test
	})

	if err != nil {
		t.Fatalf("NewApple() error = %v", err)
	}

	tests := []struct {
		name         string
		state        string
		pkce         *oauth.PKCEChallenge
		wantContains []string
	}{
		{
			name:  "basic auth URL",
			state: "test_state",
			pkce:  nil,
			wantContains: []string{
				"https://appleid.apple.com/auth/authorize",
				"client_id=com.example.app",
				"redirect_uri=https%3A%2F%2Fexample.com%2Fcallback",
				"scope=name+email",
				"state=test_state",
				"response_type=code",
				"response_mode=form_post",
			},
		},
		{
			name:  "auth URL with PKCE",
			state: "test_state",
			pkce: &oauth.PKCEChallenge{
				Challenge:       "test_challenge",
				ChallengeMethod: "S256",
				Verifier:        "test_verifier",
			},
			wantContains: []string{
				"https://appleid.apple.com/auth/authorize",
				"code_challenge=test_challenge",
				"code_challenge_method=S256",
				"state=test_state",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authURL := provider.GetAuthURL(tt.state, tt.pkce)

			for _, want := range tt.wantContains {
				if !containsString(authURL, want) {
					t.Errorf("GetAuthURL() = %v, want to contain %v", authURL, want)
				}
			}
		})
	}
}

func TestAppleProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  oauth.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config without private key",
			config: oauth.ProviderConfig{
				ClientID:    "com.example.app",
				RedirectURL: "https://example.com/callback",
				TeamID:      "TEAM123",
				KeyID:       "KEY123",
				PrivateKey:  "",
			},
			wantErr: true, // Will fail validation because private key is required
		},
		{
			name: "missing client ID",
			config: oauth.ProviderConfig{
				RedirectURL: "https://example.com/callback",
				TeamID:      "TEAM123",
				KeyID:       "KEY123",
				PrivateKey:  testApplePrivateKey,
			},
			wantErr: true,
		},
		{
			name: "missing team ID",
			config: oauth.ProviderConfig{
				ClientID:    "com.example.app",
				RedirectURL: "https://example.com/callback",
				KeyID:       "KEY123",
				PrivateKey:  testApplePrivateKey,
			},
			wantErr: true,
		},
		{
			name: "missing key ID",
			config: oauth.ProviderConfig{
				ClientID:    "com.example.app",
				RedirectURL: "https://example.com/callback",
				TeamID:      "TEAM123",
				PrivateKey:  testApplePrivateKey,
			},
			wantErr: true,
		},
		{
			name: "missing private key",
			config: oauth.ProviderConfig{
				ClientID:    "com.example.app",
				RedirectURL: "https://example.com/callback",
				TeamID:      "TEAM123",
				KeyID:       "KEY123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := oauth.NewApple(tt.config)
			if err != nil && !tt.wantErr {
				t.Errorf("NewApple() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if provider != nil {
				err = provider.ValidateConfig()
				if (err != nil) != tt.wantErr {
					t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestAppleProvider_ParseIDToken(t *testing.T) {
	provider, err := oauth.NewApple(oauth.ProviderConfig{
		ClientID:    "com.example.app",
		RedirectURL: "https://example.com/callback",
		TeamID:      "TEAM123",
		KeyID:       "KEY123",
		PrivateKey:  "", // No private key for parsing test
	})

	if err != nil {
		t.Fatalf("NewApple() error = %v", err)
	}

	// Test with a mock JWT (header.payload.signature)
	mockIDToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIwMDEyMzQuYWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoiLCJlbWFpbCI6InRlc3RAcHJpdmF0ZXJlbGF5LmFwcGxlaWQuY29tIiwiZW1haWxfdmVyaWZpZWQiOiJ0cnVlIn0.signature"

	claims, err := provider.ParseIDToken(mockIDToken)
	if err != nil {
		t.Errorf("ParseIDToken() error = %v", err)
	}

	if claims == nil {
		t.Error("ParseIDToken() should return non-nil claims")
	}

	// Test user info extraction
	userInfo, err := provider.GetUserInfoFromIDToken(mockIDToken)
	if err != nil {
		t.Errorf("GetUserInfoFromIDToken() error = %v", err)
	}

	if userInfo == nil {
		t.Error("GetUserInfoFromIDToken() should return non-nil user info")
	}

	if userInfo.Provider != "apple" {
		t.Errorf("GetUserInfoFromIDToken() provider = %v, want apple", userInfo.Provider)
	}

	// Test with invalid token
	_, err = provider.ParseIDToken("invalid.token")
	if err == nil {
		t.Error("ParseIDToken() should return error for invalid token")
	}
}

func TestAppleProvider_GetUserInfo(t *testing.T) {
	provider, err := oauth.NewApple(oauth.ProviderConfig{
		ClientID:    "com.example.app",
		RedirectURL: "https://example.com/callback",
		TeamID:      "TEAM123",
		KeyID:       "KEY123",
		PrivateKey:  "", // No private key needed for this test
	})

	if err != nil {
		t.Fatalf("NewApple() error = %v", err)
	}

	// Apple doesn't provide a traditional userinfo endpoint
	_, err = provider.GetUserInfo(context.Background(), "test_token")
	if err == nil {
		t.Error("GetUserInfo() should return error for Apple provider")
	}
}

// Mock Apple private key for testing (this is just for test purposes)
const testApplePrivateKey = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg1234567890abcdef
1234567890abcdef1234567890abcdef1234567890ahRANCAATExample1234567
890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890
abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab
-----END PRIVATE KEY-----`