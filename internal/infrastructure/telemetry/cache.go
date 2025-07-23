// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package telemetry

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rotisserie/eris"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// CacheHookConfig provides configuration for cache hooks
type CacheHookConfig struct {
	ServiceName     string
	Metrics         *Metrics
	RecordStatement bool
	MaxStatementLen int
	SkipOperations  []string
}

// DefaultCacheHookConfig returns default configuration
func DefaultCacheHookConfig(serviceName string, metrics *Metrics) CacheHookConfig {
	return CacheHookConfig{
		ServiceName:     serviceName,
		Metrics:         metrics,
		RecordStatement: false,
		MaxStatementLen: 1000,
		SkipOperations:  []string{"ping", "echo"},
	}
}

// RedisHook implements Redis hook for comprehensive instrumentation
type RedisHook struct {
	tracer  trace.Tracer
	metrics *Metrics
	config  CacheHookConfig
	skipMap map[string]bool
}

// NewRedisHook creates a new Redis hook with configuration
func NewRedisHook(config CacheHookConfig) *RedisHook {
	skipMap := make(map[string]bool, len(config.SkipOperations))
	for _, op := range config.SkipOperations {
		skipMap[strings.ToLower(op)] = true
	}

	return &RedisHook{
		tracer:  otel.Tracer(fmt.Sprintf("%s.cache", config.ServiceName)),
		metrics: config.Metrics,
		config:  config,
		skipMap: skipMap,
	}
}

// DialHook instruments Redis connection establishment
func (h *RedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port := extractHostPort(addr)

		ctx, span := h.tracer.Start(ctx, "cache.connect",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.DBSystemRedis,
				semconv.NetworkTransportKey.String(network),
				semconv.ServerAddress(host),
				semconv.ServerPort(port),
			),
		)
		defer span.End()

		start := time.Now()
		conn, err := next(ctx, network, addr)
		duration := time.Since(start)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Connection failed")

			if h.metrics != nil {
				h.metrics.RecordDatabaseConnection(ctx, "redis", false)
			}
		} else {
			span.SetStatus(codes.Ok, "Connected successfully")
			span.SetAttributes(attribute.Float64("connection.duration_ms", float64(duration.Milliseconds())))

			if h.metrics != nil {
				h.metrics.RecordDatabaseConnection(ctx, "redis", true)
				h.metrics.RecordActiveDatabaseConnections(ctx, 1)
			}
		}

		return &instrumentedConn{Conn: conn, metrics: h.metrics}, err
	}
}

// ProcessHook instruments individual Redis commands
func (h *RedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		operation := strings.ToLower(cmd.Name())

		if h.skipMap[operation] {
			return next(ctx, cmd)
		}

		start := time.Now()

		attrs := []attribute.KeyValue{
			semconv.DBSystemRedis,
			attribute.String("db.operation", operation),
		}

		if h.config.RecordStatement {
			stmt := h.sanitizeCommand(cmd)
			if len(stmt) > h.config.MaxStatementLen {
				stmt = stmt[:h.config.MaxStatementLen] + "..."
			}
			attrs = append(attrs, attribute.String("db.statement", stmt))
		}

		if key := extractKey(cmd); key != "" {
			attrs = append(attrs, attribute.String("cache.key", key))
		}

		ctx, span := h.tracer.Start(ctx, fmt.Sprintf("cache.%s", operation),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attrs...),
		)
		defer span.End()

		err := next(ctx, cmd)
		duration := time.Since(start)

		hit := h.determineCacheHit(operation, err, cmd.Err())

		span.SetAttributes(
			attribute.Bool("cache.hit", hit),
			attribute.Float64("cache.duration_ms", float64(duration.Milliseconds())),
		)

		switch {
		case err != nil:
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		case eris.Is(err, redis.Nil):
			span.SetStatus(codes.Ok, "Key not found")
		default:
			span.SetStatus(codes.Ok, "Command executed successfully")
		}

		if h.metrics != nil {
			h.metrics.RecordCacheOperation(ctx, operation, hit, duration)
		}

		return err
	}
}

// ProcessPipelineHook instruments Redis pipeline operations
func (h *RedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()

		opCounts := h.countOperations(cmds)

		attrs := []attribute.KeyValue{
			semconv.DBSystemRedis,
			attribute.String("db.operation", "pipeline"),
			attribute.Int("cache.pipeline.size", len(cmds)),
		}

		for op, count := range opCounts {
			attrs = append(attrs, attribute.Int(fmt.Sprintf("cache.pipeline.%s_count", op), count))
		}

		ctx, span := h.tracer.Start(ctx, "cache.pipeline",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attrs...),
		)
		defer span.End()

		err := next(ctx, cmds)
		duration := time.Since(start)

		successCount := 0
		for _, cmd := range cmds {
			if cmd.Err() == nil {
				successCount++
			}
		}

		span.SetAttributes(
			attribute.Int("cache.pipeline.success_count", successCount),
			attribute.Float64("cache.duration_ms", float64(duration.Milliseconds())),
		)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "Pipeline executed successfully")
		}

		if h.metrics != nil {
			h.metrics.RecordCacheOperation(ctx, "pipeline", err == nil, duration)
		}

		return err
	}
}

// extractHostPort extracts host and port from address string
func extractHostPort(addr string) (host string, port int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0
	}

	port = 0
	if portStr != "" {
		_, _ = fmt.Sscanf(portStr, "%d", &port)
	}

	return host, port
}

// extractKey extracts the key from certain Redis commands
func extractKey(cmd redis.Cmder) string {
	args := cmd.Args()
	if len(args) < 2 {
		return ""
	}

	// For most commands, the key is the second argument
	if key, ok := args[1].(string); ok {
		return key
	}

	return ""
}

// sanitizeCommand removes sensitive data from commands
func (h *RedisHook) sanitizeCommand(cmd redis.Cmder) string {
	// ! TODO(wolfred): we need to write this
	return cmd.Name()
}

// determineCacheHit determines if a cache operation was a hit
func (h *RedisHook) determineCacheHit(operation string, err, cmdErr error) bool {
	if err == nil && cmdErr == nil {
		return true
	}

	if eris.Is(err, redis.Nil) || eris.Is(cmdErr, redis.Nil) {
		switch operation {
		case "get", "mget", "hget", "hmget", "exists":
			return false
		}
	}

	return false
}

// countOperations counts operations by type in a pipeline
func (h *RedisHook) countOperations(cmds []redis.Cmder) map[string]int {
	counts := make(map[string]int)
	for _, cmd := range cmds {
		op := strings.ToLower(cmd.Name())
		counts[op]++
	}
	return counts
}

// instrumentedConn wraps net.Conn to track connection lifecycle
type instrumentedConn struct {
	net.Conn
	metrics *Metrics
	closed  bool
}

func (c *instrumentedConn) Close() error {
	err := c.Conn.Close()
	if !c.closed && c.metrics != nil {
		c.metrics.RecordActiveDatabaseConnections(context.Background(), -1)
		c.closed = true
	}
	return err
}

// CacheInstrumentation provides methods to instrument Redis clients
type CacheInstrumentation struct {
	config CacheHookConfig
}

// NewCacheInstrumentation creates a new cache instrumentation instance
func NewCacheInstrumentation(serviceName string, metrics *Metrics) *CacheInstrumentation {
	return &CacheInstrumentation{
		config: DefaultCacheHookConfig(serviceName, metrics),
	}
}

// InstrumentRedis adds instrumentation to a Redis client
func (i *CacheInstrumentation) InstrumentRedis(client *redis.Client) {
	hook := NewRedisHook(i.config)
	client.AddHook(hook)
}

// InstrumentRedisCluster adds instrumentation to a Redis cluster client
func (i *CacheInstrumentation) InstrumentRedisCluster(client *redis.ClusterClient) {
	hook := NewRedisHook(i.config)
	client.AddHook(hook)
}

// CacheOperation provides a helper for manual cache operation tracing
type CacheOperation struct {
	ctx       context.Context
	span      trace.Span
	start     time.Time
	metrics   *Metrics
	operation string
	cacheType string
}

// CacheOperationOptions provides options for cache operations
type CacheOperationOptions struct {
	CacheType string
	Key       string
	Size      int64
	TTL       time.Duration
	Attrs     []attribute.KeyValue
}

// StartCacheOperation starts a new cache operation span
func StartCacheOperation(
	ctx context.Context,
	operation string,
	metrics *Metrics,
	opts *CacheOperationOptions,
) *CacheOperation {
	tracer := otel.Tracer("cache")

	attrs := []attribute.KeyValue{
		attribute.String("db.operation", operation),
	}

	if opts != nil {
		if opts.CacheType != "" {
			attrs = append(attrs, attribute.String("cache.type", opts.CacheType))
		}
		if opts.Key != "" {
			attrs = append(attrs, attribute.String("cache.key", opts.Key))
		}
		if opts.Size > 0 {
			attrs = append(attrs, attribute.Int64("cache.item_size", opts.Size))
		}
		if opts.TTL > 0 {
			attrs = append(attrs, attribute.Float64("cache.ttl_seconds", opts.TTL.Seconds()))
		}
		attrs = append(attrs, opts.Attrs...)
	}

	cacheType := "generic"
	if opts != nil && opts.CacheType != "" {
		cacheType = opts.CacheType
	}

	ctx, span := tracer.Start(ctx, fmt.Sprintf("cache.%s.%s", cacheType, operation),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	defer span.End()

	return &CacheOperation{
		ctx:       ctx,
		span:      span,
		start:     time.Now(),
		metrics:   metrics,
		operation: operation,
		cacheType: cacheType,
	}
}

// RecordHit records a cache hit
func (co *CacheOperation) RecordHit() {
	co.span.SetAttributes(attribute.Bool("cache.hit", true))
}

// RecordMiss records a cache miss
func (co *CacheOperation) RecordMiss() {
	co.span.SetAttributes(attribute.Bool("cache.hit", false))
}

// RecordEviction records a cache eviction
func (co *CacheOperation) RecordEviction(reason string) {
	co.span.AddEvent("cache.eviction", trace.WithAttributes(
		attribute.String("eviction.reason", reason),
	))

	if co.metrics != nil {
		co.metrics.RecordCacheEviction(co.ctx, reason)
	}
}

// End completes the cache operation
func (co *CacheOperation) End(hit bool, err error) {
	duration := time.Since(co.start)

	co.span.SetAttributes(
		attribute.Bool("cache.hit", hit),
		attribute.Float64("cache.duration_ms", float64(duration.Milliseconds())),
	)

	if err != nil {
		co.span.RecordError(err)
		co.span.SetStatus(codes.Error, err.Error())
	} else {
		co.span.SetStatus(codes.Ok, "Cache operation completed")
	}

	co.span.End()

	if co.metrics != nil {
		co.metrics.RecordCacheOperation(co.ctx, co.operation, hit, duration)
	}
}

// Context returns the operation's context
func (co *CacheOperation) Context() context.Context {
	return co.ctx
}
