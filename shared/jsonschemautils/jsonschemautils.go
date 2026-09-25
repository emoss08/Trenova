package jsonschemautils

const (
	keyType                 = "type"
	keyProperties           = "properties"
	keyRequired             = "required"
	keyAdditionalProperties = "additionalProperties"
	keyEnum                 = "enum"
	keyDescription          = "description"

	typeObject = "object"
	typeString = "string"
)

func Object(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		keyType:                 typeObject,
		keyProperties:           properties,
		keyAdditionalProperties: false,
	}
	if len(required) > 0 {
		schema[keyRequired] = required
	}

	return schema
}

func Enum(description string, values ...string) map[string]any {
	return map[string]any{
		keyType:        typeString,
		keyEnum:        values,
		keyDescription: description,
	}
}
