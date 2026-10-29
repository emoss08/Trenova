package completionembed

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
)

func TestBatchEmbeddingInputs(t *testing.T) {
	t.Parallel()

	limits := embeddingBatchLimits{inputs: 3, tokens: 10}

	tests := []struct {
		name   string
		inputs []string
		want   []int
	}{
		{name: "one input", inputs: []string{"abcd"}, want: []int{1}},
		{
			name:   "splits on the input count",
			inputs: []string{"a", "b", "c", "d", "e", "f", "g"},
			want:   []int{3, 3, 1},
		},
		{
			name:   "splits on the token budget",
			inputs: []string{strings.Repeat("x", 24), strings.Repeat("x", 24), "y"},
			want:   []int{1, 2},
		},
		{
			name:   "an input over the budget travels alone",
			inputs: []string{"a", strings.Repeat("x", 400), "b"},
			want:   []int{1, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			batches := batchEmbeddingInputs(tt.inputs, limits)
			sizes := make([]int, 0, len(batches))
			total := 0
			for _, batch := range batches {
				sizes = append(sizes, len(batch.inputs))
				total += len(batch.inputs)
			}
			assert.Equal(t, tt.want, sizes)
			assert.Equal(t, len(tt.inputs), total, "every input is sent exactly once")
		})
	}
}

func TestEmbeddingLimitsFor(t *testing.T) {
	t.Parallel()

	openAI := &aiprovider.Provider{Kind: aiprovider.KindOpenAIResponses}
	gemini := &aiprovider.Provider{
		Kind:    aiprovider.KindOpenAIChat,
		BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
	}

	assert.Equal(t, EmbeddingBatchMaxInputs, embeddingLimitsFor(openAI).inputs)
	assert.Equal(t, EmbeddingBatchMaxTokens, embeddingLimitsFor(openAI).tokens)
	assert.Equal(t, GeminiEmbeddingBatchMaxInputs, embeddingLimitsFor(gemini).inputs)
}
