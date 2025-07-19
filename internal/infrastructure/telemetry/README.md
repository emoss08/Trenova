# Telemetry Package

This package provides comprehensive observability for the Trenova application using OpenTelemetry standards.

## Overview

The telemetry package implements:

- **Distributed Tracing**: Track requests across services
- **Metrics Collection**: Monitor performance and business metrics
- **Structured Logging**: Correlate logs with traces
- **Instrumentation**: Automatic instrumentation for HTTP, database, and cache operations

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Application   │────▶│   Telemetry     │────▶│ OTLP Collector  │
│                 │     │   Provider      │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                          │
                              ┌───────────────────────────┴─────────────┐
                              │                                         │
                        ┌─────▼─────┐  ┌─────────┐  ┌────────┐  ┌─────▼────┐
                        │   Tempo   │  │  Loki   │  │ Mimir  │  │ Grafana  │
                        │ (Traces)  │  │ (Logs)  │  │(Metrics)│  │  (UI)    │
                        └───────────┘  └─────────┘  └────────┘  └──────────┘
```

## Components

### 1. Telemetry Provider (`telemetry.go`)

- Initializes OpenTelemetry providers
- Manages resource attributes
- Handles graceful shutdown
- Configures sampling strategies

### 2. Metrics (`metrics.go`)

- Defines all application metrics
- Provides recording methods
- Follows Prometheus naming conventions
- Includes runtime metrics

### 3. Middleware (`middleware.go`)

- **TracingMiddleware**: Creates spans for HTTP requests
- **MetricsMiddleware**: Records HTTP metrics
- **LoggingMiddleware**: Structured request logging
- **RecoveryMiddleware**: Panic recovery with tracing

### 4. Database Instrumentation (`database.go`)

- Traces all database queries
- Records query duration and errors
- Monitors connection pool metrics
- Supports transaction tracing

### 5. Cache Instrumentation (`cache.go`)

- Traces Redis operations
- Tracks cache hit/miss rates
- Monitors operation latency
- Supports pipeline operations

## Usage

### Basic Setup

```go
// In your fx module
fx.Provide(
    telemetry.NewTelemetry,
    telemetry.NewMetrics,
),

// In your server setup
middleware := telemetry.NewCombinedMiddleware(
    config.ServiceName,
    metrics,
    logger,
)
middleware.Apply(app)
```

### Database Instrumentation

```go
// Instrument database
instrumentation, err := telemetry.InstrumentDatabase(
    db,
    config.ServiceName,
    metrics,
)

// Use transaction helper
err := telemetry.RunInTransaction(ctx, db, &telemetry.TransactionOptions{
    ReadOnly: false,
    Timeout:  30 * time.Second,
}, func(tx *bun.Tx) error {
    // Your transaction logic
    return nil
})
```

### Cache Instrumentation

```go
// Instrument Redis
cache := telemetry.NewCacheInstrumentation(config.ServiceName, metrics)
cache.InstrumentRedis(redisClient)

// Manual cache operation tracking
op := telemetry.StartCacheOperation(ctx, "get", metrics, &telemetry.CacheOperationOptions{
    CacheType: "redis",
    Key:       "user:123",
})
defer op.End(hit, err)
```

## Configuration

### Environment Variables

```yaml
telemetry:
  enabled: true
  metricsEnabled: true
  tracingEnabled: true
  serviceName: "trenova"
  serviceVersion: "1.0.0"
  environment: "production"

  otlp:
    endpoint: "otel-collector:4317"
    insecure: true
    headers: {}

  sampling:
    probability: 1.0 # Sample 100% in dev, reduce in production
    parentBased: true
```

### Middleware Configuration

```go
// Customize tracing
tracingConfig := telemetry.TracingConfig{
    ServiceName:    "my-service",
    SkipPaths:     []string{"/health", "/metrics"},
    SensitivePaths: []string{"/api/auth"},
    RecordBody:     false,
    MaxBodySize:    65536,
}

// Customize metrics
metricsConfig := telemetry.MetricsConfig{
    Metrics:      metrics,
    SkipPaths:    []string{"/health"},
    RecordSize:   true,
    GroupedPaths: map[string]string{
        "/api/v1/users/123": "/api/v1/users/:id",
    },
}
```

## Metrics Reference

### HTTP Metrics

- `trenova_http_requests_total`: Total HTTP requests
- `trenova_http_request_duration_seconds`: Request latency histogram
- `trenova_http_requests_active`: Currently active requests
- `trenova_http_request_size_bytes`: Request body size
- `trenova_http_response_size_bytes`: Response body size
- `trenova_http_errors_total`: Total HTTP errors

### Database Metrics

- `trenova_database_operations_total`: Total database operations
- `trenova_database_operation_duration_seconds`: Query latency
- `trenova_database_errors_total`: Database errors
- `trenova_database_connections_open`: Open connections
- `trenova_database_connections_in_use`: Active connections
- `trenova_database_connections_idle`: Idle connections

### Cache Metrics

- `trenova_cache_operations_total`: Total cache operations
- `trenova_cache_operation_duration_seconds`: Operation latency
- `trenova_cache_hits_total`: Cache hits
- `trenova_cache_misses_total`: Cache misses
- `trenova_cache_evictions_total`: Cache evictions

### Runtime Metrics

- `trenova_runtime_memory_alloc_bytes`: Memory allocation
- `trenova_runtime_goroutines`: Active goroutines
- `trenova_runtime_gc_pause_seconds_total`: GC pause time

## Best Practices

1. **Sampling**: Use appropriate sampling rates in production
2. **Cardinality**: Avoid high-cardinality labels (use grouped paths)
3. **Security**: Never log sensitive data (passwords, tokens)
4. **Performance**: Skip instrumentation for health checks
5. **Context**: Always propagate context for trace correlation

## Troubleshooting

### No Metrics Appearing

- Check if telemetry is enabled in configuration
- Verify OTLP collector is reachable
- Check Prometheus scrape configuration

### Missing Traces

- Ensure trace context is propagated
- Verify sampling configuration
- Check span recording status

### High Memory Usage

- Review sampling rate
- Check for metric cardinality issues
- Monitor batch sizes

## Development

### Running Locally

```bash
# Start LGTM stack
docker-compose -f docker-compose-local.yml up -d

# Access services
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090
- Tempo: http://localhost:3200
- Loki: http://localhost:3100
```

### Adding New Metrics

1. Define metric in `metrics.go`
2. Add recording method
3. Call from appropriate instrumentation point
4. Update Grafana dashboards

### Testing

```go
// Test with no-op providers
metrics, _ := telemetry.NewMetrics(nil)

// Test with mock metrics
mockMetrics := &MockMetrics{}
hook := telemetry.NewTracingHook(telemetry.DatabaseHookConfig{
    ServiceName: "test",
    Metrics:     mockMetrics,
})
```
