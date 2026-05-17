package maputils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath(t *testing.T) {
	t.Parallel()

	root := map[string]any{
		"shipment": map[string]any{
			"moves": []any{
				map[string]any{
					"stops": []any{
						map[string]any{"name": "Chicago"},
					},
				},
			},
		},
	}

	assert.Equal(t, "Chicago", Path(root, "shipment.moves.0.stops.0.name"))
	assert.Nil(t, Path(root, "shipment.moves.x"))
	assert.Nil(t, Path(root, "shipment.moves.5"))
}

func TestCloneShallow(t *testing.T) {
	t.Parallel()

	input := map[string]any{"value": "one"}
	output := CloneShallow(input)
	output["value"] = "two"

	assert.Equal(t, "one", input["value"])
	assert.Equal(t, "two", output["value"])
}
