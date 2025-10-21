# Validation Framework

## Overview

The Validation Framework provides a comprehensive, type-safe, and performant approach to validating domain entities in the Trenova transportation management system. Built with enterprise requirements in mind, it offers interface-based validation patterns, multi-tenant support, and compliance with transportation industry regulations.

## Key Features

- **Interface-Based Design**: Domain models implement standard interfaces (`TenantedEntity`, `ValidatableEntity`)
- **Multi-Tenant Support**: All validators are tenant-aware by default
- **Type Safety**: Generic validators with compile-time type checking
- **Performance Optimized**: Indexed rule storage, parallel execution, and caching
- **Fluent API**: Intuitive builder pattern for complex validations
- **Staged Validation**: Organized validation stages with priorities
- **Reusable Components**: Common validators and database validation rules
- **Testing Support**: Comprehensive testing utilities and mocks

## Architecture

### Core Interfaces

The framework is built around domain models implementing these interfaces:

```go
// Basic entity interfaces
type Validatable interface {
    Validate(multiErr *errortypes.MultiError)
}

type Identifiable interface {
    GetID() string
}

type Tableable interface {
    GetTableName() string
}

type Tenantable interface {
    GetOrganizationID() pulid.ID
    GetBusinessUnitID() pulid.ID
}

// Composite interfaces
type ValidatableEntity interface {
    Validatable
    Identifiable
    Tableable
}

type TenantedEntity interface {
    ValidatableEntity
    Tenantable
}
```

### Validation Flow

```text
┌─────────────────────┐
│   Domain Entity     │
│ (implements         │
│  TenantedEntity)    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ TenantedValidator   │
│   Factory           │
└──────────┬──────────┘
           │ creates
           ▼
┌─────────────────────┐
│ TenantedValidator   │
│ - Domain validation │
│ - ID validation     │
│ - Uniqueness checks │
│ - Custom rules      │
└──────────┬──────────┘
           │ uses
           ▼
┌─────────────────────┐
│ ValidationEngine    │
│ - Staged execution  │
│ - Parallel rules    │
│ - Error aggregation │
└─────────────────────┘
```

### Validation Stages

Validation rules execute in ordered stages:

1. **Basic** (`ValidationStageBasic`)
   - Field presence and format validation
   - Domain model's self-validation
   - Type checking

2. **Data Integrity** (`ValidationStageDataIntegrity`)
   - Uniqueness constraints
   - Foreign key validation
   - Referential integrity

3. **Business Rules** (`ValidationStageBusinessRules`)
   - Domain-specific logic
   - Cross-field dependencies
   - State transitions

4. **Compliance** (`ValidationStageCompliance`)
   - Regulatory requirements (FMCSA, DOT)
   - Industry standards
   - Legal constraints

### Validation Priorities

Within each stage, rules execute by priority:

- **High** (`ValidationPriorityHigh`): Critical, must-pass validations
- **Medium** (`ValidationPriorityMedium`): Important validations
- **Low** (`ValidationPriorityLow`): Optional, nice-to-have validations

## Implementation Guide

### 1. Domain Model Implementation

Your domain model must implement the `TenantedEntity` interface:

```go
package domain

type HoldReason struct {
    bun.BaseModel `bun:"table:hold_reasons"`

    ID             pulid.ID `json:"id" bun:"id,pk"`
    OrganizationID pulid.ID `json:"organizationId" bun:"organization_id"`
    BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id"`
    Code           string   `json:"code" bun:"code"`
    Label          string   `json:"label" bun:"label"`
    // ... other fields
}

// Implement Validatable
func (hr *HoldReason) Validate(multiErr *errortypes.MultiError) {
    // Domain-specific validation logic
    if hr.Code == "" {
        multiErr.Add("code", errortypes.ErrRequired, "Code is required")
    }
    if len(hr.Code) > 64 {
        multiErr.Add("code", errortypes.ErrInvalid, "Code must be 64 characters or less")
    }
}

// Implement Identifiable
func (hr *HoldReason) GetID() string {
    return hr.ID.String()
}

// Implement Tableable
func (hr *HoldReason) GetTableName() string {
    return "hold_reasons"
}

// Implement Tenantable
func (hr *HoldReason) GetOrganizationID() pulid.ID {
    return hr.OrganizationID
}

func (hr *HoldReason) GetBusinessUnitID() pulid.ID {
    return hr.BusinessUnitID
}
```

### 2. Create a Validator Using TenantedValidatorFactory

```go
package validators

import (
    "context"
    "github.com/emoss08/trenova/pkg/validator/framework"
    "go.uber.org/fx"
)

type ValidatorParams struct {
    fx.In
    DB *postgres.Connection
}

type HoldReasonValidator struct {
    factory *framework.TenantedValidatorFactory[*domain.HoldReason]
}

func NewHoldReasonValidator(p ValidatorParams) *HoldReasonValidator {
    factory := framework.NewTenantedValidatorFactory[*domain.HoldReason](
        func(ctx context.Context) (*bun.DB, error) {
            return p.DB.DB(ctx)
        },
    ).
    WithModelName("HoldReason").
    WithUniqueFields(func(hr *domain.HoldReason) []framework.UniqueField {
        return []framework.UniqueField{
            {
                Name:     "code",
                GetValue: func() string { return hr.Code },
                Message:  "Hold reason with code ':value' already exists",
            },
            {
                Name:     "label",
                GetValue: func() string { return hr.Label },
            },
        }
    }).
    WithCustomRules(func(hr *domain.HoldReason, valCtx *validator.ValidationContext) []framework.ValidationRule {
        var rules []framework.ValidationRule

        // Add business rules
        rules = append(rules, framework.NewBusinessRule("severity_check").
            WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
                if hr.Severity == "BLOCKING" && !hr.BlocksShipment {
                    multiErr.Add("severity", errortypes.ErrInvalid,
                        "Blocking severity must block shipments")
                }
                return nil
            }))

        return rules
    })

    return &HoldReasonValidator{factory: factory}
}

// Validate method
func (v *HoldReasonValidator) Validate(
    ctx context.Context,
    valCtx *validator.ValidationContext,
    hr *domain.HoldReason,
) *errortypes.MultiError {
    return v.factory.Validate(ctx, hr, valCtx)
}
```

### 3. Using the Validation Engine Directly

For more complex scenarios, use the ValidationEngine directly:

```go
func (v *CustomValidator) Validate(ctx context.Context, entity *Entity) *errortypes.MultiError {
    engine := framework.NewValidationEngine(&framework.EngineConfig{
        MaxParallel: 10,
        FailFast:    false,
    })

    // Add validation rules
    engine.AddRule(
        framework.NewConcreteRule("basic_validation").
            WithStage(framework.ValidationStageBasic).
            WithPriority(framework.ValidationPriorityHigh).
            WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
                entity.Validate(multiErr)
                return nil
            }),
    )

    // Add uniqueness validation
    engine.AddRule(
        framework.NewUniquenessRule("uniqueness", v.getDB).
            ForTable(entity.GetTableName()).
            ForModel("Entity").
            WithTenant(func() (pulid.ID, pulid.ID) {
                return entity.GetOrganizationID(), entity.GetBusinessUnitID()
            }).
            CheckField("code", func() string { return entity.Code }, ""),
    )

    return engine.Validate(ctx)
}
```

### 4. Using the Fluent Builder API

For complex validation scenarios with method chaining:

```go
func ValidateWithBuilder(ctx context.Context, entity *Entity) *errortypes.MultiError {
    return framework.NewValidationBuilder().
        WithConfig(&framework.EngineConfig{
            MaxParallel: 5,
            FailFast:    true,
        }).
        Basic("required_fields", func(ctx context.Context, multiErr *errortypes.MultiError) error {
            entity.Validate(multiErr)
            return nil
        }).
        DataIntegrity("uniqueness", func(ctx context.Context, multiErr *errortypes.MultiError) error {
            // Check uniqueness
            return nil
        }).
        BusinessRule("state_transition", func(ctx context.Context, multiErr *errortypes.MultiError) error {
            // Validate state transitions
            return nil
        }).
        When(func() bool { return entity.RequiresCompliance() },
            func(vb *framework.ValidationBuilder) {
                vb.Compliance("DOT", "49CFR", func(ctx context.Context, multiErr *errortypes.MultiError) error {
                    // DOT compliance checks
                    return nil
                })
            }).
        Validate(ctx)
}
```

### 5. Validating Collections with Indexed Errors

For validating arrays or slices with proper error indexing:

```go
func (v *ShipmentValidator) ValidateMoves(
    ctx context.Context,
    moves []*Move,
    parentErr *errortypes.MultiError,
) {
    for idx, move := range moves {
        engine := v.engineFactory.CreateEngine().
            ForField("moves").
            AtIndex(idx).
            WithParent(parentErr)

        engine.AddRule(
            framework.NewConcreteRule("move_validation").
                WithStage(framework.ValidationStageBasic).
                WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
                    move.Validate(multiErr)
                    return nil
                }),
        )

        // Validate nested stops
        for stopIdx, stop := range move.Stops {
            stopEngine := v.engineFactory.CreateEngine().
                ForField("stops").
                AtIndex(stopIdx).
                WithParent(multiErr)

            stopEngine.AddRule(/* stop validation rules */)
            _ = stopEngine.Validate(ctx) // Returns nil with WithParent
        }

        _ = engine.Validate(ctx) // Returns nil with WithParent
    }
}
```

## Common Validation Patterns

### Uniqueness Validation

The framework provides a reusable `UniquenessRule`:

```go
rule := framework.NewUniquenessRule("check_unique", getDB).
    ForTable("users").
    ForModel("User").
    WithTenant(func() (pulid.ID, pulid.ID) {
        return user.OrganizationID, user.BusinessUnitID
    }).
    ForOperation(isCreate).
    CheckField("email", func() string { return user.Email },
        "Email ':value' is already in use").
    CheckField("username", func() string { return user.Username },
        "Username ':value' is taken")

if !isCreate {
    rule.WithPrimaryKey(func() string { return user.ID.String() })
}
```

### Conditional Validation

Apply rules based on conditions:

```go
engine.AddRule(
    framework.NewBusinessRule("conditional_check").
        WithCondition(func() bool {
            return entity.Status == "ACTIVE"
        }).
        WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
            // Only runs if condition is true
            return nil
        }),
)
```

### Composite Rules

Combine multiple rules with logical operators:

```go
compositeRule := framework.NewCompositeRule("complex_validation").
    WithOperator(framework.OperatorAND).
    AddRule(rule1).
    AddRule(rule2).
    AddRule(
        framework.NewCompositeRule("nested").
            WithOperator(framework.OperatorOR).
            AddRule(rule3).
            AddRule(rule4),
    )
```

### Async Validation

For validations that require external calls:

```go
asyncRule := framework.NewAsyncValidationRule("external_check").
    WithTimeout(5 * time.Second).
    WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
        // Call external service
        result, err := externalService.Validate(ctx, entity)
        if err != nil {
            return err
        }
        if !result.Valid {
            multiErr.Add("external", errortypes.ErrInvalid, result.Message)
        }
        return nil
    })
```

### Batch Validation

For validating large collections efficiently:

```go
batchValidator := framework.NewBatchValidator[*Entity](
    func(ctx context.Context, entity *Entity) *errortypes.MultiError {
        return validator.Validate(ctx, entity)
    },
).WithChunkSize(100).
  WithParallel(true).
  WithMaxWorkers(10)

results := batchValidator.ValidateAll(ctx, entities)
```

## Testing

### Using Mock Validators

```go
func TestValidator(t *testing.T) {
    mockEngine := framework.NewMockValidationEngine()
    mockEngine.SetupValidate(func(ctx context.Context) *errortypes.MultiError {
        return errortypes.NewMultiError()
    })

    validator := &YourValidator{
        engine: mockEngine,
    }

    err := validator.Validate(ctx, entity)
    assert.Nil(t, err)
    assert.True(t, mockEngine.ValidateCalled())
}
```

### Testing with Assertions

```go
func TestValidationRules(t *testing.T) {
    assertions := framework.NewValidationAssertions(t)

    entity := &TestEntity{Code: ""}
    errors := validator.Validate(ctx, entity)

    assertions.
        HasError(errors, "code").
        WithType(errortypes.ErrRequired).
        WithMessage("Code is required")

    assertions.NoError(validator.Validate(ctx, validEntity))
}
```

### Benchmarking

```go
func BenchmarkValidation(b *testing.B) {
    bench := framework.NewValidationBenchmark()

    bench.Run(b, "TenantedValidator", func() {
        _ = validator.Validate(ctx, entity)
    })

    bench.Report() // Outputs performance metrics
}
```

## Performance Considerations

### Optimization Techniques

1. **Indexed Rule Storage**: Rules are stored in maps keyed by `stage:priority` for O(1) lookup
2. **Parallel Execution**: Rules within the same priority level execute concurrently
3. **Fail-Fast Mode**: Stop validation on first critical error
4. **Caching**: Validation results are cached with configurable TTL
5. **Lazy Evaluation**: Rules with conditions are only evaluated when needed

### Configuration

```go
config := &framework.EngineConfig{
    MaxParallel:     10,              // Max concurrent rules
    FailFast:        true,            // Stop on first error
    CacheTTL:        5 * time.Minute, // Cache validation results
    EnableMetrics:   true,            // Track performance metrics
    EnableTracing:   false,           // Debug mode
}
```

## Best Practices

1. **Implement Interfaces**: Always implement `TenantedEntity` for domain models
2. **Use Factory Pattern**: Leverage `TenantedValidatorFactory` for consistency
3. **Stage Appropriately**: Place validations in correct stages
4. **Reuse Components**: Use common validators and rules
5. **Handle Errors Gracefully**: Check for nil errors and provide clear messages
6. **Test Thoroughly**: Write tests for complex validation logic
7. **Monitor Performance**: Use metrics to identify bottlenecks
8. **Document Rules**: Add descriptions to complex business rules

## Integration with Dependency Injection

The framework integrates with Uber's fx:

```go
// In your module
var Module = fx.Module("validators",
    fx.Provide(
        NewHoldReasonValidator,
        // other validators...
    ),
    fx.Options(
        framework.Module, // Include framework module
    ),
)

// The framework module provides:
// - ValidationEngineFactory
// - ValidationBuilderFactory
// - ValidationContextFactory
```

## Common Validators Reference

The framework includes pre-built validators:

- **String**: Email, URL, UUID, AlphaNumeric, NoWhitespace
- **Numeric**: Range, Positive, Negative, Precision
- **Date/Time**: Before, After, Between, Future, Past
- **Collection**: MinSize, MaxSize, NotEmpty, Unique
- **US-Specific**: PhoneNumber, ZIPCode, StateCode, SSN
- **Business**: TaxID, DOTNumber, MCNumber, DUNSNumber

## Error Handling

Errors are aggregated in `MultiError` with structured information:

```go
multiErr := validator.Validate(ctx, entity)
if multiErr != nil && multiErr.HasErrors() {
    for field, errors := range multiErr.Errors {
        for _, err := range errors {
            log.Printf("Field: %s, Type: %s, Message: %s",
                field, err.Type, err.Message)
        }
    }
}
```

## Migration from Legacy Validators

Since this is a new application without legacy support:

1. All new validators should use `TenantedValidatorFactory`
2. Domain models must implement `TenantedEntity`
3. Use the framework's common validators
4. Leverage the uniqueness rule for database constraints

## Support and Documentation

- **Examples**: See `/pkg/validator/holdreasonvalidator` for reference implementation
- **Tests**: Check `/pkg/validator/framework/*_test.go` for usage patterns
- **Types**: Review `/pkg/validator/framework/types.go` for all available types
