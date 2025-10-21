# Sequence Generator Package

A high-performance, thread-safe sequence generator for creating unique identifiers in transportation management systems. Designed to handle high-volume scenarios (10,000+ sequences per day) with guaranteed uniqueness and zero duplicates.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage Examples](#usage-examples)
- [Configuration](#configuration)
- [Format Templates](#format-templates)
- [High Volume Support](#high-volume-support)
- [API Reference](#api-reference)
- [Testing](#testing)
- [Performance](#performance)

## Features

### Core Features

- **Guaranteed Uniqueness**: Atomic database operations ensure no duplicate sequences
- **High Performance**: Supports 100,000+ sequences per day with batch generation
- **Format Flexibility**: Customizable formats with templates and overrides
- **Thread-Safe**: Concurrent-safe with SERIALIZABLE isolation level
- **Caching**: Built-in format caching with configurable TTL
- **Validation**: Comprehensive sequence validation including check digit verification
- **Batch Generation**: Generate multiple sequences in a single transaction
- **Audit Trail**: Optional tracking of generated sequences

### Supported Sequence Types

- Pro Numbers (shipping/tracking numbers)
- Consolidation Numbers
- Invoice Numbers (future)
- Work Order Numbers (future)

## Architecture

### Design Principles

1. **Separation of Concerns**
   - `Generator`: Core logic for sequence generation
   - `SequenceStore`: Database operations and persistence
   - `FormatProvider`: Format configuration retrieval

2. **Database Guarantees**
   - SERIALIZABLE isolation prevents race conditions
   - Optimistic locking with version field
   - Atomic increment operations

3. **Performance Optimizations**
   - Connection pooling
   - Format caching (15-minute TTL)
   - Batch operations for bulk generation

### Component Structure

```text
seqgen/
├── generator.go      # Core generation logic
├── store.go          # Database operations
├── types.go          # Type definitions and templates
├── errors.go         # Error definitions
└── generator_test.go # Comprehensive test suite
```

## Installation

```go
import "github.com/emoss08/trenova/pkg/seqgen"
```

## Quick Start

### Basic Usage

```go
// Initialize with dependency injection
params := seqgen.GeneratorParams{
    Store:    sequenceStore,    // Implements SequenceStore interface
    Provider: formatProvider,    // Implements FormatProvider interface
    Logger:   logger,
}
generator := seqgen.NewGenerator(params)

// Generate a single sequence
req := &seqgen.GenerateRequest{
    Type:  seqgen.SequenceTypeProNumber,
    OrgID: orgID,
    BuID:  businessUnitID,
}
sequence, err := generator.Generate(ctx, req)
// Result: "S24010001234567" (example)
```

### Batch Generation

```go
// Generate multiple sequences at once (more efficient)
req := &seqgen.GenerateRequest{
    Type:  seqgen.SequenceTypeProNumber,
    OrgID: orgID,
    BuID:  businessUnitID,
    Count: 100,  // Generate 100 sequences
}
sequences, err := generator.GenerateBatch(ctx, req)
```

## Usage Examples

### Using Format Templates

```go
// Use a predefined template
format := seqgen.GetFormatFromTemplate(seqgen.TemplateYearMonth, &seqgen.Format{
    Type:   seqgen.SequenceTypeProNumber,
    Prefix: "INV",
})

req := &seqgen.GenerateRequest{
    Type:   seqgen.SequenceTypeProNumber,
    OrgID:  orgID,
    BuID:   businessUnitID,
    Format: &format,
}
sequence, err := generator.Generate(ctx, req)
// Result: "INV-2024-01-0001"
```

### Custom Format

```go
// Define a custom format
customFormat := &seqgen.Format{
    Type:                seqgen.SequenceTypeProNumber,
    Prefix:              "SHIP",
    IncludeYear:         true,
    YearDigits:          4,
    IncludeMonth:        true,
    IncludeDay:          true,
    SequenceDigits:      6,
    IncludeLocationCode: true,
    LocationCode:        "LAX",
    IncludeCheckDigit:   true,
    UseSeparators:       true,
    SeparatorChar:       "-",
}

req := &seqgen.GenerateRequest{
    Type:   seqgen.SequenceTypeProNumber,
    OrgID:  orgID,
    BuID:   businessUnitID,
    Format: customFormat,
}
sequence, err := generator.Generate(ctx, req)
// Result: "SHIP-2024-01-15-LAX-000001-7"
```

### Sequence Validation

```go
// Validate an existing sequence
format := &seqgen.Format{
    Prefix:         "PRE",
    SequenceDigits: 4,
}

err := generator.ValidateSequence("PRE0001", format)
if err != nil {
    // Sequence is invalid
}
```

### Parse Sequence Components

```go
// Extract components from a sequence
components, err := generator.ParseSequence("SHIP-2024-01-LAX-000001", format)
if err == nil {
    fmt.Printf("Prefix: %s\n", components.Prefix)
    fmt.Printf("Year: %s\n", components.Year)
    fmt.Printf("Month: %s\n", components.Month)
    fmt.Printf("Location: %s\n", components.LocationCode)
    fmt.Printf("Sequence: %s\n", components.Sequence)
}
```

## Configuration

### Format Configuration

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `Type` | SequenceType | Type of sequence | `SequenceTypeProNumber` |
| `Prefix` | string | Fixed prefix | `"SHIP"` |
| `IncludeYear` | bool | Include year in sequence | `true` |
| `YearDigits` | int | Number of year digits (2 or 4) | `4` |
| `IncludeMonth` | bool | Include month | `true` |
| `IncludeDay` | bool | Include day | `false` |
| `IncludeWeekNumber` | bool | Include ISO week number | `false` |
| `SequenceDigits` | int | Number of sequence digits (1-10) | `6` |
| `IncludeLocationCode` | bool | Include location code | `true` |
| `LocationCode` | string | Location identifier | `"LAX"` |
| `IncludeBusinessUnitCode` | bool | Include business unit | `false` |
| `BusinessUnitCode` | string | Business unit code | `"01"` |
| `IncludeRandomDigits` | bool | Add random digits | `false` |
| `RandomDigitsCount` | int | Number of random digits | `6` |
| `IncludeCheckDigit` | bool | Add Luhn check digit | `true` |
| `UseSeparators` | bool | Use separator characters | `true` |
| `SeparatorChar` | string | Separator character | `"-"` |
| `AllowCustomFormat` | bool | Use custom format string | `false` |
| `CustomFormat` | string | Custom format pattern | `"{P}{Y}{M}-{S}"` |

### Custom Format Placeholders

When using `AllowCustomFormat`, these placeholders are available:

- `{P}` - Prefix
- `{Y}` - Year
- `{M}` - Month (2 digits)
- `{W}` - ISO Week number (2 digits)
- `{D}` - Day (2 digits)
- `{L}` - Location code
- `{B}` - Business unit code
- `{S}` - Sequential number
- `{R}` - Random digits
- `{C}` - Check digit

Example: `"{P}-{Y}{M}-{L}-{S}"` produces `"SHIP-202401-LAX-000001"`

## Format Templates

Pre-defined templates for common use cases:

### Simple Sequential

```go
format := seqgen.GetFormatFromTemplate(seqgen.TemplateSimpleSequential, nil)
// Result: "SEQ000001"
```

### Year-Month

```go
format := seqgen.GetFormatFromTemplate(seqgen.TemplateYearMonth, nil)
// Result: "YM-2024-01-0001"
```

### Full Date

```go
format := seqgen.GetFormatFromTemplate(seqgen.TemplateFullDate, nil)
// Result: "FD-2024-01-15-0001"
```

### Location Based

```go
format := seqgen.GetFormatFromTemplate(seqgen.TemplateLocationBased, &seqgen.Format{
    LocationCode: "NYC",
})
// Result: "LOC-NYC-00001"
```

### With Check Digit

```go
format := seqgen.GetFormatFromTemplate(seqgen.TemplateCheckDigit, nil)
// Result: "CHK00001-9"
```

### Comprehensive

```go
format := seqgen.GetFormatFromTemplate(seqgen.TemplateComprehensive, nil)
// Result: "CMP-01-2024-01-XXX-0001-ABC-7"
```

## High Volume Support

### Capacity Planning

| Volume | Sequences/Day | Recommended Config |
|--------|--------------|-------------------|
| Small | <1,000 | `SequenceDigits: 4` |
| Medium | 1,000-10,000 | `SequenceDigits: 5` |
| High | 10,000-100,000 | `SequenceDigits: 6` |
| Very High | >100,000 | `SequenceDigits: 7-8` |

### Example: 50,000 Shipments/Day

```go
// Configuration for high volume
format := &seqgen.Format{
    Type:                seqgen.SequenceTypeProNumber,
    Prefix:              "PRO",
    IncludeYear:         true,
    YearDigits:          2,
    IncludeMonth:        true,
    SequenceDigits:      7,      // Supports 9,999,999/month
    IncludeLocationCode: true,
    LocationCode:        "LAX",
    UseSeparators:       false,  // Save space
}

// Use batch generation for efficiency
req := &seqgen.GenerateRequest{
    Type:  seqgen.SequenceTypeProNumber,
    OrgID: orgID,
    BuID:  businessUnitID,
    Count: 500,  // Generate 500 at once
}
sequences, err := generator.GenerateBatch(ctx, req)
```

### Performance Tips

1. **Use Batch Generation**: For bulk operations, generate 100-500 sequences at once
2. **Cache Formats**: Formats are cached for 15 minutes by default
3. **Optimize Format**: Avoid unnecessary components (separators, random digits)
4. **Monitor Usage**: Track sequence consumption to prevent exhaustion

## API Reference

### Generator Interface

```go
type Generator interface {
    // Generate creates a single sequence
    Generate(ctx context.Context, req *GenerateRequest) (string, error)

    // GenerateBatch creates multiple sequences
    GenerateBatch(ctx context.Context, req *GenerateRequest) ([]string, error)

    // ValidateSequence validates a sequence against a format
    ValidateSequence(sequence string, format *Format) error

    // ParseSequence extracts components from a sequence
    ParseSequence(sequence string, format *Format) (*SequenceComponents, error)

    // ClearCache clears the format cache
    ClearCache()

    // SetCacheTTL sets cache duration
    SetCacheTTL(ttl time.Duration)
}
```

### SequenceStore Interface

```go
type SequenceStore interface {
    // GetNextSequence returns the next sequence number
    GetNextSequence(ctx context.Context, req *SequenceRequest) (int64, error)

    // GetNextSequenceBatch returns multiple sequence numbers
    GetNextSequenceBatch(ctx context.Context, req *SequenceRequest) ([]int64, error)
}
```

### FormatProvider Interface

```go
type FormatProvider interface {
    // GetFormat retrieves format configuration
    GetFormat(
        ctx context.Context,
        sequenceType SequenceType,
        orgID, buID pulid.ID,
    ) (*Format, error)
}
```

## Testing

Run the test suite:

```bash
go test ./pkg/seqgen/... -v
```

Run benchmarks:

```bash
go test ./pkg/seqgen/... -bench=. -benchmem
```

## Performance

### Benchmarks

```text
BenchmarkGenerator_Generate-8           10000    115234 ns/op    4096 B/op    82 allocs/op
BenchmarkGenerator_GenerateBatch-8       2000    650123 ns/op   40960 B/op   820 allocs/op
```

### Database Indexes

Ensure these indexes exist for optimal performance:

```sql
-- Primary lookup index
CREATE UNIQUE INDEX idx_sequences_lookup
ON sequences(sequence_type, organization_id, business_unit_id, year, month);

-- Version index for optimistic locking
CREATE INDEX idx_sequences_version
ON sequences(version);
```

## Error Handling

Common errors and their meanings:

- `ErrSequenceUpdateConflict`: Concurrent update detected, will retry
- `ErrInvalidSequenceType`: Unknown sequence type provided
- `ErrSequenceFormatNil`: Format configuration is nil
- `ErrSequenceDoesNotMatch`: Sequence doesn't match expected format

## Best Practices

1. **Always use batch generation for bulk operations** - It's significantly faster
2. **Configure appropriate sequence digits** - Plan for growth
3. **Use format templates** - Consistent and tested configurations
4. **Monitor sequence usage** - Prevent exhaustion
5. **Cache formats** - Reduce database lookups
6. **Validate critical sequences** - Ensure integrity

## License

Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2
