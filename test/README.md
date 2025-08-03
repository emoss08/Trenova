<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# `/test` Directory Documentation

## Overview

The `test` directory contains all supporting test code including integration tests, performance tests, test utilities, and mock data. This directory is separate from unit tests, which should remain next to the code they test.

## Directory Structure

```markdown
/test/
├── integration/                  # Integration tests
│   ├── api/                     # API integration tests
│   │   ├── shipment_test.go
│   │   └── setup.go
│   └── infrastructure/          # Infrastructure integration tests
│       ├── database_test.go
│       └── messaging_test.go
├── load/                        # Load & performance tests
│   └── k6/                      # k6 load testing scripts
│       ├── scenarios.js
│       └── environments/
├── mocks/                       # Mock implementations
│   ├── repositories/
│   ├── services/
│   └── generators/              # Test data generators
└── testutil/                    # Shared test utilities
    ├── containers/              # Test container setup
    ├── fixtures/                # Test data fixtures
    └── assertions/              # Custom test assertions
```

## Integration Tests Guidelines

### Structure

```go
// test/integration/api/shipment_test.go
package integration

import (
    "context"
    "testing"
    "your-project/test/testutil"
)

func TestShipmentAPI(t *testing.T) {
    // Setup test environment
    ctx := context.Background()
    env, err := testutil.NewTestEnvironment(ctx)
    if err != nil {
        t.Fatal(err)
    }
    defer env.Cleanup(ctx)

    // Test cases
    tests := []struct {
        name     string
        payload  CreateShipmentRequest
        wantCode int
        wantErr  bool
    }{
        // Test cases here
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Load Testing Guidelines

### K6 Script Structure

```javascript
// test/load/k6/scenarios.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    scenarios: {
        shipment_creation: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '2m', target: 100 },
                { duration: '5m', target: 100 },
                { duration: '2m', target: 0 },
            ],
        },
    },
};

export default function() {
    const response = http.post('http://api/shipments', {
        // Test data
    });

    check(response, {
        'status is 201': (r) => r.status === 201,
        'response time < 200ms': (r) => r.timings.duration < 200,
    });

    sleep(1);
}
```

## Mock Implementation Guidelines

```go
// test/mocks/repositories/shipment_repository.go
type MockShipmentRepository struct {
    mock.Mock
}

func (m *MockShipmentRepository) Create(ctx context.Context, shipment *domain.Shipment) error {
    args := m.Called(ctx, shipment)
    return args.Error(0)
}

// Test utility for creation
func NewMockShipmentRepository() *MockShipmentRepository {
    mock := &MockShipmentRepository{}
    // Setup default behaviors
    return mock
}
```

## Test Utilities

### Container Management

```go
// test/testutil/containers/postgres.go
func NewPostgresContainer(ctx context.Context) (*PostgresContainer, error) {
    req := testcontainers.ContainerRequest{
        Image:        "postgres:14",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_DB":       "testdb",
            "POSTGRES_USER":     "test",
            "POSTGRES_PASSWORD": "test",
        },
        WaitingFor: wait.ForLog("database system is ready to accept connections"),
    }

    container, err := testcontainers.GenericContainer(ctx, req)
    if err != nil {
        return nil, err
    }

    return &PostgresContainer{
        Container: container,
    }, nil
}
```

## Best Practices

1. **Integration Tests**
   - Use test containers
   - Clean up resources
   - Isolate tests
   - Use proper timeouts

2. **Load Tests**
   - Define clear scenarios
   - Set realistic thresholds
   - Monitor resource usage
   - Test edge cases

3. **Mock Data**
   - Use realistic test data
   - Maintain data consistency
   - Document data relationships
   - Version control fixtures

4. **Test Utilities**
   - Keep utilities focused
   - Document usage
   - Handle cleanup
   - Make reusable

## What Does NOT Belong Here

1. **Unit Tests**
   - Should be next to source code
   - Keep with implementation
   - Move to respective packages

2. **Production Code**
   - No business logic
   - No actual implementations
   - No production configurations

3. **Documentation**
   - Move to /docs
   - Keep API documentation separate
   - Use appropriate tools

## Example Test Setup

```go
// test/testutil/environment.go
type TestEnvironment struct {
    DB        *sqlx.DB
    Redis     *redis.Client
    Kafka     *kafka.Client
    Cleanup   func()
}

func NewTestEnvironment(ctx context.Context) (*TestEnvironment, error) {
    // Start required containers
    postgres, err := NewPostgresContainer(ctx)
    if err != nil {
        return nil, err
    }

    redis, err := NewRedisContainer(ctx)
    if err != nil {
        postgres.Terminate(ctx)
        return nil, err
    }

    // Initialize connections
    db, err := sqlx.Connect("postgres", postgres.ConnectionString())
    if err != nil {
        postgres.Terminate(ctx)
        redis.Terminate(ctx)
        return nil, err
    }

    return &TestEnvironment{
        DB:      db,
        Redis:   redisClient,
        Cleanup: func() {
            db.Close()
            postgres.Terminate(ctx)
            redis.Terminate(ctx)
        },
    }, nil
}
```

## Testing Strategies

1. **Database Testing**
   - Use migrations
   - Reset between tests
   - Use transactions
   - Handle race conditions

2. **API Testing**
   - Test authentication
   - Validate responses
   - Check error cases
   - Test rate limiting

3. **Performance Testing**
   - Define baselines
   - Monitor trends
   - Test scalability
   - Check resource usage
