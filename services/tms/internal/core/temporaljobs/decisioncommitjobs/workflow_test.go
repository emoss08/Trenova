package decisioncommitjobs

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type commitEnv struct {
	env       *testsuite.TestWorkflowEnvironment
	started   time.Time
	committed []time.Time
}

func newCommitEnv(t *testing.T) *commitEnv {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProposalCommitWorkflow)

	var a *Activities
	env.RegisterActivity(a.CommitApprovalActivity)

	c := &commitEnv{env: env, started: env.Now()}
	env.OnActivity("CommitApprovalActivity", mock.Anything, mock.Anything).
		Return(func(context.Context, *services.CommitApprovalRequest) error {
			c.committed = append(c.committed, env.Now())

			return nil
		})

	return c
}

func (c *commitEnv) run() *CommitResult {
	c.env.ExecuteWorkflow(ProposalCommitWorkflow, &services.CommitApprovalRequest{
		OrganizationID:  pulid.ID("org_test"),
		BusinessUnitID:  pulid.ID("bu_test"),
		DecidedByUserID: pulid.ID("usr_test"),
		WorkflowID:      WorkflowID(pulid.ID("ap_test")),
		CommitsAt:       c.started.Add(services.ApprovalUndoWindow).Unix(),
	})

	var result CommitResult
	_ = c.env.GetWorkflowResult(&result)

	return &result
}

// Left alone, the approval commits once its window closes, and not before.
func TestProposalCommitWorkflow_CommitsWhenTheWindowCloses(t *testing.T) {
	t.Parallel()

	c := newCommitEnv(t)
	result := c.run()

	require.True(t, c.env.IsWorkflowCompleted())
	require.NoError(t, c.env.GetWorkflowError())
	assert.True(t, result.Committed)
	require.Len(t, c.committed, 1)
	assert.GreaterOrEqual(t, c.committed[0].Sub(c.started), services.ApprovalUndoWindow-time.Second,
		"the commit waits out the window")
}

// An undo inside the window ends the wait without committing anything.
func TestProposalCommitWorkflow_UndoBeforeTheCommitCommitsNothing(t *testing.T) {
	t.Parallel()

	c := newCommitEnv(t)
	c.env.RegisterDelayedCallback(func() {
		c.env.SignalWorkflow(UndoSignalName, nil)
	}, 2*time.Second)
	result := c.run()

	require.True(t, c.env.IsWorkflowCompleted())
	require.NoError(t, c.env.GetWorkflowError())
	assert.True(t, result.Undone)
	assert.False(t, result.Committed)
	assert.Empty(t, c.committed)
}

// "Do it now" commits at once rather than at the close of the window.
func TestProposalCommitWorkflow_CommitNowSkipsTheRestOfTheWait(t *testing.T) {
	t.Parallel()

	c := newCommitEnv(t)
	c.env.RegisterDelayedCallback(func() {
		c.env.SignalWorkflow(CommitNowSignalName, nil)
	}, time.Second)
	result := c.run()

	require.True(t, c.env.IsWorkflowCompleted())
	require.NoError(t, c.env.GetWorkflowError())
	assert.True(t, result.Committed)
	require.Len(t, c.committed, 1)
	assert.Less(t, c.committed[0].Sub(c.started), 2*time.Second)
}

// The window is rounded up to a whole second so the client's countdown never
// starts short of it.
func TestCommitsAt_RoundsTheWindowUp(t *testing.T) {
	t.Parallel()

	whole := time.Unix(1_000, 0)
	assert.Equal(t, int64(1_005), CommitsAt(whole))
	assert.Equal(t, int64(1_006), CommitsAt(whole.Add(300*time.Millisecond)))
}
