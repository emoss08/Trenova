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

func TestBoolValue(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"enabled":  true,
		"disabled": "false",
		"padded":   " true ",
		"auto":     "auto",
		"number":   1,
	}

	value, ok := BoolValue(input, "enabled")
	assert.True(t, ok)
	assert.True(t, value)

	value, ok = BoolValue(input, "disabled")
	assert.True(t, ok)
	assert.False(t, value)

	value, ok = BoolValue(input, "padded")
	assert.True(t, ok)
	assert.True(t, value)

	_, ok = BoolValue(input, "auto")
	assert.False(t, ok)

	_, ok = BoolValue(input, "number")
	assert.False(t, ok)

	_, ok = BoolValue(input, "missing")
	assert.False(t, ok)

	_, ok = BoolValue(nil, "enabled")
	assert.False(t, ok)
}

func TestCloneShallow(t *testing.T) {
	t.Parallel()

	input := map[string]any{"value": "one"}
	output := CloneShallow(input)
	output["value"] = "two"

	assert.Equal(t, "one", input["value"])
	assert.Equal(t, "two", output["value"])
}

func TestWithoutFuncValues(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"amount": 12.5,
		"label":  "linehaul",
		"nested": map[string]any{"k": "v"},
		"resolver": func(string, any) (float64, error) {
			return 0, nil
		},
	}

	output := WithoutFuncValues(input)

	assert.Len(t, output, 3)
	assert.Equal(t, 12.5, output["amount"])
	assert.Equal(t, "linehaul", output["label"])
	assert.NotContains(t, output, "resolver")

	assert.Nil(t, WithoutFuncValues(nil))
	assert.Nil(t, WithoutFuncValues(map[string]any{}))
}
