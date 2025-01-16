# `/internal/bootstrap` Directory Documentation

## Overview

The `bootstrap` directory handles application initialization and dependency injection using `uber-go/fx`. This is where all components are wired together and the application lifecycle is managed.

## Directory Structure

```markdown
/internal/bootstrap/
├── app.go                  # Main application setup and lifecycle
└── modules/               # Modular DI configuration
    ├── module.go          # Base module type/interface
    ├── api/               # API layer dependencies
    │   └── module.go
    ├── infrastructure/    # Infrastructure dependencies
    │   ├── module.go
    │   ├── database.go    # Database connections/repos
    │   ├── cache.go       # Caching services
    │   └── messaging.go   # Message queues/events
    └── services/          # Business service dependencies
        └── module.go
```

## Purpose & Guidelines

### `/bootstrap/app.go`

- Initializes the main `fx.App` instance
- Configures global application lifecycle
- Combines all modules
- Handles graceful shutdown

Example:

```go
func NewApp() *fx.App {
    return fx.New(
        // Combine all modules
        modules.Combine(
            config.NewModule(),
            database.NewModule(),
            services.NewModule(),
        ),
        
        // Register lifecycle hooks
        fx.Invoke(registerHooks),
    )
}
```

### `/bootstrap/modules`

- Each module represents a cohesive set of related dependencies
- Modules should be independent and reusable
- Follow consistent pattern for dependency declaration
- Use clear annotation for interface implementations

Example Module:

```go
func NewModule() Module {
    return Module{
        Option: fx.Options(
            fx.Provide(
                NewService,
                fx.Annotate(
                    NewRepository,
                    fx.As(new(ports.Repository)),
                ),
            ),
        ),
    }
}
```

## Best Practices

1. **Module Organization**
   - Group related dependencies together
   - Keep modules focused and single-purpose
   - Use clear, descriptive names for modules
   - Document module dependencies

2. **Dependency Declaration**
   - Use `fx.Provide` for constructors
   - Use `fx.Invoke` for side effects
   - Use `fx.Annotate` for interface bindings
   - Handle errors in constructors

3. **Lifecycle Management**
   - Implement proper startup ordering
   - Handle graceful shutdown
   - Clean up resources properly
   - Log lifecycle events

4. **Configuration**
   - Load configuration early
   - Validate configuration
   - Make configuration available to modules
   - Use strong typing for configuration

## What Does NOT Belong Here

1. **Business Logic**
   - No domain logic
   - No service implementations
   - No request handling
   - No data processing

2. **Infrastructure Code**
   - No direct database operations
   - No HTTP handlers
   - No service implementations
   - Only infrastructure initialization

3. **Application Logic**
   - No request handling
   - No business rules
   - No data transformation
   - No service orchestration

## Example Implementations

### Basic Module

```go
// modules/database/module.go
package database

import (
    "your-project/internal/core/ports"
    "your-project/internal/infrastructure/database/postgres"
    "go.uber.org/fx"
)

func NewModule() Module {
    return Module{
        Option: fx.Options(
            fx.Provide(
                postgres.NewConnection,
                fx.Annotate(
                    postgres.NewShipmentRepository,
                    fx.As(new(ports.ShipmentRepository)),
                ),
                fx.Annotate(
                    postgres.NewDriverRepository,
                    fx.As(new(ports.DriverRepository)),
                ),
            ),
        ),
    }
}
```

### Lifecycle Management

```go
// bootstrap/app.go
func registerHooks(
    lifecycle fx.Lifecycle,
    server *http.Server,
    logger Logger,
) {
    lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            go func() {
                if err := server.Start(); err != nil {
                    logger.Error("failed to start server", "error", err)
                }
            }()
            return nil
        },
        OnStop: func(ctx context.Context) error {
            return server.Shutdown(ctx)
        },
    })
}
```

### Configuration Module

```go
// modules/config/module.go
func NewModule() Module {
    return Module{
        Option: fx.Options(
            fx.Provide(
                config.LoadConfig,
                fx.Annotate(
                    config.NewValidator,
                    fx.As(new(ports.ConfigValidator)),
                ),
            ),
            // Validate configuration on startup
            fx.Invoke(validateConfig),
        ),
    }
}
```

## Best Practices for Testing

1. **Module Testing**
   - Test module initialization
   - Verify dependency graph
   - Test lifecycle hooks
   - Use fx.New for integration tests

2. **Configuration Testing**
   - Test configuration loading
   - Verify validation rules
   - Test error conditions
   - Use test configurations
