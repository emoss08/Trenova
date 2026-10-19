package deskbench

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.temporal.io/sdk/client"
)

type leftoverStarter struct {
	kernelports.WorkflowStarter

	cancelled []string
}

func (s *leftoverStarter) CancelWorkflow(_ context.Context, workflowID, _ string) error {
	s.cancelled = append(s.cancelled, workflowID)

	return nil
}

type leftoverRuns struct {
	agentports.AgentRunService

	watcher *TurnWatcher
	starter *leftoverStarter
	starts  int
}

func (r *leftoverRuns) StartForDefinition(
	_ context.Context,
	_ *agentports.StartAgentRunForDefinitionRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentRun, error) {
	r.starts++
	if len(r.starter.cancelled) == 0 {
		return nil, agentports.ErrAgentRunAlreadyOpen
	}
	run := &agent.AgentRun{ID: pulid.MustNew("ar_")}
	r.watcher.noteRun(leftoverWorkflowRun{}, []any{&agentcontract.AgentRunPayload{RunID: run.ID}})

	return run, nil
}

type leftoverWorkflowRun struct {
	client.WorkflowRun
}

func TestStartFreshRun_StopsTheRunAnEarlierBenchLeftAndFiresAgain(t *testing.T) {
	t.Parallel()

	sqlDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	sqlMock.MatchExpectationsInOrder(false)
	bunDB := bun.NewDB(sqlDB, pgdialect.New())
	t.Cleanup(func() { _ = bunDB.Close() })

	leftover := "agent-run-agdef_left-subject-inst_left"
	sqlMock.ExpectExec("SET LOCAL statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectQuery("SELECT workflow_id FROM agent_runs").
		WillReturnRows(sqlmock.NewRows([]string{"workflow_id"}).AddRow(leftover))

	starter := &leftoverStarter{}
	watcher := NewTurnWatcher(starter)
	runs := &leftoverRuns{watcher: watcher, starter: starter}
	bench := &Bench{
		deps:    deps{DB: postgres.NewTestConnection(bunDB), Runs: runs},
		watcher: watcher,
	}
	session := &Session{
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	definition := &agentdefinition.Definition{ID: pulid.MustNew("agdef_")}
	fire := &Fire{Event: agent.EventKind("insight.detected")}

	fired, err := bench.startFreshRun(
		t.Context(), session, definition, fire, pulid.MustNew("inst_"), "case")
	require.NoError(t, err)
	require.NotNil(t, fired)
	assert.Equal(t, []string{leftover}, starter.cancelled, "the leftover run is stopped once")
	assert.Equal(t, 2, runs.starts, "the fire is tried again after the leftover stops")
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}
