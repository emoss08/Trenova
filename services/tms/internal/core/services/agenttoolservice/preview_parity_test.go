package agenttoolservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/stretchr/testify/require"
)

var errWriteDuringPreview = errors.New("a preview attempted a write")

type writeGuard struct {
	locked   bool
	attempts int
}

func (g *writeGuard) write() error {
	if g == nil || !g.locked {
		return nil
	}
	g.attempts++

	return errWriteDuringPreview
}

func previewWithoutWrites(
	t *testing.T,
	guard *writeGuard,
	run func() (*agent.ToolPreview, error),
) *agent.ToolPreview {
	t.Helper()

	guard.locked = true
	preview, err := run()
	guard.locked = false

	require.NoError(t, err)
	require.NotNil(t, preview)
	require.Zero(t, guard.attempts, "a preview must not write")

	return preview
}

type stableField struct {
	Path   string
	Before any
	After  any
}

func stableFields(fields []agent.PreviewFieldChange) []stableField {
	out := make([]stableField, 0, len(fields))
	for i := range fields {
		if fields[i].Volatile {
			continue
		}
		out = append(out, stableField{
			Path:   fields[i].Path,
			Before: fields[i].Before,
			After:  fields[i].After,
		})
	}

	return out
}

func requireUpdateParity[T any](
	t *testing.T,
	change *agent.RecordChange,
	before, saved *T,
	opts ...toolpreview.Option,
) {
	t.Helper()

	require.NotNil(t, change)
	want, err := toolpreview.Changed(toolpreview.Record{
		Resource: change.Resource,
		ID:       change.EntityID,
	}, before, saved, opts...)
	require.NoError(t, err)
	require.NotEmpty(t, stableFields(want.Fields), "the write changed nothing the preview shows")
	require.Equal(t, stableFields(want.Fields), stableFields(change.Fields))
}

func requireCreateParity[T any](
	t *testing.T,
	change *agent.RecordChange,
	saved *T,
	opts ...toolpreview.Option,
) {
	t.Helper()

	require.NotNil(t, change)
	require.Equal(t, agent.PreviewOperationCreate, change.Operation)
	want, err := toolpreview.Create(toolpreview.Record{Resource: change.Resource}, saved, opts...)
	require.NoError(t, err)
	require.NotEmpty(t, stableFields(want.Fields))
	require.Equal(t, stableFields(want.Fields), stableFields(change.Fields))
}

func previewChange(t *testing.T, preview *agent.ToolPreview, index int) *agent.RecordChange {
	t.Helper()

	require.Greater(t, len(preview.Changes), index)

	return &preview.Changes[index]
}

func fieldByPath(
	t *testing.T,
	change *agent.RecordChange,
	path string,
) *agent.PreviewFieldChange {
	t.Helper()

	for i := range change.Fields {
		if change.Fields[i].Path == path {
			return &change.Fields[i]
		}
	}
	require.Failf(t, "field missing", "the preview has no %q field", path)

	return nil
}

func requireWarning(t *testing.T, preview *agent.ToolPreview, code agent.PreviewWarningCode) {
	t.Helper()

	for _, warning := range preview.Warnings {
		if warning.Code == code {
			return
		}
	}
	require.Failf(t, "warning missing", "the preview has no %q warning", code)
}
