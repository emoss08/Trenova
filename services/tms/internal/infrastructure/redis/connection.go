package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/sourcegraph/conc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ ports.CacheConnection = (*Connection)(nil)

type ConnectionParams struct {
	fx.In

	Config  *config.Config
	Logger  *zap.Logger
	Tracer  *observability.TracerProvider  `optional:"true"`
	Metrics *observability.MetricsRegistry `optional:"true"`
	LC      fx.Lifecycle
}

type Connection struct {
	client           redis.UniversalClient
	cfg              *config.CacheConfig
	logger           *zap.Logger
	scriptLoader     *ScriptLoader
	tracer           *observability.TracerProvider
	metrics          *observability.MetricsRegistry
	slowLogThreshold time.Duration
}

const (
	defaultSlowLogThreshold = 50 * time.Millisecond
)

func NewConnection(p ConnectionParams) (*Connection, error) {
	if p.Config.Cache == nil || p.Config.Cache.Provider != "redis" {
		p.Logger.Info("Redis cache is not configured or disabled")
		return nil, ErrConnectionNotInitialized
	}

	ctx := context.Background()
	logger := p.Logger.With(zap.String("component", "redis"))

	cfg := p.Config.Cache
	conn := &Connection{
		cfg:              cfg,
		logger:           logger,
		tracer:           p.Tracer,
		metrics:          p.Metrics,
		slowLogThreshold: cfg.SlowLogThreshold,
	}

	if conn.slowLogThreshold == 0 {
		conn.slowLogThreshold = defaultSlowLogThreshold
	}

	client := conn.createClient(cfg)

	conn.client = client

	if p.Tracer != nil && p.Tracer.IsEnabled() {
		if err := redisotel.InstrumentTracing(client); err != nil {
			logger.Warn("Failed to instrument Redis tracing", zap.Error(err))
		}
		if err := redisotel.InstrumentMetrics(client); err != nil {
			logger.Warn("Failed to instrument Redis metrics", zap.Error(err))
		}
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger.Info("Redis connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.Int("db", cfg.DB),
		zap.Int("poolSize", cfg.PoolSize),
		zap.Bool("clusterMode", cfg.ClusterMode),
		zap.Bool("sentinelMode", cfg.SentinelMode),
	)

	conn.warmupConnectionPool(ctx)

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go conn.monitorConnection(ctx)
			go conn.monitorSlowCommands(ctx)
			return nil
		},
		OnStop: func(context.Context) error {
			logger.Info("Closing Redis connection")
			if err := conn.Close(); err != nil {
				logger.Error("Failed to close Redis connection", zap.Error(err))
				return err
			}
			logger.Info("Redis connection closed successfully")
			return nil
		},
	})

	return conn, nil
}

func (c *Connection) createClient(cfg *config.CacheConfig) redis.UniversalClient {
	if cfg.ClusterMode {
		return c.createClusterClient(cfg)
	}

	if cfg.SentinelMode {
		return c.createFailoverClient(cfg)
	}

	return c.createStandaloneClient(cfg)
}

func (c *Connection) createStandaloneClient(cfg *config.CacheConfig) *redis.Client {
	opts := &redis.Options{
		Addr:            cfg.GetRedisAddr(),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		PoolTimeout:     cfg.PoolTimeout,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	}
	if opts.PoolSize == 0 {
		opts.PoolSize = 25
	}
	if opts.MinIdleConns == 0 {
		opts.MinIdleConns = 10
	}
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 10 * time.Second
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 5 * time.Second
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 5 * time.Second
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}

	return redis.NewClient(opts)
}

func (c *Connection) createClusterClient(cfg *config.CacheConfig) *redis.ClusterClient {
	opts := &redis.ClusterOptions{
		Addrs:           cfg.ClusterNodes,
		Password:        cfg.Password,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		PoolTimeout:     cfg.PoolTimeout,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	}

	return redis.NewClusterClient(opts)
}

func (c *Connection) createFailoverClient(cfg *config.CacheConfig) *redis.Client {
	opts := &redis.FailoverOptions{
		MasterName:       cfg.MasterName,
		SentinelAddrs:    cfg.SentinelAddrs,
		Password:         cfg.Password,
		DB:               cfg.DB,
		SentinelPassword: cfg.SentinelPassword,
		PoolSize:         cfg.PoolSize,
		MinIdleConns:     cfg.MinIdleConns,
		ConnMaxIdleTime:  cfg.ConnMaxIdleTime,
		ConnMaxLifetime:  cfg.ConnMaxLifetime,
		PoolTimeout:      cfg.PoolTimeout,
		DialTimeout:      cfg.DialTimeout,
		ReadTimeout:      cfg.ReadTimeout,
		WriteTimeout:     cfg.WriteTimeout,
		MaxRetries:       cfg.MaxRetries,
		MinRetryBackoff:  cfg.MinRetryBackoff,
		MaxRetryBackoff:  cfg.MaxRetryBackoff,
	}

	return redis.NewFailoverClient(opts)
}

func (c *Connection) Client() *redis.Client {
	if client, ok := c.client.(*redis.Client); ok {
		return client
	}
	return nil
}

func (c *Connection) SetScriptLoader(sl *ScriptLoader) {
	c.scriptLoader = sl
}

func (c *Connection) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	val, err := c.client.Get(ctx, key).Result()
	c.recordOperation(ctx, "GET", time.Since(start), err)

	if errors.Is(err, redis.Nil) {
		if c.metrics != nil {
			c.metrics.RecordCacheMiss("redis")
		}
		return "", nil
	}
	if err != nil {
		if c.metrics != nil {
			c.metrics.RecordError("redis_get", "cache")
		}
		return "", err
	}

	if c.metrics != nil {
		c.metrics.RecordCacheHit("redis")
	}
	return val, nil
}

func (c *Connection) GetJSON(ctx context.Context, key string, v any) error {
	start := time.Now()
	val, err := c.client.JSONGet(ctx, key, ".").Result()
	c.recordOperation(ctx, "GET", time.Since(start), err)

	if errors.Is(err, redis.Nil) {
		if c.metrics != nil {
			c.metrics.RecordCacheMiss("redis")
		}
		return nil
	}
	if err != nil {
		if c.metrics != nil {
			c.metrics.RecordError("redis_get_json", "cache")
		}
		return err
	}

	if c.metrics != nil {
		c.metrics.RecordCacheHit("redis")
	}

	return sonic.Unmarshal([]byte(val), v)
}

func (c *Connection) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	start := time.Now()
	err := c.client.Set(ctx, key, value, ttl).Err()
	c.recordOperation(ctx, "SET", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_set", "cache")
	}
	return err
}

func (c *Connection) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := sonic.Marshal(value)
	if err != nil {
		if c.metrics != nil {
			c.metrics.RecordError("redis_set_json_marshal", "cache")
		}
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	start := time.Now()
	err = c.client.JSONSet(ctx, key, ".", string(data)).Err()
	c.client.Expire(ctx, key, ttl)
	c.recordOperation(ctx, "SET", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_set_json", "cache")
	}
	return err
}

func (c *Connection) Delete(ctx context.Context, keys ...string) error {
	start := time.Now()
	err := c.client.Del(ctx, keys...).Err()
	c.recordOperation(ctx, "DEL", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_delete", "cache")
	}
	return err
}

func (c *Connection) Exists(ctx context.Context, keys ...string) (int64, error) {
	start := time.Now()
	val, err := c.client.Exists(ctx, keys...).Result()
	c.recordOperation(ctx, "EXISTS", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_exists", "cache")
	}
	return val, err
}

func (c *Connection) Expire(ctx context.Context, key string, ttl time.Duration) error {
	start := time.Now()
	err := c.client.Expire(ctx, key, ttl).Err()
	c.recordOperation(ctx, "EXPIRE", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_expire", "cache")
	}
	return err
}

func (c *Connection) TTL(ctx context.Context, key string) (time.Duration, error) {
	start := time.Now()
	val, err := c.client.TTL(ctx, key).Result()
	c.recordOperation(ctx, "TTL", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_ttl", "cache")
	}
	return val, err
}

func (c *Connection) HGet(ctx context.Context, key, field string) (string, error) {
	start := time.Now()
	val, err := c.client.HGet(ctx, key, field).Result()
	c.recordOperation(ctx, "HGET", time.Since(start), err)

	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_hget", "cache")
	}

	return val, err
}

func (c *Connection) HSet(ctx context.Context, key string, values ...any) error {
	start := time.Now()
	err := c.client.HSet(ctx, key, values...).Err()
	c.recordOperation(ctx, "HSET", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_hset", "cache")
	}

	return err
}

func (c *Connection) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	start := time.Now()
	val, err := c.client.HGetAll(ctx, key).Result()
	c.recordOperation(ctx, "HGETALL", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_hgetall", "cache")
	}

	return val, err
}

func (c *Connection) SAdd(ctx context.Context, key string, members ...any) error {
	start := time.Now()
	err := c.client.SAdd(ctx, key, members...).Err()
	c.recordOperation(ctx, "SADD", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_sadd", "cache")
	}

	return err
}

func (c *Connection) SRem(ctx context.Context, key string, members ...any) error {
	start := time.Now()
	err := c.client.SRem(ctx, key, members...).Err()
	c.recordOperation(ctx, "SREM", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_srem", "cache")
	}

	return err
}

func (c *Connection) SMembers(ctx context.Context, key string) ([]string, error) {
	start := time.Now()
	val, err := c.client.SMembers(ctx, key).Result()
	c.recordOperation(ctx, "SMEMBERS", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_smembers", "cache")
	}

	return val, err
}

func (c *Connection) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	start := time.Now()
	err := c.client.ZAdd(ctx, key, members...).Err()
	c.recordOperation(ctx, "ZADD", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_zadd", "cache")
	}

	return err
}

func (c *Connection) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	startTime := time.Now()
	val, err := c.client.ZRange(ctx, key, start, stop).Result()
	c.recordOperation(ctx, "ZRANGE", time.Since(startTime), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_zrange", "cache")
	}

	return val, err
}

func (c *Connection) Ping(ctx context.Context) error {
	start := time.Now()
	err := c.client.Ping(ctx).Err()
	c.recordOperation(ctx, "PING", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_ping", "cache")
	}
	return err
}

func (c *Connection) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Connection) Incr(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	val, err := c.client.Incr(ctx, key).Result()
	c.recordOperation(ctx, "INCR", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_incr", "cache")
	}
	return val, err
}

func (c *Connection) SetNX(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) (bool, error) {
	start := time.Now()
	ok, err := c.client.SetNX(ctx, key, value, ttl).Result()
	c.recordOperation(ctx, "SETNX", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_setnx", "cache")
	}
	return ok, err
}

func (c *Connection) Publish(ctx context.Context, channel string, message any) error {
	start := time.Now()
	err := c.client.Publish(ctx, channel, message).Err()
	c.recordOperation(ctx, "PUBLISH", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_publish", "cache")
	}
	return err
}

func (c *Connection) PSubscribe(ctx context.Context, channels ...string) *redis.PubSub {
	start := time.Now()
	pubsub := c.client.PSubscribe(ctx, channels...)
	duration := time.Since(start)

	if duration > c.slowLogThreshold {
		c.recordOperation(ctx, "PSUBSCRIBE", duration, nil)
	}

	return pubsub
}

func (c *Connection) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

func (c *Connection) ScriptLoad(ctx context.Context, script string) (string, error) {
	start := time.Now()
	sha, err := c.client.ScriptLoad(ctx, script).Result()
	c.recordOperation(ctx, "SCRIPT_LOAD", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_script_load", "cache")
	}
	return sha, err
}

func (c *Connection) ScriptExists(ctx context.Context, hashes ...string) ([]bool, error) {
	start := time.Now()
	result, err := c.client.ScriptExists(ctx, hashes...).Result()
	c.recordOperation(ctx, "SCRIPT_EXISTS", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_script_exists", "cache")
	}
	return result, err
}

func (c *Connection) EvalSha(
	ctx context.Context,
	sha string,
	keys []string,
	args ...any,
) (any, error) {
	start := time.Now()
	result, err := c.client.EvalSha(ctx, sha, keys, args...).Result()
	c.recordOperation(ctx, "EVALSHA", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_evalsha", "cache")
	}
	return result, err
}

func (c *Connection) Eval(
	ctx context.Context,
	script string,
	keys []string,
	args ...any,
) (any, error) {
	start := time.Now()
	result, err := c.client.Eval(ctx, script, keys, args...).Result()
	c.recordOperation(ctx, "EVAL", time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError("redis_eval", "cache")
	}
	return result, err
}

func (c *Connection) RunScript(
	ctx context.Context,
	scriptName string,
	keys []string,
	args ...any,
) (any, error) {
	if c.scriptLoader == nil {
		return nil, ErrScriptLoaderNotInitialized
	}

	start := time.Now()
	result, err := c.scriptLoader.EvalSHA(ctx, scriptName, keys, args...)
	c.recordOperation(ctx, fmt.Sprintf("SCRIPT:%s", scriptName), time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError(fmt.Sprintf("redis_script_%s", scriptName), "cache")
	}
	return result, err
}

func (c *Connection) RunScriptPipelined(
	ctx context.Context,
	scriptName string,
	keys []string,
	args ...any,
) (any, error) {
	if c.scriptLoader == nil {
		return nil, ErrScriptLoaderNotInitialized
	}

	if !c.cfg.EnablePipelining {
		return c.RunScript(ctx, scriptName, keys, args...)
	}

	start := time.Now()

	sha, exists := c.scriptLoader.GetScriptSHA(scriptName)
	if !exists {
		return c.RunScript(ctx, scriptName, keys, args...)
	}

	pipe := c.client.Pipeline()
	cmd := pipe.EvalSha(ctx, sha, keys, args...)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return c.RunScript(ctx, scriptName, keys, args...)
	}

	result, err := cmd.Result()
	c.recordOperation(ctx, fmt.Sprintf("SCRIPT:%s:PIPELINED", scriptName), time.Since(start), err)

	if err != nil && c.metrics != nil {
		c.metrics.RecordError(fmt.Sprintf("redis_script_%s_pipelined", scriptName), "cache")
	}
	return result, err
}

func (c *Connection) BatchSet(ctx context.Context, items map[string]any, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	start := time.Now()

	if c.cfg.EnablePipelining {
		pipe := c.client.Pipeline()

		for key, value := range items {
			data, err := sonic.Marshal(value)
			if err != nil {
				c.recordOperation(ctx, "BATCH_SET", time.Since(start), err)
				return err
			}
			pipe.SetEx(ctx, key, data, ttl)
		}

		_, err := pipe.Exec(ctx)
		c.recordOperation(ctx, "BATCH_SET:PIPELINED", time.Since(start), err)
		return err
	}

	for key, value := range items {
		if err := c.Set(ctx, key, value, ttl); err != nil {
			c.recordOperation(ctx, "BATCH_SET:SEQUENTIAL", time.Since(start), err)
			return err
		}
	}

	c.recordOperation(ctx, "BATCH_SET:SEQUENTIAL", time.Since(start), nil)
	return nil
}

func (c *Connection) BatchGet(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return nil, errors.New("no keys provided")
	}

	start := time.Now()
	results := make(map[string]string)

	if c.cfg.EnablePipelining {
		pipe := c.client.Pipeline()
		cmds := make([]*redis.StringCmd, len(keys))

		for i, key := range keys {
			cmds[i] = pipe.Get(ctx, key)
		}

		_, err := pipe.Exec(ctx)
		if err != nil && errors.Is(err, redis.Nil) {
			c.recordOperation(ctx, "BATCH_GET:PIPELINED", time.Since(start), err)
			return nil, err
		}

		for i, cmd := range cmds {
			val, valErr := cmd.Result()
			if valErr == nil {
				results[keys[i]] = val
			}
		}

		c.recordOperation(ctx, "BATCH_GET:PIPELINED", time.Since(start), nil)
		return results, nil
	}

	for _, key := range keys {
		val, err := c.client.Get(ctx, key).Result()
		if err == nil {
			results[key] = val
		}
	}

	c.recordOperation(ctx, "BATCH_GET:SEQUENTIAL", time.Since(start), nil)
	return results, nil
}

func (c *Connection) warmupConnectionPool(ctx context.Context) {
	if c.cfg.PoolSize <= 0 {
		return
	}

	c.logger.Debug("Warming up Redis connection pool", zap.Int("pool_size", c.cfg.PoolSize))

	warmupCount := c.cfg.MinIdleConns
	if warmupCount == 0 {
		warmupCount = c.cfg.PoolSize / 2
	}

	var wg conc.WaitGroup
	for i := 0; i < warmupCount; i++ {
		wg.Go(func() {
			_ = c.client.Ping(ctx).Err()
		})
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Debug("Connection pool warmup completed", zap.Int("connections", warmupCount))
	case <-timeoutCtx.Done():
		c.logger.Warn("Connection pool warmup timed out")
	}
}

func (c *Connection) recordOperation(
	ctx context.Context,
	operation string,
	duration time.Duration,
	err error,
) {
	if c.metrics != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		c.metrics.RecordDBQuery(operation, "redis", status, duration.Seconds())
	}

	if duration >= c.slowLogThreshold { //nolint:nestif // This is a slow operation warning
		c.logger.Warn("Slow Redis operation detected",
			zap.String("operation", operation),
			zap.Duration("duration", duration),
			zap.Error(err),
		)

		if c.tracer != nil && c.tracer.IsEnabled() {
			span := trace.SpanFromContext(ctx)
			if span.IsRecording() {
				span.SetAttributes(
					attribute.Bool("redis.slow_operation", true),
					attribute.String("redis.operation", operation),
					attribute.Float64("redis.duration_ms", float64(duration.Milliseconds())),
				)

				if duration >= c.slowLogThreshold*2 {
					span.AddEvent("Very slow Redis operation",
						trace.WithAttributes(
							attribute.String("operation", operation),
							attribute.Float64("duration_ms", float64(duration.Milliseconds())),
						),
					)
				}
			}
		}
	}
}

func (c *Connection) monitorConnection(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Ping(ctx); err != nil {
				c.logger.Error("Redis health check failed", zap.Error(err))
				if c.metrics != nil {
					c.metrics.RecordError("redis_health_check", "cache")
				}
			}

			if client, ok := c.client.(*redis.Client); ok {
				stats := client.PoolStats()
				c.logger.Debug("Redis connection pool stats",
					zap.Uint32("hits", stats.Hits),
					zap.Uint32("misses", stats.Misses),
					zap.Uint32("timeouts", stats.Timeouts),
					zap.Uint32("totalConns", stats.TotalConns),
					zap.Uint32("idleConns", stats.IdleConns),
					zap.Uint32("staleConns", stats.StaleConns),
				)
			}
		}
	}
}

func (c *Connection) monitorSlowCommands(ctx context.Context) {
	if c.slowLogThreshold == 0 {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkSlowLog(ctx)
		}
	}
}

func (c *Connection) checkSlowLog(ctx context.Context) {
	slowLogCmd := c.client.SlowLogGet(ctx, 10)
	entries, err := slowLogCmd.Result()
	if err != nil {
		c.logger.Error("Failed to get Redis slow log", zap.Error(err))
		if c.metrics != nil {
			c.metrics.RecordError("redis_slowlog_get", "cache")
		}

		return
	}

	if len(entries) == 0 {
		return
	}

	for _, entry := range entries {
		duration := time.Duration( //nolint:unconvert,durationcheck // This is a valid duration
			entry.Duration,
		) * time.Microsecond
		if duration >= c.slowLogThreshold {
			c.logger.Warn("Slow Redis command detected in slow log",
				zap.Int64("id", entry.ID),
				zap.Duration("duration", duration),
				zap.Strings("args", entry.Args),
				zap.String("clientAddr", entry.ClientAddr),
				zap.String("clientName", entry.ClientName),
			)

			if c.metrics != nil {
				c.metrics.RecordDBQuery("SLOWLOG", "redis", "slow", duration.Seconds())
			}

			if c.tracer != nil && c.tracer.IsEnabled() {
				span := trace.SpanFromContext(ctx)
				if span.IsRecording() {
					span.AddEvent("Slow Redis command in log",
						trace.WithAttributes(
							attribute.Int64("slowlog.id", entry.ID),
							attribute.Float64(
								"slowlog.duration_ms",
								float64(duration.Milliseconds()),
							),
							attribute.StringSlice("slowlog.args", entry.Args),
						),
					)
				}
			}
		}
	}

	// Clear slow log after processing by getting the current length and reading all entries
	// Note: Redis doesn't have a direct SLOWLOG RESET via go-redis, but we can achieve
	// similar behavior by reading all entries which effectively clears them from memory
	c.client.Do(ctx, "SLOWLOG", "RESET")
}
