package urlsigner

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// Global instance management
var (
	defaultInstance *Signer
	defaultOnce     sync.Once
	defaultErr      error
)

// Define standard errors for the package
var (
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrNotInitialized     = errors.New("service not initialized")
	ErrInvalidURL         = errors.New("invalid URL")
	ErrSignatureNotFound  = errors.New("signature not found")
	ErrExpirationNotFound = errors.New("expiration not found")
	ErrExpired            = errors.New("URL has expired")
	ErrInvalidSignature   = errors.New("invalid signature")
)

// Signer handles URL signing operations
type Signer struct {
	secretKey     string
	defaultExpiry time.Duration
	algorithm     string
	queryParams   SignatureParams
}

// SignatureParams customizes how signature parameters appear in URLs
type SignatureParams struct {
	Signature string // query parameter name for signature
	Expires   string // query parameter name for expiration
	Payload   string // query parameter name for additional payload
}

// SignerOptions configures the Signer behavior
type SignerOptions struct {
	SecretKey     string
	DefaultExpiry time.Duration
	Algorithm     string
	QueryParams   *SignatureParams
}

// DefaultSignatureParams returns standard query parameter names
func DefaultSignatureParams() SignatureParams {
	return SignatureParams{
		Signature: "sig",
		Expires:   "expires",
		Payload:   "payload",
	}
}

// GetConfig returns config loaded from environment
func GetConfig() (*Config, error) {
	// Create a temporary struct to handle string duration
	type tempConfig struct {
		SecretKey      string `env:"BEAVER_URLSIGNER_SECRET_KEY,required"`
		DefaultExpiry  string `env:"BEAVER_URLSIGNER_DEFAULT_EXPIRY,default:30m"`
		Algorithm      string `env:"BEAVER_URLSIGNER_ALGORITHM,default:sha256"`
		SignatureParam string `env:"BEAVER_URLSIGNER_SIGNATURE_PARAM,default:sig"`
		ExpiresParam   string `env:"BEAVER_URLSIGNER_EXPIRES_PARAM,default:expires"`
		PayloadParam   string `env:"BEAVER_URLSIGNER_PAYLOAD_PARAM,default:payload"`
	}

	tmpCfg := &tempConfig{}
	if err := config.Load(tmpCfg); err != nil {
		return nil, err
	}

	// Parse duration
	duration, err := time.ParseDuration(tmpCfg.DefaultExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid default expiry duration: %w", err)
	}

	cfg := &Config{
		SecretKey:      tmpCfg.SecretKey,
		DefaultExpiry:  duration,
		Algorithm:      tmpCfg.Algorithm,
		SignatureParam: tmpCfg.SignatureParam,
		ExpiresParam:   tmpCfg.ExpiresParam,
		PayloadParam:   tmpCfg.PayloadParam,
	}

	return cfg, nil
}

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

		defaultInstance, defaultErr = New(*cfg)
	})

	return defaultErr
}

// New creates a new instance with given config
func New(cfg Config) (*Signer, error) {
	// Validation
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	// Initialize
	return &Signer{
		secretKey:     cfg.SecretKey,
		defaultExpiry: cfg.DefaultExpiry,
		algorithm:     cfg.Algorithm,
		queryParams: SignatureParams{
			Signature: cfg.SignatureParam,
			Expires:   cfg.ExpiresParam,
			Payload:   cfg.PayloadParam,
		},
	}, nil
}

// validateConfig checks configuration validity
func validateConfig(cfg Config) error {
	if cfg.SecretKey == "" {
		return fmt.Errorf("secret key required")
	}

	if cfg.DefaultExpiry <= 0 {
		return fmt.Errorf("default expiry must be positive")
	}

	if cfg.Algorithm != "sha256" {
		return fmt.Errorf("unsupported algorithm: %s (only sha256 supported)", cfg.Algorithm)
	}

	return nil
}

// Reset clears the global instance (for testing)
func Reset() {
	defaultInstance = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}

// Service returns the global signer instance
func Service() *Signer {
	if defaultInstance == nil {
		Init() // Initialize with defaults if needed
	}
	return defaultInstance
}

// NewSigner creates a new URL signer with the given secret key and default options
func NewSigner(secretKey string) *Signer {
	return &Signer{
		secretKey:     secretKey,
		defaultExpiry: 30 * time.Minute,
		algorithm:     "sha256",
		queryParams:   DefaultSignatureParams(),
	}
}

// NewSignerWithOptions creates a new URL signer with custom options
func NewSignerWithOptions(options SignerOptions) *Signer {
	signer := &Signer{
		secretKey:     options.SecretKey,
		defaultExpiry: 30 * time.Minute,
		algorithm:     "sha256",
		queryParams:   DefaultSignatureParams(),
	}

	if options.DefaultExpiry > 0 {
		signer.defaultExpiry = options.DefaultExpiry
	}

	if options.Algorithm != "" {
		signer.algorithm = options.Algorithm
	}

	if options.QueryParams != nil {
		if options.QueryParams.Signature != "" {
			signer.queryParams.Signature = options.QueryParams.Signature
		}
		if options.QueryParams.Expires != "" {
			signer.queryParams.Expires = options.QueryParams.Expires
		}
		if options.QueryParams.Payload != "" {
			signer.queryParams.Payload = options.QueryParams.Payload
		}
	}

	return signer
}

// SignURL signs a URL with an expiration time and optional payload
func (s *Signer) SignURL(rawURL string, expiry time.Duration, payload string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Set default expiry if not provided
	if expiry <= 0 {
		expiry = s.defaultExpiry
	}

	// Calculate expiration timestamp
	expiresAt := time.Now().Add(expiry).Unix()

	// Add expiration to query params
	q := parsedURL.Query()
	q.Set(s.queryParams.Expires, strconv.FormatInt(expiresAt, 10))

	// Add payload if provided
	if payload != "" {
		encodedPayload := base64.URLEncoding.EncodeToString([]byte(payload))
		q.Set(s.queryParams.Payload, encodedPayload)
	}

	// Update URL with query params before generating signature
	parsedURL.RawQuery = q.Encode()

	// Generate signature
	signature := s.generateSignature(parsedURL.String(), expiresAt, payload)

	// Add signature to query params
	q.Set(s.queryParams.Signature, signature)
	parsedURL.RawQuery = q.Encode()

	return parsedURL.String(), nil
}

// SignURLWithDefaultExpiry signs a URL with the default expiration time
func (s *Signer) SignURLWithDefaultExpiry(rawURL string, payload string) (string, error) {
	return s.SignURL(rawURL, s.defaultExpiry, payload)
}

// VerifyURL checks if a signed URL is valid and not expired
func (s *Signer) VerifyURL(signedURL string) (bool, string, error) {
	parsedURL, err := url.Parse(signedURL)
	if err != nil {
		return false, "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Extract query parameters
	q := parsedURL.Query()

	// Get signature
	signature := q.Get(s.queryParams.Signature)
	if signature == "" {
		return false, "", ErrSignatureNotFound
	}

	// Get expiration timestamp
	expiresStr := q.Get(s.queryParams.Expires)
	if expiresStr == "" {
		return false, "", ErrExpirationNotFound
	}

	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return false, "", fmt.Errorf("invalid expiration: %w", err)
	}

	// Check if URL has expired
	if time.Now().Unix() > expires {
		return false, "", ErrExpired
	}

	// Get payload if present
	var payload string
	encodedPayload := q.Get(s.queryParams.Payload)
	if encodedPayload != "" {
		payloadBytes, err := base64.URLEncoding.DecodeString(encodedPayload)
		if err != nil {
			return false, "", fmt.Errorf("invalid payload: %w", err)
		}
		payload = string(payloadBytes)
	}

	// Remove signature from URL for verification
	q.Del(s.queryParams.Signature)
	parsedURL.RawQuery = q.Encode()

	// Calculate expected signature
	expectedSignature := s.generateSignature(parsedURL.String(), expires, payload)

	// Verify signature
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return false, "", ErrInvalidSignature
	}

	return true, payload, nil
}

// GetExpirationTime returns the expiration time from a signed URL
func (s *Signer) GetExpirationTime(signedURL string) (time.Time, error) {
	parsedURL, err := url.Parse(signedURL)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Extract expiration timestamp
	expiresStr := parsedURL.Query().Get(s.queryParams.Expires)
	if expiresStr == "" {
		return time.Time{}, ErrExpirationNotFound
	}

	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid expiration: %w", err)
	}

	return time.Unix(expires, 0), nil
}

// generateSignature creates a signature for the given URL, expiration and payload
func (s *Signer) generateSignature(urlString string, expires int64, payload string) string {
	// Combine URL, expiration and payload for signing
	dataToSign := fmt.Sprintf("%s|%d", urlString, expires)
	if payload != "" {
		dataToSign = fmt.Sprintf("%s|%s", dataToSign, payload)
	}

	// Create HMAC
	h := hmac.New(sha256.New, []byte(s.secretKey))
	h.Write([]byte(dataToSign))

	// Return hex-encoded signature
	return hex.EncodeToString(h.Sum(nil))
}

// ExtractPayload extracts and returns the payload from a signed URL
func (s *Signer) ExtractPayload(signedURL string) (string, error) {
	parsedURL, err := url.Parse(signedURL)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	encodedPayload := parsedURL.Query().Get(s.queryParams.Payload)
	if encodedPayload == "" {
		return "", nil // No payload
	}

	payloadBytes, err := base64.URLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return "", fmt.Errorf("invalid payload encoding: %w", err)
	}

	return string(payloadBytes), nil
}

// IsExpired checks if a signed URL has expired
func (s *Signer) IsExpired(signedURL string) (bool, error) {
	parsedURL, err := url.Parse(signedURL)
	if err != nil {
		return true, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	expiresStr := parsedURL.Query().Get(s.queryParams.Expires)
	if expiresStr == "" {
		return true, ErrExpirationNotFound
	}

	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return true, fmt.Errorf("invalid expiration: %w", err)
	}

	return time.Now().Unix() > expires, nil
}

// RemainingValidity returns the remaining validity time of a signed URL
func (s *Signer) RemainingValidity(signedURL string) (time.Duration, error) {
	expirationTime, err := s.GetExpirationTime(signedURL)
	if err != nil {
		return 0, err
	}

	remaining := time.Until(expirationTime)
	if remaining < 0 {
		return 0, nil // URL has already expired
	}

	return remaining, nil
}
