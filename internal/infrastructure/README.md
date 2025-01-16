# `/internal/infrastructure` Directory Documentation

## Overview

The `infrastructure` directory contains all implementations of external services, databases, caching, messaging, and third-party integrations. This layer adapts external concerns to match your domain interfaces.

## Directory Structure

```markdown
/internal/infrastructure/
├── auth/                    # Authentication implementations
│   ├── jwt/
│   └── oauth/
├── cache/                   # Caching implementations
│   ├── redis/
│   └── memcached/
├── database/               # Database implementations
│   ├── postgres/
│   │   ├── connection.go
│   │   ├── migrations/
│   │   └── repositories/
│   └── mongodb/
├── messaging/              # Message queue implementations
│   ├── kafka/
│   └── rabbitmq/
├── storage/               # File storage implementations
│   ├── s3/
│   └── gcs/
├── telemetry/            # Monitoring implementations
│   ├── prometheus/
│   └── opentelemetry/
└── external/             # Third-party service integrations
    ├── maps/
    ├── weather/
    └── traffic/
```

## Guidelines & Rules

### General Rules

- Each implementation should satisfy an interface defined in `core/ports`
- Keep external service logic isolated from domain logic
- Handle connection lifecycle management
- Implement proper error handling and retries
- Use dependency injection for configuration

### Implementation Pattern

```go
// Example repository implementation
type postgresShipmentRepository struct {
    db     *sqlx.DB
    logger logger.Logger
}

// Constructor follows dependency injection pattern
func NewShipmentRepository(db *sqlx.DB, logger logger.Logger) ports.ShipmentRepository {
    return &postgresShipmentRepository{
        db:     db,
        logger: logger,
    }
}

// Methods implement the domain interface
func (r *postgresShipmentRepository) Create(ctx context.Context, shipment *domain.Shipment) error {
    // Implementation details here
}
```

## Best Practices

1. **Error Handling**
   - Convert infrastructure errors to domain errors
   - Implement retry mechanisms for transient failures
   - Log detailed error information
   - Maintain error context

2. **Connection Management**
   - Implement connection pooling
   - Handle connection lifecycle
   - Implement health checks
   - Proper resource cleanup

3. **Monitoring & Telemetry**
   - Implement metrics collection
   - Add tracing where appropriate
   - Monitor resource usage
   - Log important operations

4. **Configuration**
   - Use environment-based configuration
   - Implement timeouts
   - Configure retry policies
   - Set appropriate limits

## What Does NOT Belong Here

1. **Domain Logic**
   - Business rules
   - Domain entities
   - Value objects
   - Domain events

2. **Application Logic**
   - Use case orchestration
   - Business process flows
   - Service coordination

3. **API Concerns**
   - Request/response handling
   - Route definitions
   - Middleware logic
   - API documentation

## Example Implementations

### Database Repository

```go
// infrastructure/database/postgres/repositories/shipment_repository.go
package postgres

import (
    "context"
    "your-project/internal/core/domain"
    "your-project/internal/core/ports"
    "github.com/jmoiron/sqlx"
)

type shipmentRepository struct {
    db     *sqlx.DB
    logger logger.Logger
}

func NewShipmentRepository(db *sqlx.DB, logger logger.Logger) ports.ShipmentRepository {
    return &shipmentRepository{
        db:     db,
        logger: logger,
    }
}

func (r *shipmentRepository) Create(ctx context.Context, shipment *domain.Shipment) error {
    query := `INSERT INTO shipments (...) VALUES (...)`
    
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Implementation with proper error handling
    if err := tx.ExecContext(ctx, query, ...); err != nil {
        return r.handleError(err)
    }

    return tx.Commit()
}

func (r *shipmentRepository) handleError(err error) error {
    // Convert infrastructure errors to domain errors
    switch {
    case isPrimaryKeyViolation(err):
        return domain.ErrDuplicateShipment
    case isConnectionError(err):
        return domain.ErrRepository
    default:
        return err
    }
}
```

### External Service Client

```go
// infrastructure/external/maps/google.go
package maps

type googleMapsClient struct {
    client    *maps.Client
    logger    logger.Logger
    apiKey    string
    rateLimit *rate.Limiter
}

func NewGoogleMapsClient(config Config, logger logger.Logger) (ports.MapsService, error) {
    client, err := maps.NewClient(maps.WithAPIKey(config.APIKey))
    if err != nil {
        return nil, fmt.Errorf("failed to create maps client: %w", err)
    }

    return &googleMapsClient{
        client:    client,
        logger:    logger,
        apiKey:    config.APIKey,
        rateLimit: rate.NewLimiter(rate.Every(time.Second), 50),
    }
}

func (c *googleMapsClient) GetDistance(ctx context.Context, origin, destination domain.Address) (*domain.Distance, error) {
    if err := c.rateLimit.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit exceeded: %w", err)
    }

    // Implementation with proper error handling
    result, err := c.client.DistanceMatrix(ctx, ...)
    if err != nil {
        c.logger.Error("failed to get distance", "error", err)
        return nil, c.handleError(err)
    }

    return mapToDistance(result), nil
}
```

## Testing Guidelines

1. **Integration Tests**
   - Test actual infrastructure when possible
   - Use test containers for databases
   - Mock external services appropriately
   - Test error conditions

2. **Mocking**
   - Create test doubles for external services
   - Use interface-based mocking
   - Test different response scenarios
   - Test timeout and error cases
