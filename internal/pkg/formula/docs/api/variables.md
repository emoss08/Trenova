# Variables Package API

The variables package manages variable definitions and resolution for formula expressions.

## Core Types

### Variable

Represents a variable definition with metadata.

```go
type Variable struct {
    Name        string               // Unique variable name
    Description string               // Human-readable description
    Category    string               // Grouping category
    DataType    formula.ValueType    // Expected data type
    Entity      string               // Source entity type
    Tags        []string            // Additional metadata tags
}
```

### VariableContext

Interface for resolving variables during expression evaluation.

```go
type VariableContext interface {
    // Resolve a variable by name
    ResolveVariable(name string) (any, error)
    
    // Get all available field sources
    GetFieldSources() map[string]any
}
```

## Main Components

### VariableRegistry

Thread-safe registry for managing variable definitions.

```go
// Create new registry
registry := NewVariableRegistry()

// Register a variable
err := registry.Register(&Variable{
    Name:        "base_rate",
    Description: "Base shipping rate per unit",
    Category:    "Pricing",
    DataType:    formula.ValueTypeNumber,
    Entity:      "rate_table",
})

// Get a variable
variable, exists := registry.Get("base_rate")

// List all variables
allVars := registry.List()

// List by category
pricingVars := registry.ListByCategory("Pricing")

// List by entity
shipmentVars := registry.ListByEntity("shipment")

// Search variables
results := registry.Search("rate")
```

**Features:**
- Thread-safe operations
- Duplicate prevention
- Categorization support
- Search functionality

### DefaultVariableContext

Default implementation bridging variable resolution with schema data.

```go
// Create context
ctx := NewDefaultVariableContext(
    fieldSources,      // map[string]any with entity data
    schemaRegistry,    // Schema registry instance
    dataResolver,      // Data resolver instance
)

// Set user context (for permissions)
ctx.SetUserID("user-123")

// Resolve variable
value, err := ctx.ResolveVariable("shipment_weight")

// Clone for modification
cloned := ctx.Clone()
```

**Features:**
- Schema-based field extraction
- Computed field resolution
- Transform function support
- User context for permissions

## Built-in Variables

The package includes pre-registered variables for common use cases:

### Hazmat Variables

```go
// Available hazmat variables
variables := []string{
    "has_hazmat",               // Boolean: has hazardous materials
    "hazmat_class",            // String: primary hazmat class
    "hazmat_classes",          // Array: all hazmat classes
    "packing_group",           // String: hazmat packing group
    "un_number",               // String: UN identification number
    "is_bulk_hazmat",          // Boolean: bulk hazmat shipment
    "requires_placards",       // Boolean: requires hazmat placards
    "tunnel_restriction_code", // String: tunnel restriction
    "hazmat_phone",           // String: emergency phone number
}
```

### Temperature Variables

```go
// Available temperature variables
variables := []string{
    "min_temperature",            // Number: minimum required temp
    "max_temperature",            // Number: maximum required temp
    "temperature_range",          // Number: temperature range
    "temperature_differential",   // Number: computed differential
    "requires_temperature_control", // Boolean: needs temp control
    "is_frozen",                 // Boolean: requires freezing
    "is_refrigerated",           // Boolean: requires refrigeration
    "is_ambient",                // Boolean: ambient temperature
    "temperature_unit",          // String: temperature unit (F/C)
}
```

## Usage Examples

### Basic Variable Registration

```go
// Create registry
registry := NewVariableRegistry()

// Register custom variables
variables := []*Variable{
    {
        Name:        "customer_discount",
        Description: "Customer-specific discount percentage",
        Category:    "Pricing",
        DataType:    formula.ValueTypeNumber,
        Entity:      "customer",
    },
    {
        Name:        "is_expedited",
        Description: "Whether shipment is expedited",
        Category:    "Service",
        DataType:    formula.ValueTypeBoolean,
        Entity:      "shipment",
    },
    {
        Name:        "route_stops",
        Description: "List of stops on the route",
        Category:    "Routing",
        DataType:    formula.ValueTypeArray,
        Entity:      "route",
    },
}

for _, v := range variables {
    if err := registry.Register(v); err != nil {
        log.Printf("Failed to register %s: %v", v.Name, err)
    }
}
```

### Custom Variable Context

```go
// Implement custom variable context
type CustomVariableContext struct {
    shipment    *domain.Shipment
    customer    *domain.Customer
    marketData  *MarketData
}

func (c *CustomVariableContext) ResolveVariable(name string) (any, error) {
    switch name {
    // Shipment variables
    case "weight":
        return c.shipment.Weight, nil
    case "distance":
        return c.shipment.Distance, nil
    case "commodity_class":
        if len(c.shipment.Commodities) > 0 {
            return c.shipment.Commodities[0].ClassCode, nil
        }
        return "", nil
        
    // Customer variables
    case "customer_discount":
        return c.customer.DiscountRate, nil
    case "payment_terms":
        return c.customer.PaymentTerms, nil
        
    // Market variables
    case "fuel_price":
        return c.marketData.FuelPrice, nil
    case "market_rate":
        return c.marketData.BaseRate, nil
        
    default:
        return nil, fmt.Errorf("unknown variable: %s", name)
    }
}

func (c *CustomVariableContext) GetFieldSources() map[string]any {
    return map[string]any{
        "shipment":    c.shipment,
        "customer":    c.customer,
        "market_data": c.marketData,
    }
}
```

### Using with Schema Integration

```go
// Setup schema registry and data resolver
schemaRegistry := schema.NewSchemaRegistry()
dataResolver := schema.NewDefaultDataResolver()

// Register schemas and computers
// ... (see schema package docs)

// Create variable context with schema support
fieldSources := map[string]any{
    "shipment": shipment,
    "customer": customer,
}

varCtx := NewDefaultVariableContext(
    fieldSources,
    schemaRegistry,
    dataResolver,
)

// Variables are resolved through schema
value, err := varCtx.ResolveVariable("temperature_differential")
// Computed from shipment commodity temperatures
```

### Variable Discovery

```go
// List available variables for UI
func GetAvailableVariables(registry *VariableRegistry, entityType string) []VariableInfo {
    var result []VariableInfo
    
    // Get entity-specific variables
    entityVars := registry.ListByEntity(entityType)
    
    // Get common variables
    commonVars := registry.ListByCategory("Common")
    
    // Combine and format
    for _, v := range append(entityVars, commonVars...) {
        result = append(result, VariableInfo{
            Name:        v.Name,
            Description: v.Description,
            Type:        v.DataType.String(),
            Category:    v.Category,
            Example:     getExampleValue(v),
        })
    }
    
    return result
}

// Search variables by keyword
func SearchVariables(registry *VariableRegistry, query string) []*Variable {
    results := registry.Search(query)
    
    // Sort by relevance
    sort.Slice(results, func(i, j int) bool {
        // Exact match first
        if results[i].Name == query {
            return true
        }
        if results[j].Name == query {
            return false
        }
        
        // Then prefix match
        if strings.HasPrefix(results[i].Name, query) &&
           !strings.HasPrefix(results[j].Name, query) {
            return true
        }
        
        // Then alphabetical
        return results[i].Name < results[j].Name
    })
    
    return results
}
```

## Integration with Expression Package

Variables are automatically resolved during expression evaluation:

```go
// In expression evaluation
evaluator := expression.NewEvaluator()

// Create variable context
varCtx := &CustomVariableContext{
    shipment: shipment,
    customer: customer,
}

// Variables are resolved automatically
evalCtx := expression.NewEvaluationContext(ctx, varCtx)
result, err := evaluator.EvaluateWithContext(
    evalCtx,
    "base_rate * weight * customer_discount",
)
```

## Best Practices

1. **Use Descriptive Names**: Variable names should be self-documenting
2. **Categorize Appropriately**: Group related variables for better organization
3. **Document Variables**: Always provide clear descriptions
4. **Type Safety**: Specify correct data types for validation
5. **Avoid Naming Conflicts**: Use prefixes for entity-specific variables
6. **Cache Variable Lookups**: Implement caching in custom contexts
7. **Handle Missing Variables**: Provide sensible defaults or clear errors
8. **Version Variable Changes**: Track when variables are added/removed

## Thread Safety

- VariableRegistry: Thread-safe for all operations
- DefaultVariableContext: Thread-safe for reads, use Clone() for modifications
- Variable struct: Immutable after creation

## Performance Considerations

1. **Registry Lookups**: O(1) for Get operations
2. **Search Operations**: O(n) where n is number of variables
3. **Variable Resolution**: Can be optimized with caching
4. **Schema Integration**: May involve reflection, cache when possible