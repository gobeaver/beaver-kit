package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestService_SendInfo(t *testing.T) {
	defer Reset() // Clean up after test

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Test with Init
	testConfig := Config{
		WebhookURL: server.URL,
		Timeout:    5 * time.Second,
	}

	err := Init(testConfig)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Get service instance and test
	service := Slack()
	resp, err := service.SendInfo("Test info message")
	if err != nil {
		t.Fatalf("Failed to send info message: %v", err)
	}
	if resp != "ok" {
		t.Errorf("Expected response 'ok', got '%s'", resp)
	}
}

func TestService_SendWarning(t *testing.T) {
	defer Reset() // Clean up after test

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Create service directly with New
	service, err := New(Config{
		WebhookURL: server.URL,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test sending a warning message
	resp, err := service.SendWarning("Test warning message")
	if err != nil {
		t.Fatalf("Failed to send warning message: %v", err)
	}
	if resp != "ok" {
		t.Errorf("Expected response 'ok', got '%s'", resp)
	}
}

func TestService_SendAlert(t *testing.T) {
	defer Reset() // Clean up after test

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Create service
	service, err := New(Config{
		WebhookURL: server.URL,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test sending an alert message
	resp, err := service.SendAlert("Test alert message")
	if err != nil {
		t.Fatalf("Failed to send alert message: %v", err)
	}
	if resp != "ok" {
		t.Errorf("Expected response 'ok', got '%s'", resp)
	}
}

func TestService_WithOptions(t *testing.T) {
	defer Reset() // Clean up after test

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Create service with config options
	service, err := New(Config{
		WebhookURL: server.URL,
		Channel:    "#testing",
		Username:   "TestBot",
		IconEmoji:  ":robot:",
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test sending a message with options
	resp, err := service.SendInfo("Test message with options")
	if err != nil {
		t.Fatalf("Failed to send message with options: %v", err)
	}
	if resp != "ok" {
		t.Errorf("Expected response 'ok', got '%s'", resp)
	}
}

func TestService_SetDefaultOptions(t *testing.T) {
	defer Reset() // Clean up after test

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Create service and set default options
	service, err := New(Config{
		WebhookURL: server.URL,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	service.SetDefaultChannel("#testing")
	service.SetDefaultUsername("TestBot")
	service.SetDefaultIcon(":robot:")

	// Test sending a message with default options
	resp, err := service.SendInfo("Test message with default options")
	if err != nil {
		t.Fatalf("Failed to send message with default options: %v", err)
	}
	if resp != "ok" {
		t.Errorf("Expected response 'ok', got '%s'", resp)
	}
}

func TestService_ErrorHandling(t *testing.T) {
	defer Reset() // Clean up after test

	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_payload"))
	}))
	defer server.Close()

	// Create service
	service, err := New(Config{
		WebhookURL: server.URL,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test error handling
	_, err = service.SendInfo("Test error handling")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Empty webhook URL",
			config: Config{
				WebhookURL: "",
				Timeout:    5 * time.Second,
			},
			wantErr: true,
			errMsg:  "webhook URL required",
		},
		{
			name: "Both icon emoji and URL",
			config: Config{
				WebhookURL: "https://hooks.slack.com/test",
				IconEmoji:  ":robot:",
				IconURL:    "https://example.com/icon.png",
				Timeout:    5 * time.Second,
			},
			wantErr: true,
			errMsg:  "cannot use both icon_emoji and icon_url",
		},
		{
			name: "Invalid timeout",
			config: Config{
				WebhookURL: "https://hooks.slack.com/test",
				Timeout:    0,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "Valid config",
			config: Config{
				WebhookURL: "https://hooks.slack.com/test",
				Timeout:    5 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
