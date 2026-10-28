package receivabletools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExceptions struct {
	watchtowerports.ExceptionService

	guard    *agenttoolstest.WriteGuard
	item     *watchtower.Item
	saved    *watchtower.Item
	comments []string
}

func (f *fakeExceptions) GetException(
	context.Context,
	watchtowerports.GetWatchtowerItemRequest,
	*permissionports.RequestActor,
) (*watchtower.Item, error) {
	copied := *f.item
	return &copied, nil
}

func (f *fakeExceptions) change(
	target watchtowerports.ExceptionTarget,
	mutate func(item *watchtower.Item) error,
) (*watchtower.Item, error) {
	working := *f.item
	if err := mutate(&working); err != nil {
		return nil, err
	}
	if target.DryRun {
		return &working, nil
	}
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.saved = &working
	return &working, nil
}

func (f *fakeExceptions) AssignException(
	_ context.Context,
	req *watchtowerports.AssignExceptionRequest,
	_ *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return f.change(req.ExceptionTarget, func(item *watchtower.Item) error {
		return item.Assign(req.AssigneeID)
	})
}

func (f *fakeExceptions) ResolveException(
	_ context.Context,
	req *watchtowerports.ResolveExceptionRequest,
	_ *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return f.change(req.ExceptionTarget, func(item *watchtower.Item) error {
		return item.Resolve(req.Code, req.Note, pulid.MustNew("usr_"), 1_790_000_000)
	})
}

func (f *fakeExceptions) CommentOnException(
	_ context.Context,
	req *watchtowerports.CommentOnExceptionRequest,
	_ *permissionports.RequestActor,
) (*watchtower.Event, error) {
	body, err := watchtower.ValidateComment(req.Body)
	if err != nil {
		return nil, err
	}
	event := watchtower.NewEvent(f.item, watchtower.EventCommented, pulid.Nil).WithNote(body)
	if req.DryRun {
		return event, nil
	}
	if err = f.guard.Write(); err != nil {
		return nil, err
	}
	f.comments = append(f.comments, body)
	return event, nil
}

func newFakeExceptions(kind watchtower.SourceKind) *fakeExceptions {
	return &fakeExceptions{guard: &agenttoolstest.WriteGuard{}, item: &watchtower.Item{
		ID:         pulid.MustNew("wt_"),
		SourceKind: kind,
		Severity:   watchtower.SeverityWarning,
		Status:     watchtower.StatusOpen,
		Title:      "Missed pickup on PRO-7",
		Version:    3,
	}}
}

func TestResolveException_PreviewsThenResolves(t *testing.T) {
	t.Parallel()

	exceptions := newFakeExceptions(watchtower.SourceMissedPickup)
	tool := NewResolveExceptionTool(exceptions)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramWatchtowerItemID: exceptions.item.ID.String(),
		paramResolutionCode:   "CarrierDelay",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		exceptions.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Resolved", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.Nil(t, exceptions.saved)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, exceptions.saved)
	assert.Equal(t, watchtower.ResolutionCarrierDelay, exceptions.saved.ResolutionCode)
}

func TestResolveException_RefusesTheSystemCodeAndAnAgentsOwnWork(t *testing.T) {
	t.Parallel()

	exceptions := newFakeExceptions(watchtower.SourceMissedPickup)
	tool := NewResolveExceptionTool(exceptions)
	cleared := agenttoolstest.ExecuteParams(map[string]any{
		paramWatchtowerItemID: exceptions.item.ID.String(),
		paramResolutionCode:   "Cleared",
	})
	require.Error(t, tool.Execute(t.Context(), cleared))

	oversight := newFakeExceptions(watchtower.SourceAgentRunFailed)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramWatchtowerItemID: oversight.item.ID.String(),
		paramResolutionCode:   "Fixed",
	})
	require.Error(t, NewResolveExceptionTool(oversight).Execute(t.Context(), params))
	assert.Nil(t, oversight.saved)
}

func TestAssignException_GivesTheItemToTheNamedPerson(t *testing.T) {
	t.Parallel()

	exceptions := newFakeExceptions(watchtower.SourcePodMissing)
	assignee := pulid.MustNew("usr_")
	tool := NewAssignExceptionTool(exceptions)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramWatchtowerItemID:  exceptions.item.ID.String(),
		paramExceptionAssignee: assignee.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		exceptions.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agent.PreviewOperationUpdate,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	assert.Nil(t, exceptions.saved)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, assignee, exceptions.saved.AssigneeID)
}

func TestCommentOnException_PreviewsWithoutWriting(t *testing.T) {
	t.Parallel()

	exceptions := newFakeExceptions(watchtower.SourceTenderExhausted)
	tool := NewCommentOnExceptionTool(exceptions)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramWatchtowerItemID: exceptions.item.ID.String(),
		paramCommentBody:      "Spot carrier quoting now",
	})

	agenttoolstest.PreviewWithoutWrites(t, exceptions.guard, func() (*agent.ToolPreview, error) {
		return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Empty(t, exceptions.comments)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []string{"Spot carrier quoting now"}, exceptions.comments)
}
