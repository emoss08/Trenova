package reflectionjobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	return env
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")}
}

func readyPlan() *agentports.ReflectionPlan {
	return &agentports.ReflectionPlan{
		TenantInfo:   tenant(),
		ReflectionID: pulid.ID("arfl_a"),
		Subject:      agentreflection.ReflectionSubjectThread,
		Request:      &llmports.StructuredCompletionRequest{TenantInfo: tenant()},
	}
}

func TestThreadReflectionWorkflow_WaitsForTheConversationToGoQuiet(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	var preparedAt time.Time

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TurnFinishedSignalName, TurnFinished{TurnID: pulid.ID("atrn_b")})
	}, 6*time.Minute)

	env.OnActivity(a.PrepareThreadReflectionActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *agentports.ReflectOnThreadRequest) (*agentports.ReflectionPlan, error) {
			preparedAt = env.Now()
			return readyPlan(), nil
		}).
		Once()
	env.OnActivity(a.ReflectionModelActivity, mock.Anything, mock.Anything).
		Return(&llmports.StructuredCompletionResult{Text: `{"lessons":[],"notes":""}`}, nil).
		Once()
	env.OnActivity(a.FinishReflectionActivity, mock.Anything, mock.Anything).
		Return(&agentports.ReflectionOutcome{Kept: 1}, nil).Once()

	started := env.Now()
	env.ExecuteWorkflow(ThreadReflectionWorkflow, &ThreadReflectionInput{
		TenantInfo: tenant(),
		ThreadID:   pulid.ID("athr_a"),
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.GreaterOrEqual(t, preparedAt.Sub(started), 6*time.Minute+QuietPeriod,
		"a turn finishing while it waited starts the quiet period again")

	var result ReflectionResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Rounds)
	assert.Equal(t, 1, result.Kept)
	env.AssertExpectations(t)
}

func TestThreadReflectionWorkflow_ABusyConversationIsReadWithinTheLongestWait(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	var preparedAt time.Time

	for minute := 5; minute < 120; minute += 5 {
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(TurnFinishedSignalName, TurnFinished{})
		}, time.Duration(minute)*time.Minute)
	}

	env.OnActivity(a.PrepareThreadReflectionActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *agentports.ReflectOnThreadRequest) (*agentports.ReflectionPlan, error) {
			if preparedAt.IsZero() {
				preparedAt = env.Now()
			}
			return &agentports.ReflectionPlan{TenantInfo: tenant()}, nil
		})

	started := env.Now()
	env.ExecuteWorkflow(ThreadReflectionWorkflow, &ThreadReflectionInput{
		TenantInfo: tenant(),
		ThreadID:   pulid.ID("athr_a"),
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.LessOrEqual(t, preparedAt.Sub(started), LongestWait)
}

func TestLookBack_AFailedModelCallIsRecordedAndNothingIsKept(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PrepareRunReflectionActivity, mock.Anything, mock.Anything).
		Return(readyPlan(), nil).Once()
	env.OnActivity(a.ReflectionModelActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("no provider", "ModelRefused", errors.New("no provider")))
	env.OnActivity(a.FailReflectionActivity, mock.Anything, mock.MatchedBy(
		func(req *agentports.FailReflectionRequest) bool {
			return req.Plan.ReflectionID == pulid.ID("arfl_a") && req.Message != ""
		},
	)).Return(nil).Once()

	env.ExecuteWorkflow(RunReflectionWorkflow, &RunReflectionInput{
		TenantInfo: tenant(),
		RunID:      pulid.ID("arun_a"),
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ReflectionResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Failed)
	assert.Equal(t, 0, result.Kept)
	env.AssertExpectations(t)
}

func TestLookBack_APlanWithNothingToAskSkipsTheModel(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PrepareRunReflectionActivity, mock.Anything, mock.Anything).
		Return(&agentports.ReflectionPlan{TenantInfo: tenant(), ReflectionID: pulid.ID("arfl_a")}, nil).
		Once()

	env.ExecuteWorkflow(RunReflectionWorkflow, &RunReflectionInput{
		TenantInfo: tenant(),
		RunID:      pulid.ID("arun_a"),
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ReflectionResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Skipped)
	env.AssertExpectations(t)
}
