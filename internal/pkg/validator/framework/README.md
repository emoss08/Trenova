# Validation Framework

## Overview

The Validation Framework provides a structured approach to validating domain objects in the Trenova transportation management system. It offers a consistent pattern for creating and executing validation rules with defined stages and priorities, making validation more maintainable, testable, and compliant with transportation industry regulations.

## Core Components

### Validation Stages

The framework organizes validation into distinct stages that execute in sequence:

1. **Basic Validation** (`ValidationStageBasic`): Validates field presence, formats, and other fundamental requirements
2. **Data Integrity** (`ValidationStageDataIntegrity`): Validates uniqueness constraints, referential integrity, and other data consistency rules
3. **Business Rules** (`ValidationStageBusinessRules`): Validates domain-specific business logic
4. **Compliance** (`ValidationStageCompliance`): Validates regulatory requirements (FMCSA, DOT, etc.)

### Validation Priorities

Within each stage, rules are executed according to their priority:

1. **High** (`ValidationPriorityHigh`): Critical validations that must pass
2. **Medium** (`ValidationPriorityMedium`): Important validations that should pass
3. **Low** (`ValidationPriorityLow`): Optional validations that are nice to have

### Key Interfaces and Types

- **ValidationRule**: Interface for individual validation rules
- **ValidationEngine**: Collects and executes validation rules
- **ValidationEngineFactory**: Creates validation engines (used with dependency injection)

## Architecture

The framework follows a modular design with dependency injection through Uber's fx:

```
┌─────────────────────┐      ┌───────────────────┐
│                     │      │                   │
│  Domain Validator   │◄─────┤ ValidationEngine  │
│ (ShipmentValidator) │      │                   │
│                     │      └───────────────────┘
└─────────────────────┘              ▲
                                     │ creates
                                     │
┌─────────────────────┐      ┌───────────────────┐
│                     │      │                   │
│     Validator       │◄─────┤ValidationEngine   │
│     Module          │      │Factory            │
│                     │      │                   │
└─────────────────────┘      └───────────────────┘
```

## How to Use

### 1. Update Your Validator to Accept ValidationEngineFactory

```go
type ValidatorParams struct {
    fx.In

    DB                     db.Connection
    ValidationEngineFactory framework.ValidationEngineFactory
    // Other dependencies...
}

type Validator struct {
    db  db.Connection
    vef framework.ValidationEngineFactory
    // Other fields...
}

func NewValidator(p ValidatorParams) *Validator {
    return &Validator{
        db:  p.DB,
        vef: p.ValidationEngineFactory,
        // Initialize other fields...
    }
}
```

### 2. Structure Your Validation Method - Return a New MultiError

If your validator returns a new `MultiError`, use the `Validate` method:

```go
func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, entity *domain.Entity) *errors.MultiError {
    // Create a validation engine
    engine := v.vef.CreateEngine()

    // Add basic validation rules
    engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh, 
        func(ctx context.Context, multiErr *errors.MultiError) error {
            entity.Validate(ctx, multiErr) // Domain model's built-in validation
            return nil
        }))

    // Add data integrity validation
    engine.AddRule(framework.NewValidationRule(framework.ValidationStageDataIntegrity, framework.ValidationPriorityHigh, 
        func(ctx context.Context, multiErr *errors.MultiError) error {
            return v.ValidateUniqueness(ctx, valCtx, entity, multiErr)
        }))

    // Execute all validation rules and return the resulting MultiError
    return engine.Validate(ctx)
}
```

### 3. Structure Your Validation Method - Use an Existing MultiError

If your validator accepts an existing `MultiError`, use the `ValidateInto` method:

```go
func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, entity *domain.Entity, multiErr *errors.MultiError) {
    // Create a validation engine
    engine := v.vef.CreateEngine()

    // Add validation rules
    engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh, 
        func(ctx context.Context, multiErr *errors.MultiError) error {
            entity.Validate(ctx, multiErr)
            return nil
        }))
    
    // Add more rules...

    // Execute all validation rules and add errors to the provided multiErr
    engine.ValidateInto(ctx, multiErr)
}
```

### 4. Structure Your Validation Method - Use Fluent API for Indexed Validation

For validating arrays or collections, use the fluent API for indexed validation:

```go
// Validating an array item with parent MultiError
func (v *MoveValidator) Validate(ctx context.Context, m *shipment.ShipmentMove, multiErr *errors.MultiError, idx int) {
    // Create validation engine with index information
    engine := v.vef.CreateEngine().
        ForField("moves").        // Field name for the array
        AtIndex(idx).             // Index of the current item
        WithParent(multiErr)      // Parent MultiError to add errors to

    // Add validation rules...
    engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh, 
        func(ctx context.Context, multiErr *errors.MultiError) error {
            m.Validate(ctx, multiErr)
            return nil
        }))

    // Validate nested arrays
    engine.AddRule(framework.NewValidationRule(framework.ValidationStageBusinessRules, framework.ValidationPriorityHigh, 
        func(ctx context.Context, multiErr *errors.MultiError) error {
            // For each nested item, create another engine with its own indexed context
            for stopIdx, stop := range m.Stops {
                stopEngine := v.vef.CreateEngine().
                    ForField("stops").
                    AtIndex(stopIdx).
                    WithParent(multiErr)
                
                // Add validation rules for the nested item
                stopEngine.AddRule(/* ... */)
                
                // Run validation - errors are added to parent MultiError
                // When WithParent is used, Validate() returns nil as errors are added directly to parent
                _ = stopEngine.Validate(ctx)
            }
            return nil
        }))

    // Run validation - errors are added to parent MultiError
    // When WithParent is used, Validate() returns nil as errors are added directly to parent
    _ = engine.Validate(ctx)
}

// Creating a new indexed MultiError without a parent
func (v *Validator) ValidateArrayItem(ctx context.Context, item *Item, idx int) *errors.MultiError {
    engine := v.vef.CreateEngine().
        ForField("items").
        AtIndex(idx)
        
    // Add validation rules...
    
    // Return the MultiError with indexed errors
    return engine.Validate(ctx)
}
```

### 5. Add Specialized Validation Rule Sets

For complex validation scenarios, create specialized rule additions:

```go
// In a separate compliance.go file
func AddComplianceRules(engine *framework.ValidationEngine, entity *domain.Entity) {
    engine.AddRule(framework.NewValidationRule(framework.ValidationStageCompliance, framework.ValidationPriorityHigh, 
        func(ctx context.Context, multiErr *errors.MultiError) error {
            // Regulatory compliance validation
            if !meetsRegulations(entity) {
                multiErr.Add("compliance", errors.ErrInvalid, "Entity does not meet regulatory requirements")
            }
            return nil
        }))
}

// Then in your validator:
func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, entity *domain.Entity) *errors.MultiError {
    engine := v.vef.CreateEngine()
    
    // Add basic rules...
    
    // Add compliance rules
    AddComplianceRules(engine, entity)
    
    return engine.Validate(ctx)
}
```

## Understanding Return Values

The validation engine follows specific patterns for return values:

1. **Without `WithParent`**:
   - `Validate()` returns a new MultiError if validation errors occur
   - `Validate()` returns nil if no validation errors occur
   - `ValidateInto()` adds errors to the provided MultiError but does not return anything

2. **With `WithParent`**:
   - `Validate()` returns nil regardless of validation status, as all errors are added to the parent MultiError
   - It's safe to discard the return value with `_ = engine.Validate(ctx)` when `WithParent` is used

This design supports nested validation while avoiding duplicate error handling code.

## Testing Validators

Use the provided mock factory for testing validators:

```go
func TestYourValidator(t *testing.T) {
    // Create a mock validation engine factory
    mockVef := &mocks.MockValidationEngineFactory{}
    
    validator := YourValidator{
        // Set dependencies...
        vef: mockVef,
    }
    
    // Test validation scenarios...
}
```

## Examples

### FMCSA Weight Compliance Validation

```go
func AddWeightComplianceRules(engine *framework.ValidationEngine, shp *shipment.Shipment) {
    engine.AddRule(framework.NewValidationRule(framework.ValidationStageCompliance, framework.ValidationPriorityHigh, 
        func(ctx context.Context, multiErr *errors.MultiError) error {
            // Calculate total weight from commodities
            var totalWeight int64
            for _, comm := range shp.Commodities {
                totalWeight += comm.Weight
            }
            
            // Check against 80,000 lbs interstate limit
            if totalWeight > 80000 {
                multiErr.Add("weight", errors.ErrInvalid, 
                    fmt.Sprintf("Total weight exceeds maximum allowed (80,000 lbs) for interstate transport. Current: %d lbs", totalWeight))
            }
            return nil
        }))
}
```

### Hazardous Materials Validation

```go
func AddHazmatComplianceRules(engine *framework.ValidationEngine, shp *shipment.Shipment) {
    engine.AddRule(framework.NewValidationRule(framework.ValidationStageCompliance, framework.ValidationPriorityHigh, 
        func(ctx context.Context, multiErr *errors.MultiError) error {
            // Check if shipment has hazmat commodities
            hasHazmat := false
            for _, comm := range shp.Commodities {
                if comm.Commodity != nil && comm.Commodity.HazardousMaterialID != nil {
                    hasHazmat = true
                    break
                }
            }
            
            // Enforce hazmat documentation requirements
            if hasHazmat {
                // Verify required documentation is present
                // ...
            }
            return nil
        }))
}
```

### Validating Nested Arrays with the Fluent API

```go
func (v *ShipmentValidator) ValidateMoves(ctx context.Context, shp *shipment.Shipment, multiErr *errors.MultiError) {
    if len(shp.Moves) == 0 {
        multiErr.Add("moves", errors.ErrInvalid, "Shipment must have at least one move")
        return
    }

    for idx, move := range shp.Moves {
        // Create a validation engine with index information
        engine := v.vef.CreateEngine().
            ForField("moves").
            AtIndex(idx).
            WithParent(multiErr)
        
        // Add validation rules for the move
        engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh, 
            func(ctx context.Context, multiErr *errors.MultiError) error {
                move.Validate(ctx, multiErr)
                return nil
            }))
        
        // Validate nested stops
        engine.AddRule(framework.NewValidationRule(framework.ValidationStageBusinessRules, framework.ValidationPriorityHigh, 
            func(ctx context.Context, multiErr *errors.MultiError) error {
                for stopIdx, stop := range move.Stops {
                    // Create a validation engine for the stop
                    stopEngine := v.vef.CreateEngine().
                        ForField("stops").
                        AtIndex(stopIdx).
                        WithParent(multiErr)
                    
                    // Add validation rules for the stop
                    stopEngine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh, 
                        func(ctx context.Context, multiErr *errors.MultiError) error {
                            stop.Validate(ctx, multiErr)
                            return nil
                        }))
                    
                    // Run validation and discard return value (it's always nil when using WithParent)
                    _ = stopEngine.Validate(ctx)
                }
                return nil
            }))
            
        // Run validation and discard return value (it's always nil when using WithParent)
        _ = engine.Validate(ctx)
    }
}
```

## Best Practices

1. **Stage Organization**: Place validations in appropriate stages for consistent execution order
2. **Priority Assignment**: Assign higher priority to critical validations
3. **Rule Grouping**: Group related validations into separate functions for maintainability
4. **Error Messages**: Provide clear, actionable error messages
5. **Early Returns**: Avoid unnecessary validation when dependencies aren't available
6. **Context Usage**: Pass context through to enable timeouts and cancellation
7. **Testing**: Create dedicated tests for complex validation rules
8. **Use Fluent API**: For indexed validation, use the fluent API with `ForField`, `AtIndex`, and `WithParent`
9. **Validate in Order**: Validate parent objects before child objects to maintain context
10. **Discard Return Value with WithParent**: When using `WithParent`, use `_ = engine.Validate(ctx)` as the return value is always nil

## Integration with Dependency Injection

The framework is integrated with Uber's fx for dependency injection:

```go
// In your module definition:
var Module = fx.Module("validators", 
    fx.Provide(
        // Your validators...
        framework.ProvideValidationEngineFactory,
        framework.ProvideLifecycle,
    ),
    fx.Options(
        framework.Module,
    ),
)
