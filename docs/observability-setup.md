# Observability Setup Guide

This guide explains how to integrate the LGTM (Loki, Grafana, Tempo, Mimir) observability stack into your Trenova application.

## Configuration

Add these environment variables to your `.env` file:

```env
# Telemetry Configuration
TELEMETRY_ENABLED=true
TELEMETRY_OTLP_ENDPOINT=localhost:4317
```

## Integration Steps

### 1. Update your domain/config.go

Add telemetry configuration to your Config struct:

```go
type Config struct {
    // ... existing fields ...
    
    Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type TelemetryConfig struct {
    Enabled      bool   `mapstructure:"enabled" env:"TELEMETRY_ENABLED" default:"false"`
    OTLPEndpoint string `mapstructure:"otlp_endpoint" env:"TELEMETRY_OTLP_ENDPOINT" default:"localhost:4317"`
}
```

### 2. Update your bootstrap module

Add the telemetry module to your fx application:

```go
import (
    "github.com/emoss08/trenova/internal/infrastructure/telemetry"
)

// In your bootstrap function
fx.New(
    // ... existing modules ...
    telemetry.Module(),
    // ... other modules ...
)
```

### 3. Update your HTTP server setup

Add the observability middleware to your Fiber app:

```go
func NewHTTPServer(
    logger *zerolog.Logger,
    telemetryMetrics *telemetry.Metrics,
    // ... other dependencies
) *fiber.App {
    app := fiber.New(fiber.Config{
        // ... existing config ...
    })
    
    // Add telemetry middleware
    app.Use(telemetry.NewTracingMiddleware("trenova"))
    app.Use(telemetry.NewMetricsMiddleware(telemetryMetrics))
    app.Use(telemetry.NewLoggingMiddleware(logger))
    app.Use(telemetry.NewRecoveryMiddleware(logger))
    
    // ... rest of your setup
}
```

### 4. Instrument your database

In your database initialization:

```go
func NewDatabase(
    config *domain.Config,
    logger *zerolog.Logger,
    metrics *telemetry.Metrics,
) (*bun.DB, error) {
    // ... existing database setup ...
    
    // Add telemetry instrumentation
    telemetry.InstrumentDatabase(db, metrics)
    
    // Start database metrics collection
    dbMetrics := telemetry.NewDatabaseMetrics(db, metrics)
    // Remember to call dbMetrics.Stop() on shutdown
    
    return db, nil
}
```

### 5. Instrument Redis

In your Redis initialization:

```go
func NewRedisClient(
    config *domain.Config,
    metrics *telemetry.Metrics,
) (*redis.Client, error) {
    // ... existing Redis setup ...
    
    // Add telemetry instrumentation
    telemetry.InstrumentRedis(client, metrics)
    
    return client, nil
}
```

## Running the Stack

1. Start the infrastructure:
   ```bash
   docker-compose -f docker-compose-local.yml up -d
   ```

2. Access the services:
   - Grafana: http://localhost:3000 (admin/admin)
   - Loki: http://localhost:3100
   - Tempo: http://localhost:3200
   - Mimir: http://localhost:9009

3. View dashboards:
   - Navigate to Grafana
   - Go to Dashboards > Trenova > Application Overview

## Adding Custom Metrics

Use the metrics instance to record custom metrics:

```go
// In your service code
func (s *MyService) ProcessOrder(ctx context.Context, order Order) error {
    start := time.Now()
    
    // Your business logic here
    err := s.doProcessing(ctx, order)
    
    // Record custom metric
    s.metrics.RecordQueueJob("order_processing", time.Since(start), err)
    
    return err
}
```

## Tracing Best Practices

1. **Add spans for important operations:**
   ```go
   func (s *Service) ImportantOperation(ctx context.Context) error {
       ctx, span := otel.Tracer("service").Start(ctx, "ImportantOperation")
       defer span.End()
       
       // Your code here
       
       return nil
   }
   ```

2. **Add attributes to spans:**
   ```go
   span.SetAttributes(
       attribute.String("user.id", userID),
       attribute.Int("items.count", len(items)),
   )
   ```

3. **Record errors:**
   ```go
   if err != nil {
       span.RecordError(err)
       span.SetStatus(codes.Error, err.Error())
   }
   ```

## Logging with Trace Correlation

Use the context-aware logger:

```go
logger := telemetry.LoggerWithContext(ctx, s.logger)
logger.Info().
    Str("order_id", orderID).
    Msg("Processing order")
```

This will automatically add trace_id and span_id to your logs, enabling correlation between logs and traces.

## Troubleshooting

1. **No metrics showing up:**
   - Check that TELEMETRY_ENABLED=true
   - Verify OTLP endpoint is correct
   - Check OpenTelemetry Collector logs

2. **No traces appearing:**
   - Ensure the tracing middleware is added
   - Check Tempo is receiving data
   - Verify service name is configured

3. **Database metrics missing:**
   - Confirm postgres-exporter is running
   - Check connection string is correct
   - Verify metrics are being scraped