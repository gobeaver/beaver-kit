package oauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gobeaver/beaver-kit/oauth"
)

func TestTwitterProvider_Basic(t *testing.T) {
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
}

func TestTwitterProvider_GetAuthURL(t *testing.T) {
	provider := oauth.NewTwitter(oauth.ProviderConfig{
		ClientID:    "twitter_client_id",
		RedirectURL: "https://example.com/callback",
		Scopes:      []string{"tweet.read", "users.read", "offline.access"},
	})

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
				"https://twitter.com/i/oauth2/authorize",
				"client_id=twitter_client_id",
				"redirect_uri=https%3A%2F%2Fexample.com%2Fcallback",
				"scope=tweet.read+users.read+offline.access",
				"state=test_state",
				"response_type=code",
				"code_challenge_method=S256",
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
				"https://twitter.com/i/oauth2/authorize",
				"code_challenge=test_challenge",
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

func TestTwitterProvider_Exchange(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		wantErr        bool
		wantToken      *oauth.Token
	}{
		{
			name: "successful exchange",
			serverResponse: `{
				"access_token": "twitter_access_token_123",
				"token_type": "bearer",
				"refresh_token": "twitter_refresh_token_123",
				"expires_in": 7200,
				"scope": "tweet.read users.read offline.access"
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			wantToken: &oauth.Token{
				AccessToken:  "twitter_access_token_123",
				TokenType:    "bearer",
				RefreshToken: "twitter_refresh_token_123",
				ExpiresIn:    7200,
				Scope:        "tweet.read users.read offline.access",
			},
		},
		{
			name: "error response",
			serverResponse: `{
				"error": "invalid_grant",
				"error_description": "Value passed for the authorization code was invalid."
			}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check that request includes proper headers
				if r.Header.Get("Accept") != "application/json" {
					t.Errorf("Expected Accept: application/json header")
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create provider with mock server URL
			provider := oauth.NewTwitter(oauth.ProviderConfig{
				ClientID:    "twitter_client_id",
				RedirectURL: "https://example.com/callback",
				TokenURL:    server.URL,
			})

			// Test exchange
			token, err := provider.Exchange(context.Background(), "test_code", &oauth.PKCEChallenge{
				Verifier: "test_verifier",
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("Exchange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantToken != nil {
				if token.AccessToken != tt.wantToken.AccessToken {
					t.Errorf("Exchange() AccessToken = %v, want %v", token.AccessToken, tt.wantToken.AccessToken)
				}
				if token.TokenType != tt.wantToken.TokenType {
					t.Errorf("Exchange() TokenType = %v, want %v", token.TokenType, tt.wantToken.TokenType)
				}
				if token.RefreshToken != tt.wantToken.RefreshToken {
					t.Errorf("Exchange() RefreshToken = %v, want %v", token.RefreshToken, tt.wantToken.RefreshToken)
				}
			}
		})
	}
}

func TestTwitterProvider_GetUserInfo(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		wantErr        bool
		wantUserInfo   *oauth.UserInfo
	}{
		{
			name: "successful user info",
			serverResponse: `{
				"data": {
					"id": "123456789",
					"username": "testuser",
					"name": "Test User",
					"profile_image_url": "https://pbs.twimg.com/profile_images/test.jpg",
					"description": "This is a test user",
					"location": "Test City",
					"url": "https://example.com",
					"verified": true,
					"protected": false,
					"public_metrics": {
						"followers_count": 1000,
						"following_count": 500,
						"tweet_count": 2000,
						"listed_count": 10
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			wantUserInfo: &oauth.UserInfo{
				ID:       "123456789",
				Name:     "Test User",
				Picture:  "https://pbs.twimg.com/profile_images/test.jpg",
				Provider: "twitter",
			},
		},
		{
			name: "API error response",
			serverResponse: `{
				"errors": [
					{
						"detail": "Unauthorized",
						"title": "Unauthorized Request",
						"type": "about:blank"
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check authorization header
				auth := r.Header.Get("Authorization")
				if auth != "Bearer test_access_token" {
					t.Errorf("Expected Authorization: Bearer test_access_token, got %s", auth)
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			provider := oauth.NewTwitter(oauth.ProviderConfig{
				UserInfoURL: server.URL,
			})

			userInfo, err := provider.GetUserInfo(context.Background(), "test_access_token")

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantUserInfo != nil {
				if userInfo.ID != tt.wantUserInfo.ID {
					t.Errorf("GetUserInfo() ID = %v, want %v", userInfo.ID, tt.wantUserInfo.ID)
				}
				if userInfo.Name != tt.wantUserInfo.Name {
					t.Errorf("GetUserInfo() Name = %v, want %v", userInfo.Name, tt.wantUserInfo.Name)
				}
				if userInfo.Provider != tt.wantUserInfo.Provider {
					t.Errorf("GetUserInfo() Provider = %v, want %v", userInfo.Provider, tt.wantUserInfo.Provider)
				}

				// Check Twitter-specific fields in raw data
				if username, ok := userInfo.Raw["username"]; !ok || username != "testuser" {
					t.Errorf("GetUserInfo() Raw[username] = %v, want testuser", username)
				}
			}
		})
	}
}

func TestTwitterProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  oauth.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config with client secret",
			config: oauth.ProviderConfig{
				ClientID:     "twitter_client_id",
				ClientSecret: "twitter_client_secret",
				RedirectURL:  "https://example.com/callback",
			},
			wantErr: false,
		},
		{
			name: "valid config without client secret (public client)",
			config: oauth.ProviderConfig{
				ClientID:    "twitter_client_id",
				RedirectURL: "https://example.com/callback",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: oauth.ProviderConfig{
				RedirectURL: "https://example.com/callback",
			},
			wantErr: true,
		},
		{
			name: "missing redirect URL",
			config: oauth.ProviderConfig{
				ClientID: "twitter_client_id",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := oauth.NewTwitter(tt.config)
			err := provider.ValidateConfig()

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTwitterProvider_PKCEFlow(t *testing.T) {
	provider := oauth.NewTwitter(oauth.ProviderConfig{
		ClientID:    "twitter_client_id",
		RedirectURL: "https://example.com/callback",
	})

	// Generate PKCE challenge
	pkce, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("GeneratePKCEChallenge() error = %v", err)
	}

	// Test auth URL with PKCE
	authURL := provider.GetAuthURL("test_state", pkce)
	if !containsString(authURL, "code_challenge") {
		t.Error("Auth URL should contain code_challenge parameter")
	}
	if !containsString(authURL, "code_challenge_method=S256") {
		t.Error("Auth URL should contain code_challenge_method=S256 parameter")
	}
}