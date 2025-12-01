package urlsigner

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSignAndVerifyURL(t *testing.T) {
	// Create a signer with a test secret key
	signer := NewSigner("test-secret-key")

	// Test URLs
	tests := []struct {
		name         string
		url          string
		expiry       time.Duration
		payload      string
		shouldVerify bool
	}{
		{
			name:         "Simple URL",
			url:          "https://example.com/resource/123",
			expiry:       10 * time.Minute,
			payload:      "",
			shouldVerify: true,
		},
		{
			name:         "URL with query params",
			url:          "https://example.com/resource?id=123&type=document",
			expiry:       10 * time.Minute,
			payload:      "",
			shouldVerify: true,
		},
		{
			name:         "URL with payload",
			url:          "https://example.com/resource/123",
			expiry:       10 * time.Minute,
			payload:      `{"user_id": 42, "permissions": ["read", "download"]}`,
			shouldVerify: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sign the URL
			signedURL, err := signer.SignURL(tt.url, tt.expiry, tt.payload)
			if err != nil {
				t.Fatalf("Failed to sign URL: %v", err)
			}

			// Verify the signed URL
			valid, payload, err := signer.VerifyURL(signedURL)
			if err != nil {
				t.Fatalf("Error verifying URL: %v", err)
			}

			if !valid {
				t.Errorf("URL verification failed")
			}

			if payload != tt.payload {
				t.Errorf("Payload mismatch. Got: %s, Want: %s", payload, tt.payload)
			}

			// Test expiration
			expiry, err := signer.GetExpirationTime(signedURL)
			if err != nil {
				t.Fatalf("Failed to get expiration time: %v", err)
			}

			expectedExpiry := time.Now().Add(tt.expiry).Unix()
			if expiry.Unix() < expectedExpiry-5 || expiry.Unix() > expectedExpiry+5 {
				t.Errorf("Expiration time mismatch. Got: %v, Want: ~%v", expiry.Unix(), expectedExpiry)
			}

			// Test tampered URL
			tamperedURL := signedURL + "&tampered=true"
			valid, _, err = signer.VerifyURL(tamperedURL)
			if valid || err == nil {
				t.Error("Tampered URL verification should fail")
			}
		})
	}
}

func TestCustomOptions(t *testing.T) {
	// Create a signer with custom options
	customParams := SignatureParams{
		Signature: "s",
		Expires:   "e",
		Payload:   "p",
	}

	options := SignerOptions{
		SecretKey:     "custom-secret",
		DefaultExpiry: 1 * time.Hour,
		Algorithm:     "sha256",
		QueryParams:   &customParams,
	}

	signer := NewSignerWithOptions(options)

	// Test signing and verification
	testURL := "https://example.com/resource/123"
	payload := "test-payload"

	signedURL, err := signer.SignURLWithDefaultExpiry(testURL, payload)
	if err != nil {
		t.Fatalf("Failed to sign URL: %v", err)
	}

	// Verify custom parameter names
	if !strings.Contains(signedURL, "s=") {
		t.Errorf("Custom signature parameter not found")
	}

	if !strings.Contains(signedURL, "e=") {
		t.Errorf("Custom expiration parameter not found")
	}

	if !strings.Contains(signedURL, "p=") {
		t.Errorf("Custom payload parameter not found")
	}

	// Verify the signed URL
	valid, extractedPayload, err := signer.VerifyURL(signedURL)
	if err != nil {
		t.Fatalf("Error verifying URL: %v", err)
	}

	if !valid {
		t.Errorf("URL verification failed")
	}

	if extractedPayload != payload {
		t.Errorf("Payload mismatch. Got: %s, Want: %s", extractedPayload, payload)
	}
}

func TestExpiredURL(t *testing.T) {
	signer := NewSigner("test-secret-key")

	// Create a URL that's already expired (negative duration)
	testURL := "https://example.com/resource/123"
	// Sign with a past expiration by using a custom timestamp
	parsedURL, _ := url.Parse(testURL)
	q := parsedURL.Query()

	// Set expiration to 1 second ago
	expiresAt := time.Now().Add(-1 * time.Second).Unix()
	q.Set("expires", strconv.FormatInt(expiresAt, 10))
	parsedURL.RawQuery = q.Encode()

	// Generate signature
	signature := signer.generateSignature(parsedURL.String(), expiresAt, "")
	q.Set("sig", signature)
	parsedURL.RawQuery = q.Encode()

	signedURL := parsedURL.String()

	// Verify the signed URL
	valid, _, err := signer.VerifyURL(signedURL)
	if !errors.Is(err, ErrExpired) {
		t.Errorf("Expected ErrExpired error for expired URL, got %v", err)
	}
	if valid {
		t.Errorf("Expired URL should not be valid")
	}

	// Test IsExpired function
	expired, err := signer.IsExpired(signedURL)
	if err != nil {
		t.Fatalf("Error checking expiration: %v", err)
	}

	if !expired {
		t.Error("URL should be reported as expired")
	}
}

func TestRemainingValidity(t *testing.T) {
	signer := NewSigner("test-secret-key")

	// Create a URL that expires in 10 minutes
	testURL := "https://example.com/resource/123"
	signedURL, err := signer.SignURL(testURL, 10*time.Minute, "")
	if err != nil {
		t.Fatalf("Failed to sign URL: %v", err)
	}

	// Check remaining validity
	remaining, err := signer.RemainingValidity(signedURL)
	if err != nil {
		t.Fatalf("Error checking remaining validity: %v", err)
	}

	// Should be close to 10 minutes
	if remaining < 9*time.Minute || remaining > 10*time.Minute {
		t.Errorf("Unexpected remaining validity: %v", remaining)
	}
}

func TestGetConfigFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("BEAVER_URLSIGNER_SECRET_KEY", "env-secret-key")
	os.Setenv("BEAVER_URLSIGNER_DEFAULT_EXPIRY", "1h")
	os.Setenv("BEAVER_URLSIGNER_ALGORITHM", "sha256")
	os.Setenv("BEAVER_URLSIGNER_SIGNATURE_PARAM", "s")
	os.Setenv("BEAVER_URLSIGNER_EXPIRES_PARAM", "e")
	os.Setenv("BEAVER_URLSIGNER_PAYLOAD_PARAM", "p")
	defer func() {
		// Clean up
		os.Unsetenv("BEAVER_URLSIGNER_SECRET_KEY")
		os.Unsetenv("BEAVER_URLSIGNER_DEFAULT_EXPIRY")
		os.Unsetenv("BEAVER_URLSIGNER_ALGORITHM")
		os.Unsetenv("BEAVER_URLSIGNER_SIGNATURE_PARAM")
		os.Unsetenv("BEAVER_URLSIGNER_EXPIRES_PARAM")
		os.Unsetenv("BEAVER_URLSIGNER_PAYLOAD_PARAM")
	}()

	cfg, err := GetConfig()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if cfg.SecretKey != "env-secret-key" {
		t.Errorf("Expected secret key 'env-secret-key', got: %s", cfg.SecretKey)
	}

	if cfg.DefaultExpiry != time.Hour {
		t.Errorf("Expected default expiry 1h, got: %v", cfg.DefaultExpiry)
	}

	if cfg.Algorithm != "sha256" {
		t.Errorf("Expected algorithm 'sha256', got: %s", cfg.Algorithm)
	}

	if cfg.SignatureParam != "s" {
		t.Errorf("Expected signature param 's', got: %s", cfg.SignatureParam)
	}
}

func TestInitAndService(t *testing.T) {
	// Reset to ensure clean state
	Reset()
	defer Reset() // Clean up after test

	testConfig := Config{
		SecretKey:      "init-secret-key",
		DefaultExpiry:  15 * time.Minute,
		Algorithm:      "sha256",
		SignatureParam: "sig",
		ExpiresParam:   "expires",
		PayloadParam:   "payload",
	}

	err := Init(testConfig)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Get service instance
	signer := Service()
	if signer == nil {
		t.Fatal("Service() returned nil")
	}

	// Test signing with the service instance
	testURL := "https://example.com/test"
	signedURL, err := signer.SignURL(testURL, 5*time.Minute, "")
	if err != nil {
		t.Fatalf("Failed to sign URL: %v", err)
	}

	// Verify the signed URL
	valid, _, err := signer.VerifyURL(signedURL)
	if err != nil {
		t.Fatalf("Error verifying URL: %v", err)
	}

	if !valid {
		t.Error("URL verification failed")
	}
}

func TestNewWithConfig(t *testing.T) {
	testConfig := Config{
		SecretKey:      "config-secret-key",
		DefaultExpiry:  20 * time.Minute,
		Algorithm:      "sha256",
		SignatureParam: "signature",
		ExpiresParam:   "exp",
		PayloadParam:   "data",
	}

	signer, err := New(testConfig)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Test signing
	testURL := "https://example.com/resource"
	signedURL, err := signer.SignURL(testURL, 0, "test-data") // Use default expiry
	if err != nil {
		t.Fatalf("Failed to sign URL: %v", err)
	}

	// Check that custom parameter names are used
	if !strings.Contains(signedURL, "signature=") {
		t.Error("Custom signature parameter not found")
	}

	if !strings.Contains(signedURL, "exp=") {
		t.Error("Custom expires parameter not found")
	}

	if !strings.Contains(signedURL, "data=") {
		t.Error("Custom payload parameter not found")
	}

	// Verify the URL
	valid, payload, err := signer.VerifyURL(signedURL)
	if err != nil {
		t.Fatalf("Error verifying URL: %v", err)
	}

	if !valid {
		t.Error("URL verification failed")
	}

	if payload != "test-data" {
		t.Errorf("Expected payload 'test-data', got: %s", payload)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: Config{
				SecretKey:     "valid-key",
				DefaultExpiry: 30 * time.Minute,
				Algorithm:     "sha256",
			},
			wantErr: false,
		},
		{
			name: "Missing secret key",
			config: Config{
				SecretKey:     "",
				DefaultExpiry: 30 * time.Minute,
				Algorithm:     "sha256",
			},
			wantErr: true,
		},
		{
			name: "Invalid expiry",
			config: Config{
				SecretKey:     "valid-key",
				DefaultExpiry: 0,
				Algorithm:     "sha256",
			},
			wantErr: true,
		},
		{
			name: "Unsupported algorithm",
			config: Config{
				SecretKey:     "valid-key",
				DefaultExpiry: 30 * time.Minute,
				Algorithm:     "md5",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
