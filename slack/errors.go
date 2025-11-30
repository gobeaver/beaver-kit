package slack

import "errors"

// Standard errors for the slack package
var (
	// Configuration errors
	ErrInvalidConfig        = errors.New("invalid configuration")
	ErrWebhookNotConfigured = errors.New("webhook URL not configured")
	ErrNotInitialized       = errors.New("service not initialized")

	// API errors
	ErrRateLimited     = errors.New("slack rate limit exceeded")
	ErrInvalidResponse = errors.New("invalid response from Slack")
	ErrMessageTooLarge = errors.New("message exceeds Slack's size limit")
	ErrInvalidChannel  = errors.New("invalid channel format")
	ErrWebhookFailed   = errors.New("webhook request failed")

	// Retry errors
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")
	ErrContextCanceled    = errors.New("context canceled")

	// Security errors
	ErrInputTooLarge      = errors.New("input exceeds maximum size")
	ErrInvalidInput       = errors.New("invalid input detected")
	ErrSanitizationFailed = errors.New("input sanitization failed")
)
