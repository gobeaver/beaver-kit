// Package slack provides methods to send notifications to Slack channels via webhooks.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
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
	debug         bool
	mu            sync.RWMutex
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
			cfg, defaultErr = GetConfig(config.LoadOptions{Prefix: "BEAVER_"})
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

	// Initialize
	return &Service{
		webhookURL: cfg.WebhookURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		defaultOpts:   defaultOpts,
		maxRetries:    cfg.MaxRetries,
		retryDelay:    cfg.RetryDelay,
		retryMaxDelay: cfg.RetryMaxDelay,
		debug:         cfg.Debug,
	}, nil
}

// validateConfig checks configuration validity
func validateConfig(cfg Config) error {
	if cfg.WebhookURL == "" {
		return fmt.Errorf("%w: webhook URL required", ErrInvalidConfig)
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

	return nil
}

// Reset clears the global instance (for testing)
func Reset() {
	defaultService = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}

// Slack returns the global slack service instance
func Slack() *Service {
	if defaultService == nil {
		Init() // Initialize with defaults if needed
	}
	return defaultService
}

// Health checks if the Slack webhook is accessible
func Health() error {
	if defaultService == nil {
		return ErrNotInitialized
	}
	return defaultService.Ping(context.Background())
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

// SendError sends an error message to Slack
func (s *Service) SendError(err error) (string, error) {
	return s.SendErrorWithContext(context.Background(), err)
}

// SendErrorWithContext sends an error message to Slack with context
func (s *Service) SendErrorWithContext(ctx context.Context, err error) (string, error) {
	if err == nil {
		return "", nil
	}
	formattedMessage := fmt.Sprintf("❌ *Error*\n```\n%v\n```", err)
	return s.SendWithContext(ctx, formattedMessage, nil)
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
	// Create the message payload
	msg := s.buildMessage(message, opts)

	// Marshal the message to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send with retry
	return s.sendWithRetry(ctx, payload)
}

// SendRichMessage sends a rich message with blocks and attachments
func (s *Service) SendRichMessage(ctx context.Context, msg *RichMessage) (string, error) {
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

	// Send with retry
	return s.sendWithRetry(ctx, payload)
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

// sendWithRetry sends the request with exponential backoff retry
func (s *Service) sendWithRetry(ctx context.Context, payload []byte) (string, error) {
	var lastErr error
	delay := s.retryDelay

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())
			case <-time.After(delay):
				// Double the delay for next attempt, up to max
				delay = time.Duration(math.Min(float64(delay*2), float64(s.retryMaxDelay)))
			}

			if s.debug {
				log.Printf("[SLACK] Retry attempt %d/%d after error: %v", attempt, s.maxRetries, lastErr)
			}
		}

		resp, err := s.doRequest(ctx, payload)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return "", err
		}
	}

	return "", fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// doRequest performs the actual HTTP request
func (s *Service) doRequest(ctx context.Context, payload []byte) (string, error) {
	if s.debug {
		log.Printf("[SLACK] Sending payload: %s", string(payload))
	}

	// Create the request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrWebhookFailed, err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if s.debug {
		log.Printf("[SLACK] Response status: %d, body: %s", resp.StatusCode, string(body))
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
	// Retry on rate limiting
	if err == ErrRateLimited {
		return true
	}

	// Retry on webhook failures (network issues)
	if err == ErrWebhookFailed {
		return true
	}

	// Check for temporary network errors
	if strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "no such host") {
		return true
	}

	return false
}