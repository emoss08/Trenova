package watchtowerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type handOffFixture struct {
	repo  *stubRepo
	svc   *Service
	runs  *stubRuns
	actor *permissionports.RequestActor
	item  *watchtower.Item
}

func newHandOffFixture(t *testing.T) *handOffFixture {
	t.Helper()

	repo := newStubRepo()
	actor := testActor()
	svc := newService(repo, &agenttest.StubPermissions{})
	runs := &stubRuns{}
	svc.runs = runs

	in := input(
		actor.TenantInfo(),
		watchtower.SourceDetentionOccurrence,
		"dto_1",
		watchtower.SeverityWarning,
		100,
	)
	in.SubjectType = agent.SubjectDetentionOccurrence
	in.SubjectID = pulid.MustNew("dto_")
	in.EventKind = agenttrigger.EventDetentionOccurrenceOpened
	require.NoError(t, svc.projector.upsert(t.Context(), in))
	item := repo.items[repo.key(in.SourceKind, in.SourceID)]
	repo.events = nil

	return &handOffFixture{repo: repo, svc: svc, runs: runs, actor: actor, item: item}
}

func (f *handOffFixture) handOff(
	t *testing.T,
	definitionID pulid.ID,
) (*watchtowerports.HandOffWatchtowerItemResult, error) {
	t.Helper()

	return f.svc.HandOff(t.Context(), watchtowerports.HandOffWatchtowerItemRequest{
		TenantInfo:        f.actor.TenantInfo(),
		ItemID:            f.item.ID,
		AgentDefinitionID: definitionID,
	}, f.actor)
}

func TestHandOff_RecordsTheAgentOnTheItem(t *testing.T) {
	t.Parallel()

	f := newHandOffFixture(t)
	chosen := pulid.MustNew("adef_")

	result, err := f.handOff(t, chosen)
	require.NoError(t, err)
	require.NotNil(t, result.Run)
	require.NotNil(t, result.Item)

	assert.Equal(t, chosen, result.Item.AgentDefinitionID)
	assert.Equal(t, result.Run.ID, result.Item.AgentRunID)
	assert.Equal(t, f.actor.UserID, result.Item.AgentHandedByID)
	require.NotNil(t, result.Item.AgentHandedAt)
	assert.Equal(t, watchtower.StatusInProgress, result.Item.Status)
	assert.Equal(t, []watchtower.EventKind{watchtower.EventHandedOff}, eventKinds(f.repo.events))
	assert.Equal(t, chosen.String(), f.repo.events[0].ToValue)
}

func TestHandOff_RefusesWhileAnAgentIsStillWorkingIt(t *testing.T) {
	t.Parallel()

	f := newHandOffFixture(t)
	first, err := f.handOff(t, pulid.MustNew("adef_"))
	require.NoError(t, err)
	run := &agent.AgentRun{ID: first.Run.ID, Status: agent.RunStatusDiagnosing}
	f.svc.runRepo = &stubRunRepo{runs: map[pulid.ID]*agent.AgentRun{run.ID: run}}

	_, err = f.handOff(t, pulid.MustNew("adef_"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already working")

	run.Status = agent.RunStatusCompleted
	again, err := f.handOff(t, pulid.MustNew("adef_"))
	require.NoError(t, err, "an agent that finished leaves the item free to hand on")
	assert.NotEqual(t, first.Run.ID, again.Item.AgentRunID)
}

func TestHandOff_RefusesAResolvedItem(t *testing.T) {
	t.Parallel()

	f := newHandOffFixture(t)
	now := int64(100)
	f.item.ResolvedAt = &now
	f.item.Status = watchtower.StatusResolved

	_, err := f.handOff(t, pulid.MustNew("adef_"))
	require.Error(t, err)
	assert.Nil(t, f.runs.last, "no run is started for an item that is already closed")
}

func TestMarkItemSeen_MarksTheItemForThisReaderOnly(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	svc := newService(repo, &agenttest.StubPermissions{})
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceInsight, "ins_1", watchtower.SeverityInfo, 300),
	)
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceInsight, "ins_2", watchtower.SeverityInfo, 300),
	)
	opened := repo.items[repo.key(watchtower.SourceInsight, "ins_1")]

	item, err := svc.MarkItemSeen(t.Context(), watchtowerports.GetWatchtowerItemRequest{
		ID:         opened.ID,
		TenantInfo: tenant,
	}, actor)
	require.NoError(t, err)
	assert.True(t, item.Seen)

	seen := listSeen(t, svc, tenant, actor)
	assert.True(t, seen["ins_1"])
	assert.False(t, seen["ins_2"])

	other := testActor()
	other.OrganizationID = actor.OrganizationID
	other.BusinessUnitID = actor.BusinessUnitID
	assert.False(t, listSeen(t, svc, tenant, other)["ins_1"],
		"opening an item marks it for the person who opened it, not for everyone")
}

func listSeen(
	t *testing.T,
	svc *Service,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
) map[string]bool {
	t.Helper()

	page, err := svc.List(
		t.Context(),
		watchtowerports.ListWatchtowerItemsRequest{TenantInfo: tenant},
		actor,
	)
	require.NoError(t, err)
	seen := map[string]bool{}
	for _, item := range page.Items {
		seen[item.SourceID] = item.Seen
	}

	return seen
}

func TestMarkItemSeen_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	svc := newService(repo, &agenttest.StubPermissions{})
	actor := testActor()
	actor.PrincipalType = permissionports.PrincipalTypeAPIKey

	_, err := svc.MarkItemSeen(t.Context(), watchtowerports.GetWatchtowerItemRequest{
		ID:         pulid.MustNew("wt_"),
		TenantInfo: actor.TenantInfo(),
	}, actor)
	require.Error(t, err)
}

func TestCleared_RefusesAnInstantThatHasNotHappened(t *testing.T) {
	t.Parallel()

	svc := newService(newStubRepo(), &agenttest.StubPermissions{})
	actor := testActor()

	_, err := svc.Cleared(t.Context(), actor.TenantInfo(), actor, svc.now()+60)
	require.Error(t, err)
	_, err = svc.Cleared(t.Context(), actor.TenantInfo(), actor, 0)
	require.Error(t, err)
}

func TestUnsnoozeException_BringsItBackAndSaysWho(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})
	snoozed, err := f.exceptions.SnoozeException(
		t.Context(),
		&watchtowerports.SnoozeExceptionRequest{
			ExceptionTarget: f.target(),
			Until:           1_700_000_000 + 3600,
		},
		f.actor,
	)
	require.NoError(t, err)
	require.Equal(t, watchtower.StatusSnoozed, snoozed.Status)

	woken, err := f.exceptions.UnsnoozeException(
		t.Context(),
		&watchtowerports.UnsnoozeExceptionRequest{
			ExceptionTarget: f.target(),
		},
		f.actor,
	)
	require.NoError(t, err)
	assert.Equal(t, watchtower.StatusOpen, woken.Status)
	assert.Nil(t, woken.SnoozedUntil)
	last := f.repo.events[len(f.repo.events)-1]
	assert.Equal(t, watchtower.EventWoke, last.Kind)
	assert.Equal(t, f.actor.UserID, last.ActorID)

	_, err = f.exceptions.UnsnoozeException(t.Context(), &watchtowerports.UnsnoozeExceptionRequest{
		ExceptionTarget: f.target(),
	}, f.actor)
	require.Error(t, err, "an exception that is not snoozed cannot be woken")
}

func TestProjector_TakesTheSourcesDeadlineInsteadOfThePolicy(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	svc := newService(repo, &agenttest.StubPermissions{})
	in := input(
		actor.TenantInfo(),
		watchtower.SourceEtaAtRisk,
		"shp_1",
		watchtower.SeverityCritical,
		100,
	)
	in.Deadline = &watchtower.Deadline{At: 1_700_000_000 + 11_400, Label: "Minneapolis dock closes"}
	in.Facts = []watchtower.Fact{{Label: "Customer", Value: "Northline Foods"}}
	in.Impact = []string{"Late fee $250"}
	in.Trail = []watchtower.TrailEvent{{Text: "ETA moved past the appointment"}}

	require.NoError(t, svc.projector.upsert(t.Context(), in))
	item := repo.items[repo.key(in.SourceKind, in.SourceID)]

	require.NotNil(t, item.DueAt)
	assert.Equal(t, int64(1_700_000_000+11_400), *item.DueAt)
	assert.Equal(t, "Minneapolis dock closes", item.DueLabel)
	assert.Equal(t, watchtower.DueSourceSource, item.DueSource)
	assert.Equal(t, in.Facts, item.Facts)
	assert.Equal(t, in.Impact, item.Impact)
	assert.Equal(t, in.Trail, item.Trail)
	for _, event := range repo.events {
		assert.NotEqual(t, watchtower.EventDueChanged, event.Kind,
			"a deadline from the source is not routed to the severity policy")
	}
}

func TestProjector_RoutesAPolicyDueOnlyWhereNobodyChoseOne(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	svc := newService(repo, &agenttest.StubPermissions{})
	in := input(
		actor.TenantInfo(),
		watchtower.SourceInsight,
		"ins_1",
		watchtower.SeverityWarning,
		100,
	)
	require.NoError(t, svc.projector.upsert(t.Context(), in))
	item := repo.items[repo.key(in.SourceKind, in.SourceID)]

	require.NotNil(t, item.DueAt)
	assert.Equal(t, watchtower.DueSourcePolicy, item.DueSource)
	assert.Empty(t, item.DueLabel)

	cleared := *item
	cleared.DueAt = nil
	cleared.DueSource = watchtower.DueSourcePerson
	routed := svc.projector.route(t.Context(), &cleared, pulid.Nil, false)
	assert.Nil(t, routed.DueAt, "a due time a person cleared stays cleared")
}
