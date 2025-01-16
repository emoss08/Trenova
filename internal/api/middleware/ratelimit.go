package middleware

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/infrastructure/cache/redis"
	tCtx "github.com/trenova-app/transport/internal/pkg/ctx"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

// RateLimitParams defines the dependencies required for rate limiting
type RateLimitParams struct {
	Logger *logger.Logger // Logger instance for recording rate limit events
	Redis  *redis.Client  // Redis client for storing rate limit data
}

// RateLimiter manages rate limiting functionality
type RateLimiter struct {
	l     *zerolog.Logger // Structured logger
	redis *redis.Client   // Redis client instance
}

// RateLimitConfig defines the configuration for rate limiting
type RateLimitConfig struct {
	MaxRequests int                   // Maximum number of requests allowed in the interval
	Interval    time.Duration         // Time window for rate limiting
	KeyPrefix   string                // Optional prefix for Redis keys
	Skip        func(*fiber.Ctx) bool // Optional function to skip rate limiting
}

// NewRateLimit creates a new rate limiter instance with the provided dependencies
func NewRateLimit(p RateLimitParams) *RateLimiter {
	log := p.Logger.
		With().
		Str("middleware", "rate-limit").
		Logger()

	return &RateLimiter{
		l:     &log,
		redis: p.Redis,
	}
}

// WithRateLimit wraps the provided handlers with rate limiting middleware
// Returns a slice of handlers with rate limiting applied
func (rl *RateLimiter) WithRateLimit(handlers []fiber.Handler, config RateLimitConfig) []fiber.Handler {
	limitHandler := rl.createLimitHandler(config)
	result := make([]fiber.Handler, 0, len(handlers)+1)
	result = append(result, limitHandler)
	result = append(result, handlers...)
	return result
}

// createLimitHandler creates a Fiber middleware handler that implements rate limiting
func (rl *RateLimiter) createLimitHandler(config RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip rate limiting if skip function is provided and returns true
		if config.Skip != nil && config.Skip(c) {
			return c.Next()
		}

		// Create a context-specific logger
		log := rl.l.With().
			Str("path", c.Path()).
			Str("method", c.Method()).
			Str("ip", c.IP()).
			Logger()

		// Build unique key for this rate limit
		key := rl.buildKey(c, config.KeyPrefix)
		allowed, remaining, reset, err := rl.checkLimit(c.Context(), key, config)
		if err != nil {
			if !eris.Is(err, redis.Nil) {
				log.Error().Err(err).Msg("rate limit check failed")
			}
			return c.Next() // Allow request on error for high availability
		}

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))

		if !allowed {
			log.Warn().
				Str("ip", c.IP()).
				Str("path", c.Path()).
				Int("limit", config.MaxRequests).
				Msg("rate limit exceeded")

			rlErr := errors.NewRateLimitError(c.Path(), "Rate limit exceeded. Please try again later.")

			return c.Status(fiber.StatusTooManyRequests).JSON(rlErr)
		}

		return c.Next()
	}
}

// checkLimit checks if the request is allowed based on the rate limit configuration
// Returns:
// - allowed: whether the request is allowed
// - remaining: number of requests remaining in the current window
// - reset: unix timestamp when the current window resets
// - error: any error that occurred during the check
func (rl *RateLimiter) checkLimit(ctx context.Context, key string, config RateLimitConfig) (bool, int, int64, error) {
	pipe := rl.redis.Pipeline()

	// Get current window and create if not exists
	windowKey := key + ":window"
	pipe.Get(ctx, windowKey)

	// Get current count
	countKey := key + ":count"
	pipe.Get(ctx, countKey)

	res, err := pipe.Exec(ctx)
	if err != nil && !eris.Is(err, redis.Nil) {
		return false, 0, 0, eris.Wrap(err, "failed to check rate limit")
	}

	now := time.Now().Unix()
	var windowStart int64
	var count int

	// Get or set window start time
	if res[0].Err() != nil {
		// New window starts now
		windowStart = now
		if err = rl.redis.Set(ctx, windowKey, windowStart, config.Interval); err != nil {
			return false, 0, 0, err
		}
	} else {
		windowStart, _ = strconv.ParseInt(res[0].(*redis.StringCmd).Val(), 10, 64)
	}

	// Calculate when this window resets
	resetTime := windowStart + int64(config.Interval.Seconds())

	// If we're past the reset time, start a new window
	if now >= resetTime {
		windowStart = now
		resetTime = windowStart + int64(config.Interval.Seconds())
		if err = rl.redis.Set(ctx, windowKey, windowStart, config.Interval); err != nil {
			return false, 0, 0, err
		}
		count = 0
	} else if res[1].Err() == nil {
		count, _ = strconv.Atoi(res[1].(*redis.StringCmd).Val())
	}

	// Check if we're allowed and calculate remaining requests
	allowed := count < config.MaxRequests
	remaining := config.MaxRequests - count - 1

	if allowed {
		// Increment counter for allowed requests
		if err = rl.redis.Set(ctx, countKey, count+1, config.Interval); err != nil {
			return false, 0, 0, err
		}
	} else {
		remaining = 0
	}

	return allowed, remaining, resetTime, nil
}

// buildKey generates a unique Redis key for the rate limit
// The key includes the prefix, HTTP method, path, IP address, and user ID (if available)
func (rl *RateLimiter) buildKey(c *fiber.Ctx, prefix string) string {
	parts := make([]string, 0, 5)

	if prefix == "" {
		parts = append(parts, "ratelimit")
	} else {
		parts = append(parts, prefix)
	}

	parts = append(parts, c.Method(), strings.Trim(c.Route().Path, "/"))

	ip := rl.getIP(c)
	parts = append(parts, ip)

	if userID, ok := c.Locals(tCtx.CTXUserID).(pulid.ID); ok && userID != "" {
		parts = append(parts, userID.String())
	}

	return strings.Join(parts, ":")
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

// PerSecond creates a rate limit configuration for requests per second
func PerSecond(n int) RateLimitConfig {
	return RateLimitConfig{
		MaxRequests: n,
		Interval:    time.Second,
	}
}

// Every creates a rate limit configuration for requests over a custom duration
func Every(n int, t time.Duration) RateLimitConfig {
	return RateLimitConfig{
		MaxRequests: n,
		Interval:    t,
	}
}

// PerMinute creates a rate limit configuration for requests per minute
func PerMinute(n int) RateLimitConfig {
	return RateLimitConfig{
		MaxRequests: n,
		Interval:    time.Minute,
	}
}

// PerHour creates a rate limit configuration for requests per hour
func PerHour(n int) RateLimitConfig {
	return RateLimitConfig{
		MaxRequests: n,
		Interval:    time.Hour,
	}
}
