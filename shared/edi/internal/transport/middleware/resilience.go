package middleware

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/transport"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/ratelimit"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	RequestsPerSecond int
	Burst             int
	PerPartner        bool // If true, rate limit per partner ID
}

// rateLimiterStore manages per-partner rate limiters
type rateLimiterStore struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	config   RateLimiterConfig
}

func newRateLimiterStore(config RateLimiterConfig) *rateLimiterStore {
	return &rateLimiterStore{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
	}
}

func (s *rateLimiterStore) getLimiter(key string) *rate.Limiter {
	s.mu.RLock()
	limiter, exists := s.limiters[key]
	s.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := s.limiters[key]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(s.config.RequestsPerSecond), s.config.Burst)
	s.limiters[key] = limiter
	
	// Cleanup old limiters if too many (prevent memory leak)
	if len(s.limiters) > 10000 {
		// Remove oldest entries (simple cleanup strategy)
		for k := range s.limiters {
			delete(s.limiters, k)
			if len(s.limiters) < 5000 {
				break
			}
		}
	}

	return limiter
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(config RateLimiterConfig) endpoint.Middleware {
	if config.PerPartner {
		store := newRateLimiterStore(config)
		return func(next endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, request interface{}) (interface{}, error) {
				// Extract partner ID from request
				key := "global"
				if req, ok := request.(interface{ GetPartnerID() string }); ok {
					if partnerID := req.GetPartnerID(); partnerID != "" {
						key = partnerID
					}
				}

				limiter := store.getLimiter(key)
				if !limiter.Allow() {
					return nil, transport.NewServiceError(
						transport.ErrorTypeRateLimit,
						fmt.Sprintf("rate limit exceeded for %s", key),
					)
				}

				return next(ctx, request)
			}
		}
	}

	// Global rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		limited := ratelimit.NewErroringLimiter(limiter)(next)
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			return limited(ctx, request)
		}
	}
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Name             string
	MaxRequests      uint32  // Maximum requests in half-open state
	Interval         time.Duration // Time window for counting failures
	Timeout          time.Duration // Timeout for open state
	FailureRatio     float64 // Failure ratio threshold
	ConsecutiveFails uint32  // Consecutive failures to open
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:             name,
		MaxRequests:      3,
		Interval:         10 * time.Second,
		Timeout:          30 * time.Second,
		FailureRatio:     0.6,
		ConsecutiveFails: 5,
	}
}

// CircuitBreakerMiddleware creates a circuit breaker middleware
func CircuitBreakerMiddleware(config CircuitBreakerConfig) endpoint.Middleware {
	settings := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= config.FailureRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// Log state changes
			fmt.Printf("Circuit breaker %s: %s -> %s\n", name, from, to)
		},
	}

	cb := gobreaker.NewCircuitBreaker(settings)
	
	return circuitbreaker.Gobreaker(cb)
}

// RetryMiddleware adds retry logic with exponential backoff
func RetryMiddleware(maxAttempts int, timeout time.Duration) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			var lastErr error
			
			for attempt := 0; attempt < maxAttempts; attempt++ {
				if attempt > 0 {
					// Exponential backoff: 100ms, 200ms, 400ms, 800ms...
					backoff := time.Duration(100*1<<uint(attempt-1)) * time.Millisecond
					
					select {
					case <-time.After(backoff):
						// Continue with retry
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}

				// Create timeout context for this attempt
				attemptCtx, cancel := context.WithTimeout(ctx, timeout)
				response, err := next(attemptCtx, request)
				cancel()

				if err == nil {
					return response, nil
				}

				// Check if error is retryable
				if !isRetryable(err) {
					return response, err
				}

				lastErr = err
			}

			return nil, fmt.Errorf("max retry attempts (%d) exceeded: %w", maxAttempts, lastErr)
		}
	}
}

// isRetryable determines if an error should trigger a retry
func isRetryable(err error) bool {
	// Don't retry validation errors
	var validationErr transport.ValidationError
	if errors.As(err, &validationErr) {
		return false
	}

	// Don't retry business logic errors
	var serviceErr transport.ServiceError
	if errors.As(err, &serviceErr) {
		switch serviceErr.Type {
		case transport.ErrorTypeValidation, 
		     transport.ErrorTypeNotFound,
		     transport.ErrorTypeConflict,
		     transport.ErrorTypeUnauthorized,
		     transport.ErrorTypeForbidden:
			return false
		}
	}

	// Retry on service unavailable, timeouts, etc.
	return true
}

// TimeoutMiddleware adds a timeout to endpoint execution
func TimeoutMiddleware(timeout time.Duration) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			type response struct {
				res interface{}
				err error
			}

			done := make(chan response, 1)

			go func() {
				res, err := next(ctx, request)
				done <- response{res, err}
			}()

			select {
			case <-ctx.Done():
				return nil, transport.NewServiceError(
					transport.ErrorTypeUnavailable,
					"request timeout exceeded",
				)
			case r := <-done:
				return r.res, r.err
			}
		}
	}
}

// BulkheadMiddleware limits concurrent requests
func BulkheadMiddleware(maxConcurrent int) endpoint.Middleware {
	sem := make(chan struct{}, maxConcurrent)
	
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				return next(ctx, request)
			default:
				return nil, transport.NewServiceError(
					transport.ErrorTypeUnavailable,
					"service at capacity",
				)
			}
		}
	}
}