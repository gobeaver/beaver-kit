package oauth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and reports OAuth metrics
type MetricsCollector interface {
	// RecordAuthRequest records an authorization request
	RecordAuthRequest(provider string, success bool, duration time.Duration)
	// RecordTokenExchange records a token exchange
	RecordTokenExchange(provider string, success bool, duration time.Duration)
	// RecordTokenRefresh records a token refresh
	RecordTokenRefresh(provider string, success bool, duration time.Duration)
	// RecordUserInfoRequest records a user info request
	RecordUserInfoRequest(provider string, success bool, duration time.Duration)
	// RecordRateLimitHit records a rate limit hit
	RecordRateLimitHit(key string)
	// RecordError records an error
	RecordError(provider string, operation string, errorType string)
	// GetMetrics returns current metrics
	GetMetrics() *Metrics
	// Reset resets all metrics
	Reset()
}

// Metrics represents collected OAuth metrics
type Metrics struct {
	// Request counts
	AuthRequests     MetricCounter `json:"auth_requests"`
	TokenExchanges   MetricCounter `json:"token_exchanges"`
	TokenRefreshes   MetricCounter `json:"token_refreshes"`
	UserInfoRequests MetricCounter `json:"user_info_requests"`

	// Error counts
	Errors map[string]MetricCounter `json:"errors"`

	// Rate limiting
	RateLimitHits MetricCounter `json:"rate_limit_hits"`

	// Performance metrics
	ResponseTimes map[string]ResponseTime `json:"response_times"`

	// Provider-specific metrics
	ProviderMetrics map[string]*ProviderMetric `json:"provider_metrics"`

	// System metrics
	ActiveSessions int64 `json:"active_sessions"`
	CachedTokens   int64 `json:"cached_tokens"`

	// Time window
	StartTime     time.Time `json:"start_time"`
	LastResetTime time.Time `json:"last_reset_time"`
}

// MetricCounter represents a counter metric
type MetricCounter struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
}

// ResponseTime represents response time statistics
type ResponseTime struct {
	Count   int64         `json:"count"`
	Total   time.Duration `json:"total"`
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
	Average time.Duration `json:"average"`
	P50     time.Duration `json:"p50"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
}

// ProviderMetric represents metrics for a specific provider
type ProviderMetric struct {
	AuthRequests     MetricCounter `json:"auth_requests"`
	TokenExchanges   MetricCounter `json:"token_exchanges"`
	TokenRefreshes   MetricCounter `json:"token_refreshes"`
	UserInfoRequests MetricCounter `json:"user_info_requests"`
	Errors           int64         `json:"errors"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastError        time.Time     `json:"last_error,omitempty"`
	LastSuccess      time.Time     `json:"last_success,omitempty"`
}

// DefaultMetricsCollector implements MetricsCollector with in-memory storage
type DefaultMetricsCollector struct {
	metrics   *Metrics
	mu        sync.RWMutex
	durations map[string][]time.Duration
}

// NewDefaultMetricsCollector creates a new default metrics collector
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		metrics: &Metrics{
			Errors:          make(map[string]MetricCounter),
			ResponseTimes:   make(map[string]ResponseTime),
			ProviderMetrics: make(map[string]*ProviderMetric),
			StartTime:       time.Now(),
			LastResetTime:   time.Now(),
		},
		durations: make(map[string][]time.Duration),
	}
}

// RecordAuthRequest records an authorization request
func (c *DefaultMetricsCollector) RecordAuthRequest(provider string, success bool, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update global counter
	atomic.AddInt64(&c.metrics.AuthRequests.Total, 1)
	if success {
		atomic.AddInt64(&c.metrics.AuthRequests.Success, 1)
	} else {
		atomic.AddInt64(&c.metrics.AuthRequests.Failed, 1)
	}

	// Update provider-specific metrics
	c.ensureProviderMetric(provider)
	pm := c.metrics.ProviderMetrics[provider]
	atomic.AddInt64(&pm.AuthRequests.Total, 1)
	if success {
		atomic.AddInt64(&pm.AuthRequests.Success, 1)
		pm.LastSuccess = time.Now()
	} else {
		atomic.AddInt64(&pm.AuthRequests.Failed, 1)
		pm.LastError = time.Now()
	}

	// Record response time
	c.recordDuration("auth_request", duration)
	c.recordDuration("auth_request_"+provider, duration)
}

// RecordTokenExchange records a token exchange
func (c *DefaultMetricsCollector) RecordTokenExchange(provider string, success bool, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update global counter
	atomic.AddInt64(&c.metrics.TokenExchanges.Total, 1)
	if success {
		atomic.AddInt64(&c.metrics.TokenExchanges.Success, 1)
	} else {
		atomic.AddInt64(&c.metrics.TokenExchanges.Failed, 1)
	}

	// Update provider-specific metrics
	c.ensureProviderMetric(provider)
	pm := c.metrics.ProviderMetrics[provider]
	atomic.AddInt64(&pm.TokenExchanges.Total, 1)
	if success {
		atomic.AddInt64(&pm.TokenExchanges.Success, 1)
		pm.LastSuccess = time.Now()
	} else {
		atomic.AddInt64(&pm.TokenExchanges.Failed, 1)
		pm.LastError = time.Now()
	}

	// Record response time
	c.recordDuration("token_exchange", duration)
	c.recordDuration("token_exchange_"+provider, duration)
}

// RecordTokenRefresh records a token refresh
func (c *DefaultMetricsCollector) RecordTokenRefresh(provider string, success bool, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update global counter
	atomic.AddInt64(&c.metrics.TokenRefreshes.Total, 1)
	if success {
		atomic.AddInt64(&c.metrics.TokenRefreshes.Success, 1)
	} else {
		atomic.AddInt64(&c.metrics.TokenRefreshes.Failed, 1)
	}

	// Update provider-specific metrics
	c.ensureProviderMetric(provider)
	pm := c.metrics.ProviderMetrics[provider]
	atomic.AddInt64(&pm.TokenRefreshes.Total, 1)
	if success {
		atomic.AddInt64(&pm.TokenRefreshes.Success, 1)
		pm.LastSuccess = time.Now()
	} else {
		atomic.AddInt64(&pm.TokenRefreshes.Failed, 1)
		pm.LastError = time.Now()
	}

	// Record response time
	c.recordDuration("token_refresh", duration)
	c.recordDuration("token_refresh_"+provider, duration)
}

// RecordUserInfoRequest records a user info request
func (c *DefaultMetricsCollector) RecordUserInfoRequest(provider string, success bool, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update global counter
	atomic.AddInt64(&c.metrics.UserInfoRequests.Total, 1)
	if success {
		atomic.AddInt64(&c.metrics.UserInfoRequests.Success, 1)
	} else {
		atomic.AddInt64(&c.metrics.UserInfoRequests.Failed, 1)
	}

	// Update provider-specific metrics
	c.ensureProviderMetric(provider)
	pm := c.metrics.ProviderMetrics[provider]
	atomic.AddInt64(&pm.UserInfoRequests.Total, 1)
	if success {
		atomic.AddInt64(&pm.UserInfoRequests.Success, 1)
		pm.LastSuccess = time.Now()
	} else {
		atomic.AddInt64(&pm.UserInfoRequests.Failed, 1)
		pm.LastError = time.Now()
	}

	// Record response time
	c.recordDuration("user_info", duration)
	c.recordDuration("user_info_"+provider, duration)
}

// RecordRateLimitHit records a rate limit hit
func (c *DefaultMetricsCollector) RecordRateLimitHit(key string) {
	atomic.AddInt64(&c.metrics.RateLimitHits.Total, 1)
}

// RecordError records an error
func (c *DefaultMetricsCollector) RecordError(provider string, operation string, errorType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create error key
	errorKey := fmt.Sprintf("%s_%s_%s", provider, operation, errorType)

	// Update error counter
	if counter, exists := c.metrics.Errors[errorKey]; exists {
		atomic.AddInt64(&counter.Total, 1)
		c.metrics.Errors[errorKey] = counter
	} else {
		c.metrics.Errors[errorKey] = MetricCounter{Total: 1}
	}

	// Update provider error count
	c.ensureProviderMetric(provider)
	atomic.AddInt64(&c.metrics.ProviderMetrics[provider].Errors, 1)
	c.metrics.ProviderMetrics[provider].LastError = time.Now()
}

// GetMetrics returns current metrics
func (c *DefaultMetricsCollector) GetMetrics() *Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Calculate response time statistics
	for key, durations := range c.durations {
		if len(durations) > 0 {
			c.metrics.ResponseTimes[key] = calculateResponseTimeStats(durations)
		}
	}

	// Calculate provider average latencies
	for provider, pm := range c.metrics.ProviderMetrics {
		key := "token_exchange_" + provider
		if durations, exists := c.durations[key]; exists && len(durations) > 0 {
			total := time.Duration(0)
			for _, d := range durations {
				total += d
			}
			pm.AverageLatency = total / time.Duration(len(durations))
		}
	}

	// Create a copy of metrics to avoid race conditions
	metricsCopy := *c.metrics

	// Deep copy maps
	metricsCopy.Errors = make(map[string]MetricCounter)
	for k, v := range c.metrics.Errors {
		metricsCopy.Errors[k] = v
	}

	metricsCopy.ResponseTimes = make(map[string]ResponseTime)
	for k, v := range c.metrics.ResponseTimes {
		metricsCopy.ResponseTimes[k] = v
	}

	metricsCopy.ProviderMetrics = make(map[string]*ProviderMetric)
	for k, v := range c.metrics.ProviderMetrics {
		pmCopy := *v
		metricsCopy.ProviderMetrics[k] = &pmCopy
	}

	return &metricsCopy
}

// Reset resets all metrics
func (c *DefaultMetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = &Metrics{
		Errors:          make(map[string]MetricCounter),
		ResponseTimes:   make(map[string]ResponseTime),
		ProviderMetrics: make(map[string]*ProviderMetric),
		StartTime:       time.Now(),
		LastResetTime:   time.Now(),
	}
	c.durations = make(map[string][]time.Duration)
}

// Helper methods

func (c *DefaultMetricsCollector) ensureProviderMetric(provider string) {
	if _, exists := c.metrics.ProviderMetrics[provider]; !exists {
		c.metrics.ProviderMetrics[provider] = &ProviderMetric{}
	}
}

func (c *DefaultMetricsCollector) recordDuration(key string, duration time.Duration) {
	if _, exists := c.durations[key]; !exists {
		c.durations[key] = make([]time.Duration, 0, 100)
	}
	c.durations[key] = append(c.durations[key], duration)

	// Keep only last 1000 samples to avoid memory issues
	if len(c.durations[key]) > 1000 {
		c.durations[key] = c.durations[key][100:]
	}
}

func calculateResponseTimeStats(durations []time.Duration) ResponseTime {
	if len(durations) == 0 {
		return ResponseTime{}
	}

	// Calculate basic stats
	total := time.Duration(0)
	minDur := durations[0]
	maxDur := durations[0]

	for _, d := range durations {
		total += d
		if d < minDur {
			minDur = d
		}
		if d > maxDur {
			maxDur = d
		}
	}

	average := total / time.Duration(len(durations))

	// Sort for percentiles (simple implementation)
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate percentiles
	p50 := sorted[len(sorted)*50/100]
	p95 := sorted[len(sorted)*95/100]
	p99 := sorted[len(sorted)*99/100]

	return ResponseTime{
		Count:   int64(len(durations)),
		Total:   total,
		Min:     minDur,
		Max:     maxDur,
		Average: average,
		P50:     p50,
		P95:     p95,
		P99:     p99,
	}
}

// MonitoringService provides monitoring capabilities
type MonitoringService struct {
	collector MetricsCollector
	config    MonitoringConfig
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// MonitoringConfig configures the monitoring service
type MonitoringConfig struct {
	Enabled         bool          `env:"OAUTH_MONITORING_ENABLED" envDefault:"true"`
	MetricsInterval time.Duration `env:"OAUTH_METRICS_INTERVAL" envDefault:"1m"`
	RetentionPeriod time.Duration `env:"OAUTH_METRICS_RETENTION" envDefault:"24h"`
	ExportEndpoint  string        `env:"OAUTH_METRICS_ENDPOINT"`
	ExportFormat    string        `env:"OAUTH_METRICS_FORMAT" envDefault:"json"`
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(config MonitoringConfig) *MonitoringService {
	return &MonitoringService{
		collector: NewDefaultMetricsCollector(),
		config:    config,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the monitoring service
func (s *MonitoringService) Start() {
	if !s.config.Enabled {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.config.MetricsInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.exportMetrics()
			case <-s.stopCh:
				return
			}
		}
	}()
}

// Stop stops the monitoring service
func (s *MonitoringService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// GetCollector returns the metrics collector
func (s *MonitoringService) GetCollector() MetricsCollector {
	return s.collector
}

// exportMetrics exports metrics to configured endpoint
func (s *MonitoringService) exportMetrics() {
	if s.config.ExportEndpoint == "" {
		return
	}

	// Get metrics for export
	_ = s.collector.GetMetrics()

	// Export metrics based on format
	switch s.config.ExportFormat {
	case "json":
		// Export as JSON (implement based on your needs)
	case "prometheus":
		// Export in Prometheus format (implement based on your needs)
	default:
		// Default export format
	}
}

// InstrumentedService wraps Service with monitoring
type InstrumentedService struct {
	service   *Service
	collector MetricsCollector
}

// NewInstrumentedService creates a new instrumented service
func NewInstrumentedService(service *Service, collector MetricsCollector) *InstrumentedService {
	return &InstrumentedService{
		service:   service,
		collector: collector,
	}
}

// GetAuthURL generates an authorization URL with monitoring
func (s *InstrumentedService) GetAuthURL(ctx context.Context) (string, error) {
	start := time.Now()
	url, err := s.service.GetAuthURL(ctx)
	duration := time.Since(start)

	s.collector.RecordAuthRequest(s.service.provider.Name(), err == nil && url != "", duration)

	return url, err
}

// Exchange exchanges an authorization code for tokens with monitoring
func (s *InstrumentedService) Exchange(ctx context.Context, code, state string) (*Token, error) {
	start := time.Now()
	token, err := s.service.Exchange(ctx, code, state)
	duration := time.Since(start)

	success := err == nil
	s.collector.RecordTokenExchange(s.service.provider.Name(), success, duration)

	if err != nil {
		s.collector.RecordError(s.service.provider.Name(), "exchange", getErrorType(err))
	}

	return token, err
}

func getErrorType(err error) string {
	// Classify error types
	switch {
	case errors.Is(err, ErrInvalidState):
		return "invalid_state"
	case errors.Is(err, ErrNoRefreshToken):
		return "no_refresh_token"
	case errors.Is(err, ErrProviderNotFound):
		return "provider_not_found"
	default:
		return "unknown"
	}
}
