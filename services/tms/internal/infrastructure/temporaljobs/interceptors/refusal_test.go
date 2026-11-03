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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

// loggedAttempts runs an activity that always fails with err under a retry
// policy of three attempts, through the worker's logging and refusal
// interceptors, and returns the level of each line the attempts logged. The
// test environment does not hand an activity its retry policy, so only a
// failure that is final whatever the policy can be asserted this way.
func loggedAttempts(t *testing.T, err error) []zapcore.Level {
	t.Helper()

	core, logs := observer.New(zapcore.DebugLevel)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{
			NewFailureLoggingInterceptor(zap.New(core)),
			NewRefusalInterceptor(),
		},
	})
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 1,
				MaximumAttempts:    3,
			},
		})
		return workflow.ExecuteActivity(ctx, refusalTestActivity).Get(ctx, nil)
	}, workflow.RegisterOptions{Name: "LoggedAttemptsWorkflow"})
	env.RegisterActivityWithOptions(func(context.Context) error {
		return err
	}, activity.RegisterOptions{Name: refusalTestActivity})

	env.ExecuteWorkflow("LoggedAttemptsWorkflow")
	require.True(t, env.IsWorkflowCompleted())

	levels := make([]zapcore.Level, 0)
	for _, entry := range logs.FilterField(zap.String("type", "activity")).All() {
		assert.Equal(t, refusalTestActivity, entry.ContextMap()["activityType"])
		levels = append(levels, entry.Level)
	}

	return levels
}

func TestRetryFollows_OnlyWhileThePolicyWillTryAgain(t *testing.T) {
	t.Parallel()

	outage := errors.New("connection reset by peer")
	policy := &temporal.RetryPolicy{
		MaximumAttempts:        3,
		NonRetryableErrorTypes: []string{"model_rejected"},
	}

	assert.True(t, retryFollows(outage, 1, policy))
	assert.True(t, retryFollows(outage, 2, policy))
	assert.False(t, retryFollows(outage, 3, policy), "the last attempt is final")
	assert.True(t, retryFollows(outage, 40, &temporal.RetryPolicy{}), "no cap retries without end")
	assert.True(t, retryFollows(outage, 40, nil))
	assert.False(t, retryFollows(
		temporal.NewNonRetryableApplicationError("gone", "resource_not_found", nil), 1, policy))
	assert.False(t, retryFollows(
		temporal.NewApplicationError("refused", "model_rejected"), 1, policy),
		"a type the policy names does not retry")
	assert.False(t, retryFollows(context.Canceled, 1, policy))
}

func TestLoggingInterceptor_AFinalRefusalIsOneError(t *testing.T) {
	t.Parallel()

	levels := loggedAttempts(t, errortypes.NewNotFoundError("Invoice not found"))

	assert.Equal(t, []zapcore.Level{zapcore.ErrorLevel}, levels)
}
