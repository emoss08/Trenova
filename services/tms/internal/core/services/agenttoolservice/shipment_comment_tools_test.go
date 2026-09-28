package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCommentModerator struct {
	guard    *writeGuard
	current  *shipment.ShipmentComment
	refusal  error
	pinned   *bool
	resolved *bool
	updated  *serviceports.UpdateShipmentCommentRequest
	deleted  *serviceports.DeleteShipmentCommentRequest
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
	*repositories.GetShipmentCommentByIDRequest,
) (*shipment.ShipmentComment, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.comment(), nil
}

func (f *fakeCommentModerator) toggle(pinned, resolved *bool) (*shipment.ShipmentComment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.pinned, f.resolved = pinned, resolved

	return f.comment(), nil
}

func (f *fakeCommentModerator) Pin(
	context.Context,
	*serviceports.ToggleShipmentCommentRequest,
	*serviceports.RequestActor,
) (*shipment.ShipmentComment, error) {
	return f.toggle(boolPointer(true), nil)
}

func (f *fakeCommentModerator) Unpin(
	context.Context,
	*serviceports.ToggleShipmentCommentRequest,
	*serviceports.RequestActor,
) (*shipment.ShipmentComment, error) {
	return f.toggle(boolPointer(false), nil)
}

func (f *fakeCommentModerator) Resolve(
	context.Context,
	*serviceports.ToggleShipmentCommentRequest,
	*serviceports.RequestActor,
) (*shipment.ShipmentComment, error) {
	return f.toggle(nil, boolPointer(true))
}

func (f *fakeCommentModerator) Unresolve(
	context.Context,
	*serviceports.ToggleShipmentCommentRequest,
	*serviceports.RequestActor,
) (*shipment.ShipmentComment, error) {
	return f.toggle(nil, boolPointer(false))
}

func (f *fakeCommentModerator) Update(
	_ context.Context,
	req *serviceports.UpdateShipmentCommentRequest,
	_ *serviceports.RequestActor,
) (*shipment.ShipmentComment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = req

	return req.Entity, nil
}

func (f *fakeCommentModerator) Delete(
	_ context.Context,
	req *serviceports.DeleteShipmentCommentRequest,
	_ *serviceports.RequestActor,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = req

	return nil
}

func (f *fakeCommentModerator) PreviewPin(
	_ context.Context,
	_ *serviceports.ToggleShipmentCommentRequest,
	actor *serviceports.RequestActor,
	pinned bool,
) (*serviceports.ShipmentCommentChange, error) {
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

	return &serviceports.ShipmentCommentChange{Before: before, After: &after}, nil
}

func (f *fakeCommentModerator) PreviewResolve(
	_ context.Context,
	_ *serviceports.ToggleShipmentCommentRequest,
	actor *serviceports.RequestActor,
	resolved bool,
) (*serviceports.ShipmentCommentChange, error) {
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

	return &serviceports.ShipmentCommentChange{Before: before, After: &after}, nil
}

func (f *fakeCommentModerator) PreviewUpdate(
	_ context.Context,
	req *serviceports.UpdateShipmentCommentRequest,
	_ *serviceports.RequestActor,
) (*serviceports.ShipmentCommentChange, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return &serviceports.ShipmentCommentChange{Before: f.comment(), After: req.Entity}, nil
}

func (f *fakeCommentModerator) PreviewDelete(
	context.Context,
	*serviceports.DeleteShipmentCommentRequest,
	*serviceports.RequestActor,
) (*serviceports.ShipmentCommentDeletion, error) {
	return &serviceports.ShipmentCommentDeletion{
		Comment:    f.comment(),
		Tombstoned: f.replies > 0,
	}, nil
}

type fakePermissionCheck struct {
	allowed  bool
	err      error
	requests []*serviceports.PermissionCheckRequest
}

func (f *fakePermissionCheck) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}

	return &serviceports.PermissionCheckResult{Allowed: f.allowed}, nil
}

func commentParams(
	comments *fakeCommentModerator,
	extra map[string]any,
) serviceports.ToolExecuteParams {
	raw := map[string]any{
		paramShipmentID: comments.comment().ShipmentID.String(),
		paramCommentID:  comments.comment().ID.String(),
	}
	for key, value := range extra {
		raw[key] = value
	}

	return executeParams(raw)
}

func TestPinShipmentComment_PreviewsThePinAndWrites(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &writeGuard{}}
	tool := newPinShipmentCommentTool(comments)
	params := commentParams(comments, nil)

	preview := previewWithoutWrites(t, comments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationUpdate, change.Operation)
	assert.NotNil(t, fieldByPath(t, change, fieldPinnedByID).After)

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
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
	comments := &fakeCommentModerator{guard: &writeGuard{}}
	comments.comment().PinnedAt = &pinnedAt
	comments.comment().PinnedByID = &pinnedBy
	tool := newUnpinShipmentCommentTool(comments)
	params := commentParams(comments, nil)

	preview := previewWithoutWrites(t, comments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Nil(t, fieldByPath(t, previewChange(t, preview, 0), fieldPinnedByID).After)

	err := tool.Execute(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, comments.pinned)
	assert.False(t, *comments.pinned)
	assert.Equal(t, permission.OpUnpin, tool.Policy().Operation)
}

func TestResolveShipmentComment_ResolvesByDefaultAndReopensOnFalse(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &writeGuard{}}
	tool := newResolveShipmentCommentTool(comments)

	err := tool.Execute(t.Context(), commentParams(comments, nil))
	require.NoError(t, err)
	require.NotNil(t, comments.resolved)
	assert.True(t, *comments.resolved)

	reopen := commentParams(comments, map[string]any{paramResolved: false})
	preview := previewWithoutWrites(t, comments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), reopen)
	})
	assert.Contains(t, preview.Summary, "resolve")
	err = tool.Execute(t.Context(), reopen)
	require.NoError(t, err)
	assert.False(t, *comments.resolved)
	assert.Equal(t, permission.OpResolve, tool.Policy().Operation)
}

func TestPinShipmentComment_AReplyIsAWouldFail(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{
		guard: &writeGuard{},
		refusal: errortypes.NewValidationError(
			"commentId", errortypes.ErrInvalid, "Replies cannot be pinned"),
	}
	tool := newPinShipmentCommentTool(comments)

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(),
		commentParams(comments, nil))
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)

	err = tool.Execute(t.Context(), executeParams(map[string]any{paramShipmentID: "x"}))
	require.Error(t, err)
}

func TestEditShipmentComment_ChangesOnlyWhatIsSentAndKeepsMentions(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &writeGuard{}}
	comments.comment().MentionedUsers = []*shipment.ShipmentCommentMention{
		{MentionedUserID: pulid.MustNew("usr_")},
	}
	permissions := &fakePermissionCheck{}
	tool := newEditShipmentCommentTool(comments, permissions)
	params := commentParams(comments, map[string]any{
		fieldPriority: string(shipment.CommentPriorityHigh),
	})
	params.Actor.UserID = comments.comment().UserID

	preview := previewWithoutWrites(t, comments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "High", fieldByPath(t, change, fieldPriority).After)
	assert.Len(t, change.Fields, 1)
	assert.Empty(t, permissions.requests, "the author needs no moderation check")

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

	comments := &fakeCommentModerator{guard: &writeGuard{}}
	tool := newEditShipmentCommentTool(comments, &fakePermissionCheck{})
	property, ok := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)[fieldPriority].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, agenttoolschema.CommentPriorities.Name, property[toolschema.KeyEnumOf])
	assert.Contains(t, property[toolschema.KeyEnum], string(shipment.CommentPriorityUrgent))

	params := commentParams(comments, map[string]any{
		fieldPriority: string(shipment.CommentPriorityUrgent),
	})
	params.Actor.UserID = comments.comment().UserID
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, comments.updated)
	assert.Equal(t, shipment.CommentPriorityUrgent, comments.updated.Entity.Priority)

	comments = &fakeCommentModerator{guard: &writeGuard{}}
	tool = newEditShipmentCommentTool(comments, &fakePermissionCheck{})
	params = commentParams(comments, map[string]any{fieldPriority: "Critical"})
	params.Actor.UserID = comments.comment().UserID
	err := tool.Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Critical")
	assert.Nil(t, comments.updated)
}

func TestEditShipmentComment_AnotherPersonsCommentNeedsTheManagePermission(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &writeGuard{}}
	permissions := &fakePermissionCheck{allowed: true}
	tool := newEditShipmentCommentTool(comments, permissions)
	params := commentParams(comments, map[string]any{paramCommentText: "Receipt received"})

	err := tool.Execute(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, comments.updated)
	assert.True(t, comments.updated.AsModerator)
	require.Len(t, permissions.requests, 1)
	assert.Equal(t, permission.OpManage, permissions.requests[0].Operation)
	assert.Equal(t, permission.ResourceShipmentComment.String(), permissions.requests[0].Resource)
	assert.Equal(t, "Receipt received", comments.updated.Entity.Comment)
}

func TestEditShipmentComment_VisibilityDecidesWhoReadsIt(t *testing.T) {
	t.Parallel()

	tool := newEditShipmentCommentTool(&fakeCommentModerator{}, &fakePermissionCheck{})
	policy := tool.Policy()
	assert.ElementsMatch(t, []agent.EgressClass{
		agent.EgressInternal, agent.EgressCustomerVisible, agent.EgressDriverVisible,
	}, policy.Egress)
	assert.Equal(t, agent.EgressCustomerVisible, policy.Classify(executeParams(map[string]any{
		fieldCommentVisib: string(shipment.CommentVisibilityCustomer),
	})).Egress)
	assert.Equal(t, agent.EgressInternal, policy.Classify(executeParams(map[string]any{})).Egress)

	err := tool.Execute(t.Context(), executeParams(map[string]any{
		paramShipmentID: pulid.MustNew("shp_").String(),
		paramCommentID:  pulid.MustNew("sc_").String(),
	}))
	require.ErrorIs(t, err, errNothingToChange)
}

func TestDeleteShipmentComment_SaysWhenAPlaceholderStays(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentModerator{guard: &writeGuard{}, replies: 2}
	tool := newDeleteShipmentCommentTool(comments, &fakePermissionCheck{allowed: true})
	params := commentParams(comments, nil)

	preview := previewWithoutWrites(t, comments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "placeholder")
	assert.Equal(t, agent.PreviewOperationDelete, previewChange(t, preview, 0).Operation)

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
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

	comments := &fakeCommentModerator{guard: &writeGuard{}}
	tool := newDeleteShipmentCommentTool(comments, &fakePermissionCheck{allowed: false})

	err := tool.Execute(t.Context(), commentParams(comments, nil))
	require.NoError(t, err)
	require.NotNil(t, comments.deleted)
	assert.False(t, comments.deleted.AsModerator)
}
