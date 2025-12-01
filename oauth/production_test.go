package oauth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gobeaver/beaver-kit/oauth"
)

func TestTokenBucketRateLimiter(t *testing.T) {
	config := oauth.RateLimiterConfig{
		Rate:      5,
		Interval:  1 * time.Second,
		BurstSize: 10,
	}

	limiter := oauth.NewTokenBucketLimiter(config)
	ctx := context.Background()
	key := "test-key"

	// Test burst capacity
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed within burst size", i+1)
		}
	}

	// 11th request should be denied
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("Request should be denied after burst capacity is exhausted")
	}

	// Get status
	status, err := limiter.GetStatus(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	if status.Remaining > 0 {
		t.Errorf("Expected 0 remaining tokens, got %d", status.Remaining)
	}

	// Reset and test again
	if err := limiter.Reset(ctx, key); err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Request should be allowed after reset")
	}
}

func TestSlidingWindowRateLimiter(t *testing.T) {
	config := oauth.RateLimiterConfig{
		Rate:     3,
		Interval: 100 * time.Millisecond,
	}

	limiter := oauth.NewSlidingWindowLimiter(config)
	ctx := context.Background()
	key := "test-key"

	// Make 3 requests (should all be allowed)
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("4th request should be denied")
	}

	// Wait for window to slide
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Request should be allowed after window slides")
	}
}

func TestCircuitBreaker(t *testing.T) {
	config := oauth.CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}

	breaker := oauth.NewDefaultCircuitBreaker(config)
	ctx := context.Background()

	// Test successful calls
	for i := 0; i < 5; i++ {
		err := breaker.Call(ctx, func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Successful call %d failed: %v", i+1, err)
		}
	}

	if breaker.GetState() != oauth.StateClosed {
		t.Errorf("Circuit should be closed, got %s", breaker.GetState())
	}

	// Test failures to open circuit
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		err := breaker.Call(ctx, func() error {
			return testErr
		})
		if !errors.Is(err, testErr) {
			t.Errorf("Expected test error, got %v", err)
		}
	}

	if breaker.GetState() != oauth.StateOpen {
		t.Errorf("Circuit should be open after %d failures, got %s", 3, breaker.GetState())
	}

	// Calls should fail immediately when open
	err := breaker.Call(ctx, func() error {
		return nil
	})
	if !errors.Is(err, oauth.ErrCircuitOpen) {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should transition to half-open
	successCount := 0
	for i := 0; i < 2; i++ {
		err := breaker.Call(ctx, func() error {
			successCount++
			return nil
		})
		if err != nil {
			t.Errorf("Half-open call %d failed: %v", i+1, err)
		}
	}

	// Circuit should be closed after success threshold
	if breaker.GetState() != oauth.StateClosed {
		t.Errorf("Circuit should be closed after success threshold, got %s", breaker.GetState())
	}

	// Get stats
	stats := breaker.GetStats()
	if stats.TotalSuccesses != int64(5+successCount) {
		t.Errorf("Expected %d total successes, got %d", 5+successCount, stats.TotalSuccesses)
	}
	if stats.TotalFailures != 3 {
		t.Errorf("Expected 3 total failures, got %d", stats.TotalFailures)
	}
}

func TestSecurityHeaders(t *testing.T) {
	middleware := oauth.NewMiddleware(oauth.MiddlewareConfig{
		EnableSecurityHeaders: true,
		EnableHSTS:            true,
		HSTSMaxAge:            31536000,
	})

	handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check security headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expected := range expectedHeaders {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("Expected %s header to be %q, got %q", header, expected, got)
		}
	}

	// CSP should be present
	if csp := w.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("Content-Security-Policy header should be present")
	}
}

func TestCORSMiddleware(t *testing.T) {
	middleware := oauth.NewMiddleware(oauth.MiddlewareConfig{
		EnableCORS:       true,
		AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	})

	handler := middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for preflight, got %d", w.Code)
	}

	// Check CORS headers
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "https://example.com" {
		t.Errorf("Expected origin header to be https://example.com, got %s", origin)
	}

	if credentials := w.Header().Get("Access-Control-Allow-Credentials"); credentials != "true" {
		t.Error("Access-Control-Allow-Credentials should be true")
	}

	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("Access-Control-Allow-Methods should be present")
	}

	// Test actual request
	req = httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "https://app.example.com" {
		t.Errorf("Expected origin header to be https://app.example.com, got %s", origin)
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := oauth.NewDefaultMetricsCollector()

	// Record some metrics
	collector.RecordAuthRequest("google", true, 100*time.Millisecond)
	collector.RecordAuthRequest("google", false, 200*time.Millisecond)
	collector.RecordTokenExchange("github", true, 150*time.Millisecond)
	collector.RecordTokenRefresh("google", true, 50*time.Millisecond)
	collector.RecordUserInfoRequest("github", false, 300*time.Millisecond)
	collector.RecordRateLimitHit("user:123")
	collector.RecordError("google", "exchange", "network_error")

	// Get metrics
	metrics := collector.GetMetrics()

	// Verify counters
	if metrics.AuthRequests.Total != 2 {
		t.Errorf("Expected 2 total auth requests, got %d", metrics.AuthRequests.Total)
	}
	if metrics.AuthRequests.Success != 1 {
		t.Errorf("Expected 1 successful auth request, got %d", metrics.AuthRequests.Success)
	}
	if metrics.AuthRequests.Failed != 1 {
		t.Errorf("Expected 1 failed auth request, got %d", metrics.AuthRequests.Failed)
	}

	if metrics.TokenExchanges.Total != 1 {
		t.Errorf("Expected 1 token exchange, got %d", metrics.TokenExchanges.Total)
	}

	if metrics.RateLimitHits.Total != 1 {
		t.Errorf("Expected 1 rate limit hit, got %d", metrics.RateLimitHits.Total)
	}

	// Check provider metrics
	googleMetrics, exists := metrics.ProviderMetrics["google"]
	if !exists {
		t.Error("Google provider metrics should exist")
	} else {
		if googleMetrics.AuthRequests.Total != 2 {
			t.Errorf("Expected 2 Google auth requests, got %d", googleMetrics.AuthRequests.Total)
		}
		if googleMetrics.Errors != 1 {
			t.Errorf("Expected 1 Google error, got %d", googleMetrics.Errors)
		}
	}

	// Test reset
	collector.Reset()
	metrics = collector.GetMetrics()
	if metrics.AuthRequests.Total != 0 {
		t.Error("Metrics should be reset")
	}
}

func TestHealthChecker(t *testing.T) {
	// Create mock components
	tokenStore := oauth.NewMemoryTokenStore(1 * time.Hour)
	tokenManager := oauth.NewAdvancedTokenManager(oauth.TokenManagerConfig{
		Store: tokenStore,
	})
	rateLimiter := oauth.NewTokenBucketLimiter(oauth.RateLimiterConfig{
		Rate:      100,
		Interval:  1 * time.Minute,
		BurstSize: 200,
	})
	metricsCollector := oauth.NewDefaultMetricsCollector()

	// Create multi-provider service
	multiService, _ := oauth.NewMultiProviderService(oauth.MultiProviderConfig{
		PKCEEnabled:    true,
		SessionTimeout: 5 * time.Minute,
	})

	// Register a test provider
	googleProvider := oauth.NewGoogle(oauth.ProviderConfig{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost/callback",
	})
	_ = multiService.RegisterProvider("google", googleProvider)

	// Create health checker
	healthChecker := oauth.NewDefaultHealthChecker(oauth.HealthCheckerConfig{
		Version:          "1.0.0",
		MultiService:     multiService,
		TokenManager:     tokenManager,
		RateLimiter:      rateLimiter,
		MetricsCollector: metricsCollector,
	})

	// Register custom check
	healthChecker.RegisterCheck("database", func(ctx context.Context) error {
		// Simulate database check
		return nil
	})

	// Perform health check
	ctx := context.Background()
	health, err := healthChecker.Check(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Verify health status
	if health.Status != oauth.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	if health.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", health.Version)
	}

	// Check individual components
	expectedChecks := []string{"session_store", "token_store", "provider_google", "rate_limiter", "database"}
	for _, checkName := range expectedChecks {
		if check, exists := health.Checks[checkName]; !exists {
			t.Errorf("Expected check %s to exist", checkName)
		} else if check.Status != oauth.HealthStatusHealthy {
			t.Errorf("Expected %s to be healthy, got %s", checkName, check.Status)
		}
	}

	// Check specific component
	result, err := healthChecker.CheckComponent(ctx, "rate_limiter")
	if err != nil {
		t.Fatalf("Failed to check rate limiter: %v", err)
	}
	if result.Status != oauth.HealthStatusHealthy {
		t.Errorf("Rate limiter should be healthy, got %s", result.Status)
	}
}

func TestHealthHandlers(t *testing.T) {
	// Create a simple health checker
	healthChecker := oauth.NewDefaultHealthChecker(oauth.HealthCheckerConfig{
		Version: "1.0.0",
	})

	handler := oauth.NewHealthHandler(healthChecker)

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.HandleHealth(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", w.Code)
	}

	// Test liveness endpoint
	req = httptest.NewRequest("GET", "/health/live", nil)
	w = httptest.NewRecorder()
	handler.HandleLiveness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Liveness check should return 200, got %d", w.Code)
	}

	// Test readiness endpoint
	req = httptest.NewRequest("GET", "/health/ready", nil)
	w = httptest.NewRecorder()
	handler.HandleReadiness(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Readiness check should return 200 or 503, got %d", w.Code)
	}
}
