# Formula Template System - Design Document

## Problem Statement

Transportation billing requires flexible rate calculations that vary by:
- Customer agreements
- Shipment characteristics (weight, distance, stops)
- Special handling (hazmat, temperature control)
- Accessorial charges

Hardcoding billing logic creates maintenance burden and limits business flexibility. Users need to define custom formulas without code changes.

## Design Philosophy

### 1. Pure Computation, No I/O

The formula engine performs **zero database queries, HTTP calls, or file operations**. It operates entirely on in-memory data structures.

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Database   │────▶│ Your Service │────▶│   Formula    │
│              │     │ (loads data) │     │   Engine     │
└──────────────┘     └──────────────┘     └──────────────┘
                            │                    │
                     Does I/O here         Pure math here
```

**Why?**
- **Testability**: Unit tests work with plain structs, no database setup required
- **Performance**: Caller controls data loading strategy (batch, preload, cache)
- **Predictability**: Same inputs always produce same outputs
- **Separation of concerns**: Engine does math, repository does data

### 2. Reflection-Based Field Resolution

The engine uses Go reflection to extract values from any struct type rather than requiring specific interfaces.

```go
// Works with ANY struct - no interface implementation needed
type Shipment struct {
    Weight int64
    Moves  []Move
}

// Engine uses reflection internally
value := reflect.ValueOf(entity).FieldByName("Weight")
```

**Why?**
- **Zero boilerplate**: No need to implement `GetFormulaValue()` on every entity
- **Nested access**: Automatically traverses `Customer.Name`, `Moves[].Distance`
- **Flexibility**: Same engine works for shipments, invoices, or any future entity

**Trade-off**: Reflection is slower than direct field access, but formula evaluation is not a hot path (runs once per billing calculation, not millions of times per second).

### 3. Schema as Optional Metadata

Schemas define what variables are available, but the engine works without them.

```
With Schema:                          Without Schema:
┌─────────────────────┐              ┌─────────────────────┐
│ Schema defines:     │              │ Fallback to:        │
│ - weight → Weight   │              │ - computed functions│
│ - name → Cust.Name  │              │   only              │
│ - totalDistance →   │              │ - totalDistance     │
│   computeTotalDist  │              │ - totalStops        │
└─────────────────────┘              │ - hasHazmat, etc.   │
                                     └─────────────────────┘
```

**Why optional?**
- **Gradual adoption**: Start without schemas, add them for more control later
- **Simpler testing**: Tests don't need schema setup
- **Computed functions**: Common calculations (totalDistance) always available

**When to use schemas?**
- Exposing direct entity fields (weight, pieces)
- Nested paths (customer.name)
- Custom transforms (decimal → float64)
- Documentation of available variables

### 4. Expression Language: expr-lang

We use [expr-lang/expr](https://github.com/expr-lang/expr) instead of building a custom parser.

**Why expr-lang?**
- **Battle-tested**: Used in production by Uber, Google, and others
- **Safe**: No arbitrary code execution, sandboxed evaluation
- **Fast**: Compiles to bytecode, cached for repeated evaluation
- **Familiar syntax**: `baseRate * distance + (hasHazmat ? 150 : 0)`

**Why not alternatives?**

| Alternative | Reason Not Chosen |
|-------------|-------------------|
| Custom lexer/parser | Maintenance burden, edge cases, security risks |
| Lua/JavaScript | Overhead of embedding runtime, security concerns |
| Go templates | Not designed for math, awkward syntax |
| CEL (Google) | More complex, designed for policy not math |

### 5. Computed Functions for Derived Values

Some values require iteration over collections. These are implemented as registered functions.

```go
// Computed function - iterates over Moves slice
func computeTotalDistance(entity any) (any, error) {
    moves := getMovesViaReflection(entity)
    var total float64
    for _, move := range moves {
        total += move.Distance
    }
    return total, nil
}
```

**Available computed functions:**
| Function | Description |
|----------|-------------|
| `computeTotalDistance` | Sum of all move distances |
| `computeTotalStops` | Count of all stops across moves |
| `computeHasHazmat` | Any commodity has hazardous material |
| `computeRequiresTemperatureControl` | Temperature min/max is set |
| `computeTemperatureDifferential` | Max temp - min temp |
| `computeTotalWeight` | Shipment weight or sum of commodity weights |
| `computeTotalPieces` | Shipment pieces or sum of commodity pieces |
| `computeTotalLinearFeet` | Sum of (pieces × linearFeetPerUnit) for all commodities |
| `computeFreightChargeAmount` | Current freight charge amount on shipment |
| `computeOtherChargeAmount` | Current other charge amount on shipment |
| `computeCurrentTotalCharge` | Current total charge amount on shipment |

**Why functions instead of expression syntax?**
- Expressions can't iterate (`for` loops not supported in expr-lang)
- Keeps expressions simple and readable
- Functions are reusable across templates

### 6. Type Transforms

Database types don't always match expression needs. Transforms convert between them.

```go
// Database stores as decimal.Decimal, expressions need float64
transforms := map[string]TransformFunc{
    "decimalToFloat64": func(v any) (any, error) {
        return v.(decimal.Decimal).InexactFloat64(), nil
    },
}
```

**Available transforms:**
| Transform | From | To |
|-----------|------|-----|
| `decimalToFloat64` | `decimal.Decimal` | `float64` |
| `int64ToFloat64` | `int64` | `float64` |
| `int16ToFloat64` | `int16` | `float64` |
| `stringToUpper` | `string` | `STRING` |
| `stringToLower` | `string` | `string` |
| `unixToISO8601` | `int64` | `2024-01-15T...` |

### 7. Built-in Math Functions

Common operations are provided as expr functions.

```
round(value, decimals)  - Round to N decimal places
ceil(value)             - Round up
floor(value)            - Round down
abs(value)              - Absolute value
min(a, b)               - Minimum of two values
max(a, b)               - Maximum of two values
sum(a, b, c, ...)       - Sum of values
avg(a, b, c, ...)       - Average of values
clamp(val, min, max)    - Constrain to range
pow(base, exp)          - Exponentiation
sqrt(value)             - Square root
coalesce(a, b, c, ...)  - First non-nil value
```

**Why built-in instead of standard library?**
- expr-lang doesn't include math functions by default
- Consistent behavior across all formulas
- Can add domain-specific functions later (e.g., `mileageBand()`)

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Formula Engine                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │   Schema    │  │  Resolver   │  │   Environment Builder   │ │
│  │  Registry   │  │             │  │                         │ │
│  │             │  │ - Fields    │  │ - Combines all sources  │ │
│  │ - Storage   │  │ - Nested    │  │ - Entity values         │ │
│  │ - Lookup    │  │ - Transforms│  │ - Computed values       │ │
│  │ - Metadata  │  │             │  │ - User variables        │ │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘ │
│         │                │                     │               │
│         └────────────────┼─────────────────────┘               │
│                          │                                     │
│                          ▼                                     │
│                 ┌─────────────────┐                            │
│                 │     Engine      │                            │
│                 │                 │                            │
│                 │ - Compile (cache)                            │
│                 │ - Evaluate                                   │
│                 │ - Validate                                   │
│                 │                 │                            │
│                 └────────┬────────┘                            │
│                          │                                     │
│                          ▼                                     │
│                 ┌─────────────────┐                            │
│                 │   expr-lang     │                            │
│                 │   (bytecode VM) │                            │
│                 └─────────────────┘                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Data Flow

```
1. CALLER LOADS DATA
   ┌────────────────────────────────────────────────────────────┐
   │ shipment := repo.GetWithPreloads(ctx, id,                  │
   │     []string{"Moves", "Moves.Stops", "Commodities"})       │
   └────────────────────────────────────────────────────────────┘
                              │
                              ▼
2. CALLER INVOKES ENGINE
   ┌────────────────────────────────────────────────────────────┐
   │ result := engine.Evaluate(&EvaluationRequest{              │
   │     Template:  template,     // has Expression, SchemaID   │
   │     Entity:    shipment,     // fully loaded struct        │
   │     Variables: userVars,     // baseRate, ratePerMile      │
   │ })                                                         │
   └────────────────────────────────────────────────────────────┘
                              │
                              ▼
3. ENGINE BUILDS ENVIRONMENT
   ┌────────────────────────────────────────────────────────────┐
   │ env := {                                                   │
   │     // From computed functions (reflection over entity)    │
   │     "totalDistance": 300.0,                                │
   │     "totalStops": 5,                                       │
   │     "hasHazmat": false,                                    │
   │                                                            │
   │     // From user-provided variables                        │
   │     "baseRate": 50.0,                                      │
   │     "ratePerMile": 1.5,                                    │
   │ }                                                          │
   └────────────────────────────────────────────────────────────┘
                              │
                              ▼
4. ENGINE EVALUATES EXPRESSION
   ┌────────────────────────────────────────────────────────────┐
   │ expression: "baseRate + (ratePerMile * totalDistance)"     │
   │                                                            │
   │ substituted: 50.0 + (1.5 * 300.0)                          │
   │                                                            │
   │ result: 500.0                                              │
   └────────────────────────────────────────────────────────────┘
                              │
                              ▼
5. CALLER RECEIVES RESULT
   ┌────────────────────────────────────────────────────────────┐
   │ EvaluationResult {                                         │
   │     Value: decimal.Decimal(500.0),                         │
   │     RawValue: 500.0,                                       │
   │     Variables: map[string]any{...},  // for debugging      │
   │ }                                                          │
   └────────────────────────────────────────────────────────────┘
```

## Security Considerations

### Expression Safety
- expr-lang is sandboxed, no arbitrary Go code execution
- No file/network access from expressions
- No loops (prevents infinite execution)
- Compilation validates syntax before execution

### Input Validation
- Expressions are validated against known variables
- Unknown variable references cause compilation errors
- Type mismatches caught at compile time

### No SQL Injection Risk
- Engine doesn't construct SQL queries
- All data access through reflection on loaded structs

## Performance Characteristics

### Compilation Caching
```go
// Expressions are compiled once, cached by string
cache sync.Map  // expression string → *vm.Program

// Subsequent evaluations skip compilation
if cached, ok := e.cache.Load(expression); ok {
    return cached.(*CompiledExpression), nil
}
```

### Reflection Cost
- One-time cost per evaluation (microseconds, not milliseconds)
- Acceptable for billing calculations (not called millions of times)
- Could optimize with code generation if needed later

### Memory
- Compiled programs cached indefinitely (call `ClearCache()` if needed)
- Environment maps created per evaluation, garbage collected

## Extension Points

### Adding Computed Functions
```go
resolver.RegisterComputed("computeCustomValue", func(entity any) (any, error) {
    // Your custom logic
    return value, nil
})
```

### Adding Transforms
```go
resolver.RegisterTransform("customTransform", func(v any) (any, error) {
    // Your transform logic
    return transformed, nil
})
```

### Adding Built-in Functions
```go
// In engine/functions.go
expr.Function("myFunc", myFuncImpl, new(func(float64) float64))
```

## Testing Strategy

### Unit Tests (no DB)
```go
// Test with plain structs
entity := &TestShipment{
    Weight: 5000,
    Moves:  []Move{{Distance: 100}, {Distance: 200}},
}
env, _ := builder.Build(entity, "")
assert.Equal(t, 300.0, env["totalDistance"])
```

### Integration Tests (no DB)
```go
// Test full evaluation flow
result, _ := engine.Evaluate(&EvaluationRequest{
    Template:  template,
    Entity:    shipment,
    Variables: map[string]any{"baseRate": 2.5},
})
assert.Equal(t, decimal.NewFromFloat(750.0), result.Value)
```

### With Database (optional)
```go
// Only if testing repository integration
// Formula engine tests don't need this
```

## Domain Model Compatibility

The formula system uses reflection and handles Go pointer types correctly. Here's how it maps to the actual domain model:

### Shipment Fields

```go
type Shipment struct {
    Weight         *int64                  // ✅ getFieldInt64 handles *int64
    Pieces         *int64                  // ✅ getFieldInt64 handles *int64
    TemperatureMin *int16                  // ✅ getFieldInt16 handles *int16
    TemperatureMax *int16                  // ✅ getFieldInt16 handles *int16
    Moves          []*ShipmentMove         // ✅ getFieldSlice handles pointer slices
    Commodities    []*ShipmentCommodity    // ✅ getFieldSlice handles pointer slices
    // ... other fields
}
```

### ShipmentMove Fields

```go
type ShipmentMove struct {
    Distance *float64   // ✅ getFieldFloat64 handles *float64
    Stops    []*Stop    // ✅ getFieldSlice handles pointer slices
}
```

### Required Related Entity Structure

For `computeHasHazmat` to work, the Commodity entity must have:

```go
type Commodity struct {
    HazardousMaterial *HazardousMaterial  // Can be nil if not hazmat
}

type HazardousMaterial struct {
    Class string  // e.g., "3", "8", etc.
}
```

### Helper Function Coverage

| Helper | Handles | Nil Behavior |
|--------|---------|--------------|
| `getFieldValue` | Pointer dereference, struct access | Returns `ErrNilPointer` |
| `getFieldSlice` | `[]T` and `[]*T` | Returns empty slice |
| `getFieldFloat64` | `float64`, `*float64`, `float32`, `int64`, `int` | Returns `0` for nil |
| `getFieldInt64` | `int64`, `*int64`, `int`, `*int` | Returns `ErrNilPointer` for nil |
| `getFieldInt16` | `int16`, `*int16` | Returns `ErrNilPointer` for nil |

### Adding New Computed Functions

When adding computed functions for your domain, follow this pattern:

```go
func computeCustomValue(entity any) (any, error) {
    // 1. Use helper functions - they handle pointers
    moves, err := getFieldSlice(entity, "Moves")
    if err != nil {
        return defaultValue, err
    }

    // 2. Iterate safely - each element may be *T
    for _, move := range moves {
        distance, err := getFieldFloat64(move, "Distance")
        if err != nil {
            continue  // Skip nil/invalid
        }
        // Use distance...
    }

    return result, nil
}
```

### Preloads Required for Computed Functions

| Computed Function | Required Preloads |
|-------------------|-------------------|
| `computeTotalDistance` | `Moves` |
| `computeTotalStops` | `Moves`, `Moves.Stops` |
| `computeHasHazmat` | `Commodities`, `Commodities.Commodity`, `Commodities.Commodity.HazardousMaterial` |
| `computeRequiresTemperatureControl` | None (uses Shipment fields) |
| `computeTemperatureDifferential` | None (uses Shipment fields) |
| `computeTotalWeight` | `Commodities` (fallback) |
| `computeTotalPieces` | `Commodities` (fallback) |
| `computeTotalLinearFeet` | `Commodities`, `Commodities.Commodity` |
| `computeFreightChargeAmount` | None (uses Shipment fields) |
| `computeOtherChargeAmount` | None (uses Shipment fields) |
| `computeCurrentTotalCharge` | None (uses Shipment fields) |

## Future Considerations

### Potential Enhancements
- **Expression versioning**: Track formula changes over time
- **Dry-run mode**: Show calculation breakdown step-by-step
- **Custom functions per tenant**: Allow users to define reusable functions
- **Expression builder UI**: Visual formula construction

### Not Planned
- **Database queries in expressions**: Violates pure computation principle
- **Async evaluation**: Expressions are fast, async adds complexity
- **Expression language change**: expr-lang is sufficient, switching has high cost
