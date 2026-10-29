package receivabletools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *fakeExceptions) UnsnoozeException(
	_ context.Context,
	req *watchtowerports.UnsnoozeExceptionRequest,
	_ *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return f.change(req.ExceptionTarget, func(item *watchtower.Item) error {
		return item.Unsnooze()
	})
}

func snoozedExceptions() *fakeExceptions {
	exceptions := newFakeExceptions(watchtower.SourceMissedPickup)
	until := int64(1_790_100_000)
	exceptions.item.Status = watchtower.StatusSnoozed
	exceptions.item.SnoozedUntil = &until

	return exceptions
}

func TestWakeException_PreviewsThenBringsItBackToTheQueue(t *testing.T) {
	t.Parallel()

	exceptions := snoozedExceptions()
	tool := NewWakeExceptionTool(exceptions)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramWatchtowerItemID: exceptions.item.ID.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, exceptions.guard, func() (
		*agentpreview.ToolPreview,
		error,
	) {
		return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Open", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.Contains(t, preview.Summary, "back to the open queue")
	assert.Nil(t, exceptions.saved)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, exceptions.saved)
	assert.Equal(t, watchtower.StatusOpen, exceptions.saved.Status)
	assert.Nil(t, exceptions.saved.SnoozedUntil)
}

func TestWakeException_RefusesAnExceptionThatIsNotSnoozed(t *testing.T) {
	t.Parallel()

	exceptions := newFakeExceptions(watchtower.SourceMissedPickup)
	tool := NewWakeExceptionTool(exceptions)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramWatchtowerItemID: exceptions.item.ID.String(),
	})

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, exceptions.saved)
}

func TestWakeException_IsTheInternalCounterpartOfSnooze(t *testing.T) {
	t.Parallel()

	policy := NewWakeExceptionTool(nil).Policy()
	snooze := NewSnoozeExceptionTool(nil).Policy()

	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	assert.Equal(t, snooze.Resource, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	assert.True(t, policy.Reversible)
}
