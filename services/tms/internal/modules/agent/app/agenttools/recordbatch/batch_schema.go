package recordbatch

import (
	"github.com/emoss08/trenova/pkg/toolschema"
)

func BulkIDsSchema(batch *RecordBatch, description string, extra map[string]any) map[string]any {
	properties := map[string]any{batch.Param: batch.property(description)}
	for name, property := range extra {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             []string{batch.Param},
		toolschema.KeyAdditionalProperties: false,
	}
}
