package factoringjobs

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type stagePoller struct {
	mu        sync.Mutex
	readyAt   int
	calls     int
	gaveUpRun bool
}

func (p *stagePoller) poll(
	_ context.Context,
	payload *factoringcontract.RemittanceStagePayload,
) (*factoringports.StagePendingResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if payload.GiveUp {
		p.gaveUpRun = true
		return &factoringports.StagePendingResult{
			Remittance: &factoring.Remittance{Status: factoring.RemittanceStatusFailed},
		}, nil
	}
	if p.readyAt > 0 && p.calls >= p.readyAt {
		return &factoringports.StagePendingResult{
			Remittance: &factoring.Remittance{Status: factoring.RemittanceStatusParsed},
		}, nil
	}
	return &factoringports.StagePendingResult{
		Remittance: &factoring.Remittance{Status: factoring.RemittanceStatusPending},
		Waiting:    true,
	}, nil
}

func stagePayload() *factoringcontract.RemittanceStagePayload {
	return &factoringcontract.RemittanceStagePayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		RemittanceID:   pulid.MustNew("facrem_"),
	}
}

func TestRemittanceStageWorkflowPollsUntilThePdfIsRead(t *testing.T) {
	t.Parallel()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	var a *Activities
	poller := &stagePoller{readyAt: 3}
	env.OnActivity(a.StageFactoringRemittanceActivity, mock.Anything, mock.Anything).
		Return(poller.poll)

	env.ExecuteWorkflow(FactoringRemittanceStageWorkflow, stagePayload())
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result RemittanceStageResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 3, result.Polls)
	assert.False(t, result.Waiting)
	assert.Equal(t, string(factoring.RemittanceStatusParsed), result.Status)
	assert.False(t, poller.gaveUpRun)
}

func TestRemittanceStageWorkflowGivesUpAtItsDeadline(t *testing.T) {
	t.Parallel()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	var a *Activities
	poller := &stagePoller{}
	env.OnActivity(a.StageFactoringRemittanceActivity, mock.Anything, mock.Anything).
		Return(poller.poll)

	env.ExecuteWorkflow(FactoringRemittanceStageWorkflow, stagePayload())
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result RemittanceStageResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.True(t, poller.gaveUpRun, "the last poll fails the remittance rather than leaving it waiting")
	assert.Equal(t, string(factoring.RemittanceStatusFailed), result.Status)
	assert.LessOrEqual(t, result.Polls, int(remittanceStageDeadline/remittanceStagePoll)+1)
}
