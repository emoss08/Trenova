# Database Read Replicas - Quick Start Guide

## Overview

Trenova now supports read/write database separation to improve performance and scalability. This guide will help you set up and test read replicas locally.

## Quick Start

### Option 1: Using Bitnami PostgreSQL (Recommended for Testing)

The easiest way to test read replicas locally:

```bash
# Create the network if it doesn't exist
docker network create app-network

# Start PostgreSQL with replicas using Bitnami images
docker-compose -f docker-compose-bitnami-replicas.yml up -d

# Wait for replicas to sync (about 30 seconds)
sleep 30

# Verify all databases are running
docker-compose -f docker-compose-bitnami-replicas.yml ps

# Check replication status
docker exec -it trenova-db-1 psql -U postgres -d trenova_go_db -c "SELECT * FROM pg_stat_replication;"
```

### Option 2: Using Standard PostgreSQL

For production-like setup with standard PostgreSQL:

```bash
# Start the standard services
docker-compose -f docker-compose-local.yml up -d

# The read replicas will be available at:
# - Primary: localhost:5432
# - Replica 1: localhost:5433  
# - Replica 2: localhost:5434
```

## Configuration

### 1. Update your config file

Add read replica configuration to your `config/local/config.local.yaml`:

```yaml
db:
  # Primary database
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "postgres"
  database: "trenova_go_db"
  
  # Enable read/write separation
  enableReadWriteSeparation: true
  replicaLagThreshold: 5  # seconds
  
  # Read replicas
  readReplicas:
    - name: "replica1"
      host: "localhost"
      port: 5433
      weight: 1
    
    - name: "replica2"
      host: "localhost"
      port: 5434
      weight: 1
```

### 2. Start the application

```bash
# Run the API with read replica support
task run
```

## Testing Read/Write Separation

### Monitor Query Distribution

Watch the logs to see queries being distributed:

```bash
# Terminal 1 - Watch primary database logs
docker logs -f trenova-db-1 2>&1 | grep -E "statement:|LOG:"

# Terminal 2 - Watch replica 1 logs  
docker logs -f trenova-db-replica1-1 2>&1 | grep -E "statement:|LOG:"

# Terminal 3 - Watch replica 2 logs
docker logs -f trenova-db-replica2-1 2>&1 | grep -E "statement:|LOG:"
```

### Test with API Calls

```bash
# Read operations will use replicas
curl http://localhost:3001/api/v1/equipment-types

# Write operations will use primary
curl -X POST http://localhost:3001/api/v1/equipment-types \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Equipment", "class": "TRACTOR"}'
```

## Monitoring

### Check Replication Status

```sql
-- On primary database
SELECT client_addr, state, sync_state, replay_lag 
FROM pg_stat_replication;

-- On replica
SELECT now() - pg_last_xact_replay_timestamp() AS replication_lag;
```

### Application Logs

Look for these messages in your application logs:

```
INFO: 🚀 Established connection to primary Postgres database!
INFO: 🚀 Established connection to read replica! replica=replica1
INFO: 🚀 Established connection to read replica! replica=replica2
WARN: no healthy read replicas available, falling back to primary
```

## Troubleshooting

### Replicas Not Receiving Queries

1. Check configuration:

   ```bash
   cat config/local/config.local.yaml | grep -A 20 "enableReadWriteSeparation"
   ```

2. Verify replicas are healthy:

   ```bash
   docker exec trenova-db-replica1-1 pg_isready
   docker exec trenova-db-replica2-1 pg_isready
   ```

3. Check application logs for connection errors

### High Replication Lag

1. Check current lag:

   ```bash
   docker exec trenova-db-replica1-1 psql -U postgres -c "SELECT now() - pg_last_xact_replay_timestamp() AS lag;"
   ```

2. Increase `replicaLagThreshold` in config if needed

### Replicas Not Starting

For Bitnami setup:

```bash
# Check logs
docker-compose -f docker-compose-bitnami-replicas.yml logs db-replica1

# Restart replicas
docker-compose -f docker-compose-bitnami-replicas.yml restart db-replica1 db-replica2
```

## Clean Up

```bash
# Stop all services
docker-compose -f docker-compose-bitnami-replicas.yml down

# Remove volumes (warning: deletes all data)
docker-compose -f docker-compose-bitnami-replicas.yml down -v
```

## Next Steps

- Configure production replicas with proper resource limits
- Set up monitoring and alerting for replication lag
- Test failover scenarios
- Implement read-after-write consistency where needed
