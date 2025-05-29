# Trenova System Requirements

This document outlines the system resource requirements needed to run the Trenova application in a production environment. These requirements are based on the production Docker Compose configuration and can be scaled according to your user base and usage patterns.

## Overview

Trenova is a containerized application consisting of multiple services that work together to provide a comprehensive Transportation Management System (TMS). The application is designed to be scalable and can be adjusted based on your organization's needs.

## Infrastructure Components

### Database Layer

- **PostgreSQL**: Primary relational database
- **PGBouncer**: Connection pooling for PostgreSQL
- **Redis**: Caching and session management

### Application Layer

- **Trenova API**: Go-based backend service
- **Trenova Client**: React-based frontend application

### Supporting Services

- **MinIO**: Object storage for file management
- **RabbitMQ**: Message queue for async processing
- **Caddy**: Reverse proxy and web server

### Microservices Layer

- **Email Service**: Go-based microservice for handling email operations
- **Workflow Service**: Go-based microservice using Hatchet for workflow orchestration
- **Hatchet Engine**: Workflow orchestration engine with PostgreSQL and RabbitMQ

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

**Scaling Recommendations**:

- For 50+ users: Increase memory to 1GB, consider multiple instances
- For 200+ users: Deploy multiple API instances behind load balancer
- Monitor memory usage and adjust based on application behavior

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

## Microservices Resource Allocation

### Email Service (trenova-email)

- **Memory Recommendation**: 256MB-512MB
- **CPU Recommendation**: 0.5-1 cores
- **Language**: Go (efficient memory usage)
- **Purpose**: Handles email sending operations via RabbitMQ
- **Dependencies**: RabbitMQ, SMTP/SendGrid providers

**Scaling Recommendations**:

- For high email volume: Increase memory to 1GB, deploy multiple instances
- Monitor email queue depth and processing times
- Consider rate limiting to avoid provider throttling

### Workflow Service (trenova-workflow)

- **Memory Recommendation**: 512MB-1GB
- **CPU Recommendation**: 1-2 cores
- **Language**: Go with Hatchet framework
- **Purpose**: Workflow orchestration and automation
- **Dependencies**: Hatchet Engine, PostgreSQL, RabbitMQ

**Scaling Recommendations**:

- For complex workflows: Increase memory to 2GB+
- Deploy multiple worker instances for high throughput
- Monitor workflow execution times and queue depths

### Hatchet Engine (workflow orchestration)

- **Memory Recommendation**: 1GB-2GB
- **CPU Recommendation**: 1-2 cores
- **Technology**: Hatchet workflow engine
- **Purpose**: Workflow execution and state management
- **Dependencies**: Dedicated PostgreSQL, RabbitMQ

**Components**:

- **Hatchet PostgreSQL**: Dedicated database for workflow state
  - Memory: 1GB+ (separate from main application database)
  - Storage: 10GB+ for workflow history and state
- **Hatchet Dashboard**: Web interface for workflow monitoring
  - Memory: 256MB
  - Port: 8080 (workflow management UI)
- **Hatchet RabbitMQ**: Message queue for workflow tasks
  - Memory: 512MB+ (separate from main application queue)
  - Ports: 5673 (AMQP), 15673 (Management UI)

## Storage Requirements

### Persistent Volumes

- **PostgreSQL Data**: 10GB+ (grows with data)
- **Redis Data**: 1GB+ (cache data)
- **MinIO Data**: 5GB+ (file storage, grows significantly)
- **RabbitMQ Data**: 1GB+ (message persistence)
- **Caddy Data**: 100MB (certificates and config)
- **Hatchet PostgreSQL Data**: 5GB+ (workflow state and history)
- **Hatchet RabbitMQ Data**: 500MB+ (workflow message persistence)
- **Hatchet Config**: 100MB (workflow configuration and certificates)

### Total Storage Needs

- **Base Installation**: 30GB
- **Growth Planning**:
  - Small organization (1-50 users): 100GB
  - Medium organization (50-200 users): 200GB
  - Large organization (200+ users): 500GB+

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

#### Microservices Ports

- **8082**: Email Service API (internal)
- **8083**: Email Service Template Management UI (development only)
- **5435**: Hatchet PostgreSQL (internal)
- **5673/15673**: Hatchet RabbitMQ (internal)
- **7077**: Hatchet Engine gRPC (internal)
- **8080**: Hatchet Dashboard (workflow management UI)

### Bandwidth Recommendations

- **Minimum**: 10 Mbps up/down
- **Recommended**: 100 Mbps up/down
- **High-traffic**: 1 Gbps up/down

## Scaling Considerations

### Horizontal Scaling Options

1. **Database**: PostgreSQL read replicas, connection pooling
2. **API**: Multiple API instances with load balancing
3. **Cache**: Redis clustering or sharding
4. **Storage**: Distributed MinIO deployment
5. **Message Queue**: RabbitMQ clustering
6. **Email Service**: Multiple email service instances for high volume
7. **Workflow Service**: Multiple workflow worker instances
8. **Hatchet Engine**: Horizontal scaling via additional worker nodes

### Monitoring Recommendations

- **CPU Usage**: Monitor per-service CPU utilization
- **Memory Usage**: Track memory consumption and swap usage
- **Disk I/O**: Monitor database and storage performance
- **Network**: Track bandwidth utilization and latency
- **Application Metrics**: Monitor API response times and error rates
- **Email Service**: Monitor email queue depth, send rates, and delivery status
- **Workflow Service**: Monitor workflow execution times, queue depths, and failure rates
- **Hatchet Engine**: Monitor workflow state, active jobs, and resource utilization

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

# Email Service Configuration
EMAIL_ENV=production
EMAIL_PORT=8082
EMAIL_RABBITMQ_HOST=tren-rabbitmq
EMAIL_RABBITMQ_PORT=5674
EMAIL_RABBITMQ_USER=user
EMAIL_RABBITMQ_PASSWORD=password
EMAIL_SMTP_HOST=your-smtp-server
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USER=your-smtp-user
EMAIL_SMTP_PASSWORD=your-smtp-password

# Workflow Service Configuration
HATCHET_DATABASE_URL=postgres://hatchet:hatchet@hatchet-postgres:5432/hatchet
HATCHET_RABBITMQ_URL=amqp://user:password@hatchet-rabbitmq:5672/
SERVER_GRPC_BIND_ADDRESS=0.0.0.0
SERVER_GRPC_INSECURE=t
```

## Deployment Checklist

### Pre-Deployment

- [ ] Verify minimum system requirements (including microservices)
- [ ] Install Docker and Docker Compose
- [ ] Configure environment variables for all services
- [ ] Set up persistent storage for main app and microservices
- [ ] Configure network access and port routing
- [ ] Set up email provider (SMTP/SendGrid) for email service
- [ ] Configure workflow engine prerequisites

### Post-Deployment

- [ ] Verify all services are healthy (main app + microservices)
- [ ] Test application functionality
- [ ] Test email sending functionality
- [ ] Verify workflow engine operation
- [ ] Configure monitoring for all components
- [ ] Set up backup procedures for all databases
- [ ] Document configuration changes
- [ ] Test microservices scaling if needed

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

This resource guide should be reviewed and updated regularly as your usage patterns change and the application evolves. Monitor your actual resource usage and adjust allocations accordingly for optimal performance.
