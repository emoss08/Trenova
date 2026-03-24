package formulatypes

type VariableValueType string

const (
	VariableValueTypeNumber  = VariableValueType("Number")
	VariableValueTypeString  = VariableValueType("String")
	VariableValueTypeBoolean = VariableValueType("Boolean")
	VariableValueTypeDate    = VariableValueType("Date")
	VariableValueTypeArray   = VariableValueType("Array")
	VariableValueTypeObject  = VariableValueType("Object")
	VariableValueTypeAny     = VariableValueType("Any") // used for runtime-determined types
)

type VariableSource string

const (
	VariableSourceShipment = VariableSource("Shipment")
)

type VariableContext interface {
	GetEntity() any
	GetField(path string) (any, error)
	GetComputed(function string) (any, error)
	GetMetadata() map[string]any
}

type Variable interface {
	Name() string
	Description() string
	Type() VariableValueType
	Category() string
	Resolve(ctx VariableContext) (any, error)
	Validate(value any) error
}

type VariableDefinition struct {
	Name         string            `json:"name"`
	Type         VariableValueType `json:"type"`
	Description  string            `json:"description"`
	Required     bool              `json:"required"`
	DefaultValue any               `json:"defaultValue,omitempty"`
	Source       string            `json:"source,omitempty"`
}
