package middleware

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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

	// Enable sampling to reduce logging volume (default: 1.0 - log everything)
	SamplingRate float64

	// Buffer size for log message channel (default: 1000)
	BufferSize int

	// Enable gzip detection to prevent logging compressed bodies
	DetectGzip bool
}

// DefaultLogConfig returns a default configuration for the logging middleware
func DefaultLogConfig() LogConfig {
	return LogConfig{
		SlowRequestThreshold: 500 * time.Millisecond,
		MaxBodySize:          1024,
		LogHeaders:           []string{"Content-Type", "X-Request-ID"},
		ExcludePaths:         []string{"/health", "/metrics"},
		SamplingRate:         1.0, // Log everything by default
		BufferSize:           1000,
		DetectGzip:           true,
	}
}

// entryCacheSize defines the size of the sync.Pool for log entries
const entryCacheSize = 1000

// entryPool is a pool of log entry maps to reduce allocations
var entryPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]interface{}, 16) // Pre-allocate with reasonable size
	},
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
// using a more efficient prefix check
func shouldSkipPath(path string, excludePaths []string) bool {
	for _, excludePath := range excludePaths {
		if strings.HasPrefix(path, excludePath) {
			return true
		}
	}
	return false
}

// safelyTruncateJSON truncates a JSON string while preserving valid JSON structure
func safelyTruncateJSON(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Try to parse as JSON to intelligently truncate
	var parsed interface{}
	if err := json.Unmarshal([]byte(s), &parsed); err == nil {
		// For JSON objects or arrays, create a "truncated" version
		truncated := map[string]interface{}{
			"truncated": true,
			"message":   "Content exceeded max length and was truncated",
		}

		// Include the content type if we can determine it
		switch v := parsed.(type) {
		case map[string]interface{}:
			truncated["type"] = "object"

			// Include a few fields from the original if it's not too big
			if len(v) <= 3 {
				truncated["partial"] = v
			} else {
				sample := make(map[string]interface{}, 3)
				count := 0
				for k, val := range v {
					sample[k] = val
					count++
					if count >= 3 {
						break
					}
				}
				truncated["partial"] = sample
			}
		case []interface{}:
			truncated["type"] = "array"
			truncated["length"] = len(v)
			if len(v) > 0 && len(v) <= 2 {
				truncated["partial"] = v
			} else if len(v) > 2 {
				truncated["partial"] = v[:2]
			}
		default:
			truncated["type"] = fmt.Sprintf("%T", parsed)
		}

		// Convert back to JSON
		if bytes, err := json.Marshal(truncated); err == nil && len(bytes) <= maxLen {
			return string(bytes)
		}
	}

	// If not valid JSON or can't intelligently truncate, use safe UTF-8 truncation
	return safelyTruncateString(s, maxLen)
}

// safelyTruncateString truncates a string ensuring it doesn't break UTF-8 characters
func safelyTruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Ensure we don't break in the middle of a UTF-8 character
	if maxLen > 3 {
		// Leave room for "..." suffix
		maxLen -= 3
	}

	// Find a safe position to truncate (not breaking UTF-8)
	validLen := 0
	for pos := 0; pos < maxLen && pos < len(s); {
		r, size := utf8.DecodeRuneInString(s[pos:])
		if r == utf8.RuneError {
			break
		}
		pos += size
		validLen = pos
	}

	return s[:validLen] + "..."
}

// extractHeaders extracts specified headers from the request
// with a pre-allocated map to reduce allocations
func extractHeaders(c *fiber.Ctx, headerNames []string) map[string]string {
	if len(headerNames) == 0 {
		return nil
	}

	headers := make(map[string]string, len(headerNames))
	for _, name := range headerNames {
		if value := c.Get(name); value != "" {
			headers[name] = value
		}
	}
	return headers
}

// isGzipped checks if the content is gzipped based on content-encoding header
func isGzipped(c *fiber.Ctx) bool {
	return strings.Contains(strings.ToLower(c.Get("Content-Encoding")), "gzip")
}

// truncateBodySafely truncates body content appropriately based on content type
func truncateBodySafely(body []byte, contentType string, maxSize int) string {
	if len(body) == 0 {
		return ""
	}

	strBody := string(body)
	if len(strBody) <= maxSize {
		return strBody
	}

	// Check content type to determine truncation method
	contentTypeLower := strings.ToLower(contentType)
	if strings.Contains(contentTypeLower, "json") {
		return safelyTruncateJSON(strBody, maxSize)
	}

	return safelyTruncateString(strBody, maxSize)
}

// shouldSample determines if this request should be logged based on sampling rate
func shouldSample(rate float64) bool {
	if rate >= 1.0 {
		return true
	}
	if rate <= 0.0 {
		return false
	}

	// Generate a random byte to compare against the rate
	randByte := make([]byte, 1)
	n, err := rand.Reader.Read(randByte)
	if err != nil || n != 1 {
		return true // Default to logging on error
	}

	return float64(randByte[0])/255.0 < rate
}

// Logger represents an asynchronous logger that buffers log entries
type Logger struct {
	logChan   chan map[string]interface{}
	logger    *logger.Logger
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    bool
}

// NewAsyncLogger creates a new asynchronous logger
func NewAsyncLogger(l *logger.Logger, bufferSize int) *Logger {
	log := &Logger{
		logChan: make(chan map[string]interface{}, bufferSize),
		logger:  l,
	}

	log.wg.Add(1)
	go log.processLogs()

	return log
}

// processLogs handles log entries asynchronously
func (l *Logger) processLogs() {
	defer l.wg.Done()

	for entry := range l.logChan {
		msg, _ := entry["message"].(string)
		delete(entry, "message")

		isWarn, _ := entry["isWarn"].(bool)
		delete(entry, "isWarn")

		if isWarn {
			evt := l.logger.Warn()
			for k, v := range entry {
				evt = evt.Interface(k, v)
			}
			evt.Msg(msg)
		} else {
			evt := l.logger.Debug()
			for k, v := range entry {
				evt = evt.Interface(k, v)
			}
			evt.Msg(msg)
		}

		// Return the entry to the pool
		for k := range entry {
			delete(entry, k)
		}
		entryPool.Put(entry)
	}
}

// Log adds a log entry to the channel
func (l *Logger) Log(entry map[string]interface{}) {
	if l.closed {
		return
	}
	select {
	case l.logChan <- entry:
		// Successfully sent to channel
	default:
		// Channel is full, drop the log entry and return map to pool
		entryPool.Put(entry)
	}
}

// Close shuts down the logger
func (l *Logger) Close() {
	l.closeOnce.Do(func() {
		l.closed = true
		close(l.logChan)
		l.wg.Wait()
	})
}

// NewLogger creates a new logging middleware with the given configuration
func NewLogger(l *logger.Logger, config ...LogConfig) fiber.Handler {
	// Use default config if none provided
	cfg := DefaultLogConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Create async logger
	asyncLogger := NewAsyncLogger(l, cfg.BufferSize)

	// Create entropy source for ULID generation
	entropy := ulid.Monotonic(rand.Reader, 0)

	return func(c *fiber.Ctx) error {
		// Skip logging for excluded paths - cheap operation first
		if shouldSkipPath(c.Path(), cfg.ExcludePaths) {
			return c.Next()
		}

		// Skip if custom skip function returns true
		if cfg.Skip != nil && cfg.Skip(c) {
			return c.Next()
		}

		// Apply sampling if configured
		if !shouldSample(cfg.SamplingRate) {
			return c.Next()
		}

		// Generate request ID if not present
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
			c.Set("X-Request-ID", requestID)
		}

		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Extract request body if configured (skip for gzipped content)
		var reqBody string
		if cfg.LogRequestBody && len(c.Body()) > 0 && !(cfg.DetectGzip && isGzipped(c)) {
			contentType := c.Get("Content-Type")
			reqBody = truncateBodySafely(c.Body(), contentType, cfg.MaxBodySize)
		}

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)
		formattedDuration := formatDuration(duration)
		statusCode := c.Response().StatusCode()

		// Get a log entry from the pool
		entry := entryPool.Get().(map[string]interface{})

		// Basic info every log will have
		entry["requestId"] = requestID
		entry["method"] = method
		entry["path"] = path
		entry["status"] = statusCode
		entry["ip"] = c.IP()
		entry["latency"] = formattedDuration
		entry["userAgent"] = c.Get("User-Agent")
		entry["message"] = "HTTP Request"

		// Add query params if they exist
		if queries := c.Queries(); len(queries) > 0 {
			entry["queryParams"] = queries
		}

		// Add headers if configured
		if len(cfg.LogHeaders) > 0 {
			if headers := extractHeaders(c, cfg.LogHeaders); len(headers) > 0 {
				entry["headers"] = headers
			}
		}

		// Add request body if configured and available
		if cfg.LogRequestBody && reqBody != "" {
			entry["requestBody"] = reqBody
		}

		// Add response body if configured and not gzipped
		if cfg.LogResponseBody && len(c.Response().Body()) > 0 &&
			!(cfg.DetectGzip && strings.Contains(strings.ToLower(c.GetRespHeader("Content-Encoding")), "gzip")) {

			// log the size of the response body in bytes
			entry["responseBodySize"] = len(c.Response().Body())

			contentType := c.GetRespHeader("Content-Type")
			respBody := truncateBodySafely(c.Response().Body(), contentType, cfg.MaxBodySize)
			entry["responseBody"] = respBody
		}

		// Add custom tags
		for key, value := range cfg.CustomTags {
			entry[key] = value
		}

		// Mark as warning if request is slow
		if duration > cfg.SlowRequestThreshold {
			entry["isWarn"] = true
			entry["message"] = "Slow HTTP Request"
		} else {
			entry["isWarn"] = false
			entry["status"] = "OK"
		}

		// Send to async logger
		asyncLogger.Log(entry)

		return err
	}
}
