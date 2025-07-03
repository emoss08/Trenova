package middleware

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	tCtx "github.com/emoss08/trenova/internal/pkg/appctx"
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
	LastBlocked time.Time `json:"lastBlocked"`
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
func (rl *RateLimiter) WithRateLimit(
	handlers []fiber.Handler,
	config *RateLimitConfig,
) []fiber.Handler {
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
		if rl.shouldSkipRateLimiting(c, config) {
			return c.Next()
		}

		// Create a context-specific logger
		log := rl.createContextLogger(c)

		// Get appropriate rate limit for this user/role
		maxRequests, interval := rl.getRoleLimits(c, config)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Context(), config.Timeout)
		defer cancel()

		// Build unique key for this rate limit
		key := rl.generateRateLimitKey(c, config)

		// Check rate limit according to strategy
		allowed, remaining, reset, err := rl.checkRateLimit(ctx, key, maxRequests, interval, config)
		// Handle errors in rate limit check
		if err != nil {
			return rl.handleRateLimitError(c, config, err)
		}

		// * Reset time edge cases are now handled atomically in the Lua script

		// Set rate limit headers
		rl.setRateLimitHeaders(c, maxRequests, remaining, reset, config.Strategy)

		// Handle the case when rate limit is exceeded
		if !allowed {
			return rl.handleRateLimitExceeded(ctx, c, key, config, maxRequests, &log)
		}

		// Update metrics if enabled
		if config.Metrics {
			rl.updateAllowMetrics(ctx, key)
		}

		return c.Next()
	}
}

// shouldSkipRateLimiting determines if rate limiting should be skipped for this request
func (rl *RateLimiter) shouldSkipRateLimiting(c *fiber.Ctx, config *RateLimitConfig) bool {
	// Skip if skip function is provided and returns true
	if config.Skip != nil && config.Skip(c) {
		return true
	}

	// Skip if bypass is enabled and check returns true
	if config.EnableBypass && config.BypassCheck != nil && config.BypassCheck(c) {
		c.Set("X-RateLimit-Bypassed", "true")
		return true
	}

	return false
}

// createContextLogger creates a context-specific logger with request details
func (rl *RateLimiter) createContextLogger(c *fiber.Ctx) zerolog.Logger {
	return rl.l.With().
		Str("path", c.Path()).
		Str("method", c.Method()).
		Str("ip", c.IP()).
		Logger()
}

// generateRateLimitKey generates the key for rate limiting
func (rl *RateLimiter) generateRateLimitKey(c *fiber.Ctx, config *RateLimitConfig) string {
	if config.GroupKey != nil {
		return config.GroupKey(c)
	}
	return config.KeyGenerator(c)
}

// checkRateLimit checks if the request is allowed based on the rate limit strategy
func (rl *RateLimiter) checkRateLimit(
	ctx context.Context,
	key string,
	maxRequests int,
	interval time.Duration,
	config *RateLimitConfig,
) (isAllowed bool, remaining int, reset int64, err error) {
	switch config.Strategy {
	case "sliding":
		return rl.checkSlidingWindowLimit(ctx, key, maxRequests, interval)
	case "token":
		return rl.checkTokenBucketLimit(ctx, key, config)
	default:
		return rl.checkFixedWindowLimit(ctx, key, maxRequests, interval)
	}
}

// handleRateLimitError handles errors that occur during rate limit checking
func (rl *RateLimiter) handleRateLimitError(
	c *fiber.Ctx,
	config *RateLimitConfig,
	err error,
) error {
	log := rl.createContextLogger(c)

	// Check if this is a circuit breaker error
	if strings.Contains(err.Error(), "circuit breaker is open") {
		log.Warn().
			Err(err).
			Str("fallback_behavior", config.FallbackBehavior).
			Msg("rate limit check failed due to circuit breaker, applying fallback")
	} else {
		log.Error().Err(err).Msg("rate limit check failed")
	}

	// Determine what to do on failure
	if config.FallbackBehavior == "deny" {
		rlErr := errors.NewRateLimitError(
			c.Path(),
			"Rate limiting unavailable. Please try again later.",
		)
		return c.Status(fiber.StatusServiceUnavailable).JSON(rlErr)
	}

	// For circuit breaker failures, apply more permissive fallback
	if strings.Contains(err.Error(), "circuit breaker is open") {
		// Set conservative rate limit headers to indicate degraded service
		maxRequests, interval := rl.getRoleLimits(c, config)
		now := time.Now().Unix()
		reset := now + int64(interval.Seconds())

		rl.setRateLimitHeaders(c, maxRequests, maxRequests/2, reset, "fallback")

		log.Info().
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Msg("allowing request with fallback rate limiting due to Redis unavailability")
	}

	return c.Next() // Allow request on error for high availability
}



// setRateLimitHeaders sets the rate limit headers in the response
func (rl *RateLimiter) setRateLimitHeaders(
	c *fiber.Ctx,
	maxRequests, remaining int,
	reset int64,
	strategy string,
) {
	c.Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
	c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
	c.Set("X-RateLimit-Strategy", strategy)
}

// handleRateLimitExceeded handles the case when rate limit is exceeded
func (rl *RateLimiter) handleRateLimitExceeded(
	ctx context.Context,
	c *fiber.Ctx,
	key string,
	config *RateLimitConfig,
	maxRequests int,
	log *zerolog.Logger,
) error {
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

// resetCounter resets the rate limit counter based on the strategy
func (rl *RateLimiter) resetCounter(ctx context.Context, key, strategy string) error {
	var err error
	switch strategy {
	case "sliding":
		// For sliding window, just delete the key
		err = rl.redis.Del(ctx, key)
	case "token":
		// For token bucket, reset to full bucket size
		if err = rl.redis.Del(ctx, key); err != nil {
			return err
		}
		err = rl.redis.Del(ctx, key+":timestamp")
	default:
		// For fixed window, delete the new hash key and old keys for migration
		if err = rl.redis.Del(ctx, key+":data"); err != nil {
			return err
		}
		// Clean up old format keys if they exist
		rl.redis.Del(ctx, key, key+":window")
	}

	if err != nil {
		rl.l.Error().Err(err).Str("key", key).Msg("failed to reset counter")
	}

	return err
}


// deleteRedisKeyHandlingCircuitBreaker attempts to delete a Redis key and handles circuit breaker errors.
func (rl *RateLimiter) deleteRedisKeyHandlingCircuitBreaker(
	ctx context.Context,
	key, errMsgOnFailure string,
) error {
	if err := rl.redis.Del(ctx, key); err != nil {
		// If circuit breaker is open, ignore cleanup errors
		if strings.Contains(err.Error(), "circuit breaker is open") {
			rl.l.Debug().Str("key", key).Msg("skipping key cleanup due to circuit breaker")
			return nil // Treat as success or non-critical failure
		}
		return eris.Wrap(err, errMsgOnFailure)
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
func (rl *RateLimiter) checkFixedWindowLimit(
	ctx context.Context,
	key string,
	maxRequests int,
	interval time.Duration,
) (isAllowed bool, remaining int, reset int64, err error) {
	windowKey := key + ":window"

	// * Consistency is now maintained atomically in the Lua script

	now := time.Now().Unix()

	result, err := rl.scriptLoader.EvalSHA(
		ctx,
		"fixed_window",
		[]string{key, windowKey},
		maxRequests,
		int(interval.Seconds()),
		now,
	)
	if err != nil {
		return false, 0, 0, eris.Wrap(err, "failed to execute fixed window script")
	}

	results, ok := result.([]any)
	if !ok || len(results) != 3 {
		return false, 0, 0, eris.New("invalid response from fixed window script")
	}

	// Parse results
	allowed, ok := results[0].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid allowed value from fixed window script")
	}
	remainingVal, ok := results[1].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid remaining value from fixed window script")
	}
	resetVal, ok := results[2].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid reset value from fixed window script")
	}

	isAllowed = allowed == 1
	remaining = int(remainingVal)
	reset = resetVal

	// * Clock skew is now handled in the Lua script with a grace period

	return isAllowed, remaining, reset, nil
}

// checkSlidingWindowLimit implements sliding window rate limiting
func (rl *RateLimiter) checkSlidingWindowLimit(
	ctx context.Context,
	key string,
	maxRequests int,
	interval time.Duration,
) (isAllowed bool, remaining int, reset int64, err error) {
	now := time.Now().Unix()

	result, err := rl.scriptLoader.EvalSHA(
		ctx,
		"sliding_window",
		[]string{key},
		maxRequests,
		int(interval.Seconds()),
		now,
	)
	if err != nil {
		return false, 0, 0, eris.Wrap(err, "failed to execute sliding window script")
	}

	results, ok := result.([]any)
	if !ok || len(results) != 3 {
		return false, 0, 0, eris.New("invalid response from sliding window script")
	}

	// Parse results
	allowed, ok := results[0].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid allowed value from sliding window script")
	}
	remainingVal, ok := results[1].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid remaining value from sliding window script")
	}
	resetVal, ok := results[2].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid reset value from sliding window script")
	}

	isAllowed = allowed == 1
	remaining = int(remainingVal)
	reset = resetVal

	return isAllowed, remaining, reset, nil
}

// checkTokenBucketLimit implements token bucket rate limiting
func (rl *RateLimiter) checkTokenBucketLimit(
	ctx context.Context,
	key string,
	config *RateLimitConfig,
) (isAllowed bool, remaining int, reset int64, err error) {
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
	allowed, ok := results[0].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid allowed value from token bucket script")
	}
	remainingVal, ok := results[1].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid remaining value from token bucket script")
	}
	resetVal, ok := results[2].(int64)
	if !ok {
		return false, 0, 0, eris.New("invalid reset value from token bucket script")
	}

	isAllowed = allowed == 1
	remaining = int(remainingVal)
	reset = resetVal

	return isAllowed, remaining, reset, nil
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
		if err = sonic.Unmarshal([]byte(val), metrics); err == nil {
			metrics.Allowed++
		}
	}

	// Marshal and save metrics
	if data, dErr := sonic.Marshal(metrics); dErr == nil {
		if err = rl.redis.Set(ctx, metricsKey, string(data), 24*time.Hour); err != nil {
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
		if err = sonic.Unmarshal([]byte(val), metrics); err == nil {
			metrics.Blocked++
			metrics.LastBlocked = time.Now()
		}
	}

	// Marshal and save metrics
	if data, dErr := sonic.Marshal(metrics); dErr == nil {
		if err = rl.redis.Set(ctx, metricsKey, string(data), 24*time.Hour); err != nil {
			rl.l.Error().Err(err).Msg("failed to update block metrics")
		}
	}
}

// GetMetrics retrieves rate limit metrics for a key
func (rl *RateLimiter) GetMetrics(ctx context.Context, key string) (*RateLimitMetrics, error) {
	metricsKey := key + ":metrics"

	val, err := rl.redis.Get(ctx, metricsKey)
	if err != nil {
		if eris.Is(err, redis.ErrNil) {
			return &RateLimitMetrics{}, nil
		}
		return nil, err
	}

	metrics := &RateLimitMetrics{}
	if err = sonic.Unmarshal([]byte(val), metrics); err != nil {
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
	// Use SCAN to find all matching keys (including new :data suffix)
	patterns := []string{keyPattern + "*", keyPattern + "*:data"}
	
	for _, pattern := range patterns {
		iter := rl.redis.Scan(ctx, 0, pattern, 100).Iterator()

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
