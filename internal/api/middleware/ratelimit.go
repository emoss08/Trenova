package middleware

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	tCtx "github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

// RoleLimit defines rate limits based on user roles
type RoleLimit struct {
	Role        string        // User role
	MaxRequests int           // Maximum number of requests allowed
	Interval    time.Duration // Time window for rate limiting
}

// RateLimitParams defines the dependencies required for rate limiting
type RateLimitParams struct {
	Logger       *logger.Logger      // Logger instance for recording rate limit events
	Redis        *redis.Client       // Redis client for storing rate limit data
	ScriptLoader *redis.ScriptLoader // Script loader for Redis Lua scripts
}

// RateLimiter manages rate limiting functionality
type RateLimiter struct {
	l            *zerolog.Logger     // Structured logger
	redis        *redis.Client       // Redis client instance
	scriptLoader *redis.ScriptLoader // Script loader for Redis Lua scripts
}

// RateLimitConfig defines the configuration for rate limiting
type RateLimitConfig struct {
	MaxRequests      int                     // Maximum number of requests allowed in the interval
	Interval         time.Duration           // Time window for rate limiting
	KeyPrefix        string                  // Optional prefix for Redis keys
	Skip             func(*fiber.Ctx) bool   // Optional function to skip rate limiting
	RoleLimits       []RoleLimit             // Optional role-based limits
	TokenBucketSize  int                     // Optional token bucket size for burst handling (0 disables)
	TokenRefillRate  float64                 // Optional tokens to refill per second (0 disables)
	CustomResponse   func(*fiber.Ctx) error  // Optional custom response when rate limited
	GroupKey         func(*fiber.Ctx) string // Optional function to group endpoints under the same bucket
	Strategy         string                  // Rate limiting strategy: "fixed", "sliding", "token"
	KeyGenerator     func(*fiber.Ctx) string // Optional custom key generator
	Metrics          bool                    // Enable metrics collection
	MaxRetries       int                     // Maximum number of Redis retries on failure
	EnableBypass     bool                    // Enable bypass tokens
	BypassCheck      func(*fiber.Ctx) bool   // Function to check if request can bypass rate limits
	Timeout          time.Duration           // Timeout for Redis operations
	FallbackBehavior string                  // "allow" or "deny" on Redis failure
}

// RateLimitMetrics stores rate limit metrics
type RateLimitMetrics struct {
	Allowed     int64     `json:"allowed"`
	Blocked     int64     `json:"blocked"`
	LastBlocked time.Time `json:"last_blocked"`
}

// NewRateLimit creates a new rate limiter instance with the provided dependencies
func NewRateLimit(p RateLimitParams) *RateLimiter {
	log := p.Logger.
		With().
		Str("middleware", "rate-limit").
		Logger()

	return &RateLimiter{
		l:            &log,
		redis:        p.Redis,
		scriptLoader: p.ScriptLoader,
	}
}

// WithRateLimit wraps the provided handlers with rate limiting middleware
// Returns a slice of handlers with rate limiting applied
func (rl *RateLimiter) WithRateLimit(handlers []fiber.Handler, config *RateLimitConfig) []fiber.Handler {
	// Set defaults if not provided
	rl.setConfigDefaults(config)

	limitHandler := rl.createLimitHandler(config)
	result := make([]fiber.Handler, 0, len(handlers)+1)
	result = append(result, limitHandler)
	result = append(result, handlers...)
	return result
}

// setConfigDefaults sets default values for the rate limit configuration
func (rl *RateLimiter) setConfigDefaults(config *RateLimitConfig) {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit"
	}

	if config.Strategy == "" {
		config.Strategy = "fixed" // Default to fixed window
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.Timeout == 0 {
		config.Timeout = 100 * time.Millisecond
	}

	if config.FallbackBehavior == "" {
		config.FallbackBehavior = "allow" // Default to fail-open
	}

	if config.KeyGenerator == nil {
		config.KeyGenerator = rl.defaultKeyGenerator
	}
}

// createLimitHandler creates a Fiber middleware handler that implements rate limiting
func (rl *RateLimiter) createLimitHandler(config *RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip rate limiting if skip function is provided and returns true
		if config.Skip != nil && config.Skip(c) {
			return c.Next()
		}

		// Check for bypass token if enabled
		if config.EnableBypass && config.BypassCheck != nil && config.BypassCheck(c) {
			c.Set("X-RateLimit-Bypassed", "true")
			return c.Next()
		}

		// Create a context-specific logger
		log := rl.l.With().
			Str("path", c.Path()).
			Str("method", c.Method()).
			Str("ip", c.IP()).
			Logger()

		// Get appropriate rate limit for this user/role
		maxRequests, interval := rl.getRoleLimits(c, config)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Context(), config.Timeout)
		defer cancel()

		// Build unique key for this rate limit
		var key string
		if config.GroupKey != nil {
			key = config.GroupKey(c)
		} else {
			key = config.KeyGenerator(c)
		}

		var allowed bool
		var remaining int
		var reset int64
		var err error

		// Choose strategy based on configuration
		switch config.Strategy {
		case "sliding":
			allowed, remaining, reset, err = rl.checkSlidingWindowLimit(ctx, key, maxRequests, interval)
		case "token":
			allowed, remaining, reset, err = rl.checkTokenBucketLimit(ctx, key, config)
		default:
			allowed, remaining, reset, err = rl.checkFixedWindowLimit(ctx, key, maxRequests, interval)
		}

		if err != nil {
			log.Error().Err(err).Msg("rate limit check failed")
			// Determine what to do on failure
			if config.FallbackBehavior == "deny" {
				rlErr := errors.NewRateLimitError(c.Path(), "Rate limiting unavailable. Please try again later.")
				return c.Status(fiber.StatusServiceUnavailable).JSON(rlErr)
			}
			return c.Next() // Allow request on error for high availability
		}

		// Ensure reset time is not in the past
		now := time.Now().Unix()
		if reset <= now {
			// If reset time is in the past, set it to a small time in the future
			reset = now + 1

			// Also, we should reset the counter if the reset time was in the past
			if !allowed {
				log.Warn().
					Str("ip", c.IP()).
					Str("path", c.Path()).
					Msg("reset time was in the past but limit was exceeded; forcing counter reset")

				// Force reset the counter
				rl.resetCounter(ctx, key, config.Strategy)

				// Allow this request since we're resetting
				allowed = true
				remaining = maxRequests - 1
			}
		}

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		c.Set("X-RateLimit-Strategy", config.Strategy)

		if !allowed {
			log.Warn().
				Str("ip", c.IP()).
				Str("path", c.Path()).
				Int("limit", maxRequests).
				Msg("rate limit exceeded")

			// Update metrics if enabled
			if config.Metrics {
				rl.updateBlockMetrics(ctx, key)
			}

			// Use custom response if provided, otherwise use default
			if config.CustomResponse != nil {
				return config.CustomResponse(c)
			}

			rlErr := errors.NewRateLimitError(c.Path(), "Rate limit exceeded. Please try again later.")
			return c.Status(fiber.StatusTooManyRequests).JSON(rlErr)
		}

		// Update metrics if enabled
		if config.Metrics {
			rl.updateAllowMetrics(ctx, key)
		}

		return c.Next()
	}
}

// resetCounter resets the rate limit counter based on the strategy
func (rl *RateLimiter) resetCounter(ctx context.Context, key string, strategy string) error {
	var err error
	switch strategy {
	case "sliding":
		// For sliding window, just delete the key
		err = rl.redis.Del(ctx, key)
	case "token":
		// For token bucket, reset to full bucket size
		err = rl.redis.Del(ctx, key)
		err = rl.redis.Del(ctx, key+":timestamp")
	default:
		// For fixed window, delete both counter and window keys
		err = rl.redis.Del(ctx, key)
		err = rl.redis.Del(ctx, key+":window")
	}

	if err != nil {
		rl.l.Error().Err(err).Str("key", key).Msg("failed to reset counter")
	}

	return err
}

// verifyCounterConsistency ensures window and counter keys are consistent
func (rl *RateLimiter) verifyCounterConsistency(ctx context.Context, key string, windowKey string) error {
	// Check if window exists but counter doesn't or vice versa
	exists, err := rl.redis.Exists(ctx, windowKey)
	if err != nil {
		return eris.Wrap(err, "failed to check window key existence")
	}

	counterExists, err := rl.redis.Exists(ctx, key)
	if err != nil {
		return eris.Wrap(err, "failed to check counter key existence")
	}

	// If only one of the keys exists, delete both to ensure consistency
	if (exists && !counterExists) || (!exists && counterExists) {
		rl.l.Warn().
			Str("key", key).
			Str("windowKey", windowKey).
			Msg("inconsistent rate limit state detected, resetting")

		if err := rl.redis.Del(ctx, key); err != nil {
			return eris.Wrap(err, "failed to delete counter key")
		}

		if err := rl.redis.Del(ctx, windowKey); err != nil {
			return eris.Wrap(err, "failed to delete window key")
		}
	}

	return nil
}

// defaultKeyGenerator generates a unique Redis key for the rate limit
func (rl *RateLimiter) defaultKeyGenerator(c *fiber.Ctx) string {
	parts := make([]string, 0, 5)
	parts = append(parts, "ratelimit", c.Method(), strings.Trim(c.Route().Path, "/"))

	ip := rl.getIP(c)
	parts = append(parts, ip)

	if userID, ok := c.Locals(tCtx.CTXUserID).(pulid.ID); ok && userID != "" {
		parts = append(parts, userID.String())
	}

	return strings.Join(parts, ":")
}

// getRoleLimits determines the appropriate rate limits based on user role
func (rl *RateLimiter) getRoleLimits(c *fiber.Ctx, config *RateLimitConfig) (int, time.Duration) {
	if len(config.RoleLimits) == 0 {
		return config.MaxRequests, config.Interval
	}

	// Get user role from context
	role, ok := c.Locals(tCtx.CTXUserRole).(string)
	if !ok {
		// Default to configured limits if role not found
		return config.MaxRequests, config.Interval
	}

	// Find matching role limit
	for _, rl := range config.RoleLimits {
		if rl.Role == role {
			return rl.MaxRequests, rl.Interval
		}
	}

	// Default to configured limits if no matching role
	return config.MaxRequests, config.Interval
}

// checkFixedWindowLimit implements fixed window rate limiting with consistency check
func (rl *RateLimiter) checkFixedWindowLimit(ctx context.Context, key string, maxRequests int, interval time.Duration) (bool, int, int64, error) {
	windowKey := key + ":window"

	// First verify counter consistency
	if err := rl.verifyCounterConsistency(ctx, key, windowKey); err != nil {
		rl.l.Warn().Err(err).Msg("counter consistency check failed, proceeding with rate limit check")
	}

	now := time.Now().Unix()

	result, err := rl.scriptLoader.EvalSHA(ctx, "fixed_window", []string{key, windowKey}, maxRequests, int(interval.Seconds()), now)
	if err != nil {
		return false, 0, 0, eris.Wrap(err, "failed to execute fixed window script")
	}

	results, ok := result.([]any)
	if !ok || len(results) != 3 {
		return false, 0, 0, eris.New("invalid response from fixed window script")
	}

	// Parse results
	allowed := results[0].(int64) == 1
	remaining := int(results[1].(int64))
	reset := results[2].(int64)

	// Double-check: if reset time is in the past, something is wrong
	if reset < now {
		rl.l.Warn().
			Str("key", key).
			Int64("reset", reset).
			Int64("now", now).
			Msg("reset time is in the past, possible clock skew or expired window")

		// Force reset the window
		err := rl.resetCounter(ctx, key, "fixed")
		if err != nil {
			rl.l.Error().Err(err).Msg("failed to reset counter after detecting past reset time")
		}

		// Set new reset time
		reset = now + int64(interval.Seconds())
		allowed = true
		remaining = maxRequests - 1
	}

	return allowed, remaining, reset, nil
}

// checkSlidingWindowLimit implements sliding window rate limiting
func (rl *RateLimiter) checkSlidingWindowLimit(ctx context.Context, key string, maxRequests int, interval time.Duration) (bool, int, int64, error) {
	now := time.Now().Unix()

	result, err := rl.scriptLoader.EvalSHA(ctx, "sliding_window", []string{key}, maxRequests, int(interval.Seconds()), now)
	if err != nil {
		return false, 0, 0, eris.Wrap(err, "failed to execute sliding window script")
	}

	results, ok := result.([]any)
	if !ok || len(results) != 3 {
		return false, 0, 0, eris.New("invalid response from sliding window script")
	}

	// Parse results
	allowed := results[0].(int64) == 1
	remaining := int(results[1].(int64))
	reset := results[2].(int64)

	return allowed, remaining, reset, nil
}

// checkTokenBucketLimit implements token bucket rate limiting
func (rl *RateLimiter) checkTokenBucketLimit(ctx context.Context, key string, config *RateLimitConfig) (bool, int, int64, error) {
	if config.TokenBucketSize == 0 || config.TokenRefillRate == 0 {
		// Fall back to fixed window if token bucket is not configured
		return rl.checkFixedWindowLimit(ctx, key, config.MaxRequests, config.Interval)
	}

	timestampKey := key + ":timestamp"
	now := time.Now().Unix()

	result, err := rl.scriptLoader.EvalSHA(
		ctx,
		"token_bucket",
		[]string{key, timestampKey},
		config.TokenBucketSize,
		config.TokenRefillRate,
		now,
	)
	if err != nil {
		return false, 0, 0, eris.Wrap(err, "failed to execute token bucket script")
	}

	results, ok := result.([]any)
	if !ok || len(results) != 3 {
		return false, 0, 0, eris.New("invalid response from token bucket script")
	}

	// Parse results
	allowed := results[0].(int64) == 1
	remaining := int(results[1].(int64))
	reset := results[2].(int64)

	return allowed, remaining, reset, nil
}

// updateAllowMetrics updates metrics for allowed requests
func (rl *RateLimiter) updateAllowMetrics(ctx context.Context, key string) {
	metricsKey := key + ":metrics"

	// Get existing metrics
	metrics := &RateLimitMetrics{
		Allowed: 1,
	}

	val, err := rl.redis.Get(ctx, metricsKey)
	if err == nil {
		// Unmarshal existing metrics
		if err := json.Unmarshal([]byte(val), metrics); err == nil {
			metrics.Allowed++
		}
	}

	// Marshal and save metrics
	if data, err := json.Marshal(metrics); err == nil {
		err = rl.redis.Set(ctx, metricsKey, string(data), 24*time.Hour)
		if err != nil {
			rl.l.Error().Err(err).Msg("failed to update allow metrics")
		}
	}
}

// updateBlockMetrics updates metrics for blocked requests
func (rl *RateLimiter) updateBlockMetrics(ctx context.Context, key string) {
	metricsKey := key + ":metrics"

	// Get existing metrics
	metrics := &RateLimitMetrics{
		Blocked:     1,
		LastBlocked: time.Now(),
	}

	val, err := rl.redis.Get(ctx, metricsKey)
	if err == nil {
		// Unmarshal existing metrics
		if err := json.Unmarshal([]byte(val), metrics); err == nil {
			metrics.Blocked++
			metrics.LastBlocked = time.Now()
		}
	}

	// Marshal and save metrics
	if data, err := json.Marshal(metrics); err == nil {
		err = rl.redis.Set(ctx, metricsKey, string(data), 24*time.Hour)
		if err != nil {
			rl.l.Error().Err(err).Msg("failed to update block metrics")
		}
	}
}

// GetMetrics retrieves rate limit metrics for a key
func (rl *RateLimiter) GetMetrics(ctx context.Context, key string) (*RateLimitMetrics, error) {
	metricsKey := key + ":metrics"

	val, err := rl.redis.Get(ctx, metricsKey)
	if err != nil {
		if eris.Is(err, redis.Nil) {
			return &RateLimitMetrics{}, nil
		}
		return nil, err
	}

	metrics := &RateLimitMetrics{}
	if err := json.Unmarshal([]byte(val), metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

// getIP retrieves the client's IP address, handling X-Forwarded-For header
// Returns "unknown" if the IP address cannot be parsed
func (rl *RateLimiter) getIP(c *fiber.Ctx) string {
	ip := c.IP()

	// If the request has a X-Forwarded-For header, use the first IP address
	if forwardedIP := c.Get("X-Forwarded-For"); forwardedIP != "" {
		ips := strings.Split(forwardedIP, ",")
		ip = strings.TrimSpace(ips[0])
	}

	// If the IP address cannot be parsed, return "unknown"
	if parsedIP := net.ParseIP(ip); parsedIP == nil {
		return "unknown"
	}

	return ip
}

// ClearRateLimits clears all rate limits for a specific key pattern
func (rl *RateLimiter) ClearRateLimits(ctx context.Context, keyPattern string) error {
	// Use SCAN to find all matching keys
	iter := rl.redis.Scan(ctx, 0, keyPattern+"*", 100).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())

		// Delete in batches of 100 to avoid blocking Redis
		if len(keys) >= 100 {
			if err := rl.redis.Del(ctx, keys...); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}

	// Delete any remaining keys
	if len(keys) > 0 {
		if err := rl.redis.Del(ctx, keys...); err != nil {
			return err
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	return nil
}

// Pre-defined rate limit configurations
func PerSecond(n int) *RateLimitConfig {
	return &RateLimitConfig{
		MaxRequests: n,
		Interval:    time.Second,
		Strategy:    "fixed",
	}
}

func PerMinute(n int) *RateLimitConfig {
	return &RateLimitConfig{
		MaxRequests: n,
		Interval:    time.Minute,
		Strategy:    "fixed",
	}
}

func PerHour(n int) *RateLimitConfig {
	return &RateLimitConfig{
		MaxRequests: n,
		Interval:    time.Hour,
		Strategy:    "fixed",
	}
}

func SlidingWindow(n int, t time.Duration) *RateLimitConfig {
	return &RateLimitConfig{
		MaxRequests: n,
		Interval:    t,
		Strategy:    "sliding",
	}
}

func TokenBucket(bucketSize int, refillRate float64) *RateLimitConfig {
	return &RateLimitConfig{
		TokenBucketSize: bucketSize,
		TokenRefillRate: refillRate,
		Strategy:        "token",
		Interval:        time.Minute, // Fallback
	}
}

// WithKeyPrefix adds a custom key prefix to the rate limit configuration
func (c *RateLimitConfig) WithKeyPrefix(prefix string) *RateLimitConfig {
	c.KeyPrefix = prefix
	return c
}

// WithCustomResponse adds a custom response handler when rate limited
func (c *RateLimitConfig) WithCustomResponse(fn func(*fiber.Ctx) error) *RateLimitConfig {
	c.CustomResponse = fn
	return c
}

// WithSkipFunction adds a function to skip rate limiting for certain requests
func (c *RateLimitConfig) WithSkipFunction(fn func(*fiber.Ctx) bool) *RateLimitConfig {
	c.Skip = fn
	return c
}

// WithRoleLimits adds role-based rate limits
func (c *RateLimitConfig) WithRoleLimits(limits ...RoleLimit) *RateLimitConfig {
	c.RoleLimits = limits
	return c
}

// WithGroupKey adds a function to group endpoints under the same rate limit bucket
func (c *RateLimitConfig) WithGroupKey(fn func(*fiber.Ctx) string) *RateLimitConfig {
	c.GroupKey = fn
	return c
}

// WithMetrics enables metrics collection
func (c *RateLimitConfig) WithMetrics() *RateLimitConfig {
	c.Metrics = true
	return c
}

// WithBypass enables bypass tokens
func (c *RateLimitConfig) WithBypass(fn func(*fiber.Ctx) bool) *RateLimitConfig {
	c.EnableBypass = true
	c.BypassCheck = fn
	return c
}
