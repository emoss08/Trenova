<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Database Read/Write Separation - Production Checklist

## Pre-Production Checklist

### ✅ Configuration Validation

- [ ] Validate all replica hosts are reachable
- [ ] Ensure replica ports are open and accessible
- [ ] Verify SSL/TLS configuration matches primary
- [ ] Check replica lag threshold is appropriate (recommended: 5-10 seconds)
- [ ] Validate connection pool sizes based on expected load

### ✅ Performance Testing

- [ ] Load test with expected traffic patterns
- [ ] Verify connection overhead is < 1ms
- [ ] Confirm fair distribution across replicas
- [ ] Test failover scenarios under load
- [ ] Measure replication lag under peak load

### ✅ Monitoring Setup

- [ ] Prometheus metrics endpoint configured
- [ ] Grafana dashboards imported
- [ ] Alerts configured for:
  - [ ] Replica health
  - [ ] Replication lag
  - [ ] Connection pool exhaustion
  - [ ] Query latency

### ✅ Security Considerations

- [ ] Use encrypted connections (SSL/TLS) for replicas
- [ ] Ensure replica credentials are properly secured
- [ ] Network isolation between application and databases
- [ ] Audit logging enabled on all databases

## Production Configuration Example

```yaml
db:
  driver: "postgresql"
  host: "${DB_PRIMARY_HOST}"
  port: ${DB_PRIMARY_PORT:-5432}
  username: "${DB_USERNAME}"
  password: "${DB_PASSWORD}"
  database: "${DB_NAME}"
  sslMode: "require"
  
  # Production connection pool settings
  maxConnections: 100
  maxIdleConns: 20
  connMaxLifetime: 3600  # 1 hour
  connMaxIdleTime: 900   # 15 minutes
  
  # Read/Write Separation
  enableReadWriteSeparation: true
  replicaLagThreshold: 10  # 10 seconds
  
  readReplicas:
    - name: "prod-replica-1"
      host: "${DB_REPLICA1_HOST}"
      port: ${DB_REPLICA1_PORT:-5432}
      weight: 2  # More powerful server
      maxConnections: 80
      maxIdleConns: 20
    
    - name: "prod-replica-2"
      host: "${DB_REPLICA2_HOST}"
      port: ${DB_REPLICA2_PORT:-5432}
      weight: 1
      maxConnections: 60
      maxIdleConns: 15
    
    - name: "prod-replica-3"
      host: "${DB_REPLICA3_HOST}"
      port: ${DB_REPLICA3_PORT:-5432}
      weight: 1
      maxConnections: 60
      maxIdleConns: 15
```

## Deployment Steps

### 1. Deploy Without Read/Write Separation First

```yaml
# Initial deployment
enableReadWriteSeparation: false
```

Verify:

- Application starts successfully
- All queries work correctly
- No performance degradation

### 2. Enable Read/Write Separation

```yaml
# Enable feature
enableReadWriteSeparation: true
```

Monitor for 24 hours:

- Check metrics show read distribution
- Verify no increase in errors
- Confirm performance improvement

### 3. Gradual Rollout

If using feature flags:

```go
if featureFlags.IsEnabled("read_write_separation") {
    // Use read/write separation
} else {
    // Use primary only
}
```

## Operational Procedures

### Adding a New Replica

1. Configure new replica in database
2. Test connectivity from application servers
3. Add to configuration:

   ```yaml
   readReplicas:
     - name: "prod-replica-new"
       host: "${NEW_REPLICA_HOST}"
       port: 5432
       weight: 1
   ```

4. Deploy configuration change
5. Monitor metrics for proper distribution

### Removing a Replica

1. Set weight to 0 (stops new connections)
2. Wait for existing connections to drain
3. Remove from configuration
4. Deploy configuration change

### Emergency Procedures

#### All Replicas Down

The system automatically falls back to primary:

```
WARN: no healthy read replicas available, falling back to primary
```

**Actions:**

1. Check replica health
2. Investigate root cause
3. No application changes needed (automatic failover)

#### Primary Database Issues

**Actions:**

1. Promote a replica to primary
2. Update configuration to point to new primary
3. Reconfigure other replicas to follow new primary

## Performance Tuning

### Connection Pool Sizing

```
Total Connections = (Worker Processes × Connections per Worker)

Example:
- 10 app servers
- 10 workers per server
- 2 connections per worker (1 read, 1 write)
= 200 total connections

Distribute across databases:
- Primary: 100 connections (writes + read overflow)
- Replica 1: 50 connections
- Replica 2: 50 connections
```

### Optimal Replica Weights

Based on server capacity:

```yaml
# 16 CPU, 64GB RAM server
weight: 4

# 8 CPU, 32GB RAM server
weight: 2

# 4 CPU, 16GB RAM server
weight: 1
```

## Monitoring Queries

### Replication Lag

```sql
-- On Primary
SELECT 
    client_addr,
    state,
    sync_state,
    replay_lag
FROM pg_stat_replication
ORDER BY replay_lag DESC;

-- On Replica
SELECT 
    now() - pg_last_xact_replay_timestamp() AS replication_lag;
```

### Connection Usage

```sql
-- Active connections by database
SELECT 
    datname,
    usename,
    application_name,
    count(*) as connection_count
FROM pg_stat_activity
WHERE state != 'idle'
GROUP BY datname, usename, application_name
ORDER BY connection_count DESC;
```

### Query Performance

```sql
-- Slow queries
SELECT 
    query,
    mean_exec_time,
    calls,
    total_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- queries taking > 100ms
ORDER BY mean_exec_time DESC
LIMIT 20;
```

## Success Metrics

After enabling read/write separation, you should see:

1. **Reduced Primary CPU**: 30-50% reduction
2. **Lower Query Latency**: 20-40% improvement
3. **Higher Throughput**: 2-3x read capacity
4. **Better Resource Utilization**: Even distribution

## Troubleshooting Guide

### Issue: Uneven replica distribution

**Check:**

```bash
curl http://localhost:9090/metrics | grep trenova_database_read_write_distribution_total
```

**Fix:**

- Adjust replica weights
- Check replica health status
- Verify round-robin is working

### Issue: High replication lag

**Check:**

```sql
SELECT replay_lag FROM pg_stat_replication;
```

**Fix:**

- Increase `replicaLagThreshold`
- Add more replica resources
- Optimize primary write load

### Issue: Connection pool exhaustion

**Check:**

```bash
curl http://localhost:9090/metrics | grep trenova_database_connection_pool_stats
```

**Fix:**

- Increase `maxConnections`
- Add more replicas
- Optimize query performance

## Final Notes

The read/write separation system is designed to be:

1. **Safe**: Automatic fallback to primary
2. **Performant**: Minimal overhead (<1ms)
3. **Observable**: Comprehensive metrics
4. **Maintainable**: Clear configuration

Regular monitoring and tuning based on actual load patterns will ensure optimal performance.
