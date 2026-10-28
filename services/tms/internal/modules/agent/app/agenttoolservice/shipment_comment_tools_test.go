package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCommentModerator struct {
	guard    *agenttoolstest.WriteGuard
	current  *shipment.ShipmentComment
	refusal  error
	pinned   *bool
	resolved *bool
	updated  *shipmentcommentports.UpdateShipmentCommentRequest
	deleted  *shipmentcommentports.DeleteShipmentCommentRequest
	replies  int64
}

func (f *fakeCommentModerator) comment() *shipment.ShipmentComment {
	if f.current == nil {
		f.current = &shipment.ShipmentComment{
			ID:         pulid.MustNew("sc_"),
			ShipmentID: pulid.MustNew("shp_"),
			UserID:     pulid.MustNew("usr_"),
			Comment:    "Waiting on the lumper receipt",
			Type:       shipment.CommentTypeInternal,
			Visibility: shipment.CommentVisibilityInternal,
			Priority:   shipment.CommentPriorityNormal,
			Version:    2,
		}
	}

	return f.current
}

func (f *fakeCommentModerator) GetByID(
	context.Context,
	*shipmentcommentports.GetShipmentCommentByIDRequest,
) (*shipment.ShipmentComment, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.comment(), nil
}

func (f *fakeCommentModerator) toggle(pinned, resolved *bool) (*shipment.ShipmentComment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.pinned, f.resolved = pinned, resolved

	return f.comment(), nil
}

func (f *fakeCommentModerator) Pin(
	context.Context,
	*shipmentcommentports.ToggleShipmentCommentRequest,
	*permissionports.RequestActor,
) (*shipment.ShipmentComment, error) {
	return f.toggle(shipmenttools.BoolPointer(true), nil)
}

func (f *fakeCommentModerator) Unpin(
	context.Context,
	*shipmentcommentports.ToggleShipmentCommentRequest,
	*permissionports.RequestActor,
) (*shipment.ShipmentComment, error) {
	return f.toggle(shipmenttools.BoolPointer(false), nil)
}

func (f *fakeCommentModerator) Resolve(
	context.Context,
	*shipmentcommentports.ToggleShipmentCommentRequest,
	*permissionports.RequestActor,
) (*shipment.ShipmentComment, error) {
	return f.toggle(nil, shipmenttools.BoolPointer(true))
}

func (f *fakeCommentModerator) Unresolve(
	context.Context,
	*shipmentcommentports.ToggleShipmentCommentRequest,
	*permissionports.RequestActor,
) (*shipment.ShipmentComment, error) {
	return f.toggle(nil, shipmenttools.BoolPointer(false))
}

func (f *fakeCommentModerator) Update(
	_ context.Context,
	req *shipmentcommentports.UpdateShipmentCommentRequest,
	_ *permissionports.RequestActor,
) (*shipment.ShipmentComment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = req

	return req.Entity, nil
}

func (f *fakeCommentModerator) Delete(
	_ context.Context,
	req *shipmentcommentports.DeleteShipmentCommentRequest,
	_ *permissionports.RequestActor,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.deleted = req

	return nil
}

func (f *fakeCommentModerator) PreviewPin(
	_ context.Context,
	_ *shipmentcommentports.ToggleShipmentCommentRequest,
	actor *permissionports.RequestActor,
	pinned bool,
) (*shipmentcommentports.ShipmentCommentChange, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	before := f.comment()
	after := *before
	if pinned {
		now := int64(1_700_000_000)
		after.PinnedAt = &now
		after.PinnedByID = &actor.UserID
	} else {
		after.PinnedAt = nil
		after.PinnedByID = nil
	}

	return &shipmentcommentports.ShipmentCommentChange{Before: before, After: &after}, nil
}

func (f *fakeCommentModerator) PreviewResolve(
	_ context.Context,
	_ *shipmentcommentports.ToggleShipmentCommentRequest,
	actor *permissionports.RequestActor,
	resolved bool,
) (*shipmentcommentports.ShipmentCommentChange, error) {
	before := f.comment()
	after := *before
	if resolved {
		now := int64(1_700_000_000)
		after.ResolvedAt = &now
		after.ResolvedByID = &actor.UserID
	} else {
		after.ResolvedAt = nil
		after.ResolvedByID = nil
	}

	return &shipmentcommentports.ShipmentCommentChange{Before: before, After: &after}, nil
}

func (f *fakeCommentModerator) PreviewUpdate(
	_ context.Context,
	req *shipmentcommentports.UpdateShipmentCommentRequest,
	_ *permissionports.RequestActor,
) (*shipmentcommentports.ShipmentCommentChange, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return &shipmentcommentports.ShipmentCommentChange{Before: f.comment(), After: req.Entity}, nil
}

func (f *fakeCommentModerator) PreviewDelete(
	context.Context,
	*shipmentcommentports.DeleteShipmentCommentRequest,
	*permissionports.RequestActor,
) (*shipmentcommentports.ShipmentCommentDeletion, error) {
	return &shipmentcommentports.ShipmentCommentDeletion{
		Comment:    f.comment(),
		Tombstoned: f.replies > 0,
	}, nil
}

func commentParams(
	comments *fakeCommentModerator,
	extra map[string]any,
) agenttoolports.ToolExecuteParams {
	raw := map[string]any{
		toolparams.ParamShipmentID:   comments.comment().ShipmentID.String(),
		shipmenttools.ParamCommentID: comments.comment().ID.String(),
	}
	for key, value := range extra {
		raw[key] = value
	}

	return agenttoolstest.ExecuteParams(raw)
}

func TestPinShipmentComment_PreviewsThePinAndWrites(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewPinShipmentCommentTool(comments)
	params := commentParams(comments, nil)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		comments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationUpdate, change.Operation)
	assert.NotNil(t, agenttoolstest.FieldByPath(t, change, shipmenttools.FieldPinnedByID).After)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "pinned", result.Action)
	require.NotNil(t, comments.pinned)
	assert.True(t, *comments.pinned)

	policy := tool.Policy()
	assert.Equal(t, permission.OpPin, policy.Operation)
	assert.True(t, policy.Reversible)
}

func TestUnpinShipmentComment_ClearsThePin(t *testing.T) {
	t.Parallel()

	pinnedAt := int64(1_690_000_000)
	pinnedBy := pulid.MustNew("usr_")
	comments := &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}}
	comments.comment().PinnedAt = &pinnedAt
	comments.comment().PinnedByID = &pinnedBy
	tool := shipmenttools.NewUnpinShipmentCommentTool(comments)
	params := commentParams(comments, nil)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		comments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Nil(
		t,
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			shipmenttools.FieldPinnedByID,
		).After,
	)

	err := tool.Execute(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, comments.pinned)
	assert.False(t, *comments.pinned)
	assert.Equal(t, permission.OpUnpin, tool.Policy().Operation)
}

func TestResolveShipmentComment_ResolvesByDefaultAndReopensOnFalse(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewResolveShipmentCommentTool(comments)

	err := tool.Execute(t.Context(), commentParams(comments, nil))
	require.NoError(t, err)
	require.NotNil(t, comments.resolved)
	assert.True(t, *comments.resolved)

	reopen := commentParams(comments, map[string]any{shipmenttools.ParamResolved: false})
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		comments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), reopen)
		},
	)
	assert.Contains(t, preview.Summary, "resolve")
	err = tool.Execute(t.Context(), reopen)
	require.NoError(t, err)
	assert.False(t, *comments.resolved)
	assert.Equal(t, permission.OpResolve, tool.Policy().Operation)
}

func TestPinShipmentComment_AReplyIsAWouldFail(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{
		guard: &agenttoolstest.WriteGuard{},
		refusal: errortypes.NewValidationError(
			"commentId", errortypes.ErrInvalid, "Replies cannot be pinned"),
	}
	tool := shipmenttools.NewPinShipmentCommentTool(comments)

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(),
		commentParams(comments, nil))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)

	err = tool.Execute(
		t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{toolparams.ParamShipmentID: "x"}),
	)
	require.Error(t, err)
}

func TestEditShipmentComment_ChangesOnlyWhatIsSentAndKeepsMentions(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}}
	comments.comment().MentionedUsers = []*shipment.ShipmentCommentMention{
		{MentionedUserID: pulid.MustNew("usr_")},
	}
	permissions := &agenttoolstest.FakePermissionCheck{}
	tool := shipmenttools.NewEditShipmentCommentTool(comments, permissions)
	params := commentParams(comments, map[string]any{
		shipmenttools.FieldPriority: string(shipment.CommentPriorityHigh),
	})
	params.Actor.UserID = comments.comment().UserID

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		comments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(
		t,
		"High",
		agenttoolstest.FieldByPath(t, change, shipmenttools.FieldPriority).After,
	)
	assert.Len(t, change.Fields, 1)
	assert.Empty(t, permissions.Requests, "the author needs no moderation check")

	err := tool.Execute(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, comments.updated)
	assert.False(t, comments.updated.AsModerator)
	assert.Equal(t, "Waiting on the lumper receipt", comments.updated.Entity.Comment)
	assert.Len(t, comments.updated.Entity.MentionedUserIDs, 1)
	assert.Equal(t, int64(2), comments.updated.Entity.Version)
}

func TestEditShipmentComment_TakesEveryDomainPriorityAndRefusesTheRest(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewEditShipmentCommentTool(
		comments,
		&agenttoolstest.FakePermissionCheck{},
	)
	property, ok := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)[shipmenttools.FieldPriority].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, agenttoolschema.CommentPriorities.Name, property[toolschema.KeyEnumOf])
	assert.Contains(t, property[toolschema.KeyEnum], string(shipment.CommentPriorityUrgent))

	params := commentParams(comments, map[string]any{
		shipmenttools.FieldPriority: string(shipment.CommentPriorityUrgent),
	})
	params.Actor.UserID = comments.comment().UserID
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, comments.updated)
	assert.Equal(t, shipment.CommentPriorityUrgent, comments.updated.Entity.Priority)

	comments = &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}}
	tool = shipmenttools.NewEditShipmentCommentTool(comments, &agenttoolstest.FakePermissionCheck{})
	params = commentParams(comments, map[string]any{shipmenttools.FieldPriority: "Critical"})
	params.Actor.UserID = comments.comment().UserID
	err := tool.Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Critical")
	assert.Nil(t, comments.updated)
}

func TestEditShipmentComment_AnotherPersonsCommentNeedsTheManagePermission(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}}
	permissions := &agenttoolstest.FakePermissionCheck{Allowed: true}
	tool := shipmenttools.NewEditShipmentCommentTool(comments, permissions)
	params := commentParams(
		comments,
		map[string]any{shipmenttools.ParamCommentText: "Receipt received"},
	)

	err := tool.Execute(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, comments.updated)
	assert.True(t, comments.updated.AsModerator)
	require.Len(t, permissions.Requests, 1)
	assert.Equal(t, permission.OpManage, permissions.Requests[0].Operation)
	assert.Equal(t, permission.ResourceShipmentComment.String(), permissions.Requests[0].Resource)
	assert.Equal(t, "Receipt received", comments.updated.Entity.Comment)
}

func TestEditShipmentComment_VisibilityDecidesWhoReadsIt(t *testing.T) {
	t.Parallel()

	tool := shipmenttools.NewEditShipmentCommentTool(
		&fakeCommentModerator{},
		&agenttoolstest.FakePermissionCheck{},
	)
	policy := tool.Policy()
	assert.ElementsMatch(t, []agent.EgressClass{
		agent.EgressInternal, agent.EgressCustomerVisible, agent.EgressDriverVisible,
	}, policy.Egress)
	assert.Equal(
		t,
		agent.EgressCustomerVisible,
		policy.Classify(agenttoolstest.ExecuteParams(map[string]any{
			shipmenttools.FieldCommentVisib: string(shipment.CommentVisibilityCustomer),
		})).Egress,
	)
	assert.Equal(
		t,
		agent.EgressInternal,
		policy.Classify(agenttoolstest.ExecuteParams(map[string]any{})).Egress,
	)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamShipmentID:   pulid.MustNew("shp_").String(),
		shipmenttools.ParamCommentID: pulid.MustNew("sc_").String(),
	}))
	require.ErrorIs(t, err, toolparams.ErrNothingToChange)
}

func TestDeleteShipmentComment_SaysWhenAPlaceholderStays(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}, replies: 2}
	tool := shipmenttools.NewDeleteShipmentCommentTool(
		comments,
		&agenttoolstest.FakePermissionCheck{Allowed: true},
	)
	params := commentParams(comments, nil)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		comments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "placeholder")
	assert.Equal(
		t,
		agent.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "deleted", result.Action)
	require.NotNil(t, comments.deleted)
	assert.True(t, comments.deleted.AsModerator)

	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
}

func TestDeleteShipmentComment_WithoutTheManagePermissionRunsAsTheAuthor(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewDeleteShipmentCommentTool(
		comments,
		&agenttoolstest.FakePermissionCheck{Allowed: false},
	)

	err := tool.Execute(t.Context(), commentParams(comments, nil))
	require.NoError(t, err)
	require.NotNil(t, comments.deleted)
	assert.False(t, comments.deleted.AsModerator)
}
