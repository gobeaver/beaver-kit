// Package slack provides methods to send notifications to Slack channels via webhooks.
package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/gobeaver/beaver-kit/config"
)

// Global instance management
var (
	defaultService *Service
	defaultOnce    sync.Once
	defaultErr     error
)

// Standard errors for the package
var (
	ErrInvalidConfig = errors.New("invalid configuration")
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
	webhookURL  string
	httpClient  *http.Client
	defaultOpts *MessageOptions
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
		defaultOpts: defaultOpts,
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

// SetDefaultChannel sets the default channel for all messages sent by this service
func (s *Service) SetDefaultChannel(channel string) *Service {
	s.defaultOpts.Channel = channel
	return s
}

// SetDefaultUsername sets the default username for all messages sent by this service
func (s *Service) SetDefaultUsername(username string) *Service {
	s.defaultOpts.Username = username
	return s
}

// SetDefaultIcon sets the default icon emoji for all messages sent by this service
func (s *Service) SetDefaultIcon(iconEmoji string) *Service {
	s.defaultOpts.IconEmoji = iconEmoji
	return s
}

// SetDefaultIconURL sets the default icon URL for all messages sent by this service
func (s *Service) SetDefaultIconURL(iconURL string) *Service {
	s.defaultOpts.IconURL = iconURL
	return s
}

// SendInfo sends an informational message to Slack
func (s *Service) SendInfo(message string) (string, error) {
	formattedMessage := fmt.Sprintf("ℹ️ %s ℹ️", message)
	return s.Send(formattedMessage, nil)
}

// SendInfoWithOptions sends an informational message to Slack with custom options
func (s *Service) SendInfoWithOptions(message string, opts *MessageOptions) (string, error) {
	formattedMessage := fmt.Sprintf("ℹ️ %s ℹ️", message)
	return s.Send(formattedMessage, opts)
}

// SendWarning sends a warning message to Slack
func (s *Service) SendWarning(message string) (string, error) {
	formattedMessage := fmt.Sprintf("⚠️ %s ⚠️", message)
	return s.Send(formattedMessage, nil)
}

// SendWarningWithOptions sends a warning message to Slack with custom options
func (s *Service) SendWarningWithOptions(message string, opts *MessageOptions) (string, error) {
	formattedMessage := fmt.Sprintf("⚠️ %s ⚠️", message)
	return s.Send(formattedMessage, opts)
}

// SendAlert sends an alert message to Slack
func (s *Service) SendAlert(message string) (string, error) {
	formattedMessage := fmt.Sprintf("‼️ Alert ‼️ \n%s", message)
	return s.Send(formattedMessage, nil)
}

// SendAlertWithOptions sends an alert message to Slack with custom options
func (s *Service) SendAlertWithOptions(message string, opts *MessageOptions) (string, error) {
	formattedMessage := fmt.Sprintf("‼️ Alert ‼️ \n%s", message)
	return s.Send(formattedMessage, opts)
}

// Send sends a raw message to Slack
func (s *Service) Send(message string, opts *MessageOptions) (string, error) {
	// Create the message payload
	msg := Message{
		Text: message,
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

	// Marshal the message to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create the request
	req, err := http.NewRequest(http.MethodPost, s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("slack API returned error: %s", body)
	}

	return string(body), nil
}
