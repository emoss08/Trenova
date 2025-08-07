<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Database Interface Design Pattern

## Overview

The database interface pattern in Trenova provides an abstraction layer between the application logic and the database implementation. This document explains the design decisions, benefits, and implementation details of this pattern.

## Core Interface

```go
type Connection interface {
    DB(ctx context.Context) (*bun.DB, error)
    Close() error
}
```

## Design Philosophy

Our database interface design follows several key principles:

1. **Dependency Inversion Principle (DIP)**:
   - The application core depends on abstractions (interfaces) rather than concrete implementations
   - This allows the database implementation to be swapped without changing the application logic
   - Higher-level modules (repositories) remain independent of lower-level modules (database drivers)

2. **Single Responsibility Principle (SRP)**:
   - The interface focuses solely on database connection management
   - Implementation details (connection pooling, reconnection logic) are encapsulated
   - Each concrete implementation handles its specific database type concerns

3. **Interface Segregation Principle (ISP)**:
   - The interface is minimal and focused
   - Only essential methods are exposed
   - Additional functionality can be added through composition

## Benefits

### 1. Database Portability

```go
// PostgreSQL Implementation
type PostgresConnection struct {
    // PostgreSQL specific fields
}

// MySQL Implementation
type MySQLConnection struct {
    // MySQL specific fields
}
```

Both implementations satisfy the same interface, making database switching possible without changing application code.

### 2. Testing

```go
// Mock Implementation
type MockConnection struct {
    mock.Mock
}

func (m *MockConnection) DB(ctx context.Context) (*bun.DB, error) {
    args := m.Called(ctx)
    return args.Get(0).(*bun.DB), args.Error(1)
}
```

- Easy to mock for unit testing
- No real database needed for tests
- Can simulate different scenarios and error conditions

### 3. Connection Management

```go
func (c *Connection) DB(ctx context.Context) (*bun.DB, error) {
    c.mu.RLock()
    if c.db != nil {
        defer c.mu.RUnlock()
        return c.db, nil
    }
    c.mu.RUnlock()

    // Connection creation logic
}
```

- Centralized connection handling
- Thread-safe implementation
- Connection pooling and lifecycle management

### 4. Error Handling

```go
// Consistent error handling across different database implementations
var (
    DBConnStringEmpty = eris.New("database connection string is empty")
    DBConfigNil       = eris.New("database config is nil")
    AppConfigNil      = eris.New("application config is nil")
)
```

## Implementation Example

### Repository Layer

```go
type WorkerRepository struct {
    conn db.Connection
    // other fields
}

func (r *WorkerRepository) Create(ctx context.Context, worker *Worker) error {
    db, err := r.conn.DB(ctx)
    if err != nil {
        return err
    }
    
    return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
        // Transaction logic
    })
}
```

## Migration Strategy

To migrate to a different database:

1. Create new implementation of the Connection interface
2. Update dependency injection configuration
3. Implement database-specific migrations
4. No changes needed in repository or business logic

```go
// Example of switching databases in dependency injection
func NewApp(dbType string) *fx.App {
    return fx.New(
        fx.Provide(
            func(p ConnectionParams) db.Connection {
                switch dbType {
                case "postgres":
                    return postgres.NewConnection(p)
                case "mysql":
                    return mysql.NewConnection(p)
                default:
                    panic("unsupported database type")
                }
            },
        ),
    )
}
```

## Best Practices

1. **Keep the Interface Minimal**:
   - Only include methods that are truly database-agnostic
   - Avoid database-specific features in the interface

2. **Use Context**:
   - All database operations should accept context
   - Enables timeout and cancellation handling

3. **Error Handling**:
   - Define clear error types
   - Wrap low-level database errors
   - Provide meaningful error messages

4. **Configuration**:
   - Use dependency injection for configuration
   - Keep database-specific configuration separate

## Limitations and Considerations

1. **ORM Specificity**:
   - Our interface currently returns `*bun.DB`
   - This couples us to the Bun ORM
   - Consider using a more generic interface if ORM independence is needed

2. **Feature Parity**:
   - Different databases have different features
   - Interface should focus on common functionality
   - Database-specific features should be handled separately

3. **Performance**:
   - Abstract layers add minimal overhead
   - Benefits of maintainability outweigh performance impact
   - Critical paths can be optimized if needed

## Conclusion

The database interface pattern provides a robust foundation for enterprise applications by:

- Enabling database portability
- Improving testability
- Centralizing connection management
- Providing consistent error handling

While it adds some complexity, the benefits of maintainability, testability, and flexibility make it a valuable architectural pattern for enterprise applications.
