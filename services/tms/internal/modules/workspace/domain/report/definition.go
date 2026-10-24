package report

import (
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel/analytics/ir"
)

var errNoDefinition = errors.New("a report definition is required")

func DecodeDefinition(value any) (*ir.Definition, error) {
	var encoded []byte
	switch typed := value.(type) {
	case nil:
		return nil, errNoDefinition
	case []byte:
		encoded = typed
	case string:
		encoded = []byte(typed)
	case map[string]any:
		if _, versioned := typed["version"]; !versioned {
			typed = maps.Clone(typed)
			typed["version"] = ir.Version
		}
		raw, err := sonic.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("encode report definition: %w", err)
		}
		encoded = raw
	default:
		raw, err := sonic.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("encode report definition: %w", err)
		}
		encoded = raw
	}

	return ir.Decode(encoded)
}

var definitionSchema = sync.OnceValues(func() (map[string]any, error) {
	var schema map[string]any
	if err := sonic.Unmarshal(ir.Schema(), &schema); err != nil {
		return nil, fmt.Errorf("decode the report definition schema: %w", err)
	}
	return schema, nil
})

func DefinitionJSONSchema() map[string]any {
	schema, err := definitionSchema()
	if err != nil {
		return map[string]any{
			"type":        "object",
			"description": "A keel analytics report definition.",
		}
	}
	return maps.Clone(schema)
}

const definitionToolNote = "A report in keel analytics IR version 2, whole: its entity, " +
	"columns, filters and sort all go here and nowhere else. Fields are dotted paths from " +
	"the entity, such as customer.name. A dimension column groups, a measure column " +
	"aggregates, and a report with only dimensions lists rows."

func DefinitionToolSchema() map[string]any {
	schema := DefinitionJSONSchema()
	schema["description"] = definitionToolNote
	return schema
}

func ParameterNames(def *ir.Definition) []string {
	if def == nil {
		return nil
	}
	names := make([]string, 0, len(def.Parameters))
	for i := range def.Parameters {
		names = append(names, def.Parameters[i].Name)
	}
	return names
}

func CloneDefinition(def *ir.Definition) (*ir.Definition, error) {
	if def == nil {
		return nil, errNoDefinition
	}
	encoded, err := ir.Encode(def)
	if err != nil {
		return nil, fmt.Errorf("encode report definition: %w", err)
	}
	return ir.Decode(encoded)
}
