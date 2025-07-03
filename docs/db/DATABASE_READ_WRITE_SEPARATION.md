# Database Read/Write Separation

This document explains the implementation of read/write database separation in Trenova, which allows distributing read queries across multiple replicas while ensuring write operations go to the primary database.

## Overview

The read/write separation feature provides:

- Automatic routing of read queries to read replicas
- Load balancing across multiple replicas
- Health checks and automatic failover
- Monitoring of replication lag
- Backward compatibility with existing code

## Architecture

### Connection Interface

The `db.Connection` interface has been extended with read/write specific methods:

```go
type Connection interface {
    // For backward compatibility - returns write connection
    DB(ctx context.Context) (*bun.DB, error)
    
    // Returns a read-only connection (replica or primary)
    ReadDB(ctx context.Context) (*bun.DB, error)
    
    // Returns the primary write connection
    WriteDB(ctx context.Context) (*bun.DB, error)
    
    // Other existing methods...
}
```

### Configuration

Configure read replicas in your `config.yaml`:

```yaml
db:
  # Primary database configuration
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "postgres"
  database: "trenova_go_db"
  
  # Read/Write separation settings
  enableReadWriteSeparation: true
  replicaLagThreshold: 5  # seconds
  
  readReplicas:
    - name: "replica1"
      host: "localhost"
      port: 5433
      weight: 2  # Higher weight = more traffic
      maxConnections: 20
      maxIdleConns: 10
    
    - name: "replica2"
      host: "localhost"
      port: 5434
      weight: 1
      maxConnections: 20
      maxIdleConns: 10
```

## Repository Implementation

### Using the Connection Selector

The `dbutil` package provides utilities for selecting appropriate connections:

```go
import "github.com/emoss08/trenova/internal/pkg/dbutil"

type repository struct {
    db       db.Connection
    dbSelect *dbutil.ConnectionSelector
    txHelper *dbutil.TransactionHelper
}

func NewRepository(db db.Connection) *repository {
    return &repository{
        db:       db,
        dbSelect: dbutil.NewConnectionSelector(db),
        txHelper: dbutil.NewTransactionHelper(db),
    }
}
```

### Read Operations

Use `Read()` for read-only operations:

```go
func (r *repository) List(ctx context.Context) ([]*Entity, error) {
    // Read operations use read replicas
    db, err := r.dbSelect.Read(ctx)
    if err != nil {
        return nil, err
    }
    
    var entities []*Entity
    err = db.NewSelect().Model(&entities).Scan(ctx)
    return entities, err
}

func (r *repository) GetByID(ctx context.Context, id string) (*Entity, error) {
    // Single entity reads also use replicas
    db, err := r.dbSelect.Read(ctx)
    if err != nil {
        return nil, err
    }
    
    entity := new(Entity)
    err = db.NewSelect().Model(entity).Where("id = ?", id).Scan(ctx)
    return entity, err
}
```

### Write Operations

Use `Write()` for write operations:

```go
func (r *repository) Create(ctx context.Context, entity *Entity) error {
    // Write operations always use primary
    db, err := r.dbSelect.Write(ctx)
    if err != nil {
        return err
    }
    
    _, err = db.NewInsert().Model(entity).Exec(ctx)
    return err
}

func (r *repository) Update(ctx context.Context, entity *Entity) error {
    // Updates use primary database
    db, err := r.dbSelect.Write(ctx)
    if err != nil {
        return err
    }
    
    _, err = db.NewUpdate().Model(entity).WherePK().Exec(ctx)
    return err
}
```

### Transactions

Transactions always use the primary database:

```go
func (r *repository) UpdateWithTransaction(ctx context.Context, entity *Entity) error {
    // Transactions always use write connection
    return r.txHelper.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
        // All operations within transaction use primary
        _, err := tx.NewUpdate().Model(entity).WherePK().Exec(ctx)
        return err
    })
}
```

## Docker Compose Setup

### Local Development

The local docker-compose includes PostgreSQL replicas:

```yaml
services:
  db:
    image: postgres:latest
    ports:
      - "5432:5432"
    environment:
      POSTGRES_REPLICATION_MODE: master
      POSTGRES_REPLICATION_USER: replicator
      POSTGRES_REPLICATION_PASSWORD: replicator_password
    volumes:
      - ./scripts/init-replication.sh:/docker-entrypoint-initdb.d/init-replication.sh:ro

  db-replica1:
    image: postgres:latest
    ports:
      - "5433:5432"
    environment:
      POSTGRES_REPLICATION_MODE: slave
      POSTGRES_REPLICATION_USER: replicator
      POSTGRES_REPLICATION_PASSWORD: replicator_password
      POSTGRES_MASTER_SERVICE: db
    depends_on:
      - db
```

### Production

Production setup includes multiple replicas with resource limits:

```yaml
services:
  tren-db:
    # Primary database configuration
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: "2"

  tren-db-replica1:
    # Read replica configuration
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: "2"
```

## Health Checks and Monitoring

### Automatic Health Checks

The system performs health checks every 30 seconds:

- Ping test to verify connectivity
- Replication lag monitoring
- Automatic removal of unhealthy replicas

### Replication Lag Monitoring

If a replica's lag exceeds the configured threshold:

```go
replicaLagThreshold: 10  # seconds
```

The replica is temporarily marked as unhealthy and removed from the pool.

### Load Balancing

Read queries are distributed using weighted round-robin:

- Replicas with higher weights receive more traffic
- Only healthy replicas receive queries
- Automatic failover to primary if all replicas are unhealthy

## Best Practices

1. **Always use appropriate connections**:
   - Read operations → `Read()`
   - Write operations → `Write()`
   - Transactions → `TransactionHelper`

2. **Consider consistency requirements**:
   - For read-after-write consistency, use the primary
   - For eventual consistency, use replicas

3. **Monitor replication lag**:
   - Set appropriate thresholds based on your needs
   - Monitor replica health in production

4. **Test failover scenarios**:
   - Ensure your application handles replica failures gracefully
   - Test with all replicas down

## Migration Guide

### For Existing Repositories

1. Add the connection selector to your repository:

```go
dbSelect *dbutil.ConnectionSelector
```

2. Update read methods to use `Read()`:

```go
// Before
db, err := r.db.DB(ctx)

// After
db, err := r.dbSelect.Read(ctx)
```

3. Update write methods to use `Write()`:

```go
// Before
db, err := r.db.DB(ctx)

// After
db, err := r.dbSelect.Write(ctx)
```

4. Update transactions to use `TransactionHelper`:

```go
// Before
db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {

// After
r.txHelper.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
```

### Backward Compatibility

The `DB()` method continues to work and returns the write connection, ensuring existing code remains functional.

## Troubleshooting

### Common Issues

1. **All queries going to primary**:
   - Check `enableReadWriteSeparation` is true
   - Verify replicas are configured
   - Check replica health in logs

2. **Replica not receiving traffic**:
   - Check replica health status
   - Verify replication lag is within threshold
   - Check network connectivity

3. **High replication lag**:
   - Increase `replicaLagThreshold`
   - Check replica resources
   - Review primary database load

### Monitoring Logs

Look for these log messages:

```
INFO: 🚀 Established connection to read replica! replica=replica1
WARN: replica lag exceeds threshold, marking unhealthy replica=replica1 lag=15s
ERROR: read replica health check failed replica=replica2
INFO: read replica health check passed, marking healthy replica=replica1
```
