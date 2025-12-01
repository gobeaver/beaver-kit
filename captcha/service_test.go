package captcha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDisabledService(t *testing.T) {
	service := &DisabledService{}

	ctx := context.Background()
	valid, err := service.Validate(ctx, "any-token", "127.0.0.1")
	if err != nil {
		t.Errorf("DisabledService.Validate() error = %v", err)
	}
	if !valid {
		t.Errorf("DisabledService.Validate() = false, want true")
	}

	html := service.GenerateHTML()
	if html != "" {
		t.Errorf("DisabledService.GenerateHTML() = %v, want empty string", html)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "disabled config needs no validation",
			config: Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "enabled config requires provider",
			config: Config{
				Enabled: true,
			},
			wantErr: true,
			errMsg:  "provider required when enabled",
		},
		{
			name: "enabled config requires keys",
			config: Config{
				Enabled:  true,
				Provider: "recaptcha",
			},
			wantErr: true,
			errMsg:  "both site key and secret key required",
		},
		{
			name: "invalid provider",
			config: Config{
				Enabled:   true,
				Provider:  "invalid",
				SiteKey:   "test",
				SecretKey: "test",
			},
			wantErr: true,
			errMsg:  "invalid captcha provider",
		},
		{
			name: "valid recaptcha v2",
			config: Config{
				Enabled:   true,
				Provider:  "recaptcha",
				SiteKey:   "test-site-key",
				SecretKey: "test-secret-key",
				Version:   2,
			},
			wantErr: false,
		},
		{
			name: "valid recaptcha v3",
			config: Config{
				Enabled:   true,
				Provider:  "recaptcha",
				SiteKey:   "test-site-key",
				SecretKey: "test-secret-key",
				Version:   3,
			},
			wantErr: false,
		},
		{
			name: "invalid recaptcha version",
			config: Config{
				Enabled:   true,
				Provider:  "recaptcha",
				SiteKey:   "test-site-key",
				SecretKey: "test-secret-key",
				Version:   4,
			},
			wantErr: true,
			errMsg:  "invalid recaptcha version",
		},
		{
			name: "valid hcaptcha",
			config: Config{
				Enabled:   true,
				Provider:  "hcaptcha",
				SiteKey:   "test-site-key",
				SecretKey: "test-secret-key",
			},
			wantErr: false,
		},
		{
			name: "valid turnstile",
			config: Config{
				Enabled:   true,
				Provider:  "turnstile",
				SiteKey:   "test-site-key",
				SecretKey: "test-secret-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
				t.Errorf("validateConfig() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantType string
		wantErr  bool
	}{
		{
			name: "disabled returns DisabledService",
			config: Config{
				Enabled: false,
			},
			wantType: "*captcha.DisabledService",
			wantErr:  false,
		},
		{
			name: "recaptcha service",
			config: Config{
				Enabled:   true,
				Provider:  "recaptcha",
				SiteKey:   "test",
				SecretKey: "test",
				Version:   2,
			},
			wantType: "*captcha.GoogleCaptchaService",
			wantErr:  false,
		},
		{
			name: "hcaptcha service",
			config: Config{
				Enabled:   true,
				Provider:  "hcaptcha",
				SiteKey:   "test",
				SecretKey: "test",
			},
			wantType: "*captcha.HCaptchaService",
			wantErr:  false,
		},
		{
			name: "turnstile service",
			config: Config{
				Enabled:   true,
				Provider:  "turnstile",
				SiteKey:   "test",
				SecretKey: "test",
			},
			wantType: "*captcha.TurnstileService",
			wantErr:  false,
		},
		{
			name: "invalid config",
			config: Config{
				Enabled: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && service != nil {
				gotType := getTypeName(service)
				if gotType != tt.wantType {
					t.Errorf("New() returned type = %v, want %v", gotType, tt.wantType)
				}
			}
		})
	}
}

func TestGoogleCaptchaValidate(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and content type
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		// Parse form data
		_ = r.ParseForm()
		secret := r.FormValue("secret")
		token := r.FormValue("response")

		// Return different responses based on token
		var response RecaptchaResponse
		switch token {
		case "valid-token":
			response = RecaptchaResponse{
				Success:     true,
				ChallengeTS: "2024-01-01T00:00:00Z",
				Hostname:    "test.com",
			}
		case "invalid-token":
			response = RecaptchaResponse{
				Success:    false,
				ErrorCodes: []string{"invalid-input-response"},
			}
		default:
			response = RecaptchaResponse{
				Success:    false,
				ErrorCodes: []string{"missing-input-response"},
			}
		}

		// Check secret key
		if secret != "test-secret" {
			response.Success = false
			response.ErrorCodes = []string{"invalid-input-secret"}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create service with test server URL
	service := &GoogleCaptchaService{
		client:    createHTTPClient(),
		siteKey:   "test-site",
		secretKey: "test-secret",
		verifyURL: server.URL,
		version:   2,
	}

	tests := []struct {
		name    string
		token   string
		want    bool
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   "valid-token",
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid-token",
			want:    false,
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := service.Validate(ctx, tt.token, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHCaptchaValidate(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse form data
		_ = r.ParseForm()
		token := r.FormValue("response")

		// Return different responses based on token
		var response HCaptchaResponse
		if token == "valid-token" {
			response = HCaptchaResponse{
				Success:     true,
				ChallengeTS: "2024-01-01T00:00:00Z",
				Hostname:    "test.com",
			}
		} else {
			response = HCaptchaResponse{
				Success:    false,
				ErrorCodes: []string{"invalid-input-response"},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create service with test server URL
	service := &HCaptchaService{
		client:    createHTTPClient(),
		siteKey:   "test-site",
		secretKey: "test-secret",
		verifyURL: server.URL,
	}

	ctx := context.Background()

	// Test valid token
	valid, err := service.Validate(ctx, "valid-token", "")
	if err != nil {
		t.Errorf("Validate() with valid token error = %v", err)
	}
	if !valid {
		t.Errorf("Validate() with valid token = false, want true")
	}

	// Test invalid token
	valid, err = service.Validate(ctx, "invalid-token", "")
	if err == nil {
		t.Errorf("Validate() with invalid token error = nil, want error")
	}
	if valid {
		t.Errorf("Validate() with invalid token = true, want false")
	}
}

func TestTurnstileValidate(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse JSON body
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)

		token := req["response"]

		// Return different responses based on token
		var response TurnstileResponse
		if token == "valid-token" {
			response = TurnstileResponse{
				Success:     true,
				ChallengeTS: "2024-01-01T00:00:00Z",
				Hostname:    "test.com",
			}
		} else {
			response = TurnstileResponse{
				Success:    false,
				ErrorCodes: []string{"invalid-input-response"},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create service with test server URL
	service := &TurnstileService{
		client:    createHTTPClient(),
		siteKey:   "test-site",
		secretKey: "test-secret",
		verifyURL: server.URL,
	}

	ctx := context.Background()

	// Test valid token
	valid, err := service.Validate(ctx, "valid-token", "")
	if err != nil {
		t.Errorf("Validate() with valid token error = %v", err)
	}
	if !valid {
		t.Errorf("Validate() with valid token = false, want true")
	}

	// Test invalid token
	valid, err = service.Validate(ctx, "invalid-token", "")
	if err == nil {
		t.Errorf("Validate() with invalid token error = nil, want error")
	}
	if valid {
		t.Errorf("Validate() with invalid token = true, want false")
	}
}

func TestGenerateHTML(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		contains string
	}{
		{
			name: "Google reCAPTCHA v2",
			service: &GoogleCaptchaService{
				siteKey: "test-site-key",
				version: 2,
			},
			contains: "g-recaptcha",
		},
		{
			name: "Google reCAPTCHA v3",
			service: &GoogleCaptchaService{
				siteKey: "test-site-key",
				version: 3,
			},
			contains: "grecaptcha.execute",
		},
		{
			name: "hCaptcha",
			service: &HCaptchaService{
				siteKey: "test-site-key",
			},
			contains: "h-captcha",
		},
		{
			name: "Turnstile",
			service: &TurnstileService{
				siteKey: "test-site-key",
			},
			contains: "cf-turnstile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := tt.service.GenerateHTML()
			if !containsString(html, tt.contains) {
				t.Errorf("GenerateHTML() does not contain %v", tt.contains)
			}
			if !containsString(html, "test-site-key") {
				t.Errorf("GenerateHTML() does not contain site key")
			}
		})
	}
}

// Helper functions
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s[1:], substr) || len(substr) > 0 && s[0] == substr[0] && containsString(s[1:], substr[1:]))
}

func getTypeName(v interface{}) string {
	if v == nil {
		return "nil"
	}
	switch v.(type) {
	case *DisabledService:
		return "*captcha.DisabledService"
	case *GoogleCaptchaService:
		return "*captcha.GoogleCaptchaService"
	case *HCaptchaService:
		return "*captcha.HCaptchaService"
	case *TurnstileService:
		return "*captcha.TurnstileService"
	default:
		return "unknown"
	}
}
