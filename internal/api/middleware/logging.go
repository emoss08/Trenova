// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package middleware

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
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

// entryPool is a pool of log entry maps to reduce allocations
var entryPool = sync.Pool{
	New: func() any {
		return make(map[string]any, 16) // Pre-allocate with reasonable size
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
	var parsed any
	if err := sonic.Unmarshal([]byte(s), &parsed); err == nil {
		truncated := createTruncatedObject(parsed)

		// Convert back to JSON
		if bytes, bErr := sonic.Marshal(truncated); bErr == nil && len(bytes) <= maxLen {
			return string(bytes)
		}
	}

	// If not valid JSON or can't intelligently truncate, use safe UTF-8 truncation
	return safelyTruncateString(s, maxLen)
}

// createTruncatedObject creates a truncated representation of the parsed JSON
func createTruncatedObject(parsed any) map[string]any {
	truncated := map[string]any{
		"truncated": true,
		"message":   "Content exceeded max length and was truncated",
	}

	// Include the content type if we can determine it
	switch v := parsed.(type) {
	case map[string]any:
		truncated["type"] = "object"
		truncated["partial"] = truncateMapSample(v)
	case []any:
		truncated["type"] = "array"
		truncated["length"] = len(v)
		truncated["partial"] = truncateArraySample(v)
	default:
		truncated["type"] = fmt.Sprintf("%T", parsed)
	}

	return truncated
}

// truncateMapSample takes a sample of a map for truncation
func truncateMapSample(m map[string]any) any {
	if len(m) <= 3 {
		return m
	}

	sample := make(map[string]any, 3)
	count := 0
	for k, val := range m {
		sample[k] = val
		count++
		if count >= 3 {
			break
		}
	}
	return sample
}

// truncateArraySample takes a sample of an array for truncation
func truncateArraySample(arr []any) any {
	if len(arr) == 0 {
		return arr
	}

	if len(arr) <= 2 {
		return arr
	}

	return arr[:2]
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
	logChan   chan map[string]any
	logger    *logger.Logger
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    bool
}

// NewAsyncLogger creates a new asynchronous logger
func NewAsyncLogger(l *logger.Logger, bufferSize int) *Logger {
	log := &Logger{
		logChan: make(chan map[string]any, bufferSize),
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
func (l *Logger) Log(entry map[string]any) {
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

	return func(c *fiber.Ctx) error {
		// Check if we should skip logging this request
		if shouldSkipLogging(c, &cfg) {
			return c.Next()
		}

		// Ensure request ID exists
		requestID := ensureRequestID(c)

		// Capture timing and request info
		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Extract request body if configured
		reqBody := extractRequestBody(c, &cfg)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)
		formattedDuration := formatDuration(duration)

		// Create and populate log entry
		entry := createLogEntry(
			c,
			&cfg,
			requestID,
			method,
			path,
			duration,
			formattedDuration,
			reqBody,
		)

		// Send to async logger
		asyncLogger.Log(entry)

		return err
	}
}

// shouldSkipLogging determines if request logging should be skipped
func shouldSkipLogging(c *fiber.Ctx, cfg *LogConfig) bool {
	// Skip logging for excluded paths - cheap operation first
	if shouldSkipPath(c.Path(), cfg.ExcludePaths) {
		return true
	}

	// Skip if custom skip function returns true
	if cfg.Skip != nil && cfg.Skip(c) {
		return true
	}

	// Apply sampling if configured
	if !shouldSample(cfg.SamplingRate) {
		return true
	}

	return false
}

// ensureRequestID ensures a request ID exists, generating one if needed
func ensureRequestID(c *fiber.Ctx) string {
	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = ulid.Make().String()
		c.Set("X-Request-ID", requestID)
	}
	return requestID
}

// extractRequestBody extracts and truncates the request body if configured
func extractRequestBody(c *fiber.Ctx, cfg *LogConfig) string {
	if !cfg.LogRequestBody || len(c.Body()) == 0 || (cfg.DetectGzip && isGzipped(c)) {
		return ""
	}

	contentType := c.Get("Content-Type")
	return truncateBodySafely(c.Body(), contentType, cfg.MaxBodySize)
}

// createLogEntry creates and populates a log entry with request details
func createLogEntry(
	c *fiber.Ctx,
	cfg *LogConfig,
	requestID, method, path string,
	duration time.Duration,
	formattedDuration, reqBody string,
) map[string]any {
	// Get a log entry from the pool
	entryObj := entryPool.Get()
	entry, ok := entryObj.(map[string]any)
	if !ok {
		// If type assertion fails, create a new map
		entry = make(map[string]any, 16)
	}
	statusCode := c.Response().StatusCode()

	// Add basic info
	populateBasicInfo(
		entry,
		requestID,
		method,
		path,
		statusCode,
		c.IP(),
		formattedDuration,
		c.Get("User-Agent"),
	)

	// Add optional info
	addOptionalInfo(c, cfg, entry, reqBody, duration)

	return entry
}

// populateBasicInfo adds basic request information to the log entry
func populateBasicInfo(
	entry map[string]any,
	requestID, method, path string,
	statusCode int,
	ip, latency, userAgent string,
) {
	entry["requestId"] = requestID
	entry["method"] = method
	entry["path"] = path
	entry["status"] = statusCode
	entry["ip"] = ip
	entry["latency"] = latency
	entry["userAgent"] = userAgent
	entry["message"] = "HTTP Request"
}

// addOptionalInfo adds optional information to the log entry based on configuration
func addOptionalInfo(
	c *fiber.Ctx,
	cfg *LogConfig,
	entry map[string]any,
	reqBody string,
	duration time.Duration,
) {
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

	// Add response body if configured
	addResponseBody(c, cfg, entry)

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
}

// addResponseBody adds response body to the log entry if configured
func addResponseBody(c *fiber.Ctx, cfg *LogConfig, entry map[string]any) {
	if !cfg.LogResponseBody || len(c.Response().Body()) == 0 {
		return
	}

	gzipCondition := cfg.DetectGzip &&
		strings.Contains(strings.ToLower(c.GetRespHeader("Content-Encoding")), "gzip")
	if gzipCondition {
		return
	}

	// Log the size of the response body in bytes
	entry["responseBodySize"] = len(c.Response().Body())

	contentType := c.GetRespHeader("Content-Type")
	respBody := truncateBodySafely(c.Response().Body(), contentType, cfg.MaxBodySize)
	entry["responseBody"] = respBody
}
