# Observability Guide

This guide explains how to use the tracing and metrics capabilities in the Trenova application.

## Table of Contents

- [Configuration](#configuration)
- [Tracing](#tracing)
- [Metrics](#metrics)
- [Using in Services](#using-in-services)
- [Using in Repositories](#using-in-repositories)
- [Custom Business Metrics](#custom-business-metrics)
- [Best Practices](#best-practices)

## Configuration

### Development Environment

By default, observability is **disabled** in development. To enable it for testing:

```yaml
# config/config.yaml
monitoring:
  metrics:
    enabled: true  # Enable metrics collection
    provider: prometheus
    port: 9090
    path: /metrics
    namespace: trenova
    subsystem: api
  tracing:
    enabled: true  # Enable distributed tracing
    provider: stdout  # Use stdout for local development
    endpoint: localhost:4318
    service_name: trenova-api
    sampling_rate: 1.0  # Sample 100% in dev (use 0.1 in prod)
```

### Production Environment

For production, use external backends:

```yaml
monitoring:
  metrics:
    enabled: true
    provider: prometheus
    port: 9090
    path: /metrics
    namespace: trenova
    subsystem: api
  tracing:
    enabled: true
    provider: otlp  # or otlp-grpc, zipkin
    endpoint: your-jaeger-endpoint:4318
    service_name: trenova-api
    sampling_rate: 0.1  # Sample 10% of requests
```

## Tracing

### Automatic HTTP Tracing

All HTTP requests are automatically traced through the middleware. No additional code needed.

### Creating Spans in Services

```go
package services

import (
    "context"
    "github.com/emoss08/trenova/internal/infrastructure/observability"
)

type ShipmentService struct {
    tracer *observability.TracerProvider
    metrics *observability.MetricsRegistry
    // ... other dependencies
}

func (s *ShipmentService) CreateShipment(ctx context.Context, data ShipmentData) (*Shipment, error) {
    // Start a new span for this operation
    ctx, span := s.tracer.StartSpan(ctx, "ShipmentService.CreateShipment")
    defer span.End()
    
    // Add attributes to the span
    s.tracer.SetAttributes(ctx,
        attribute.String("customer.id", data.CustomerID),
        attribute.String("shipment.type", data.Type),
        attribute.Float64("shipment.weight", data.Weight),
    )
    
    // Validate the shipment
    if err := s.validateShipment(ctx, data); err != nil {
        // Record errors in the span
        s.tracer.RecordError(ctx, err)
        return nil, err
    }
    
    // Add an event to track important milestones
    s.tracer.AddEvent(ctx, "shipment.validated",
        attribute.String("status", "success"),
    )
    
    // Create the shipment
    shipment, err := s.repo.Create(ctx, data)
    if err != nil {
        s.tracer.RecordError(ctx, err)
        return nil, err
    }
    
    // Record business metrics
    s.metrics.RecordShipment(
        "created",
        data.Type,
        data.CustomerID,
        time.Since(startTime).Seconds(),
    )
    
    return shipment, nil
}

func (s *ShipmentService) validateShipment(ctx context.Context, data ShipmentData) error {
    // Child spans automatically inherit the parent trace
    ctx, span := s.tracer.StartSpan(ctx, "ShipmentService.validateShipment")
    defer span.End()
    
    // Validation logic...
    
    return nil
}
```

### Propagating Context

Always pass context through your call chain to maintain trace continuity:

```go
func (s *ShipmentService) ProcessBatch(ctx context.Context, shipments []ShipmentData) error {
    ctx, span := s.tracer.StartSpan(ctx, "ShipmentService.ProcessBatch")
    defer span.End()
    
    for i, shipment := range shipments {
        // Create a span for each shipment in the batch
        ctx, itemSpan := s.tracer.StartSpan(ctx, "ProcessBatchItem",
            trace.WithAttributes(attribute.Int("batch.index", i)))
        
        if err := s.processShipment(ctx, shipment); err != nil {
            s.tracer.RecordError(ctx, err)
            itemSpan.End()
            continue
        }
        
        itemSpan.End()
    }
    
    return nil
}
```

## Metrics

### Recording HTTP Metrics

HTTP metrics are automatically collected. No additional code needed.

### Database Metrics

```go
func (r *ShipmentRepository) Create(ctx context.Context, data ShipmentData) (*Shipment, error) {
    start := time.Now()
    
    // Your database operation
    result, err := r.db.NewInsert().
        Model(&data).
        Returning("*").
        Exec(ctx)
    
    // Record database metrics
    duration := time.Since(start).Seconds()
    operation := "insert"
    table := "shipments"
    
    if err != nil {
        r.metrics.RecordDatabaseQuery(operation, table, "error", duration)
        return nil, err
    }
    
    r.metrics.RecordDatabaseQuery(operation, table, "success", duration)
    return shipment, nil
}
```

### Cache Metrics

```go
func (c *CacheService) Get(ctx context.Context, key string) (interface{}, error) {
    start := time.Now()
    
    value, err := c.redis.Get(ctx, key).Result()
    duration := time.Since(start).Seconds()
    
    if err == redis.Nil {
        c.metrics.RecordCacheOperation("get", "miss", duration)
        return nil, nil
    } else if err != nil {
        c.metrics.RecordCacheOperation("get", "error", duration)
        return nil, err
    }
    
    c.metrics.RecordCacheOperation("get", "hit", duration)
    return value, nil
}
```

## Using in Services

### Service with Full Observability

```go
package services

import (
    "context"
    "time"
    
    "github.com/emoss08/trenova/internal/infrastructure/observability"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type DocumentServiceParams struct {
    fx.In
    
    Tracer   *observability.TracerProvider
    Metrics  *observability.MetricsRegistry
    Logger   *zap.Logger
    Repo     DocumentRepository
}

type DocumentService struct {
    tracer  *observability.TracerProvider
    metrics *observability.MetricsRegistry
    logger  *zap.Logger
    repo    DocumentRepository
}

func NewDocumentService(p DocumentServiceParams) *DocumentService {
    return &DocumentService{
        tracer:  p.Tracer,
        metrics: p.Metrics,
        logger:  p.Logger,
        repo:    p.Repo,
    }
}

func (s *DocumentService) ProcessDocument(ctx context.Context, docID string) error {
    // Start timing for metrics
    start := time.Now()
    
    // Create a span for tracing
    ctx, span := s.tracer.StartSpan(ctx, "DocumentService.ProcessDocument",
        trace.WithAttributes(attribute.String("document.id", docID)))
    defer span.End()
    
    // Get trace ID for logging correlation
    traceID := observability.GetTraceID(ctx)
    logger := s.logger.With(zap.String("trace_id", traceID))
    
    logger.Info("Processing document", zap.String("document_id", docID))
    
    // Fetch document
    doc, err := s.repo.GetByID(ctx, docID)
    if err != nil {
        s.tracer.RecordError(ctx, err)
        logger.Error("Failed to fetch document", zap.Error(err))
        s.metrics.RecordDocument(doc.Type, "fetch_error", time.Since(start).Seconds())
        return err
    }
    
    // Add document info to span
    s.tracer.SetAttributes(ctx,
        attribute.String("document.type", doc.Type),
        attribute.Int64("document.size", doc.Size),
        attribute.String("document.status", doc.Status),
    )
    
    // Process based on type
    switch doc.Type {
    case "invoice":
        err = s.processInvoice(ctx, doc)
    case "manifest":
        err = s.processManifest(ctx, doc)
    default:
        err = s.processGeneric(ctx, doc)
    }
    
    if err != nil {
        s.tracer.RecordError(ctx, err)
        s.metrics.RecordDocument(doc.Type, "processing_error", time.Since(start).Seconds())
        return err
    }
    
    // Record success metrics
    s.metrics.RecordDocument(doc.Type, "processed", time.Since(start).Seconds())
    
    // Add completion event
    s.tracer.AddEvent(ctx, "document.processed",
        attribute.String("document.id", docID),
        attribute.Float64("duration_seconds", time.Since(start).Seconds()),
    )
    
    logger.Info("Document processed successfully",
        zap.Float64("duration_seconds", time.Since(start).Seconds()))
    
    return nil
}
```

## Using in Repositories

### Repository with Observability

```go
package repositories

import (
    "context"
    "database/sql"
    "time"
    
    "github.com/emoss08/trenova/internal/infrastructure/observability"
    "go.opentelemetry.io/otel/attribute"
)

type ShipmentRepository struct {
    db      *bun.DB
    tracer  *observability.TracerProvider
    metrics *observability.MetricsRegistry
}

func (r *ShipmentRepository) GetByID(ctx context.Context, id string) (*Shipment, error) {
    // Create a span for the database operation
    ctx, span := r.tracer.StartSpan(ctx, "ShipmentRepository.GetByID",
        trace.WithAttributes(
            attribute.String("db.operation", "select"),
            attribute.String("db.table", "shipments"),
            attribute.String("shipment.id", id),
        ))
    defer span.End()
    
    start := time.Now()
    
    var shipment Shipment
    err := r.db.NewSelect().
        Model(&shipment).
        Where("id = ?", id).
        Scan(ctx)
    
    duration := time.Since(start).Seconds()
    
    if err == sql.ErrNoRows {
        // Not found is not an error for metrics
        r.metrics.RecordDatabaseQuery("select", "shipments", "not_found", duration)
        return nil, err
    } else if err != nil {
        // Record actual errors
        r.tracer.RecordError(ctx, err)
        r.metrics.RecordDatabaseQuery("select", "shipments", "error", duration)
        return nil, err
    }
    
    // Record success
    r.metrics.RecordDatabaseQuery("select", "shipments", "success", duration)
    
    // Add query stats to span
    span.SetAttributes(
        attribute.Float64("db.query.duration_seconds", duration),
        attribute.Bool("db.query.found", true),
    )
    
    return &shipment, nil
}

func (r *ShipmentRepository) ListByCustomer(ctx context.Context, customerID string, limit int) ([]*Shipment, error) {
    ctx, span := r.tracer.StartSpan(ctx, "ShipmentRepository.ListByCustomer")
    defer span.End()
    
    start := time.Now()
    
    var shipments []*Shipment
    err := r.db.NewSelect().
        Model(&shipments).
        Where("customer_id = ?", customerID).
        Limit(limit).
        Scan(ctx)
    
    duration := time.Since(start).Seconds()
    
    // Record metrics
    if err != nil {
        r.tracer.RecordError(ctx, err)
        r.metrics.RecordDatabaseQuery("select", "shipments", "error", duration)
        return nil, err
    }
    
    r.metrics.RecordDatabaseQuery("select", "shipments", "success", duration)
    
    // Add detailed attributes
    span.SetAttributes(
        attribute.String("customer.id", customerID),
        attribute.Int("result.count", len(shipments)),
        attribute.Float64("db.query.duration_seconds", duration),
    )
    
    return shipments, nil
}
```

## Custom Business Metrics

### Recording Business Events

```go
// In your service layer
func (s *ComplianceService) CheckHazmatCompliance(ctx context.Context, shipment *Shipment) error {
    ctx, span := s.tracer.StartSpan(ctx, "ComplianceService.CheckHazmatCompliance")
    defer span.End()
    
    start := time.Now()
    
    // Perform compliance check
    violations := s.runComplianceChecks(ctx, shipment)
    
    // Record compliance metrics
    if len(violations) > 0 {
        s.metrics.RecordCompliance("hazmat", "failed", time.Since(start).Seconds())
        
        // Add violation details to span
        for _, violation := range violations {
            s.tracer.AddEvent(ctx, "compliance.violation",
                attribute.String("violation.type", violation.Type),
                attribute.String("violation.severity", violation.Severity),
            )
        }
        
        return ErrComplianceViolation
    }
    
    s.metrics.RecordCompliance("hazmat", "passed", time.Since(start).Seconds())
    return nil
}
```

### Tracking User Activity

```go
func (s *AuthService) Login(ctx context.Context, credentials Credentials) (*User, error) {
    ctx, span := s.tracer.StartSpan(ctx, "AuthService.Login")
    defer span.End()
    
    user, err := s.authenticate(ctx, credentials)
    if err != nil {
        s.metrics.RecordError("authentication", "/login")
        return nil, err
    }
    
    // Track active users
    s.metrics.IncrementActiveUsers()
    
    // Add user context to span
    observability.SetUserContext(ctx, user.ID, user.OrganizationID)
    
    return user, nil
}

func (s *AuthService) Logout(ctx context.Context, userID string) error {
    ctx, span := s.tracer.StartSpan(ctx, "AuthService.Logout")
    defer span.End()
    
    // Decrement active users
    s.metrics.DecrementActiveUsers()
    
    return s.invalidateSession(ctx, userID)
}
```

## Best Practices

### 1. Always Pass Context

```go
// ✅ Good - context flows through
func (s *Service) DoWork(ctx context.Context) error {
    return s.repo.Save(ctx, data)
}

// ❌ Bad - breaks trace continuity
func (s *Service) DoWork() error {
    return s.repo.Save(context.Background(), data)
}
```

### 2. Name Spans Descriptively

```go
// ✅ Good - clear hierarchy
ctx, span := tracer.StartSpan(ctx, "ShipmentService.CreateShipment")
ctx, span := tracer.StartSpan(ctx, "ShipmentRepository.Insert")

// ❌ Bad - unclear naming
ctx, span := tracer.StartSpan(ctx, "create")
ctx, span := tracer.StartSpan(ctx, "db_operation")
```

### 3. Add Relevant Attributes

```go
// ✅ Good - useful for debugging
span.SetAttributes(
    attribute.String("customer.id", customerID),
    attribute.String("shipment.type", shipmentType),
    attribute.Float64("shipment.value", value),
)

// ❌ Bad - too generic
span.SetAttributes(
    attribute.String("id", id),
    attribute.String("type", "data"),
)
```

### 4. Record Errors Properly

```go
// ✅ Good - error recorded with context
if err != nil {
    tracer.RecordError(ctx, err)
    span.SetStatus(codes.Error, "Failed to process shipment")
    return err
}

// ❌ Bad - error not recorded in trace
if err != nil {
    return err
}
```

### 5. Avoid High Cardinality Labels

```go
// ✅ Good - bounded cardinality
metrics.RecordHTTPRequest(method, "/api/v1/shipments/:id", status)

// ❌ Bad - unbounded cardinality
metrics.RecordHTTPRequest(method, "/api/v1/shipments/abc-123-def", status)
```

### 6. Use Sampling in Production

```yaml
# ✅ Good - reasonable sampling
tracing:
  sampling_rate: 0.1  # 10% sampling

# ❌ Bad - too much data in production
tracing:
  sampling_rate: 1.0  # 100% sampling
```

### 7. Correlate Logs with Traces

```go
// ✅ Good - logs include trace ID
logger := s.logger.With(
    zap.String("trace_id", observability.GetTraceID(ctx)),
    zap.String("span_id", observability.GetSpanID(ctx)),
)
logger.Info("Processing shipment", zap.String("shipment_id", id))
```

## Viewing Metrics

### Prometheus Endpoint

Access metrics at: `http://localhost:8080/metrics`

### Example Queries

```promql
# Request rate
rate(trenova_http_requests_total[5m])

# 95th percentile latency
histogram_quantile(0.95, rate(trenova_http_request_duration_seconds_bucket[5m]))

# Error rate
rate(trenova_errors_total[5m])

# Active users
trenova_users_active_total

# Database query performance
histogram_quantile(0.99, rate(trenova_database_query_duration_seconds_bucket[5m]))
```

## Viewing Traces

### Development (stdout)

Traces are printed to console in JSON format when `provider: stdout`

### Production (Jaeger)

1. Configure OTLP endpoint in config
2. Access Jaeger UI at `http://jaeger-host:16686`
3. Search by service name: `trenova-api`
4. Filter by trace ID from logs

## Troubleshooting

### Metrics Not Showing

1. Check if metrics are enabled in config
2. Verify the `/metrics` endpoint is accessible
3. Check for errors in application logs

### Traces Not Appearing

1. Verify tracing is enabled in config
2. Check endpoint connectivity
3. Verify sampling rate is > 0
4. Check for schema version conflicts in logs

### High Memory Usage

1. Reduce sampling rate
2. Check for metric cardinality explosion
3. Review batch sizes in tracer configuration

## Environment Variables

You can override config values with environment variables:

```bash
# Enable tracing in development
export MONITORING_TRACING_ENABLED=true
export MONITORING_TRACING_PROVIDER=stdout
export MONITORING_TRACING_SAMPLING_RATE=1.0

# Enable metrics in development
export MONITORING_METRICS_ENABLED=true
export MONITORING_METRICS_PROVIDER=prometheus
```

## Integration with CI/CD

### Running with Observability in Tests

```go
func TestWithObservability(t *testing.T) {
    // Create test tracer
    tracer := observability.NewTestTracer()
    metrics := observability.NewTestMetrics()
    
    service := NewService(tracer, metrics)
    
    // Run test
    err := service.DoWork(context.Background())
    
    // Assert metrics were recorded
    assert.Equal(t, 1, metrics.GetCounter("operations_total"))
}
```

## Support

For questions or issues with observability:

1. Check this README first
2. Review the example implementations
3. Check application logs for errors
4. Reach out to the platform team
