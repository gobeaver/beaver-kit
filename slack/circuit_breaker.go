package slack

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker protects against cascading failures
type CircuitBreaker struct {
	maxFailures    int
	timeout        time.Duration
	maxRequests    int
	
	state          CircuitState
	failures       int
	successCount   int
	lastFailTime   time.Time
	mu             sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, timeout time.Duration, maxRequests int) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		maxRequests: maxRequests,
		state:       CircuitClosed,
	}
}

// Allow checks if a request is allowed through the circuit
func (cb *CircuitBreaker) Allow() error {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitOpen:
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			if cb.state == CircuitOpen {
				cb.state = CircuitHalfOpen
				cb.successCount = 0
			}
			cb.mu.Unlock()
			cb.mu.RLock()
		} else {
			return ErrCircuitOpen
		}
	case CircuitHalfOpen:
		if cb.successCount >= cb.maxRequests {
			return ErrCircuitOpen
		}
	}
	
	return nil
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.maxRequests {
			cb.state = CircuitClosed
			cb.failures = 0
		}
	case CircuitClosed:
		cb.failures = 0
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.maxFailures {
			cb.state = CircuitOpen
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failures = 0
	cb.successCount = 0
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.Allow(); err != nil {
		return err
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// CircuitBreakerError represents a circuit breaker error
type CircuitBreakerError struct {
	State CircuitState
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}

var (
	// ErrCircuitOpen is returned when the circuit is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
)