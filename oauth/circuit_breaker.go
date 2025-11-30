package oauth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreaker states
const (
	StateClosed   = "closed"
	StateOpen     = "open"
	StateHalfOpen = "half_open"
)

// CircuitBreaker errors
var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// CircuitBreaker protects external API calls from cascading failures
type CircuitBreaker interface {
	// Call executes the function if the circuit allows it
	Call(ctx context.Context, fn func() error) error
	// GetState returns the current state of the circuit
	GetState() string
	// GetStats returns circuit breaker statistics
	GetStats() *CircuitStats
	// Reset resets the circuit breaker
	Reset()
}

// CircuitStats represents circuit breaker statistics
type CircuitStats struct {
	State                string    `json:"state"`
	Requests             int64     `json:"requests"`
	TotalSuccesses       int64     `json:"total_successes"`
	TotalFailures        int64     `json:"total_failures"`
	ConsecutiveSuccesses int64     `json:"consecutive_successes"`
	ConsecutiveFailures  int64     `json:"consecutive_failures"`
	LastFailureTime      time.Time `json:"last_failure_time,omitempty"`
	LastSuccessTime      time.Time `json:"last_success_time,omitempty"`
	NextRetryTime        time.Time `json:"next_retry_time,omitempty"`
}

// DefaultCircuitBreaker implements a circuit breaker with configurable thresholds
type DefaultCircuitBreaker struct {
	config CircuitBreakerConfig

	state                int32 // 0=closed, 1=open, 2=half-open
	requests             int64
	totalSuccesses       int64
	totalFailures        int64
	consecutiveSuccesses int64
	consecutiveFailures  int64
	lastFailureTime      time.Time
	lastSuccessTime      time.Time
	nextRetryTime        time.Time
	halfOpenRequests     int32

	mu sync.RWMutex
}

// CircuitBreakerConfig configures the circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes in half-open before closing
	SuccessThreshold int
	// Timeout is how long to wait before trying half-open state
	Timeout time.Duration
	// MaxHalfOpenRequests is the maximum number of requests in half-open state
	MaxHalfOpenRequests int
	// OnStateChange is called when the circuit changes state
	OnStateChange func(from, to string)
}

// NewDefaultCircuitBreaker creates a new circuit breaker with default settings
func NewDefaultCircuitBreaker(config CircuitBreakerConfig) *DefaultCircuitBreaker {
	// Set defaults
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxHalfOpenRequests <= 0 {
		config.MaxHalfOpenRequests = 1
	}

	return &DefaultCircuitBreaker{
		config: config,
		state:  0, // Start closed
	}
}

// Call executes the function if the circuit allows it
func (cb *DefaultCircuitBreaker) Call(ctx context.Context, fn func() error) error {
	state := atomic.LoadInt32(&cb.state)

	switch state {
	case 0: // Closed
		return cb.callClosed(fn)
	case 1: // Open
		return cb.callOpen(fn)
	case 2: // Half-open
		return cb.callHalfOpen(fn)
	default:
		return fmt.Errorf("unknown circuit state: %d", state)
	}
}

func (cb *DefaultCircuitBreaker) callClosed(fn func() error) error {
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.requests, 1)

	if err != nil {
		atomic.AddInt64(&cb.totalFailures, 1)
		atomic.AddInt64(&cb.consecutiveFailures, 1)
		atomic.StoreInt64(&cb.consecutiveSuccesses, 0)
		cb.lastFailureTime = time.Now()

		// Check if we should open the circuit
		if cb.consecutiveFailures >= int64(cb.config.FailureThreshold) {
			cb.setState(1) // Open
			cb.nextRetryTime = time.Now().Add(cb.config.Timeout)
		}
	} else {
		atomic.AddInt64(&cb.totalSuccesses, 1)
		atomic.AddInt64(&cb.consecutiveSuccesses, 1)
		atomic.StoreInt64(&cb.consecutiveFailures, 0)
		cb.lastSuccessTime = time.Now()
	}

	return err
}

func (cb *DefaultCircuitBreaker) callOpen(fn func() error) error {
	cb.mu.RLock()
	nextRetry := cb.nextRetryTime
	cb.mu.RUnlock()

	// Check if timeout has passed
	if time.Now().After(nextRetry) {
		// Try to move to half-open
		if atomic.CompareAndSwapInt32(&cb.state, 1, 2) {
			atomic.StoreInt32(&cb.halfOpenRequests, 0)
			if cb.config.OnStateChange != nil {
				cb.config.OnStateChange(StateOpen, StateHalfOpen)
			}
			return cb.callHalfOpen(fn)
		}
	}

	return ErrCircuitOpen
}

func (cb *DefaultCircuitBreaker) callHalfOpen(fn func() error) error {
	// Check if we've reached max half-open requests
	current := atomic.AddInt32(&cb.halfOpenRequests, 1)
	if current > int32(cb.config.MaxHalfOpenRequests) {
		atomic.AddInt32(&cb.halfOpenRequests, -1)
		return ErrTooManyRequests
	}
	defer atomic.AddInt32(&cb.halfOpenRequests, -1)

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.requests, 1)

	if err != nil {
		atomic.AddInt64(&cb.totalFailures, 1)
		atomic.StoreInt64(&cb.consecutiveSuccesses, 0)
		cb.lastFailureTime = time.Now()

		// Move back to open
		cb.setState(1) // Open
		cb.nextRetryTime = time.Now().Add(cb.config.Timeout)
	} else {
		atomic.AddInt64(&cb.totalSuccesses, 1)
		atomic.AddInt64(&cb.consecutiveSuccesses, 1)
		cb.lastSuccessTime = time.Now()

		// Check if we should close the circuit
		if cb.consecutiveSuccesses >= int64(cb.config.SuccessThreshold) {
			cb.setState(0) // Closed
			atomic.StoreInt64(&cb.consecutiveFailures, 0)
		}
	}

	return err
}

// GetState returns the current state of the circuit
func (cb *DefaultCircuitBreaker) GetState() string {
	state := atomic.LoadInt32(&cb.state)
	switch state {
	case 0:
		return StateClosed
	case 1:
		return StateOpen
	case 2:
		return StateHalfOpen
	default:
		return "unknown"
	}
}

// GetStats returns circuit breaker statistics
func (cb *DefaultCircuitBreaker) GetStats() *CircuitStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stats := &CircuitStats{
		State:                cb.GetState(),
		Requests:             atomic.LoadInt64(&cb.requests),
		TotalSuccesses:       atomic.LoadInt64(&cb.totalSuccesses),
		TotalFailures:        atomic.LoadInt64(&cb.totalFailures),
		ConsecutiveSuccesses: atomic.LoadInt64(&cb.consecutiveSuccesses),
		ConsecutiveFailures:  atomic.LoadInt64(&cb.consecutiveFailures),
		LastFailureTime:      cb.lastFailureTime,
		LastSuccessTime:      cb.lastSuccessTime,
	}

	if cb.GetState() == StateOpen {
		stats.NextRetryTime = cb.nextRetryTime
	}

	return stats
}

// Reset resets the circuit breaker
func (cb *DefaultCircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.GetState()

	atomic.StoreInt32(&cb.state, 0) // Closed
	atomic.StoreInt64(&cb.requests, 0)
	atomic.StoreInt64(&cb.totalSuccesses, 0)
	atomic.StoreInt64(&cb.totalFailures, 0)
	atomic.StoreInt64(&cb.consecutiveSuccesses, 0)
	atomic.StoreInt64(&cb.consecutiveFailures, 0)
	atomic.StoreInt32(&cb.halfOpenRequests, 0)
	cb.lastFailureTime = time.Time{}
	cb.lastSuccessTime = time.Time{}
	cb.nextRetryTime = time.Time{}

	if cb.config.OnStateChange != nil && oldState != StateClosed {
		cb.config.OnStateChange(oldState, StateClosed)
	}
}

func (cb *DefaultCircuitBreaker) setState(newState int32) {
	oldStateInt := atomic.SwapInt32(&cb.state, newState)

	if cb.config.OnStateChange != nil {
		oldState := ""
		newStateStr := ""

		switch oldStateInt {
		case 0:
			oldState = StateClosed
		case 1:
			oldState = StateOpen
		case 2:
			oldState = StateHalfOpen
		}

		switch newState {
		case 0:
			newStateStr = StateClosed
		case 1:
			newStateStr = StateOpen
		case 2:
			newStateStr = StateHalfOpen
		}

		if oldState != newStateStr {
			cb.config.OnStateChange(oldState, newStateStr)
		}
	}
}

// CircuitBreakerManager manages circuit breakers for multiple endpoints
type CircuitBreakerManager struct {
	breakers map[string]CircuitBreaker
	config   CircuitBreakerConfig
	mu       sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(config CircuitBreakerConfig) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]CircuitBreaker),
		config:   config,
	}
}

// GetBreaker returns a circuit breaker for the given key
func (m *CircuitBreakerManager) GetBreaker(key string) CircuitBreaker {
	m.mu.RLock()
	breaker, exists := m.breakers[key]
	m.mu.RUnlock()

	if exists {
		return breaker
	}

	// Create new breaker
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := m.breakers[key]; exists {
		return breaker
	}

	breaker = NewDefaultCircuitBreaker(m.config)
	m.breakers[key] = breaker

	return breaker
}

// Call executes the function using the circuit breaker for the given key
func (m *CircuitBreakerManager) Call(ctx context.Context, key string, fn func() error) error {
	breaker := m.GetBreaker(key)
	return breaker.Call(ctx, fn)
}

// GetAllStats returns statistics for all circuit breakers
func (m *CircuitBreakerManager) GetAllStats() map[string]*CircuitStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]*CircuitStats)
	for key, breaker := range m.breakers {
		stats[key] = breaker.GetStats()
	}

	return stats
}

// Reset resets a specific circuit breaker
func (m *CircuitBreakerManager) Reset(key string) {
	m.mu.RLock()
	breaker, exists := m.breakers[key]
	m.mu.RUnlock()

	if exists {
		breaker.Reset()
	}
}

// ResetAll resets all circuit breakers
func (m *CircuitBreakerManager) ResetAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, breaker := range m.breakers {
		breaker.Reset()
	}
}

// ProviderWithCircuitBreaker wraps a Provider with circuit breaker protection
type ProviderWithCircuitBreaker struct {
	provider Provider
	breaker  CircuitBreaker
}

// NewProviderWithCircuitBreaker creates a new provider with circuit breaker
func NewProviderWithCircuitBreaker(provider Provider, config CircuitBreakerConfig) *ProviderWithCircuitBreaker {
	return &ProviderWithCircuitBreaker{
		provider: provider,
		breaker:  NewDefaultCircuitBreaker(config),
	}
}

// Name returns the provider name
func (p *ProviderWithCircuitBreaker) Name() string {
	return p.provider.Name()
}

// GetAuthURL generates an authorization URL
func (p *ProviderWithCircuitBreaker) GetAuthURL(state string, pkce *PKCEChallenge) string {
	// Auth URL generation doesn't need circuit breaker
	return p.provider.GetAuthURL(state, pkce)
}

// Exchange exchanges an authorization code for tokens
func (p *ProviderWithCircuitBreaker) Exchange(ctx context.Context, code string, pkce *PKCEChallenge) (*Token, error) {
	var token *Token
	var exchangeErr error

	err := p.breaker.Call(ctx, func() error {
		token, exchangeErr = p.provider.Exchange(ctx, code, pkce)
		return exchangeErr
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

// RefreshToken refreshes an access token
func (p *ProviderWithCircuitBreaker) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	var token *Token
	var refreshErr error

	err := p.breaker.Call(ctx, func() error {
		token, refreshErr = p.provider.RefreshToken(ctx, refreshToken)
		return refreshErr
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

// GetUserInfo retrieves user information
func (p *ProviderWithCircuitBreaker) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	var userInfo *UserInfo
	var infoErr error

	err := p.breaker.Call(ctx, func() error {
		userInfo, infoErr = p.provider.GetUserInfo(ctx, accessToken)
		return infoErr
	})

	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

// RevokeToken revokes an access or refresh token
func (p *ProviderWithCircuitBreaker) RevokeToken(ctx context.Context, token string) error {
	return p.breaker.Call(ctx, func() error {
		return p.provider.RevokeToken(ctx, token)
	})
}

// ValidateConfig validates the provider configuration
func (p *ProviderWithCircuitBreaker) ValidateConfig() error {
	return p.provider.ValidateConfig()
}

// SupportsPKCE returns true if the provider supports PKCE
func (p *ProviderWithCircuitBreaker) SupportsPKCE() bool {
	return p.provider.SupportsPKCE()
}

// SupportsRefresh returns true if the provider supports token refresh
func (p *ProviderWithCircuitBreaker) SupportsRefresh() bool {
	return p.provider.SupportsRefresh()
}

// GetCircuitStats returns circuit breaker statistics
func (p *ProviderWithCircuitBreaker) GetCircuitStats() *CircuitStats {
	return p.breaker.GetStats()
}
