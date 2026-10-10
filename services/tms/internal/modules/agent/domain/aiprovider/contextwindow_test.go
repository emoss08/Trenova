package aiprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextWindowFor_ReadsTheFamilyWhateverFormTheIDTakes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		model string
		want  int
	}{
		{"claude-sonnet-4-5", 200_000},
		{"us.anthropic.claude-haiku-4-5-20251001-v1:0", 200_000},
		{"anthropic/claude-opus-4-1", 200_000},
		{"gpt-4.1-mini", 1_047_576},
		{"gpt-5-mini", 272_000},
		{"gpt-4o-2024-08-06", 128_000},
		{"o3-mini", 200_000},
		{"openai/o4-mini", 200_000},
		{"gemini-2.5-pro", 1_048_576},
		{"llama3.3:70b", 128_000},
		{"qwen2.5-coder:32b", 32_768},
		{"house-model-v2", DefaultContextWindow},
		{"", DefaultContextWindow},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ContextWindowFor(tc.model), tc.model)
	}
}

/*
An operator serving a Qwen model with its context stretched to 128k, or a
Llama with it cut to 16k to fit a GPU, knows the window better than the id
does, and a window read off the id would compact the first conversation far
too late or far too soon.
*/
func TestContextWindow_TheConfiguredWindowWinsOverTheModelID(t *testing.T) {
	t.Parallel()

	stretched := 131_072
	provider := &Provider{Model: "qwen2.5-coder:32b", ContextWindowTokens: &stretched}
	assert.Equal(t, 131_072, provider.ContextWindow())
	assert.Equal(t, 131_072, provider.ConfiguredContextWindow())

	provider.ContextWindowTokens = nil
	assert.Equal(t, 32_768, provider.ContextWindow(), "left unset, the id is read")
	assert.Zero(t, provider.ConfiguredContextWindow())

	assert.Equal(t, 16_384, WindowOr(16_384, "llama3.3:70b"))
	assert.Equal(t, 128_000, WindowOr(0, "llama3.3:70b"))
}
