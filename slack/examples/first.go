package examples

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gobeaver/beaver-kit/slack"
)

// ZeroConfigExample shows how to use slack with environment variables
func ZeroConfigExample() {
	// Make sure BEAVER_SLACK_WEBHOOK_URL is set in environment
	// Then simply call Init() with no arguments
	if err := slack.Init(); err != nil {
		log.Fatalf("Failed to initialize slack: %v", err)
	}

	// Get the global service instance
	service := slack.Slack()

	// Send messages
	resp, err := service.SendInfo("System started successfully")
	if err != nil {
		log.Fatalf("Failed to send info message: %v", err)
	}
	fmt.Printf("Response: %s\n", resp)
}

// DirectConfigExample shows how to use slack with direct configuration
func DirectConfigExample() {
	// Initialize with direct config
	err := slack.Init(slack.Config{
		WebhookURL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
		Channel:    "#monitoring",
		Username:   "MonitorBot",
		IconEmoji:  ":robot_face:",
		Timeout:    15 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Get service and send messages
	service := slack.Slack()

	// Send an informational message
	resp, err := service.SendInfo("Daily system health check passed")
	if err != nil {
		log.Fatalf("Failed to send info message: %v", err)
	}
	fmt.Printf("Response: %s\n", resp)

	// Send a warning message
	resp, err = service.SendWarning("High CPU usage detected")
	if err != nil {
		log.Fatalf("Failed to send warning message: %v", err)
	}
	fmt.Printf("Response: %s\n", resp)

	// Send an alert message
	resp, err = service.SendAlert("Database connection failed")
	if err != nil {
		log.Fatalf("Failed to send alert message: %v", err)
	}
	fmt.Printf("Response: %s\n", resp)
}

// MultipleInstancesExample shows how to use multiple slack instances
func MultipleInstancesExample() {
	// Create first instance for production alerts
	prodService, err := slack.New(slack.Config{
		WebhookURL: os.Getenv("PROD_SLACK_WEBHOOK_URL"),
		Channel:    "#prod-alerts",
		Username:   "ProdBot",
		IconEmoji:  ":warning:",
		Timeout:    10 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create prod service: %v", err)
	}

	// Create second instance for dev notifications
	devService, err := slack.New(slack.Config{
		WebhookURL: os.Getenv("DEV_SLACK_WEBHOOK_URL"),
		Channel:    "#dev-notifications",
		Username:   "DevBot",
		IconEmoji:  ":computer:",
		Timeout:    10 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create dev service: %v", err)
	}

	// Send to production channel
	_, err = prodService.SendAlert("Production deployment failed")
	if err != nil {
		log.Printf("Failed to send prod alert: %v", err)
	}

	// Send to dev channel
	_, err = devService.SendInfo("Development build completed")
	if err != nil {
		log.Printf("Failed to send dev info: %v", err)
	}
}

// CustomOptionsExample shows how to override default options
func CustomOptionsExample() {
	// Initialize with config
	err := slack.Init(slack.Config{
		WebhookURL: os.Getenv("BEAVER_SLACK_WEBHOOK_URL"),
		Channel:    "#general",
		Username:   "BeaverBot",
		Timeout:    10 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	service := slack.Slack()

	// Send with default options
	_, err = service.SendInfo("Using default channel and username")
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}

	// Override channel for specific message
	customOpts := &slack.MessageOptions{
		Channel: "#alerts",
	}
	_, err = service.SendAlertWithOptions("Critical issue detected", customOpts)
	if err != nil {
		log.Printf("Failed to send alert: %v", err)
	}

	// Override multiple options
	specialOpts := &slack.MessageOptions{
		Channel:   "#dev-alerts",
		Username:  "DeployBot",
		IconEmoji: ":rocket:",
	}
	_, err = service.SendInfoWithOptions("New version deployed", specialOpts)
	if err != nil {
		log.Printf("Failed to send deployment message: %v", err)
	}
}
