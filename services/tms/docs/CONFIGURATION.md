# Configuration Guide

This guide covers all configuration options available in Trenova. The application uses YAML configuration files and supports environment variable overrides for sensitive data.

## Table of Contents

- [Configuration Files](#configuration-files)
- [Application Settings](#application-settings)
- [Server Configuration](#server-configuration)
- [Database Configuration](#database-configuration)
- [Cache Configuration (Redis)](#cache-configuration-redis)
- [Security Settings](#security-settings)
- [Logging Configuration](#logging-configuration)
- [Monitoring & Observability](#monitoring--observability)
- [Optional Services](#optional-services)
  - [Temporal Workflow Engine](#temporal-workflow-engine)
  - [Message Queue](#message-queue)
  - [Change Data Capture (CDC)](#change-data-capture-cdc)
  - [Object Storage](#object-storage)
  - [Email Service](#email-service)
- [Environment-Specific Configurations](#environment-specific-configurations)

## Configuration Files

The application looks for configuration files in the following order:

1. `config/config.yaml` (default)
2. `config/config.{environment}.yaml` (environment-specific)
3. Environment variables (override YAML values)

## Application Settings

### Basic Application Configuration

```yaml
app:
  name: trenova                    # Application name (used in monitoring/tracing)
  env: development                  # Environment: development, staging, production, test
  debug: true                       # Enable debug mode (verbose logging)
  version: "0.7.4-preview"         # Application version
```

## Server Configuration

### HTTP Server Settings

```yaml
server:
  host: 0.0.0.0                    # Server bind address
  port: 8080                        # Server port
  mode: debug                       # Gin mode: debug, release, test
  
  # Timeout configurations (with time units)
  read_timeout: 30s                 # Maximum duration for reading request
  write_timeout: 30s                # Maximum duration for writing response
  idle_timeout: 120s                # Maximum idle time for keep-alive connections
  shutdown_timeout: 10s             # Graceful shutdown timeout
```

### CORS Configuration

```yaml
server:
  cors:
    enabled: true
    allowed_origins:
      - http://localhost:5173       # Frontend development server
      - http://localhost:3000       # Alternative frontend port
    allowed_methods:
      - GET
      - POST
      - PUT
      - DELETE
      - PATCH
      - OPTIONS
    allowed_headers:
      - Authorization
      - Content-Type
      - Accept
    expose_headers:
      - X-Request-Id
    credentials: true              # Allow cookies/auth headers
    max_age: 86400                # Preflight cache duration (seconds)
```

## Database Configuration

### PostgreSQL Settings

```yaml
database:
  host: localhost
  port: 5432
  name: trenova_go_db
  user: postgres
  
  # Password management
  password_source: env             # Options: env, file, secret
  password: postgres               # Used when password_source: env
  password_file: /path/to/file    # Used when password_source: file
  password_secret: secret-name     # Used when password_source: secret
  
  sslmode: disable                 # Options: disable, require, verify-ca, verify-full
  
  # Connection pool settings
  max_idle_conns: 10               # Maximum idle connections
  max_open_conns: 100              # Maximum open connections
  conn_max_lifetime: 3600s         # Maximum connection lifetime
  conn_max_idle_time: 300s         # Maximum idle time before closing
```

**Features:**

- ✅ Connection pooling with health monitoring
- ✅ Automatic retry with exponential backoff
- ✅ Slow query detection and logging
- ✅ Full OpenTelemetry tracing support
- ✅ Prometheus metrics for all operations
- ✅ Secure password handling (environment, file, or secret manager)

## Cache Configuration (Redis)

### Basic Redis Configuration

```yaml
cache:
  provider: redis                  # Options: redis, memory
  host: localhost
  port: 6379
  password: ""                     # Leave empty if no authentication
  db: 0                            # Database number (0-15)
  
  # Connection pool settings
  pool_size: 10                    # Number of connections in pool
  min_idle_conns: 5                # Minimum idle connections
  max_retries: 3                   # Maximum retry attempts
  
  # Timeouts
  dial_timeout: 5s                 # Connection timeout
  read_timeout: 3s                 # Read operation timeout
  write_timeout: 3s                # Write operation timeout
  pool_timeout: 4s                 # Pool checkout timeout
  
  # Connection lifecycle
  conn_max_idle_time: 5m           # Maximum idle time
  conn_max_lifetime: 30m           # Maximum connection lifetime
  
  # Retry backoff
  min_retry_backoff: 8ms           # Minimum retry delay
  max_retry_backoff: 512ms         # Maximum retry delay
  
  # Performance tuning
  default_ttl: 5m                  # Default TTL for cache entries
  enable_pipelining: false         # Enable command pipelining
  slow_log_threshold: 50ms         # Threshold for slow command logging
```

### Redis Cluster Mode

For high availability and horizontal scaling:

```yaml
cache:
  cluster_mode: true
  cluster_nodes:
    - redis-node1:6379
    - redis-node2:6379
    - redis-node3:6379
  # Other settings remain the same
```

**Setup Guide:** [Redis Cluster Tutorial](https://redis.io/docs/manual/scaling/)

### Redis Sentinel Mode

For high availability with automatic failover:

```yaml
cache:
  sentinel_mode: true
  master_name: mymaster            # Sentinel master name
  sentinel_addrs:
    - sentinel1:26379
    - sentinel2:26379
    - sentinel3:26379
  sentinel_password: ""            # Sentinel authentication (if required)
  # Other settings remain the same
```

**Setup Guide:** [Redis Sentinel Documentation](https://redis.io/docs/manual/sentinel/)

**Features:**

- ✅ Support for standalone, cluster, and sentinel modes
- ✅ Connection pooling with configurable limits
- ✅ Automatic retries with backoff
- ✅ Slow command detection and logging
- ✅ Full OpenTelemetry tracing
- ✅ Prometheus metrics for cache hits/misses
- ✅ Health monitoring with periodic pings

## Security Settings

### Session Configuration

```yaml
security:
  session:
    secret: your-secret-key-change-in-production  # Min 32 characters
    name: trv-session-id           # Session cookie name
    max_age: 24h                   # Session duration
    http_only: true                # Prevent JavaScript access
    secure: false                  # Set to true for HTTPS only
    same_site: lax                 # Options: strict, lax, none
    domain: ""                     # Leave empty for current domain
    path: "/"
    refresh_window: 1h             # Auto-refresh if expiring soon
```

### API Token Configuration

```yaml
security:
  api_token:
    enabled: true
    default_expiry: 720h           # 30 days - Default expiry for API tokens
    max_expiry: 8760h              # 365 days - Maximum allowed expiry
    max_tokens_per_user: 10        # Maximum tokens per user
```

### Rate Limiting

```yaml
security:
  rate_limit:
    enabled: true
    requests_per_minute: 60        # Request limit per minute
    burst_size: 10                 # Burst capacity
    cleanup_interval: 1m           # Cleanup interval for expired entries
```

### CSRF Protection

```yaml
security:
  csrf:
    token_name: csrf_token         # Token parameter name
    header_name: X-CSRF-Token      # Header name for CSRF token
```

## Logging Configuration

```yaml
logging:
  level: debug                     # Options: debug, info, warn, error
  format: json                     # Options: json, text
  output: stdout                   # Options: stdout, stderr, file
  sampling: false                  # Enable sampling (production optimization)
  stacktrace: true                 # Include stacktrace on errors
  
  # File logging (when output: file)
  file:
    path: logs/trenova.log
    max_size: 100                  # Maximum file size in MB
    max_age: 30                    # Maximum age in days
    max_backups: 10                # Maximum number of backups
    compress: true                 # Compress rotated files
```

## Monitoring & Observability

### Health Checks

```yaml
monitoring:
  health:
    path: /health                  # Main health endpoint
    readiness_path: /ready         # Readiness probe endpoint
    liveness_path: /live           # Liveness probe endpoint
    check_interval: 30s            # Health check interval
    timeout: 5s                    # Health check timeout
```

### Metrics (Prometheus)

```yaml
monitoring:
  metrics:
    enabled: false                 # Enable metrics collection
    provider: prometheus           # Options: prometheus, datadog
    port: 9090                     # Metrics endpoint port
    path: /metrics                 # Metrics endpoint path
    namespace: trenova             # Metric namespace
    subsystem: api                 # Metric subsystem
    api_key: ""                    # API key for Datadog (if used)
```

**Setup Guide:** [Prometheus Getting Started](https://prometheus.io/docs/prometheus/latest/getting_started/)

### Distributed Tracing

```yaml
monitoring:
  tracing:
    enabled: false                 # Enable distributed tracing
    provider: stdout               # Options: otlp, otlp-grpc, jaeger, zipkin, stdout
    endpoint: localhost:4318       # Tracing endpoint
    service_name: trenova-api      # Service name in traces
    sampling_rate: 1.0             # Sampling rate (0.0-1.0)
```

**Provider Configuration:**

- **OTLP HTTP:** `endpoint: localhost:4318`
- **OTLP gRPC:** `endpoint: localhost:4317`
- **Jaeger:** `endpoint: localhost:14268/api/traces`
- **Zipkin:** `endpoint: http://localhost:9411/api/v2/spans`

**Setup Guides:**

- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/getting-started/)
- [Jaeger Quick Start](https://www.jaegertracing.io/docs/latest/getting-started/)
- [Zipkin Quick Start](https://zipkin.io/pages/quickstart)

## Optional Services

These services are optional and can be configured based on your requirements.

### Temporal Workflow Engine

Temporal is used for reliable background job processing and workflow orchestration.

```yaml
temporal:
  hostPort: "localhost:7233"      # Temporal server address
  security:
    enableEncryption: false        # Enable payload encryption
    encryptionKeyID: "default"    # Key identifier for encryption
    enableCompression: true        # Enable payload compression
    compressionThreshold: 1024     # Compress payloads > 1KB
```

**Environment Variables:**

```bash
# Encryption key (minimum 32 characters)
export TEMPORAL_ENCRYPTION_KEY="your-32-character-encryption-key"

# Or use key-specific variable
export TEMPORAL_ENCRYPTION_KEY_production="production-key"
```

**Production Recommendations:**

- Use [Temporal Cloud](https://temporal.io/cloud) for managed infrastructure
- Enable encryption for sensitive data
- Configure appropriate compression threshold

For detailed setup instructions, see [Temporal Setup Guide](./TEMPORAL.md).

### Message Queue

For asynchronous messaging and event-driven architecture.

```yaml
queue:
  provider: kafka                  # Options: kafka, rabbitmq, redis
  brokers:
    - kafka1:9092
    - kafka2:9092
  consumer_group: trenova-api      # Consumer group name
  topics:
    events: trenova.events         # Event topic
    commands: trenova.commands     # Command topic
```

### Change Data Capture (CDC)

For real-time data synchronization and event streaming.

```yaml
cdc:
  enabled: false
  brokers:
    - kafka1:9092
    - kafka2:9092
  consumer_group: trenova-cdc
  topic_pattern: "trenova.cdc.*"  # Topic pattern for CDC events
  schema_registry_url: "http://schema-registry:8081"
  start_offset: latest             # Options: earliest, latest
  max_retry_attempts: 3
  processing:
    batch_size: 100                # Process in batches
    batch_timeout: 5s              # Batch timeout
    worker_count: 4                # Parallel workers
  subscriptions:
    max_per_user: 10               # Max subscriptions per user
    max_filters: 5                 # Max filters per subscription
    webhook_timeout: 30s           # Webhook call timeout
    retention_period: 168h         # 7 days retention
```

### Object Storage

For file uploads and document management.

```yaml
storage:
  provider: minio                  # Options: minio, s3, local
  endpoint: localhost:9000         # MinIO/S3 endpoint
  access_key: minioadmin           # Access key
  secret_key: minioadmin           # Secret key
  region: us-east-1                # AWS region (for S3)
  bucket: trenova-uploads          # Bucket name
  use_ssl: false                   # Use SSL/TLS
  
  # For local storage
  # provider: local
  # local_path: ./uploads
```

### Email Service

For sending transactional emails.

```yaml
email:
  provider: smtp                   # Options: smtp, sendgrid, ses
  from: noreply@trenova.io        # Default sender
  
  # SMTP configuration
  smtp:
    host: smtp.gmail.com
    port: 587
    username: your-email@gmail.com
    password: your-app-password
    use_tls: true
  
  # For SendGrid or SES
  # api_key: your-api-key
```

## Environment-Specific Configurations

### Development Configuration

```yaml
# config/config.development.yaml
app:
  env: development
  debug: true

server:
  mode: debug

database:
  sslmode: disable

monitoring:
  metrics:
    enabled: false                 # Disable in development
  tracing:
    enabled: false                 # Or use stdout provider
    provider: stdout
    sampling_rate: 1.0             # Sample everything in dev
```

### Production Configuration

```yaml
# config/config.production.yaml
app:
  env: production
  debug: false

server:
  mode: release
  cors:
    secure: true                   # HTTPS only

database:
  sslmode: require                 # Require SSL
  max_open_conns: 200             # Higher connection limit

security:
  session:
    secure: true                   # HTTPS only cookies

logging:
  level: info
  sampling: true                   # Enable log sampling

monitoring:
  metrics:
    enabled: true
  tracing:
    enabled: true
    provider: otlp
    sampling_rate: 0.1             # Sample 10% in production
```

## Configuration Validation

The application validates all configuration on startup. Key validation rules:

### Required Fields
- All top-level sections except optional services must be present
- Fields marked with `validate:"required"` must have values
- Conditional requirements (e.g., `required_if`) are enforced

### Validation Examples

```yaml
# Valid environment values
app:
  env: development  # Must be: development, staging, production, or test
  
# Valid server modes  
server:
  mode: debug      # Must be: debug, release, or test
  
# Valid SSL modes
database:
  sslmode: disable # Must be: disable, require, verify-ca, or verify-full
  
# Session same-site values
security:
  session:
    same_site: lax # Must be: strict, lax, or none
```

### Common Validation Errors

```
# Port out of range
Error: Key: 'Config.Server.Port' Error:Field validation for 'Port' failed on the 'max' tag

# Invalid environment
Error: Key: 'Config.App.Env' Error:Field validation for 'Env' failed on the 'oneof' tag

# Missing required field
Error: Key: 'Config.Temporal.Security.EncryptionKeyID' Error:Field validation for 'EncryptionKeyID' failed on the 'required_if' tag
```

## Environment Variables

Override configuration values using environment variables:

```bash
# Database
export DB_PASSWORD=secret_password
export DB_HOST=production-db.example.com

# Redis
export REDIS_PASSWORD=redis_secret
export REDIS_HOST=redis.example.com

# Security
export SESSION_SECRET=your-very-long-secret-key-min-32-chars

# Temporal Encryption
export TEMPORAL_ENCRYPTION_KEY="your-32-character-encryption-key"
# Or for specific key IDs
export TEMPORAL_ENCRYPTION_KEY_production="production-encryption-key"

# Object Storage (MinIO/S3)
export STORAGE_ACCESS_KEY=your-access-key
export STORAGE_SECRET_KEY=your-secret-key

# Email Service
export SMTP_PASSWORD=your-app-password
export SENDGRID_API_KEY=your-sendgrid-key

# Monitoring
export OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```

## Advanced Topics

### Using Secret Managers

When `password_source: secret` is configured:

**AWS Secrets Manager:**

```yaml
database:
  password_source: secret
  password_secret: arn:aws:secretsmanager:region:account:secret:name
```

**HashiCorp Vault:**

```yaml
database:
  password_source: secret
  password_secret: secret/data/database/postgres
```

**Google Secret Manager:**

```yaml
database:
  password_source: secret
  password_secret: projects/PROJECT_ID/secrets/SECRET_NAME/versions/latest
```

### High Availability Configurations

#### Database High Availability

For production deployments, consider:

- PostgreSQL streaming replication
- PgBouncer for connection pooling
- Read replicas for scaling reads

**Resources:**

- [PostgreSQL High Availability](https://www.postgresql.org/docs/current/high-availability.html)
- [PgBouncer Documentation](https://www.pgbouncer.org/)

#### Redis High Availability

Options:

1. **Redis Sentinel** - Automatic failover for master-slave setup
2. **Redis Cluster** - Horizontal scaling and sharding
3. **Redis Enterprise** - Commercial solution with built-in HA

**Resources:**

- [Redis Persistence](https://redis.io/docs/manual/persistence/)
- [Redis Replication](https://redis.io/docs/manual/replication/)

### Performance Tuning

#### Database Performance

```yaml
database:
  # Adjust based on your workload
  max_idle_conns: 25              # 25% of max_open_conns
  max_open_conns: 100             # Based on available connections
  conn_max_lifetime: 1h           # Rotate connections hourly
  conn_max_idle_time: 10m         # Close idle connections
```

#### Redis Performance

```yaml
cache:
  # For high-throughput applications
  pool_size: 50                   # Increase pool size
  enable_pipelining: true         # Batch commands
  slow_log_threshold: 10ms        # Strict slow query detection
```

### Monitoring Best Practices

1. **Enable metrics in production** for visibility
2. **Use sampling** to reduce overhead (10-20% is usually sufficient)
3. **Set up alerts** for slow queries and high error rates
4. **Monitor connection pools** to avoid exhaustion
5. **Track cache hit rates** to optimize caching strategy

## Troubleshooting

### Common Issues

**Database Connection Errors:**

- Check `sslmode` settings match your PostgreSQL configuration
- Verify network connectivity and firewall rules
- Ensure connection pool settings don't exceed PostgreSQL limits

**Redis Connection Issues:**

- Verify Redis is running and accessible
- Check authentication settings if Redis requires a password
- For cluster/sentinel, ensure all nodes are reachable

**Performance Issues:**

- Enable slow query logging for database and cache
- Check connection pool metrics for exhaustion
- Review trace data for bottlenecks
- Monitor resource usage (CPU, memory, network)

## Complete Configuration Example

Here's a complete configuration file with all available options:

```yaml
# config/config.yaml
app:
  name: trenova
  env: development
  debug: true
  version: "0.7.4"

server:
  host: 0.0.0.0
  port: 8080
  mode: debug
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  shutdown_timeout: 10s
  cors:
    enabled: true
    allowed_origins: ["http://localhost:5173"]
    allowed_methods: [GET, POST, PUT, DELETE, PATCH, OPTIONS]
    allowed_headers: [Authorization, Content-Type, Accept, X-Request-Id]
    expose_headers: [X-Request-Id]
    credentials: true
    max_age: 86400

database:
  host: localhost
  port: 5432
  name: trenova_go_db
  user: postgres
  password_source: env
  password: postgres
  sslmode: disable
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600s
  conn_max_idle_time: 300s

cache:
  provider: redis
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 10
  min_idle_conns: 5
  max_retries: 3
  default_ttl: 5m
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  pool_timeout: 4s
  conn_max_idle_time: 5m
  conn_max_lifetime: 30m
  min_retry_backoff: 8ms
  max_retry_backoff: 512ms
  cluster_mode: false
  sentinel_mode: false
  enable_pipelining: false
  slow_log_threshold: 50ms

security:
  session:
    secret: your-secret-key-change-in-production-min-32-chars
    name: trv-session-id
    max_age: 24h
    http_only: true
    secure: false
    same_site: lax
    domain: ""
    path: "/"
    refresh_window: 1h
  api_token:
    enabled: true
    default_expiry: 720h
    max_expiry: 8760h
    max_tokens_per_user: 10
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    cleanup_interval: 1m
  csrf:
    token_name: csrf_token
    header_name: X-CSRF-Token

logging:
  level: debug
  format: json
  output: stdout
  sampling: false
  stacktrace: true
  file:
    path: logs/trenova.log
    max_size: 100
    max_age: 30
    max_backups: 10
    compress: true

monitoring:
  health:
    path: /health
    readiness_path: /ready
    liveness_path: /live
    check_interval: 30s
    timeout: 5s
  metrics:
    enabled: false
    provider: prometheus
    port: 9090
    path: /metrics
    namespace: trenova
    subsystem: api
  tracing:
    enabled: false
    provider: stdout
    endpoint: localhost:4318
    service_name: trenova-api
    sampling_rate: 1.0

# Optional Services
temporal:
  hostPort: "localhost:7233"
  security:
    enableEncryption: false
    encryptionKeyID: "default"
    enableCompression: true
    compressionThreshold: 1024

queue:
  provider: kafka
  brokers: ["localhost:9092"]
  consumer_group: trenova-api
  topics:
    events: trenova.events

cdc:
  enabled: false
  brokers: ["localhost:9092"]
  consumer_group: trenova-cdc
  topic_pattern: "trenova.cdc.*"
  schema_registry_url: "http://localhost:8081"

storage:
  provider: local
  local_path: ./uploads

email:
  provider: smtp
  from: noreply@trenova.io
  smtp:
    host: smtp.gmail.com
    port: 587
    username: your-email@gmail.com
    password: your-password
    use_tls: true
```

## Additional Resources

- [Twelve-Factor App Configuration](https://12factor.net/config)
- [PostgreSQL Connection Strings](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
- [Redis Configuration](https://redis.io/docs/manual/config/)
- [OpenTelemetry Best Practices](https://opentelemetry.io/docs/reference/specification/resource/sdk/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Temporal Setup Guide](./TEMPORAL.md)
