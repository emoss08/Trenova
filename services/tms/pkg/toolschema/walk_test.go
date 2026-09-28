package toolschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalk_VisitsEverySubschemaWithItsPath(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipment": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"moves": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"stops": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "object"},
								},
							},
						},
					},
				},
			},
			"amount": map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string"},
					map[string]any{"type": "number"},
				},
			},
			"extra": map[string]any{
				"type":                 "object",
				"additionalProperties": map[string]any{"type": "string"},
			},
		},
		"$defs": map[string]any{
			"ref": map[string]any{"type": "object"},
		},
	}

	var paths []string
	Walk(schema, func(path string, node map[string]any) {
		assert.NotNil(t, node)
		paths = append(paths, path)
	})

	assert.Equal(t, []string{
		"",
		"amount",
		"amount.anyOf[0]",
		"amount.anyOf[1]",
		"extra",
		"extra.additionalProperties",
		"shipment",
		"shipment.moves",
		"shipment.moves[]",
		"shipment.moves[].stops",
		"shipment.moves[].stops[]",
		"$defs[ref]",
	}, paths)

	Walk(nil, func(string, map[string]any) { t.Fatal("nothing to walk") })
}
