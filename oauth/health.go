package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Status    HealthStatus           `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version,omitempty"`
	Uptime    time.Duration          `json:"uptime,omitempty"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
	Metrics   *HealthMetrics         `json:"metrics,omitempty"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status      HealthStatus  `json:"status"`
	Message     string        `json:"message,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	LastChecked time.Time     `json:"last_checked"`
}

// HealthMetrics represents health-related metrics
type HealthMetrics struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	ErrorRate         float64 `json:"error_rate"`
	AverageLatency    int64   `json:"average_latency_ms"`
	ActiveSessions    int64   `json:"active_sessions"`
	CachedTokens      int64   `json:"cached_tokens"`
}

// HealthChecker performs health checks
type HealthChecker interface {
	// Check performs a health check
	Check(ctx context.Context) (*HealthCheck, error)
	// CheckComponent checks a specific component
	CheckComponent(ctx context.Context, component string) (*CheckResult, error)
	// RegisterCheck registers a custom health check
	RegisterCheck(name string, check func(context.Context) error)
}

// DefaultHealthChecker implements HealthChecker
type DefaultHealthChecker struct {
	service          *Service
	multiService     *MultiProviderService
	tokenManager     TokenManager
	rateLimiter      RateLimiter
	metricsCollector MetricsCollector
	customChecks     map[string]func(context.Context) error
	startTime        time.Time
	version          string
	mu               sync.RWMutex
}

// HealthCheckerConfig configures the health checker
type HealthCheckerConfig struct {
	Version          string
	Service          *Service
	MultiService     *MultiProviderService
	TokenManager     TokenManager
	RateLimiter      RateLimiter
	MetricsCollector MetricsCollector
}

// NewDefaultHealthChecker creates a new health checker
func NewDefaultHealthChecker(config HealthCheckerConfig) *DefaultHealthChecker {
	return &DefaultHealthChecker{
		service:          config.Service,
		multiService:     config.MultiService,
		tokenManager:     config.TokenManager,
		rateLimiter:      config.RateLimiter,
		metricsCollector: config.MetricsCollector,
		customChecks:     make(map[string]func(context.Context) error),
		startTime:        time.Now(),
		version:          config.Version,
	}
}

// Check performs a comprehensive health check
func (h *DefaultHealthChecker) Check(ctx context.Context) (*HealthCheck, error) {
	health := &HealthCheck{
		Status:    HealthStatusHealthy,
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime),
		Checks:    make(map[string]CheckResult),
	}

	// Check session store
	if h.service != nil || h.multiService != nil {
		health.Checks["session_store"] = h.checkSessionStore(ctx)
	}

	// Check token store
	if h.tokenManager != nil {
		health.Checks["token_store"] = h.checkTokenStore(ctx)
	}

	// Check providers
	if h.multiService != nil {
		for _, provider := range h.multiService.ListProviders() {
			key := fmt.Sprintf("provider_%s", provider)
			health.Checks[key] = h.checkProvider(ctx, provider)
		}
	}

	// Check rate limiter
	if h.rateLimiter != nil {
		health.Checks["rate_limiter"] = h.checkRateLimiter(ctx)
	}

	// Run custom checks
	h.mu.RLock()
	customChecks := make(map[string]func(context.Context) error)
	for name, check := range h.customChecks {
		customChecks[name] = check
	}
	h.mu.RUnlock()

	for name, check := range customChecks {
		health.Checks[name] = h.runCheck(ctx, check)
	}

	// Determine overall status
	degradedCount := 0
	unhealthyCount := 0
	for _, check := range health.Checks {
		switch check.Status {
		case HealthStatusDegraded:
			degradedCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		}
	}

	if unhealthyCount > 0 {
		health.Status = HealthStatusUnhealthy
	} else if degradedCount > 0 {
		health.Status = HealthStatusDegraded
	}

	// Add metrics if available
	if h.metricsCollector != nil {
		health.Metrics = h.getHealthMetrics()
	}

	return health, nil
}

// CheckComponent checks a specific component
func (h *DefaultHealthChecker) CheckComponent(ctx context.Context, component string) (*CheckResult, error) {
	switch component {
	case "session_store":
		result := h.checkSessionStore(ctx)
		return &result, nil
	case "token_store":
		result := h.checkTokenStore(ctx)
		return &result, nil
	case "rate_limiter":
		result := h.checkRateLimiter(ctx)
		return &result, nil
	default:
		// Check if it's a provider
		if h.multiService != nil {
			providers := h.multiService.ListProviders()
			for _, provider := range providers {
				if component == fmt.Sprintf("provider_%s", provider) {
					result := h.checkProvider(ctx, provider)
					return &result, nil
				}
			}
		}

		// Check custom checks
		h.mu.RLock()
		check, exists := h.customChecks[component]
		h.mu.RUnlock()

		if exists {
			result := h.runCheck(ctx, check)
			return &result, nil
		}

		return nil, fmt.Errorf("unknown component: %s", component)
	}
}

// RegisterCheck registers a custom health check
func (h *DefaultHealthChecker) RegisterCheck(name string, check func(context.Context) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.customChecks[name] = check
}

// Helper methods

func (h *DefaultHealthChecker) checkSessionStore(ctx context.Context) CheckResult {
	start := time.Now()

	// Try to store and retrieve a test session
	testKey := fmt.Sprintf("health_check_%d", time.Now().UnixNano())
	testData := &SessionData{
		State:     testKey,
		Provider:  "health_check",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Minute),
	}

	var store SessionStore
	if h.service != nil {
		store = h.service.sessions
	} else if h.multiService != nil {
		store = h.multiService.sessions
	}

	if store == nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Message:     "Session store not configured",
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Store test session
	if err := store.Store(ctx, testKey, testData); err != nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Error:       err.Error(),
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Retrieve test session
	if _, err := store.Retrieve(ctx, testKey); err != nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Error:       err.Error(),
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Clean up
	store.Delete(ctx, testKey)

	return CheckResult{
		Status:      HealthStatusHealthy,
		Message:     "Session store is operational",
		Duration:    time.Since(start),
		LastChecked: time.Now(),
	}
}

func (h *DefaultHealthChecker) checkTokenStore(ctx context.Context) CheckResult {
	start := time.Now()

	// Try to get token stats
	if h.tokenManager != nil {
		stats := h.tokenManager.GetTokenStats()

		status := HealthStatusHealthy
		message := fmt.Sprintf("Token store operational: %d total tokens, %d active",
			stats.TotalTokens, stats.ActiveTokens)

		// Check for high expired token ratio
		if stats.TotalTokens > 0 {
			expiredRatio := float64(stats.ExpiredTokens) / float64(stats.TotalTokens)
			if expiredRatio > 0.5 {
				status = HealthStatusDegraded
				message = fmt.Sprintf("High expired token ratio: %.2f%%", expiredRatio*100)
			}
		}

		return CheckResult{
			Status:      status,
			Message:     message,
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	return CheckResult{
		Status:      HealthStatusUnhealthy,
		Message:     "Token manager not configured",
		Duration:    time.Since(start),
		LastChecked: time.Now(),
	}
}

func (h *DefaultHealthChecker) checkProvider(ctx context.Context, providerName string) CheckResult {
	start := time.Now()

	if h.multiService == nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Message:     "Multi-provider service not configured",
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	provider, err := h.multiService.GetProvider(providerName)
	if err != nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Error:       err.Error(),
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Check if provider has valid config
	if err := provider.ValidateConfig(); err != nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Error:       fmt.Sprintf("Invalid configuration: %v", err),
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Check circuit breaker status if available
	if pwcb, ok := provider.(*ProviderWithCircuitBreaker); ok {
		stats := pwcb.GetCircuitStats()
		if stats.State == StateOpen {
			return CheckResult{
				Status:      HealthStatusDegraded,
				Message:     fmt.Sprintf("Circuit breaker open, retry at %s", stats.NextRetryTime.Format(time.RFC3339)),
				Duration:    time.Since(start),
				LastChecked: time.Now(),
			}
		} else if stats.State == StateHalfOpen {
			return CheckResult{
				Status:      HealthStatusDegraded,
				Message:     "Circuit breaker in half-open state",
				Duration:    time.Since(start),
				LastChecked: time.Now(),
			}
		}
	}

	return CheckResult{
		Status:      HealthStatusHealthy,
		Message:     fmt.Sprintf("Provider %s is operational", providerName),
		Duration:    time.Since(start),
		LastChecked: time.Now(),
	}
}

func (h *DefaultHealthChecker) checkRateLimiter(ctx context.Context) CheckResult {
	start := time.Now()

	if h.rateLimiter == nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Message:     "Rate limiter not configured",
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Test rate limiter with a test key
	testKey := fmt.Sprintf("health_check_%d", time.Now().UnixNano())

	// Check if we can make a request
	allowed, err := h.rateLimiter.Allow(ctx, testKey)
	if err != nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Error:       err.Error(),
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Get status
	status, err := h.rateLimiter.GetStatus(ctx, testKey)
	if err != nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Error:       err.Error(),
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	// Clean up
	h.rateLimiter.Reset(ctx, testKey)

	message := fmt.Sprintf("Rate limiter operational: limit=%d, remaining=%d",
		status.Limit, status.Remaining)

	if !allowed {
		return CheckResult{
			Status:      HealthStatusDegraded,
			Message:     "Rate limiter is limiting requests",
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	return CheckResult{
		Status:      HealthStatusHealthy,
		Message:     message,
		Duration:    time.Since(start),
		LastChecked: time.Now(),
	}
}

func (h *DefaultHealthChecker) runCheck(ctx context.Context, check func(context.Context) error) CheckResult {
	start := time.Now()

	// Run check with timeout
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := check(checkCtx)

	if err != nil {
		return CheckResult{
			Status:      HealthStatusUnhealthy,
			Error:       err.Error(),
			Duration:    time.Since(start),
			LastChecked: time.Now(),
		}
	}

	return CheckResult{
		Status:      HealthStatusHealthy,
		Duration:    time.Since(start),
		LastChecked: time.Now(),
	}
}

func (h *DefaultHealthChecker) getHealthMetrics() *HealthMetrics {
	if h.metricsCollector == nil {
		return nil
	}

	metrics := h.metricsCollector.GetMetrics()

	// Calculate requests per second
	uptime := time.Since(metrics.StartTime).Seconds()
	rps := float64(0)
	if uptime > 0 {
		totalRequests := metrics.AuthRequests.Total + metrics.TokenExchanges.Total +
			metrics.TokenRefreshes.Total + metrics.UserInfoRequests.Total
		rps = float64(totalRequests) / uptime
	}

	// Calculate error rate
	errorRate := float64(0)
	totalRequests := metrics.AuthRequests.Total + metrics.TokenExchanges.Total +
		metrics.TokenRefreshes.Total + metrics.UserInfoRequests.Total
	if totalRequests > 0 {
		totalErrors := metrics.AuthRequests.Failed + metrics.TokenExchanges.Failed +
			metrics.TokenRefreshes.Failed + metrics.UserInfoRequests.Failed
		errorRate = float64(totalErrors) / float64(totalRequests)
	}

	// Get average latency
	avgLatency := int64(0)
	if rt, exists := metrics.ResponseTimes["token_exchange"]; exists && rt.Count > 0 {
		avgLatency = int64(rt.Average.Milliseconds())
	}

	return &HealthMetrics{
		RequestsPerSecond: rps,
		ErrorRate:         errorRate,
		AverageLatency:    avgLatency,
		ActiveSessions:    metrics.ActiveSessions,
		CachedTokens:      metrics.CachedTokens,
	}
}

// HealthHandler provides HTTP handlers for health checks
type HealthHandler struct {
	checker HealthChecker
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(checker HealthChecker) *HealthHandler {
	return &HealthHandler{
		checker: checker,
	}
}

// HandleHealth handles the main health check endpoint
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health, err := h.checker.Check(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set status code based on health status
	statusCode := http.StatusOK
	switch health.Status {
	case HealthStatusDegraded:
		statusCode = http.StatusOK // Still return 200 for degraded
	case HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// HandleLiveness handles the liveness probe endpoint
func (h *HealthHandler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - just return OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "alive",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// HandleReadiness handles the readiness probe endpoint
func (h *HealthHandler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health, err := h.checker.Check(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Only return ready if health is not unhealthy
	if health.Status == HealthStatusUnhealthy {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "not_ready",
			"reason":    "unhealthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
