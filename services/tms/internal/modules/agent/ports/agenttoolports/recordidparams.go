package agenttoolports

import "github.com/emoss08/trenova/pkg/toolschema"

// RecordIDParams is the arguments a get tool takes to read one record: its
// id under the one parameter that names it, as a list when the parameter
// takes several. It is false for a tool whose schema names no record.
func RecordIDParams(schema map[string]any, id string) (map[string]any, bool) {
	param, ok := recordIDParam(schema)
	if !ok {
		return nil, false
	}

	return map[string]any{param: recordIDArgument(schema, param, id)}, true
}

func recordIDArgument(schema map[string]any, param, id string) any {
	properties, _ := schema[toolschema.KeyProperties].(map[string]any)
	property, _ := properties[param].(map[string]any)
	if kind, _ := property[toolschema.KeyType].(string); kind == toolschema.TypeArray {
		return []any{id}
	}

	return id
}

func recordIDParam(schema map[string]any) (string, bool) {
	if required, _ := schema[toolschema.KeyRequired].([]string); len(required) == 1 {
		return required[0], true
	}

	properties, _ := schema[toolschema.KeyProperties].(map[string]any)
	found := ""
	for name, raw := range properties {
		property, _ := raw.(map[string]any)
		items, _ := property[toolschema.KeyItems].(map[string]any)
		if toolschema.RecordOf(property) == "" && toolschema.RecordOf(items) == "" {
			continue
		}
		if found != "" {
			return "", false
		}
		found = name
	}

	return found, found != ""
}
