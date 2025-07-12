# Formula Package API Documentation

This directory contains detailed API documentation for each package in the formula system.

## Package Overview

| Package | Description | Documentation |
|---------|-------------|---------------|
| [expression](expression.md) | Core expression parsing and evaluation engine | Tokenizer, Parser, Evaluator, AST nodes |
| [variables](variables.md) | Variable management and resolution | Registry, contexts, built-in variables |
| [schema](schema.md) | JSON Schema validation and data extraction | Registry, resolvers, computed fields |
| [errors](errors.md) | Specialized error types with context | Error types, handling patterns |
| [conversion](conversion.md) | Type conversion utilities | Safe conversions, type coercion |

## Quick Links

### Core Components

- [Evaluator API](expression.md#evaluator) - Main expression evaluation interface
- [Variable Registry](variables.md#variableregistry) - Variable definition management
- [Schema Registry](schema.md#schemaregistry) - JSON Schema management
- [Error Types](errors.md#error-types) - Formula-specific error types

### Common Tasks

- [Evaluating Expressions](expression.md#usage-examples)
- [Registering Variables](variables.md#basic-variable-registration)
- [Adding Computed Fields](schema.md#custom-computer-functions)
- [Type Conversions](conversion.md#usage-patterns)
- [Error Handling](errors.md#error-handling-patterns)

## Architecture Overview

```
┌────────────────────────────────────────────────────────────┐
│                      Formula Service                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                 Expression Package                  │   │
│  │  ┌──────────┐  ┌────────┐  ┌───────────┐            │   │
│  │  │Tokenizer │→ │ Parser │→ │ Evaluator │            │   │
│  │  └──────────┘  └────────┘  └───────────┘            │   │
│  │                      ↓                              │   │
│  │               ┌─────────────┐                       │   │
│  │               │   AST Nodes │                       │   │
│  │               └─────────────┘                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                           ↓                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Variables Package                      │   │
│  │  ┌──────────┐        ┌─────────────────┐            │   │
│  │  │ Registry │ ←───── │ Variable Context│            │   │
│  │  └──────────┘        └─────────────────┘            │   │
│  └─────────────────────────────────────────────────────┘   │
│                           ↓                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                Schema Package                       │   │
│  │  ┌──────────┐    ┌──────────────┐    ┌─────────┐    │   │
│  │  │ Registry │    │Data Resolver │    │Computers│    │   │
│  │  └──────────┘    └──────────────┘    └─────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────┘
```

## Integration Flow

1. **Expression Input** → Tokenizer breaks into tokens
2. **Token Stream** → Parser builds Abstract Syntax Tree
3. **AST + Variables** → Evaluator computes result
4. **Variable Resolution** → Registry provides definitions
5. **Schema Validation** → Ensures data integrity
6. **Error Handling** → Rich context for debugging

## Common Integration Patterns

### Basic Setup

```go
import (
    "github.com/emoss08/trenova/internal/pkg/formula"
    "github.com/emoss08/trenova/internal/pkg/formula/expression"
    "github.com/emoss08/trenova/internal/pkg/formula/variables"
    "github.com/emoss08/trenova/internal/pkg/formula/schema"
)

// Initialize components
varRegistry := variables.NewVariableRegistry()
schemaRegistry := schema.NewSchemaRegistry()
dataResolver := schema.NewDefaultDataResolver()

// Register built-ins
variables.RegisterBuiltinVariables(varRegistry)
schema.RegisterShipmentComputers(dataResolver)

// Create service
service := formula.NewService(varRegistry, schemaRegistry, dataResolver)
```

### Custom Extensions

```go
// Add custom variable
varRegistry.Register(&variables.Variable{
    Name:        "peak_season_multiplier",
    Description: "Rate multiplier during peak season",
    Category:    "Seasonal",
    DataType:    formula.ValueTypeNumber,
})

// Add custom computer
dataResolver.RegisterComputer("days_in_transit", 
    func(ctx context.Context, entity any, vars map[string]any) (any, error) {
        // Custom computation logic
        return calculateTransitDays(entity), nil
    },
)

// Add custom function (requires modifying function registry)
customFunc := &myCustomFunction{}
expression.DefaultFunctionRegistry()["myFunc"] = customFunc
```

## Performance Tips

1. **Cache Evaluators**: Reuse evaluator instances
2. **Pre-compile Expressions**: Use `Compile()` for repeated use
3. **Batch Operations**: Use `EvaluateBatch()` for bulk processing
4. **Variable Caching**: Implement caching in custom contexts
5. **Schema Validation**: Validate once, evaluate many times

## Security Considerations

1. **Expression Limits**: Maximum 10,000 characters, 1,000 tokens
2. **Complexity Bounds**: Maximum complexity of 1,000
3. **Memory Limits**: Configurable per evaluation context
4. **Timeout Protection**: Use context with deadline
5. **No Code Execution**: Pure expression evaluation only

## Debugging

### Enable Debug Logging

```go
tokenizer := expression.NewTokenizer(expr)
tokenizer.EnableDebug()
```

### Trace Evaluation

```go
tracer := expression.NewTracingEvaluator(evaluator)
result, trace, err := tracer.EvaluateWithTrace(ctx, expr, vars)
// Examine trace for step-by-step evaluation
```

### Error Context

```go
if err != nil {
    if evalErr, ok := err.(*errors.EvaluationError); ok {
        log.Printf("Error in expression: %s", evalErr.Expression)
        log.Printf("Variable: %s", evalErr.Variable)
        for k, v := range evalErr.Context {
            log.Printf("  %s: %v", k, v)
        }
    }
}
```

## Version Compatibility

The formula API follows semantic versioning:

- **Major**: Breaking changes to expression syntax or API
- **Minor**: New functions, operators, or backwards-compatible features
- **Patch**: Bug fixes and performance improvements

## Support

For questions or issues:

1. Check the [Integration Guide](../integration-guide.md)
2. Review [Expression Syntax](../expression-syntax.md)
3. See [Function Reference](../function-reference.md)
4. File issues in the project repository
