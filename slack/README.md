# Slack Package

The slack package provides a simple and efficient way to send notifications to Slack channels via webhooks. It follows the Beaver Kit conventions for configuration and initialization.

## Features

- 🚀 Zero-config initialization with environment variables
- 📨 Send info, warning, and alert messages with pre-formatted styles
- ⚙️ Configurable default options (channel, username, icon)
- 🔄 Support for multiple instances
- 🧪 Easy testing with Reset() function
- ⏱️ Configurable timeout for HTTP requests

## Installation

```bash
go get github.com/gobeaver/beaver-kit
```

## Configuration

The package uses environment variables with the `BEAVER_` prefix:

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `BEAVER_SLACK_WEBHOOK_URL` | Slack webhook URL (required) | - |
| `BEAVER_SLACK_CHANNEL` | Default channel for messages | - |
| `BEAVER_SLACK_USERNAME` | Default username for messages | `Beaver` |
| `BEAVER_SLACK_ICON_EMOJI` | Default emoji icon (e.g., `:robot:`) | - |
| `BEAVER_SLACK_ICON_URL` | Default icon URL (cannot be used with emoji) | - |
| `BEAVER_SLACK_TIMEOUT` | HTTP request timeout | `10s` |

## Usage

### Zero-Config Usage

Set the required environment variable and use the package immediately:

```go
package main

import (
    "log"
    "github.com/gobeaver/beaver-kit/slack"
)

func main() {
    // Requires BEAVER_SLACK_WEBHOOK_URL to be set
    if err := slack.Init(); err != nil {
        log.Fatal(err)
    }
    
    // Get the global instance
    service := slack.Slack()
    
    // Send messages
    service.SendInfo("Application started successfully")
    service.SendWarning("CPU usage is high")
    service.SendAlert("Database connection lost!")
}
```

### Direct Configuration

Initialize with explicit configuration:

```go
err := slack.Init(slack.Config{
    WebhookURL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    Channel:    "#monitoring",
    Username:   "MonitorBot",
    IconEmoji:  ":robot_face:",
    Timeout:    15 * time.Second,
})
if err != nil {
    log.Fatal(err)
}

service := slack.Slack()
service.SendInfo("System health check passed")
```

### Multiple Instances

Create separate instances for different purposes:

```go
// Production alerts instance
prodService, err := slack.New(slack.Config{
    WebhookURL: os.Getenv("PROD_WEBHOOK_URL"),
    Channel:    "#prod-alerts",
    Username:   "ProdBot",
    IconEmoji:  ":warning:",
    Timeout:    10 * time.Second,
})

// Development notifications instance
devService, err := slack.New(slack.Config{
    WebhookURL: os.Getenv("DEV_WEBHOOK_URL"),
    Channel:    "#dev-notifications",
    Username:   "DevBot",
    IconEmoji:  ":computer:",
    Timeout:    10 * time.Second,
})

// Use different instances
prodService.SendAlert("Production deployment failed")
devService.SendInfo("Development build completed")
```

### Custom Options per Message

Override default options for specific messages:

```go
service := slack.Slack()

// Send to a different channel
opts := &slack.MessageOptions{
    Channel: "#urgent-alerts",
}
service.SendAlertWithOptions("Critical issue detected", opts)

// Custom appearance
specialOpts := &slack.MessageOptions{
    Channel:   "#releases",
    Username:  "ReleaseBot",
    IconEmoji: ":rocket:",
}
service.SendInfoWithOptions("v2.0.0 deployed to production", specialOpts)
```

### Method Chaining

Configure service options using method chaining:

```go
service := slack.Slack()
service.
    SetDefaultChannel("#notifications").
    SetDefaultUsername("AppBot").
    SetDefaultIcon(":bell:")

service.SendInfo("Configuration updated")
```

## API Reference

### Configuration Type

```go
type Config struct {
    WebhookURL string        // Slack webhook URL (required)
    Channel    string        // Default channel for messages
    Username   string        // Default username for messages
    IconEmoji  string        // Default emoji icon
    IconURL    string        // Default icon URL
    Timeout    time.Duration // HTTP request timeout
}
```

### Message Options

```go
type MessageOptions struct {
    Channel   string // Target channel (overrides default)
    Username  string // Bot username (overrides default)
    IconEmoji string // Emoji icon (overrides default)
    IconURL   string // Icon URL (overrides default)
}
```

### Initialization Functions

- `Init(configs ...Config) error` - Initialize global instance
- `GetConfig() (*Config, error)` - Get config from environment
- `New(cfg Config) (*Service, error)` - Create new instance
- `Slack() *Service` - Get global instance
- `Reset()` - Reset global instance (for testing)

### Message Functions

- `SendInfo(message string) (string, error)` - Send info message with ℹ️ formatting
- `SendInfoWithOptions(message string, opts *MessageOptions) (string, error)`
- `SendWarning(message string) (string, error)` - Send warning message with ⚠️ formatting
- `SendWarningWithOptions(message string, opts *MessageOptions) (string, error)`
- `SendAlert(message string) (string, error)` - Send alert message with ‼️ formatting
- `SendAlertWithOptions(message string, opts *MessageOptions) (string, error)`
- `Send(message string, opts *MessageOptions) (string, error)` - Send raw message

### Configuration Methods

- `SetDefaultChannel(channel string) *Service` - Set default channel
- `SetDefaultUsername(username string) *Service` - Set default username
- `SetDefaultIcon(iconEmoji string) *Service` - Set default emoji icon
- `SetDefaultIconURL(iconURL string) *Service` - Set default icon URL

## Testing

Use the `Reset()` function to clean up between tests:

```go
func TestMyFunction(t *testing.T) {
    defer slack.Reset() // Clean up after test
    
    // Mock configuration
    err := slack.Init(slack.Config{
        WebhookURL: "https://example.com/webhook",
        Timeout:    5 * time.Second,
    })
    if err != nil {
        t.Fatal(err)
    }
    
    // Test your code that uses slack
}
```

## Error Handling

The package provides clear error messages for common issues:

```go
service, err := slack.New(slack.Config{
    WebhookURL: "", // This will error
})
// Error: invalid config: invalid configuration: webhook URL required

service, err := slack.New(slack.Config{
    WebhookURL: "https://example.com",
    IconEmoji:  ":robot:",
    IconURL:    "https://example.com/icon.png", // This will error
})
// Error: invalid config: invalid configuration: cannot use both icon_emoji and icon_url
```

## Message Formatting

Messages are automatically formatted based on their type:

- **Info**: `ℹ️ Your message ℹ️`
- **Warning**: `⚠️ Your message ⚠️`
- **Alert**: `‼️ Alert ‼️\nYour message`

## Examples

See the [examples](examples/) directory for complete working examples:

- `ZeroConfigExample` - Using environment variables
- `DirectConfigExample` - Direct configuration
- `MultipleInstancesExample` - Managing multiple Slack webhooks
- `CustomOptionsExample` - Overriding options per message

## Best Practices

1. **Use environment variables** for configuration in production
2. **Set meaningful defaults** at the service level to avoid repetition
3. **Use appropriate message types** (Info, Warning, Alert) for clarity
4. **Handle errors** from send operations, especially in critical paths
5. **Use Reset() in tests** to ensure clean state between test cases
6. **Validate webhook URLs** before deploying to production

## Getting a Slack Webhook URL

1. Go to your Slack workspace's App Directory
2. Search for "Incoming WebHooks" and add it
3. Choose a channel and click "Add Incoming WebHooks Integration"
4. Copy the Webhook URL
5. Set it as `BEAVER_SLACK_WEBHOOK_URL` environment variable

## License

This package is part of the Beaver Kit project. See the main project repository for license information.