package agenttoolstest

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/stretchr/testify/require"
)

func FindChange(
	t *testing.T,
	preview *agentpreview.ToolPreview,
	operation agentpreview.PreviewOperation,
) *agentpreview.RecordChange {
	t.Helper()

	for i := range preview.Changes {
		if preview.Changes[i].Operation == operation {
			return &preview.Changes[i]
		}
	}
	require.Failf(t, "no change", "the preview has no %s change", operation)

	return nil
}

func FindField(
	t *testing.T,
	change *agentpreview.RecordChange,
	path string,
) agentpreview.PreviewFieldChange {
	t.Helper()

	for _, field := range change.Fields {
		if field.Path == path {
			return field
		}
	}
	require.Failf(t, "no field", "the change has no %q field", path)

	return agentpreview.PreviewFieldChange{}
}
