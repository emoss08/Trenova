package configspec

import "strings"

type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeURL      FieldType = "url"
	FieldTypePassword FieldType = "password"
	FieldTypeSelect   FieldType = "select"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeNumber   FieldType = "number"
	FieldTypeMulti    FieldType = "multi-select"
)

type Field struct {
	Key         string    `json:"key"`
	Label       string    `json:"label"`
	Type        FieldType `json:"type"`
	Required    bool      `json:"required"`
	Sensitive   bool      `json:"sensitive"`
	Placeholder string    `json:"placeholder,omitempty"`
	HelpText    string    `json:"helpText,omitempty"`
	Default     string    `json:"default,omitempty"`
	Options     []string  `json:"options,omitempty"`
}

type FieldValue struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	HasValue bool   `json:"hasValue"`
}

func ReadString(configuration map[string]any, key string) string {
	if len(configuration) == 0 {
		return ""
	}

	value, ok := configuration[key]
	if !ok || value == nil {
		return ""
	}

	stringValue, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(stringValue)
}

func HasRequired(configuration map[string]any, fields []Field) bool {
	return len(MissingRequired(configuration, fields)) == 0
}

func MissingRequired(configuration map[string]any, fields []Field) []string {
	missing := make([]string, 0, len(fields))
	for idx := range fields {
		if fields[idx].Required && ReadString(configuration, fields[idx].Key) == "" {
			missing = append(missing, fields[idx].Key)
		}
	}

	return missing
}

func MissingRequiredValues(values map[string]string, fields []Field) []string {
	missing := make([]string, 0, len(fields))
	for idx := range fields {
		if fields[idx].Required && strings.TrimSpace(values[fields[idx].Key]) == "" {
			missing = append(missing, fields[idx].Key)
		}
	}

	return missing
}

func Values(configuration map[string]any, fields []Field) []FieldValue {
	values := make([]FieldValue, 0, len(fields))
	for idx := range fields {
		field := &fields[idx]
		stored := ReadString(configuration, field.Key)
		value := FieldValue{Key: field.Key, HasValue: stored != ""}
		if !field.Sensitive {
			value.Value = stored
		}
		values = append(values, value)
	}

	return values
}

func ByKey(fields []Field) map[string]*Field {
	byKey := make(map[string]*Field, len(fields))
	for idx := range fields {
		byKey[fields[idx].Key] = &fields[idx]
	}

	return byKey
}
