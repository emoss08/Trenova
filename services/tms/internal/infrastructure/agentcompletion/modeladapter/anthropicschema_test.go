package modeladapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnthropicOutputSchema_DropsTheBoundsAnthropicRefuses(t *testing.T) {
	t.Parallel()

	sent := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"chips": map[string]any{
				"type":     "array",
				"minItems": 2,
				"maxItems": 4,
				"items":    map[string]any{"type": "string", "maxLength": 40},
			},
			"score":   map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"tags":    map[string]any{"type": "array", "minItems": float64(1)},
			"maximum": map[string]any{"type": "integer"},
		},
		"required":             []any{"chips"},
		"additionalProperties": false,
	}

	got := anthropicOutputSchema(sent)

	properties := got["properties"].(map[string]any)
	chips := properties["chips"].(map[string]any)
	assert.NotContains(t, chips, "maxItems")
	assert.NotContains(t, chips, "minItems", "Anthropic takes minItems only as 0 or 1")
	assert.NotContains(t, chips["items"].(map[string]any), "maxLength")
	assert.NotContains(t, properties["score"].(map[string]any), "minimum")
	assert.Equal(t, float64(1), properties["tags"].(map[string]any)["minItems"])
	assert.Contains(t, properties, "maximum", "a field named like a keyword is kept")
	assert.Equal(t, false, got["additionalProperties"])
	assert.Equal(t, []any{"chips"}, got["required"])
	assert.Contains(t, sent["properties"].(map[string]any)["chips"], "maxItems",
		"the caller's schema is not changed")
}
