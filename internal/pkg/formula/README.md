<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Trenova Formula Package

The Formula package provides a powerful, extensible expression evaluation engine for the Trenova transportation management system. It enables dynamic calculations for pricing, routing, and business rules using a safe, sandboxed expression language with deep integration into the Trenova domain model.

## Table of Contents

- [Features](#features)
- [Architecture Overview](#architecture-overview)
- [Quick Start](#quick-start)
- [Integration with Trenova](#integration-with-trenova)
- [Expression Language](#expression-language)
- [Security & Performance](#security--performance)
- [Testing](#testing)
- [Contributing](#contributing)

## Features

- **Safe Expression Evaluation**: Sandboxed execution environment with configurable resource limits
- **Rich Function Library**: 25 built-in functions for math, type conversion, arrays, and conditionals
- **Type System**: Strong type support with automatic coercion where appropriate
- **Schema-Driven**: JSON Schema-based field definitions with automatic variable registration
- **Database Integration**: Seamless loading of entity data with optimized queries
- **Computed Fields**: Support for derived values calculated at runtime
- **Performance Optimized**: Expression caching, arena allocation, and parallel evaluation
- **Domain Integration**: Deep integration with Trenova's shipment and pricing models

## Architecture Overview

The formula package follows a modular architecture with clear separation of concerns:

```
formula/
├── expression/          # Core expression evaluation engine
│   ├── ast.go          # Abstract Syntax Tree definitions
│   ├── tokenizer.go    # Lexical analysis
│   ├── parser.go       # Syntax analysis and AST construction
│   ├── evaluator.go    # Expression compilation and evaluation
│   ├── functions.go    # Built-in function library
│   ├── operators.go    # Operator implementations
│   ├── arena.go        # Memory arena for efficient allocation
│   └── lru_cache.go    # Expression caching
├── schema/             # JSON Schema integration
│   ├── registry.go     # Schema management
│   ├── resolver.go     # Field resolution and transformation
│   ├── computers.go    # Computed field implementations
│   └── definitions/    # JSON schema files
├── variables/          # Variable system
│   ├── registry.go     # Variable registration and lookup
│   ├── context.go      # Runtime variable resolution
│   └── builtin/        # Pre-defined variables
├── infrastructure/     # External integrations
│   └── postgres_data_loader.go  # Database data loading
├── services/           # High-level services
│   └── formula_evaluation_service.go
└── module.go           # Dependency injection setup
```

## Quick Start

### Basic Usage

```go
// 1. Create a formula template
template := &formulatemplate.FormulaTemplate{
    Name:       "Standard Mileage Rate",
    Expression: "distance * base_rate * (1 + fuel_surcharge/100)",
    Variables: []formulatemplate.Variable{
        {Name: "distance", Type: "number", Required: true},
        {Name: "base_rate", Type: "number", Required: true},
        {Name: "fuel_surcharge", Type: "number", Required: true},
    },
}

// 2. Calculate a shipment rate
rate, err := formulaService.CalculateShipmentRate(
    ctx,
    templateID,
    shipment,
    userID,
)

// 3. Test a formula with sample data
result, err := formulaService.TestFormula(ctx, &formula.TestFormulaRequest{
    Expression: "weight * 0.05 + distance * 2.50",
    TestData: map[string]any{
        "weight":   1000,
        "distance": 450,
    },
})
```

### Integration with Shipment Pricing

```go
// In your shipment update logic
shipment.RatingMethod = shipment.RatingMethodFormulaTemplate
shipment.FormulaTemplateID = &templateID

// The ShipmentCalculator automatically uses the formula
updated, err := shipmentRepo.Update(ctx, shipment, userID)
// Rate is calculated using the assigned formula template
```

## Integration with Trenova

### Module Initialization

The formula package integrates seamlessly with Trenova's dependency injection:

```go
// In bootstrap/app.go
app := fx.New(
    // ... other modules
    formula.Module,
    services.Module,
    // ...
)
```

### Data Flow

1. **Schema Definition**: Fields are defined in JSON schemas with database mappings
2. **Variable Registration**: Schema fields automatically become available as variables
3. **Formula Creation**: Users create formula templates through the API
4. **Rate Calculation**: When a shipment uses formula-based rating:
   - Formula service loads the template
   - Data loader fetches required shipment data
   - Expression evaluator calculates the rate
   - Result is applied to the shipment

### Available Variables

Variables are automatically available from the shipment schema:

**Direct Fields**:

- `weight` - Total shipment weight
- `pieces` - Number of pieces
- `distance` - Total distance (from moves)
- `temperatureMin/Max` - Temperature requirements
- `freightChargeAmount` - Base freight charge

**Computed Fields**:

- `hasHazmat` - Whether shipment contains hazardous materials
- `temperatureDifferential` - Difference between min/max temperature
- `requiresTemperatureControl` - Whether temperature control is needed
- `totalStops` - Total number of stops across all moves
- `isExpedited` - Whether service type is expedited

**Nested Access**:

- `customer.name` - Customer information
- `tractorType.costPerMile` - Equipment type details
- `commodities[0].weight` - Individual commodity data

## Expression Language

### Basic Syntax

```javascript
// Arithmetic
base_rate * distance + fuel_surcharge

// Conditionals (ternary operator)
distance > 500 ? long_haul_rate : short_haul_rate

// Function calls
round(weight * 0.05, 2)

// Array operations
sum([1, 2, 3, 4, 5])
```

### Operators

- **Arithmetic**: `+`, `-`, `*`, `/`, `%`, `^` (power)
- **Comparison**: `==`, `!=`, `>`, `<`, `>=`, `<=`
- **Logical**: `&&`, `||`, `!`
- **Conditional**: `? :` (ternary)

### Built-in Functions

**Math Functions**:

- `abs(x)`, `min(...args)`, `max(...args)`
- `round(x, decimals)`, `floor(x)`, `ceil(x)`
- `sqrt(x)`, `pow(x, y)`, `log(x, base?)`, `exp(x)`

**Type Conversion**:

- `number(x)`, `string(x)`, `bool(x)`

**Array Functions**:

- `len(array)`, `sum(array)`, `avg(array)`
- `contains(array, value)`, `indexOf(array, value)`
- `slice(array, start, end)`

**Conditional**:

- `if(condition, trueVal, falseVal)`
- `coalesce(...values)` - returns first non-null value

### Complex Examples

```javascript
// Tiered pricing based on weight
if(weight <= 1000, 
   weight * 0.10,
   if(weight <= 5000,
      1000 * 0.10 + (weight - 1000) * 0.08,
      1000 * 0.10 + 4000 * 0.08 + (weight - 5000) * 0.06
   )
)

// Multi-factor pricing
base_rate * distance * 
  (1 + fuel_surcharge/100) *
  (hasHazmat ? 1.25 : 1.0) *
  (requiresTemperatureControl ? 1.15 : 1.0) *
  (isExpedited ? 1.50 : 1.0)

// Zone-based pricing with array lookup
[100, 150, 200, 250][min(floor(distance / 100), 3)] + weight * 0.05
```

## Security & Performance

### Security Features

- **Sandboxed Execution**: No system access or code execution
- **Resource Limits**:
  - Max expression length: 10,000 characters
  - Max evaluation depth: 50
  - Timeout: 100ms (configurable)
  - Memory limit: 1MB
- **Input Validation**: All inputs sanitized and validated
- **Permission Checks**: Formula operations require appropriate permissions

### Performance Optimizations

- **Expression Caching**: Parsed expressions are cached with LRU eviction
- **Arena Allocation**: Reduces garbage collection pressure
- **Parallel Evaluation**: Batch operations use worker pools
- **Selective Data Loading**: Only loads fields required by the formula
- **String Interning**: Common strings are reused

### Benchmarks

```bash
# Run performance benchmarks
go test -bench=. ./internal/pkg/formula/expression

# Example results:
BenchmarkSimpleExpression-8         1000000      1050 ns/op
BenchmarkComplexExpression-8         300000      4250 ns/op  
BenchmarkParallelEvaluation-8       2000000       750 ns/op
```

## Testing

The package includes comprehensive test coverage:

```bash
# Run all tests
go test ./internal/pkg/formula/...

# Run with coverage
go test -cover ./internal/pkg/formula/...

# Run integration tests
go test ./internal/pkg/formula -run Integration

# Run benchmarks
go test -bench=. ./internal/pkg/formula/expression
```

### Test Categories

- **Unit Tests**: Each component tested in isolation
- **Integration Tests**: End-to-end formula evaluation
- **Performance Tests**: Benchmarks for critical paths
- **Concurrent Tests**: Verify thread safety

## Contributing

When contributing to the formula package:

1. **Maintain Backward Compatibility**: Existing formulas must continue to work
2. **Add Tests**: New functions or features need comprehensive tests
3. **Update Documentation**: Keep docs in sync with code changes
4. **Follow Patterns**: Use existing error handling and type patterns
5. **Consider Performance**: Profile changes that might impact performance
6. **Schema Updates**: Coordinate schema changes with frontend team

### Adding a New Function

1. Add function implementation in `expression/functions.go`
2. Register in the function map
3. Add tests in `expression/functions_test.go`
4. Update function reference documentation
5. Add integration test showing usage

### Adding a Computed Field

1. Implement compute function in `schema/computers.go`
2. Register with the data resolver
3. Add field definition to relevant schema JSON
4. Update schema documentation
5. Add tests verifying the computation

## License

This package is part of the Trenova transportation management system and follows the project's licensing terms.
