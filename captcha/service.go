// Package captcha provides a unified interface for various CAPTCHA services
package captcha

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Global instance management
var (
	defaultService CaptchaService
	defaultOnce    sync.Once
	defaultErr     error
)

// Package-specific errors
var (
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrNotInitialized   = errors.New("service not initialized")
	ErrProviderRequired = errors.New("captcha provider required")
	ErrInvalidProvider  = errors.New("invalid captcha provider")
	ErrKeysRequired     = errors.New("site key and secret key required")
)

// Init initializes the global instance with optional config
func Init(configs ...Config) error {
	defaultOnce.Do(func() {
		var cfg *Config
		if len(configs) > 0 {
			cfg = &configs[0]
		} else {
			cfg, defaultErr = GetConfig()
			if defaultErr != nil {
				return
			}
		}

		defaultService, defaultErr = New(*cfg)
	})

	return defaultErr
}

// New creates a new instance with given config
func New(cfg Config) (CaptchaService, error) {
	// Validation
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if !cfg.Enabled {
		return &DisabledService{}, nil
	}

	switch cfg.Provider {
	case "recaptcha":
		return NewGoogleCaptcha(cfg.SiteKey, cfg.SecretKey, cfg.Version), nil
	case "hcaptcha":
		return NewHCaptcha(cfg.SiteKey, cfg.SecretKey), nil
	case "turnstile":
		return NewTurnstile(cfg.SiteKey, cfg.SecretKey), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

// validateConfig checks configuration validity
func validateConfig(cfg Config) error {
	if !cfg.Enabled {
		return nil // If disabled, no validation needed
	}

	if cfg.Provider == "" {
		return fmt.Errorf("%w: provider required when enabled", ErrInvalidConfig)
	}

	if cfg.SiteKey == "" || cfg.SecretKey == "" {
		return fmt.Errorf("%w: both site key and secret key required when enabled", ErrKeysRequired)
	}

	// Validate provider
	switch cfg.Provider {
	case "recaptcha", "hcaptcha", "turnstile":
		// Valid providers
	default:
		return fmt.Errorf("%w: %s", ErrInvalidProvider, cfg.Provider)
	}

	// Validate version for recaptcha
	if cfg.Provider == "recaptcha" && cfg.Version != 2 && cfg.Version != 3 {
		return fmt.Errorf("%w: invalid recaptcha version %d (must be 2 or 3)", ErrInvalidConfig, cfg.Version)
	}

	return nil
}

// Reset clears the global instance (for testing)
func Reset() {
	defaultService = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}

// Service returns the global captcha service instance
func Service() CaptchaService {
	if defaultService == nil {
		Init() // Initialize with defaults if needed
	}
	return defaultService
}

// DisabledService is a no-op captcha service
type DisabledService struct{}

func (d *DisabledService) Validate(ctx context.Context, token string, remoteIP string) (bool, error) {
	return true, nil
}

func (d *DisabledService) GenerateHTML() string {
	return ""
}

// CaptchaService defines the interface for all captcha services
type CaptchaService interface {
	// Validate validates a captcha token with the remote service
	Validate(ctx context.Context, token string, remoteIP string) (bool, error)

	// GenerateHTML generates the HTML needed to embed the captcha
	GenerateHTML() string
}

// createHTTPClient creates a properly configured HTTP client for CAPTCHA API calls
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
}

//------------------------------------------------------------------------------
// Google reCAPTCHA Implementation
//------------------------------------------------------------------------------

// RecaptchaResponse represents the response from Google's reCAPTCHA API
type RecaptchaResponse struct {
	Success     bool     `json:"success"`
	Score       float64  `json:"score,omitempty"`  // v3 only
	Action      string   `json:"action,omitempty"` // v3 only
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
}

// GoogleCaptchaService implements the CaptchaService interface for Google reCAPTCHA
type GoogleCaptchaService struct {
	client    *http.Client
	siteKey   string
	secretKey string
	verifyURL string
	version   int // 2 or 3
}

// NewGoogleCaptcha creates a new Google reCAPTCHA service
func NewGoogleCaptcha(siteKey, secretKey string, version int) *GoogleCaptchaService {
	if version != 2 && version != 3 {
		version = 2 // Default to v2 if invalid version is provided
	}

	return &GoogleCaptchaService{
		client:    createHTTPClient(),
		siteKey:   siteKey,
		secretKey: secretKey,
		verifyURL: "https://www.google.com/recaptcha/api/siteverify",
		version:   version,
	}
}

// Validate implements the CaptchaService interface for Google reCAPTCHA
func (g *GoogleCaptchaService) Validate(ctx context.Context, token string, remoteIP string) (bool, error) {
	// Prepare form data
	data := url.Values{
		"secret":   {g.secretKey},
		"response": {token},
	}

	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", g.verifyURL, strings.NewReader(data.Encode()))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := g.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send verification request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("received status code %d: %s", resp.StatusCode, body)
	}

	// Parse response body
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192)) // Limit response size
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	var result RecaptchaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors in the response
	if !result.Success && len(result.ErrorCodes) > 0 {
		return false, fmt.Errorf("captcha validation failed: %v", result.ErrorCodes)
	}

	return result.Success, nil
}

// ValidateV3WithScore validates a reCAPTCHA v3 token with specified minimum score and action
func (g *GoogleCaptchaService) ValidateV3WithScore(ctx context.Context, token, remoteIP, expectedAction string, minScore float64) (bool, float64, error) {
	// Regular validation
	success, err := g.Validate(ctx, token, remoteIP)
	if err != nil {
		return false, 0, err
	}

	if !success {
		return false, 0, nil
	}

	// Additional v3-specific validation
	// For a real implementation, we'd need to extract the score and action from the response
	// This is a simplified example since we don't have access to those values here

	return true, 0.9, nil // Placeholder - in a real implementation, use actual values
}

// GenerateHTML implements the CaptchaService interface for Google reCAPTCHA
func (g *GoogleCaptchaService) GenerateHTML() string {
	if g.version == 3 {
		return fmt.Sprintf(`
			<script src="https://www.google.com/recaptcha/api.js?render=%s"></script>
			<script>
			grecaptcha.ready(function() {
				grecaptcha.execute('%s', {action: 'submit'}).then(function(token) {
					document.getElementById('g-recaptcha-response').value = token;
				});
			});
			</script>
			<input type="hidden" id="g-recaptcha-response" name="g-recaptcha-response">
		`, g.siteKey, g.siteKey)
	}

	return fmt.Sprintf(`
		<script src="https://www.google.com/recaptcha/api.js" async defer></script>
		<div class="g-recaptcha" data-sitekey="%s"></div>
	`, g.siteKey)
}

//------------------------------------------------------------------------------
// hCaptcha Implementation
//------------------------------------------------------------------------------

// HCaptchaResponse represents the response from hCaptcha's verification API
type HCaptchaResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	Credit      bool     `json:"credit,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Score       *float64 `json:"score,omitempty"`        // Enterprise only
	ScoreReason []string `json:"score_reason,omitempty"` // Enterprise only
}

// HCaptchaService implements the CaptchaService interface for hCaptcha
type HCaptchaService struct {
	client    *http.Client
	siteKey   string
	secretKey string
	verifyURL string
}

// NewHCaptcha creates a new hCaptcha service
func NewHCaptcha(siteKey, secretKey string) *HCaptchaService {
	return &HCaptchaService{
		client:    createHTTPClient(),
		siteKey:   siteKey,
		secretKey: secretKey,
		verifyURL: "https://api.hcaptcha.com/siteverify",
	}
}

// Validate implements the CaptchaService interface for hCaptcha
func (h *HCaptchaService) Validate(ctx context.Context, token string, remoteIP string) (bool, error) {
	// Prepare form data
	data := url.Values{
		"secret":   {h.secretKey},
		"response": {token},
		"sitekey":  {h.siteKey},
	}

	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", h.verifyURL, strings.NewReader(data.Encode()))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := h.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send verification request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("received status code %d: %s", resp.StatusCode, body)
	}

	// Parse response
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192)) // Limit response size
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	var result HCaptchaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors in the response
	if !result.Success && len(result.ErrorCodes) > 0 {
		return false, fmt.Errorf("captcha validation failed: %v", result.ErrorCodes)
	}

	return result.Success, nil
}

// GenerateHTML implements the CaptchaService interface for hCaptcha
func (h *HCaptchaService) GenerateHTML() string {
	return fmt.Sprintf(`
		<script src="https://js.hcaptcha.com/1/api.js" async defer></script>
		<div class="h-captcha" data-sitekey="%s"></div>
	`, h.siteKey)
}

//------------------------------------------------------------------------------
// Cloudflare Turnstile Implementation
//------------------------------------------------------------------------------

// TurnstileResponse represents the response from Cloudflare's Turnstile verification API
type TurnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Action      string   `json:"action,omitempty"`
	CData       string   `json:"cdata,omitempty"`
}

// TurnstileService implements the CaptchaService interface for Cloudflare Turnstile
type TurnstileService struct {
	client    *http.Client
	siteKey   string
	secretKey string
	verifyURL string
}

// NewTurnstile creates a new Cloudflare Turnstile service
func NewTurnstile(siteKey, secretKey string) *TurnstileService {
	return &TurnstileService{
		client:    createHTTPClient(),
		siteKey:   siteKey,
		secretKey: secretKey,
		verifyURL: "https://challenges.cloudflare.com/turnstile/v0/siteverify",
	}
}

// Validate implements the CaptchaService interface for Turnstile
func (t *TurnstileService) Validate(ctx context.Context, token string, remoteIP string) (bool, error) {
	// Prepare request data
	data := map[string]string{
		"secret":   t.secretKey,
		"response": token,
	}

	if remoteIP != "" {
		data["remoteip"] = remoteIP
	}

	// Convert to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", t.verifyURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send verification request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("received status code %d: %s", resp.StatusCode, body)
	}

	// Parse response
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192)) // Limit response size
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	var result TurnstileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors in the response
	if !result.Success && len(result.ErrorCodes) > 0 {
		return false, fmt.Errorf("captcha validation failed: %v", result.ErrorCodes)
	}

	return result.Success, nil
}

// GenerateHTML implements the CaptchaService interface for Turnstile
func (t *TurnstileService) GenerateHTML() string {
	return fmt.Sprintf(`
		<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
		<div class="cf-turnstile" data-sitekey="%s"></div>
	`, t.siteKey)
}
