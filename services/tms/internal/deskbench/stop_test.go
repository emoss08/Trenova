package deskbench

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
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
	bench := &Bench{
		deps:    deps{RunRecords: records},
		facts:   factRows(t, "SELECT id FROM agent_proposals", nil),
		watcher: NewTurnWatcher(starter),
	}
	session := &Session{
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	workflowID := "agent-run-agdef_stop-subject-bqi_stop"
	fired := &firedRun{
		run:      &agent.AgentRun{ID: runID, WorkflowID: workflowID},
		workflow: stoppedWorkflowRun{},
	}
	record := &TurnRecord{Proposals: []*ProposalRecord{{Status: agent.ProposalStatusPending}}}
	outcome := &runOutcome{ID: runID, Status: agent.RunStatusAwaitingDecision}

	bench.stopWaitingRun(t.Context(), session, fired, record, outcome, "case")

	assert.Equal(t, []string{workflowID}, starter.cancelled)
	assert.True(t, outcome.Stopped)
	assert.Equal(t, agent.RunStatusFailed, outcome.Status)
	assert.Equal(t, records.run.ErrorMessage, record.Failure)
	assert.Empty(t, record.Proposals, "the proposals are read again, so none is left to reject")
	assert.Empty(t, record.CaptureError)

	checks := runChecks(&Expect{Run: "Failed"}, outcome)
	require.Len(t, checks, 2)
	for _, check := range checks {
		assert.True(t, check.Passed, check.Name)
	}
}

func TestStopWaitingRun_FailsTheCaseWhenTheRunWasNotWaiting(t *testing.T) {
	t.Parallel()

	starter := &leftoverStarter{}
	bench := &Bench{watcher: NewTurnWatcher(starter)}
	outcome := &runOutcome{Status: agent.RunStatusCompleted}

	bench.stopWaitingRun(t.Context(), &Session{}, &firedRun{run: &agent.AgentRun{}},
		&TurnRecord{}, outcome, "case")

	assert.Empty(t, starter.cancelled, "a run that already finished is left alone")
	checks := runChecks(&Expect{}, outcome)
	require.Len(t, checks, 1)
	assert.False(t, checks[0].Passed)
	assert.Equal(t, "it was Completed, not waiting for a decision", checks[0].Detail)
}

func TestTurnsCompleted_TakesTheStopTheBenchMadeAsTheRunsOutcome(t *testing.T) {
	t.Parallel()

	turn := &TurnRecord{
		Status:  conversation.AssistantTurnStatusFailed,
		Failure: "the run was stopped while its proposals waited for a decision",
	}

	stopped := turnsCompleted(&stepEvidence{
		turns: []*TurnRecord{turn},
		run:   &runOutcome{Stopping: true, Stopped: true},
	})
	assert.True(t, stopped.Passed, stopped.Detail)

	failed := turnsCompleted(&stepEvidence{
		turns: []*TurnRecord{turn},
		run:   &runOutcome{Stopping: true},
	})
	assert.False(t, failed.Passed, "a run the bench did not stop failed on its own")

	captured := turnsCompleted(&stepEvidence{
		turns: []*TurnRecord{{
			Status:       conversation.AssistantTurnStatusFailed,
			CaptureError: "the stopped run did not close within 1m",
		}},
		run: &runOutcome{Stopping: true, Stopped: true},
	})
	assert.False(t, captured.Passed)
}
