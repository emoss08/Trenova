# Trenova Self-Hosting System Requirements

> **Important**: This document is specifically for organizations planning to **self-host** Trenova on their own infrastructure. If you're interested in our **Managed Hosting Services**, these requirements don't apply to you - we handle all infrastructure, scaling, and maintenance.

This document outlines the system resource requirements needed to self-host the Trenova application in a production environment. These requirements are based on the production Docker Compose configuration and can be scaled according to your user base and usage patterns.

## Overview

Trenova is a containerized application with integrated background job processing that provides a comprehensive Transportation Management System (TMS). When self-hosting, you'll be responsible for managing all these components on your own infrastructure. The application is designed to be scalable and can be adjusted based on your organization's needs and available resources.

## Infrastructure Components

### Database Layer

- **PostgreSQL**: Primary relational database
- **PGBouncer**: Connection pooling for PostgreSQL
- **Redis**: Caching and session management

### Application Layer

- **Trenova API**: Go-based backend service with integrated background job processing
- **Trenova Client**: React-based frontend application
- **Asynq**: Background job queue and task processing system

### Supporting Services

- **MinIO**: Object storage for file management
- **RabbitMQ**: Message queue for job distribution
- **Caddy**: Reverse proxy and web server
- **Asynq Dashboard**: Web UI for monitoring background jobs

## Minimum System Requirements

### Base Server Specifications

- **CPU**: 6 cores (Intel/AMD x86_64)
- **RAM**: 12GB
- **Storage**: 30GB available disk space
- **Network**: Stable internet connection
- **Operating System**: Linux (Ubuntu 20.04+, CentOS 8+, or similar)

### Container Runtime

- **Docker**: Version 20.10+ with Docker Compose V2
- **Available Docker Memory**: 10GB allocated to Docker
- **Available Docker CPU**: 6 cores allocated to Docker

## Recommended Production Requirements

### Server Specifications (1-50 concurrent users)

- **CPU**: 12+ cores
- **RAM**: 24GB+
- **Storage**: 100GB+ SSD
- **Network**: High-speed internet with low latency

### Server Specifications (50-200 concurrent users)

- **CPU**: 20+ cores
- **RAM**: 48GB+
- **Storage**: 200GB+ SSD
- **Network**: Dedicated bandwidth with redundancy

### Server Specifications (200+ concurrent users)

- **CPU**: 32+ cores
- **RAM**: 96GB+
- **Storage**: 500GB+ SSD with high IOPS
- **Network**: Enterprise-grade connectivity
- **Additional**: Consider load balancing and horizontal scaling

## Detailed Resource Allocation

### PostgreSQL Database (tren-db)

- **Memory Limit**: 512MB (configurable)
- **Memory Reservation**: 256MB
- **CPU Limit**: 1 core
- **Storage**: Persistent volume for data
- **Configuration**: Optimized for 512MB memory allocation
  - `shared_buffers`: 128MB (25% of memory)
  - `effective_cache_size`: 384MB (75% of memory)
  - `max_connections`: 100 (adjustable based on load)

**Scaling Recommendations**:

- For 50+ users: Increase memory to 2GB, CPU to 2 cores
- For 200+ users: Increase memory to 4GB+, CPU to 4+ cores
- Consider increasing `max_connections` to 200-500 for high concurrency

### Redis Cache (tren-redis)

- **Memory Limit**: 2GB
- **Memory Reservation**: 1GB
- **CPU Limit**: 1 core
- **Max Memory Policy**: `allkeys-lru`
- **Configured Memory**: 1.5GB (75% of limit)

**Scaling Recommendations**:

- For high-traffic applications: Increase to 4GB+ memory
- For session-heavy workloads: Consider Redis clustering

### PGBouncer (tren-pgbouncer)

- **Memory Limit**: 128MB
- **Memory Reservation**: 64MB
- **Purpose**: Connection pooling and database load management

**Scaling Recommendations**:

- Adjust pool sizes in configuration based on concurrent users
- Monitor connection utilization and adjust accordingly

### Trenova API (tren-api)

- **Memory Limit**: 512MB
- **CPU Limit**: 1 core
- **Language**: Go (efficient memory usage)
- **Features**: Integrated background job processing with Asynq

**Scaling Recommendations**:

- For 50+ users: Increase memory to 1GB, consider multiple instances
- For 200+ users: Deploy multiple API instances behind load balancer
- Monitor memory usage and adjust based on application behavior
- Scale background job workers based on queue depth

### Trenova Client (tren-client)

- **Memory Limit**: 256MB
- **CPU Limit**: 0.5 cores
- **Technology**: React frontend served statically

**Scaling Recommendations**:

- Consider CDN for static asset delivery
- Multiple instances for high availability

### MinIO Object Storage (tren-minio)

- **Memory Limit**: 512MB
- **Storage**: Persistent volume for object data
- **Ports**: 9000 (API), 9001 (Console)

**Scaling Recommendations**:

- Increase memory based on concurrent file operations
- Consider distributed MinIO setup for high availability
- Monitor storage usage and plan for growth

### RabbitMQ (tren-rabbitmq)

- **Memory**: No explicit limit (monitor usage)
- **Storage**: Persistent volume for message durability
- **Ports**: 5674 (AMQP), 15674 (Management)

**Scaling Recommendations**:

- Set memory limits based on message volume
- Consider RabbitMQ clustering for high availability

### Caddy Reverse Proxy (tren-caddy)

- **Memory Limit**: 128MB
- **CPU Limit**: 0.5 cores
- **Features**: Automatic HTTPS, reverse proxy

## Background Job Processing

### Asynq Job Queue System

- **Memory Recommendation**: 256MB-512MB (integrated within API)
- **CPU Recommendation**: Shared with API (additional workers scale automatically)
- **Technology**: Asynq with Redis backend
- **Purpose**: Background job processing, email sending, workflow automation
- **Dependencies**: Redis, RabbitMQ

**Features**:

- **Job Processing**: Handles email sending, document processing, and automation tasks
- **Queue Management**: Multiple priority queues for different job types
- **Retry Logic**: Automatic retry with exponential backoff for failed jobs
- **Monitoring**: Built-in dashboard for job monitoring and management

**Scaling Recommendations**:

- Monitor job queue depth and processing times
- Increase worker concurrency for high-volume processing
- Scale Redis memory based on job queue size
- Deploy multiple API instances for horizontal job processing

### Asynq Dashboard

- **Memory Recommendation**: 128MB
- **CPU Recommendation**: 0.5 cores
- **Technology**: Web-based monitoring interface
- **Purpose**: Monitor job queues, retry failed jobs, view job statistics
- **Port**: 8080 (job monitoring UI)

**Features**:

- Real-time job queue monitoring
- Job retry and deletion capabilities
- Performance metrics and statistics
- Queue management tools

## Storage Requirements

### Persistent Volumes

- **PostgreSQL Data**: 10GB+ (grows with data)
- **Redis Data**: 1GB+ (cache data)
- **MinIO Data**: 5GB+ (file storage, grows significantly)
- **RabbitMQ Data**: 1GB+ (message persistence and job queues)
- **Caddy Data**: 100MB (certificates and config)

### Total Storage Needs

- **Base Installation**: 25GB
- **Growth Planning**:
  - Small organization (1-50 users): 75GB
  - Medium organization (50-200 users): 150GB
  - Large organization (200+ users): 400GB+

## Network Requirements

### Port Configuration

#### Main Application Ports

- **80/443**: HTTP/HTTPS (Caddy proxy)
- **3001**: API server (internal)
- **5173**: Client application (internal)
- **5432**: PostgreSQL (internal)
- **6432**: PGBouncer (internal)
- **6379**: Redis (internal)
- **9000/9001**: MinIO API/Console (internal)
- **5674/15674**: RabbitMQ (internal)

#### Background Job Processing Ports

- **8080**: Asynq Dashboard (job monitoring UI)

### Bandwidth Recommendations

- **Minimum**: 10 Mbps up/down
- **Recommended**: 100 Mbps up/down
- **High-traffic**: 1 Gbps up/down

## Scaling Considerations

### Horizontal Scaling Options

1. **Database**: PostgreSQL read replicas, connection pooling
2. **API**: Multiple API instances with load balancing (includes job workers)
3. **Cache**: Redis clustering or sharding
4. **Storage**: Distributed MinIO deployment
5. **Message Queue**: RabbitMQ clustering
6. **Background Jobs**: Scale job workers within API instances or deploy dedicated worker instances

### Monitoring Recommendations

- **CPU Usage**: Monitor per-service CPU utilization
- **Memory Usage**: Track memory consumption and swap usage
- **Disk I/O**: Monitor database and storage performance
- **Network**: Track bandwidth utilization and latency
- **Application Metrics**: Monitor API response times and error rates
- **Background Jobs**: Monitor job queue depth, processing rates, and failure rates
- **Asynq Dashboard**: Monitor job statistics, retry rates, and queue health

### Performance Tuning

1. **Database Optimization**:
   - Adjust PostgreSQL configuration based on available memory
   - Optimize queries and add indexes as needed
   - Monitor slow query logs

2. **Cache Optimization**:
   - Tune Redis memory policies
   - Monitor cache hit ratios
   - Adjust cache expiration policies

3. **Application Tuning**:
   - Monitor Go application garbage collection
   - Optimize API endpoint performance
   - Implement proper error handling and retry logic

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
- Consider VPN access for management interfaces

## Backup Requirements

### Data Backup Storage

- **Database Backups**: Plan for 3x database size
- **File Storage Backups**: Plan for 2x MinIO storage size
- **Configuration Backups**: Minimal space required
- **Job Queue State**: Included in Redis backup requirements

### Backup Infrastructure

- Separate storage for backups (external disk, cloud storage)
- Automated backup scheduling capabilities
- Network bandwidth for backup transfers

## Environment Variables and Configuration

### Critical Environment Variables

```bash
# Database Configuration
TRENOVA_DB_HOST=tren-pgbouncer
TRENOVA_DB_PORT=6432
TRENOVA_DB_USER=postgres
TRENOVA_DB_PASSWORD=yourSecurePassword
TRENOVA_DB_NAME=trenova_go_db

# Security
TRENOVA_SERVER_SECRET_KEY=yourVerySecureSecretKey
TRENOVA_REDIS_PASSWORD=yourStrongRedisPassword

# MinIO Configuration
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=secureMinioPassword

# Application Environment
TRENOVA_APP_ENVIRONMENT=production

# Background Job Configuration (Asynq)
ASYNQ_REDIS_HOST=tren-redis
ASYNQ_REDIS_PORT=6379
ASYNQ_REDIS_PASSWORD=yourStrongRedisPassword
ASYNQ_DASHBOARD_PORT=8080

# Email Configuration (integrated)
SMTP_HOST=your-smtp-server
SMTP_PORT=587
SMTP_USER=your-smtp-user
SMTP_PASSWORD=your-smtp-password
```

## Deployment Checklist

### Pre-Deployment

- [ ] Verify minimum system requirements
- [ ] Install Docker and Docker Compose
- [ ] Configure environment variables for all services
- [ ] Set up persistent storage for application
- [ ] Configure network access and port routing
- [ ] Set up email provider (SMTP/SendGrid)
- [ ] Configure Asynq job processing settings

### Post-Deployment

- [ ] Verify all services are healthy
- [ ] Test application functionality
- [ ] Test email sending functionality
- [ ] Verify background job processing (check Asynq dashboard)
- [ ] Configure monitoring for all components
- [ ] Set up backup procedures for database and Redis
- [ ] Document configuration changes
- [ ] Test job processing scaling if needed

## Troubleshooting Common Issues

### Resource-Related Issues

1. **Out of Memory**: Increase container memory limits or server RAM
2. **High CPU Usage**: Scale horizontally or increase CPU allocation
3. **Slow Database**: Tune PostgreSQL configuration or add more memory
4. **Slow Response Times**: Check network latency and resource utilization

### Performance Optimization

1. **Database Performance**: Monitor query performance and optimize
2. **Cache Performance**: Tune Redis configuration and monitor hit rates
3. **Application Performance**: Profile Go application and optimize bottlenecks
4. **Network Performance**: Optimize network configuration and bandwidth

## Alternative: Managed Hosting Services

If managing these infrastructure requirements seems overwhelming, consider our **Managed Hosting Services** where we handle all of these requirements for you:

- ✅ **No Infrastructure Management** - We provision and manage all servers, databases, and services
- ✅ **Automatic Scaling** - We handle scaling based on your usage patterns  
- ✅ **Zero-Downtime Updates** - We manage all application updates and maintenance
- ✅ **24/7 Monitoring** - Professional monitoring and support included
- ✅ **Enterprise Security** - Advanced security measures and compliance certifications
- ✅ **Guaranteed SLAs** - Contractual uptime and performance guarantees

Contact us at <sales@trenova.app> to learn more about our managed hosting options.

---

**For Self-Hosting**: This resource guide should be reviewed and updated regularly as your usage patterns change and the application evolves. Monitor your actual resource usage and adjust allocations accordingly for optimal performance.
