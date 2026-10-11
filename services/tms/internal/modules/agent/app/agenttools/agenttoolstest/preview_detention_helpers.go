package agenttoolstest

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/require"
)

func FindChange(
	t *testing.T,
	preview *agent.ToolPreview,
	operation agent.PreviewOperation,
) *agent.RecordChange {
	t.Helper()

	for i := range preview.Changes {
		if preview.Changes[i].Operation == operation {
			return &preview.Changes[i]
		}
	}
	require.Failf(t, "no change", "the preview has no %s change", operation)

	return nil
}

func FindField(t *testing.T, change *agent.RecordChange, path string) agent.PreviewFieldChange {
	t.Helper()

	for _, field := range change.Fields {
		if field.Path == path {
			return field
		}
	}
	require.Failf(t, "no field", "the change has no %q field", path)

	return agent.PreviewFieldChange{}
}
