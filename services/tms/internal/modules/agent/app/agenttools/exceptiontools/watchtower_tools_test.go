package exceptiontools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWatchtower struct {
	guard     *agenttoolstest.WriteGuard
	item      *watchtower.Item
	dismissed bool
}

func (f *fakeWatchtower) PlanDismiss(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	*permissionports.RequestActor,
) (*kernelports.RecordChange[watchtower.Item], error) {
	after := *f.item
	now := int64(1_790_000_000)
	after.ResolvedAt = &now

	return &kernelports.RecordChange[watchtower.Item]{Before: f.item, After: &after}, nil
}

func (f *fakeWatchtower) Dismiss(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	*permissionports.RequestActor,
) (*watchtower.Item, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.dismissed = true

	return f.item, nil
}

func TestDismissWatchtowerItem_LeavesAgentOversightForAPerson(t *testing.T) {
	t.Parallel()

	items := &fakeWatchtower{guard: &agenttoolstest.WriteGuard{}, item: &watchtower.Item{
		ID:         pulid.MustNew("wti_"),
		SourceKind: watchtower.SourceWeatherAlert,
		Severity:   watchtower.Severity("Warning"),
		Title:      "Ice storm in Amarillo",
		Version:    1,
	}}
	tool := NewDismissWatchtowerItemTool(items)
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramWatchtowerItemID: items.item.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		items.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationArchive,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, items.dismissed)

	items.dismissed = false
	items.item.SourceKind = watchtower.SourceAgentRunFailed
	require.Error(t, tool.Execute(t.Context(), params))
	assert.False(t, items.dismissed)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}
