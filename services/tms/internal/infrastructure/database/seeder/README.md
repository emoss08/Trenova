# Database Seeding System

A robust, environment-aware database seeding system with dependency management and transaction safety.

## Overview

The seeding system provides:

- **Environment-aware seeding**: Different seeds for production, development, and testing
- **Dependency management**: Seeds declare dependencies and run in topological order
- **Transaction safety**: Each seed runs in its own transaction with automatic rollback on failure
- **Idempotency**: Seeds are tracked and won't re-run unless forced
- **CLI integration**: Full command-line interface for seeding operations

## Quick Start

### Creating a New Seed

1. Create a new file in the appropriate directory:
   - `seeds/base/` - Production/base seeds (all environments)
   - `seeds/development/` - Development-only seeds
   - `seeds/testing/` - Test-only seeds

2. Implement the `Seed` interface:

```go
package base

import (
    "context"

    "github.com/emoss08/trenova/internal/infrastructure/database/common"
    "github.com/emoss08/trenova/pkg/seedhelpers"
    "github.com/uptrace/bun"
)

type MyNewSeed struct {
    seedhelpers.BaseSeed
}

func NewMyNewSeed() *MyNewSeed {
    seed := &MyNewSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(
        "MyNew",                              // Unique name
        "1.0.0",                              // Version
        "Description of what this seed does", // Description
        []common.Environment{                 // Target environments
            common.EnvProduction,
            common.EnvStaging,
            common.EnvDevelopment,
            common.EnvTest,
        },
    )
    return seed
}

func (s *MyNewSeed) Run(ctx context.Context, tx bun.Tx) error {
    // Check if data already exists (idempotency)
    var count int
    err := tx.NewSelect().
        Model((*MyModel)(nil)).
        ColumnExpr("count(*)").
        Scan(ctx, &count)
    if err != nil {
        return err
    }

    if count > 0 {
        return nil // Already seeded
    }

    // Insert your seed data
    data := []MyModel{
        {Name: "Example 1"},
        {Name: "Example 2"},
    }

    _, err = tx.NewInsert().Model(&data).Exec(ctx)
    return err
}
```

3. Register the seed in `seeds/register.go`:

```go
func Register(r *seeder.Registry) {
    // Base seeds
    r.MustRegister(base.NewUSStatesSeed())
    r.MustRegister(base.NewMyNewSeed())  // Add your seed

    // Development seeds
    r.MustRegister(development.NewTestOrganizationsSeed())
}
```

### Adding Dependencies

If your seed depends on other seeds, declare them using typed `SeedID` constants:

```go
func NewUsersSeed() *UsersSeed {
    seed := &UsersSeed{}
    seed.BaseSeed = *seedhelpers.NewBaseSeed(
        "Users",
        "1.0.0",
        "Creates default users",
        []common.Environment{common.EnvDevelopment},
    )
    seed.SetDependencies(seedhelpers.BaseSeedIDs...)
    return seed
}
```

Development seeds typically depend on all base seeds. Use `BaseSeedIDs...` to automatically include them all.

For specific dependencies, use individual constants:

```go
seed.SetDependencies(seedhelpers.SeedUSStates, seedhelpers.SeedPermissions)
```

Typed `SeedID` constants are auto-generated and provide compile-time validation. See the [Code Generation](#code-generation) section below.

## Environment Types

| Environment | Seeds Applied | Use Case |
|-------------|---------------|----------|
| `production` | Base only | Live systems - minimal, essential data |
| `staging` | Base only | Pre-production testing |
| `development` | Base + Development | Local development with rich test data |
| `test` | Base + Test | Automated testing with minimal fixtures |

**Important**: Test and Development environments are isolated. Test seeds do NOT include Development seeds.

## CLI Commands

```bash
# Apply seeds for current environment
trenova db seed

# Override environment
trenova db seed --env development
trenova db seed --env test

# Run a specific seed (includes its dependencies)
trenova db seed --target Users

# Force re-apply already applied seeds
trenova db seed --force

# Continue on seed failure
trenova db seed --ignore-errors

# Preview what would be applied
trenova db seed --dry-run

# Interactive mode (confirm before applying)
trenova db seed -i

# Show verbose output
trenova db seed --verbose

# View seed status
trenova db status

# Check for orphaned seeds
trenova db seed-check

# Clean orphaned seed entries
trenova db seed-clean
```

## Code Generation

### Seed ID Generation

Seed IDs are automatically generated as typed constants for compile-time safety:

```bash
# Regenerate seed IDs
task generate-seeds

# Check if IDs need regeneration (for CI)
task generate-seeds-check
```

This generates `pkg/seedhelpers/seed_ids_gen.go` with:

```go
const (
    SeedUSStates          SeedID = "USStates"
    SeedTestOrganizations SeedID = "TestOrganizations"
)

var BaseSeedIDs = []SeedID{SeedUSStates}
var DevelopmentSeedIDs = []SeedID{SeedTestOrganizations}
var TestSeedIDs = []SeedID{}
```

### Automatic Regeneration

Seed IDs are automatically regenerated when using:

- `task db-seed` - Regenerates before seeding
- `task db-reset` - Regenerates before reset
- `task db-setup` - Regenerates before setup
- `task db-create-seed name=foo` - Regenerates after creating new seed

### Creating New Seeds

Create a new seed file with automatic ID generation:

```bash
# Create a base seed (all environments)
task db-create-seed name=MyNewSeed

# Create a development-only seed
task db-create-seed name=TestData env=dev

# Create a test-only seed
task db-create-seed name=TestFixtures env=test
```

After creation, a new typed constant `seedhelpers.SeedMyNewSeed` will be available for use as a dependency.

## Architecture

```
seeder/
├── seed.go          # Seed interface and types
├── registry.go      # Seed registration and validation
├── graph.go         # Dependency graph with topological sort
├── engine.go        # Main execution engine
├── tracker.go       # Seed history tracking (database)
├── reporter.go      # Progress reporting
└── errors.go        # Error types

seeds/
├── register.go      # Central registration
├── base/            # Production seeds (all environments)
├── development/     # Development-only seeds
└── testing/         # Test-only seeds
```

## Seed Interface

```go
type Seed interface {
    Name() string                        // Unique identifier
    Version() string                     // Semantic version
    Description() string                 // Human-readable description
    Environments() []common.Environment  // Target environments
    Dependencies() []string              // Seeds that must run first
    Run(ctx context.Context, tx bun.Tx) error  // Seed logic
}
```

## Best Practices

### 1. Make Seeds Idempotent

Always check if data exists before inserting:

```go
func (s *MySeed) Run(ctx context.Context, tx bun.Tx) error {
    exists, err := tx.NewSelect().
        Model((*MyModel)(nil)).
        Where("code = ?", "UNIQUE_CODE").
        Exists(ctx)
    if err != nil {
        return err
    }
    if exists {
        return nil
    }
    // ... insert data
}
```

### 2. Use Conflict Handling

For bulk inserts, use `ON CONFLICT`:

```go
_, err := tx.NewInsert().
    Model(&data).
    On("CONFLICT (code) DO NOTHING").
    Exec(ctx)
```

### 3. Keep Seeds Small and Focused

Each seed should have a single responsibility. Create multiple seeds rather than one large seed.

### 4. Version Your Seeds

Increment the version when seed logic changes significantly:

```go
seed.BaseSeed = *seedhelpers.NewBaseSeed(
    "Users",
    "2.0.0",  // Bumped from 1.0.0
    "Creates default users with new fields",
    // ...
)
```

### 5. Use Meaningful Names

Seed names should clearly indicate what they create:

- Good: `USStates`, `DefaultPermissions`, `TestOrganizations`
- Bad: `Seed1`, `Data`, `Stuff`

### 6. Document Dependencies

If your seed has dependencies, document why:

```go
// UsersSeed creates default users.
// Depends on:
//   - Organizations: users belong to organizations
//   - Roles: users are assigned roles
type UsersSeed struct {
    seedhelpers.BaseSeed
}
```

## Troubleshooting

### Seed Not Running

1. Check if it's registered in `seeds/register.go`
2. Check if the environment matches: `trenova db seed --env development`
3. Check if it's already applied: `trenova db status`
4. Force re-run: `trenova db seed --force --target MySeed`

### Circular Dependency Error

The seeder detects circular dependencies. Review your `SetDependencies()` calls:

```
Error: circular dependency detected: A -> B -> C -> A
```

### Missing Dependency Error

Ensure all declared dependencies are registered:

```
Error: seed "Users" has missing dependencies: Roles
```

Register the missing seed or remove the dependency.

## Seed History

Applied seeds are tracked in the `seed_history` table:

| Column | Description |
|--------|-------------|
| `name` | Seed name |
| `version` | Seed version |
| `environment` | Environment it was applied in |
| `status` | Active, Inactive, or Orphaned |
| `applied_at` | Timestamp |
| `duration_ms` | Execution time |
| `error` | Error message if failed |
