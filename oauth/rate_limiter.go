package oauth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality for OAuth operations
type RateLimiter interface {
	// Allow checks if a request should be allowed
	Allow(ctx context.Context, key string) (bool, error)
	// AllowN checks if n requests should be allowed
	AllowN(ctx context.Context, key string, n int) (bool, error)
	// Reset resets the rate limit for a key
	Reset(ctx context.Context, key string) error
	// GetStatus returns the current rate limit status for a key
	GetStatus(ctx context.Context, key string) (*RateLimitStatus, error)
}

// RateLimitStatus provides information about current rate limit state
type RateLimitStatus struct {
	Limit      int           `json:"limit"`
	Remaining  int           `json:"remaining"`
	Reset      time.Time     `json:"reset"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// TokenBucketLimiter implements token bucket algorithm for rate limiting
type TokenBucketLimiter struct {
	buckets map[string]*bucket
	mu      sync.RWMutex
	config  RateLimiterConfig
}

// RateLimiterConfig configures the rate limiter
type RateLimiterConfig struct {
	Rate            int           // Number of tokens per interval
	Interval        time.Duration // Time interval for rate
	BurstSize       int           // Maximum burst size
	MaxEntries      int           // Maximum number of tracked entries
	CleanupInterval time.Duration // Interval for cleaning up old entries
}

// bucket represents a token bucket for a single key
type bucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(config RateLimiterConfig) *TokenBucketLimiter {
	if config.Rate <= 0 {
		config.Rate = 100 // Default: 100 requests
	}
	if config.Interval <= 0 {
		config.Interval = 1 * time.Minute // Default: per minute
	}
	if config.BurstSize <= 0 {
		config.BurstSize = config.Rate * 2 // Default: 2x rate
	}
	if config.MaxEntries <= 0 {
		config.MaxEntries = 10000 // Default: 10k entries
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute // Default: 5 minutes
	}

	limiter := &TokenBucketLimiter{
		buckets: make(map[string]*bucket),
		config:  config,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Allow checks if a single request should be allowed
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks if n requests should be allowed
func (l *TokenBucketLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return false, fmt.Errorf("n must be positive")
	}

	l.mu.Lock()
	b, exists := l.buckets[key]
	if !exists {
		// Check if we've reached max entries
		if len(l.buckets) >= l.config.MaxEntries {
			l.mu.Unlock()
			return false, fmt.Errorf("rate limiter at capacity")
		}

		b = &bucket{
			tokens:     float64(l.config.BurstSize),
			lastRefill: time.Now(),
		}
		l.buckets[key] = b
	}
	l.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := elapsed.Seconds() * (float64(l.config.Rate) / l.config.Interval.Seconds())

	b.tokens = min(b.tokens+tokensToAdd, float64(l.config.BurstSize))
	b.lastRefill = now

	// Check if we have enough tokens
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true, nil
	}

	return false, nil
}

// Reset resets the rate limit for a key
func (l *TokenBucketLimiter) Reset(ctx context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.buckets, key)
	return nil
}

// GetStatus returns the current rate limit status for a key
func (l *TokenBucketLimiter) GetStatus(ctx context.Context, key string) (*RateLimitStatus, error) {
	l.mu.RLock()
	b, exists := l.buckets[key]
	l.mu.RUnlock()

	if !exists {
		return &RateLimitStatus{
			Limit:     l.config.Rate,
			Remaining: l.config.Rate,
			Reset:     time.Now().Add(l.config.Interval),
		}, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate current tokens after refill
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := elapsed.Seconds() * (float64(l.config.Rate) / l.config.Interval.Seconds())
	currentTokens := min(b.tokens+tokensToAdd, float64(l.config.BurstSize))

	// Calculate time until next token
	var retryAfter time.Duration
	if currentTokens < 1 {
		secondsPerToken := l.config.Interval.Seconds() / float64(l.config.Rate)
		retryAfter = time.Duration((1-currentTokens)*secondsPerToken) * time.Second
	}

	return &RateLimitStatus{
		Limit:      l.config.Rate,
		Remaining:  int(currentTokens),
		Reset:      now.Add(l.config.Interval),
		RetryAfter: retryAfter,
	}, nil
}

// cleanup removes old buckets periodically
func (l *TokenBucketLimiter) cleanup() {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, b := range l.buckets {
			b.mu.Lock()
			// Remove buckets that haven't been used for 2x the interval
			if now.Sub(b.lastRefill) > l.config.Interval*2 {
				delete(l.buckets, key)
			}
			b.mu.Unlock()
		}
		l.mu.Unlock()
	}
}

// SlidingWindowLimiter implements sliding window algorithm for rate limiting
type SlidingWindowLimiter struct {
	windows map[string]*window
	mu      sync.RWMutex
	config  RateLimiterConfig
}

// window represents a sliding window for a single key
type window struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(config RateLimiterConfig) *SlidingWindowLimiter {
	if config.Rate <= 0 {
		config.Rate = 100
	}
	if config.Interval <= 0 {
		config.Interval = 1 * time.Minute
	}
	if config.MaxEntries <= 0 {
		config.MaxEntries = 10000
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	limiter := &SlidingWindowLimiter{
		windows: make(map[string]*window),
		config:  config,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Allow checks if a single request should be allowed
func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks if n requests should be allowed
func (l *SlidingWindowLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return false, fmt.Errorf("n must be positive")
	}

	l.mu.Lock()
	w, exists := l.windows[key]
	if !exists {
		// Check if we've reached max entries
		if len(l.windows) >= l.config.MaxEntries {
			l.mu.Unlock()
			return false, fmt.Errorf("rate limiter at capacity")
		}

		w = &window{
			requests: make([]time.Time, 0, l.config.Rate),
		}
		l.windows[key] = w
	}
	l.mu.Unlock()

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.config.Interval)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0, len(w.requests))
	for _, req := range w.requests {
		if req.After(windowStart) {
			validRequests = append(validRequests, req)
		}
	}
	w.requests = validRequests

	// Check if adding n requests would exceed the limit
	if len(w.requests)+n > l.config.Rate {
		return false, nil
	}

	// Add the new requests
	for i := 0; i < n; i++ {
		w.requests = append(w.requests, now)
	}

	return true, nil
}

// Reset resets the rate limit for a key
func (l *SlidingWindowLimiter) Reset(ctx context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.windows, key)
	return nil
}

// GetStatus returns the current rate limit status for a key
func (l *SlidingWindowLimiter) GetStatus(ctx context.Context, key string) (*RateLimitStatus, error) {
	l.mu.RLock()
	w, exists := l.windows[key]
	l.mu.RUnlock()

	if !exists {
		return &RateLimitStatus{
			Limit:     l.config.Rate,
			Remaining: l.config.Rate,
			Reset:     time.Now().Add(l.config.Interval),
		}, nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.config.Interval)

	// Count valid requests in the current window
	validCount := 0
	var oldestRequest time.Time
	for _, req := range w.requests {
		if req.After(windowStart) {
			validCount++
			if oldestRequest.IsZero() || req.Before(oldestRequest) {
				oldestRequest = req
			}
		}
	}

	remaining := l.config.Rate - validCount

	// Calculate retry after if rate limit exceeded
	var retryAfter time.Duration
	if remaining <= 0 && !oldestRequest.IsZero() {
		retryAfter = l.config.Interval - now.Sub(oldestRequest)
	}

	return &RateLimitStatus{
		Limit:      l.config.Rate,
		Remaining:  max(0, remaining),
		Reset:      now.Add(l.config.Interval),
		RetryAfter: retryAfter,
	}, nil
}

// cleanup removes old windows periodically
func (l *SlidingWindowLimiter) cleanup() {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-l.config.Interval * 2)

		for key, w := range l.windows {
			w.mu.Lock()
			hasRecentRequests := false
			for _, req := range w.requests {
				if req.After(windowStart) {
					hasRecentRequests = true
					break
				}
			}
			if !hasRecentRequests {
				delete(l.windows, key)
			}
			w.mu.Unlock()
		}
		l.mu.Unlock()
	}
}
