package modeladapter

// anthropicUnsupportedBounds are the constraints Anthropic's structured output
// refuses outright ("For 'array' type, property 'maxItems' is not supported"),
// which sent every such call on to the next provider. Only the copy sent to
// Anthropic loses them; the shape the reply must take is unchanged.
var anthropicUnsupportedBounds = []string{
	"maxItems",
	"minimum",
	"maximum",
	"exclusiveMinimum",
	"exclusiveMaximum",
	"multipleOf",
	"minLength",
	"maxLength",
}

var anthropicSchemaMaps = map[string]bool{
	"properties":        true,
	"patternProperties": true,
	"$defs":             true,
	"definitions":       true,
}

func anthropicOutputSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	return anthropicSchema(schema)
}

func anthropicSchema(schema map[string]any) map[string]any {
	out := make(map[string]any, len(schema))
	for key, value := range schema {
		if named, isMap := value.(map[string]any); isMap && anthropicSchemaMaps[key] {
			out[key] = anthropicNamedSchemas(named)

			continue
		}
		out[key] = anthropicSchemaValue(value)
	}
	for _, key := range anthropicUnsupportedBounds {
		delete(out, key)
	}
	if minItems, ok := out["minItems"]; ok && !atMostOne(minItems) {
		delete(out, "minItems")
	}

	return out
}

func anthropicNamedSchemas(named map[string]any) map[string]any {
	out := make(map[string]any, len(named))
	for name, value := range named {
		out[name] = anthropicSchemaValue(value)
	}

	return out
}

func anthropicSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return anthropicSchema(typed)
	case []any:
		out := make([]any, len(typed))
		for idx, inner := range typed {
			out[idx] = anthropicSchemaValue(inner)
		}

		return out
	default:
		return value
	}
}

func atMostOne(value any) bool {
	switch typed := value.(type) {
	case int:
		return typed <= 1
	case int64:
		return typed <= 1
	case float64:
		return typed <= 1
	default:
		return false
	}
}
