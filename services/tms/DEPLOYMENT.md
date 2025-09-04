# Trenova TMS - Production Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying Trenova TMS in a production environment. The deployment includes a full microservices stack with PostgreSQL (with replicas), Redis, Kafka, MinIO, and your choice of reverse proxy (Nginx or Traefik).

## Table of Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Detailed Setup](#detailed-setup)
- [Configuration](#configuration)
- [SSL/TLS Setup](#ssltls-setup)
- [Monitoring & Maintenance](#monitoring--maintenance)
- [Troubleshooting](#troubleshooting)
- [Security Considerations](#security-considerations)
- [Scaling](#scaling)
- [Backup & Recovery](#backup--recovery)

## Architecture

The production deployment consists of:

### Core Services

- **API Server** (Go/Fiber) - Main application backend
- **UI Client** (React/Vite) - Frontend application
- **PostgreSQL** - Primary database with read replicas
- **PgBouncer** - Connection pooling for PostgreSQL
- **Redis** - Caching and session storage
- **MinIO** - S3-compatible object storage

### Message Queue & Streaming

- **Apache Kafka** - Event streaming and CDC
- **Zookeeper** - Kafka coordination
- **Schema Registry** - Avro schema management
- **Kafka Connect** - Database change data capture

### Reverse Proxy Options

- **Nginx** - High-performance web server and reverse proxy
- **Traefik** - Cloud-native proxy with automatic SSL

## Prerequisites

### System Requirements

- **OS**: Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- **RAM**: Minimum 8GB, Recommended 16GB+
- **CPU**: Minimum 4 cores, Recommended 8+ cores
- **Storage**: Minimum 50GB SSD, Recommended 200GB+ NVMe
- **Network**: Static IP address and domain name (for production SSL)

### Software Requirements

- Docker 24.0+
- Docker Compose 2.20+
- Git
- openssl
- curl/wget

### Installation

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose (if not included)
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Logout and login to apply group changes
```

## Quick Start

For a rapid production deployment:

```bash
# Clone the repository
git clone https://github.com/emoss08/Trenova.git
cd Trenova/services/tms

# Configure environment
cp .env.production.example .env.production
nano .env.production  # Edit with your settings

# Deploy with Nginx (recommended for most users)
./scripts/deploy-prod.sh --nginx

# OR deploy with Traefik (automatic SSL)
./scripts/deploy-prod.sh --traefik
```

## Detailed Setup

### 1. Environment Configuration

Copy and configure the environment file:

```bash
cp .env.production.example .env.production
```

**Critical settings to configure:**

```bash
# Domain and SSL
DOMAIN=yourdomain.com
ACME_EMAIL=admin@yourdomain.com

# Database security
DB_PASSWORD=your_strong_database_password
REPLICATION_PASSWORD=your_replication_password

# Redis security
REDIS_PASSWORD=your_strong_redis_password

# MinIO credentials
MINIO_ROOT_USER=your_minio_admin
MINIO_ROOT_PASSWORD=your_strong_minio_password

# Application secrets
SERVER_SECRET_KEY=$(openssl rand -base64 32)

# AI Services (Optional)
CLAUDE_API_KEY=sk-ant-api03-your-claude-key
OPENAI_API_KEY=sk-proj-your-openai-key
```

### 2. SSL Certificate Setup

#### Option A: Self-Signed Certificates (Development/Testing)

```bash
./scripts/generate-certs.sh
```

#### Option B: Let's Encrypt (Production)

```bash
DOMAIN=yourdomain.com ./scripts/generate-certs.sh --letsencrypt
```

### 3. Manual Deployment Steps

If you prefer manual control:

```bash
# Build images
docker build -t trenova-api:latest -f Dockerfile .
docker build -t trenova-ui:latest -f ../ui/Dockerfile ../ui/

# For Nginx
docker build -t trenova-nginx:latest -f nginx/Dockerfile .
docker-compose -f docker-compose-prod.yml -f docker-compose-prod.nginx.yml up -d

# For Traefik
docker build -t trenova-traefik:latest -f traefik/Dockerfile .
docker-compose -f docker-compose-prod.yml -f docker-compose-prod.traefik.yml up -d
```

## Configuration

### Port Configuration

Production uses different ports to avoid conflicts with development:

| Service | Development | Production |
|---------|-------------|-----------|
| PostgreSQL | 5432 | 15432 |
| PgBouncer | - | 16432 |
| Redis | 6379 | 16379 |
| API | 3001 | 13001 |
| UI | 5173 | 15173 |
| MinIO API | 9000 | 19000 |
| MinIO Console | 9001 | 19001 |
| Kafka | 9092 | 19092 |

### Database Configuration

The production setup includes:

- Primary PostgreSQL instance
- Two read replicas for load distribution
- PgBouncer connection pooling
- Automated replication setup

### Redis Configuration

Redis is configured with:

- Password authentication
- Memory optimization (2GB limit)
- Persistence enabled
- Connection pooling

## SSL/TLS Setup

### Nginx SSL Configuration

Nginx is configured with:

- Modern TLS 1.2/1.3 support
- Strong cipher suites
- HSTS headers
- Certificate auto-renewal support

### Traefik SSL Configuration

Traefik provides:

- Automatic Let's Encrypt certificates
- Certificate renewal
- Advanced routing rules
- Built-in monitoring dashboard

## Monitoring & Maintenance

### Health Checks

All services include comprehensive health checks:

```bash
# Check all service status
docker-compose ps

# View service logs
docker-compose logs -f [service_name]

# Check specific service health
docker-compose exec tren-api wget -q --spider http://localhost:3001/health
```

### Log Management

Logs are organized by service:

- `logs/api/` - Application logs
- `logs/nginx/` - Nginx access/error logs  
- `logs/traefik/` - Traefik logs

### Certificate Renewal

For Let's Encrypt certificates:

```bash
# Manual renewal
./scripts/renew-certs.sh nginx  # or traefik

# Add to crontab for automatic renewal
crontab -e
# Add: 0 2 * * * /path/to/Trenova/services/tms/scripts/renew-certs.sh nginx
```

## Troubleshooting

### Common Issues

#### 1. Services Not Starting

```bash
# Check logs
docker-compose logs [service_name]

# Check system resources
docker stats

# Restart specific service
docker-compose restart [service_name]
```

#### 2. Database Connection Issues

```bash
# Check PgBouncer status
docker-compose exec tren-pgbouncer psql -h localhost -U postgres -d trenova_go_db -c "SHOW POOLS;"

# Test direct database connection
docker-compose exec tren-db psql -U postgres -d trenova_go_db -c "SELECT 1;"
```

#### 3. SSL Certificate Issues

```bash
# Verify certificates
./scripts/generate-certs.sh --verify

# Check certificate expiration
openssl x509 -in certs/cert.pem -noout -dates
```

#### 4. Memory Issues

```bash
# Check container memory usage
docker stats --no-stream

# Increase swap if needed (Ubuntu/Debian)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### Performance Tuning

#### PostgreSQL Optimization

```bash
# Edit config/postgresql.conf
shared_buffers = 256MB              # 25% of RAM
effective_cache_size = 1GB          # 75% of RAM
work_mem = 4MB
maintenance_work_mem = 64MB
```

#### Redis Optimization

```bash
# Adjust Redis memory in .env.production
REDIS_MAX_MEMORY=2gb
```

## Security Considerations

### Network Security

- Use a firewall to restrict access to service ports
- Only expose ports 80 and 443 to the internet
- Consider VPN access for administration

### Container Security

- Run containers as non-root users where possible
- Use Docker secrets for sensitive data
- Regularly update base images

### Database Security

- Use strong passwords for all database users
- Enable SSL for database connections in production
- Regularly backup and test restoration procedures

### Application Security

- Change all default passwords
- Use environment variables for all secrets
- Enable rate limiting in reverse proxy
- Monitor access logs for suspicious activity

## Scaling

### Horizontal Scaling

Scale specific services based on load:

```bash
# Scale API servers
docker-compose up -d --scale tren-api=3

# Scale UI servers (behind load balancer)
docker-compose up -d --scale tren-client=2
```

### Database Scaling

The setup includes read replicas:

- Primary: Write operations
- Replica 1: Read operations
- Replica 2: Analytics/reporting

### Load Balancing

Both Nginx and Traefik support load balancing:

- Round-robin distribution
- Health check integration
- Session affinity (sticky sessions)

## Backup & Recovery

### Database Backups

Automated backups are configured in the application:

- Daily full backups
- 30-day retention
- Compressed storage

Manual backup:

```bash
# Create backup
docker-compose exec tren-db pg_dump -U postgres trenova_go_db > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore from backup
docker-compose exec -T tren-db psql -U postgres trenova_go_db < backup_file.sql
```

### File Storage Backups

MinIO data should be backed up regularly:

```bash
# Sync MinIO data
docker-compose exec tren-minio mc mirror /data /backup/minio/
```

### Configuration Backups

Backup configuration files:

```bash
# Backup configurations
tar -czf trenova_config_$(date +%Y%m%d).tar.gz \
  .env.production \
  config/ \
  certs/ \
  docker-compose*.yml
```

## Support

For additional support:

- Check the [GitHub Issues](https://github.com/emoss08/Trenova/issues)
- Join the [Discord Community](https://discord.gg/XDBqyvrryq)
- Review the [Documentation](https://docs.trenova.com)

## License

Trenova is licensed under the Functional Source License (FSL-1.1). See [LICENSE.md](https://github.com/emoss08/Trenova/blob/master/LICENSE.md) for details.
