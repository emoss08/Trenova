# `/cmd` Directory Documentation

## Overview

The `cmd` directory contains the entry point for all applications in the project. Each subdirectory represents a separate executable/binary with its own `main` function. These should be kept minimal and focused solely on application bootstrapping.

## Directory Structure

```markdown
/cmd/
├── api/                    # Main API server
│   └── main.go
├── worker/                # Background job processor
│   └── main.go
└── scheduler/            # Route optimization scheduler
    └── main.go
```

## Guidelines & Rules

### General Rules

- Each subdirectory should have a single `main.go` file
- Keep `main.go` files light and focused on bootstrapping
- Use dependency injection for wiring components
- Handle graceful shutdown properly
- Implement proper signal handling

### What Should Be in `main.go`

1. Configuration loading
2. Dependency injection setup
3. Application bootstrapping
4. Signal handling
5. Graceful shutdown

## Example Implementation

```go
// cmd/api/main.go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    "your-project/internal/bootstrap"
    "your-project/internal/pkg/config"
    "your-project/internal/pkg/logger"
)

func main() {
    // Initialize logger first for early error reporting
    log := logger.New()

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("failed to load config", "error", err)
    }

    // Create application context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Initialize the application using DI
    app := bootstrap.NewApp(
        bootstrap.WithConfig(cfg),
        bootstrap.WithLogger(log),
    )

    // Start the application
    if err := app.Start(ctx); err != nil {
        log.Fatal("failed to start application", "error", err)
    }

    // Handle shutdown signals
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

    // Wait for shutdown signal
    sig := <-shutdown
    log.Info("received shutdown signal", "signal", sig)

    // Create shutdown context with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    // Graceful shutdown
    if err := app.Shutdown(shutdownCtx); err != nil {
        log.Error("failed to shutdown gracefully", "error", err)
        os.Exit(1)
    }

    log.Info("application shutdown complete")
}
```

## Best Practices

1. **Configuration Management**
   - Load configuration early
   - Validate configuration
   - Use environment-based config
   - Fail fast on invalid config

2. **Error Handling**
   - Log errors with context
   - Fail fast on critical errors
   - Provide clear error messages
   - Handle panics

3. **Graceful Shutdown**
   - Handle multiple signals
   - Set reasonable timeouts
   - Close resources properly
   - Wait for operations to complete

4. **Dependency Management**
   - Use dependency injection
   - Initialize in proper order
   - Handle cleanup properly
   - Log initialization steps

## What Does NOT Belong Here

1. **Business Logic**
   - No domain logic
   - No request handling
   - No data processing
   - No business rules

2. **Infrastructure Code**
   - No database operations
   - No direct external service calls
   - No complex initialization logic
   - Move to appropriate packages

3. **Configuration Logic**
   - No complex config processing
   - No business config rules
   - Move to config package

## Different Binary Types

### API Server (`cmd/api/main.go`)

- HTTP server initialization
- Route setup
- Middleware configuration
- API documentation setup

### Worker (`cmd/worker/main.go`)

- Job queue connections
- Worker pool setup
- Task processing initialization
- Retry configuration

### Scheduler (`cmd/scheduler/main.go`)

- Cron job setup
- Task scheduling
- Background process management
- Resource management

## Testing Guidelines

1. **Integration Testing**
   - Test application startup
   - Test configuration loading
   - Test shutdown handling
   - Test with different configs

2. **Error Cases**
   - Test invalid configuration
   - Test startup failures
   - Test shutdown failures
   - Test signal handling

Example Test:

```go
func TestAPIServerStartup(t *testing.T) {
    tests := []struct {
        name    string
        config  config.Config
        wantErr bool
    }{
        {
            name: "valid configuration",
            config: config.Config{
                Server: config.ServerConfig{
                    Port: 8080,
                },
                Database: config.DatabaseConfig{
                    URL: "postgres://localhost:5432/test",
                },
            },
            wantErr: false,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```
