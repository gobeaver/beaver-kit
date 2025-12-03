package filekit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// ChangeToken Implementations
// ============================================================================

// CallbackChangeToken is a ChangeToken that supports active callbacks.
// Used by drivers that have native file system events (local, memory).
type CallbackChangeToken struct {
	mu        sync.RWMutex
	changed   atomic.Bool
	callbacks []func()
}

// NewCallbackChangeToken creates a new ChangeToken that supports active callbacks.
func NewCallbackChangeToken() *CallbackChangeToken {
	return &CallbackChangeToken{}
}

func (t *CallbackChangeToken) HasChanged() bool {
	return t.changed.Load()
}

func (t *CallbackChangeToken) ActiveChangeCallbacks() bool {
	return true
}

func (t *CallbackChangeToken) RegisterChangeCallback(callback func()) (unregister func()) {
	t.mu.Lock()
	t.callbacks = append(t.callbacks, callback)
	index := len(t.callbacks) - 1
	t.mu.Unlock()

	return func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		if index < len(t.callbacks) {
			// Set to nil instead of removing to avoid index shifting
			t.callbacks[index] = nil
		}
	}
}

// SignalChange marks the token as changed and invokes all callbacks.
// This should be called by the driver when a change is detected.
func (t *CallbackChangeToken) SignalChange() {
	if t.changed.Swap(true) {
		return // Already changed
	}

	t.mu.RLock()
	callbacks := make([]func(), len(t.callbacks))
	copy(callbacks, t.callbacks)
	t.mu.RUnlock()

	for _, cb := range callbacks {
		if cb != nil {
			cb()
		}
	}
}

// ============================================================================
// Polling ChangeToken
// ============================================================================

// pollingChangeToken is a ChangeToken for backends without native events.
// It polls for changes at a specified interval.
type pollingChangeToken struct {
	mu         sync.RWMutex
	changed    atomic.Bool
	callbacks  []func()
	cancel     context.CancelFunc
	checkFunc  func() bool // Returns true if change detected
	interval   time.Duration
	lastCheck  time.Time
	started    bool
}

// PollingConfig configures a polling change token.
type PollingConfig struct {
	// Interval between polls (default: 5 seconds)
	Interval time.Duration
	// CheckFunc returns true if a change is detected
	CheckFunc func() bool
}

// NewPollingChangeToken creates a ChangeToken that polls for changes.
// The checkFunc is called periodically and should return true if a change occurred.
func NewPollingChangeToken(ctx context.Context, config PollingConfig) *pollingChangeToken {
	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(ctx)
	t := &pollingChangeToken{
		checkFunc: config.CheckFunc,
		interval:  config.Interval,
		cancel:    cancel,
		lastCheck: time.Now(),
	}

	// Start polling goroutine
	go t.poll(ctx)

	return t
}

func (t *pollingChangeToken) poll(ctx context.Context) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if t.checkFunc != nil && t.checkFunc() {
				t.signalChange()
				return // Token is now "spent"
			}
		}
	}
}

func (t *pollingChangeToken) signalChange() {
	if t.changed.Swap(true) {
		return
	}

	t.mu.RLock()
	callbacks := make([]func(), len(t.callbacks))
	copy(callbacks, t.callbacks)
	t.mu.RUnlock()

	for _, cb := range callbacks {
		if cb != nil {
			cb()
		}
	}
}

func (t *pollingChangeToken) HasChanged() bool {
	return t.changed.Load()
}

func (t *pollingChangeToken) ActiveChangeCallbacks() bool {
	// Polling tokens do support callbacks, but polling is more efficient
	// for some scenarios. Return true since we do invoke callbacks.
	return true
}

func (t *pollingChangeToken) RegisterChangeCallback(callback func()) (unregister func()) {
	t.mu.Lock()
	t.callbacks = append(t.callbacks, callback)
	index := len(t.callbacks) - 1
	t.mu.Unlock()

	return func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		if index < len(t.callbacks) {
			t.callbacks[index] = nil
		}
	}
}

// Stop stops the polling goroutine.
func (t *pollingChangeToken) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
}

// ============================================================================
// Composite ChangeToken
// ============================================================================

// CompositeChangeToken combines multiple ChangeTokens into one.
// HasChanged returns true if ANY of the underlying tokens has changed.
type CompositeChangeToken struct {
	tokens []ChangeToken
}

// NewCompositeChangeToken creates a token that combines multiple tokens.
func NewCompositeChangeToken(tokens ...ChangeToken) *CompositeChangeToken {
	return &CompositeChangeToken{tokens: tokens}
}

func (c *CompositeChangeToken) HasChanged() bool {
	for _, t := range c.tokens {
		if t.HasChanged() {
			return true
		}
	}
	return false
}

func (c *CompositeChangeToken) ActiveChangeCallbacks() bool {
	// Return true only if ALL tokens support active callbacks
	for _, t := range c.tokens {
		if !t.ActiveChangeCallbacks() {
			return false
		}
	}
	return len(c.tokens) > 0
}

func (c *CompositeChangeToken) RegisterChangeCallback(callback func()) (unregister func()) {
	var unregisters []func()

	for _, t := range c.tokens {
		unregister := t.RegisterChangeCallback(callback)
		unregisters = append(unregisters, unregister)
	}

	return func() {
		for _, u := range unregisters {
			u()
		}
	}
}

// ============================================================================
// Static/Cancelled ChangeToken
// ============================================================================

// CancelledChangeToken is a ChangeToken that is already in a "changed" state.
// Useful for signaling that watching is not supported.
type CancelledChangeToken struct{}

func (CancelledChangeToken) HasChanged() bool {
	return true
}

func (CancelledChangeToken) ActiveChangeCallbacks() bool {
	return false
}

func (CancelledChangeToken) RegisterChangeCallback(callback func()) func() {
	// Immediately invoke the callback since we're already "changed"
	callback()
	return func() {}
}

// NeverChangeToken is a ChangeToken that never changes.
// Useful for static content that will never be modified.
type NeverChangeToken struct{}

func (NeverChangeToken) HasChanged() bool {
	return false
}

func (NeverChangeToken) ActiveChangeCallbacks() bool {
	return false
}

func (NeverChangeToken) RegisterChangeCallback(callback func()) func() {
	// Never call the callback
	return func() {}
}

// ============================================================================
// Helper: ChangeToken.OnChange
// ============================================================================

// OnChange is a helper that continuously watches for changes.
// It creates new tokens when the previous one is triggered.
// Returns a cancel function to stop watching.
//
// Example:
//
//	cancel := filekit.OnChange(
//	    func() (filekit.ChangeToken, error) {
//	        return fs.(filekit.Watcher).Watch(ctx, "config.json")
//	    },
//	    func() {
//	        log.Println("Config changed, reloading...")
//	        reloadConfig()
//	    },
//	)
//	defer cancel()
func OnChange(tokenProducer func() (ChangeToken, error), changeAction func()) (cancel func()) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		for {
			token, err := tokenProducer()
			if err != nil {
				return
			}

			// Wait for change
			done := make(chan struct{})
			unregister := token.RegisterChangeCallback(func() {
				close(done)
			})

			select {
			case <-ctx.Done():
				unregister()
				return
			case <-done:
				unregister()
				changeAction()
				// Loop to create a new token
			}
		}
	}()

	return cancelFunc
}
