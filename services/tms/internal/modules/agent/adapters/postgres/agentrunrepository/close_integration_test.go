//go:build integration

package agentrunrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type closeFixture struct {
	t      *testing.T
	ctx    context.Context
	db     *keel.DB
	repo   *repository
	tenant pagination.TenantInfo
	now    int64
}

func setupCloseFixture(t *testing.T) *closeFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)

	return &closeFixture{
		t:      t,
		ctx:    data.Ctx(ctx),
		db:     db,
		repo:   New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()}).(*repository),
		tenant: pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID},
		now:    timeutils.NowUnix(),
	}
}

func (f *closeFixture) run(
	trigger agent.RunTrigger,
	status agent.RunStatus,
	workflowID string,
	definitionID, subjectID pulid.ID,
) (*agent.AgentRun, error) {
	f.t.Helper()

	return f.repo.Create(f.ctx, &agent.AgentRun{
		ID:                pulid.MustNew("ar_"),
		OrganizationID:    f.tenant.OrgID,
		BusinessUnitID:    f.tenant.BuID,
		AgentType:         agent.TypeGeneral,
		AgentDefinitionID: definitionID,
		Trigger:           trigger,
		SubjectType:       agent.SubjectShipment,
		SubjectID:         subjectID,
		Status:            status,
		WorkflowID:        workflowID,
		PromptVersion:     "v1",
		InputContextHash:  "hash",
		StartedAt:         f.now,
	})
}

func (f *closeFixture) waiting(workflowID string) *agent.AgentRun {
	f.t.Helper()

	run, err := f.run(agent.RunTriggerEvent, agent.RunStatusAwaitingDecision, workflowID,
		pulid.MustNew("agd_"), pulid.MustNew("shp_"))
	require.NoError(f.t, err)

	return run
}

func (f *closeFixture) propose(runID pulid.ID, status agent.ProposalStatus) {
	f.t.Helper()

	require.NoError(f.t, agentdb.AgentProposals.Create(f.ctx, f.db, &agent.AgentProposal{
		ID:             pulid.MustNew("agp_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		RunID:          runID,
		ToolName:       "update_shipment",
		ToolParams:     map[string]any{},
		Rationale:      "a recorded proposal",
		Status:         status,
	}))
}

func (f *closeFixture) closeDecided(runID pulid.ID) *agent.AgentRun {
	f.t.Helper()

	run, err := f.repo.CloseDecided(f.ctx, &agentruntimeports.CloseDecidedRunRequest{
		TenantInfo: f.tenant, RunID: runID, Now: f.now,
	})
	require.NoError(f.t, err)

	return run
}

// A run stays waiting while any proposal is pending and closes once none is;
// a second close finds nothing to do, so the last decision and the expiry
// sweep racing for it close it once.
func TestCloseDecidedClosesOnlyWhenNothingIsPending(t *testing.T) {
	t.Parallel()

	f := setupCloseFixture(t)
	run := f.waiting("agent-run-" + pulid.MustNew("ar_").String())
	f.propose(run.ID, agent.ProposalStatusAccepted)
	f.propose(run.ID, agent.ProposalStatusPending)

	assert.Nil(t, f.closeDecided(run.ID), "a pending proposal keeps the run waiting")

	_, err := keel.Pool(f.db).Exec(f.ctx,
		`UPDATE agent_proposals SET status = 'Rejected' WHERE run_id = $1 AND status = 'Pending'`, run.ID)
	require.NoError(t, err)

	closed := f.closeDecided(run.ID)
	require.NotNil(t, closed)
	assert.Equal(t, agent.RunStatusCompleted, closed.Status)
	require.NotNil(t, closed.CompletedAt)
	assert.Nil(t, f.closeDecided(run.ID), "a closed run is closed once")
}

// A run started inline for a conversation keeps taking proposals and is closed
// by whoever started it, never by a decision; one an event started, a
// routine's, closes like a workflow's.
func TestCloseDecidedClosesOnlyRunsThatCloseWhenDecided(t *testing.T) {
	t.Parallel()

	f := setupCloseFixture(t)
	chat, err := f.run(agent.RunTriggerChat, agent.RunStatusAwaitingDecision, "",
		pulid.MustNew("agd_"), pulid.MustNew("shp_"))
	require.NoError(t, err)
	f.propose(chat.ID, agent.ProposalStatusAccepted)
	assert.Nil(t, f.closeDecided(chat.ID), "a conversation's inline run stays open")

	routine := f.waiting("")
	f.propose(routine.ID, agent.ProposalStatusAccepted)
	closed := f.closeDecided(routine.ID)
	require.NotNil(t, closed, "an event's inline run closes once decided")
	assert.Equal(t, agent.RunStatusCompleted, closed.Status)
}

// The sweep closes, across organizations, waiting runs whose proposals all
// expired, and leaves a run with anything pending.
func TestCloseSettledClosesRunsWhoseProposalsExpired(t *testing.T) {
	t.Parallel()

	f := setupCloseFixture(t)
	expired := f.waiting("agent-run-" + pulid.MustNew("ar_").String())
	f.propose(expired.ID, agent.ProposalStatusExpired)
	pending := f.waiting("agent-run-" + pulid.MustNew("ar_").String())
	f.propose(pending.ID, agent.ProposalStatusPending)

	closed, err := f.repo.CloseSettled(f.ctx, &agentruntimeports.CloseSettledRunsRequest{
		Now: f.now, Limit: 1000,
	})
	require.NoError(t, err)

	ids := storedRunIDs(closed)
	assert.Contains(t, ids, expired.ID)
	assert.NotContains(t, ids, pending.ID)
}

// A stopped run that waited on decisions fails with its reason; a run that
// was not waiting is left alone.
func TestFailWaitingFailsOnlyAWaitingRun(t *testing.T) {
	t.Parallel()

	f := setupCloseFixture(t)
	run := f.waiting("agent-run-" + pulid.MustNew("ar_").String())

	failed, err := f.repo.FailWaiting(f.ctx, &agentruntimeports.FailWaitingRunRequest{
		TenantInfo: f.tenant, RunID: run.ID, Reason: "stopped", Now: f.now,
	})
	require.NoError(t, err)
	require.NotNil(t, failed)
	assert.Equal(t, agent.RunStatusFailed, failed.Status)
	assert.Equal(t, "stopped", failed.ErrorMessage)

	again, err := f.repo.FailWaiting(f.ctx, &agentruntimeports.FailWaitingRunRequest{
		TenantInfo: f.tenant, RunID: run.ID, Reason: "stopped", Now: f.now,
	})
	require.NoError(t, err)
	assert.Nil(t, again)
}

// One open event or wait run per agent and subject, held by the index now
// that the run's workflow ends before its decisions: a second is refused as
// already open, and a run of another kind is not held to it.
func TestCreateRefusesASecondOpenRunForASubject(t *testing.T) {
	t.Parallel()

	f := setupCloseFixture(t)
	definitionID, subjectID := pulid.MustNew("agd_"), pulid.MustNew("shp_")

	_, err := f.run(agent.RunTriggerEvent, agent.RunStatusAwaitingDecision, "wf-1", definitionID, subjectID)
	require.NoError(t, err)

	_, err = f.run(agent.RunTriggerWait, agent.RunStatusPending, "wf-2", definitionID, subjectID)
	require.ErrorIs(t, err, agentruntimeports.ErrAgentRunAlreadyOpen)

	_, err = f.run(agent.RunTriggerManual, agent.RunStatusPending, "wf-3", definitionID, subjectID)
	require.NoError(t, err, "a manual run is not one of the subject's event runs")

	_, err = f.run(agent.RunTriggerEvent, agent.RunStatusAwaitingDecision, "", definitionID, subjectID)
	require.NoError(t, err, "an inline run carries no workflow and is not the subject's run")
}
