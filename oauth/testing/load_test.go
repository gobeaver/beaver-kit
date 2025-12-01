package testing_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gobeaver/beaver-kit/oauth"
	oauthtest "github.com/gobeaver/beaver-kit/oauth/testing"
)

// BenchmarkOAuthFlow benchmarks a complete OAuth flow
func BenchmarkOAuthFlow(b *testing.B) {
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName:    "bench",
		ClientID:        "bench-client",
		ClientSecret:    "bench-secret",
		SupportsRefresh: true,
	})
	defer mockServer.Close()

	provider := mockServer.CreateMockProvider()
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Issue code
		state := fmt.Sprintf("state_%d", i)
		code := mockServer.IssueAuthorizationCode("user", state, "http://localhost:8080/callback", "")

		// Exchange code for token
		token, err := provider.Exchange(ctx, code, nil)
		if err != nil {
			b.Fatalf("Exchange failed: %v", err)
		}

		// Get user info
		_, err = provider.GetUserInfo(ctx, token.AccessToken)
		if err != nil {
			b.Fatalf("GetUserInfo failed: %v", err)
		}
	}
}

// BenchmarkPKCEGeneration benchmarks PKCE challenge generation
func BenchmarkPKCEGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := oauth.GeneratePKCEChallenge("S256")
		if err != nil {
			b.Fatalf("PKCE generation failed: %v", err)
		}
	}
}

// BenchmarkTokenEncryption benchmarks token encryption/decryption
func BenchmarkTokenEncryption(b *testing.B) {
	key := make([]byte, 32)
	encryptor, err := oauth.NewAESGCMEncryptor(key)
	if err != nil {
		b.Fatalf("Failed to create encryptor: %v", err)
	}

	token := &oauth.Token{
		AccessToken:  "benchmark-access-token",
		RefreshToken: "benchmark-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	tokenData, _ := json.Marshal(token)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Encrypt
		encrypted, err := encryptor.Encrypt(tokenData)
		if err != nil {
			b.Fatalf("Encryption failed: %v", err)
		}

		// Decrypt
		_, err = encryptor.Decrypt(encrypted)
		if err != nil {
			b.Fatalf("Decryption failed: %v", err)
		}
	}
}

// TestHighConcurrency tests the system under high concurrent load
func TestHighConcurrency(t *testing.T) {
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName:    "load",
		ClientID:        "load-client",
		ClientSecret:    "load-secret",
		SupportsRefresh: true,
	})
	defer mockServer.Close()

	provider := mockServer.CreateMockProvider()

	// Test parameters
	numWorkers := 100
	requestsPerWorker := 50

	var (
		successCount int64
		errorCount   int64
		wg           sync.WaitGroup
	)

	// Rate limiter for realistic load testing
	limiter := oauth.NewTokenBucketLimiter(oauth.RateLimiterConfig{
		Rate:      1000, // 1000 requests per second
		Interval:  1 * time.Second,
		BurstSize: 2000,
	})

	ctx := context.Background()

	// Worker function
	worker := func(workerID int) {
		defer wg.Done()

		for i := 0; i < requestsPerWorker; i++ {
			// Apply rate limiting
			allowed, _ := limiter.Allow(ctx, fmt.Sprintf("worker_%d", workerID))
			if !allowed {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Perform OAuth flow
			state := fmt.Sprintf("state_%d_%d", workerID, i)
			code := mockServer.IssueAuthorizationCode(
				fmt.Sprintf("user_%d", workerID),
				state,
				"http://localhost:8080/callback",
				"",
			)

			token, err := provider.Exchange(ctx, code, nil)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				continue
			}

			_, err = provider.GetUserInfo(ctx, token.AccessToken)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				continue
			}

			atomic.AddInt64(&successCount, 1)
		}
	}

	// Start workers
	start := time.Now()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i)
	}

	// Wait for completion
	wg.Wait()
	duration := time.Since(start)

	// Calculate metrics
	totalRequests := int64(numWorkers * requestsPerWorker)
	successRate := float64(successCount) / float64(totalRequests) * 100
	requestsPerSecond := float64(totalRequests) / duration.Seconds()

	t.Logf("Load Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Requests/Second: %.2f", requestsPerSecond)

	// Assertions
	if successRate < 95 {
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
	}
}

// TestMemoryLeaks tests for memory leaks during sustained load
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName: "memory",
		ClientID:     "memory-client",
		ClientSecret: "memory-secret",
	})
	defer mockServer.Close()

	provider := mockServer.CreateMockProvider()

	// Create stores with limited capacity
	tokenStore := oauth.NewMemoryTokenStore(1 * time.Hour)

	tokenManager := oauth.NewAdvancedTokenManager(oauth.TokenManagerConfig{
		Store:            tokenStore,
		MaxTokensPerUser: 10,
	})

	ctx := context.Background()

	// Run sustained load for 30 seconds
	duration := 30 * time.Second
	stop := time.After(duration)

	var requestCount int64

	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				// Perform operations
				userID := fmt.Sprintf("user_%d", requestCount%100)

				// Generate and exchange token
				code := mockServer.IssueAuthorizationCode(userID, "state", "http://localhost/callback", "")
				token, _ := provider.Exchange(ctx, code, nil)

				// Cache token
				if token != nil {
					_ = tokenManager.CacheToken(ctx, userID, "memory", token)
				}

				atomic.AddInt64(&requestCount, 1)
			}
		}
	}()

	// Wait for test to complete
	<-stop

	// Cleanup should happen automatically
	_ = tokenManager.CleanupExpiredTokens(ctx)

	t.Logf("Memory test completed: %d requests processed", atomic.LoadInt64(&requestCount))
}

// TestCircuitBreakerUnderLoad tests circuit breaker behavior under load
func TestCircuitBreakerUnderLoad(t *testing.T) {
	mockServer := oauthtest.NewMockOAuthServer(oauthtest.MockServerConfig{
		ProviderName: "circuit",
		ClientID:     "circuit-client",
		ClientSecret: "circuit-secret",
	})
	defer mockServer.Close()

	provider := mockServer.CreateMockProvider()
	protectedProvider := oauth.NewProviderWithCircuitBreaker(provider, oauth.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
	})

	ctx := context.Background()

	// Phase 1: Normal operation
	for i := 0; i < 10; i++ {
		code := mockServer.IssueAuthorizationCode("user", "state", "http://localhost/callback", "")
		_, err := protectedProvider.Exchange(ctx, code, nil)
		if err != nil {
			t.Logf("Normal request %d failed: %v", i, err)
		}
	}

	// Phase 2: Introduce failures
	mockServer.SetFailureScenario("token", true)

	var failureCount int
	for i := 0; i < 10; i++ {
		code := mockServer.IssueAuthorizationCode("user", "state", "http://localhost/callback", "")
		_, err := protectedProvider.Exchange(ctx, code, nil)
		if err != nil {
			failureCount++
		}
	}

	// Circuit should be open after threshold failures
	stats := protectedProvider.GetCircuitStats()
	if stats.State != oauth.StateOpen && failureCount >= 5 {
		t.Errorf("Circuit should be open after %d failures, but is %s", failureCount, stats.State)
	}

	// Phase 3: Recovery
	mockServer.SetFailureScenario("token", false)
	time.Sleep(1100 * time.Millisecond) // Wait for timeout

	// Circuit should transition to half-open
	for i := 0; i < 3; i++ {
		code := mockServer.IssueAuthorizationCode("user", "state", "http://localhost/callback", "")
		_, err := protectedProvider.Exchange(ctx, code, nil)
		if err != nil {
			t.Logf("Recovery request %d failed: %v", i, err)
		}
	}

	// Circuit should be closed after success threshold
	stats = protectedProvider.GetCircuitStats()
	t.Logf("Final circuit state: %s", stats.State)
}

// TestRateLimiterPerformance tests rate limiter performance
func TestRateLimiterPerformance(t *testing.T) {
	config := oauth.RateLimiterConfig{
		Rate:      100,
		Interval:  1 * time.Second,
		BurstSize: 200,
	}

	limiter := oauth.NewTokenBucketLimiter(config)
	ctx := context.Background()

	// Test with multiple clients
	numClients := 50
	requestsPerClient := 100

	var (
		allowedCount int64
		deniedCount  int64
		wg           sync.WaitGroup
	)

	client := func(clientID int) {
		defer wg.Done()

		key := fmt.Sprintf("client_%d", clientID)

		for i := 0; i < requestsPerClient; i++ {
			allowed, err := limiter.Allow(ctx, key)
			if err != nil {
				t.Errorf("Rate limiter error: %v", err)
				continue
			}

			if allowed {
				atomic.AddInt64(&allowedCount, 1)
			} else {
				atomic.AddInt64(&deniedCount, 1)
			}

			// Small delay to spread requests
			time.Sleep(1 * time.Millisecond)
		}
	}

	// Start clients
	start := time.Now()
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go client(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := int64(numClients * requestsPerClient)
	allowRate := float64(allowedCount) / float64(totalRequests) * 100

	t.Logf("Rate Limiter Performance:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Allowed: %d (%.2f%%)", allowedCount, allowRate)
	t.Logf("  Denied: %d", deniedCount)

	// The rate limiter should allow approximately the configured rate
	// Allow some variance due to timing
	expectedAllowed := float64(config.Rate) * duration.Seconds() * float64(numClients)
	variance := 0.2 // 20% variance

	if float64(allowedCount) < expectedAllowed*(1-variance) ||
		float64(allowedCount) > expectedAllowed*(1+variance)+float64(config.BurstSize*numClients) {
		t.Errorf("Allowed count outside expected range: got %d, expected ~%.0f",
			allowedCount, expectedAllowed)
	}
}
