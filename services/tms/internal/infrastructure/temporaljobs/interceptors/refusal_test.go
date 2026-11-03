package interceptors

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const refusalTestActivity = "RefusalTestActivity"

func retryingWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 1,
			MaximumAttempts:    5,
		},
	})

	return workflow.ExecuteActivity(ctx, refusalTestActivity).Get(ctx, nil)
}

// attemptsUntilDone runs an activity that always fails with err under a retry
// policy of five attempts, through the refusal interceptor, and counts the
// attempts it took.
func attemptsUntilDone(t *testing.T, err error) (int32, error) {
	t.Helper()

	var attempts atomic.Int32
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{NewRefusalInterceptor()},
	})
	env.RegisterWorkflow(retryingWorkflow)
	env.RegisterActivityWithOptions(func(context.Context) error {
		attempts.Add(1)
		return err
	}, activity.RegisterOptions{Name: refusalTestActivity})

	env.ExecuteWorkflow(retryingWorkflow)
	require.True(t, env.IsWorkflowCompleted())

	return attempts.Load(), env.GetWorkflowError()
}

func TestRefusalInterceptor_AMissingRecordFailsOnTheFirstAttempt(t *testing.T) {
	t.Parallel()

	attempts, err := attemptsUntilDone(
		t,
		errortypes.NewNotFoundError("The document's file could not be found in storage"),
	)

	require.Error(t, err)
	assert.Equal(t, int32(1), attempts)
	appErr, ok := errors.AsType[*temporal.ApplicationError](err)
	require.True(t, ok)
	assert.Equal(t, "resource_not_found", appErr.Type())
}

func TestRefusalInterceptor_AnOutageStillRetriesUnderItsPolicy(t *testing.T) {
	t.Parallel()

	attempts, err := attemptsUntilDone(t, errors.New("connection reset by peer"))

	require.Error(t, err)
	assert.Equal(t, int32(5), attempts)
}
