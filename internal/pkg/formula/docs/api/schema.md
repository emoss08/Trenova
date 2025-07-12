# Schema Package API

The schema package provides JSON Schema validation and data extraction for formula expressions.

## Core Types

### SchemaRegistry

Thread-safe registry for managing JSON schemas.

```go
type SchemaRegistry struct {
    schemas map[string]*jsonschema.Schema
    mu      sync.RWMutex
}
```

### DataResolver

Interface for resolving data from entities.

```go
type DataResolver interface {
    // Resolve a field from an entity
    Resolve(ctx context.Context, fieldPath string, entity any, variables map[string]any) (any, error)
    
    // List available fields for an entity type
    ListFields(entityType string) []string
}
```

### Computer

Function type for computing derived values.

```go
type Computer func(ctx context.Context, entity any, variables map[string]any) (any, error)
```

### TransformFunc

Function type for transforming field values.

```go
type TransformFunc func(value any) (any, error)
```

## Main Components

### SchemaRegistry

Manages JSON schemas for entity validation.

```go
// Create registry
registry := NewSchemaRegistry()

// Register a schema
schemaJSON := []byte(`{
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
        }
    }
}`)

err := registry.RegisterSchema("shipment", schemaJSON)

// Validate entity against schema
err = registry.Validate("shipment", shipmentData)

// Get schema
schema, exists := registry.GetSchema("shipment")

// Extract available fields
fields := registry.ExtractFields("shipment")
// Returns: ["weight", "distance"]
```

### DefaultDataResolver

Default implementation of DataResolver with computed fields support.

```go
// Create resolver
resolver := NewDefaultDataResolver()

// Register a computer function
resolver.RegisterComputer("total_weight", func(ctx context.Context, entity any, variables map[string]any) (any, error) {
    shipment, ok := entity.(*domain.Shipment)
    if !ok {
        return nil, errors.New("entity is not a shipment")
    }
    
    total := shipment.Weight
    for _, item := range shipment.Items {
        total += item.Weight
    }
    
    return total, nil
})

// Register a transform function
resolver.RegisterTransform("uppercase", func(value any) (any, error) {
    str, ok := value.(string)
    if !ok {
        return nil, errors.New("value must be string")
    }
    return strings.ToUpper(str), nil
})

// Resolve field
value, err := resolver.Resolve(ctx, "total_weight", shipment, nil)

// Resolve with transform
value, err := resolver.Resolve(ctx, "customer_name|uppercase", customer, nil)
```

## Built-in Computers

The package includes pre-registered computer functions for shipments:

```go
// Register all shipment computers
RegisterShipmentComputers(resolver)

// Available computed fields:
// - has_hazmat: Boolean indicating hazmat presence
// - temperature_differential: Difference between min and max temps
// - requires_temperature_control: Boolean for temp control needs
// - total_stops: Count of all stops in shipment
```

## Usage Examples

### Schema Registration

```go
// Define schemas for your entities
schemas := map[string]string{
    "shipment": `{
        "type": "object",
        "properties": {
            "id": {"type": "string"},
            "weight": {"type": "number", "minimum": 0},
            "distance": {"type": "number", "minimum": 0},
            "pieces": {"type": "integer", "minimum": 1},
            "commodities": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "class": {"type": "string"},
                        "weight": {"type": "number"},
                        "hazmat": {"type": "boolean"}
                    }
                }
            }
        }
    }`,
    
    "customer": `{
        "type": "object", 
        "properties": {
            "id": {"type": "string"},
            "name": {"type": "string"},
            "discount_rate": {"type": "number", "minimum": 0, "maximum": 1},
            "credit_limit": {"type": "number", "minimum": 0}
        }
    }`,
}

// Register all schemas
registry := NewSchemaRegistry()
for entityType, schemaStr := range schemas {
    if err := registry.RegisterSchema(entityType, []byte(schemaStr)); err != nil {
        log.Fatalf("Failed to register %s schema: %v", entityType, err)
    }
}
```

### Custom Computer Functions

```go
// Create resolver with custom computers
resolver := NewDefaultDataResolver()

// Dimensional weight calculator
resolver.RegisterComputer("dimensional_weight", func(ctx context.Context, entity any, vars map[string]any) (any, error) {
    shipment, ok := entity.(*domain.Shipment)
    if !ok {
        return nil, errors.New("entity must be shipment")
    }
    
    if shipment.Length == 0 || shipment.Width == 0 || shipment.Height == 0 {
        return 0.0, nil
    }
    
    // Use dimensional factor from variables or default
    factor := 166.0
    if f, ok := vars["dim_factor"].(float64); ok {
        factor = f
    }
    
    return (shipment.Length * shipment.Width * shipment.Height) / factor, nil
})

// Chargeable weight (greater of actual or dimensional)
resolver.RegisterComputer("chargeable_weight", func(ctx context.Context, entity any, vars map[string]any) (any, error) {
    shipment, ok := entity.(*domain.Shipment)
    if !ok {
        return nil, errors.New("entity must be shipment")
    }
    
    // Get dimensional weight
    dimWeight, err := resolver.Resolve(ctx, "dimensional_weight", entity, vars)
    if err != nil {
        return shipment.Weight, nil // Fall back to actual weight
    }
    
    dimWeightFloat, _ := dimWeight.(float64)
    if shipment.Weight > dimWeightFloat {
        return shipment.Weight, nil
    }
    
    return dimWeightFloat, nil
})

// Service level computer
resolver.RegisterComputer("service_days", func(ctx context.Context, entity any, vars map[string]any) (any, error) {
    shipment, ok := entity.(*domain.Shipment)
    if !ok {
        return nil, errors.New("entity must be shipment")
    }
    
    // Base on distance and service type
    switch shipment.ServiceType {
    case "overnight":
        return 1, nil
    case "express":
        if shipment.Distance < 500 {
            return 2, nil
        }
        return 3, nil
    case "standard":
        if shipment.Distance < 200 {
            return 3, nil
        } else if shipment.Distance < 1000 {
            return 5, nil
        }
        return 7, nil
    default:
        return 5, nil
    }
})
```

### Transform Functions

```go
// Register useful transform functions
resolver := NewDefaultDataResolver()

// String transforms
resolver.RegisterTransform("uppercase", strings.ToUpper)
resolver.RegisterTransform("lowercase", strings.ToLower)
resolver.RegisterTransform("trim", strings.TrimSpace)

// Date formatting
resolver.RegisterTransform("date_format", func(value any) (any, error) {
    switch v := value.(type) {
    case time.Time:
        return v.Format("2006-01-02"), nil
    case string:
        // Parse and reformat
        t, err := time.Parse(time.RFC3339, v)
        if err != nil {
            return nil, err
        }
        return t.Format("2006-01-02"), nil
    default:
        return nil, fmt.Errorf("cannot format %T as date", value)
    }
})

// Numeric transforms
resolver.RegisterTransform("round", func(value any) (any, error) {
    f, ok := conversion.ToFloat64(value)
    if !ok {
        return nil, errors.New("value must be numeric")
    }
    return math.Round(f), nil
})

// Using transforms
value, err := resolver.Resolve(ctx, "customer_name|uppercase|trim", customer, nil)
```

### Field Path Resolution

```go
// Nested field resolution
resolver := NewDefaultDataResolver()

// Resolve nested fields using dot notation
value, err := resolver.Resolve(ctx, "shipment.origin.city", data, nil)

// Array index access
value, err := resolver.Resolve(ctx, "commodities.0.weight", shipment, nil)

// With transforms
value, err := resolver.Resolve(ctx, "customer.name|uppercase", data, nil)

// Computed fields
value, err := resolver.Resolve(ctx, "total_weight", shipment, nil)
```

### Integration with Variables

```go
// Create integrated context
fieldSources := map[string]any{
    "shipment": shipment,
    "customer": customer,
}

varCtx := variables.NewDefaultVariableContext(
    fieldSources,
    schemaRegistry,
    dataResolver,
)

// Variables can access schema fields
value, err := varCtx.ResolveVariable("shipment.weight")

// And computed fields
value, err := varCtx.ResolveVariable("chargeable_weight")

// And transformed values
value, err := varCtx.ResolveVariable("customer.name|uppercase")
```

## Schema Definition Best Practices

```json
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "title": "Shipment",
    "description": "Transportation shipment entity",
    "type": "object",
    "required": ["id", "weight", "origin", "destination"],
    "properties": {
        "id": {
            "type": "string",
            "description": "Unique shipment identifier"
        },
        "weight": {
            "type": "number",
            "description": "Total weight in pounds",
            "minimum": 0,
            "maximum": 80000
        },
        "dimensions": {
            "type": "object",
            "properties": {
                "length": {"type": "number", "minimum": 0},
                "width": {"type": "number", "minimum": 0},
                "height": {"type": "number", "minimum": 0}
            },
            "required": ["length", "width", "height"]
        },
        "commodities": {
            "type": "array",
            "description": "List of commodities in shipment",
            "items": {
                "$ref": "#/definitions/commodity"
            }
        }
    },
    "definitions": {
        "commodity": {
            "type": "object",
            "properties": {
                "class": {
                    "type": "string",
                    "enum": ["50", "55", "60", "65", "70", "77.5", "85", "92.5", "100"]
                },
                "nmfc": {"type": "string"},
                "hazmat": {"type": "boolean"},
                "description": {"type": "string"}
            }
        }
    }
}
```

## Performance Optimization

### Caching Computed Values

```go
type CachedResolver struct {
    base  DataResolver
    cache map[string]any
    mu    sync.RWMutex
}

func (r *CachedResolver) Resolve(ctx context.Context, path string, entity any, vars map[string]any) (any, error) {
    // Generate cache key
    key := fmt.Sprintf("%s:%v", path, entity)
    
    // Check cache
    r.mu.RLock()
    if val, ok := r.cache[key]; ok {
        r.mu.RUnlock()
        return val, nil
    }
    r.mu.RUnlock()
    
    // Compute value
    val, err := r.base.Resolve(ctx, path, entity, vars)
    if err != nil {
        return nil, err
    }
    
    // Cache result
    r.mu.Lock()
    r.cache[key] = val
    r.mu.Unlock()
    
    return val, nil
}
```

## Error Handling

Common errors and their handling:

```go
// Schema validation error
err := registry.Validate("shipment", data)
if err != nil {
    var validationErr *jsonschema.ValidationError
    if errors.As(err, &validationErr) {
        // Handle validation details
        for _, e := range validationErr.Causes {
            log.Printf("Validation error at %s: %s", e.Field, e.Description)
        }
    }
}

// Field resolution error
value, err := resolver.Resolve(ctx, "unknown.field", entity, nil)
if err != nil {
    if errors.Is(err, ErrFieldNotFound) {
        // Handle missing field
    }
}

// Computer function error
value, err := resolver.Resolve(ctx, "computed_field", entity, nil)
if err != nil {
    // Log and fall back
    log.Printf("Computer error: %v", err)
    value = defaultValue
}
```

## Thread Safety

- SchemaRegistry: Thread-safe for all operations
- DefaultDataResolver: Thread-safe for reads, use mutex for registration
- Computer functions: Should be thread-safe implementations
- Transform functions: Should be pure functions without side effects