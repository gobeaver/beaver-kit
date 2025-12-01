// Package slack provides methods to send notifications to Slack channels via webhooks.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// Global instance management
var (
	defaultService *Service
	defaultOnce    sync.Once
	defaultErr     error
)

// Builder provides a way to create Slack service instances with custom prefixes
type Builder struct {
	prefix string
}

// WithPrefix creates a new Builder with the specified prefix
func WithPrefix(prefix string) *Builder {
	return &Builder{prefix: prefix}
}

// Init initializes the global Slack service using the builder's prefix
func (b *Builder) Init() error {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return err
	}
	return Init(*cfg)
}

// New creates a new Slack service using the builder's prefix
func (b *Builder) New() (*Service, error) {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return nil, err
	}
	return New(*cfg)
}

// Service represents a Slack notification service
type Service struct {
	webhookURL    string
	httpClient    *http.Client
	defaultOpts   *MessageOptions
	maxRetries    int
	retryDelay    time.Duration
	retryMaxDelay time.Duration
	retryJitter   bool
	debug         bool

	// Production features
	rateLimiter    RateLimiter
	circuitBreaker *CircuitBreaker
	metrics        *Metrics
	logger         *Logger
	requestLogger  *RequestLogger

	// Security
	maxMessageSize int
	sanitizeInput  bool
	redactErrors   bool

	// State management
	mu             sync.RWMutex
	shutdown       chan struct{}
	wg             sync.WaitGroup
	isShuttingDown bool
}

// MessageOptions contains optional parameters for Slack messages
type MessageOptions struct {
	Channel   string
	Username  string
	IconEmoji string
	IconURL   string
}

// Message represents a Slack message to be sent
type Message struct {
	Text      string `json:"text"`
	Channel   string `json:"channel,omitempty"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	IconURL   string `json:"icon_url,omitempty"`
}

// Init initializes the global instance with optional config
func Init(configs ...Config) error {
	defaultOnce.Do(func() {
		var cfg *Config
		if len(configs) > 0 {
			cfg = &configs[0]
		} else {
			cfg, defaultErr = GetConfig(config.LoadOptions{Prefix: "BEAVER_SLACK_"})
			if defaultErr != nil {
				return
			}
		}

		defaultService, defaultErr = New(*cfg)
	})

	return defaultErr
}

// New creates a new instance with given config
func New(cfg Config) (*Service, error) {
	// Validation
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create default options from config
	defaultOpts := &MessageOptions{
		Channel:   cfg.Channel,
		Username:  cfg.Username,
		IconEmoji: cfg.IconEmoji,
		IconURL:   cfg.IconURL,
	}

	// Create rate limiter
	var rateLimiter RateLimiter
	if cfg.RateLimit > 0 {
		rateLimiter = NewTokenBucketLimiter(cfg.RateLimit, cfg.RateBurst)
	} else {
		rateLimiter = &NoOpLimiter{}
	}

	// Create circuit breaker
	circuitBreaker := NewCircuitBreaker(
		cfg.CircuitThreshold,
		cfg.CircuitTimeout,
		cfg.CircuitMaxRequests,
	)

	// Create metrics
	var metrics *Metrics
	if cfg.EnableMetrics {
		metrics = NewMetrics()
	}

	// Create logger
	logger := NewLogger(cfg.EnableLogging, cfg.LogLevel)
	requestLogger := NewRequestLogger(logger, cfg.RedactErrors)

	// Initialize
	return &Service{
		webhookURL: cfg.WebhookURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		defaultOpts:    defaultOpts,
		maxRetries:     cfg.MaxRetries,
		retryDelay:     cfg.RetryDelay,
		retryMaxDelay:  cfg.RetryMaxDelay,
		retryJitter:    cfg.RetryJitter,
		debug:          cfg.Debug,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
		metrics:        metrics,
		logger:         logger,
		requestLogger:  requestLogger,
		maxMessageSize: cfg.MaxMessageSize,
		sanitizeInput:  cfg.SanitizeInput,
		redactErrors:   cfg.RedactErrors,
		shutdown:       make(chan struct{}),
	}, nil
}

// validateConfig checks configuration validity
func validateConfig(cfg Config) error {
	if cfg.WebhookURL == "" {
		return fmt.Errorf("%w: webhook URL required", ErrInvalidConfig)
	}

	// Validate webhook URL format
	if _, err := url.Parse(cfg.WebhookURL); err != nil {
		return fmt.Errorf("%w: invalid webhook URL format", ErrInvalidConfig)
	}

	// Icon validation - can't have both IconEmoji and IconURL
	if cfg.IconEmoji != "" && cfg.IconURL != "" {
		return fmt.Errorf("%w: cannot use both icon_emoji and icon_url", ErrInvalidConfig)
	}

	// Timeout validation
	if cfg.Timeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive", ErrInvalidConfig)
	}

	// Retry validation
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("%w: max retries cannot be negative", ErrInvalidConfig)
	}

	if cfg.RetryDelay <= 0 {
		return fmt.Errorf("%w: retry delay must be positive", ErrInvalidConfig)
	}

	// Rate limit validation
	if cfg.RateLimit < 0 {
		return fmt.Errorf("%w: rate limit cannot be negative", ErrInvalidConfig)
	}

	// Circuit breaker validation
	if cfg.CircuitThreshold <= 0 {
		return fmt.Errorf("%w: circuit threshold must be positive", ErrInvalidConfig)
	}

	// Message size validation
	if cfg.MaxMessageSize <= 0 || cfg.MaxMessageSize > 40000 {
		return fmt.Errorf("%w: max message size must be between 1 and 40000", ErrInvalidConfig)
	}

	return nil
}

// Reset clears the global instance (for testing)
func Reset() {
	if defaultService != nil {
		_ = defaultService.Shutdown(context.Background())
	}
	defaultService = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}

// Slack returns the global slack service instance
func Slack() *Service {
	if defaultService == nil {
		_ = Init() // Initialize with defaults if needed
	}
	return defaultService
}

// Health checks if the Slack webhook is accessible
func Health() error {
	if defaultService == nil {
		return ErrNotInitialized
	}
	return defaultService.Health(context.Background())
}

// Health performs a health check without sending a message
func (s *Service) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.isShuttingDown {
		return fmt.Errorf("service is shutting down")
	}

	// Check circuit breaker state
	if s.circuitBreaker != nil && s.circuitBreaker.State() == CircuitOpen {
		return fmt.Errorf("circuit breaker is open")
	}

	// Perform a lightweight check - validate webhook URL is reachable
	// We'll do a HEAD request to the webhook domain
	u, err := url.Parse(s.webhookURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Create a health check request
	healthURL := fmt.Sprintf("https://%s", u.Host)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Use a shorter timeout for health checks
	healthClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := healthClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	// Any response means the host is reachable
	return nil
}

// Ping sends a test message to verify webhook connectivity
func (s *Service) Ping(ctx context.Context) error {
	_, err := s.SendWithContext(ctx, "🏓 Ping", nil)
	return err
}

// PingWithOptions sends a test message with custom options
func (s *Service) PingWithOptions(ctx context.Context, opts *MessageOptions) error {
	_, err := s.SendWithContext(ctx, "🏓 Ping", opts)
	return err
}

// Shutdown gracefully shuts down the service
func (s *Service) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.isShuttingDown {
		s.mu.Unlock()
		return nil
	}
	s.isShuttingDown = true
	close(s.shutdown)
	s.mu.Unlock()

	// Wait for ongoing operations with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if s.logger != nil {
			s.logger.Info("Slack service shutdown complete")
		}
		return nil
	case <-ctx.Done():
		if s.logger != nil {
			s.logger.Warn("Slack service shutdown timeout")
		}
		return ctx.Err()
	}
}

// GetStats returns service statistics
func (s *Service) GetStats() *Stats {
	if s.metrics == nil {
		return nil
	}
	stats := s.metrics.GetStats()
	return &stats
}

// SetDefaultChannel sets the default channel for all messages sent by this service
func (s *Service) SetDefaultChannel(channel string) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultOpts.Channel = channel
	return s
}

// SetDefaultUsername sets the default username for all messages sent by this service
func (s *Service) SetDefaultUsername(username string) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultOpts.Username = username
	return s
}

// SetDefaultIcon sets the default icon emoji for all messages sent by this service
func (s *Service) SetDefaultIcon(iconEmoji string) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultOpts.IconEmoji = iconEmoji
	s.defaultOpts.IconURL = "" // Clear URL when setting emoji
	return s
}

// SetDefaultIconURL sets the default icon URL for all messages sent by this service
func (s *Service) SetDefaultIconURL(iconURL string) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultOpts.IconURL = iconURL
	s.defaultOpts.IconEmoji = "" // Clear emoji when setting URL
	return s
}

// SetDebug enables or disables debug logging
func (s *Service) SetDebug(debug bool) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.debug = debug
	return s
}

// SendInfo sends an informational message to Slack
func (s *Service) SendInfo(message string) (string, error) {
	return s.SendInfoWithContext(context.Background(), message)
}

// SendInfoWithContext sends an informational message to Slack with context
func (s *Service) SendInfoWithContext(ctx context.Context, message string) (string, error) {
	formattedMessage := fmt.Sprintf("ℹ️ %s", message)
	return s.SendWithContext(ctx, formattedMessage, nil)
}

// SendInfoWithOptions sends an informational message to Slack with custom options
func (s *Service) SendInfoWithOptions(message string, opts *MessageOptions) (string, error) {
	return s.SendInfoWithOptionsContext(context.Background(), message, opts)
}

// SendInfoWithOptionsContext sends an informational message with context and options
func (s *Service) SendInfoWithOptionsContext(ctx context.Context, message string, opts *MessageOptions) (string, error) {
	formattedMessage := fmt.Sprintf("ℹ️ %s", message)
	return s.SendWithContext(ctx, formattedMessage, opts)
}

// SendWarning sends a warning message to Slack
func (s *Service) SendWarning(message string) (string, error) {
	return s.SendWarningWithContext(context.Background(), message)
}

// SendWarningWithContext sends a warning message to Slack with context
func (s *Service) SendWarningWithContext(ctx context.Context, message string) (string, error) {
	formattedMessage := fmt.Sprintf("⚠️ %s", message)
	return s.SendWithContext(ctx, formattedMessage, nil)
}

// SendWarningWithOptions sends a warning message to Slack with custom options
func (s *Service) SendWarningWithOptions(message string, opts *MessageOptions) (string, error) {
	return s.SendWarningWithOptionsContext(context.Background(), message, opts)
}

// SendWarningWithOptionsContext sends a warning message with context and options
func (s *Service) SendWarningWithOptionsContext(ctx context.Context, message string, opts *MessageOptions) (string, error) {
	formattedMessage := fmt.Sprintf("⚠️ %s", message)
	return s.SendWithContext(ctx, formattedMessage, opts)
}

// SendAlert sends an alert message to Slack
func (s *Service) SendAlert(message string) (string, error) {
	return s.SendAlertWithContext(context.Background(), message)
}

// SendAlertWithContext sends an alert message to Slack with context
func (s *Service) SendAlertWithContext(ctx context.Context, message string) (string, error) {
	formattedMessage := fmt.Sprintf("🚨 *Alert*\n%s", message)
	return s.SendWithContext(ctx, formattedMessage, nil)
}

// SendAlertWithOptions sends an alert message to Slack with custom options
func (s *Service) SendAlertWithOptions(message string, opts *MessageOptions) (string, error) {
	return s.SendAlertWithOptionsContext(context.Background(), message, opts)
}

// SendAlertWithOptionsContext sends an alert message with context and options
func (s *Service) SendAlertWithOptionsContext(ctx context.Context, message string, opts *MessageOptions) (string, error) {
	formattedMessage := fmt.Sprintf("🚨 *Alert*\n%s", message)
	return s.SendWithContext(ctx, formattedMessage, opts)
}

// SendError sends an error message to Slack with proper redaction
func (s *Service) SendError(err error) (string, error) {
	return s.SendErrorWithContext(context.Background(), err)
}

// SendErrorWithContext sends an error message to Slack with context
func (s *Service) SendErrorWithContext(ctx context.Context, err error) (string, error) {
	if err == nil {
		return "", nil
	}

	// Redact sensitive information if enabled
	errorMsg := err.Error()
	if s.redactErrors {
		errorMsg = s.sanitizeErrorMessage(errorMsg)
	}

	formattedMessage := fmt.Sprintf("❌ *Error*\n```\n%v\n```", errorMsg)
	return s.SendWithContext(ctx, formattedMessage, nil)
}

// sanitizeErrorMessage removes sensitive information from error messages
func (s *Service) sanitizeErrorMessage(msg string) string {
	// Remove potential secrets/tokens
	patterns := []string{
		"token", "secret", "password", "key", "credential",
		"auth", "bearer", "api_key", "access_token",
	}

	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(msg), pattern) {
			// Replace the value after the pattern
			msg = sanitizePattern(msg, pattern)
		}
	}

	return msg
}

// sanitizePattern replaces sensitive values in a message
func sanitizePattern(msg, pattern string) string {
	lower := strings.ToLower(msg)
	idx := strings.Index(lower, pattern)
	if idx == -1 {
		return msg
	}

	// Find the value after the pattern (usually after '=' or ':')
	valueStart := idx + len(pattern)
	for valueStart < len(msg) && (msg[valueStart] == ' ' || msg[valueStart] == '=' || msg[valueStart] == ':') {
		valueStart++
	}

	if valueStart >= len(msg) {
		return msg
	}

	// Find the end of the value
	valueEnd := valueStart
	for valueEnd < len(msg) && msg[valueEnd] != ' ' && msg[valueEnd] != ',' && msg[valueEnd] != '\n' {
		valueEnd++
	}

	// Replace the value with REDACTED
	return msg[:valueStart] + "REDACTED" + msg[valueEnd:]
}

// SendSuccess sends a success message to Slack
func (s *Service) SendSuccess(message string) (string, error) {
	return s.SendSuccessWithContext(context.Background(), message)
}

// SendSuccessWithContext sends a success message to Slack with context
func (s *Service) SendSuccessWithContext(ctx context.Context, message string) (string, error) {
	formattedMessage := fmt.Sprintf("✅ %s", message)
	return s.SendWithContext(ctx, formattedMessage, nil)
}

// Send sends a raw message to Slack (deprecated: use SendWithContext)
func (s *Service) Send(message string, opts *MessageOptions) (string, error) {
	return s.SendWithContext(context.Background(), message, opts)
}

// SendWithContext sends a raw message to Slack with context support
func (s *Service) SendWithContext(ctx context.Context, message string, opts *MessageOptions) (string, error) {
	// Check shutdown state
	s.mu.RLock()
	if s.isShuttingDown {
		s.mu.RUnlock()
		return "", fmt.Errorf("service is shutting down")
	}
	s.mu.RUnlock()

	// Track operation
	s.wg.Add(1)
	defer s.wg.Done()

	// Validate and sanitize input
	if err := s.validateInput(message); err != nil {
		if s.metrics != nil {
			s.metrics.RecordFailure(err)
		}
		return "", err
	}

	if s.sanitizeInput {
		message = s.sanitizeMessage(message)
	}

	// Apply rate limiting
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Wait(ctx); err != nil {
			if s.metrics != nil {
				s.metrics.RecordRateLimit()
			}
			return "", fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Create the message payload
	msg := s.buildMessage(message, opts)

	// Marshal the message to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.RecordMessage()
	}

	// Send with retry and circuit breaker
	start := time.Now()
	resp, err := s.sendWithRetry(ctx, payload)

	if s.metrics != nil {
		if err == nil {
			s.metrics.RecordSuccess(time.Since(start))
		} else {
			s.metrics.RecordFailure(err)
		}
	}

	return resp, err
}

// validateInput validates the input message
func (s *Service) validateInput(message string) error {
	if len(message) == 0 {
		return fmt.Errorf("%w: message cannot be empty", ErrInvalidInput)
	}

	if len(message) > s.maxMessageSize {
		return fmt.Errorf("%w: message size %d exceeds limit %d", ErrInputTooLarge, len(message), s.maxMessageSize)
	}

	return nil
}

// sanitizeMessage sanitizes the input message
func (s *Service) sanitizeMessage(message string) string {
	// HTML escape to prevent injection
	message = html.EscapeString(message)

	// Unescape common safe characters for readability
	message = strings.ReplaceAll(message, "&lt;", "<")
	message = strings.ReplaceAll(message, "&gt;", ">")
	message = strings.ReplaceAll(message, "&#39;", "'")
	message = strings.ReplaceAll(message, "&quot;", "\"")

	return message
}

// SendRichMessage sends a rich message with blocks and attachments
func (s *Service) SendRichMessage(ctx context.Context, msg *RichMessage) (string, error) {
	// Check shutdown state
	s.mu.RLock()
	if s.isShuttingDown {
		s.mu.RUnlock()
		return "", fmt.Errorf("service is shutting down")
	}
	s.mu.RUnlock()

	// Track operation
	s.wg.Add(1)
	defer s.wg.Done()

	// Apply defaults if not set
	if msg.Username == "" && s.defaultOpts.Username != "" {
		msg.Username = s.defaultOpts.Username
	}
	if msg.Channel == "" && s.defaultOpts.Channel != "" {
		msg.Channel = s.defaultOpts.Channel
	}
	if msg.IconEmoji == "" && s.defaultOpts.IconEmoji != "" {
		msg.IconEmoji = s.defaultOpts.IconEmoji
	}
	if msg.IconURL == "" && s.defaultOpts.IconURL != "" {
		msg.IconURL = s.defaultOpts.IconURL
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal rich message: %w", err)
	}

	// Validate payload size
	if len(payload) > s.maxMessageSize {
		return "", fmt.Errorf("%w: payload size %d exceeds limit %d", ErrInputTooLarge, len(payload), s.maxMessageSize)
	}

	// Apply rate limiting
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Wait(ctx); err != nil {
			if s.metrics != nil {
				s.metrics.RecordRateLimit()
			}
			return "", fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.RecordMessage()
	}

	// Send with retry and circuit breaker
	start := time.Now()
	resp, err := s.sendWithRetry(ctx, payload)

	if s.metrics != nil {
		if err == nil {
			s.metrics.RecordSuccess(time.Since(start))
		} else {
			s.metrics.RecordFailure(err)
		}
	}

	return resp, err
}

// SendBatch sends multiple messages in parallel
func (s *Service) SendBatch(ctx context.Context, messages []string, opts *MessageOptions) []BatchResult {
	results := make([]BatchResult, len(messages))
	var wg sync.WaitGroup

	for i, msg := range messages {
		wg.Add(1)
		go func(index int, message string) {
			defer wg.Done()
			resp, err := s.SendWithContext(ctx, message, opts)
			results[index] = BatchResult{
				Index:    index,
				Response: resp,
				Error:    err,
			}
		}(i, msg)
	}

	wg.Wait()
	return results
}

// SendRichBatch sends multiple rich messages in parallel
func (s *Service) SendRichBatch(ctx context.Context, messages []*RichMessage) []BatchResult {
	results := make([]BatchResult, len(messages))
	var wg sync.WaitGroup

	for i, msg := range messages {
		wg.Add(1)
		go func(index int, message *RichMessage) {
			defer wg.Done()
			resp, err := s.SendRichMessage(ctx, message)
			results[index] = BatchResult{
				Index:    index,
				Response: resp,
				Error:    err,
			}
		}(i, msg)
	}

	wg.Wait()
	return results
}

// buildMessage constructs a message with defaults applied
func (s *Service) buildMessage(text string, opts *MessageOptions) Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg := Message{
		Text: text,
	}

	// Apply default options
	if s.defaultOpts.Channel != "" {
		msg.Channel = s.defaultOpts.Channel
	}
	if s.defaultOpts.Username != "" {
		msg.Username = s.defaultOpts.Username
	}
	if s.defaultOpts.IconEmoji != "" {
		msg.IconEmoji = s.defaultOpts.IconEmoji
	}
	if s.defaultOpts.IconURL != "" {
		msg.IconURL = s.defaultOpts.IconURL
	}

	// Override with provided options if any
	if opts != nil {
		if opts.Channel != "" {
			msg.Channel = opts.Channel
		}
		if opts.Username != "" {
			msg.Username = opts.Username
		}
		if opts.IconEmoji != "" {
			msg.IconEmoji = opts.IconEmoji
		}
		if opts.IconURL != "" {
			msg.IconURL = opts.IconURL
		}
	}

	return msg
}

// sendWithRetry sends the request with exponential backoff retry and jitter
func (s *Service) sendWithRetry(ctx context.Context, payload []byte) (string, error) {
	var lastErr error
	delay := s.retryDelay

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with optional jitter
			actualDelay := delay
			if s.retryJitter {
				// Add up to 25% jitter
				jitter := time.Duration(rand.Float64() * float64(delay) * 0.25) //nolint:gosec // jitter doesn't require crypto strength
				actualDelay = delay + jitter
			}

			// Wait before retry with exponential backoff
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
			case <-time.After(actualDelay):
				// Double the delay for next attempt, up to max
				delay = time.Duration(math.Min(float64(delay*2), float64(s.retryMaxDelay)))
			}

			if s.logger != nil {
				s.logger.Debug("Retry attempt %d/%d after error: %v", attempt, s.maxRetries, lastErr)
			}
		}

		// Use circuit breaker if available
		if s.circuitBreaker != nil {
			resp, err := s.executeWithCircuitBreaker(ctx, payload)
			if err == nil {
				return resp, nil
			}
			lastErr = err
		} else {
			resp, err := s.doRequest(ctx, payload)
			if err == nil {
				return resp, nil
			}
			lastErr = err
		}

		// Check if error is retryable
		if !isRetryableError(lastErr) {
			return "", lastErr
		}
	}

	return "", fmt.Errorf("%w: %w", ErrMaxRetriesExceeded, lastErr)
}

// executeWithCircuitBreaker executes the request with circuit breaker protection
func (s *Service) executeWithCircuitBreaker(ctx context.Context, payload []byte) (string, error) {
	var resp string
	var err error

	cbErr := s.circuitBreaker.Execute(ctx, func() error {
		resp, err = s.doRequest(ctx, payload)
		return err
	})

	if cbErr != nil {
		if errors.Is(cbErr, ErrCircuitOpen) && s.metrics != nil {
			s.metrics.RecordCircuitOpen()
		}
		return "", cbErr
	}

	return resp, err
}

// doRequest performs the actual HTTP request
func (s *Service) doRequest(ctx context.Context, payload []byte) (string, error) {
	if s.requestLogger != nil {
		s.requestLogger.LogRequest(ctx, string(payload))
	}

	// Create the request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Beaver-Kit-Slack/1.0")

	// Send the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrWebhookFailed, err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if s.requestLogger != nil {
		s.requestLogger.LogResponse(ctx, resp.StatusCode, string(body))
	}

	// Check for rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return string(body), ErrRateLimited
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("%w: status=%d, body=%s", ErrInvalidResponse, resp.StatusCode, body)
	}

	// Check Slack's response
	bodyStr := string(body)
	if bodyStr != "ok" && !strings.Contains(bodyStr, "\"ok\":true") {
		return bodyStr, fmt.Errorf("%w: %s", ErrInvalidResponse, bodyStr)
	}

	return bodyStr, nil
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	// Don't retry circuit breaker open errors
	if errors.Is(err, ErrCircuitOpen) {
		return false
	}

	// Retry on rate limiting
	if errors.Is(err, ErrRateLimited) {
		return true
	}

	// Retry on webhook failures (network issues)
	if errors.Is(err, ErrWebhookFailed) {
		return true
	}

	// Check for temporary network errors
	errStr := err.Error()
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "i/o timeout") {
		return true
	}

	return false
}
