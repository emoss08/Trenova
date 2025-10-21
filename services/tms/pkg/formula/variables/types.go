package variables

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/formulatypes"
)

// VariableSource identifies WHERE the variable data comes from
//
// IMPORTANT: This is completely different from FormulaTemplate categories!
// - FormulaTemplate categories = CALCULATION METHODS (e.g., "DistanceBased", "WeightBased")
// - VariableSource (here) = DATA SOURCES that variables pull from
//
// Example: A "DistanceBased" formula template might use variables from multiple sources:
// - SourceShipment for distance and weight data
// - SourceHazmat for hazmat surcharges
// - SourceEnvironment for temperature-based adjustments
type VariableSource string

const (
	SourceShipment = VariableSource(
		"shipment",
	) // Data from the shipment entity (weight, distance, stops, etc.)
	SourceEnvironment = VariableSource(
		"environment",
	) // Environmental conditions (temperature ranges, etc.)
	SourceHazmat = VariableSource(
		"hazmat",
	) // Hazardous material classifications and requirements
	SourceEquipment = VariableSource(
		"equipment",
	) // Equipment specifications (tractor type, trailer type, etc.)
	SourceCustom = VariableSource("custom") // User-defined custom variables
)

func (s VariableSource) String() string {
	return string(s)
}

func (s VariableSource) IsValid() bool {
	switch s {
	case SourceShipment, SourceEnvironment, SourceHazmat, SourceEquipment, SourceCustom:
		return true
	default:
		return false
	}
}

type Variable interface {
	Name() string
	Description() string
	Type() formulatypes.ValueType
	Category() string
	Resolve(ctx VariableContext) (any, error)
	Validate(value any) error
}

type VariableContext interface {
	GetEntity() any
	GetField(path string) (any, error)
	GetComputed(function string) (any, error)
	GetMetadata() map[string]any
}

type VariableDefinition struct {
	name        string
	description string
	valueType   formulatypes.ValueType
	source      VariableSource
	resolver    VariableResolver
	validator   VariableValidator
}

type VariableResolver func(ctx VariableContext) (any, error)

type VariableValidator func(value any) error

func NewVariable(
	name, description string,
	valueType formulatypes.ValueType,
	source VariableSource,
	resolver VariableResolver,
) *VariableDefinition {
	return &VariableDefinition{
		name:        name,
		description: description,
		valueType:   valueType,
		source:      source,
		resolver:    resolver,
		validator:   defaultValidator(valueType),
	}
}

func NewVariableWithValidator(
	name, description string,
	valueType formulatypes.ValueType,
	source VariableSource,
	resolver VariableResolver,
	validator VariableValidator,
) *VariableDefinition {
	return &VariableDefinition{
		name:        name,
		description: description,
		valueType:   valueType,
		source:      source,
		resolver:    resolver,
		validator:   validator,
	}
}

func (v *VariableDefinition) Name() string {
	return v.name
}

func (v *VariableDefinition) Description() string {
	return v.description
}

func (v *VariableDefinition) Type() formulatypes.ValueType {
	return v.valueType
}

func (v *VariableDefinition) Category() string {
	return v.source.String()
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

func defaultValidator(valueType formulatypes.ValueType) VariableValidator {
	return func(value any) error {
		if value == nil {
			return nil // Allow nil values by default
		}

		// TODO(wolfred): add missing case: formula.ValueTypeDate, formulatypes.ValueType, formulatypes.ValueTypeAny
		switch valueType { //nolint:exhaustive // reference above todo
		case formulatypes.ValueTypeNumber:
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

		case formulatypes.ValueTypeString:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("expected string, got %T", value)
			}

		case formulatypes.ValueTypeBoolean:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("expected boolean, got %T", value)
			}

		case formulatypes.ValueTypeArray:
			// Check if it's a slice or array
			switch value.(type) {
			case []any, []string, []float64, []int:
				return nil
			default:
				return fmt.Errorf("expected array, got %T", value)
			}

		case formulatypes.ValueTypeObject:
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
