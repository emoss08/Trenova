package airetrieval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelKeyDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  string
		want int
		ok   bool
	}{
		{key: "api.voyageai.com/voyage-3.5@1024", want: 1024, ok: true},
		{key: "localhost:11434/nomic-embed-text@768", want: 768, ok: true},
		{key: "api.openai.com/text-embedding-3-small@1536", want: 1536, ok: true},
		{key: "api.example.com/model@512"},
		{key: "api.example.com/model@"},
		{key: "@1024"},
		{key: "api.example.com/model"},
		{key: ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()

			got, ok := ModelKeyDimensions(tt.key)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModelKeyModel(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"localhost:11434/nomic-embed-text@768":         "nomic-embed-text",
		"localhost:11434/nomic-embed-text:latest@768":  "nomic-embed-text",
		"eval.local/nomic-embed-text@768":              "nomic-embed-text",
		"api.openai.com/text-embedding-3-small@1536":   "text-embedding-3-small",
		"nomic-embed-text":                             "nomic-embed-text",
		"api.together.xyz/BAAI/bge-large-en-v1.5@1024": "bge-large-en-v1.5",
		"": "",
	}
	for key, want := range tests {
		assert.Equal(t, want, ModelKeyModel(key), key)
	}
}

func TestSimilarityFloorFollowsTheEmbeddingModel(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.58, SimilarityFloor("localhost:11434/nomic-embed-text:latest@768"), 1e-9,
		"nomic scores unrelated text up to about 0.55, so its floor sits above that")
	assert.InDelta(t, DefaultSimilarityFloor,
		SimilarityFloor("api.openai.com/text-embedding-3-small@1536"), 1e-9)
	assert.InDelta(t, DefaultSimilarityFloor, SimilarityFloor(""), 1e-9)
}
