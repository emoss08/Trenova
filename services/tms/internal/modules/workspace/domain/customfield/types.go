package customfield

type FieldType string

const (
	FieldTypeText        FieldType = "text"
	FieldTypeNumber      FieldType = "number"
	FieldTypeDate        FieldType = "date"
	FieldTypeBoolean     FieldType = "boolean"
	FieldTypeSelect      FieldType = "select"
	FieldTypeMultiSelect FieldType = "multiSelect"
)

func (ft FieldType) IsValid() bool {
	switch ft {
	case FieldTypeText,
		FieldTypeNumber,
		FieldTypeDate,
		FieldTypeBoolean,
		FieldTypeSelect,
		FieldTypeMultiSelect:
		return true
	default:
		return false
	}
}

func (ft FieldType) String() string {
	return string(ft)
}

func (ft FieldType) RequiresOptions() bool {
	return ft == FieldTypeSelect || ft == FieldTypeMultiSelect
}

type SelectOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

type ValidationRules struct {
	MinLength *int    `json:"minLength,omitempty"`
	MaxLength *int    `json:"maxLength,omitempty"`
	Min       *int    `json:"min,omitempty"`
	Max       *int    `json:"max,omitempty"`
	Pattern   *string `json:"pattern,omitempty"`
}

type UIAttributes struct {
	Placeholder string `json:"placeholder,omitempty"`
	HelpText    string `json:"helpText,omitempty"`
	Width       string `json:"width,omitempty"`
}
