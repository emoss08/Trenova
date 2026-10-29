package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/stretchr/testify/assert"
)

func TestChangeSize_CountsTheLargestRecordList(t *testing.T) {
	t.Parallel()

	schema := runtimetest.BatchSchema()
	schema["properties"].(map[string]any)["shipment_ids"] = map[string]any{
		"type": "array", "items": map[string]any{"type": "string"},
	}
	schema["properties"].(map[string]any)["lines"] = map[string]any{
		"type": "array", "items": map[string]any{"type": "object"},
	}

	tests := []struct {
		name string
		args map[string]any
		want int
	}{
		{"a marked record set", map[string]any{"billingQueueItemIds": runtimetest.IDs(3)}, 3},
		{"ids by name", map[string]any{"shipment_ids": []string{"a", "b"}}, 2},
		{"the larger list wins", map[string]any{
			"billingQueueItemIds": runtimetest.IDs(2), "shipment_ids": []string{"a", "b", "c", "d"},
		}, 4},
		{"repeats count once", map[string]any{"billingQueueItemIds": []any{"x", "x", " x "}}, 1},
		{"lines are not records", map[string]any{"lines": []any{"a", "b", "c"}}, 0},
		{"nothing listed", map[string]any{"note": "hi"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, changeSize(schema, tt.args))
		})
	}
}
