package jsonschemautils

const (
	keyType                 = "type"
	keyProperties           = "properties"
	keyRequired             = "required"
	keyAdditionalProperties = "additionalProperties"
	keyEnum                 = "enum"
	keyDescription          = "description"
	keyItems                = "items"
	keyMaxItems             = "maxItems"
	keyMaxLength            = "maxLength"
	keyMinimum              = "minimum"
	keyMaximum              = "maximum"

	typeObject = "object"
	typeString = "string"
	typeArray  = "array"
	typeNumber = "number"
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

func String(maxLength int) map[string]any {
	schema := map[string]any{keyType: typeString}
	if maxLength > 0 {
		schema[keyMaxLength] = maxLength
	}

	return schema
}

func Number(minimum, maximum float64) map[string]any {
	return map[string]any{
		keyType:    typeNumber,
		keyMinimum: minimum,
		keyMaximum: maximum,
	}
}

func Array(items map[string]any, maxItems int) map[string]any {
	schema := map[string]any{
		keyType:  typeArray,
		keyItems: items,
	}
	if maxItems > 0 {
		schema[keyMaxItems] = maxItems
	}

	return schema
}
