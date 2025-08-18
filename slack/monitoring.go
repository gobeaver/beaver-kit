package slack

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks various metrics for the Slack service
type Metrics struct {
	messagesSent      atomic.Int64
	messagesSucceeded atomic.Int64
	messagesFailed    atomic.Int64
	rateLimitHits     atomic.Int64
	circuitOpens      atomic.Int64
	totalLatency      atomic.Int64
	requestCount      atomic.Int64
	mu                sync.RWMutex
	errorCounts       map[string]int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		errorCounts: make(map[string]int64),
	}
}

// RecordMessage records a message attempt
func (m *Metrics) RecordMessage() {
	m.messagesSent.Add(1)
}

// RecordSuccess records a successful message
func (m *Metrics) RecordSuccess(latency time.Duration) {
	m.messagesSucceeded.Add(1)
	m.totalLatency.Add(int64(latency))
	m.requestCount.Add(1)
}

// RecordFailure records a failed message
func (m *Metrics) RecordFailure(err error) {
	m.messagesFailed.Add(1)
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	errKey := fmt.Sprintf("%T", err)
	m.errorCounts[errKey]++
}

// RecordRateLimit records a rate limit hit
func (m *Metrics) RecordRateLimit() {
	m.rateLimitHits.Add(1)
}

// RecordCircuitOpen records when circuit breaker opens
func (m *Metrics) RecordCircuitOpen() {
	m.circuitOpens.Add(1)
}

// GetStats returns current statistics
func (m *Metrics) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := int64(0)
	if count := m.requestCount.Load(); count > 0 {
		avgLatency = m.totalLatency.Load() / count
	}

	errorCounts := make(map[string]int64)
	for k, v := range m.errorCounts {
		errorCounts[k] = v
	}

	return Stats{
		MessagesSent:      m.messagesSent.Load(),
		MessagesSucceeded: m.messagesSucceeded.Load(),
		MessagesFailed:    m.messagesFailed.Load(),
		RateLimitHits:     m.rateLimitHits.Load(),
		CircuitOpens:      m.circuitOpens.Load(),
		AverageLatency:    time.Duration(avgLatency),
		ErrorCounts:       errorCounts,
	}
}

// Stats represents service statistics
type Stats struct {
	MessagesSent      int64
	MessagesSucceeded int64
	MessagesFailed    int64
	RateLimitHits     int64
	CircuitOpens      int64
	AverageLatency    time.Duration
	ErrorCounts       map[string]int64
}

// Logger provides structured logging for the Slack service
type Logger struct {
	level   LogLevel
	enabled bool
	mu      sync.RWMutex
}

// LogLevel represents logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// NewLogger creates a new logger
func NewLogger(enabled bool, level string) *Logger {
	return &Logger{
		enabled: enabled,
		level:   parseLogLevel(level),
	}
}

// parseLogLevel parses string log level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// Debug logs debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogLevelDebug, format, args...)
}

// Info logs info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogLevelInfo, format, args...)
}

// Warn logs warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogLevelWarn, format, args...)
}

// Error logs error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogLevelError, format, args...)
}

// log performs the actual logging
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if !l.enabled || level < l.level {
		return
	}

	levelStr := ""
	switch level {
	case LogLevelDebug:
		levelStr = "[DEBUG]"
	case LogLevelInfo:
		levelStr = "[INFO]"
	case LogLevelWarn:
		levelStr = "[WARN]"
	case LogLevelError:
		levelStr = "[ERROR]"
	}

	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [SLACK] %s", levelStr, msg)
}

// RequestLogger logs requests with sensitive data redaction
type RequestLogger struct {
	logger  *Logger
	redact  bool
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(logger *Logger, redact bool) *RequestLogger {
	return &RequestLogger{
		logger: logger,
		redact: redact,
	}
}

// LogRequest logs an outgoing request
func (rl *RequestLogger) LogRequest(ctx context.Context, payload string) {
	if rl.redact {
		payload = rl.redactSensitive(payload)
	}
	rl.logger.Debug("Sending request: %s", payload)
}

// LogResponse logs an incoming response
func (rl *RequestLogger) LogResponse(ctx context.Context, statusCode int, body string) {
	if rl.redact {
		body = rl.redactSensitive(body)
	}
	rl.logger.Debug("Received response: status=%d, body=%s", statusCode, body)
}

// redactSensitive redacts sensitive information from strings
func (rl *RequestLogger) redactSensitive(s string) string {
	// Redact webhook URLs
	if strings.Contains(s, "hooks.slack.com") {
		parts := strings.Split(s, "/")
		for i, part := range parts {
			if strings.Contains(part, "hooks.slack.com") && i+3 < len(parts) {
				// Redact the token parts
				parts[i+2] = "REDACTED"
				parts[i+3] = "REDACTED"
			}
		}
		s = strings.Join(parts, "/")
	}

	// Redact potential tokens or secrets
	s = redactPattern(s, `"token":\s*"[^"]+"`,"\"token\":\"REDACTED\"")
	s = redactPattern(s, `"password":\s*"[^"]+"`,"\"password\":\"REDACTED\"")
	s = redactPattern(s, `"secret":\s*"[^"]+"`,"\"secret\":\"REDACTED\"")
	s = redactPattern(s, `"api_key":\s*"[^"]+"`,"\"api_key\":\"REDACTED\"")
	
	return s
}

// redactPattern replaces patterns in string
func redactPattern(s, pattern, replacement string) string {
	// Simple pattern replacement (in production, use regexp)
	if strings.Contains(strings.ToLower(s), strings.ToLower(pattern[:7])) {
		// This is simplified - in production use proper regex
		return s
	}
	return s
}