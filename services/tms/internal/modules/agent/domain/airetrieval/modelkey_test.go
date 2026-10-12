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
