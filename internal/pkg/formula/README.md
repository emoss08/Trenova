# Formula Package

The Formula package provides a powerful expression evaluation engine for the Trenova transportation management system. It enables dynamic calculations for pricing, routing, and business rules using a safe, sandboxed expression language.

## Features

- **Safe Expression Evaluation**: Sandboxed execution environment with memory and timeout limits
- **Rich Function Library**: Mathematical, string, array, and conditional functions
- **Type System**: Support for numbers, strings, booleans, and arrays
- **Variable Resolution**: Integration with domain entities and custom variables
- **Schema Validation**: JSON Schema-based validation for formula templates
- **Performance**: Expression compilation and caching for repeated evaluations

## Quick Start

```go
import (
    "context"
    "github.com/emoss08/trenova/internal/pkg/formula"
    "github.com/emoss08/trenova/internal/pkg/formula/expression"
)

// Create a formula service
service := formula.NewService(variableRegistry, schemaRegistry, dataResolver)

// Test a formula
result, err := service.TestFormula(context.Background(), &formula.TestFormulaRequest{
    Expression: "base_rate * distance * (1 + fuel_surcharge)",
    TestData: map[string]any{
        "base_rate": 2.50,
        "distance": 450,
        "fuel_surcharge": 0.15,
    },
})

// Calculate shipment rate
rate, err := service.CalculateShipmentRate(ctx, &formula.CalculateRateRequest{
    FormulaTemplateID: templateID,
    ShipmentID: shipmentID,
    Variables: map[string]any{
        "peak_season_multiplier": 1.25,
    },
})
```

## Package Structure

```
formula/
├── expression/      # Expression parsing and evaluation engine
├── schema/          # JSON Schema validation and field extraction  
├── variables/       # Variable definitions and registry
├── errors/          # Error types and handling
├── conversion/      # Type conversion utilities
└── docs/           # Detailed documentation
```

## Documentation

- [Expression Syntax Guide](docs/expression-syntax.md) - Learn the formula expression language
- [Function Reference](docs/function-reference.md) - Complete list of available functions
- [Integration Guide](docs/integration-guide.md) - How to integrate formulas in your application
- [API Documentation](docs/api/) - Package-level API documentation

## Expression Examples

### Basic Arithmetic
```javascript
// Simple calculation
base_rate * distance * fuel_surcharge

// With conditionals
if(distance > 500, base_rate * 0.9, base_rate) * distance

// Using functions
round(base_rate * distance * (1 + fuel_surcharge/100), 2)
```

### Array Operations
```javascript
// Array literals and indexing
prices[0] * quantity
[10, 20, 30][index]

// Array functions
sum(prices) / len(prices)  // Average
contains(hazmat_classes, "3")
slice(route_stops, 1, -1)  // All except first and last
```

### Advanced Calculations
```javascript
// Tiered pricing
if(weight <= 1000, 
   base_rate * weight,
   if(weight <= 5000,
      base_rate * 1000 + (weight - 1000) * base_rate * 0.8,
      base_rate * 1000 + 4000 * base_rate * 0.8 + (weight - 5000) * base_rate * 0.6
   )
)

// Distance-based with fuel surcharge
base_rate * distance * (1 + fuel_surcharge/100) * 
  if(has_hazmat, 1.25, 1.0) *
  if(requires_temperature_control, 1.15, 1.0)
```

## Security

The formula engine implements multiple security measures:

- **No Code Execution**: Pure expression evaluation, no arbitrary code execution
- **Resource Limits**: Configurable memory and CPU time limits
- **Complexity Bounds**: Maximum expression complexity to prevent DoS
- **Input Validation**: All inputs are validated and sanitized
- **Sandboxed Environment**: Expressions cannot access system resources

## Performance

- **Compiled Expressions**: Expressions are parsed once and cached
- **Optimized Evaluation**: Minimal allocations during evaluation
- **Concurrent Safe**: Thread-safe for concurrent evaluations
- **Batch Processing**: Efficient batch evaluation support

## Testing

The package includes comprehensive test coverage:

```bash
# Run all tests
go test ./internal/pkg/formula/...

# Run with coverage
go test -cover ./internal/pkg/formula/...

# Run benchmarks
go test -bench=. ./internal/pkg/formula/expression
```

## Contributing

When contributing to the formula package:

1. Maintain backward compatibility for expressions
2. Add tests for new functions or operators
3. Update documentation for any API changes
4. Follow the existing error handling patterns
5. Ensure thread safety for concurrent usage

## License

This package is part of the Trenova transportation management system and follows the project's licensing terms.