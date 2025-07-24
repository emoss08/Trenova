<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Trenova Code Generator

A powerful type-safe code generator for Trenova domain entities that eliminates boilerplate, prevents runtime errors, and provides a fluent query API.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Usage Examples](#usage-examples)
- [Field Configuration](#field-configuration)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The Trenova code generator automatically analyzes your Bun ORM domain entities and generates comprehensive type-safe query helpers. It uses Go's AST parser to extract type information and generate code that matches your exact schema.

### Key Benefits

1. **Zero Runtime Errors**: All field names and types are validated at compile time
2. **Full Type Safety**: Uses actual Go types (`pulid.ID`, `domain.Status`, `bool`, `map[string]any`) instead of `any`
3. **IDE Intelligence**: Complete autocomplete for all fields and query methods
4. **Chainable API**: Fluent interface inspired by Ent for building complex queries
5. **Automatic Detection**: Scans your codebase and generates for all entities automatically
6. **Smart Imports**: Automatically adds required imports using goimports

## Features

### 🎯 Type-Safe Query Building

- Strongly typed WHERE predicates (EQ, NEQ, GT, LT, Contains, etc.)
- Null checks for pointer types
- String operations (Contains, HasPrefix, HasSuffix)
- Numeric comparisons for all number types
- IN/NOT IN operations with proper slice types

### 🔗 Chainable Query API

```go
// Build complex queries with method chaining
entities, err := ShipmentTypeBuild(db).
    WhereStatusEQ(domain.StatusActive).
    WhereCodeContains("LTL").
    WhereTenant(orgID, buID).
    OrderBy("created_at", true).
    Limit(10).
    All(ctx)
```

### 📊 Comprehensive Field Information

- Field name constants
- Column name helpers with table aliases
- Sortable/filterable field detection
- Field type preservation

### 🛡️ Built-in Safety Features

- Prevents SQL injection with proper identifier escaping
- Optimistic locking support for updates
- Tenant isolation helpers

## Installation

The generator is included in the Trenova codebase. No separate installation needed.

```bash
# Ensure you have the required dependencies
go mod tidy
```

## Quick Start

### Option 1: Generate All Entities (Recommended)

```bash
# Generate for ALL domain entities at once
go run ./cmd/gen-trenova -domain-path=./internal/core/domain

# Or use the task command
task codegen
```

This automatically:

- Scans for all structs with `bun.BaseModel`
- Generates type-safe helpers for each entity
- Places generated files next to source files

### Option 2: Generate Specific Entity

```bash
# Generate for a specific type
go run ./cmd/gen-trenova -type=ShipmentType

# Generate multiple types
go run ./cmd/gen-trenova -type=ShipmentType,Customer,Location
```

### Option 3: Use go:generate

```go
//go:generate go run github.com/emoss08/trenova/cmd/gen-trenova -type=ShipmentType

type ShipmentType struct {
    bun.BaseModel `bun:"table:shipment_types,alias:st" json:"-"`
    // ... fields
}
```

## API Reference

### Generated Structure

Each generated file contains a comprehensive query API:

```go
// ShipmentTypeQuery - Main query helper
var ShipmentTypeQuery = struct {
    // Table metadata
    Table    string  // "shipment_types"
    Alias    string  // "st"
    IDPrefix string  // "st_"
    
    // Field names
    Field struct {
        ID          string  // "id"
        Code        string  // "code"
        Status      string  // "status"
        // ... all fields
    }
    
    // Column helpers
    Column  func(field string) string           // Returns "st.field"
    Columns func(fields ...string) []string    // Returns ["st.field1", "st.field2"]
    
    // WHERE predicates
    Where struct {
        IDEQ           func(q *bun.SelectQuery, v pulid.ID) *bun.SelectQuery
        CodeContains   func(q *bun.SelectQuery, v string) *bun.SelectQuery
        StatusIn       func(q *bun.SelectQuery, v []domain.Status) *bun.SelectQuery
        CreatedAtGTE   func(q *bun.SelectQuery, v int64) *bun.SelectQuery
        Tenant         func(q *bun.SelectQuery, orgID, buID pulid.ID) *bun.SelectQuery
        // ... comprehensive predicates for each field
    }
    
    // UPDATE helpers
    Update struct {
        WhereIDAndVersion func(q *bun.UpdateQuery, id pulid.ID, version int64) *bun.UpdateQuery
    }
    
    // ORDER BY helpers
    OrderBy struct {
        Field     func(field string, desc bool) string
        Default   func() []string
        CreatedAt func(desc bool) string
        // ... helpers for common sort fields
    }
    
    // Field configuration
    FieldConfig  func() map[string]FieldConfig
    IsSortable   func(field string) bool
    IsFilterable func(field string) bool
}
```

### Chainable Query Builder

```go
// Create a new query builder
builder := NewShipmentTypeQuery(db)
// Or: builder := ShipmentTypeBuild(db)

// Chain methods for complex queries
result, err := builder.
    WhereStatusEQ(domain.StatusActive).
    WhereCodeContains("TEST").
    WhereCreatedAtGTE(startTime).
    WhereTenant(orgID, buID).
    OrderBy("created_at", true).
    Limit(20).
    Offset(40).
    All(ctx)
```

### Available Predicates by Type

#### String Fields

- `EQ`, `NEQ` - Exact match
- `In`, `NotIn` - Match any/none of values
- `GT`, `GTE`, `LT`, `LTE` - Lexicographic comparison
- `Contains` - LIKE %value%
- `HasPrefix` - LIKE value%
- `HasSuffix` - LIKE %value

#### Numeric Fields (int*, uint*, float*, decimal)

- `EQ`, `NEQ` - Exact match
- `In`, `NotIn` - Match any/none of values
- `GT`, `GTE`, `LT`, `LTE` - Numeric comparison

#### Boolean Fields

- `EQ`, `NEQ` - True/false comparison

#### Pointer/Nullable Fields

- All standard predicates plus:
- `IsNull` - Check for NULL
- `IsNotNull` - Check for NOT NULL

#### Custom Types

- Preserves exact type (e.g., `domain.Status`, `SuggestionStatus`)
- Appropriate predicates based on underlying type

## Usage Examples

### Basic Queries

```go
// Simple equality check
q := ShipmentTypeQuery.Where.CodeEQ(q, "LTL")

// Multiple conditions
q := db.NewSelect().Model(&entities)
q = ShipmentTypeQuery.Where.StatusEQ(q, domain.StatusActive)
q = ShipmentTypeQuery.Where.CreatedAtGTE(q, startTime)
```

### Using the Chainable API

```go
// Find active shipment types created in the last 30 days
thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Unix()

types, err := ShipmentTypeBuild(db).
    WhereStatusEQ(domain.StatusActive).
    WhereCreatedAtGTE(thirtyDaysAgo).
    OrderBy("code", false).
    All(ctx)

// Get a single record
shipmentType, err := ShipmentTypeBuild(db).
    WhereIDEQ(id).
    WhereTenant(orgID, buID).
    One(ctx)

// Count matching records
count, err := ShipmentTypeBuild(db).
    WhereStatusIn([]domain.Status{domain.StatusActive, domain.StatusPending}).
    Count(ctx)

// Check existence
exists, err := ShipmentTypeBuild(db).
    WhereCodeEQ("LTL").
    Exists(ctx)
```

### Complex Queries

```go
// Search with multiple criteria
results, err := CustomerBuild(db).
    WhereNameContains(searchTerm).
    WhereStatusEQ(customer.StatusActive).
    WhereStateIn([]string{"CA", "TX", "NY"}).
    WhereCreatedAtBetween(startDate, endDate).
    WhereCreditLimitGTE(decimal.NewFromFloat(10000)).
    OrderBy("credit_limit", true).
    Limit(50).
    All(ctx)

// Working with nullable fields
active, err := LocationBuild(db).
    WhereDeletedAtIsNull().
    WhereArchivedAtIsNull().
    WhereParentIDIsNotNull().
    All(ctx)
```

### Repository Integration

```go
func (r *repository) List(ctx context.Context, filter *ListFilter) ([]*ShipmentType, error) {
    q := r.db.NewSelect().Model(&entities)
    
    // Apply tenant filter using generated helper
    q = ShipmentTypeQuery.Where.Tenant(q, filter.OrgID, filter.BuID)
    
    // Apply status filter if provided
    if filter.Status != "" {
        q = ShipmentTypeQuery.Where.StatusEQ(q, filter.Status)
    }
    
    // Apply search
    if filter.Search != "" {
        searchPattern := "%" + filter.Search + "%"
        q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
            q = ShipmentTypeQuery.Where.CodeContains(q, filter.Search)
            q = q.WhereOr("? LIKE ?", 
                bun.Ident(ShipmentTypeQuery.Column("description")), 
                searchPattern)
            return q
        })
    }
    
    // Apply ordering
    for _, order := range ShipmentTypeQuery.OrderBy.Default() {
        q = q.Order(order)
    }
    
    return entities, q.Scan(ctx)
}
```

### Update with Optimistic Locking

```go
func (r *repository) Update(ctx context.Context, entity *ShipmentType) error {
    currentVersion := entity.Version
    entity.Version++ // Increment version
    
    q := r.db.NewUpdate().Model(entity)
    q = ShipmentTypeQuery.Update.WhereIDAndVersion(q, entity.ID, currentVersion)
    
    result, err := q.Exec(ctx)
    if err != nil {
        return err
    }
    
    if rows, _ := result.RowsAffected(); rows == 0 {
        return errors.New("version mismatch - record was modified")
    }
    
    return nil
}
```

## Field Configuration

### Automatic Detection

The generator intelligently detects field characteristics:

**Automatically Sortable:**

- Timestamp fields ending in `At` (CreatedAt, UpdatedAt)
- Enum/Status fields
- Code and Name fields

**Automatically Filterable:**

- All ID fields (primary and foreign keys)
- Timestamp fields
- Status/enum fields
- VARCHAR fields (not TEXT)

### Manual Control with Tags

Override automatic detection using the `codegen` tag:

```go
type Product struct {
    bun.BaseModel `bun:"table:products,alias:p"`
    
    // Force field to be sortable and filterable
    SKU string `bun:"sku" codegen:"sortable,filterable"`
    
    // Disable automatic detection
    LongDescription string `bun:"description,type:TEXT" codegen:"-sortable,-filterable"`
    
    // Mixed configuration
    Color string `bun:"color" codegen:"sortable,-filterable"`
}
```

## Advanced Features

### Custom Type Support

The generator correctly handles:

- Custom enum types (e.g., `type Status string`)
- Decimal types (`decimal.Decimal`, `decimal.NullDecimal`)
- JSON fields (`map[string]any`)
- Nested types (`*domain.Status`)
- Arrays and slices (`[]string`, `[]*Document`)

### Smart Import Management

The generator automatically:

- Detects required imports from field types
- Adds common package imports (pulid, decimal, sql)
- Uses goimports to organize and format imports
- Handles import aliases correctly

### Tenant Isolation

For multi-tenant entities with `OrganizationID` and `BusinessUnitID`:

```go
// Automatic tenant helper
q = ShipmentTypeQuery.Where.Tenant(q, orgID, buID)

// Chainable API
entities, err := ShipmentTypeBuild(db).
    WhereTenant(orgID, buID).
    WhereStatusEQ(domain.StatusActive).
    All(ctx)
```

### Comparison with Raw Queries

```go
// ❌ Before: Error-prone, no type safety
q.Where("st.sttaus = ?", "Active")  // Typo!
q.Where("st.created_at > ?", "2024-01-01")  // Wrong type!

// ✅ After: Compile-time validation
q = ShipmentTypeQuery.Where.StatusEQ(q, domain.StatusActive)
q = ShipmentTypeQuery.Where.CreatedAtGT(q, timestamp)
```

## Best Practices

### 1. Regenerate After Schema Changes

```bash
# After modifying domain entities:
task codegen

# Or for specific domain:
go generate ./internal/core/domain/...
```

### 2. Use Chainable API for New Code

The chainable API provides better ergonomics:

```go
// Prefer this:
results, err := ShipmentTypeBuild(db).
    WhereStatusEQ(status).
    OrderBy("code").
    All(ctx)

// Over this:
q := db.NewSelect().Model(&results)
q = ShipmentTypeQuery.Where.StatusEQ(q, status)
q = q.Order(ShipmentTypeQuery.OrderBy.Code(false))
err := q.Scan(ctx)
```

### 3. Leverage Field Constants

```go
// Use generated constants for dynamic queries
fieldName := ShipmentTypeQuery.Field.Status
if ShipmentTypeQuery.IsSortable(fieldName) {
    q = q.Order(ShipmentTypeQuery.OrderBy.Field(fieldName, desc))
}
```

### 4. Commit Generated Files

Always commit `*_gen.go` files:

- They're part of your codebase
- Required for compilation
- Help with code reviews

## Troubleshooting

### "Cannot find package" errors

**Solution:** Run `go mod tidy` and ensure all imports are available.

### Fields have wrong types (e.g., bool shown as string)

**Solution:** Check that you're using the latest generator. The AST parser now correctly handles all Go types.

### Missing predicates for custom types

**Solution:** The generator creates basic predicates (EQ, NEQ) for unknown types. For richer predicates, ensure the type is recognized in the template.

### "Type X not found or not a struct"

**Solution:** Ensure:

1. The type name matches exactly
2. The type is in the current package
3. The type has `bun.BaseModel` embedded

### Unused imports in generated code

**Solution:** The generator uses goimports to clean up. If issues persist, run `goimports -w *_gen.go`.

## Performance

All generated code has zero runtime overhead:

- Constants are resolved at compile time
- Functions are inlined by the compiler
- No reflection or interface{} boxing
- Type assertions eliminated

## Contributing

When extending the generator:

1. Update type extraction in `extractGoType()` for new types
2. Add predicates in `template_combined.go` for new operations
3. Update import mappings for new packages
4. Add examples to this README
5. Run tests to ensure compatibility

## License

Part of the Trenova project. See LICENSE for details.
