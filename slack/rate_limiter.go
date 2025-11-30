package slack

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality
type RateLimiter interface {
	Allow(ctx context.Context) error
	Wait(ctx context.Context) error
}

// TokenBucketLimiter implements token bucket rate limiting
type TokenBucketLimiter struct {
	tokens    float64
	capacity  float64
	rate      float64
	lastCheck time.Time
	mu        sync.Mutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(rate int, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		tokens:    float64(burst),
		capacity:  float64(burst),
		rate:      float64(rate),
		lastCheck: time.Now(),
	}
}

// Allow checks if a request is allowed
func (l *TokenBucketLimiter) Allow(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastCheck).Seconds()
	l.lastCheck = now

	// Add tokens based on elapsed time
	l.tokens = min(l.capacity, l.tokens+elapsed*l.rate)

	if l.tokens < 1 {
		return ErrRateLimited
	}

	l.tokens--
	return nil
}

// Wait waits until a request is allowed
func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := l.Allow(ctx); err == nil {
				return nil
			}
			// Wait a bit before retrying
			time.Sleep(time.Duration(1000/l.rate) * time.Millisecond)
		}
	}
}

// SlidingWindowLimiter implements sliding window rate limiting
type SlidingWindowLimiter struct {
	requests []time.Time
	limit    int
	window   time.Duration
	mu       sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		requests: make([]time.Time, 0, limit),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request is allowed
func (l *SlidingWindowLimiter) Allow(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0, l.limit)
	for _, t := range l.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	l.requests = validRequests

	if len(l.requests) >= l.limit {
		return ErrRateLimited
	}

	l.requests = append(l.requests, now)
	return nil
}

// Wait waits until a request is allowed
func (l *SlidingWindowLimiter) Wait(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := l.Allow(ctx); err == nil {
				return nil
			}
			// Wait a bit before retrying
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// NoOpLimiter is a rate limiter that allows all requests
type NoOpLimiter struct{}

// Allow always returns nil
func (l *NoOpLimiter) Allow(ctx context.Context) error {
	return nil
}

// Wait always returns nil immediately
func (l *NoOpLimiter) Wait(ctx context.Context) error {
	return nil
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimitedError wraps an error with rate limit information
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retry after %v", e.RetryAfter)
}
