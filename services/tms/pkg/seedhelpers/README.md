# Trenova TMS Seeding System

Comprehensive database seeding system with dependency management, rollback support, and data externalization.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Creating Seeds](#creating-seeds)
- [Advanced Features](#advanced-features)
- [CLI Commands](#cli-commands)
- [Best Practices](#best-practices)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)

## Overview

The Trenova seeding system provides:

- **Dependency Management**: Seeds execute in correct order based on dependencies
- **Rollback Support**: Undo seeds during development with transaction safety
- **Persistent Tracking**: Entity tracking survives across application restarts
- **Shared State**: Pass data between seeds without database queries
- **Data Externalization**: Separate data from code using YAML/JSON files
- **Environment Support**: Different seeds for development, test, staging, production
- **Comprehensive Logging**: Visibility into seed operations and cache performance

## Quick Start

### Creating Your First Seed

```bash
# Generate scaffold
trenova db create-seed Worker --env development

# Edit the generated file
vim internal/infrastructure/database/seeds/development/03_worker.go

# Apply the seed
trenova db seed
```

### Basic Seed Structure

```go
package development

import (
    "context"
    "github.com/emoss08/trenova/internal/infrastructure/database/common"
    "github.com/emoss08/trenova/pkg/seedhelpers"
    "github.com/uptrace/bun"
)

type WorkerSeed struct {
    seedhelpers.BaseSeed
}

func NewWorkerSeed() *WorkerSeed {
    seed := &WorkerSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(
        "Worker",
        "1.0.0",
        "Creates worker data",
        []common.Environment{common.EnvDevelopment},
    )
    seed.SetDependencies(seedhelpers.SeedAdminAccount)
    return seed
}

func (s *WorkerSeed) Run(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            // Get default organization (no boilerplate!)
            org, err := sc.GetDefaultOrganization()
            if err != nil {
                return err
            }

            // Create worker
            worker := &worker.Worker{
                OrganizationID: org.ID,
                BusinessUnitID: org.BusinessUnitID,
                Name:           "John Doe",
                Code:           "JD001",
            }

            if _, err := tx.NewInsert().Model(worker).Exec(ctx); err != nil {
                return err
            }

            // Track for rollback
            if err := sc.TrackCreated(ctx, "workers", worker.ID, s.Name()); err != nil {
                return err
            }

            return nil
        })
}

func (s *WorkerSeed) Down(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
        })
}

func (s *WorkerSeed) CanRollback() bool {
    return true
}
```

## Core Concepts

### SeedContext

`SeedContext` is the central coordinator for seed operations. It provides:

1. **Shared State** - Store and retrieve data between seeds
2. **Entity Tracking** - Automatic tracking of created entities for rollback
3. **Helper Methods** - Simplified entity creation with validation
4. **Caching** - Reduce redundant database queries

```go
// Store data for other seeds
sc.Set("default_org", org)

// Retrieve data (no query!)
org, err := sc.GetOrganization("default_org")

// Create with helpers
user, err := sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
    OrganizationID: org.ID,
    Name:           "Admin User",
    Email:          "admin@example.com",
    Password:       "secure123",
}, "MySeed")
```

### Entity Tracking

All created entities are automatically tracked in the `seed_created_entities` table. This enables:

- **Rollback support** across application restarts
- **Audit trail** of what each seed created
- **Dependency validation** prevents breaking rollbacks

```go
// Manual tracking
if err := sc.TrackCreated(ctx, "organizations", org.ID, s.Name()); err != nil {
    return err
}

// Automatic tracking (when using helpers)
org, err := sc.CreateOrganization(ctx, tx, opts, s.Name())
// ^ Automatically tracked!
```

### Dependency Management

Seeds declare dependencies that determine execution order:

```go
func NewWorkerSeed() *WorkerSeed {
    seed := &WorkerSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(
        "Worker",
        "1.0.0",
        "Creates worker data",
        []common.Environment{common.EnvDevelopment},
    )

    // Worker depends on AdminAccount
    seed.SetDependencies(seedhelpers.SeedAdminAccount)

    return seed
}
```

The engine automatically:
- Resolves dependency order using topological sort
- Detects circular dependencies
- Validates all dependencies exist

## Creating Seeds

### Using Scaffold Generator

```bash
# Base seed (all environments)
trenova db create-seed MyEntity

# Development seed
trenova db create-seed MyEntity --env development

# With YAML data file
trenova db create-seed MyEntity --env development --with-data
```

### Manual Creation

1. Create file: `internal/infrastructure/database/seeds/{env}/##_myentity.go`
2. Implement `Seed` interface:
   - `Run(ctx, tx)` - Create entities
   - `Down(ctx, tx)` - Rollback (optional)
   - `CanRollback()` - Return true if rollback supported
3. Register in `internal/infrastructure/database/seeder/seeds/register.go`
4. Run `task generate-seeds` to update seed IDs

### With YAML Data

**Step 1: Create YAML file**

`seeds/development/data/workers.yaml`:
```yaml
workers:
  - name: John Doe
    code: JD001
    type: driver
  - name: Jane Smith
    code: JS002
    type: warehouse
```

**Step 2: Load in seed**

```go
func (s *WorkerSeed) Run(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            loader := seedhelpers.NewDataLoader("./internal/.../seeds/development/data")

            var data struct {
                Workers []struct {
                    Name string `yaml:"name"`
                    Code string `yaml:"code"`
                    Type string `yaml:"type"`
                } `yaml:"workers"`
            }

            if err := loader.LoadYAML("workers.yaml", &data); err != nil {
                return err
            }

            org, _ := sc.GetDefaultOrganization()

            for _, w := range data.Workers {
                worker := &worker.Worker{
                    OrganizationID: org.ID,
                    Name:           w.Name,
                    Code:           w.Code,
                    WorkerType:     w.Type,
                }

                if _, err := tx.NewInsert().Model(worker).Exec(ctx); err != nil {
                    return err
                }

                if err := sc.TrackCreated(ctx, "workers", worker.ID, s.Name()); err != nil {
                    return err
                }
            }

            return nil
        })
}
```

## Advanced Features

### Shared State Pattern

**Problem:** FormulaTemplate needs the organization that AdminAccount just created.

**Old Way (50+ lines):**
```go
// Query database
var org tenant.Organization
err := tx.NewSelect().Model(&org).Where("scac_code = ?", "TRNV").Scan(ctx)
// ... error handling, validation, etc.
```

**New Way (1 line):**
```go
org, err := sc.GetOrganization("default_org")
```

**How it works:**

1. AdminAccount stores org in shared state:
   ```go
   sc.Set("default_org", org)
   ```

2. FormulaTemplate retrieves it (no query):
   ```go
   org, err := sc.GetOrganization("default_org")
   ```

### Helper Methods

#### CreateOrganization

```go
org, err := sc.CreateOrganization(ctx, tx, &seedhelpers.OrganizationOptions{
    BusinessUnitID: bu.ID,
    Name:           "Acme Corp",
    ScacCode:       "ACME",
    AddressLine1:   "123 Main St",
    City:           "Los Angeles",
    StateID:        state.ID,
    PostalCode:     "90001",
    Timezone:       "America/Los_Angeles",
}, s.Name())
```

**Benefits:**
- Automatic validation
- Automatic tracking
- Consistent entity creation
- Error wrapping with context

#### CreateUser

```go
user, err := sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
    OrganizationID: org.ID,
    BusinessUnitID: bu.ID,
    Name:           "Admin User",
    Username:       "admin",
    Email:          "admin@example.com",
    Password:       "secure123!",
    Status:         domaintypes.StatusActive,
}, s.Name())
```

#### CreateBusinessUnit

```go
bu, err := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
    Name: "Main Business Unit",
    Code: "MAIN",
}, s.Name())
```

### Caching

SeedContext automatically caches:
- States (by abbreviation)
- Organizations (by SCAC code)
- Business units

```go
// First call: queries database
state, err := sc.GetState(ctx, "CA")

// Second call: returns from cache
state, err := sc.GetState(ctx, "CA")  // Instant!
```

Cache statistics are shown with `--verbose`:
```
Statistics: Cache: 2 hits, 1 miss (66.7% hit rate)
```

### Logging Levels

```go
logger := seedhelpers.NewConsoleSeedLogger(true)  // verbose

// Or custom logger
type MyLogger struct{}

func (l *MyLogger) EntityCreated(table string, id pulid.ID, desc string) {
    // Custom logic
}

// Implement other SeedLogger methods...
```

### Transaction Management

All seeds run in transactions for data consistency:

```go
return seedhelpers.RunInTransaction(ctx, tx, s.Name(), logger,
    func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
        // Your seed logic here
        // If any error occurs, entire transaction rolls back
        return nil
    })
```

## CLI Commands

### Seeding

```bash
# Apply all seeds for current environment
trenova db seed

# Apply specific seed (with dependencies)
trenova db seed --target WorkerSeed

# Force re-apply already applied seeds
trenova db seed --force

# Show what would be applied without applying
trenova db seed --dry-run

# Verbose output with statistics
trenova db seed --verbose

# Override environment
trenova db seed --env staging
```

### Rollback

```bash
# Rollback specific seed
trenova db seed rollback FormulaTemplate

# Preview rollback (dry-run)
trenova db seed rollback FormulaTemplate --dry-run

# Rollback all seeds in reverse order
trenova db seed rollback --all

# Verbose output
trenova db seed rollback FormulaTemplate --verbose
```

### Status and Validation

```bash
# Show all applied seeds
trenova db seed status

# Check for orphaned seeds
trenova db seed-check

# Clean up orphaned history entries
trenova db seed-clean

# Show migrations and seeds
trenova db status
```

### Development

```bash
# Create new seed
trenova db create-seed Worker --env development

# Regenerate seed registry
trenova db seed-sync

# Watch for changes and auto-regenerate
trenova db seed-watch
```

## Best Practices

### 1. Use Descriptive Names

**Good:**
```go
seedhelpers.NewBaseSeed("Worker", "1.0.0", "Creates driver and warehouse worker accounts", ...)
```

**Bad:**
```go
seedhelpers.NewBaseSeed("W", "1.0.0", "Workers", ...)
```

### 2. Always Track Created Entities

```go
// Even if not planning to rollback, track for audit purposes
if err := sc.TrackCreated(ctx, "workers", worker.ID, s.Name()); err != nil {
    return err
}
```

### 3. Use Helper Methods

**Good:**
```go
org, err := sc.CreateOrganization(ctx, tx, opts, s.Name())
```

**Bad:**
```go
org := &tenant.Organization{...}
_, err := tx.NewInsert().Model(org).Exec(ctx)
// Missing: validation, tracking, error context
```

### 4. Leverage Shared State

**Good:**
```go
// AdminAccount seed
sc.Set("default_org", org)

// Worker seed
org, err := sc.GetOrganization("default_org")
```

**Bad:**
```go
// Query database again
var org tenant.Organization
err := tx.NewSelect().Model(&org).Where(...).Scan(ctx)
```

### 5. Externalize Data

**Good:**
```yaml
# data/workers.yaml
workers:
  - name: John Doe
    code: JD001
```

**Bad:**
```go
// Hardcoded in Go
workers := []Worker{
    {Name: "John Doe", Code: "JD001"},
    {Name: "Jane Smith", Code: "JS002"},
    // ... 100 more
}
```

### 6. Implement Rollback Support

```go
func (s *MySeed) Down(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
        })
}

func (s *MySeed) CanRollback() bool {
    return true  // Enable rollback
}
```

### 7. Declare Dependencies

```go
// Worker depends on AdminAccount (creates org)
seed.SetDependencies(seedhelpers.SeedAdminAccount)

// Multiple dependencies
seed.SetDependencies(
    seedhelpers.SeedAdminAccount,
    seedhelpers.SeedUSStates,
)
```

### 8. Use Proper Error Wrapping

```go
if err := sc.CreateUser(...); err != nil {
    return fmt.Errorf("create admin user: %w", err)
}
```

### 9. Validate Options

```go
type WorkerOptions struct {
    Name string
    Code string
}

func (opts *WorkerOptions) Validate() error {
    if opts.Name == "" {
        return fmt.Errorf("name is required")
    }
    if opts.Code == "" {
        return fmt.Errorf("code is required")
    }
    return nil
}
```

### 10. Keep Seeds Idempotent

Seeds should be safe to run multiple times:

```go
// Check if entity exists first
var existing Worker
err := tx.NewSelect().Model(&existing).Where("code = ?", "JD001").Scan(ctx)
if err == nil {
    // Already exists, skip
    return nil
}
```

## API Reference

### SeedContext Methods

#### Shared State

```go
// Store value
Set(key string, value any) error

// Retrieve value
Get(key string) (any, bool)

// Type-safe getters
GetOrganization(key string) (*tenant.Organization, error)
GetUser(key string) (*tenant.User, error)
GetBusinessUnit(key string) (*tenant.BusinessUnit, error)
```

#### Entity Tracking

```go
// Track created entity
TrackCreated(ctx context.Context, table string, id pulid.ID, seedName string) error

// Get tracked entities for seed
GetCreatedEntities(ctx context.Context, seedName string) ([]TrackedEntity, error)

// Get all tracked entities
GetAllCreatedEntities(ctx context.Context) ([]TrackedEntity, error)

// Delete tracking records
DeleteTrackedEntities(ctx context.Context, seedName string) error
```

#### Helper Methods

```go
// Create organization
CreateOrganization(ctx context.Context, tx bun.Tx, opts *OrganizationOptions, seedName string) (*tenant.Organization, error)

// Create user
CreateUser(ctx context.Context, tx bun.Tx, opts *UserOptions, seedName string) (*tenant.User, error)

// Create business unit
CreateBusinessUnit(ctx context.Context, tx bun.Tx, opts *BusinessUnitOptions, seedName string) (*tenant.BusinessUnit, error)
```

#### Query Helpers

```go
// Get default organization
GetDefaultOrganization() (*tenant.Organization, error)

// Get default business unit
GetDefaultBusinessUnit(ctx context.Context) (*tenant.BusinessUnit, error)

// Get state by abbreviation (cached)
GetState(ctx context.Context, abbreviation string) (*usstate.UsState, error)

// Get organization by SCAC code (cached)
GetOrganizationByScac(ctx context.Context, scacCode string) (*tenant.Organization, error)
```

### DataLoader

```go
// Create loader
loader := seedhelpers.NewDataLoader("./path/to/data")

// Load YAML
err := loader.LoadYAML("file.yaml", &data)

// Load JSON
err := loader.LoadJSON("file.json", &data)
```

### Logger Interface

```go
type SeedLogger interface {
    EntityCreated(table string, id pulid.ID, description string)
    EntityQueried(table string, id pulid.ID)
    CacheHit(key string)
    CacheMiss(key string)
    BulkInsert(table string, count int)
    Debug(format string, args ...any)
    Info(format string, args ...any)
    Warn(format string, args ...any)
    Error(format string, args ...any)
    SetLevel(level LogLevel)
    GetStats() *LogStats
    PrintStats()
}
```

### Rollback Helpers

```go
// Delete all tracked entities for a seed
DeleteTrackedEntities(ctx context.Context, tx bun.Tx, seedName string, sc *SeedContext) error

// Delete entities by table and IDs
DeleteEntitiesByTable(ctx context.Context, tx bun.Tx, table string, ids []string) error

// Verify entity exists
VerifyEntityExists(ctx context.Context, tx bun.Tx, table string, id string) (bool, error)
```

## Troubleshooting

### Seed Not Applying

**Problem:** Seed exists but doesn't run.

**Solutions:**
1. Check environment: `trenova db seed --env development`
2. Check if already applied: `trenova db seed status`
3. Force re-apply: `trenova db seed --force`
4. Check dependencies: `trenova db seed-check`

### Rollback Fails

**Problem:** Cannot rollback seed.

**Causes:**
1. **Dependent seeds exist**
   ```
   Error: cannot rollback AdminAccount: dependent seeds exist: [FormulaTemplate, Worker]
   ```
   **Solution:** Rollback dependents first

2. **Rollback not supported**
   ```
   Error: rollback not supported for this seed
   ```
   **Solution:** Implement `Down()` and return `true` from `CanRollback()`

3. **Entities not tracked**
   ```
   Error: no tracked entities found
   ```
   **Solution:** Ensure all entities are tracked with `sc.TrackCreated()`

### Shared State Key Not Found

**Problem:** `GetOrganization()` returns "key not found".

**Causes:**
1. Dependency not declared
2. Key name mismatch
3. Seed order incorrect

**Solution:**
```go
// In dependent seed, declare dependency
seed.SetDependencies(seedhelpers.SeedAdminAccount)

// Use correct key name
org, err := sc.GetOrganization("default_org")  // Match what AdminAccount used
```

### YAML Parse Error

**Problem:** `yaml: line 95: mapping values are not allowed`.

**Cause:** Unquoted special characters (`:`, `?`, `{`, `}`)

**Solution:**
```yaml
# Bad
expression: hasHazmat ? hazmatFee : 0

# Good
expression: "hasHazmat ? hazmatFee : 0"
```

### Circular Dependency

**Problem:** `Error: circular dependency detected`.

**Cause:** Seed A depends on B, B depends on A.

**Solution:** Refactor to remove circular dependency or merge seeds.

### Migration Not Found

**Problem:** `Error: table seed_created_entities does not exist`.

**Cause:** Migration not run.

**Solution:**
```bash
trenova db migrate
```

## Examples

### Example 1: Simple Entity Seed

```go
type ProductSeed struct {
    seedhelpers.BaseSeed
}

func NewProductSeed() *ProductSeed {
    seed := &ProductSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(
        "Product",
        "1.0.0",
        "Creates sample products",
        []common.Environment{common.EnvDevelopment},
    )
    seed.SetDependencies(seedhelpers.SeedAdminAccount)
    return seed
}

func (s *ProductSeed) Run(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            org, err := sc.GetDefaultOrganization()
            if err != nil {
                return err
            }

            products := []struct {
                Name  string
                SKU   string
                Price float64
            }{
                {"Widget", "WDG-001", 19.99},
                {"Gadget", "GDG-002", 29.99},
            }

            for _, p := range products {
                product := &catalog.Product{
                    OrganizationID: org.ID,
                    Name:           p.Name,
                    SKU:            p.SKU,
                    Price:          p.Price,
                }

                if _, err := tx.NewInsert().Model(product).Exec(ctx); err != nil {
                    return fmt.Errorf("insert product %s: %w", p.Name, err)
                }

                if err := sc.TrackCreated(ctx, "products", product.ID, s.Name()); err != nil {
                    return err
                }
            }

            return nil
        })
}

func (s *ProductSeed) Down(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
        })
}

func (s *ProductSeed) CanRollback() bool {
    return true
}
```

### Example 2: Seed with YAML Data

**data/customers.yaml:**
```yaml
customers:
  - name: Acme Corp
    contact: John Doe
    email: john@acme.com
  - name: Widget Inc
    contact: Jane Smith
    email: jane@widget.com
```

**Seed:**
```go
func (s *CustomerSeed) Run(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            loader := seedhelpers.NewDataLoader("./internal/.../data")

            var data struct {
                Customers []struct {
                    Name    string `yaml:"name"`
                    Contact string `yaml:"contact"`
                    Email   string `yaml:"email"`
                } `yaml:"customers"`
            }

            if err := loader.LoadYAML("customers.yaml", &data); err != nil {
                return err
            }

            org, _ := sc.GetDefaultOrganization()

            for _, c := range data.Customers {
                customer := &customer.Customer{
                    OrganizationID: org.ID,
                    Name:           c.Name,
                    ContactName:    c.Contact,
                    Email:          c.Email,
                }

                if _, err := tx.NewInsert().Model(customer).Exec(ctx); err != nil {
                    return fmt.Errorf("insert customer %s: %w", c.Name, err)
                }

                if err := sc.TrackCreated(ctx, "customers", customer.ID, s.Name()); err != nil {
                    return err
                }
            }

            return nil
        })
}
```

### Example 3: Seed with Relationships

```go
func (s *OrderSeed) Run(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            org, _ := sc.GetDefaultOrganization()

            // Get customer from shared state
            customer, err := sc.Get("sample_customer")
            if err != nil {
                return fmt.Errorf("get sample customer: %w", err)
            }

            // Create order
            order := &order.Order{
                OrganizationID: org.ID,
                CustomerID:     customer.ID,
                OrderNumber:    "ORD-001",
                Status:         "pending",
            }

            if _, err := tx.NewInsert().Model(order).Exec(ctx); err != nil {
                return err
            }

            if err := sc.TrackCreated(ctx, "orders", order.ID, s.Name()); err != nil {
                return err
            }

            // Store for other seeds
            sc.Set("sample_order", order)

            return nil
        })
}
```

## Performance Tips

1. **Use Bulk Inserts** for large datasets:
   ```go
   _, err := tx.NewInsert().Model(&entities).Exec(ctx)
   ```

2. **Leverage Caching** - Query once, use many times:
   ```go
   state, _ := sc.GetState(ctx, "CA")  // Cached after first call
   ```

3. **Use Shared State** - Avoid redundant queries:
   ```go
   org, _ := sc.GetOrganization("default_org")  // No query!
   ```

4. **Batch Track Calls** if needed:
   ```go
   for _, entity := range entities {
       sc.TrackCreated(ctx, "table", entity.ID, s.Name())
   }
   ```

5. **Minimize Transactions** - Use single transaction per seed:
   ```go
   seedhelpers.RunInTransaction(...)  // One transaction for entire seed
   ```

## Security Considerations

1. **Never commit sensitive data** to YAML files
2. **Use environment variables** for secrets:
   ```go
   password := os.Getenv("SEED_ADMIN_PASSWORD")
   ```
3. **Hash passwords** before storing:
   ```go
   hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
   ```
4. **Validate input** from YAML files:
   ```go
   if err := validateEmail(c.Email); err != nil {
       return err
   }
   ```
5. **Use transactions** to prevent partial data:
   ```go
   seedhelpers.RunInTransaction(...)  // Automatic rollback on error
   ```

## Migration Guide

### From Old Seeding System

**Old:**
```go
func (s *MySeed) Run(ctx context.Context, tx bun.Tx) error {
    // 50+ lines of getOrCreate helpers
    org, err := s.getOrCreateOrganization(ctx, tx)
    // ...
}
```

**New:**
```go
func (s *MySeed) Run(ctx context.Context, tx bun.Tx) error {
    return seedhelpers.RunInTransaction(ctx, tx, s.Name(), nil,
        func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
            org, _ := sc.GetDefaultOrganization()  // 1 line!
            // ...
        })
}
```

## Contributing

When adding new seeds:

1. Use scaffold generator: `trenova db create-seed <name>`
2. Follow naming conventions: `{number}_{entity}.go`
3. Implement rollback support
4. Track all created entities
5. Add tests
6. Update documentation
7. Run validation: `trenova db seed-check`

## License

Copyright © 2024 Trenova. All rights reserved.
