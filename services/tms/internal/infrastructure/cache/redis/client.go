/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/telemetry"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
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

type Z = redis.Z

// Client wraps the Redis client with additional functionality including circuit breaker
type Client struct {
	*redis.Client
	l              *zerolog.Logger
	circuitBreaker *CircuitBreaker
	errBuilder     oops.OopsErrorBuilder
}

type ClientParams struct {
	fx.In

	Config           *config.Manager
	Logger           *logger.Logger
	TelemetryMetrics *telemetry.Metrics `name:"telemetryMetrics" optional:"true"`
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

		return nil, oops.In("redis_client").
			With("addr", cfg.Addr).
			With("db", cfg.DB).
			With("timeout", cfg.ConnTimeout).
			With("error", err).
			Tags("redis_client").
			Wrap(err)
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
		errBuilder:     oops.With("redis_client").Tags("redis_client"),
		circuitBreaker: circuitBreaker,
	}

	// Add telemetry instrumentation if available
	if p.TelemetryMetrics != nil {
		instrumentation := telemetry.NewCacheInstrumentation(
			p.Config.App().Name,
			p.TelemetryMetrics,
		)
		instrumentation.InstrumentRedis(redisClient)
		log.Info().Msg("Redis telemetry instrumentation added")
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

// executeWithCircuitBreaker executes a Redis operation with circuit breaker protection
func (c *Client) executeWithCircuitBreaker(
	ctx context.Context,
	operation string,
	fn func() error,
) error {
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
			return c.errBuilder.With("operation", operation).With("error", err).Wrap(err)
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
func (c *Client) Get(ctx context.Context, key string) (val string, err error) {
	c.l.Debug().
		Str("key", key).
		Msg("attempting redis get operation")

	executeErr := c.executeWithCircuitBreaker(ctx, "GET", func() error {
		val, err = c.Client.Get(ctx, key).Result()
		return err
	})

	if executeErr != nil {
		if eris.Is(executeErr, ErrCircuitBreakerOpen) {
			return "", c.errBuilder.With("operation", "GET").With("key", key).Wrap(executeErr)
		}
		return "", c.errBuilder.With("operation", "GET").With("key", key).Wrap(executeErr)
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
		return "", c.errBuilder.With("operation", "GET").With("key", key).Wrap(err)
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
			return c.errBuilder.With("operation", "SET").With("key", key).Wrap(err)
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
func (c *Client) GetJSON(ctx context.Context, path, key string, dest any) error {
	return c.executeWithCircuitBreaker(ctx, "JSON.GET", func() error {
		val, err := c.Client.JSONGet(ctx, key, path).Result()
		if err != nil {
			if eris.Is(err, redis.Nil) {
				c.l.Debug().
					Str("key", key).
					Msg("redis JSON key not found")
				return redis.Nil
			}
			return c.errBuilder.With("operation", "JSON.GET").With("key", key).Wrap(err)
		}

		if err = sonic.Unmarshal([]byte(val), dest); err != nil {
			return c.errBuilder.With("operation", "JSON.GET").With("key", key).Wrap(err)
		}

		return nil
	})
}

// SetJSON marshals a value as JSON and stores it in Redis with a specified expiration time.
func (c *Client) SetJSON(
	ctx context.Context,
	path string,
	key string,
	value any,
	expiration time.Duration,
) error {
	return c.executeWithCircuitBreaker(ctx, "JSON.SET", func() error {
		data, err := sonic.Marshal(value)
		if err != nil {
			return c.errBuilder.With("operation", "JSON.SET").With("key", key).Wrap(err)
		}

		// Use Redis JSON.SET command to store as proper JSON
		err = c.Client.JSONSet(ctx, key, path, string(data)).Err()
		if err != nil {
			return c.errBuilder.With("operation", "JSON.SET").With("key", key).Wrap(err)
		}

		// Set expiration if specified
		if expiration > 0 {
			err = c.Client.Expire(ctx, key, expiration).Err()
			if err != nil {
				return c.errBuilder.With("operation", "JSON.SET").With("key", key).Wrap(err)
			}
		}

		c.l.Trace().
			Str("key", key).
			Interface("value", value).
			Dur("expiration", expiration).
			Msg("JSON key set successfully")

		return nil
	})
}

// Incr increments the integer value of a key in Redis by 1.
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, c.errBuilder.With("operation", "INCR").With("key", key).Wrap(err)
	}

	return val, nil
}

// IncrBy increments the integer value of a key in Redis by a specified amount.
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.Client.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, c.errBuilder.With("operation", "INCR_BY").With("key", key).Wrap(err)
	}

	return val, nil
}

// HSet sets a field in a Redis hash to a specified value.
func (c *Client) HSet(ctx context.Context, key, field string, value any) error {
	err := c.Client.HSet(ctx, key, field, value).Err()
	if err != nil {
		return c.errBuilder.With("operation", "HSET").
			With("key", key).
			With("field", field).
			Wrap(err)
	}

	c.l.Trace().
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
		return "", c.errBuilder.With("operation", "HGET").
			With("key", key).
			With("field", field).
			Wrap(err)
	}

	return val, nil
}

// SAdd adds one or more members to a Redis set.
func (c *Client) SAdd(ctx context.Context, key string, members ...any) error {
	err := c.Client.SAdd(ctx, key, members...).Err()
	if err != nil {
		return c.errBuilder.With("operation", "SADD").With("key", key).Wrap(err)
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
		return c.errBuilder.With("operation", "SREM").With("key", key).Wrap(err)
	}

	return nil
}

// SMembers retrieves all members of a Redis set.
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	members, err := c.Client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, c.errBuilder.With("operation", "SMEMBERS").With("key", key).Wrap(err)
	}

	return members, nil
}

// Expire sets an expiration time for a key in Redis.
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	ok, err := c.Client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return c.errBuilder.With("operation", "EXPIRE").With("key", key).Wrap(err)
	}
	if !ok {
		return c.errBuilder.With("operation", "EXPIRE").
			With("key", key).
			Wrap(eris.Errorf("key does not exist: %s", key))
	}

	return nil
}

// Del is a helper function to delete a key from redis
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if err := c.Client.Del(ctx, keys...).Err(); err != nil {
		return c.errBuilder.With("operation", "DEL").With("keys", keys).Wrap(err)
	}

	c.l.Debug().
		Str("keys", strings.Join(keys, ", ")).
		Msg("redis del success")

	return nil
}

// IncreaseWithExpiry increments a key's value and sets an expiration time atomically.
func (c *Client) IncreaseWithExpiry(
	ctx context.Context,
	key string,
	expiry time.Duration,
) (int64, error) {
	pipe := c.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiry)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, c.errBuilder.With("operation", "INCR_BY").With("key", key).Wrap(err)
	}

	return incr.Val(), nil
}

// Transaction executes a transactional operation on Redis.
func (c *Client) Transaction(ctx context.Context, fn func(tx *redis.Tx) error) error {
	return c.Watch(ctx, fn, "")
}

// GetInt retrieves an integer value for a key from Redis, returning a default value if the key does not exist.
func (c *Client) GetInt(ctx context.Context, key string, defaultVal int) (int, error) {
	val, err := c.Get(ctx, key)
	if err != nil {
		if eris.Is(err, redis.Nil) {
			return defaultVal, nil
		}
		return defaultVal, c.errBuilder.With("operation", "GET").With("key", key).Wrap(err)
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal, c.errBuilder.With("operation", "GET").With("key", key).Wrap(err)
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
		return 0, c.errBuilder.With("operation", "TTL").With("key", key).Wrap(err)
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
		return false, c.errBuilder.With("operation", "EXISTS").With("key", key).Wrap(err)
	}
	return n > 0, nil
}

// Keys returns all keys matching a pattern
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := c.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, c.errBuilder.With("operation", "KEYS").With("pattern", pattern).Wrap(err)
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
		return c.errBuilder.With("operation", "PING").Wrap(err)
	}
	c.l.Debug().Msg("redis ping successful")
	return nil
}

// HealthCheck performs a comprehensive health check of the Redis connection
func (c *Client) HealthCheck(ctx context.Context) error {
	c.l.Info().Msg("performing redis health check")

	// Check basic connectivity
	if err := c.Ping(ctx); err != nil {
		return c.errBuilder.With("operation", "PING").Wrap(err)
	}

	// Try a simple set/get operation
	testKey := "health_check_" + time.Now().Format("20060102150405")
	testValue := "ok"

	if err := c.Set(ctx, testKey, testValue, time.Second*10); err != nil {
		c.l.Error().Err(err).Msg("redis health check failed: set operation")
		return c.errBuilder.With("operation", "SET").Wrap(err)
	}

	val, err := c.Get(ctx, testKey)
	if err != nil {
		c.l.Error().Err(err).Msg("redis health check failed: get operation")
		return c.errBuilder.With("operation", "GET").Wrap(err)
	}

	if val != testValue {
		c.l.Error().
			Str("expected", testValue).
			Str("actual", val).
			Msg("redis health check failed: value mismatch")
		return c.errBuilder.With("operation", "GET").With("key", testKey).Wrap(err)
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
