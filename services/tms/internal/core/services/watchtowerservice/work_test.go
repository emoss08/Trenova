package watchtowerservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type workRepo struct {
	*stubRepo

	runs    []*agent.AgentRun
	snoozed []repositories.SnoozeWatchtowerItemRequest
}

func (r *workRepo) LatestRunsBySubject(
	_ context.Context,
	_ repositories.LatestRunsBySubjectRequest,
) ([]*agent.AgentRun, error) {
	return r.runs, nil
}

func (r *workRepo) Snooze(_ context.Context, req repositories.SnoozeWatchtowerItemRequest) error {
	r.snoozed = append(r.snoozed, req)

	return nil
}

// The tower says who is on each record: a run that began once the item was
// open, or one still going, and never an old run that finished before it.
func TestList_NamesTheRunWorkingOnEachRecordAndAnAgentThatCouldTakeIt(t *testing.T) {
	t.Parallel()

	stub := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	fresh, stale := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	repo := &workRepo{stubRepo: stub, runs: []*agent.AgentRun{
		{ID: pulid.MustNew("ar_"), SubjectID: fresh, Status: agent.RunStatusCompleted, CreatedAt: 150},
		{ID: pulid.MustNew("ar_"), SubjectID: stale, Status: agent.RunStatusCompleted, CreatedAt: 50},
	}}
	credentials := &agentdefinition.Definition{ID: pulid.MustNew("agd_"), Name: "Credential desk"}
	svc := newService(stub, &agentruntimetest.StubPermissions{})
	svc.repo = repo
	svc.definitions = &stubDefinitions{subscribers: []*agentdefinition.Definition{credentials}}

	for _, subject := range []pulid.ID{fresh, stale} {
		in := input(tenant, watchtower.SourceWorkerCredential, subject.String(), watchtower.SeverityWarning, 100)
		in.SubjectType = agent.SubjectWorker
		in.SubjectID = subject
		in.EventKind = agent.EventWorkerCredentialExpiring
		svc.projector.Upsert(t.Context(), in)
	}

	page, err := svc.List(
		t.Context(),
		services.ListWatchtowerItemsRequest{TenantInfo: tenant, UnresolvedOnly: true},
		actor,
	)
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	for _, item := range page.Items {
		assert.Equal(t, credentials, item.SuggestedAgent)
		if item.SubjectID == fresh {
			require.NotNil(t, item.ActiveRun)
		} else {
			assert.Nil(t, item.ActiveRun)
		}
	}
	require.NotNil(t, stub.lastList.Snoozed)
	assert.Equal(t, actor.UserID, stub.lastList.Snoozed.UserID)
}

func TestSnooze_PutsAnItemAsideForTheReaderWithinThirtyDays(t *testing.T) {
	t.Parallel()

	stub := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	repo := &workRepo{stubRepo: stub}
	svc := newService(stub, &agentruntimetest.StubPermissions{})
	svc.repo = repo
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityWarning, 100),
	)
	var id pulid.ID
	for _, item := range stub.items {
		id = item.ID
	}

	_, err := svc.Snooze(t.Context(), tenant, id, svc.now()-1, actor)
	require.Error(t, err)
	_, err = svc.Snooze(t.Context(), tenant, id, svc.now()+maxSnooze+1, actor)
	require.Error(t, err)
	assert.Empty(t, repo.snoozed)

	item, err := svc.Snooze(t.Context(), tenant, id, svc.now()+3600, actor)
	require.NoError(t, err)
	assert.Equal(t, id, item.ID)
	require.Len(t, repo.snoozed, 1)
	assert.Equal(t, actor.UserID, repo.snoozed[0].UserID)
	assert.EqualValues(t, svc.now()+3600, repo.snoozed[0].Until)
}
