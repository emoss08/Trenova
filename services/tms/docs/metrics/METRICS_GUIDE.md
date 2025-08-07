<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Metrics Guide

## Overview

Trenova uses Prometheus for metrics collection and monitoring. The read/write separation feature includes comprehensive metrics to monitor performance and health.

## Available Metrics

### Database Metrics

#### Connection Metrics

- `trenova_database_connections_total{connection_type, status}` - Total connection attempts
- `trenova_database_connection_pool_stats{connection_type, stat_type}` - Connection pool statistics

#### Operation Metrics

- `trenova_database_operations_total{operation_type, connection_type}` - Total database operations
- `trenova_database_operation_duration_seconds{operation_type, connection_type}` - Operation duration histogram

#### Read/Write Distribution

- `trenova_database_read_write_distribution_total{connection_name, operation_type}` - Distribution of operations

#### Replica Health

- `trenova_database_replica_health{replica_name}` - Replica health status (1=healthy, 0=unhealthy)
- `trenova_database_replication_lag_seconds{replica_name}` - Replication lag in seconds

## Accessing Metrics

### Metrics Endpoint

Metrics are exposed at `/metrics`:

```bash
curl http://localhost:3001/metrics
```

### Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'trenova'
    static_configs:
      - targets: ['localhost:3001']
    scrape_interval: 15s
```

## Grafana Dashboards

### Database Performance Dashboard

```json
{
  "title": "Trenova Database Performance",
  "panels": [
    {
      "title": "Read/Write Distribution",
      "targets": [{
        "expr": "rate(trenova_database_read_write_distribution_total[5m])"
      }]
    },
    {
      "title": "Operation Latency (p95)",
      "targets": [{
        "expr": "histogram_quantile(0.95, rate(trenova_database_operation_duration_seconds_bucket[5m]))"
      }]
    },
    {
      "title": "Replica Health",
      "targets": [{
        "expr": "trenova_database_replica_health"
      }]
    },
    {
      "title": "Replication Lag",
      "targets": [{
        "expr": "trenova_database_replication_lag_seconds"
      }]
    }
  ]
}
```

## Useful Queries

### Performance Monitoring

```promql
# Average query duration by operation type
rate(trenova_database_operation_duration_seconds_sum[5m]) / 
rate(trenova_database_operation_duration_seconds_count[5m])

# Read vs Write ratio
sum(rate(trenova_database_read_write_distribution_total{operation_type="read"}[5m])) /
sum(rate(trenova_database_read_write_distribution_total{operation_type="write"}[5m]))

# Connection pool utilization
trenova_database_connection_pool_stats{stat_type="active"} / 
trenova_database_connection_pool_stats{stat_type="total"}
```

### Health Monitoring

```promql
# Unhealthy replicas
trenova_database_replica_health == 0

# High replication lag
trenova_database_replication_lag_seconds > 10

# Connection failures
rate(trenova_database_connections_total{status="failure"}[5m]) > 0
```

## Alerts

### Example AlertManager Rules

```yaml
groups:
  - name: database
    rules:
      - alert: DatabaseReplicaDown
        expr: trenova_database_replica_health == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database replica {{ $labels.replica_name }} is down"

      - alert: HighReplicationLag
        expr: trenova_database_replication_lag_seconds > 30
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High replication lag on {{ $labels.replica_name }}: {{ $value }}s"

      - alert: HighDatabaseLatency
        expr: |
          histogram_quantile(0.95, 
            rate(trenova_database_operation_duration_seconds_bucket[5m])
          ) > 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "95th percentile database latency > 500ms"
```

## Custom Metrics

To add custom metrics:

```go
import "github.com/emoss08/trenova/internal/pkg/metrics"

// Record a custom operation
start := time.Now()
// ... perform operation ...
metrics.RecordDatabaseOperation("custom_operation", "primary", time.Since(start))

// Update custom gauge
metrics.DatabaseReplicaHealth.WithLabelValues("custom").Set(1.0)
```

## Performance Impact

Metrics collection adds minimal overhead:

- Counter increment: ~10ns
- Histogram observation: ~50ns
- Gauge update: ~20ns

The metrics endpoint itself uses ~1MB of memory and minimal CPU.
