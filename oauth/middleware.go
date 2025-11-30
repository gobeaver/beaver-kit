package oauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Middleware provides HTTP middleware for OAuth operations
type Middleware struct {
	service      *Service
	multiService *MultiProviderService
	rateLimiter  RateLimiter
	config       MiddlewareConfig
}

// MiddlewareConfig configures the OAuth middleware
type MiddlewareConfig struct {
	// Security headers
	EnableSecurityHeaders bool `env:"OAUTH_SECURITY_HEADERS,default:true"`
	EnableHSTS            bool `env:"OAUTH_HSTS_ENABLED,default:true"`
	HSTSMaxAge            int  `env:"OAUTH_HSTS_MAX_AGE,default:31536000"`

	// CORS configuration
	EnableCORS       bool     `env:"OAUTH_CORS_ENABLED,default:true"`
	AllowedOrigins   []string `env:"OAUTH_CORS_ORIGINS"`
	AllowedMethods   []string `env:"OAUTH_CORS_METHODS,default:GET,POST,OPTIONS"`
	AllowedHeaders   []string `env:"OAUTH_CORS_HEADERS,default:Content-Type,Authorization"`
	AllowCredentials bool     `env:"OAUTH_CORS_CREDENTIALS,default:true"`
	MaxAge           int      `env:"OAUTH_CORS_MAX_AGE,default:3600"`

	// Rate limiting
	EnableRateLimiting bool          `env:"OAUTH_RATE_LIMITING,default:true"`
	RateLimit          int           `env:"OAUTH_RATE_LIMIT,default:100"`
	RateInterval       time.Duration `env:"OAUTH_RATE_INTERVAL,default:1m"`
	RateBurstSize      int           `env:"OAUTH_RATE_BURST,default:200"`

	// Request logging
	EnableLogging    bool `env:"OAUTH_REQUEST_LOGGING,default:true"`
	LogSensitiveData bool `env:"OAUTH_LOG_SENSITIVE,default:false"`

	// Timeout configuration
	RequestTimeout time.Duration `env:"OAUTH_REQUEST_TIMEOUT,default:30s"`
	IdleTimeout    time.Duration `env:"OAUTH_IDLE_TIMEOUT,default:120s"`

	// Security
	RequireHTTPS   bool     `env:"OAUTH_REQUIRE_HTTPS,default:true"`
	TrustedProxies []string `env:"OAUTH_TRUSTED_PROXIES"`
}

// NewMiddleware creates a new OAuth middleware
func NewMiddleware(config MiddlewareConfig) *Middleware {
	var rateLimiter RateLimiter
	if config.EnableRateLimiting {
		rateLimiter = NewTokenBucketLimiter(RateLimiterConfig{
			Rate:      config.RateLimit,
			Interval:  config.RateInterval,
			BurstSize: config.RateBurstSize,
		})
	}

	return &Middleware{
		rateLimiter: rateLimiter,
		config:      config,
	}
}

// WithService sets the OAuth service for the middleware
func (m *Middleware) WithService(service *Service) *Middleware {
	m.service = service
	return m
}

// WithMultiProviderService sets the multi-provider service for the middleware
func (m *Middleware) WithMultiProviderService(service *MultiProviderService) *Middleware {
	m.multiService = service
	return m
}

// SecurityHeaders adds security headers to responses
func (m *Middleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.config.EnableSecurityHeaders {
			// Basic security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Content Security Policy
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline'; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data: https:; "+
					"font-src 'self' data:; "+
					"connect-src 'self'; "+
					"frame-ancestors 'none';")

			// HSTS header
			if m.config.EnableHSTS && (r.TLS != nil || m.isFromTrustedProxy(r)) {
				hstsValue := fmt.Sprintf("max-age=%d; includeSubDomains; preload", m.config.HSTSMaxAge)
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			// Permissions Policy
			w.Header().Set("Permissions-Policy",
				"accelerometer=(), "+
					"camera=(), "+
					"geolocation=(), "+
					"gyroscope=(), "+
					"magnetometer=(), "+
					"microphone=(), "+
					"payment=(), "+
					"usb=()")
		}

		next.ServeHTTP(w, r)
	})
}

// CORS handles Cross-Origin Resource Sharing
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.EnableCORS {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if m.isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)

			if m.config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.config.AllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", m.config.MaxAge))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Add headers for actual requests
			w.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset")
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimit implements rate limiting middleware
func (m *Middleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.EnableRateLimiting || m.rateLimiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Get rate limit key (IP address or user ID)
		key := m.getRateLimitKey(r)

		// Check rate limit
		allowed, err := m.rateLimiter.Allow(r.Context(), key)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Get status for headers
		status, _ := m.rateLimiter.GetStatus(r.Context(), key)
		if status != nil {
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", status.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", status.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", status.Reset.Unix()))

			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(status.RetryAfter.Seconds())))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequestLogging logs incoming requests
func (m *Middleware) RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.EnableLogging {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Log request
		logEntry := RequestLog{
			Method:     r.Method,
			Path:       r.URL.Path,
			RemoteAddr: m.getClientIP(r),
			UserAgent:  r.UserAgent(),
			RequestID:  r.Header.Get("X-Request-ID"),
			StartTime:  start,
		}

		// Add to context for later use
		ctx := context.WithValue(r.Context(), "request_log", &logEntry)
		r = r.WithContext(ctx)

		// Process request
		next.ServeHTTP(wrapped, r)

		// Complete log entry
		logEntry.StatusCode = wrapped.statusCode
		logEntry.Duration = time.Since(start)
		logEntry.BytesWritten = wrapped.bytesWritten

		// Log the request (implement actual logging based on your needs)
		m.logRequest(logEntry)
	})
}

// RequireHTTPS ensures requests are made over HTTPS
func (m *Middleware) RequireHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.RequireHTTPS {
			next.ServeHTTP(w, r)
			return
		}

		// Check if request is HTTPS
		if r.TLS == nil && !m.isFromTrustedProxy(r) {
			// Check X-Forwarded-Proto header
			if r.Header.Get("X-Forwarded-Proto") != "https" {
				http.Error(w, "HTTPS Required", http.StatusBadRequest)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// Timeout adds request timeout
func (m *Middleware) Timeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.config.RequestTimeout <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), m.config.RequestTimeout)
		defer cancel()

		r = r.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			next.ServeHTTP(w, r)
			close(done)
		}()

		select {
		case <-done:
			// Request completed successfully
		case <-ctx.Done():
			// Request timed out
			http.Error(w, "Request Timeout", http.StatusRequestTimeout)
		}
	})
}

// Chain combines multiple middleware functions
func (m *Middleware) Chain(handlers ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(handlers) - 1; i >= 0; i-- {
			final = handlers[i](final)
		}
		return final
	}
}

// DefaultChain returns the default middleware chain
func (m *Middleware) DefaultChain() func(http.Handler) http.Handler {
	return m.Chain(
		m.RequireHTTPS,
		m.SecurityHeaders,
		m.CORS,
		m.RateLimit,
		m.RequestLogging,
		m.Timeout,
	)
}

// Helper methods

func (m *Middleware) isOriginAllowed(origin string) bool {
	if len(m.config.AllowedOrigins) == 0 {
		return false
	}

	for _, allowed := range m.config.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}

func (m *Middleware) getRateLimitKey(r *http.Request) string {
	// Try to get user ID from context or header
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return "user:" + userID
	}

	// Fall back to IP address
	return "ip:" + m.getClientIP(r)
}

func (m *Middleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}

	return ip
}

func (m *Middleware) isFromTrustedProxy(r *http.Request) bool {
	if len(m.config.TrustedProxies) == 0 {
		return false
	}

	clientIP := m.getClientIP(r)
	for _, proxy := range m.config.TrustedProxies {
		if proxy == clientIP {
			return true
		}
	}

	return false
}

func (m *Middleware) logRequest(log RequestLog) {
	// Implement actual logging based on your logging infrastructure
	// This is a placeholder that could write to stdout, a file, or a logging service
	if !m.config.LogSensitiveData {
		// Redact sensitive information
		log.sanitize()
	}

	// Log the request (implement based on your needs)
	// fmt.Printf("[%s] %s %s %d %v\n",
	//     log.StartTime.Format(time.RFC3339),
	//     log.Method,
	//     log.Path,
	//     log.StatusCode,
	//     log.Duration)
}

// RequestLog represents a logged request
type RequestLog struct {
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	StatusCode   int           `json:"status_code"`
	Duration     time.Duration `json:"duration"`
	RemoteAddr   string        `json:"remote_addr"`
	UserAgent    string        `json:"user_agent"`
	RequestID    string        `json:"request_id"`
	BytesWritten int           `json:"bytes_written"`
	StartTime    time.Time     `json:"start_time"`
}

func (l *RequestLog) sanitize() {
	// Remove sensitive query parameters
	if idx := strings.Index(l.Path, "?"); idx != -1 {
		l.Path = l.Path[:idx] + "?[REDACTED]"
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}
