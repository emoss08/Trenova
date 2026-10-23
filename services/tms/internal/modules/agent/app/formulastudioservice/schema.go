package formulastudioservice

const schemaType = "type"

func objectSchema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{
		schemaType:             "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func arraySchema(items map[string]any, description string) map[string]any {
	schema := map[string]any{schemaType: "array", "items": items}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func typedField(kind, description string) map[string]any {
	field := map[string]any{schemaType: kind}
	if description != "" {
		field["description"] = description
	}
	return field
}
