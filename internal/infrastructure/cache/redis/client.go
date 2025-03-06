package redis

import (
	"context"
	"strconv"
	"strings"
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
)

type StringCmd = redis.StringCmd

var (
	// Nil reply returned by Redis when key does not exist.
	Nil = redis.Nil
	// Error returned when the redis config is nil
	ErrConfigNil = eris.New("redis config is nil")
)

// Client wraps the Redis client with additional functionality
type Client struct {
	*redis.Client
	l *zerolog.Logger
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
		return nil, ErrConfigNil
	}

	// Set the defaults
	setDefaults(cfg)

	opts := &redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		ConnMaxIdleTime: cfg.ConnTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			return cn.Ping(ctx).Err()
		},
	}

	redisClient := redis.NewClient(opts)

	// TODO(Wolfred): see if we need this because their is the OnConnect function
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnTimeout)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, eris.Wrap(err, "failed to connect to redis")
	}

	client := &Client{
		Client: redisClient,
		l:      &log,
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

// Get is a helper function to get a value from redis
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.Client.Get(ctx, key).Result()
	if err != nil {
		if eris.Is(err, redis.Nil) {
			c.l.Debug().
				Str("key", key).
				Msg("redis key not found")
			return "", redis.Nil
		}
		c.l.Error().
			Err(err).
			Str("key", key).
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
func (c *Client) HSet(ctx context.Context, key, field string, value interface{}) error {
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
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	err := c.Client.SAdd(ctx, key, members...).Err()
	if err != nil {
		return eris.Wrapf(err, "failed to add members to set: %s", key)
	}

	c.l.Debug().
		Str("key", key).
		Interface("members", members).
		Msg("members added to set successfully")

	return nil
}

// SRem removes one or more members from a Redis set.
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
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
	n, err := c.Client.Exists(ctx, key).Result()
	if err != nil {
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
