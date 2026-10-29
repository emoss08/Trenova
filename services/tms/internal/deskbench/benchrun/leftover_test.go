package benchrun

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchsql"
	"github.com/emoss08/trenova/internal/deskbench/benchwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

type leftoverStarter struct {
	kernelports.WorkflowStarter

	cancelled []string
}

func (s *leftoverStarter) StartWorkflow(
	context.Context,
	client.StartWorkflowOptions,
	any,
	...any,
) (client.WorkflowRun, error) {
	return leftoverWorkflowRun{}, nil
}

func (s *leftoverStarter) CancelWorkflow(_ context.Context, workflowID, _ string) error {
	s.cancelled = append(s.cancelled, workflowID)

	return nil
}

type leftoverRuns struct {
	agentports.AgentRunService

	watcher *benchwatch.TurnWatcher
	starter *leftoverStarter
	starts  int
}

func (r *leftoverRuns) StartForDefinition(
	ctx context.Context,
	_ *agentports.StartAgentRunForDefinitionRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentRun, error) {
	r.starts++
	if len(r.starter.cancelled) == 0 {
		return nil, agentports.ErrAgentRunAlreadyOpen
	}
	run := &agent.AgentRun{ID: pulid.MustNew("ar_")}
	if _, err := r.watcher.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{},
		agentcontract.AgentRunWorkflowName,
		&agentcontract.AgentRunPayload{RunID: run.ID},
	); err != nil {
		return nil, err
	}

	return run, nil
}

type leftoverWorkflowRun struct {
	client.WorkflowRun
}

func TestStartFreshRun_StopsTheRunAnEarlierBenchLeftAndFiresAgain(t *testing.T) {
	t.Parallel()

	leftover := "agent-run-agdef_left-subject-inst_left"
	starter := &leftoverStarter{}
	watcher := benchwatch.New(starter)
	runs := &leftoverRuns{watcher: watcher, starter: starter}
	runner := &Runner{
		runs:    runs,
		facts:   factRows(t, "SELECT workflow_id FROM agent_runs", []string{leftover}),
		watcher: watcher,
	}
	session := &deskbench.Session{
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	definition := &agentdefinition.Definition{ID: pulid.MustNew("agdef_")}
	fire := &benchscenario.Fire{Event: agenttrigger.EventKind("insight.detected")}

	fired, err := runner.startFreshRun(
		t.Context(), session, definition, fire, pulid.MustNew("inst_"), "case")
	require.NoError(t, err)
	require.NotNil(t, fired)
	assert.Equal(t, []string{leftover}, starter.cancelled, "the leftover run is stopped once")
	assert.Equal(t, 2, runs.starts, "the fire is tried again after the leftover stops")
}

func factRows(t *testing.T, statement string, rows []string) benchsql.FactSource {
	t.Helper()

	return func(_ context.Context, _ *deskbench.Session, query string) ([]string, error) {
		require.Contains(t, query, statement)

		return rows, nil
	}
}
