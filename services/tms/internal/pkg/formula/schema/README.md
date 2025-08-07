<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Formula Schema System

This directory contains the schema system for formula contexts, which provides a flexible way to define data structures and their sources for use in formula calculations.

## Overview

The schema system uses JSON Schema with custom extensions to define:

- Data structure and validation rules
- How to fetch data from the database
- How to extract and transform field values
- Computed fields and their dependencies

## Schema Structure

### Basic Schema

Every schema must include:

- `$schema`: Reference to JSON Schema version
- `$id`: Unique identifier for the schema
- `title`: Human-readable title
- `type`: Root type (usually "object")

### Custom Extensions

#### x-formula-context

Defines metadata about the formula context:

```json
"x-formula-context": {
  "category": "shipment",           // Category for grouping
  "entities": ["Shipment"],         // Which domain entities this applies to
  "permissions": ["formula:read:shipment"], // Required permissions
  "tags": ["pricing", "optional"]   // Optional searchable tags
}
```

#### x-data-source

Defines how to fetch data from the database:

```json
"x-data-source": {
  "table": "shipments",            // Database table
  "entity": "domain.Shipment",     // Go type for reflection
  "preload": ["Customer"],         // Relations to preload
  "joins": {...},                  // Optional: Complex joins
  "filters": [...],                // Optional: Default filters
  "orderBy": "created_at DESC"     // Optional: Default ordering
}
```

#### x-source (Field Level)

Defines how to extract field values:

```json
"x-source": {
  "field": "pro_number",           // Database column
  "path": "ProNumber",             // Go struct field path
  "nullable": true,                // Can be null
  "transform": "int64ToFloat64",   // Transform function
  "computed": true,                // Computed field
  "function": "computeTotal",      // Compute function
  "requires": ["field1", "field2"] // Dependencies
}
```

## Field Types

### Simple Fields

```json
"proNumber": {
  "type": "string",
  "description": "PRO number",
  "x-source": {
    "field": "pro_number",
    "path": "ProNumber"
  }
}
```

### Nullable Fields

```json
"weight": {
  "type": ["number", "null"],
  "x-source": {
    "path": "Weight",
    "nullable": true,
    "transform": "int64ToFloat64"
  }
}
```

### Computed Fields

```json
"hasHazmat": {
  "type": "boolean",
  "x-source": {
    "computed": true,
    "function": "computeHasHazmat",
    "requires": ["commodities"],
    "preload": ["Commodities.Commodity.HazardousMaterial"]
  }
}
```

### Nested Objects

```json
"customer": {
  "type": "object",
  "x-source": {
    "relation": "Customer",
    "preload": true
  },
  "properties": {
    "name": {
      "type": "string",
      "x-source": {
        "path": "Customer.Name"
      }
    }
  }
}
```

### Arrays

```json
"commodities": {
  "type": "array",
  "x-source": {
    "relation": "Commodities",
    "preload": true
  },
  "items": {
    "type": "object",
    "properties": {
      "weight": {
        "type": "integer",
        "x-source": {
          "path": "Weight"
        }
      }
    }
  }
}
```

## Available Transform Functions

- `decimalToFloat64` - Convert decimal.Decimal to float64
- `int64ToFloat64` - Convert *int64 to float64  
- `int16ToFloat64` - Convert *int16 to float64
- `stringToUpper` - Convert to uppercase
- `stringToLower` - Convert to lowercase
- `unixToISO8601` - Convert unix timestamp to ISO8601

## Creating a New Schema

1. Copy `template.schema.json` as a starting point
2. Update the `$id`, `title`, and `description`
3. Define your `x-formula-context` metadata
4. Configure `x-data-source` for database access
5. Define properties with appropriate `x-source` configurations
6. Register computed field functions if needed
7. Test with sample data

## Example Usage

```go
// Register a schema
registry := schema.NewSchemaRegistry()
schemaJSON, _ := os.ReadFile("definitions/shipment.json")
err := registry.RegisterSchema("shipment", schemaJSON)

// Use with resolver
resolver := schema.NewDefaultDataResolver()
schema.RegisterShipmentComputers(resolver)

// Resolve field values
fieldSource := &schema.FieldSource{
    Path: "Customer.Name",
}
value, err := resolver.ResolveField(shipment, fieldSource)
```

## Best Practices

1. Keep schemas focused on a single entity type
2. Use meaningful field names that match the domain
3. Document all fields with descriptions
4. Specify nullable fields explicitly
5. Use appropriate transforms for type conversions
6. Define computed fields for derived values
7. Preload only necessary relations
8. Use filters to limit data when appropriate
