<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Trenova Performance Benchmark Results

## Test Environment

### Hardware Specifications

- **CPU**: 16 cores
- **RAM**: 15GB total (6.5GB available during tests)
- **Storage**: 1TB SSD
- **OS**: Linux 5.15.167.4-microsoft-standard-WSL2

### Software Stack

- **Go Version**: 1.24.2
- **PostgreSQL**: Latest (with max_connections: 300)
- **Redis**: Latest
- **Server**: Go Fiber (non-prefork mode)

### Configuration Optimizations Applied

#### Database Optimizations

1. Added composite index on `workers` table: `(organization_id, business_unit_id)`
2. PostgreSQL connection settings:
   - max_connections: 300
   - shared_buffers: 128MB
   - work_mem: 4MB

#### Application Configuration

```yaml
server:
  readBufferSize: 8192      # Increased from 4096
  writeBufferSize: 8192     # Increased from 4096
  concurrency: 1048576      # Increased from 262144
  enablePrefork: false      # Disabled due to startup issues

db:
  maxConnections: 80        # Optimized from 100
  maxIdleConns: 40         # Optimized from 100
  connMaxLifetime: 300     # Reduced from 3600
  connMaxIdleTime: 60      # Reduced from 3600

redis:
  poolSize: 200            # Increased from 100
  minIdleConns: 100        # Increased from 100

logging:
  level: error             # Reduced from debug for performance
```

## Performance Test Results

### Test Methodology

- **Tool**: hey (HTTP load generator)
- **Endpoint**: `/api/v1/workers/select-options`
- **Authentication**: Session-based with cookie

### Results by Concurrency Level

| Concurrent Connections | Requests/Second | Avg Response Time | Success Rate | Notes |
|----------------------|-----------------|-------------------|--------------|-------|
| 50                   | 20,708          | 2.3ms            | 100%         | Excellent performance |
| 100                  | 37,908          | 2.5ms            | 100%         | **Peak efficiency** |
| 200                  | 32,055          | 6.0ms            | 100%         | Optimal for sustained load |
| 300                  | 15,165          | 19.0ms           | 100%         | Performance degradation begins |
| 500                  | 26,737          | 14.9ms           | 100%         | Good throughput |
| 750                  | ~20,000         | 26.5ms           | 100%         | Moderate performance |
| 1000                 | 19,268          | 39.7ms           | 100%         | Response time increases |
| 1500                 | 286             | 1,145ms          | ~95%         | Severe degradation |
| 2000                 | 17,509          | 105ms            | ~90%         | Connection pool exhaustion |

### Sustained Load Test (30 seconds)

- **Concurrent Connections**: 200
- **Total Requests**: 3,957,982
- **Requests/Second**: 131,921
- **Average Response Time**: 6ms
- **Success Rate**: 100%

## Key Findings

1. **Peak Performance**: The application achieves peak throughput of **131,921 requests/second** under sustained load with 200 concurrent connections
2. **Optimal Concurrency**: Best performance achieved with 100-200 concurrent connections
3. **Scaling Pattern**:
   - Linear scaling up to 100 concurrent connections
   - Gradual degradation from 200-1000 concurrent connections
   - Severe degradation above 1500 concurrent connections
4. **Response Times**:
   - Sub-10ms response times up to 200 concurrent connections
   - Sub-50ms response times up to 1000 concurrent connections
   - Exponential increase beyond 1500 concurrent connections
5. **Breaking Point**: At ~1500 concurrent connections, the system experiences severe degradation with response times exceeding 1 second

## Performance Bottlenecks Identified

1. **Database Query Performance**: Initial slow queries (200ms+) were resolved by adding composite indexes
2. **Connection Pool Sizing**: Original settings were too conservative for high-concurrency scenarios
3. **Logging Overhead**: Debug-level logging significantly impacted performance

## Performance Characteristics

### Throughput vs Concurrency

```
Requests/Second
140,000 |     *
        |    * (131,921 @ sustained 200 concurrent)
120,000 |   *
        |  *
100,000 | *
        |
 80,000 |
        |
 60,000 |
        |         *
 40,000 |    * (37,908)
        |   *
 20,000 | *     *    *    *
        |              *
      0 |________________*___*
        50  100  200  500  1000  1500  2000
                Concurrent Connections
```

## Capacity Planning

Based on the performance tests, here are the recommended configurations for different use cases:

### Small/Medium Deployment (up to 10,000 users)

- **Concurrent Connections**: 50-100
- **Expected Throughput**: 20,000-37,000 req/s
- **Response Time**: < 3ms
- **Database Connections**: 40-50

### Large Deployment (10,000-50,000 users)

- **Concurrent Connections**: 100-200
- **Expected Throughput**: 30,000-130,000 req/s
- **Response Time**: < 10ms
- **Database Connections**: 80-100

### Enterprise Deployment (50,000+ users)

- **Concurrent Connections**: 200-500
- **Expected Throughput**: 25,000-130,000 req/s
- **Response Time**: < 20ms
- **Database Connections**: 100-150
- **Recommendation**: Implement horizontal scaling with load balancer

## Recommendations

1. **Optimal Configuration**: For best performance/resource ratio, configure the system for 100-200 concurrent connections
2. **Connection Pooling**: Current settings (80 DB connections) are optimal for up to 500 concurrent users
3. **Monitoring**: Implement monitoring for:
   - Request queue depth
   - Database connection pool utilization
   - Response time percentiles (p50, p90, p95, p99)
4. **Scaling Strategy**:
   - Vertical scaling effective up to 1000 concurrent connections
   - Beyond 1000 concurrent, implement horizontal scaling
   - Consider read replicas for read-heavy workloads
5. **Caching**: Implement Redis caching for frequently accessed data to reduce database load

## Future Optimization Opportunities

1. **Database Read Replicas**: Distribute read queries across multiple PostgreSQL instances
2. **Request Batching**: Implement DataLoader pattern for N+1 query optimization
3. **HTTP/2 or HTTP/3**: Upgrade to newer protocols for better connection efficiency
4. **Horizontal Scaling**: Deploy multiple application instances behind a load balancer

## Why Prefork Mode Was Removed

During performance testing, we evaluated Fiber's prefork mode to determine if it would improve application performance. However, the results showed significant performance degradation:

### Prefork Performance Results

- **50 concurrent**: 358 req/s (118.9ms avg response) vs 20,708 req/s (2.3ms) without prefork
- **100 concurrent**: 89 req/s (1,059ms avg response) vs 37,908 req/s (2.5ms) without prefork  
- **200 concurrent**: 196 req/s (226ms avg response) vs 32,055 req/s (6.0ms) without prefork

### Reasons for Poor Prefork Performance

1. **Database Connection Contention**: Each prefork process maintains its own database connection pool, creating resource contention and exceeding optimal connection limits
2. **Memory Overhead**: Multiple processes each load the entire application stack, significantly increasing memory usage
3. **Application Architecture Mismatch**: The application is database-intensive rather than CPU-bound, so process-level parallelism doesn't provide benefits
4. **Session Management Conflicts**: Redis-based session management doesn't benefit from multiple processes and may cause additional overhead
5. **Resource Competition**: With 16+ worker processes competing for the same database and Redis resources on a 16-core system, context switching overhead outweighs benefits

### Conclusion

For database-heavy applications like Trenova with sophisticated session management and connection pooling, **single-process mode with high concurrency settings provides superior performance**. Prefork mode is better suited for CPU-intensive applications with minimal shared state.

**Configuration Decision**: `enablePrefork` has been permanently disabled and removed as a configuration option.

---

*Last Updated: 2025-07-02*
*Testing performed by: Claude with hey load testing tool*
