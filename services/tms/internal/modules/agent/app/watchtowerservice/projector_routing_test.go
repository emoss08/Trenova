package watchtowerservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeControlRepo struct {
	dispatchcontrolports.DispatchControlRepository

	control *dispatchcontrol.DispatchControl
	reads   int
}

func (f *fakeControlRepo) GetByOrgID(
	context.Context,
	dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	f.reads++

	return f.control, nil
}

func newRoutingProjector(
	repo *stubRepo,
	perms permissionports.PermissionEngine,
	control *dispatchcontrol.DispatchControl,
) (*Projector, *fakeControlRepo) {
	controls := &fakeControlRepo{control: control}
	projector := NewProjector(ProjectorParams{
		Logger:      zap.NewNop(),
		Repo:        repo,
		Router:      NewRouter(RouterParams{Logger: zap.NewNop(), Controls: controls}),
		Permissions: perms,
	})
	projector.now = func() int64 { return 1_700_000_000 }

	return projector, controls
}

func TestUpsert_RoutesANewItemToTheOwnerWithADueTimeBySeverity(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	fallback := pulid.MustNew("usr_")
	control := &dispatchcontrol.DispatchControl{
		ExceptionDueCriticalMinutes: 30,
		ExceptionDefaultAssigneeID:  fallback,
	}
	projector, controls := newRoutingProjector(repo, &agenttest.StubPermissions{}, control)
	tenant := testActor().TenantInfo()
	owner := pulid.MustNew("usr_")

	in := input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityCritical, 100)
	in.OwnerID = owner
	require.NoError(t, projector.upsert(t.Context(), in))

	item := repo.items["ServiceFailure:sf_1"]
	assert.Equal(t, owner, item.AssigneeID)
	require.NotNil(t, item.DueAt)
	assert.Equal(t, int64(1_700_000_000+30*60), *item.DueAt)

	other := input(tenant, watchtower.SourceServiceFailure, "sf_2", watchtower.SeverityInfo, 100)
	require.NoError(t, projector.upsert(t.Context(), other))
	second := repo.items["ServiceFailure:sf_2"]
	assert.Equal(t, fallback, second.AssigneeID)
	require.NotNil(t, second.DueAt)
	assert.Equal(
		t,
		int64(1_700_000_000+dispatchcontrol.DefaultExceptionDueInfoMinutes*60),
		*second.DueAt,
	)
	assert.Equal(t, 2, controls.reads, "each routing reads the cached dispatch control")
}

func TestUpsert_SkipsAnOwnerWhoCannotSeeTheItem(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	owner := pulid.MustNew("usr_")
	fallback := pulid.MustNew("usr_")
	projector, _ := newRoutingProjector(
		repo,
		&userDeniedPermissions{StubPermissions: &agenttest.StubPermissions{}, denied: owner},
		&dispatchcontrol.DispatchControl{ExceptionDefaultAssigneeID: fallback},
	)
	in := input(
		testActor().TenantInfo(),
		watchtower.SourceServiceFailure,
		"sf_1",
		watchtower.SeverityWarning,
		100,
	)
	in.OwnerID = owner
	require.NoError(t, projector.upsert(t.Context(), in))

	assert.Equal(t, fallback, repo.items["ServiceFailure:sf_1"].AssigneeID)
}

func TestUpsert_LeavesARoutedItemAloneOnTheNextReport(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	projector, _ := newRoutingProjector(repo, &agenttest.StubPermissions{}, nil)
	in := input(
		testActor().TenantInfo(),
		watchtower.SourceServiceFailure,
		"sf_1",
		watchtower.SeverityWarning,
		100,
	)
	require.NoError(t, projector.upsert(t.Context(), in))
	first := *repo.items["ServiceFailure:sf_1"].DueAt
	events := len(repo.events)

	projector.now = func() int64 { return 1_700_009_999 }
	require.NoError(t, projector.upsert(t.Context(), in))
	assert.Equal(t, first, *repo.items["ServiceFailure:sf_1"].DueAt)
	assert.Len(t, repo.events, events)
}

func TestReconcileSubject_RaisesWhatIsOpenAndResolvesTheRest(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	projector, _ := newRoutingProjector(repo, &agenttest.StubPermissions{}, nil)
	tenant := testActor().TenantInfo()
	shipmentID := pulid.MustNew("shp_")

	stale := input(tenant, watchtower.SourceMissedPickup, "stp_old", watchtower.SeverityWarning, 100)
	stale.SubjectType = agent.SubjectShipment
	stale.SubjectID = shipmentID
	require.NoError(t, projector.upsert(t.Context(), stale))

	other := input(tenant, watchtower.SourceMissedPickup, "stp_other", watchtower.SeverityWarning, 100)
	other.SubjectType = agent.SubjectShipment
	other.SubjectID = pulid.MustNew("shp_")
	require.NoError(t, projector.upsert(t.Context(), other))

	fresh := input(tenant, watchtower.SourceMissedPickup, "stp_new", watchtower.SeverityWarning, 200)
	fresh.SubjectType = agent.SubjectShipment
	fresh.SubjectID = shipmentID
	projector.ReconcileSubject(t.Context(), &watchtowerports.ReconcileSubjectRequest{
		TenantInfo: tenant,
		SourceKind: watchtower.SourceMissedPickup,
		SubjectID:  shipmentID,
		Open:       []watchtowerports.WatchtowerItemInput{fresh},
	})

	assert.NotNil(t, repo.items["MissedPickup:stp_old"].ResolvedAt)
	assert.Nil(t, repo.items["MissedPickup:stp_new"].ResolvedAt)
	assert.Nil(t, repo.items["MissedPickup:stp_other"].ResolvedAt)
}
