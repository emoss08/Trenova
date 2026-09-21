package toolschema

import (
	"fmt"
	"sort"

	"github.com/emoss08/trenova/shared/stringutils"
)

// Kind is the control a field takes when a person edits it.
type Kind string

const (
	KindText      Kind = "Text"
	KindMultiline Kind = "Multiline"
	KindInteger   Kind = "Integer"
	KindNumber    Kind = "Number"
	KindBoolean   Kind = "Boolean"
	KindChoice    Kind = "Choice"
	KindList      Kind = "List"
	KindJSON      Kind = "JSON"
)

// Field is one top-level parameter of a tool as a person can edit it: what
// it is called, what it means, what kind of value it takes and the bounds
// the tool's schema puts on it. It is derived from the tool's JSON schema,
// so a tool that grows a parameter is editable without anyone adding it
// here, and a value that does not fit is refused by the same schema.
type Field struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Kind        Kind     `json:"kind"`
	Required    bool     `json:"required"`
	Options     []string `json:"options"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	MaxLength   int      `json:"maxLength,omitempty"`
}

// multilineThreshold is the declared length past which a string is prose.
const multilineThreshold = 160

// multilineNames are parameters that are prose whatever their length says.
var multilineNames = map[string]struct{}{
	"message": {}, "body": {}, "note": {}, "notes": {}, "comment": {}, "comments": {},
	"description": {}, "reason": {}, "summary": {}, "text": {}, "instructions": {},
	"rationale": {}, "content": {}, "remarks": {},
}

// Fields lists a schema's top-level properties as editable fields, required
// ones first and then by name, since a map carries no order of its own.
func Fields(schema map[string]any) []Field {
	properties, _ := schema["properties"].(map[string]any)
	required := requiredSet(schema)

	fields := make([]Field, 0, len(properties))
	for name, raw := range properties {
		property, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		fields = append(fields, fieldFrom(name, property, required[name]))
	}

	sort.SliceStable(fields, func(i, j int) bool {
		if fields[i].Required != fields[j].Required {
			return fields[i].Required
		}

		return fields[i].Name < fields[j].Name
	})

	return fields
}

func requiredSet(schema map[string]any) map[string]bool {
	required := make(map[string]bool)
	switch names := schema["required"].(type) {
	case []string:
		for _, name := range names {
			required[name] = true
		}
	case []any:
		for _, name := range names {
			if text, ok := name.(string); ok {
				required[text] = true
			}
		}
	}

	return required
}

func fieldFrom(name string, property map[string]any, required bool) Field {
	field := Field{
		Name:        name,
		Label:       stringutils.HumanizeCamelCaseSentence(name),
		Description: stringOf(property["description"]),
		Required:    required,
		Options:     []string{},
		Minimum:     numberOf(property["minimum"]),
		Maximum:     numberOf(property["maximum"]),
		MaxLength:   intOf(property["maxLength"]),
	}

	if options := optionsOf(property["enum"]); len(options) > 0 {
		field.Kind = KindChoice
		field.Options = options

		return field
	}

	switch typeOf(property["type"]) {
	case "string":
		field.Kind = KindText
		if _, prose := multilineNames[name]; prose || field.MaxLength >= multilineThreshold {
			field.Kind = KindMultiline
		}
	case "integer":
		field.Kind = KindInteger
	case "number":
		field.Kind = KindNumber
	case "boolean":
		field.Kind = KindBoolean
	case "array":
		field.Kind = listKind(property, &field)
	default:
		field.Kind = KindJSON
	}

	return field
}

// listKind is a list of strings, with its choices when the items are an
// enum; a list of anything else is edited as JSON.
func listKind(property map[string]any, field *Field) Kind {
	items, ok := property["items"].(map[string]any)
	if !ok {
		return KindJSON
	}
	if options := optionsOf(items["enum"]); len(options) > 0 {
		field.Options = options

		return KindList
	}
	if typeOf(items["type"]) == "string" {
		return KindList
	}

	return KindJSON
}

// typeOf reads a schema type, which may be one name or a list of names with
// "null" among them.
func typeOf(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case []string:
		for _, name := range value {
			if name != "null" {
				return name
			}
		}
	case []any:
		for _, name := range value {
			if text, ok := name.(string); ok && text != "null" {
				return text
			}
		}
	}

	return ""
}

func optionsOf(raw any) []string {
	switch values := raw.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		options := make([]string, 0, len(values))
		for _, value := range values {
			options = append(options, fmt.Sprint(value))
		}

		return options
	}

	return nil
}

func stringOf(raw any) string {
	text, _ := raw.(string)

	return text
}

func numberOf(raw any) *float64 {
	var value float64
	switch number := raw.(type) {
	case float64:
		value = number
	case float32:
		value = float64(number)
	case int:
		value = float64(number)
	case int64:
		value = float64(number)
	case int32:
		value = float64(number)
	default:
		return nil
	}

	return &value
}

func intOf(raw any) int {
	if number := numberOf(raw); number != nil {
		return int(*number)
	}

	return 0
}
