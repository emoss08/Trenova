package agenttoolschema

import "github.com/emoss08/trenova/pkg/toolschema"

const MaxReasonLength = 600

func Reason(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        "string",
		toolschema.KeyDescription: description,
		toolschema.KeyMaxLength:   MaxReasonLength,
	}
}
