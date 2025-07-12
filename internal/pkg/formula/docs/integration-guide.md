# Integration Guide

This guide demonstrates how to integrate the formula system into your Trenova application, including common use cases and best practices.

## Table of Contents

- [Getting Started](#getting-started)
- [Basic Integration](#basic-integration)
- [Working with Variables](#working-with-variables)
- [Schema Integration](#schema-integration)
- [Formula Templates](#formula-templates)
- [Real-World Examples](#real-world-examples)
- [Testing Formulas](#testing-formulas)
- [Performance Optimization](#performance-optimization)
- [Error Handling](#error-handling)
- [Security Considerations](#security-considerations)

## Getting Started

The formula system consists of three main components that work together:

1. **Variable Registry** - Defines available variables
2. **Schema Registry** - Validates and extracts data from entities
3. **Formula Service** - Evaluates expressions using variables and schemas

### Dependency Injection Setup

The formula module uses Uber fx for dependency injection:

```go
import (
    "github.com/emoss08/trenova/internal/pkg/formula"
    "go.uber.org/fx"
)

// In your application module
var AppModule = fx.Module("app",
    // Other dependencies...
    formula.Module,
    fx.Provide(
        NewShipmentCalculator,
    ),
)
```

## Basic Integration

### Simple Expression Evaluation

For basic expression evaluation without variables:

```go
import (
    "github.com/emoss08/trenova/internal/pkg/formula/expression"
)

func calculateSimple() {
    evaluator := expression.NewEvaluator()
    
    result, err := evaluator.Evaluate(
        context.Background(),
        "2 * 3 + 4",
        nil, // No variables
    )
    
    if err != nil {
        log.Printf("Evaluation error: %v", err)
        return
    }
    
    // result is 10.0
    fmt.Printf("Result: %v\n", result)
}
```

### With Variables

```go
func calculateWithVariables() {
    evaluator := expression.NewEvaluator()
    
    variables := map[string]any{
        "base_rate": 2.50,
        "distance": 450,
        "fuel_surcharge": 0.15,
    }
    
    result, err := evaluator.Evaluate(
        context.Background(),
        "base_rate * distance * (1 + fuel_surcharge)",
        variables,
    )
    
    if err != nil {
        log.Printf("Evaluation error: %v", err)
        return
    }
    
    // result is 1293.75
    fmt.Printf("Total cost: $%.2f\n", result)
}
```

## Working with Variables

### Defining Custom Variables

Create domain-specific variables for your business logic:

```go
import (
    "github.com/emoss08/trenova/internal/core/types/formula"
    "github.com/emoss08/trenova/internal/pkg/formula/variables"
)

func RegisterCustomVariables(registry *variables.VariableRegistry) {
    // Fuel price variable
    registry.Register(&variables.Variable{
        Name:        "fuel_price_per_gallon",
        Description: "Current fuel price per gallon",
        Category:    "Fuel",
        DataType:    formula.ValueTypeNumber,
        Entity:      "market_rates",
    })
    
    // Boolean flag
    registry.Register(&variables.Variable{
        Name:        "is_peak_season",
        Description: "Whether current date is in peak season",
        Category:    "Seasonal",
        DataType:    formula.ValueTypeBoolean,
        Entity:      "calendar",
    })
    
    // Array variable
    registry.Register(&variables.Variable{
        Name:        "surcharge_rates",
        Description: "List of applicable surcharge rates",
        Category:    "Pricing",
        DataType:    formula.ValueTypeArray,
        Entity:      "rate_tables",
    })
}
```

### Variable Context Implementation

Implement the `VariableContext` interface to resolve variables:

```go
type CustomVariableContext struct {
    shipment *domain.Shipment
    marketData *MarketData
    baseCtx variables.VariableContext
}

func (c *CustomVariableContext) ResolveVariable(name string) (any, error) {
    switch name {
    case "fuel_price_per_gallon":
        return c.marketData.FuelPrice, nil
    
    case "is_peak_season":
        now := time.Now()
        return now.Month() >= 6 && now.Month() <= 8, nil
    
    case "surcharge_rates":
        return c.marketData.SurchargeRates, nil
    
    default:
        // Delegate to base context for standard variables
        return c.baseCtx.ResolveVariable(name)
    }
}

func (c *CustomVariableContext) GetFieldSources() map[string]any {
    sources := c.baseCtx.GetFieldSources()
    sources["market_rates"] = c.marketData
    sources["calendar"] = map[string]any{
        "current_date": time.Now(),
    }
    return sources
}
```

## Schema Integration

### Registering Entity Schemas

Define JSON schemas for your domain entities:

```go
func RegisterEntitySchemas(registry *schema.SchemaRegistry) error {
    // Register shipment schema
    shipmentSchema := []byte(`{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "weight": {
                "type": "number",
                "minimum": 0
            },
            "distance": {
                "type": "number",
                "minimum": 0
            },
            "commodity_class": {
                "type": "string",
                "enum": ["50", "55", "60", "65", "70", "77.5", "85", "92.5", "100"]
            }
        }
    }`)
    
    if err := registry.RegisterSchema("shipment", shipmentSchema); err != nil {
        return err
    }
    
    return nil
}
```

### Adding Computed Fields

Register computed fields that derive values from entity data:

```go
func RegisterComputedFields(resolver *schema.DefaultDataResolver) {
    // Dimensional weight calculation
    resolver.RegisterComputer("dimensional_weight", func(ctx context.Context, entity any, variables map[string]any) (any, error) {
        shipment, ok := entity.(*domain.Shipment)
        if !ok {
            return nil, fmt.Errorf("entity is not a shipment")
        }
        
        if shipment.Length == 0 || shipment.Width == 0 || shipment.Height == 0 {
            return 0.0, nil
        }
        
        // Standard dimensional factor
        dimFactor := 166.0
        if intl, ok := variables["is_international"].(bool); ok && intl {
            dimFactor = 139.0
        }
        
        return (shipment.Length * shipment.Width * shipment.Height) / dimFactor, nil
    })
    
    // Chargeable weight (greater of actual or dimensional)
    resolver.RegisterComputer("chargeable_weight", func(ctx context.Context, entity any, variables map[string]any) (any, error) {
        shipment, ok := entity.(*domain.Shipment)
        if !ok {
            return nil, fmt.Errorf("entity is not a shipment")
        }
        
        dimWeight, err := resolver.Resolve(ctx, "dimensional_weight", entity, variables)
        if err != nil {
            return nil, err
        }
        
        return math.Max(shipment.Weight, dimWeight.(float64)), nil
    })
}
```

## Formula Templates

### Creating Formula Templates

Formula templates define reusable calculation logic:

```go
type FormulaTemplate struct {
    ID          string
    Name        string
    Description string
    Expression  string
    Variables   []string // Required variables
    Entity      string   // Required entity type
}

// Example templates
var (
    StandardRateTemplate = FormulaTemplate{
        ID:          "standard-rate",
        Name:        "Standard Shipping Rate",
        Description: "Basic weight and distance calculation",
        Expression:  "base_rate * chargeable_weight * distance_tier_multiplier",
        Variables:   []string{"base_rate"},
        Entity:      "shipment",
    }
    
    HazmatSurchargeTemplate = FormulaTemplate{
        ID:          "hazmat-surcharge",
        Name:        "Hazmat Surcharge",
        Description: "Additional charge for hazardous materials",
        Expression:  "if(has_hazmat, base_rate * 0.25 * weight, 0)",
        Variables:   []string{"base_rate"},
        Entity:      "shipment",
    }
    
    FuelSurchargeTemplate = FormulaTemplate{
        ID:          "fuel-surcharge",
        Name:        "Fuel Surcharge",
        Description: "Fuel surcharge based on current prices",
        Expression:  "distance * fuel_consumption_rate * (fuel_price_per_gallon - fuel_base_price) * 0.01",
        Variables:   []string{"fuel_price_per_gallon", "fuel_base_price", "fuel_consumption_rate"},
        Entity:      "shipment",
    }
)
```

### Using Formula Templates

```go
type ShipmentCalculator struct {
    formulaService *formula.Service
    templateRepo   ports.FormulaTemplateRepository
}

func (s *ShipmentCalculator) CalculateTotalRate(ctx context.Context, shipmentID, customerID string) (decimal.Decimal, error) {
    // Get customer's rate configuration
    customer, err := s.customerRepo.FindByID(ctx, customerID)
    if err != nil {
        return decimal.Zero, err
    }
    
    // Calculate base rate
    baseRate, err := s.formulaService.CalculateShipmentRate(ctx, &formula.CalculateRateRequest{
        FormulaTemplateID: customer.BaseRateTemplateID,
        ShipmentID:        shipmentID,
        Variables: map[string]any{
            "base_rate": customer.BaseRate,
        },
    })
    
    // Add surcharges
    var totalRate decimal.Decimal = baseRate
    
    // Hazmat surcharge
    if customer.HazmatSurchargeEnabled {
        hazmatCharge, err := s.formulaService.CalculateShipmentRate(ctx, &formula.CalculateRateRequest{
            FormulaTemplateID: HazmatSurchargeTemplate.ID,
            ShipmentID:        shipmentID,
            Variables: map[string]any{
                "base_rate": customer.BaseRate,
            },
        })
        totalRate = totalRate.Add(hazmatCharge)
    }
    
    // Fuel surcharge
    fuelCharge, err := s.formulaService.CalculateShipmentRate(ctx, &formula.CalculateRateRequest{
        FormulaTemplateID: FuelSurchargeTemplate.ID,
        ShipmentID:        shipmentID,
        Variables: map[string]any{
            "fuel_price_per_gallon":   s.marketData.FuelPrice,
            "fuel_base_price":         2.50,
            "fuel_consumption_rate":   0.15, // gallons per mile
        },
    })
    totalRate = totalRate.Add(fuelCharge)
    
    return totalRate, nil
}
```

## Real-World Examples

### Tiered Pricing System

```go
// Formula expression for tiered weight-based pricing
const tieredPricingFormula = `
// Define tier breakpoints and rates
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
`

variables := map[string]any{
    "weight":     750,
    "tier1_rate": 5.00,  // $5/lb for first 100 lbs
    "tier2_rate": 4.00,  // $4/lb for 101-500 lbs
    "tier3_rate": 3.00,  // $3/lb for 501-1000 lbs
    "tier4_rate": 2.50,  // $2.50/lb for 1000+ lbs
}
```

### Zone-Based Pricing

```go
// Define zones as arrays of zip code prefixes
variables := map[string]any{
    "origin_zip":      "10001",
    "destination_zip": "90210",
    "zone1_prefixes":  []any{"100", "101", "102"},
    "zone2_prefixes":  []any{"200", "201", "300"},
    "zone3_prefixes":  []any{"400", "500", "600"},
    "zone4_prefixes":  []any{"700", "800", "900"},
    "zone1_rate":      1.0,
    "zone2_rate":      1.5,
    "zone3_rate":      2.0,
    "zone4_rate":      2.5,
}

const zoneBasedFormula = `
// Extract destination prefix
dest_prefix = slice(destination_zip, 0, 3)

// Determine zone multiplier
zone_multiplier = if(contains(zone1_prefixes, dest_prefix), zone1_rate,
                    if(contains(zone2_prefixes, dest_prefix), zone2_rate,
                       if(contains(zone3_prefixes, dest_prefix), zone3_rate,
                          if(contains(zone4_prefixes, dest_prefix), zone4_rate,
                             3.0  // Default rate for unknown zones
                          )
                       )
                    )
                 )

// Calculate final rate
base_rate * distance * zone_multiplier
`
```

### Time-Based Pricing

```go
// Peak hours and weekend surcharges
const timeBasedFormula = `
// Check if delivery is during peak hours (8-10 AM, 4-6 PM)
is_peak_hour = (delivery_hour >= 8 && delivery_hour < 10) || 
               (delivery_hour >= 16 && delivery_hour < 18)

// Weekend surcharge
is_weekend = day_of_week == 0 || day_of_week == 6

// Calculate multipliers
time_multiplier = if(is_peak_hour, 1.25, 1.0)
weekend_multiplier = if(is_weekend, 1.15, 1.0)

// Apply to base rate
base_rate * distance * time_multiplier * weekend_multiplier
`
```

## Testing Formulas

### Unit Testing Formulas

```go
func TestTieredPricingFormula(t *testing.T) {
    service := setupTestFormulaService(t)
    
    tests := []struct {
        name     string
        weight   float64
        expected float64
    }{
        {"Under 100 lbs", 50, 250},      // 50 * 5
        {"Exactly 100 lbs", 100, 500},   // 100 * 5
        {"Between tiers", 250, 1100},    // 100*5 + 150*4
        {"Over 1000 lbs", 1200, 2900},   // 100*5 + 400*4 + 500*3 + 200*2.5
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := service.TestFormula(context.Background(), &formula.TestFormulaRequest{
                Expression: tieredPricingFormula,
                TestData: map[string]any{
                    "weight":     tt.weight,
                    "tier1_rate": 5.00,
                    "tier2_rate": 4.00,
                    "tier3_rate": 3.00,
                    "tier4_rate": 2.50,
                },
            })
            
            require.NoError(t, err)
            assert.InDelta(t, tt.expected, result.Result, 0.01)
        })
    }
}
```

### Integration Testing

```go
func TestShipmentRateCalculation(t *testing.T) {
    // Setup test database and services
    db := setupTestDB(t)
    service := setupFormulaService(t, db)
    
    // Create test shipment
    shipment := &domain.Shipment{
        ID:       "test-123",
        Weight:   500,
        Distance: 1000,
        Commodities: []domain.Commodity{
            {ClassCode: "85", IsHazmat: true},
        },
    }
    
    // Create formula template
    template := &domain.FormulaTemplate{
        ID:         "test-template",
        Expression: "base_rate * weight * distance * if(has_hazmat, 1.25, 1.0)",
    }
    
    // Calculate rate
    rate, err := service.CalculateShipmentRate(context.Background(), &formula.CalculateRateRequest{
        FormulaTemplateID: template.ID,
        ShipmentID:        shipment.ID,
        Variables: map[string]any{
            "base_rate": 0.002,
        },
    })
    
    require.NoError(t, err)
    expected := 0.002 * 500 * 1000 * 1.25
    assert.Equal(t, expected, rate.InexactFloat64())
}
```

## Performance Optimization

### Expression Caching

The formula service automatically caches compiled expressions:

```go
// Expressions are cached by default
evaluator := expression.NewEvaluator()

// First call compiles and caches
result1, _ := evaluator.Evaluate(ctx, complexFormula, vars1)

// Subsequent calls use cached AST
result2, _ := evaluator.Evaluate(ctx, complexFormula, vars2)
```

### Batch Evaluation

For bulk operations, use batch evaluation:

```go
func CalculateBulkRates(shipments []*domain.Shipment, formula string) ([]float64, error) {
    evaluator := expression.NewEvaluator()
    
    // Pre-compile the expression once
    compiled, err := evaluator.Compile(formula)
    if err != nil {
        return nil, err
    }
    
    results := make([]float64, len(shipments))
    errors := make([]error, len(shipments))
    
    // Use goroutines for parallel evaluation
    var wg sync.WaitGroup
    for i, shipment := range shipments {
        wg.Add(1)
        go func(idx int, s *domain.Shipment) {
            defer wg.Done()
            
            vars := map[string]any{
                "weight":   s.Weight,
                "distance": s.Distance,
            }
            
            result, err := evaluator.EvaluateCompiled(ctx, compiled, vars)
            if err != nil {
                errors[idx] = err
                return
            }
            
            results[idx] = result.(float64)
        }(i, shipment)
    }
    
    wg.Wait()
    
    // Check for errors
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }
    
    return results, nil
}
```

### Variable Resolution Optimization

Cache frequently accessed variables:

```go
type CachedVariableContext struct {
    base  variables.VariableContext
    cache map[string]any
    mu    sync.RWMutex
}

func (c *CachedVariableContext) ResolveVariable(name string) (any, error) {
    // Check cache first
    c.mu.RLock()
    if val, ok := c.cache[name]; ok {
        c.mu.RUnlock()
        return val, nil
    }
    c.mu.RUnlock()
    
    // Resolve and cache
    val, err := c.base.ResolveVariable(name)
    if err != nil {
        return nil, err
    }
    
    c.mu.Lock()
    c.cache[name] = val
    c.mu.Unlock()
    
    return val, nil
}
```

## Error Handling

### Graceful Error Handling

```go
func CalculateRateWithFallback(ctx context.Context, shipment *domain.Shipment) (decimal.Decimal, error) {
    // Try primary formula
    rate, err := calculateWithFormula(ctx, shipment, "primary_formula")
    if err == nil {
        return rate, nil
    }
    
    // Log error and try fallback
    log.Printf("Primary formula failed: %v", err)
    
    // Try simplified fallback formula
    rate, err = calculateWithFormula(ctx, shipment, "fallback_formula")
    if err == nil {
        return rate, nil
    }
    
    // Final fallback to fixed rate
    log.Printf("Fallback formula failed: %v", err)
    return calculateFixedRate(shipment), nil
}
```

### Validation Before Execution

```go
func ValidateFormula(expression string, requiredVars []string) error {
    evaluator := expression.NewEvaluator()
    
    // Parse to check syntax
    compiled, err := evaluator.Compile(expression)
    if err != nil {
        return fmt.Errorf("invalid syntax: %w", err)
    }
    
    // Extract variables used
    usedVars := compiled.ExtractVariables()
    
    // Check required variables are present
    for _, required := range requiredVars {
        found := false
        for _, used := range usedVars {
            if used == required {
                found = true
                break
            }
        }
        if !found {
            return fmt.Errorf("formula must use required variable: %s", required)
        }
    }
    
    // Test with sample data
    testVars := make(map[string]any)
    for _, v := range usedVars {
        testVars[v] = 1.0 // Dummy value
    }
    
    _, err = evaluator.EvaluateCompiled(context.Background(), compiled, testVars)
    if err != nil {
        return fmt.Errorf("formula execution error: %w", err)
    }
    
    return nil
}
```

## Security Considerations

### Input Sanitization

Always validate and sanitize user-provided formulas:

```go
func CreateUserFormula(ctx context.Context, userID string, formula string) error {
    // Check formula length
    if len(formula) > MaxFormulaLength {
        return errors.New("formula too long")
    }
    
    // Check complexity
    compiled, err := evaluator.Compile(formula)
    if err != nil {
        return err
    }
    
    if compiled.Complexity() > MaxComplexity {
        return errors.New("formula too complex")
    }
    
    // Restrict available functions for user formulas
    allowedFunctions := []string{"min", "max", "round", "if"}
    usedFunctions := extractFunctions(compiled)
    
    for _, fn := range usedFunctions {
        allowed := false
        for _, allowedFn := range allowedFunctions {
            if fn == allowedFn {
                allowed = true
                break
            }
        }
        if !allowed {
            return fmt.Errorf("function not allowed: %s", fn)
        }
    }
    
    // Store validated formula
    return storeFormula(ctx, userID, formula)
}
```

### Resource Limits

Set appropriate limits for formula execution:

```go
func ExecuteWithLimits(ctx context.Context, formula string, vars map[string]any) (any, error) {
    // Set timeout
    ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
    defer cancel()
    
    // Create limited context
    evalCtx := &expression.EvaluationContext{
        Context:     ctx,
        Variables:   vars,
        MemoryLimit: 1024 * 1024, // 1MB
    }
    
    // Execute with panic recovery
    var result any
    var err error
    
    done := make(chan bool)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("formula panic: %v", r)
            }
            done <- true
        }()
        
        result, err = evaluator.EvaluateWithContext(evalCtx, formula)
    }()
    
    select {
    case <-done:
        return result, err
    case <-ctx.Done():
        return nil, errors.New("formula execution timeout")
    }
}
```

## Best Practices

1. **Keep Formulas Simple**: Break complex calculations into multiple steps
2. **Use Descriptive Variable Names**: Make formulas self-documenting
3. **Validate Early**: Check formulas when created, not when executed
4. **Cache Aggressively**: Compiled expressions and computed values
5. **Handle Errors Gracefully**: Always have fallback calculations
6. **Test Thoroughly**: Include edge cases and error conditions
7. **Monitor Performance**: Track execution times and cache hit rates
8. **Document Formulas**: Explain business logic and assumptions
9. **Version Control**: Track formula changes over time
10. **Limit User Input**: Restrict functions and complexity for user-created formulas