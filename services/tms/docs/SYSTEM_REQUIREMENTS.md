# Trenova Self-Hosting System Requirements

> **Important**: This document is specifically for organizations planning to **self-host** Trenova on their own infrastructure. If you're interested in our **Managed Hosting Services**, these requirements don't apply to you - we handle all infrastructure, scaling, and maintenance.

This document outlines the system resource requirements needed to self-host the Trenova application in a production environment. These requirements are based on the current production architecture and can be scaled according to your user base and usage patterns.

## Overview

Trenova is a containerized application with integrated workflow orchestration that provides a comprehensive Transportation Management System (TMS). When self-hosting, you'll be responsible for managing all these components on your own infrastructure. The application is designed to be scalable and can be adjusted based on your organization's needs and available resources.

## Quick Sizing Guide

| User Count | Concurrent Users | CPU Cores | RAM | Storage | Expected Performance |
|------------|------------------|-----------|-----|---------|---------------------|
| < 10,000 | 50 | 8 | 16GB | 100GB SSD | 20,000+ req/s |
| 10K-50K | 100-200 | 16 | 32GB | 200GB SSD | 30,000-130,000 req/s |
| 50K-100K | 200-500 | 24 | 64GB | 500GB NVMe | 25,000-130,000 req/s |
| 100K+ | 500+ | 32+ | 96GB+ | 1TB+ NVMe | Requires clustering |

> **Performance Note**: These specifications are based on real-world performance testing achieving up to 131,921 requests/second sustained load.

## Infrastructure Components

### Database Layer

- **PostgreSQL with PostGIS**: Primary relational database with spatial extensions
- **Redis Stack**: Caching, session management, and RedisInsight UI

### Application Layer

- **Trenova API**: Go-based backend service with integrated workflow processing
- **Trenova Client**: React-based frontend application
- **Temporal**: Workflow orchestration engine for background jobs and long-running processes

### Event Streaming Layer

- **Kafka**: Distributed event streaming platform for CDC (Change Data Capture)
- **Zookeeper**: Coordination service for Kafka
- **Schema Registry**: Schema management for Kafka messages
- **Kafka Connect**: Data integration with Debezium PostgreSQL connector
- **Kafka UI**: Web-based management interface for Kafka

### Supporting Services

- **MinIO**: Object storage for file management
- **Caddy** (optional): Reverse proxy and web server for production deployments

## Minimum System Requirements

### Base Server Specifications

- **CPU**: 4 cores (Intel/AMD x86_64)
- **RAM**: 8GB
- **Storage**: 50GB available disk space
- **Network**: Stable internet connection (10+ Mbps)
- **Operating System**: Linux (Ubuntu 20.04+, CentOS 8+, or similar)

### Container Runtime

- **Docker**: Version 20.10+ with Docker Compose V2
- **Available Docker Memory**: 6GB allocated to Docker
- **Available Docker CPU**: 4 cores allocated to Docker

## Recommended Production Requirements

> **Note**: These requirements are based on actual performance testing achieving up to 130,000+ requests/second on optimized hardware.

### Small Deployment (up to 10,000 users / 50 concurrent)

- **CPU**: 8 cores
- **RAM**: 16GB
- **Storage**: 100GB SSD
- **Network**: 100 Mbps internet connection
- **Expected Performance**: 20,000+ requests/second
- **Average Response Time**: < 3ms

### Medium Deployment (10,000-50,000 users / 100-200 concurrent)

- **CPU**: 16 cores
- **RAM**: 32GB
- **Storage**: 200GB SSD (NVMe preferred)
- **Network**: 1 Gbps connection
- **Expected Performance**: 30,000-130,000 requests/second
- **Average Response Time**: < 10ms

### Large Deployment (50,000-100,000 users / 200-500 concurrent)

- **CPU**: 24+ cores
- **RAM**: 64GB+
- **Storage**: 500GB NVMe SSD
- **Network**: 1 Gbps dedicated connection
- **Expected Performance**: 25,000-130,000 requests/second
- **Average Response Time**: < 20ms

### Enterprise Deployment (100,000+ users / 500+ concurrent)

- **CPU**: 32+ cores
- **RAM**: 96GB+
- **Storage**: 1TB+ NVMe SSD with high IOPS
- **Network**: 10 Gbps connection
- **Additional**: Horizontal scaling with load balancer required
- **Expected Performance**: Variable based on cluster size
- **Average Response Time**: < 50ms

## Detailed Resource Allocation

### PostgreSQL Database with PostGIS (db)

- **Memory Limit**: 2GB (minimum for production)
- **Memory Reservation**: 512MB (development)
- **CPU Limit**: 2 cores
- **Storage**: Persistent volume for data
- **Extensions**: PostGIS, pg_stat_statements, pg_trgm, btree_gin, btree_gist
- **Critical Configuration Settings**:
  - `max_connections`: 300 (tested optimal)
  - `shared_buffers`: 512MB (25% of memory)
  - `effective_cache_size`: 1.5GB (75% of memory)
  - `work_mem`: 4MB
  - **Important**: Add composite index on frequently queried columns (e.g., `(organization_id, business_unit_id)`)

**Scaling Recommendations by User Load**:

- **Small (< 10K users)**: 2GB memory, 2 cores, 100 connections
- **Medium (10-50K users)**: 4GB memory, 4 cores, 200 connections
- **Large (50-100K users)**: 8GB memory, 8 cores, 300 connections
- **Enterprise (100K+ users)**: 16GB+ memory, 16+ cores, 500+ connections, consider read replicas

### Redis Stack (redis)

- **Memory Limit**: 2GB
- **Memory Reservation**: 1GB
- **CPU Limit**: 1 core
- **Max Memory Policy**: `allkeys-lru`
- **Configured Memory**: 1.5GB (75% of limit)
- **Additional Features**: RedisInsight UI on port 8001

**Scaling Recommendations**:

- For high-traffic applications: Increase to 4GB+ memory
- For session-heavy workloads: Consider Redis clustering

### Trenova API (tren-api)

- **Memory Limit**: 2GB (recommended for production)
- **CPU Limit**: 4 cores
- **Language**: Go (efficient memory usage)
- **Features**: Integrated workflow processing with Temporal
- **Critical Configuration**:
  - Database connection pool: 80 connections (optimal)
  - Redis connection pool: 200 connections
  - Server concurrency: 1048576
  - Read/Write buffer size: 8192 bytes

**Scaling Recommendations by User Load**:

- **Small (< 10K users)**: 1GB memory, 2 cores, single instance
- **Medium (10-50K users)**: 2GB memory, 4 cores, 2-3 instances with load balancer
- **Large (50-100K users)**: 4GB memory, 8 cores, 4-6 instances
- **Enterprise (100K+ users)**: 8GB+ memory, 16+ cores, horizontal scaling with multiple instances

**Performance Expectations**:

- Can handle 20,000-130,000 requests/second depending on concurrency
- Maintains < 10ms response time with proper configuration

### Trenova Client (tren-client)

- **Memory Limit**: 256MB
- **CPU Limit**: 0.5 cores
- **Technology**: React frontend served statically

**Scaling Recommendations**:

- Consider CDN for static asset delivery
- Multiple instances for high availability

### MinIO Object Storage (minio)

- **Memory Limit**: 512MB
- **Storage**: Persistent volume for object data
- **Ports**: 9000 (API), 9001 (Console)

**Scaling Recommendations**:

- Increase memory based on concurrent file operations
- Consider distributed MinIO setup for high availability
- Monitor storage usage and plan for growth

### Kafka Ecosystem

#### Apache Kafka (kafka)

- **Memory Limit**: 2GB
- **CPU Limit**: 2 cores
- **Storage**: Persistent volume for message data
- **Heap Size**: 1GB (Xms/Xmx)
- **Purpose**: Event streaming for Change Data Capture (CDC)
- **Retention**: 168 hours (7 days)

**Scaling Recommendations**:

- Increase to 4GB+ memory for high-throughput applications
- Consider Kafka clustering for high availability
- Monitor topic sizes and partition count

#### Zookeeper (zookeeper)

- **Memory Limit**: 512MB
- **Storage**: Persistent volumes for data and logs
- **Purpose**: Coordination service for Kafka cluster

#### Schema Registry (schema-registry)

- **Memory Limit**: 512MB
- **Purpose**: Schema management for Kafka messages
- **Port**: 8081

#### Kafka Connect (kafka-connect)

- **Memory Limit**: 1GB
- **Purpose**: Data integration with Debezium PostgreSQL connector
- **Port**: 8083
- **Features**: Change Data Capture (CDC) from PostgreSQL

#### Kafka UI (kafka-ui)

- **Memory Limit**: 256MB (recommended)
- **Purpose**: Web-based management and monitoring interface
- **Port**: 8090

### Temporal Workflow Engine

> **Recommended**: Use [Temporal Cloud](https://temporal.io/cloud) for production deployments to avoid operational overhead.

#### Temporal Server (if self-hosting)

- **Memory Recommendation**: 2GB-4GB
- **CPU Recommendation**: 2-4 cores
- **Storage**: Persistent database (PostgreSQL recommended)
- **Purpose**: Workflow orchestration for background jobs
- **Port**: 7233 (gRPC), 8233 (Web UI)

**Features**:

- **Workflow Processing**: Handles audit logging, email sending, notifications, shipment processing
- **Durable Execution**: Automatic retries and error handling
- **Monitoring**: Built-in Web UI for workflow monitoring
- **Scaling**: Horizontal scaling with multiple worker instances

**Scaling Recommendations**:

- Monitor workflow execution times and queue depths
- Increase worker concurrency for high-volume processing
- Scale worker instances horizontally for better throughput
- See [TEMPORAL.md](TEMPORAL.md) for detailed setup instructions

**Task Queues**:

- `shipment-queue`: Shipment-related workflows
- `notification-queue`: Notification processing
- `email-queue`: Email sending workflows
- `audit-queue`: Audit logging workflows
- `ailog-queue`: AI logging workflows
- `system-queue`: System maintenance tasks

## Storage Requirements

### Persistent Volumes

- **PostgreSQL Data**: 10GB+ (grows with data)
- **Redis Data**: 1GB+ (cache data)
- **MinIO Data**: 5GB+ (file storage, grows significantly)
- **Kafka Data**: 5GB+ (message persistence based on retention)
- **Zookeeper Data**: 1GB+ (coordination data)
- **Temporal Data** (if self-hosting): 2GB+ (workflow state)
- **Caddy Data** (if used): 100MB (certificates and config)

### Total Storage Needs

- **Base Installation**: 50GB
- **Growth Planning**:
  - Small organization (1-50 users): 100GB
  - Medium organization (50-200 users): 250GB
  - Large organization (200+ users): 500GB+

## Network Requirements

### Port Configuration

#### Main Application Ports

- **80/443**: HTTP/HTTPS (when using Caddy reverse proxy)
- **3001**: API server (internal or exposed)
- **5173**: Client application (development)
- **5432**: PostgreSQL (internal)
- **6379**: Redis (internal)
- **8001**: RedisInsight UI (optional, internal)
- **9000/9001**: MinIO API/Console (internal)

#### Kafka Ecosystem Ports

- **2181**: Zookeeper (internal)
- **9092**: Kafka external listener
- **29092**: Kafka internal listener
- **8081**: Schema Registry (internal)
- **8083**: Kafka Connect (internal)
- **8090**: Kafka UI (optional, internal)

#### Workflow Processing Ports

- **7233**: Temporal gRPC (internal or exposed for Cloud)
- **8233**: Temporal Web UI (optional, internal)

### Bandwidth Recommendations

- **Minimum**: 10 Mbps up/down
- **Recommended**: 100 Mbps up/down
- **High-traffic**: 1 Gbps up/down

## Scaling Considerations

### Horizontal Scaling Options

1. **Database**: PostgreSQL read replicas, connection pooling
2. **API**: Multiple API instances with load balancing (includes workflow workers)
3. **Cache**: Redis clustering or sharding
4. **Storage**: Distributed MinIO deployment
5. **Event Streaming**: Kafka clustering with multiple brokers
6. **Workflow Processing**: Scale Temporal workers horizontally across multiple instances

### Monitoring Recommendations

- **CPU Usage**: Monitor per-service CPU utilization
- **Memory Usage**: Track memory consumption and swap usage
- **Disk I/O**: Monitor database and storage performance
- **Network**: Track bandwidth utilization and latency
- **Application Metrics**: Monitor API response times and error rates
- **Kafka Metrics**: Monitor consumer lag, throughput, partition distribution
- **Temporal Metrics**: Monitor workflow execution times, task queue depths, worker health
- **CDC Performance**: Monitor Debezium connector lag and event processing times

### Performance Tuning

1. **Database Optimization**:
   - **Critical**: Add composite indexes on frequently queried columns (e.g., `CREATE INDEX idx_workers_org_bu ON workers(organization_id, business_unit_id)`)
   - Set `max_connections` to at least 300 for production
   - Configure connection pooling: 80 connections for API
   - Monitor slow query logs (queries > 100ms need optimization)
   - Adjust `shared_buffers` to 25% of allocated PostgreSQL memory
   - Set `effective_cache_size` to 75% of allocated memory

2. **Cache Optimization**:
   - Configure Redis pool size to 200 connections
   - Set `minIdleConns` to 100 for consistent performance
   - Monitor cache hit ratios (target > 80%)
   - Use `allkeys-lru` eviction policy

3. **Application Tuning**:
   - Set server `readBufferSize` and `writeBufferSize` to 8192
   - Configure server `concurrency` to 1048576
   - Set logging level to `error` in production (debug logging impacts performance)
   - Database connection pool: `maxConnections`: 80, `maxIdleConns`: 40
   - Connection lifetimes: `connMaxLifetime`: 300s, `connMaxIdleTime`: 60s

4. **Kafka Optimization**:
   - Tune `log.segment.bytes` and `log.retention.hours` based on throughput
   - Monitor consumer lag and adjust partition count if needed
   - Set appropriate replication factors for production (minimum 3)
   - Configure `compression.type` for better storage efficiency

5. **Performance Benchmarks** (tested on 16-core, 32GB RAM server):
   - 50 concurrent users: 20,708 req/s (2.3ms avg response)
   - 100 concurrent users: 37,908 req/s (2.5ms avg response)
   - 200 concurrent users: 131,921 req/s sustained (6ms avg response)
   - Breaking point: ~1500 concurrent connections

## Security Considerations

### Resource Security

- Ensure proper container isolation
- Implement resource limits to prevent DoS
- Monitor for unusual resource consumption patterns
- Regular security updates for base images

### Network Security

- Use internal Docker networks for service communication
- Expose only necessary ports to external networks
- Implement proper firewall rules
- Consider VPN access for management interfaces (Kafka UI, RedisInsight, Temporal UI, MinIO Console)

### SSO Configuration Security (Optional Feature)

**Note:** SSO (Single Sign-On) via OIDC is an **optional** feature. Only follow this section if you plan to enable SSO authentication.

#### Encryption Key Management

Trenova encrypts sensitive SSO data (OIDC client secrets) using AES-256-GCM encryption before storing in the database. Proper encryption key management is **critical**.

#### ⚠️ CRITICAL: Never Store Encryption Keys in Config Files

**BAD (Never do this):**

```yaml
# config.yaml - DO NOT STORE KEYS HERE!
security:
  encryption:
    key: "my-secret-encryption-key-12345"  # ❌ SECURITY RISK
```

**GOOD (Use environment variables or secrets manager):**

```yaml
# config.yaml
security:
  encryption:
    key: "${ENCRYPTION_KEY}"  # ✅ Reads from environment or secrets manager
```

#### Encryption Key Deployment Options

##### Option 1: Environment Variables (Development/Small Deployments)

**Generate secure key:**

```bash
# Generate a random 32-byte key
openssl rand -base64 32
```

**For Docker Compose:**

```bash
# .env file (add to .gitignore!)
ENCRYPTION_KEY=your-generated-32-byte-key-here

# Set proper permissions
chmod 600 .env
chown app-user:app-user .env
```

**For systemd services:**

```ini
# /etc/systemd/system/trenova-tms.service
[Service]
Environment="ENCRYPTION_KEY=your-32-byte-encryption-key-here"
# or use EnvironmentFile=/etc/trenova/.env.secret
```

##### Option 2: Secrets Manager (Production - Recommended)

Trenova includes built-in support for multiple secrets managers. Configure in `config.yaml`:

**AWS Secrets Manager:**

```yaml
secrets:
  provider: "aws"
  provider_config:
    region: "us-east-1"
    secret_name: "trenova/tms/encryption"

security:
  encryption:
    key: "${ENCRYPTION_KEY}"  # Auto-fetched from AWS
```

**HashiCorp Vault:**

```yaml
secrets:
  provider: "vault"
  provider_config:
    address: "https://vault.company.com"
    path: "secret/data/trenova/tms"

security:
  encryption:
    key: "${ENCRYPTION_KEY}"  # Auto-fetched from Vault
```

**Azure Key Vault:**

```yaml
secrets:
  provider: "azure"
  provider_config:
    vault_url: "https://trenova-vault.vault.azure.net"

security:
  encryption:
    key: "${ENCRYPTION_KEY}"  # Auto-fetched from Azure
```

**Google Cloud Secret Manager:**

```yaml
secrets:
  provider: "gcp"
  provider_config:
    project_id: "trenova-prod"

security:
  encryption:
    key: "${ENCRYPTION_KEY}"  # Auto-fetched from GCP
```

**Other supported providers:**

- `kubernetes`: Kubernetes Secrets
- `file`: File-based secrets (for development only)
- `environment`: Environment variables (default)

#### Security Best Practices for SSO

1. **Encryption Key Length**: Must be exactly 32 bytes for AES-256

   ```bash
   # Verify key length
   echo -n "your-key" | wc -c  # Should output 32
   ```

2. **Key Storage Priority** (in order of security):
   - ✅ **Best**: Cloud KMS (AWS/Azure/GCP Secrets Manager) or HashiCorp Vault
   - ✅ **Good**: Kubernetes Secrets with encryption at rest
   - ⚠️ **Acceptable**: Environment variables (development/testing only)
   - ❌ **Never**: Config files, code repositories, container images

3. **File Permissions** (if using file-based config):

   ```bash
   chmod 600 /opt/trenova/config.yaml
   chown trenova-user:trenova-user /opt/trenova/config.yaml
   
   # If using .env file
   chmod 600 /opt/trenova/.env
   chown trenova-user:trenova-user /opt/trenova/.env
   ```

4. **HTTPS Required**: SSO callbacks MUST use HTTPS in production

   ```yaml
   # SSO Config Example
   oidc_redirect_url: "https://app.trenova.com/auth/callback"  # ✅ HTTPS
   # NOT: http://app.trenova.com/auth/callback  # ❌ Insecure
   ```

5. **Key Rotation**: Rotate encryption keys every 90 days for compliance

#### Key Rotation Process

When rotating encryption keys:

```bash
# 1. Generate new key
NEW_KEY=$(openssl rand -base64 32)

# 2. Update in secrets manager
aws secretsmanager update-secret \
    --secret-id trenova/tms/encryption \
    --secret-string "{\"ENCRYPTION_KEY\":\"$NEW_KEY\"}"

# 3. Deploy app with new key (it will read automatically)
# 4. Re-encrypt existing SSO configs via API or migration script
```

#### Production Deployment Recommendations

| Deployment Type | Recommended Key Storage | Setup Complexity |
|-----------------|-------------------------|------------------|
| Development | Environment Variable | Low |
| Testing/Staging | Environment Variable or K8s Secrets | Low-Medium |
| Production (AWS) | AWS Secrets Manager | Medium |
| Production (Azure) | Azure Key Vault | Medium |
| Production (GCP) | Google Secret Manager | Medium |
| Production (On-Prem) | HashiCorp Vault | High |
| Production (Multi-Cloud) | HashiCorp Vault | High |

**For small deployments (<100 users):**

- Environment variables with proper permissions are acceptable
- Ensure backups include encryption keys (stored separately from database)

**For enterprise deployments (100+ users):**

- Use managed secrets service
- Implement 90-day key rotation schedule
- Set up audit alerts for key access

## Backup Requirements

### Data Backup Storage

- **Database Backups**: Plan for 3x database size
- **File Storage Backups**: Plan for 2x MinIO storage size
- **Kafka Backups**: Plan for topic data retention requirements
- **Temporal Backups** (if self-hosting): Include workflow state database
- **Configuration Backups**: Minimal space required

### Backup Infrastructure

- Separate storage for backups (external disk, cloud storage)
- Automated backup scheduling capabilities
- Network bandwidth for backup transfers

### Critical Backup Items

- PostgreSQL database (includes SSO configurations)
- MinIO object storage
- Configuration files (excluding secrets)
- Encryption keys (stored separately and securely)

## Environment Variables and Configuration

### Critical Environment Variables

```bash
# Database Configuration
TRENOVA_DB_HOST=db
TRENOVA_DB_PORT=5432
TRENOVA_DB_USER=postgres
TRENOVA_DB_PASSWORD=yourSecurePassword
TRENOVA_DB_NAME=trenova_go_db

# Redis Configuration
TRENOVA_REDIS_HOST=redis
TRENOVA_REDIS_PORT=6379
TRENOVA_REDIS_PASSWORD=yourStrongRedisPassword

# MinIO Configuration
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=secureMinioPassword

# Kafka Configuration
KAFKA_BOOTSTRAP_SERVERS=kafka:29092

# Temporal Configuration
TEMPORAL_HOST=localhost  # or temporal.tmprl.cloud for Temporal Cloud
TEMPORAL_PORT=7233

# Application Environment
TRENOVA_APP_ENVIRONMENT=production

# Security (Optional - only if using SSO)
ENCRYPTION_KEY=your-32-byte-encryption-key  # For SSO client secrets

# Email Configuration
SMTP_HOST=your-smtp-server
SMTP_PORT=587
SMTP_USER=your-smtp-user
SMTP_PASSWORD=your-smtp-password
```

## Deployment Checklist

### Pre-Deployment

- [ ] Verify minimum system requirements
- [ ] Install Docker and Docker Compose V2
- [ ] Configure environment variables for all services
- [ ] Set up persistent storage volumes
- [ ] Configure network access and port routing
- [ ] Set up email provider (SMTP/SendGrid)
- [ ] Configure Temporal (recommend Temporal Cloud for production)
- [ ] If using SSO: Configure encryption key management
- [ ] If using SSO: Choose secrets provider (AWS/Azure/GCP/Vault)
- [ ] Configure Kafka retention policies based on storage capacity
- [ ] Set up monitoring for all components

### Post-Deployment

- [ ] Verify all services are healthy (check `docker ps`)
- [ ] Test application functionality
- [ ] Test email sending functionality
- [ ] Verify Temporal workflows are running (check Web UI at port 8233)
- [ ] Verify Kafka consumers are processing messages (check Kafka UI at port 8090)
- [ ] If using SSO: Test SSO login flow
- [ ] If using SSO: Verify encryption is working (check database)
- [ ] Configure monitoring and alerts
- [ ] Set up backup procedures for PostgreSQL, MinIO, and Kafka
- [ ] Document configuration changes
- [ ] Load test critical workflows

## Troubleshooting Common Issues

### Resource-Related Issues

1. **Out of Memory**: Increase container memory limits or server RAM
2. **High CPU Usage**: Scale horizontally or increase CPU allocation
3. **Slow Database**: Tune PostgreSQL configuration or add more memory
4. **Slow Response Times**: Check network latency and resource utilization
5. **Kafka Consumer Lag**: Increase partition count or add more consumer instances
6. **Temporal Workflow Delays**: Scale worker instances or increase concurrency

### Performance Optimization

1. **Database Performance**: Monitor query performance and optimize with indexes
2. **Cache Performance**: Tune Redis configuration and monitor hit rates
3. **Application Performance**: Profile Go application and optimize bottlenecks
4. **Network Performance**: Optimize network configuration and bandwidth
5. **Event Streaming**: Monitor Kafka consumer lag and optimize partition distribution
6. **Workflow Performance**: Monitor Temporal task queue depths and worker utilization

### SSO-Related Issues (if enabled)

**Issue: "Encryption key is required" error**

```bash
# Verify ENCRYPTION_KEY is set
echo $ENCRYPTION_KEY

# Check secrets manager configuration
cat config.yaml | grep -A 5 "secrets:"
```

**Issue: "Failed to decrypt" error**

```bash
# This occurs when encryption key changed without re-encrypting data
# Solution: Restore correct encryption key or re-encrypt SSO configs
```

**Issue: SSO login fails**

```bash
# Check OIDC configuration
# Verify redirect URL matches exactly in IdP configuration
# Check logs for detailed error messages
docker logs trenova-api | grep -i "sso\|oidc"
```

## Alternative: Managed Hosting Services

If managing these infrastructure requirements seems overwhelming, consider our **Managed Hosting Services** where we handle all of these requirements for you:

- ✅ **No Infrastructure Management** - We provision and manage all servers, databases, and services
- ✅ **Automatic Scaling** - We handle scaling based on your usage patterns
- ✅ **Zero-Downtime Updates** - We manage all application updates and maintenance
- ✅ **24/7 Monitoring** - Professional monitoring and support included
- ✅ **Enterprise Security** - Advanced security measures and compliance certifications
- ✅ **Guaranteed SLAs** - Contractual uptime and performance guarantees
- ✅ **Managed Temporal Cloud** - Enterprise-grade workflow orchestration
- ✅ **Managed Kafka** - Fully managed event streaming

Contact us at <sales@trenova.app> to learn more about our managed hosting options.

---

**For Self-Hosting**: This resource guide should be reviewed and updated regularly as your usage patterns change and the application evolves. Monitor your actual resource usage and adjust allocations accordingly for optimal performance.

**Last Updated**: 2025-10-22
