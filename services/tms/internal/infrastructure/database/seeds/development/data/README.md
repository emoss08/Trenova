# Seed Data Files

This directory contains YAML data files used by the development seeds. These files demonstrate the **data externalization** feature of the Trenova seeding system.

## Why YAML Data Files?

Instead of hard-coding data directly in Go code, we externalize it to YAML files for several benefits:

1. **Easier Maintenance**: Non-developers can update seed data without touching Go code
2. **Better Readability**: YAML is more human-readable than Go structs
3. **Separation of Concerns**: Data is separated from business logic
4. **Version Control**: Changes to data are clearly visible in git diffs
5. **Reusability**: Same data can be used across multiple seeds or environments

## How It Works

### 1. Create a YAML File

Create a `.yaml` file in this directory with your seed data:

```yaml
# example.yaml
entities:
  - name: "Example 1"
    value: 100
    active: true
  - name: "Example 2"
    value: 200
    active: false
```

### 2. Define a Go Struct

In your seed file, define a struct matching the YAML structure:

```go
type EntityData struct {
    Name   string `yaml:"name"`
    Value  int    `yaml:"value"`
    Active bool   `yaml:"active"`
}
```

### 3. Load the Data

Use the `DataLoader` to load the YAML file:

```go
loader := seedhelpers.NewDataLoader("./internal/.../seeds/development/data")

var data struct {
    Entities []EntityData `yaml:"entities"`
}

if err := loader.LoadYAML("example.yaml", &data); err != nil {
    return fmt.Errorf("load example.yaml: %w", err)
}

// Now use data.Entities in your seed
for _, entity := range data.Entities {
    // Create database records...
}
```

## Example Files

### `workers.yaml`

Demonstrates a complete worker seeding example with:
- Multiple worker types (Driver, Dispatcher, Warehouse, etc.)
- Various data types (strings, booleans, dates, lists)
- Comments and documentation
- Active and inactive records

See `03_worker_example.go` for the companion seed implementation.

### `formula_templates.yaml`

Real production data for formula templates (referenced by FormulaTemplateSeed).

## YAML Syntax Guide

### Basic Types

```yaml
# Strings (quotes optional unless special characters)
name: John Smith
description: "A string with: special, characters!"

# Numbers
age: 30
price: 99.99

# Booleans
is_active: true
enabled: false

# Null values
middle_name: null
# or
middle_name: ~
# or omit the field entirely

# Dates (use ISO format)
hire_date: "2023-01-15"
date_of_birth: "1990-05-20"
```

### Lists

```yaml
# Simple list
tags:
  - red
  - green
  - blue

# List of objects
workers:
  - name: "John"
    age: 30
  - name: "Jane"
    age: 25
```

### Nested Objects

```yaml
employee:
  personal_info:
    first_name: "John"
    last_name: "Doe"
  contact:
    email: "john@example.com"
    phone: "555-0100"
```

### Multi-line Strings

```yaml
# Preserve line breaks
description: |
  This is line 1
  This is line 2
  This is line 3

# Fold into single line
summary: >
  This long text
  will be folded
  into a single line
```

## Best Practices

### 1. Use Comments Liberally

```yaml
# Production database connection
workers:
  # Driver - High priority routes
  - code: "DRV001"
    # Certified for hazmat since 2020
    endorsements:
      - Hazmat
```

### 2. Consistent Naming

Use snake_case for field names to match database conventions:

```yaml
# Good
first_name: "John"
date_of_birth: "1990-01-01"

# Avoid
firstName: "John"
DateOfBirth: "1990-01-01"
```

### 3. Group Related Data

```yaml
# Drivers first
drivers:
  - code: "DRV001"
  - code: "DRV002"

# Then dispatchers
dispatchers:
  - code: "DSP001"
```

### 4. Use Default Values in Go

Don't include fields that have sensible defaults:

```yaml
# In YAML, omit fields with defaults
workers:
  - name: "John"
    # is_active defaults to true in Go
```

```go
// In Go seed, set defaults
worker := &Worker{
    Name:     workerData.Name,
    IsActive: true,  // default
}
if workerData.IsActive != nil {
    worker.IsActive = *workerData.IsActive
}
```

### 5. Validate Data in Seed

```go
for _, workerData := range data.Workers {
    if workerData.Code == "" {
        sc.Logger().Warn("Skipping worker with empty code")
        continue
    }

    if workerData.Email == "" {
        return fmt.Errorf("worker %s missing required email", workerData.Code)
    }

    // Create worker...
}
```

### 6. Handle Missing References

```go
state, err := sc.GetStateByAbbreviation(ctx, workerData.StateAbbreviation)
if err != nil {
    sc.Logger().Warn("State not found: %s, skipping worker", workerData.StateAbbreviation)
    continue  // or use a default state
}
```

## File Organization

```
data/
├── README.md                    # This file
├── workers.yaml                 # Worker seed data
├── formula_templates.yaml       # Formula template data
├── customers.yaml               # Customer seed data
└── locations.yaml               # Location seed data
```

## Environment-Specific Data

You can create environment-specific YAML files:

```
seeds/
├── development/
│   └── data/
│       └── workers.yaml         # Dev data (7 workers)
├── staging/
│   └── data/
│       └── workers.yaml         # Staging data (100 workers)
└── production/
    └── data/
        └── workers.yaml         # Production data (1000+ workers)
```

Load based on environment:

```go
env := os.Getenv("APP_ENV")
filename := fmt.Sprintf("workers_%s.yaml", env)
loader.LoadYAML(filename, &data)
```

## Troubleshooting

### YAML Parse Errors

```
Error: yaml: line 15: mapping values are not allowed in this context
```

**Solution**: Check for unquoted strings with colons:
```yaml
# Bad
description: Time: 9:00 AM

# Good
description: "Time: 9:00 AM"
```

### Type Mismatch

```
Error: cannot unmarshal !!str `true` into bool
```

**Solution**: Remove quotes from booleans and numbers:
```yaml
# Bad
is_active: "true"
age: "30"

# Good
is_active: true
age: 30
```

### File Not Found

```
Error: open workers.yaml: no such file or directory
```

**Solution**: Use correct relative path in DataLoader:
```go
// From seed file: seeds/development/03_worker_example.go
// To data file:   seeds/development/data/workers.yaml
loader := seedhelpers.NewDataLoader("./internal/infrastructure/database/seeds/development/data")
```

## Testing Your YAML

Validate YAML syntax online: https://www.yamllint.com/

Or use the `yq` command-line tool:
```bash
yq eval workers.yaml
```

## Further Reading

- [YAML Specification](https://yaml.org/spec/1.2.2/)
- [Trenova Seeding System README](../../../../../../../pkg/seedhelpers/README.md)
- [Migration Guide](../../../../../../../docs/SEEDING_MIGRATION_GUIDE.md)
