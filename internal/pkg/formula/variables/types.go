package variables

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/types/formula"
)

// * Variable represents a formula variable that can be used in calculations
type Variable interface {
	// Name returns the variable name as used in formulas
	Name() string
	
	// Description returns a human-readable description
	Description() string
	
	// Type returns the value type of this variable
	Type() formula.ValueType
	
	// Category returns the variable category for grouping
	Category() string
	
	// Resolve extracts the variable value from the given context
	Resolve(ctx VariableContext) (any, error)
	
	// Validate checks if the resolved value is valid
	Validate(value any) error
}

// * VariableContext provides data for variable resolution
type VariableContext interface {
	// GetEntity returns the primary entity (e.g., Shipment)
	GetEntity() any
	
	// GetField retrieves a field value by path
	GetField(path string) (any, error)
	
	// GetComputed retrieves a computed value by function name
	GetComputed(function string) (any, error)
	
	// GetMetadata returns context metadata
	GetMetadata() map[string]any
}

// * VariableDefinition defines a variable's metadata and resolution
type VariableDefinition struct {
	name        string
	description string
	valueType   formula.ValueType
	category    string
	resolver    VariableResolver
	validator   VariableValidator
}

// * VariableResolver extracts the variable value from context
type VariableResolver func(ctx VariableContext) (any, error)

// * VariableValidator validates a resolved value
type VariableValidator func(value any) error

// * NewVariable creates a new variable definition
func NewVariable(name, description string, valueType formula.ValueType, category string, resolver VariableResolver) *VariableDefinition {
	return &VariableDefinition{
		name:        name,
		description: description,
		valueType:   valueType,
		category:    category,
		resolver:    resolver,
		validator:   defaultValidator(valueType),
	}
}

// * NewVariableWithValidator creates a new variable with custom validation
func NewVariableWithValidator(
	name, description string,
	valueType formula.ValueType,
	category string,
	resolver VariableResolver,
	validator VariableValidator,
) *VariableDefinition {
	return &VariableDefinition{
		name:        name,
		description: description,
		valueType:   valueType,
		category:    category,
		resolver:    resolver,
		validator:   validator,
	}
}

// Implement Variable interface
func (v *VariableDefinition) Name() string {
	return v.name
}

func (v *VariableDefinition) Description() string {
	return v.description
}

func (v *VariableDefinition) Type() formula.ValueType {
	return v.valueType
}

func (v *VariableDefinition) Category() string {
	return v.category
}

func (v *VariableDefinition) Resolve(ctx VariableContext) (any, error) {
	if v.resolver == nil {
		return nil, fmt.Errorf("no resolver defined for variable %s", v.name)
	}
	return v.resolver(ctx)
}

func (v *VariableDefinition) Validate(value any) error {
	if v.validator == nil {
		return nil
	}
	return v.validator(value)
}

// * defaultValidator returns a basic validator for the given type
func defaultValidator(valueType formula.ValueType) VariableValidator {
	return func(value any) error {
		if value == nil {
			return nil // Allow nil values by default
		}
		
		switch valueType {
		case formula.ValueTypeNumber:
			switch v := value.(type) {
			case float64, float32:
				return nil
			case int, int8, int16, int32, int64:
				return nil
			case uint, uint8, uint16, uint32, uint64:
				return nil
			default:
				return fmt.Errorf("expected number, got %T", v)
			}
			
		case formula.ValueTypeString:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("expected string, got %T", value)
			}
			
		case formula.ValueTypeBoolean:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("expected boolean, got %T", value)
			}
			
		case formula.ValueTypeArray:
			// Check if it's a slice or array
			switch value.(type) {
			case []any, []string, []float64, []int:
				return nil
			default:
				return fmt.Errorf("expected array, got %T", value)
			}
			
		case formula.ValueTypeObject:
			// Check if it's a map or struct
			switch value.(type) {
			case map[string]any:
				return nil
			default:
				// For structs, we accept any non-primitive type
				return nil
			}
		}
		
		return nil
	}
}

// * VariableCategory constants
const (
	CategoryShipment    = "shipment"
	CategoryEnvironment = "environment"
	CategoryHazmat      = "hazmat"
	CategoryEquipment   = "equipment"
	CategoryPricing     = "pricing"
	CategoryCustom      = "custom"
)