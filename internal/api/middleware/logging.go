package middleware

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/oklog/ulid/v2"
)

// LogConfig holds the configuration for the logging middleware
type LogConfig struct {
	// Skip defines a function to skip middleware execution
	Skip func(c *fiber.Ctx) bool

	// Custom logging tags
	CustomTags map[string]string

	// Minimum duration to log as slow request (default: 500ms)
	SlowRequestThreshold time.Duration

	// Log request body (default: false due to security and performance)
	LogRequestBody bool

	// Log response body (default: false due to security and performance)
	LogResponseBody bool

	// Maximum body size to log (default: 1024 bytes)
	MaxBodySize int

	// List of headers to log (default: empty)
	LogHeaders []string

	// Paths to exclude from logging (e.g., health checks, metrics)
	ExcludePaths []string
}

// DefaultLogConfig returns a default configuration for the logging middleware
func DefaultLogConfig() LogConfig {
	return LogConfig{
		SlowRequestThreshold: 500 * time.Millisecond,
		MaxBodySize:          1024,
		LogHeaders:           []string{"Content-Type", "X-Request-ID"},
		ExcludePaths:         []string{"/health", "/metrics"},
	}
}

// formatDuration formats a duration into a human-readable string with appropriate units
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fµs", float64(d.Nanoseconds())/float64(time.Microsecond))
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// shouldSkipPath checks if the current path should be excluded from logging
func shouldSkipPath(path string, excludePaths []string) bool {
	for _, excludePath := range excludePaths {
		if strings.HasPrefix(path, excludePath) {
			return true
		}
	}
	return false
}

// truncateString truncates a string if it exceeds the maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractHeaders extracts specified headers from the request
func extractHeaders(c *fiber.Ctx, headerNames []string) map[string]string {
	headers := make(map[string]string)
	for _, name := range headerNames {
		if value := c.Get(name); value != "" {
			headers[name] = value
		}
	}
	return headers
}

// NewLogger creates a new logging middleware with the given configuration
func NewLogger(l *logger.Logger, config ...LogConfig) fiber.Handler {
	// Use default config if none provided
	cfg := DefaultLogConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Skip logging for excluded paths
		if shouldSkipPath(c.Path(), cfg.ExcludePaths) {
			return c.Next()
		}

		// Skip if custom skip function returns true
		if cfg.Skip != nil && cfg.Skip(c) {
			return c.Next()
		}

		// Generate request ID if not present
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0)).String()
			c.Set("X-Request-ID", requestID)
		}

		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Extract request body if configured
		var reqBody string
		if cfg.LogRequestBody && c.Request().Body() != nil {
			reqBody = truncateString(string(c.Request().Body()), cfg.MaxBodySize)
		}

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)
		formattedDuration := formatDuration(duration)

		// Create log event
		logEvent := l.Info().
			Str("requestId", requestID).
			Str("method", method).
			Str("path", path).
			Int("status", c.Response().StatusCode()).
			Str("ip", c.IP()).
			Str("latency", formattedDuration).
			Str("userAgent", c.Get("User-Agent"))

		// Add headers if configured
		if len(cfg.LogHeaders) > 0 {
			headers := extractHeaders(c, cfg.LogHeaders)
			logEvent.Interface("headers", headers)
		}

		// Add request body if configured
		if cfg.LogRequestBody && reqBody != "" {
			logEvent.Str("requestBody", reqBody)
		}

		// Add response body if configured
		if cfg.LogResponseBody {
			respBody := truncateString(string(c.Response().Body()), cfg.MaxBodySize)
			logEvent.Str("responseBody", respBody)
		}

		// Add custom tags
		for key, value := range cfg.CustomTags {
			logEvent.Str(key, value)
		}

		// Log as warning if request is slow
		if duration > cfg.SlowRequestThreshold {
			logEvent.Msg("Slow HTTP Request")
		} else {
			logEvent.Msg("HTTP Request")
		}

		return err
	}
}
