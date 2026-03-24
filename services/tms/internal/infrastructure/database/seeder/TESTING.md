# Testing Guide

This document covers the test suite for the database seeding infrastructure.

## Running Tests

```bash
# Run all seeder tests
go test ./internal/infrastructure/database/seeder/...

# Run with verbose output
go test -v ./internal/infrastructure/database/seeder/...

# Run specific test file
go test -v ./internal/infrastructure/database/seeder -run "TestGraph"

# Run with coverage
go test -cover ./internal/infrastructure/database/seeder/...
```

## Test Structure

### Unit Tests (No Database Required)

| File | Coverage |
|------|----------|
| `graph_test.go` | Dependency graph algorithms |
| `registry_test.go` | Seed registration and retrieval |
| `engine_test.go` | Execution engine with mocked tracker |
| `errors_test.go` | Error types and formatting |
| `reporter_test.go` | Progress reporter implementations |
| `gen/main_test.go` | Code generator (toConstName, parseFile, findSeeds, generateFile) |

### Seed Helper Tests

| File | Coverage |
|------|----------|
| `pkg/seedhelpers/base_test.go` | BaseSeed getters, setters, dependencies |
| `pkg/seedhelpers/seed_ids_test.go` | SeedID type and ValidateSeedID |
| `pkg/seedhelpers/context_test.go` | SeedContext initialization |

## Test Utilities

### Shared TestUtil Package (`shared/testutil/`)

The shared testutil package provides reusable test infrastructure:

```go
import (
    "github.com/emoss08/trenova/shared/testutil"
    seedermocks "github.com/emoss08/trenova/shared/testutil/seeder"
)

// Create test context with automatic cleanup
tc := testutil.NewTestContext(t)
tc.RegisterCleanup()

// For integration tests with PostgreSQL
pg := testutil.SetupPostgres(t, tc)
db := pg.DB()

// Mock seeds with functional options
seed := seedermocks.NewMockSeed("TestSeed",
    seedermocks.WithDependencies("Dep1", "Dep2"),
    seedermocks.WithEnvironments(common.EnvDevelopment),
    seedermocks.WithVersion("2.0.0"),
)

// Mock tracker for engine tests
tracker := seedermocks.NewMockTracker()
tracker.MarkApplied("ExistingSeed", "1.0.0", common.EnvDevelopment)

// Mock reporter for verifying callbacks
reporter := seedermocks.NewMockReporter()
```

### Mock Options

**MockSeed Options:**
- `WithVersion(v string)` - Set version
- `WithDescription(d string)` - Set description
- `WithEnvironments(envs ...Environment)` - Set target environments
- `WithDependencies(deps ...string)` - Set dependencies
- `WithRunFunc(fn func(ctx, tx) error)` - Custom run behavior
- `WithRunError(err error)` - Make Run() return error

**MockTracker:**
- `MarkApplied(name, version, env)` - Pre-mark seeds as applied
- `InitializeFunc` - Custom initialize behavior
- `IsAppliedFunc` - Custom IsApplied behavior

**MockReporter:**
- Tracks all callback invocations
- `StartCalls`, `SeedStarts`, `SeedCompletes`, `SeedErrors`, `CompleteCalls`

## Writing New Tests

### Graph Algorithm Tests

```go
func TestGraph_YourFeature(t *testing.T) {
    t.Parallel()

    seeds := []Seed{
        seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
        seedermocks.NewMockSeed("B"),
    }

    g := NewGraph()
    g.BuildFromSeeds(seeds)

    // Test your feature
    result, err := g.YourFeature()
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Registry Tests

```go
func TestRegistry_YourFeature(t *testing.T) {
    t.Parallel()

    r := NewRegistry()
    r.MustRegister(seedermocks.NewMockSeed("TestSeed",
        seedermocks.WithEnvironments(common.EnvDevelopment),
    ))

    // Test your feature
    result := r.YourFeature()
    assert.NotNil(t, result)
}
```

### Engine Tests (with mocked tracker)

```go
func TestEngine_YourFeature(t *testing.T) {
    t.Parallel()

    tracker := &testTracker{} // local mock or use seedermocks.NewMockTracker()
    reporter := seedermocks.NewMockReporter()

    r := NewRegistry()
    r.MustRegister(seedermocks.NewMockSeed("TestSeed",
        seedermocks.WithEnvironments(common.EnvDevelopment),
    ))

    e := NewEngine(nil, r)
    e.SetTracker(tracker)
    e.SetReporter(reporter)

    report, err := e.Execute(context.Background(), ExecuteOptions{
        Environment: common.EnvDevelopment,
        DryRun:      true, // avoid db access
    })

    require.NoError(t, err)
    assert.Equal(t, 1, report.Applied)
}
```

### Generator Tests

```go
func TestGenerator_YourFeature(t *testing.T) {
    t.Parallel()

    tmpDir := t.TempDir()
    seedFile := filepath.Join(tmpDir, "test.go")

    content := `package test
import "github.com/emoss08/trenova/pkg/seedhelpers"
func NewTest() {
    seedhelpers.NewBaseSeed("MyTest", "1.0.0", "desc", nil)
}
`
    os.WriteFile(seedFile, []byte(content), 0o644)

    seeds, err := parseFile(seedFile, "base")
    require.NoError(t, err)
    assert.Len(t, seeds, 1)
}
```

## Integration Tests

Integration tests using testcontainers are supported but not yet implemented. The infrastructure is ready in `shared/testutil/postgres.go`:

```go
//go:build integration

func TestTracker_Integration(t *testing.T) {
    testutil.RequireIntegration(t)

    tc := testutil.NewTestContext(t)
    tc.RegisterCleanup()

    pg := testutil.SetupPostgres(t, tc)
    db := pg.DB()

    tracker := NewTracker(db)
    err := tracker.Initialize(context.Background())
    require.NoError(t, err)

    // Test with real database
}
```

Run integration tests with:

```bash
go test -v ./... -tags=integration
```

## Test Coverage

Run coverage report:

```bash
go test -coverprofile=coverage.out ./internal/infrastructure/database/seeder/...
go tool cover -html=coverage.out -o coverage.html
```

## Concurrency Testing

Registry and graph tests include concurrent access tests:

```go
func TestRegistry_Concurrent(t *testing.T) {
    r := NewRegistry()
    var wg sync.WaitGroup

    for i := range 100 {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            seed := seedermocks.NewMockSeed(fmt.Sprintf("Seed%d", n))
            _ = r.Register(seed)
        }(i)
    }

    wg.Wait()
    assert.Equal(t, 100, r.Size())
}
```

## Code Generator Testing

The generator tests verify:
- `toConstName` - Name transformation edge cases
- `parseFile` - AST parsing for NewBaseSeed calls
- `findSeeds` - Directory traversal and filtering
- `generateFile` - Template execution and categorization
- Determinism - Multiple runs produce identical output
