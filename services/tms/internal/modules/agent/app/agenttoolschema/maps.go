package agenttoolschema

import "github.com/emoss08/trenova/pkg/toolschema"

// StringMap is an object of names to text values, each at most maxLength
// long: values a tool types somewhere by the name the destination gives them.
// It is closed by the schema its values follow.
func StringMap(description string, maxLength int) map[string]any {
	return map[string]any{
		toolschema.KeyType:        "object",
		toolschema.KeyDescription: description,
		toolschema.KeyAdditionalProperties: map[string]any{
			toolschema.KeyType:      "string",
			toolschema.KeyMaxLength: maxLength,
		},
	}
}
