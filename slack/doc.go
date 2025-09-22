// Package slack provides a production-ready Slack integration library with advanced features
// including rate limiting, circuit breakers, rich message formatting, and comprehensive monitoring.
//
// This package is designed for applications that need reliable Slack notifications with
// proper error handling, retry logic, and production hardening features. It supports both
// simple text messages and rich Block Kit formatted messages with attachments.
//
// # Quick Start
//
// Initialize the Slack service using environment variables:
//
//	import "github.com/gobeaver/beaver-kit/slack"
//
//	// Initialize with environment variables (BEAVER_SLACK_WEBHOOK_URL, etc.)
//	err := slack.Init()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send a simple message
//	response, err := slack.Slack().Send("Hello from Beaver Kit!")
//	if err != nil {
//	    log.Printf("Failed to send message: %v", err)
//	}
//
// # Custom Configuration
//
// Configure the service programmatically:
//
//	config := slack.Config{
//	    WebhookURL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
//	    Channel:    "#alerts",
//	    Username:   "AlertBot",
//	    IconEmoji:  ":robot_face:",
//	    MaxRetries: 3,
//	    Timeout:    time.Second * 30,
//	}
//
//	service, err := slack.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Environment Variables
//
// The package supports configuration via environment variables with the BEAVER_SLACK_ prefix:
//   - BEAVER_SLACK_WEBHOOK_URL: Slack webhook URL (required)
//   - BEAVER_SLACK_CHANNEL: Default channel (e.g., "#alerts")
//   - BEAVER_SLACK_USERNAME: Bot username
//   - BEAVER_SLACK_ICON_EMOJI: Bot emoji icon (e.g., ":robot_face:")
//   - BEAVER_SLACK_ICON_URL: Bot icon URL (alternative to emoji)
//   - BEAVER_SLACK_MAX_RETRIES: Maximum retry attempts (default: 3)
//   - BEAVER_SLACK_RETRY_DELAY: Initial retry delay (default: 1s)
//   - BEAVER_SLACK_TIMEOUT: HTTP timeout (default: 30s)
//   - BEAVER_SLACK_DEBUG: Enable debug logging (default: false)
//
// # Custom Prefixes
//
// Use custom prefixes to avoid environment variable conflicts:
//
//	// Initialize with custom prefix
//	service := slack.WithPrefix("MYAPP_SLACK_").New()
//
// # Production Features
//
// Advanced configuration for production environments:
//
//	config := slack.Config{
//	    WebhookURL:       "https://hooks.slack.com/services/...",
//	    RateLimit:        10,  // Requests per second
//	    RateBurst:        20,  // Burst capacity
//	    CircuitThreshold: 5,   // Failures before circuit opens
//	    CircuitTimeout:   time.Minute * 5,
//	    EnableMetrics:    true,
//	    EnableLogging:    true,
//	    LogLevel:         "info",
//	    MaxMessageSize:   40000, // Slack's limit
//	    SanitizeInput:    true,
//	    RedactErrors:     true,
//	}
//
// # Message Types
//
// Send different types of messages:
//
//	// Simple text message
//	slack.Slack().Send("Simple message")
//
//	// Alert with error formatting
//	slack.Slack().SendAlert("Critical issue detected!")
//
//	// Success message
//	slack.Slack().SendSuccess("Deployment completed successfully")
//
//	// Error with proper redaction
//	err := errors.New("database connection failed: token=secret123")
//	slack.Slack().SendError(err) // Sensitive data automatically redacted
//
//	// Info message
//	slack.Slack().SendInfo("System maintenance scheduled")
//
//	// Warning message
//	slack.Slack().SendWarning("High memory usage detected")
//
// # Rich Messages with Block Kit
//
// Create rich formatted messages using Slack's Block Kit:
//
//	// Create a rich message with blocks
//	message := &slack.RichMessage{
//	    Text: "Deployment Status",
//	    Blocks: []slack.Block{
//	        slack.NewHeaderBlock("🚀 Deployment Status"),
//	        slack.NewSectionBlock("Application deployed successfully to production", true),
//	        slack.NewDividerBlock(),
//	        slack.NewContextBlock([]slack.BlockElement{
//	            {Type: "mrkdwn", Text: &slack.TextObject{Type: "mrkdwn", Text: "*Environment:* Production"}},
//	            {Type: "mrkdwn", Text: &slack.TextObject{Type: "mrkdwn", Text: "*Version:* v1.2.3"}},
//	        }),
//	    },
//	}
//
//	response, err := service.SendRichMessage(ctx, message)
//
// # Message Options
//
// Customize individual messages:
//
//	opts := &slack.MessageOptions{
//	    Channel:   "#different-channel",
//	    Username:  "CustomBot",
//	    IconEmoji: ":warning:",
//	}
//
//	slack.Slack().SendWithOptions("Custom message", opts)
//
// # Batch Operations
//
// Send multiple messages efficiently:
//
//	messages := []string{
//	    "Message 1",
//	    "Message 2", 
//	    "Message 3",
//	}
//
//	results := service.SendBatch(ctx, messages)
//	for _, result := range results {
//	    if result.Error != nil {
//	        log.Printf("Message %d failed: %v", result.Index, result.Error)
//	    }
//	}
//
// # Error Handling
//
// The package provides specific error types:
//   - ErrWebhookNotConfigured: Webhook URL not provided
//   - ErrNotInitialized: Service not properly initialized
//   - ErrRateLimited: Rate limit exceeded
//   - ErrMessageTooLarge: Message exceeds Slack's size limit
//   - ErrWebhookFailed: HTTP request to webhook failed
//   - ErrMaxRetriesExceeded: All retry attempts failed
//
// # Rate Limiting
//
// Built-in rate limiting prevents API abuse:
//
//	// Token bucket rate limiter
//	rateLimiter := slack.NewTokenBucketLimiter(10, 20) // 10 req/sec, 20 burst
//
//	// Sliding window rate limiter
//	rateLimiter := slack.NewSlidingWindowLimiter(100, time.Minute) // 100 req/min
//
// # Circuit Breaker
//
// Automatic circuit breaker for resilient external API calls:
//
//	// Circuit opens after 5 failures, stays open for 30 seconds
//	config.CircuitThreshold = 5
//	config.CircuitTimeout = time.Second * 30
//
// # Monitoring and Metrics
//
// Built-in metrics collection:
//
//	// Enable metrics
//	config.EnableMetrics = true
//
//	// Get service statistics
//	stats := service.GetStats()
//	fmt.Printf("Messages sent: %d\n", stats.MessagesSent)
//	fmt.Printf("Success rate: %.2f%%\n", 
//	    float64(stats.MessagesSucceeded)/float64(stats.MessagesSent)*100)
//
// # Health Checks
//
// Service health monitoring:
//
//	// Check if service is healthy
//	err := service.Health(ctx)
//	if err != nil {
//	    log.Printf("Slack service unhealthy: %v", err)
//	}
//
//	// Use global health check
//	err = slack.Health()
//
// # Graceful Shutdown
//
// Properly shutdown the service:
//
//	// Shutdown with timeout
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
//	defer cancel()
//
//	err := service.Shutdown(ctx)
//	if err != nil {
//	    log.Printf("Shutdown timeout: %v", err)
//	}
//
// # Security Features
//
//   - Input sanitization to prevent injection attacks
//   - Sensitive data redaction in error messages
//   - Webhook URL validation
//   - Request/response logging with data redaction
//   - Message size limits
//
// # Integration Patterns
//
// Common integration patterns:
//
//	// Error handler middleware
//	func errorHandler(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        defer func() {
//	            if err := recover(); err != nil {
//	                slack.Slack().SendAlert(fmt.Sprintf("Panic in %s: %v", r.URL.Path, err))
//	            }
//	        }()
//	        next.ServeHTTP(w, r)
//	    })
//	}
//
//	// Background job notifications
//	func processJob(job Job) error {
//	    slack.Slack().SendInfo(fmt.Sprintf("Starting job: %s", job.ID))
//	    
//	    err := job.Process()
//	    if err != nil {
//	        slack.Slack().SendError(fmt.Errorf("Job %s failed: %w", job.ID, err))
//	        return err
//	    }
//	    
//	    slack.Slack().SendSuccess(fmt.Sprintf("Job %s completed successfully", job.ID))
//	    return nil
//	}
//
// # Best Practices
//
//   - Use appropriate message types (SendAlert, SendError, SendSuccess)
//   - Enable rate limiting in production
//   - Set up proper error handling
//   - Use rich messages for complex notifications
//   - Monitor service health and metrics
//   - Enable input sanitization for user-generated content
//   - Use batch operations for multiple messages
//   - Implement graceful shutdown
//   - Test webhook connectivity during deployment
//
// For complete examples and advanced usage patterns, see the project documentation
// and example applications.
package slack