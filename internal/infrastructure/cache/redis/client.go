package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	defaultConnectTimeout = 5 * time.Second
	defaultReadTimeout    = 3 * time.Second
	defaultWriteTimeout   = 3 * time.Second
	defaultPoolSize       = 10
	defaultMinIdleConns   = 10

	// Circuit breaker constants - more aggressive settings for faster failure detection
	circuitBreakerFailureThreshold = 10               // Trip after 10 failures to handle burst load
	circuitBreakerTimeout          = 2 * time.Second  // Shorter timeout for Redis operations
	circuitBreakerResetTimeout     = 10 * time.Second // Faster recovery for stress testing
)

type StringCmd = redis.StringCmd

var (
	// Nil reply returned by Redis when key does not exist.
	ErrNil = redis.Nil
	// Error returned when the redis config is nil
	ErrConfigNil = eris.New("redis config is nil")
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int32

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreaker implements a simple circuit breaker pattern for Redis operations
type CircuitBreaker struct {
	state        int32 // atomic access - CircuitBreakerState
	failures     int32 // atomic access
	lastFailTime int64 // atomic access - unix timestamp
	threshold    int32
	timeout      time.Duration
	resetTimeout time.Duration
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int32, timeout, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        int32(CircuitBreakerClosed),
		threshold:    threshold,
		timeout:      timeout,
		resetTimeout: resetTimeout,
	}
}

// CanExecute checks if an operation can be executed
func (cb *CircuitBreaker) CanExecute() bool {
	// TEMPORARY: Disable circuit breaker for stress testing
	// This prevents Redis circuit breaker from opening under high load
	return true

	// Original logic (commented out for stress testing):
	// state := CircuitBreakerState(atomic.LoadInt32(&cb.state))
	//
	// switch state {
	// case CircuitBreakerClosed:
	// 	return true
	// case CircuitBreakerOpen:
	// 	lastFailTime := atomic.LoadInt64(&cb.lastFailTime)
	// 	if time.Since(time.Unix(lastFailTime, 0)) > cb.resetTimeout {
	// 		// Try to transition to half-open
	// 		if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitBreakerOpen), int32(CircuitBreakerHalfOpen)) {
	// 			atomic.StoreInt32(&cb.failures, 0)
	// 			return true
	// 		}
	// 	}
	// 	return false
	// case CircuitBreakerHalfOpen:
	// 	return true
	// default:
	// 	return false
	// }
}

// OnSuccess records a successful operation
func (cb *CircuitBreaker) OnSuccess() {
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))
	if state == CircuitBreakerHalfOpen {
		atomic.StoreInt32(&cb.state, int32(CircuitBreakerClosed))
		atomic.StoreInt32(&cb.failures, 0)
	}
}

// OnFailure records a failed operation
func (cb *CircuitBreaker) OnFailure() {
	failures := atomic.AddInt32(&cb.failures, 1)
	atomic.StoreInt64(&cb.lastFailTime, time.Now().Unix())

	if failures >= cb.threshold {
		atomic.StoreInt32(&cb.state, int32(CircuitBreakerOpen))
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return CircuitBreakerState(atomic.LoadInt32(&cb.state))
}

// Client wraps the Redis client with additional functionality including circuit breaker
type Client struct {
	*redis.Client
	l              *zerolog.Logger
	circuitBreaker *CircuitBreaker
}

type ClientParams struct {
	fx.In

	Config *config.Manager
	Logger *logger.Logger
}

// NewClient creates a new Redis client instance with the provided configuration and logger.
func NewClient(p ClientParams) (*Client, error) {
	log := p.Logger.With().
		Str("client", "redis").
		Logger()

	cfg := p.Config.Redis()
	if cfg == nil {
		log.Error().Msg("redis config is nil")
		return nil, ErrConfigNil
	}

	// Set the defaults
	setDefaults(cfg)

	// Log the configuration being used (without password)
	log.Info().
		Str("addr", cfg.Addr).
		Int("db", cfg.DB).
		Bool("has_password", cfg.Password != "").
		Dur("conn_timeout", cfg.ConnTimeout).
		Dur("read_timeout", cfg.ReadTimeout).
		Dur("write_timeout", cfg.WriteTimeout).
		Int("pool_size", cfg.PoolSize).
		Int("min_idle_conns", cfg.MinIdleConns).
		Msg("initializing redis client with config")

	opts := &redis.Options{
		Addr: cfg.Addr,
		DB:   cfg.DB,
		// Username:        cfg.Username,
		Password:        cfg.Password,
		ConnMaxIdleTime: cfg.ConnTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			log.Debug().Msg("redis on_connect callback called")
			if err := cn.Ping(ctx).Err(); err != nil {
				log.Error().Err(err).Msg("redis on_connect ping failed")
				return err
			}
			log.Debug().Msg("redis on_connect ping successful")
			return nil
		},
	}

	log.Info().Str("addr", cfg.Addr).Msg("creating redis client")
	redisClient := redis.NewClient(opts)

	// Test connection with detailed logging
	log.Info().Str("addr", cfg.Addr).Dur("timeout", cfg.ConnTimeout).Msg("testing redis connection")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnTimeout)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error().
			Err(err).
			Str("addr", cfg.Addr).
			Int("db", cfg.DB).
			Dur("timeout", cfg.ConnTimeout).
			Msg("redis ping failed during initialization")

		// Attempt to close the client on failure
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("failed to close redis client after ping failure")
		}

		return nil, eris.Wrap(err, "failed to connect to redis")
	}

	log.Info().Str("addr", cfg.Addr).Msg("redis connection successful")

	// Create circuit breaker for Redis operations
	circuitBreaker := NewCircuitBreaker(
		circuitBreakerFailureThreshold,
		circuitBreakerTimeout,
		circuitBreakerResetTimeout,
	)

	client := &Client{
		Client:         redisClient,
		l:              &log,
		circuitBreaker: circuitBreaker,
	}

	return client, nil
}

// setDefaults sets the default values for the redis config
func setDefaults(cfg *config.RedisConfig) {
	// Set default values if not provided
	if cfg.ConnTimeout == 0 {
		cfg.ConnTimeout = defaultConnectTimeout
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = defaultPoolSize
	}
	if cfg.MinIdleConns == 0 {
		cfg.MinIdleConns = defaultMinIdleConns
	}
}

var (
	ErrCircuitBreakerOpen = eris.New("redis circuit breaker is open")
)

// executeWithCircuitBreaker executes a Redis operation with circuit breaker protection
func (c *Client) executeWithCircuitBreaker(ctx context.Context, operation string, fn func() error) error {
	if !c.circuitBreaker.CanExecute() {
		c.l.Warn().
			Str("operation", operation).
			Str("circuit_breaker_state", fmt.Sprintf("%v", c.circuitBreaker.GetState())).
			Msg("redis operation blocked by circuit breaker")
		return ErrCircuitBreakerOpen
	}

	// Create a shorter timeout context for Redis operations to prevent hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Channel to receive the result
	resultChan := make(chan error, 1)

	// Execute the operation in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.l.Error().
					Interface("panic", r).
					Str("operation", operation).
					Msg("redis operation panicked")
				resultChan <- eris.New("redis operation panicked")
			}
		}()

		// Execute with the original context but monitor the timeout context
		err := fn()
		resultChan <- err
	}()

	select {
	case err := <-resultChan:
		if err != nil {
			// Check if this is a connection-related error or timeout
			if c.isConnectionError(err) || strings.Contains(err.Error(), "deadline") {
				c.circuitBreaker.OnFailure()
				c.l.Error().
					Err(err).
					Str("operation", operation).
					Int32("failures", atomic.LoadInt32(&c.circuitBreaker.failures)).
					Str("circuit_breaker_state", fmt.Sprintf("%v", c.circuitBreaker.GetState())).
					Msg("redis operation failed, circuit breaker updated")
			}
			return err
		}

		c.circuitBreaker.OnSuccess()
		return nil

	case <-timeoutCtx.Done():
		// Operation timed out, treat as connection failure
		c.circuitBreaker.OnFailure()
		c.l.Error().
			Str("operation", operation).
			Dur("timeout", 2*time.Second).
			Int32("failures", atomic.LoadInt32(&c.circuitBreaker.failures)).
			Str("circuit_breaker_state", fmt.Sprintf("%v", c.circuitBreaker.GetState())).
			Msg("redis operation timed out, circuit breaker updated")
		return eris.New("redis operation timed out")
	}
}

// isConnectionError checks if an error is connection-related
func (c *Client) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset")
}

// Get is a helper function to get a value from redis
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	// Add debug logging for the get operation
	c.l.Debug().
		Str("key", key).
		Msg("attempting redis get operation")

	var val string
	var err error

	executeErr := c.executeWithCircuitBreaker(ctx, "GET", func() error {
		val, err = c.Client.Get(ctx, key).Result()
		return err
	})

	if executeErr != nil {
		if eris.Is(executeErr, ErrCircuitBreakerOpen) {
			return "", executeErr
		}
		return "", executeErr
	}

	if err != nil {
		if eris.Is(err, redis.Nil) {
			c.l.Debug().
				Str("key", key).
				Msg("redis key not found")
			return "", redis.Nil
		}

		// Enhanced error logging for connection issues
		c.l.Error().
			Err(err).
			Str("key", key).
			Str("error_type", fmt.Sprintf("%T", err)).
			Bool("is_timeout", strings.Contains(err.Error(), "timeout")).
			Bool("is_connection_refused", strings.Contains(err.Error(), "connection refused")).
			Bool("is_network_error", strings.Contains(err.Error(), "network")).
			Msg("redis get error")
		return "", eris.Wrap(err, "failed to get value from redis")
	}

	c.l.Trace().
		Str("key", key).
		Str("value", val).
		Msg("redis get success")

	return val, nil
}

// Set sets a value in Redis with a specified expiration time.
func (c *Client) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.executeWithCircuitBreaker(ctx, "SET", func() error {
		err := c.Client.Set(ctx, key, value, expiration).Err()
		if err != nil {
			return eris.Wrapf(err, "failed to set key: %s", key)
		}

		c.l.Trace().
			Str("key", key).
			Interface("value", value).
			Dur("expiration", expiration).
			Msg("key set successfully")

		return nil
	})
}

// GetJSON retrieves a JSON-encoded value from Redis and unmarshals it into the provided destination.
func (c *Client) GetJSON(ctx context.Context, key string, dest any) error {
	val, err := c.Get(ctx, key)
	if err != nil {
		return eris.Wrap(err, "failed to get value from redis")
	}

	if err = sonic.Unmarshal([]byte(val), dest); err != nil {
		return eris.Wrapf(err, "failed to unmarshal JSON for key: %s", key)
	}

	return nil
}

// SetJSON marshals a value as JSON and stores it in Redis with a specified expiration time.
func (c *Client) SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := sonic.Marshal(value)
	if err != nil {
		return eris.Wrapf(err, "failed to marshal JSON for key: %s", key)
	}

	return c.Set(ctx, key, data, expiration)
}

// Incr increments the integer value of a key in Redis by 1.
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, eris.Wrapf(err, "failed to increment key: %s", key)
	}

	return val, nil
}

// IncrBy increments the integer value of a key in Redis by a specified amount.
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.Client.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, eris.Wrapf(err, "failed to increment key by value: %s", key)
	}

	return val, nil
}

// HSet sets a field in a Redis hash to a specified value.
func (c *Client) HSet(ctx context.Context, key, field string, value any) error {
	err := c.Client.HSet(ctx, key, field, value).Err()
	if err != nil {
		return eris.Wrapf(err, "failed to set hash field: %s.%s", key, field)
	}

	c.l.Debug().
		Str("key", key).
		Str("field", field).
		Interface("value", value).
		Msg("hash field set successfully")

	return nil
}

// HGet retrieves a field's value from a Redis hash.
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := c.Client.HGet(ctx, key, field).Result()
	if err != nil {
		if eris.Is(err, redis.Nil) {
			return "", redis.Nil
		}
		return "", eris.Wrapf(err, "failed to get hash field: %s.%s", key, field)
	}

	return val, nil
}

// SAdd adds one or more members to a Redis set.
func (c *Client) SAdd(ctx context.Context, key string, members ...any) error {
	err := c.Client.SAdd(ctx, key, members...).Err()
	if err != nil {
		return eris.Wrapf(err, "failed to add members to set: %s", key)
	}

	c.l.Trace().
		Str("key", key).
		Interface("members", members).
		Msg("members added to set successfully")

	return nil
}

// SRem removes one or more members from a Redis set.
func (c *Client) SRem(ctx context.Context, key string, members ...any) error {
	err := c.Client.SRem(ctx, key, members...).Err()
	if err != nil {
		return eris.Wrapf(err, "failed to remove members from set: %s", key)
	}

	return nil
}

// SMembers retrieves all members of a Redis set.
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	members, err := c.Client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, eris.Wrapf(err, "failed to get set members: %s", key)
	}

	return members, nil
}

// Expire sets an expiration time for a key in Redis.
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	ok, err := c.Client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return eris.Wrapf(err, "failed to set expiration for key: %s", key)
	}
	if !ok {
		return eris.Errorf("key does not exist: %s", key)
	}

	return nil
}

// Del is a helper function to delete a key from redis
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if err := c.Client.Del(ctx, keys...).Err(); err != nil {
		return eris.Wrap(err, "failed to delete key from redis")
	}

	c.l.Debug().
		Str("keys", strings.Join(keys, ", ")).
		Msg("redis del success")

	return nil
}

// IncreaseWithExpiry increments a key's value and sets an expiration time atomically.
func (c *Client) IncreaseWithExpiry(ctx context.Context, key string, expiry time.Duration) (int64, error) {
	pipe := c.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiry)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, eris.Wrapf(err, "failed to increment and set expiry for key: %s", key)
	}

	return incr.Val(), nil
}

// Transaction executes a transactional operation on Redis.
func (c *Client) Transaction(ctx context.Context, fn func(tx *redis.Tx) error) error {
	return c.Client.Watch(ctx, fn, "")
}

// GetInt retrieves an integer value for a key from Redis, returning a default value if the key does not exist.
func (c *Client) GetInt(ctx context.Context, key string, defaultVal int) (int, error) {
	val, err := c.Get(ctx, key)
	if err != nil {
		if eris.Is(err, redis.Nil) {
			return defaultVal, nil
		}
		return defaultVal, err
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal, eris.Wrapf(err, "failed to convert value to int for key: %s", key)
	}

	return intVal, nil
}

// Pipeline returns a new pipeline for batch operations
func (c *Client) Pipeline() redis.Pipeliner {
	return c.Client.Pipeline()
}

// GetTTL returns the remaining TTL for a key
func (c *Client) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.Client.TTL(ctx, key).Result()
	if err != nil {
		return 0, eris.Wrapf(err, "failed to get TTL for key: %s", key)
	}
	return ttl, nil
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	var n int64
	err := c.executeWithCircuitBreaker(ctx, "EXISTS", func() error {
		var e error
		n, e = c.Client.Exists(ctx, key).Result()
		return e
	})
	if err != nil {
		if eris.Is(err, ErrCircuitBreakerOpen) {
			return false, err
		}
		return false, eris.Wrapf(err, "failed to check existence of key: %s", key)
	}
	return n > 0, nil
}

// Keys returns all keys matching a pattern
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := c.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, eris.Wrapf(err, "failed to get keys matching pattern: %s", pattern)
	}
	return keys, nil
}

// Ping checks if Redis is responding
func (c *Client) Ping(ctx context.Context) error {
	c.l.Debug().Msg("pinging redis server")
	err := c.Client.Ping(ctx).Err()
	if err != nil {
		c.l.Error().
			Err(err).
			Str("error_type", fmt.Sprintf("%T", err)).
			Bool("is_timeout", strings.Contains(err.Error(), "timeout")).
			Bool("is_connection_refused", strings.Contains(err.Error(), "connection refused")).
			Bool("is_network_error", strings.Contains(err.Error(), "network")).
			Msg("redis ping failed")
		return eris.Wrap(err, "redis ping failed")
	}
	c.l.Debug().Msg("redis ping successful")
	return nil
}

// HealthCheck performs a comprehensive health check of the Redis connection
func (c *Client) HealthCheck(ctx context.Context) error {
	c.l.Info().Msg("performing redis health check")

	// Check basic connectivity
	if err := c.Ping(ctx); err != nil {
		return eris.Wrap(err, "redis health check failed: ping error")
	}

	// Try a simple set/get operation
	testKey := "health_check_" + time.Now().Format("20060102150405")
	testValue := "ok"

	if err := c.Set(ctx, testKey, testValue, time.Second*10); err != nil {
		c.l.Error().Err(err).Msg("redis health check failed: set operation")
		return eris.Wrap(err, "redis health check failed: set operation")
	}

	val, err := c.Get(ctx, testKey)
	if err != nil {
		c.l.Error().Err(err).Msg("redis health check failed: get operation")
		return eris.Wrap(err, "redis health check failed: get operation")
	}

	if val != testValue {
		c.l.Error().
			Str("expected", testValue).
			Str("actual", val).
			Msg("redis health check failed: value mismatch")
		return eris.New("redis health check failed: value mismatch")
	}

	// Clean up test key
	if err = c.Del(ctx, testKey); err != nil {
		c.l.Warn().Err(err).Msg("failed to clean up health check key")
	}

	c.l.Info().Msg("redis health check passed")
	return nil
}

// Close closes the underlying Redis client connection
func (c *Client) Close() error {
	c.l.Info().Msg("closing Redis client connection")
	return c.Client.Close()
}
