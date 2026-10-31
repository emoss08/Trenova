package modeladapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/require"
)

func TestWithSchemaInstruction(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
	}

	t.Run("omits the schema when the endpoint enforces it", func(t *testing.T) {
		t.Parallel()
		// Repeating the schema in the prompt would waste context on a provider
		// that already constrains decoding against it.
		got := WithSchemaInstruction("base", schema, aiprovider.StructuredOutputJSONSchema)
		require.Equal(t, "base", got)
	})

	t.Run("carries the schema when only JSON validity is guaranteed", func(t *testing.T) {
		t.Parallel()
		got := WithSchemaInstruction("base", schema, aiprovider.StructuredOutputJSONMode)
		require.Contains(t, got, "Required Output Format")
		require.Contains(t, got, `"properties"`)
	})

	t.Run("carries the schema when nothing is guaranteed", func(t *testing.T) {
		t.Parallel()
		got := WithSchemaInstruction("base", schema, aiprovider.StructuredOutputPrompted)
		require.Contains(t, got, "Required Output Format")
		require.Contains(t, got, "single JSON object")
	})

	t.Run("is a no-op without a schema", func(t *testing.T) {
		t.Parallel()
		got := WithSchemaInstruction("base", nil, aiprovider.StructuredOutputPrompted)
		require.Equal(t, "base", got)
	})
}
