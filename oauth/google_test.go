package oauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gobeaver/beaver-kit/oauth"
)

func TestGoogleProvider_GetAuthURL(t *testing.T) {
	provider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:    "test_client_id",
		RedirectURL: "http://localhost:8080/callback",
		Scopes:      []string{"openid", "profile", "email"},
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
				"https://accounts.google.com/o/oauth2/v2/auth",
				"client_id=test_client_id",
				"redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback",
				"scope=openid+profile+email",
				"state=test_state",
				"response_type=code",
				"access_type=offline",
				"prompt=consent",
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
				"https://accounts.google.com/o/oauth2/v2/auth",
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

func TestGoogleProvider_Exchange(t *testing.T) {
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
				"access_token": "ya29.test_access_token",
				"token_type": "Bearer",
				"refresh_token": "1//test_refresh_token",
				"expires_in": 3600,
				"id_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.test",
				"scope": "openid profile email"
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			wantToken: &oauth.Token{
				AccessToken:  "ya29.test_access_token",
				TokenType:    "Bearer",
				RefreshToken: "1//test_refresh_token",
				ExpiresIn:    3600,
				IDToken:      "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.test",
				Scope:        "openid profile email",
			},
		},
		{
			name: "error response",
			serverResponse: `{
				"error": "invalid_grant",
				"error_description": "Bad Request"
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
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create provider with mock server URL
			provider := oauth.NewGoogle(oauth.ProviderConfig{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				RedirectURL:  "http://localhost:8080/callback",
				TokenURL:     server.URL,
			})

			// Test exchange
			token, err := provider.Exchange(context.Background(), "test_code", nil)

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

func TestGoogleProvider_RefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		wantErr        bool
	}{
		{
			name: "successful refresh",
			serverResponse: `{
				"access_token": "ya29.new_access_token",
				"token_type": "Bearer",
				"expires_in": 3600,
				"scope": "openid profile email"
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "invalid refresh token",
			serverResponse: `{
				"error": "invalid_grant",
				"error_description": "Token has been expired or revoked"
			}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			provider := oauth.NewGoogle(oauth.ProviderConfig{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				TokenURL:     server.URL,
			})

			token, err := provider.RefreshToken(context.Background(), "test_refresh_token")

			if (err != nil) != tt.wantErr {
				t.Errorf("RefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && token == nil {
				t.Error("RefreshToken() expected non-nil token")
			}
		})
	}
}

func TestGoogleProvider_GetUserInfo(t *testing.T) {
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
				"id": "12345",
				"email": "test@gmail.com",
				"verified_email": true,
				"name": "Test User",
				"given_name": "Test",
				"family_name": "User",
				"picture": "https://lh3.googleusercontent.com/a/test",
				"locale": "en"
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			wantUserInfo: &oauth.UserInfo{
				ID:            "12345",
				Email:         "test@gmail.com",
				EmailVerified: true,
				Name:          "Test User",
				FirstName:     "Test",
				LastName:      "User",
				Picture:       "https://lh3.googleusercontent.com/a/test",
				Locale:        "en",
				Provider:      "google",
			},
		},
		{
			name:           "unauthorized",
			serverResponse: "Unauthorized",
			statusCode:     http.StatusUnauthorized,
			wantErr:        true,
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
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			provider := oauth.NewGoogle(oauth.ProviderConfig{
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
				if userInfo.Email != tt.wantUserInfo.Email {
					t.Errorf("GetUserInfo() Email = %v, want %v", userInfo.Email, tt.wantUserInfo.Email)
				}
				if userInfo.Name != tt.wantUserInfo.Name {
					t.Errorf("GetUserInfo() Name = %v, want %v", userInfo.Name, tt.wantUserInfo.Name)
				}
			}
		})
	}
}

func TestGoogleProvider_Methods(t *testing.T) {
	provider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
	})

	// Test provider info methods
	if provider.Name() != "google" {
		t.Errorf("Name() = %v, want google", provider.Name())
	}

	if !provider.SupportsRefresh() {
		t.Error("SupportsRefresh() = false, want true")
	}

	if !provider.SupportsPKCE() {
		t.Error("SupportsPKCE() = false, want true")
	}
}

func TestGoogleProvider_PKCEFlow(t *testing.T) {
	provider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
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

func TestGoogleProvider_ParseIDToken(t *testing.T) {
	provider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
	})

	// Test with a mock JWT (header.payload.signature)
	// This is a simplified test - in production, you'd want proper JWT validation
	mockIDToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.signature"

	claims, err := provider.ParseIDToken(mockIDToken)
	if err != nil {
		t.Errorf("ParseIDToken() error = %v", err)
	}

	if claims == nil {
		t.Error("ParseIDToken() should return non-nil claims")
	}

	// Test with invalid token
	_, err = provider.ParseIDToken("invalid.token")
	if err == nil {
		t.Error("ParseIDToken() should return error for invalid token")
	}
}
