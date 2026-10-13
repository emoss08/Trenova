package watchtowerservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type userDeniedPermissions struct {
	*agenttest.StubPermissions

	denied pulid.ID
}

func (p *userDeniedPermissions) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	if req.UserID == p.denied {
		return &permissionports.PermissionCheckResult{Allowed: false, Reason: "denied"}, nil
	}

	return p.StubPermissions.Check(ctx, req)
}

type exceptionFixture struct {
	repo       *stubRepo
	exceptions *Exceptions
	actor      *permissionports.RequestActor
	item       *watchtower.Item
}

func newExceptionFixture(t *testing.T, perms permissionports.PermissionEngine) *exceptionFixture {
	t.Helper()

	repo := newStubRepo()
	projector := NewProjector(ProjectorParams{Logger: zap.NewNop(), Repo: repo})
	exceptions := NewExceptions(ExceptionsParams{
		Logger:      zap.NewNop(),
		Repo:        repo,
		Projector:   projector,
		Router:      NewRouter(RouterParams{Logger: zap.NewNop()}),
		Permissions: perms,
	})
	exceptions.now = func() int64 { return 1_700_000_000 }

	actor := testActor()
	in := input(
		actor.TenantInfo(),
		watchtower.SourceServiceFailure,
		"sf_1",
		watchtower.SeverityWarning,
		100,
	)
	require.NoError(t, projector.upsert(t.Context(), in))
	item := repo.items[repo.key(in.SourceKind, in.SourceID)]
	item.Status = watchtower.StatusOpen
	item.Version = 1
	repo.events = nil

	return &exceptionFixture{repo: repo, exceptions: exceptions, actor: actor, item: item}
}

func (f *exceptionFixture) target() watchtowerports.ExceptionTarget {
	return watchtowerports.ExceptionTarget{TenantInfo: f.actor.TenantInfo(), ItemID: f.item.ID}
}

func eventKinds(events []*watchtower.Event) []watchtower.EventKind {
	kinds := make([]watchtower.EventKind, 0, len(events))
	for _, event := range events {
		kinds = append(kinds, event.Kind)
	}

	return kinds
}

func TestAssignException_RecordsTheHandOverAndRefusesSomeoneWhoCannotSeeIt(t *testing.T) {
	t.Parallel()

	blocked := pulid.MustNew("usr_")
	f := newExceptionFixture(t, &userDeniedPermissions{
		StubPermissions: &agenttest.StubPermissions{},
		denied:          blocked,
	})
	assignee := pulid.MustNew("usr_")

	saved, err := f.exceptions.AssignException(t.Context(), &watchtowerports.AssignExceptionRequest{
		ExceptionTarget: f.target(),
		AssigneeID:      assignee,
		Note:            "Yours",
	}, f.actor)
	require.NoError(t, err)
	assert.Equal(t, assignee, saved.AssigneeID)
	assert.Equal(t, int64(2), saved.Version)
	require.Len(t, f.repo.events, 1)
	assert.Equal(t, watchtower.EventAssigned, f.repo.events[0].Kind)
	assert.Equal(t, assignee.String(), f.repo.events[0].ToValue)
	assert.Equal(t, "Yours", f.repo.events[0].Note)

	_, err = f.exceptions.AssignException(t.Context(), &watchtowerports.AssignExceptionRequest{
		ExceptionTarget: f.target(),
		AssigneeID:      blocked,
	}, f.actor)
	require.Error(t, err)
	assert.Equal(t, assignee, f.repo.items["ServiceFailure:sf_1"].AssigneeID)
}

func TestAcknowledgeException_ClaimsAnUnownedItemForTheReader(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})

	saved, err := f.exceptions.AcknowledgeException(t.Context(), new(f.target()), f.actor)
	require.NoError(t, err)
	assert.Equal(t, watchtower.StatusAcknowledged, saved.Status)
	assert.Equal(t, f.actor.UserID, saved.AssigneeID)
	require.NotNil(t, saved.AcknowledgedAt)
	assert.Equal(
		t,
		[]watchtower.EventKind{watchtower.EventAcknowledged, watchtower.EventAssigned},
		eventKinds(f.repo.events),
	)
}

func TestApplyException_RefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})
	target := f.target()
	target.ExpectedVersion = 7

	_, err := f.exceptions.StartException(t.Context(), &target, f.actor)
	require.Error(t, err)
	assert.Empty(t, f.repo.events)
	assert.Equal(t, watchtower.StatusOpen, f.repo.items["ServiceFailure:sf_1"].Status)
}

func TestApplyException_RefusesAReaderWhoCannotSeeTheKind(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{
		Denied: map[string]bool{"service_failure:read": true},
	})

	_, err := f.exceptions.ResolveException(t.Context(), &watchtowerports.ResolveExceptionRequest{
		ExceptionTarget: f.target(),
		Code:            watchtower.ResolutionFixed,
	}, f.actor)
	require.Error(t, err)
	var authErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authErr)
	assert.Empty(t, f.repo.events)
}

func TestResolveException_RequiresANoteForOtherAndRefusesTheSystemCode(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})

	_, err := f.exceptions.ResolveException(t.Context(), &watchtowerports.ResolveExceptionRequest{
		ExceptionTarget: f.target(),
		Code:            watchtower.ResolutionOther,
	}, f.actor)
	require.Error(t, err)

	_, err = f.exceptions.ResolveException(t.Context(), &watchtowerports.ResolveExceptionRequest{
		ExceptionTarget: f.target(),
		Code:            watchtower.ResolutionCleared,
	}, f.actor)
	require.Error(t, err)

	saved, err := f.exceptions.ResolveException(t.Context(), &watchtowerports.ResolveExceptionRequest{
		ExceptionTarget: f.target(),
		Code:            watchtower.ResolutionOther,
		Note:            "Customer called it in",
	}, f.actor)
	require.NoError(t, err)
	assert.Equal(t, watchtower.StatusResolved, saved.Status)
	assert.Equal(t, f.actor.UserID, saved.ResolvedByID)
	assert.Equal(t, "Customer called it in", saved.ResolutionNote)
	assert.Equal(t, []watchtower.EventKind{watchtower.EventResolved}, eventKinds(f.repo.events))
}

func TestReopenException_ClearsTheResolutionAndSetsAFreshDueTime(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})
	_, err := f.exceptions.ResolveException(t.Context(), &watchtowerports.ResolveExceptionRequest{
		ExceptionTarget: f.target(),
		Code:            watchtower.ResolutionFixed,
	}, f.actor)
	require.NoError(t, err)

	saved, err := f.exceptions.ReopenException(t.Context(), &watchtowerports.ReopenExceptionRequest{
		ExceptionTarget: f.target(),
		Note:            "Not fixed after all",
	}, f.actor)
	require.NoError(t, err)
	assert.Equal(t, watchtower.StatusOpen, saved.Status)
	assert.Nil(t, saved.ResolvedAt)
	assert.Empty(t, saved.ResolutionCode)
	require.NotNil(t, saved.DueAt)
	assert.Equal(t, int64(1_700_000_000+240*60), *saved.DueAt)
}

func TestSnoozeException_RefusesATimeInThePast(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})

	_, err := f.exceptions.SnoozeException(t.Context(), &watchtowerports.SnoozeExceptionRequest{
		ExceptionTarget: f.target(),
		Until:           1_600_000_000,
	}, f.actor)
	require.Error(t, err)

	saved, err := f.exceptions.SnoozeException(t.Context(), &watchtowerports.SnoozeExceptionRequest{
		ExceptionTarget: f.target(),
		Until:           1_700_003_600,
	}, f.actor)
	require.NoError(t, err)
	assert.Equal(t, watchtower.StatusSnoozed, saved.Status)
	require.NotNil(t, saved.SnoozedUntil)
	assert.Equal(t, int64(1_700_003_600), *saved.SnoozedUntil)
}

func TestCommentOnException_KeepsTheCommentInTheHistory(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})

	_, err := f.exceptions.CommentOnException(
		t.Context(),
		&watchtowerports.CommentOnExceptionRequest{ExceptionTarget: f.target(), Body: "   "},
		f.actor,
	)
	require.Error(t, err)

	event, err := f.exceptions.CommentOnException(
		t.Context(),
		&watchtowerports.CommentOnExceptionRequest{
			ExceptionTarget: f.target(),
			Body:            "  Carrier says 30 minutes out  ",
		},
		f.actor,
	)
	require.NoError(t, err)
	assert.Equal(t, "Carrier says 30 minutes out", event.Note)

	history, err := f.exceptions.ExceptionHistory(t.Context(), &watchtowerports.ExceptionHistoryRequest{
		TenantInfo: f.actor.TenantInfo(),
		ItemID:     f.item.ID,
	}, f.actor)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, watchtower.EventCommented, history[0].Kind)
	assert.Equal(t, f.actor.UserID, history[0].ActorID)
}
