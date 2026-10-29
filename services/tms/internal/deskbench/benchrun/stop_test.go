package benchrun

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchcheck"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

type stoppedWorkflowRun struct {
	client.WorkflowRun
}

func (stoppedWorkflowRun) Get(context.Context, any) error {
	return context.Canceled
}

func TestStopWaitingRun_StopsTheRunAndGradesHowItClosed(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("arun_")
	records := &runRecordsFake{run: &agent.AgentRun{
		ID:           runID,
		Status:       agent.RunStatusFailed,
		ErrorMessage: "the run was stopped while its proposals waited for a decision",
	}}
	starter := &leftoverStarter{}
	runner := &Runner{
		runRecords: records,
		facts:      factRows(t, "SELECT id FROM agent_proposals", nil),
		watcher:    benchwatch.New(starter),
	}
	session := &deskbench.Session{
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	workflowID := "agent-run-agdef_stop-subject-bqi_stop"
	fired := &firedRun{
		run:      &agent.AgentRun{ID: runID, WorkflowID: workflowID},
		workflow: stoppedWorkflowRun{},
	}
	record := &benchrecord.TurnRecord{
		Proposals: []*benchrecord.ProposalRecord{{Status: agent.ProposalStatusPending}},
	}
	outcome := &benchcheck.RunOutcome{ID: runID, Status: agent.RunStatusAwaitingDecision}

	runner.stopWaitingRun(t.Context(), session, fired, record, outcome, "case")

	assert.Equal(t, []string{workflowID}, starter.cancelled)
	assert.True(t, outcome.Stopped)
	assert.Equal(t, agent.RunStatusFailed, outcome.Status)
	assert.Equal(t, records.run.ErrorMessage, record.Failure)
	assert.Empty(t, record.Proposals, "the proposals are read again, so none is left to reject")
	assert.Empty(t, record.CaptureError)

	checks := benchcheck.RunChecks(&benchscenario.Expect{Run: "Failed"}, outcome)
	require.Len(t, checks, 2)
	for _, check := range checks {
		assert.True(t, check.Passed, check.Name)
	}
}

func TestStopWaitingRun_FailsTheCaseWhenTheRunWasNotWaiting(t *testing.T) {
	t.Parallel()

	starter := &leftoverStarter{}
	runner := &Runner{watcher: benchwatch.New(starter)}
	outcome := &benchcheck.RunOutcome{Status: agent.RunStatusCompleted}

	runner.stopWaitingRun(t.Context(), &deskbench.Session{}, &firedRun{run: &agent.AgentRun{}},
		&benchrecord.TurnRecord{}, outcome, "case")

	assert.Empty(t, starter.cancelled, "a run that already finished is left alone")
	checks := benchcheck.RunChecks(&benchscenario.Expect{}, outcome)
	require.Len(t, checks, 1)
	assert.False(t, checks[0].Passed)
	assert.Equal(t, "it was Completed, not waiting for a decision", checks[0].Detail)
}
