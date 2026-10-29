package assistanttest

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/stretchr/testify/require"
)

func FieldsByKey(t *testing.T, artifact *assistantartifact.Artifact) map[string]map[string]any {
	t.Helper()

	fields, ok := artifact.Payload["fields"].([]map[string]any)
	require.True(t, ok)
	out := make(map[string]map[string]any, len(fields))
	for _, field := range fields {
		out[field["key"].(string)] = field
	}

	return out
}
