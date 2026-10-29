package toolparams

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
)

func BooleanProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeBoolean,
		toolschema.KeyDescription: description,
	}
}

func IntegerProperty(description string, minimum, maximum int) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeInteger,
		toolschema.KeyMinimum:     minimum,
		toolschema.KeyMaximum:     maximum,
		toolschema.KeyDescription: description,
	}
}

func ObjectParams(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
	if len(required) > 0 {
		schema[toolschema.KeyRequired] = required
	}

	return schema
}

func AmountProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}

func WorkerProperty() map[string]any {
	return agenttoolschema.RecordIDText(
		permission.ResourceWorker,
		"The driver, from worker_list. Never guess one.",
	)
}

func StringProperty(description string, maxLength int) map[string]any {
	property := map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
	if maxLength > 0 {
		property[toolschema.KeyMaxLength] = maxLength
	}

	return property
}
