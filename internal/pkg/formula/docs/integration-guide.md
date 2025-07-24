<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Formula Integration Guide

This guide explains how to integrate the formula package into your Trenova application components and create custom formula-based features.

## Table of Contents

- [Overview](#overview)
- [Integration Patterns](#integration-patterns)
- [Using Formulas in Services](#using-formulas-in-services)
- [Creating Formula Templates](#creating-formula-templates)
- [Schema Integration](#schema-integration)
- [Adding Custom Variables](#adding-custom-variables)
- [Error Handling](#error-handling)
- [Testing Formula Integration](#testing-formula-integration)
- [Performance Optimization](#performance-optimization)
- [Security Considerations](#security-considerations)
- [Best Practices](#best-practices)

## Overview

The formula package provides three main integration points:

1. **Service Layer**: High-level APIs for formula evaluation
2. **Schema System**: Define available fields and computed values
3. **Variable Registry**: Register custom variables and functions

## Integration Patterns

### 1. Formula-Based Pricing

The most common use case is dynamic pricing for shipments:

```go
// In your shipment service
func (s *ShipmentService) CalculateRate(ctx context.Context, shipment *Shipment) (decimal.Decimal, error) {
    if shipment.RatingMethod == RatingMethodFormulaTemplate {
        return s.formulaService.CalculateShipmentRate(
            ctx,
            *shipment.FormulaTemplateID,
            shipment,
            userID,
        )
    }
    // Other rating methods...
}
```

### 2. Business Rule Evaluation

Use formulas for complex business rules:

```go
// Example: Determine if expedited shipping is required
template := &FormulaTemplate{
    Expression: "weight > 100 && hasHazmat && contains(['HIGH', 'URGENT'], customer.priority)",
}

result, err := formulaService.EvaluateBoolean(ctx, template, shipment)
if result {
    shipment.ServiceType = ServiceTypeExpedited
}
```

### 3. Dynamic Field Calculation

Calculate fields based on other values:

```go
// Example: Calculate insurance amount
template := &FormulaTemplate{
    Expression: "max(declaredValue * 0.01, 25)",
}

insurance, err := formulaService.Calculate(ctx, template, shipment)
shipment.InsuranceAmount = decimal.NewFromFloat(insurance)
```

## Using Formulas in Services

### Dependency Injection

First, inject the formula service into your service:

```go
type MyServiceParams struct {
    fx.In
    
    FormulaService *formula.Service
    Logger         *logger.Logger
}

type MyService struct {
    formulaService *formula.Service
    logger         *zerolog.Logger
}

func NewMyService(p MyServiceParams) *MyService {
    return &MyService{
        formulaService: p.FormulaService,
        logger:         p.Logger,
    }
}
```

### Testing Formulas

The formula service provides a test endpoint for validating expressions:

```go
func (s *MyService) ValidateFormula(ctx context.Context, expression string) error {
    result, err := s.formulaService.TestFormula(ctx, &formula.TestFormulaRequest{
        Expression: expression,
        TestData: map[string]any{
            "weight": 1000,
            "distance": 500,
            // ... other test values
        },
    })
    
    if err != nil {
        return fmt.Errorf("formula validation failed: %w", err)
    }
    
    if !result.Success {
        return fmt.Errorf("formula error: %s", result.Error)
    }
    
    return nil
}
```

### Batch Evaluation

For performance, evaluate formulas in batches:

```go
// Evaluate pricing for multiple shipments
shipments := []Shipment{...}
contexts := make([]variables.VariableContext, len(shipments))

for i, shipment := range shipments {
    contexts[i] = variables.NewDefaultContext(&shipment, resolver)
}

results, err := evaluator.EvaluateBatch(ctx, expression, contexts)
```

## Creating Formula Templates

### Template Structure

Formula templates define reusable expressions:

```go
template := &formulatemplate.FormulaTemplate{
    Name:        "Zone-Based Pricing",
    Description: "Calculates rate based on distance zones",
    Category:    formulatemplate.CategoryShipmentRating,
    Expression:  "base_rate * zone_rates[min(zone_index, len(zone_rates) - 1)] + weight * 0.05",
    
    Variables: []formulatemplate.Variable{
        {
            Name:        "base_rate",
            Type:        "number",
            Required:    true,
            Description: "Base rate per zone",
        },
        {
            Name:         "zone_multiplier",
            Type:         "array",
            Required:     true,
            DefaultValue: []float64{1.0, 1.2, 1.5, 2.0},
        },
    },
    
    Parameters: []formulatemplate.Parameter{
        {
            Name:         "fuel_surcharge",
            Type:         "number",
            DefaultValue: 15.0,
            Description:  "Current fuel surcharge percentage",
        },
    },
    
    MinRate: &minRate, // Optional constraints
    MaxRate: &maxRate,
}
```

### Template Categories

Organize templates by use case:

- `CategoryShipmentRating` - Pricing calculations
- `CategoryAccessorial` - Additional charges
- `CategoryCustom` - User-defined formulas

### Real-World Examples

#### Tiered Pricing System

```javascript
// Formula expression for tiered weight-based pricing
if(weight <= 100,
   weight * tier1_rate,
   if(weight <= 500,
      100 * tier1_rate + (weight - 100) * tier2_rate,
      if(weight <= 1000,
         100 * tier1_rate + 400 * tier2_rate + (weight - 500) * tier3_rate,
         100 * tier1_rate + 400 * tier2_rate + 500 * tier3_rate + (weight - 1000) * tier4_rate
      )
   )
)
```

#### Zone-Based Pricing

```javascript
// Apply zone-based pricing based on destination prefix
// Extract first 3 digits of zip and apply appropriate multiplier
base_rate * distance * 
  if(contains(zone1_prefixes, slice(destination_zip, 0, 3)), zone1_rate,
     if(contains(zone2_prefixes, slice(destination_zip, 0, 3)), zone2_rate,
        if(contains(zone3_prefixes, slice(destination_zip, 0, 3)), zone3_rate,
           if(contains(zone4_prefixes, slice(destination_zip, 0, 3)), zone4_rate,
              3.0  // Default rate for unknown zones
           )
        )
     )
  )
```

#### Time-Based Pricing

```javascript
// Apply time-based pricing with peak hours and weekend surcharges
base_rate * distance * 
  // Peak hour multiplier (8-10am or 4-6pm)
  if((delivery_hour >= 8 && delivery_hour < 10) || 
     (delivery_hour >= 16 && delivery_hour < 18), 1.25, 1.0) *
  // Weekend multiplier (Sunday=0 or Saturday=6)
  if(day_of_week == 0 || day_of_week == 6, 1.15, 1.0)
```

## Schema Integration

### Defining a New Schema

Create a JSON schema file in `schema/definitions/`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://trenova.com/schemas/formula/myentity.schema.json",
  "title": "MyEntity",
  "type": "object",
  
  "x-formula-context": {
    "category": "myentity",
    "entities": ["MyEntity"],
    "permissions": ["formula:read:myentity"]
  },
  
  "x-data-source": {
    "table": "my_entities",
    "entity": "github.com/emoss08/trenova/internal/core/domain/myentity.MyEntity",
    "preload": ["RelatedEntity"]
  },
  
  "properties": {
    "customField": {
      "description": "A custom field for calculations",
      "type": "number",
      "x-source": {
        "field": "custom_field",
        "path": "CustomField",
        "transform": "decimalToFloat64"
      }
    },
    
    "computedValue": {
      "description": "A computed value",
      "type": "number",
      "x-source": {
        "computed": true,
        "function": "computeMyValue",
        "requires": ["customField", "relatedField"]
      }
    }
  }
}
```

### Registering the Schema

Add schema registration in the module:

```go
func newSchemaRegistryWithSchemas(p SchemaRegistryParams) (*schema.SchemaRegistry, error) {
    registry := schema.NewSchemaRegistry()
    
    // Load your schema
    schemaJSON, err := schemaDefinitions.ReadFile("schema/definitions/myentity.json")
    if err != nil {
        return nil, err
    }
    
    if err := registry.RegisterSchema("myentity", schemaJSON); err != nil {
        return nil, err
    }
    
    return registry, nil
}
```

### Adding Computed Fields

Implement compute functions in `schema/computers.go`:

```go
func computeMyValue(entity any) (any, error) {
    e, ok := entity.(*myentity.MyEntity)
    if !ok {
        return nil, fmt.Errorf("expected MyEntity, got %T", entity)
    }
    
    // Perform calculation
    // Perform calculation
    result := e.CustomField * 1.5
    if e.RelatedEntity != nil {
        result = result + e.RelatedEntity.Modifier
    }
    
    return result, nil
}

// Register in the resolver
func RegisterMyEntityComputers(resolver *DefaultDataResolver) {
    resolver.RegisterComputer("computeMyValue", computeMyValue)
}
```

## Adding Custom Variables

### Creating a Variable

Define custom variables for domain-specific calculations:

```go
// In variables/custom/myvariable.go
package custom

import (
    "github.com/emoss08/trenova/internal/pkg/formula/variables"
    "github.com/emoss08/trenova/internal/core/types/formula"
)

func NewFuelSurchargeVariable() variables.Variable {
    return variables.NewVariable(
        "current_fuel_surcharge",
        "Current fuel surcharge percentage",
        formula.ValueTypeNumber,
        variables.VariableSourceCustom,
        func(ctx variables.VariableContext) (any, error) {
            // Get from configuration or external service
            return getFuelSurchargeFromConfig(), nil
        },
    )
}
```

### Registering Variables

Register during module initialization:

```go
func RegisterCustomVariables(registry *variables.Registry) {
    registry.Register(custom.NewFuelSurchargeVariable())
    registry.Register(custom.NewPeakSeasonVariable())
    // ... more variables
}
```

## Error Handling

### Formula Errors

Handle different types of formula errors:

```go
result, err := formulaService.CalculateShipmentRate(ctx, templateID, shipment, userID)
if err != nil {
    switch {
    case errors.Is(err, formula.ErrInvalidExpression):
        // Handle syntax errors
        return nil, fmt.Errorf("invalid formula syntax: %w", err)
        
    case errors.Is(err, formula.ErrVariableNotFound):
        // Handle missing variables
        return nil, fmt.Errorf("formula references unknown variable: %w", err)
        
    case errors.Is(err, formula.ErrTimeout):
        // Handle evaluation timeout
        return nil, fmt.Errorf("formula evaluation timed out: %w", err)
        
    default:
        // Handle other errors
        return nil, fmt.Errorf("formula evaluation failed: %w", err)
    }
}
```

### Validation

Validate formulas before saving:

```go
func (s *FormulaService) ValidateTemplate(template *FormulaTemplate) error {
    // Test with sample data
    testResult, err := s.TestFormula(ctx, &TestFormulaRequest{
        Expression: template.Expression,
        TestData:   generateSampleData(template),
    })
    
    if err != nil {
        return fmt.Errorf("formula validation failed: %w", err)
    }
    
    if !testResult.Success {
        return fmt.Errorf("formula error: %s", testResult.Error)
    }
    
    // Verify all required variables are used
    for _, variable := range template.Variables {
        if variable.Required && !contains(testResult.UsedVariables, variable.Name) {
            return fmt.Errorf("required variable '%s' is not used in formula", variable.Name)
        }
    }
    
    return nil
}
```

## Testing Formula Integration

### Unit Tests

Test formula integration in your services:

```go
func TestShipmentRateCalculation(t *testing.T) {
    // Setup
    formulaService := setupMockFormulaService()
    shipmentService := NewShipmentService(formulaService)
    
    // Create test shipment
    shipment := &Shipment{
        Weight:            1000,
        Distance:          500,
        RatingMethod:      RatingMethodFormulaTemplate,
        FormulaTemplateID: &testTemplateID,
    }
    
    // Test calculation
    rate, err := shipmentService.CalculateRate(ctx, shipment)
    require.NoError(t, err)
    assert.Equal(t, "1250.00", rate.String()) // Expected: 1000 * 0.05 + 500 * 2.00
}
```

### Integration Tests

Test end-to-end formula evaluation:

```go
func TestFormulaIntegration(t *testing.T) {
    // Setup complete formula system
    schemaRegistry := schema.NewSchemaRegistry()
    varRegistry := variables.NewRegistry()
    resolver := schema.NewDefaultDataResolver()
    
    // Register schemas and variables
    registerTestSchemas(schemaRegistry)
    registerTestVariables(varRegistry)
    
    // Create service
    dataLoader := infrastructure.NewPostgresDataLoader(db, schemaRegistry)
    evalService := services.NewFormulaEvaluationService(
        dataLoader, schemaRegistry, varRegistry, resolver,
    )
    
    // Test evaluation
    result, err := evalService.EvaluateFormula(
        ctx,
        "weight * 0.05 + distance * 2.00",
        "shipment",
        shipmentID,
    )
    
    require.NoError(t, err)
    assert.Equal(t, 1250.0, result)
}
```

### Performance Tests

Benchmark formula evaluation:

```go
func BenchmarkFormulaEvaluation(b *testing.B) {
    // Setup
    service := setupFormulaService()
    contexts := generateTestContexts(1000)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := service.EvaluateBatch(
            context.Background(),
            "weight * 0.05 + distance * 2.00",
            contexts,
        )
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Performance Optimization

### Expression Caching

The evaluator automatically caches compiled expressions:

```go
// First evaluation compiles and caches the expression
result1, _ := evaluator.Evaluate(ctx, complexFormula, vars1)

// Subsequent evaluations use the cached AST
result2, _ := evaluator.Evaluate(ctx, complexFormula, vars2)
```

### Batch Processing

Process multiple evaluations efficiently:

```go
// Pre-compile expression once
compiled, err := evaluator.Compile(expression)
if err != nil {
    return nil, err
}

// Evaluate for multiple contexts in parallel
results := make([]float64, len(contexts))
var wg sync.WaitGroup

for i, ctx := range contexts {
    wg.Add(1)
    go func(idx int, varCtx variables.VariableContext) {
        defer wg.Done()
        result, _ := evaluator.EvaluateCompiled(ctx, compiled, varCtx)
        results[idx] = result.(float64)
    }(i, ctx)
}

wg.Wait()
```

### Arena Allocation

The evaluator uses arena allocation to reduce GC pressure:

- Memory is allocated in blocks
- Objects are pooled for reuse
- Strings are interned
- Minimal allocations during evaluation

## Security Considerations

### Input Validation

Always validate user-provided formulas:

```go
func ValidateUserFormula(formula string) error {
    // Check length
    if len(formula) > MaxFormulaLength {
        return errors.New("formula too long")
    }
    
    // Parse and check complexity
    compiled, err := evaluator.Compile(formula)
    if err != nil {
        return fmt.Errorf("invalid syntax: %w", err)
    }
    
    if compiled.Complexity() > MaxComplexity {
        return errors.New("formula too complex")
    }
    
    // Restrict functions for user formulas
    allowedFunctions := []string{"min", "max", "round", "if", "sum", "avg"}
    for _, fn := range compiled.Functions() {
        if !contains(allowedFunctions, fn) {
            return fmt.Errorf("function not allowed: %s", fn)
        }
    }
    
    return nil
}
```

### Resource Limits

The evaluator enforces resource limits:

- **Timeout**: 100ms default (configurable)
- **Memory**: 1MB limit per evaluation
- **Depth**: Maximum nesting depth of 50
- **Iterations**: Maximum 10,000 evaluations

### Safe Execution

Formulas execute in a sandboxed environment:

- No system access
- No file operations
- No network calls
- No code execution
- Pure mathematical operations only

## Best Practices

1. **Cache Formula Templates**: Load templates once and reuse
2. **Validate Early**: Check formulas when created, not during evaluation
3. **Use Descriptive Names**: Make variable and template names self-documenting
4. **Handle Errors Gracefully**: Always have fallback calculations
5. **Test Edge Cases**: Include zero, negative, and extreme values in tests
6. **Monitor Performance**: Track evaluation times and optimize slow formulas
7. **Version Templates**: Track changes to formula logic over time
8. **Document Business Logic**: Explain why formulas work the way they do
9. **Limit User Input**: Restrict complexity and functions for user formulas
10. **Use Batch Operations**: Process multiple evaluations together when possible

## Common Pitfalls

### Division by Zero

Always guard against division by zero:

```javascript
// Bad: Risk of division by zero
distance / time

// Good: Safe with fallback
time > 0 ? distance / time : 0
```

### Null Values

Handle missing data gracefully:

```javascript
// Use coalesce to provide defaults
coalesce(weight, pieces * 10, 100)
```

### Type Mismatches

Be explicit about type conversions:

```javascript
// Ensure numeric comparison
if(number(status_code) == 200, "success", "failure")
```

### Performance Issues

Break complex formulas into steps:

```javascript
// Instead of one complex formula, break it down:
// Option 1: Create separate formula templates for each component
// Option 2: Use nested expressions with clear structure
(weight * rate_per_pound) +     // Base charge
(distance * fuel_rate)          // Fuel charge
```