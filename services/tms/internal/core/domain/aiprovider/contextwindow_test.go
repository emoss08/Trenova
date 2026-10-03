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
