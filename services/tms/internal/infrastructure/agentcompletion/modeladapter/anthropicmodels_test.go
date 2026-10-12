package modeladapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// What a Claude model takes for thinking depends on the model, and the model
// is configured as a free-text id in whichever form the endpoint uses: the
// Claude API's plain id, Bedrock's prefixed one, Vertex's dated one, or a name
// a gateway made up.
func TestAnthropicTraits(t *testing.T) {
	t.Parallel()

	cases := []struct {
		model string
		want  anthropicModel
	}{
		{"claude-opus-5-5", anthropicModel{adaptive: true, thinksAlways: true, bindsPrefix: true}},
		{"anthropic.claude-opus-5-5", anthropicModel{adaptive: true, thinksAlways: true, bindsPrefix: true}},
		{"us.anthropic.claude-opus-5-5", anthropicModel{adaptive: true, thinksAlways: true, bindsPrefix: true}},
		{"claude-opus-5-5@20260901", anthropicModel{adaptive: true, thinksAlways: true, bindsPrefix: true}},
		{"claude-opus-5-6", anthropicModel{adaptive: true, thinksAlways: true, bindsPrefix: true}},
		{"claude-opus-6", anthropicModel{adaptive: true, thinksAlways: true, bindsPrefix: true}},
		{"claude-opus-5", anthropicModel{adaptive: true, disableNeedsLowEffort: true}},
		{"claude-opus-4-8", anthropicModel{adaptive: true}},
		{"claude-opus-4-7", anthropicModel{adaptive: true}},
		{"claude-opus-4-6", anthropicModel{adaptive: true}},
		{"claude-opus-4-5-20251101", anthropicModel{}},
		{"claude-opus-4-5@20251101", anthropicModel{}},
		{"claude-sonnet-5-5", anthropicModel{adaptive: true, disableRefused: true, bindsPrefix: true}},
		{"claude-sonnet-5", anthropicModel{adaptive: true}},
		{"claude-sonnet-4-6", anthropicModel{adaptive: true}},
		{"claude-sonnet-4-5", anthropicModel{}},
		{"claude-haiku-5-5", anthropicModel{adaptive: true, disableNeedsLowEffort: true}},
		{"claude-haiku-4-5", anthropicModel{}},
		{"claude-haiku-4-5-20251001", anthropicModel{}},
		{"claude-fable-5-1", anthropicModel{adaptive: true, thinksAlways: true, bindsPrefix: true}},
		{"claude-fable-5", anthropicModel{adaptive: true, thinksAlways: true}},
		{"claude-mythos-5-1", anthropicModel{adaptive: true, thinksAlways: true}},
		{"claude-mythos-5", anthropicModel{adaptive: true, thinksAlways: true}},
		{"claude-3-7-sonnet-20250219", anthropicModel{}},
		{"anthropic.claude-haiku-4-5-20251001-v1:0", anthropicModel{}},
		{"us.anthropic.claude-sonnet-4-6-v1:0", anthropicModel{adaptive: true}},
		{"my-gateway-alias", anthropicModel{}},
		{"", anthropicModel{}},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, anthropicTraits(tc.model), tc.model)
	}
}
