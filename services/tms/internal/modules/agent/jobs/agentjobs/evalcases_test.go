package agentjobs

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestCaptureEvalCaseCandidatesWorkflow_StopsWhenARoundCapturesNothing(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	rounds := []agentports.CaptureEvalCaseCandidatesResult{
		{Scanned: evalCaseCaptureBatch, Captured: 90, Duplicates: 4, Failed: 6},
		{Scanned: evalCaseCaptureBatch, Failed: evalCaseCaptureBatch},
		{Scanned: 3, Captured: 3},
	}
	var sinces []int64
	var a *Activities
	env.OnActivity(a.CaptureEvalCaseCandidatesActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *CaptureEvalCaseCandidatesInput,
		) (*agentports.CaptureEvalCaseCandidatesResult, error) {
			sinces = append(sinces, input.Since)
			next := rounds[0]
			rounds = rounds[1:]
			return &next, nil
		})

	env.ExecuteWorkflow(CaptureEvalCaseCandidatesWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result agentports.CaptureEvalCaseCandidatesResult
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.Len(t, sinces, 2, "a round of failures ends the sweep instead of spinning on it")
	assert.Equal(t, sinces[0], sinces[1], "one window for the whole sweep")
	assert.Less(t, sinces[0], time.Now().Add(-5*time.Hour).Unix())
	assert.Equal(t, 90, result.Captured)
	assert.Equal(t, evalCaseCaptureBatch+6, result.Failed)
	assert.Equal(t, 2*evalCaseCaptureBatch, result.Scanned)
}

func TestPurgeEvalCasesWorkflow_PurgesAtTheWorkflowsClock(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	var now int64
	var a *Activities
	env.OnActivity(a.PurgeEvalCasesActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input *PurgeEvalCasesInput,
		) (*agentports.PurgeEvalCasesResult, error) {
			now = input.Now
			return &agentports.PurgeEvalCasesResult{Retained: 2, Expired: 1, Orphaned: 3}, nil
		}).Once()

	env.ExecuteWorkflow(PurgeEvalCasesWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result agentports.PurgeEvalCasesResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Positive(t, now)
	assert.Equal(t, agentports.PurgeEvalCasesResult{Retained: 2, Expired: 1, Orphaned: 3}, result)
}
