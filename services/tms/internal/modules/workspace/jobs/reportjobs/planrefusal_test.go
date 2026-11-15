package reportjobs

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

const planWording = "This report reads data your plan does not include: " +
	"Factoring Schedules are part of the Accounting & Cash add-on, which is not on your plan"

func planRefusal() *errortypes.PlanRestrictionError {
	restriction := errortypes.NewFeatureRestrictionError("tms.factoring", "Accounting & Cash")
	restriction.Message = "This report reads data your plan does not include: {0}"
	restriction.Args = []any{
		"Factoring Schedules are part of the Accounting & Cash add-on, which is not on your plan",
	}
	return restriction
}

func TestClassifyCompileError_APlanRefusalFailsTheRunWithoutRetrying(t *testing.T) {
	t.Parallel()

	err := classifyCompileError(fmt.Errorf("check: %w", planRefusal()))

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	assert.True(t, appErr.NonRetryable())
	assert.Equal(t, ErrTypeReportPlan, appErr.Type())

	runErr := runErrorFrom(err)
	assert.Equal(t, "PLAN_RESTRICTED", runErr.Code)
	assert.Equal(t, planWording, runErr.Message)
}

func TestRunReportWorkflow_APlanRefusalIsNotRetriedAndToldWithThePlanWording(t *testing.T) {
	env := newEnv(t)

	attempts := 0
	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Run(func(mock.Arguments) { attempts++ }).
		Return(nil, classifyCompileError(planRefusal()))

	var finalized *FinalizePayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizePayload)
		}).
		Return(nil)

	env.ExecuteWorkflow(RunReportWorkflow, testPayload())
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	assert.Equal(t, 1, attempts, "a plan refusal is not transient, so it is not retried")
	require.NotNil(t, finalized)
	assert.Equal(t, report.RunStatusFailed, finalized.Status)
	require.NotNil(t, finalized.Error)
	assert.Equal(t, ErrTypeReportPlan, finalized.Error.Code)
	assert.Equal(t, planWording, finalized.Error.Message)

	notice := runNotificationContext(&report.ReportRun{
		Status: report.RunStatusFailed,
		Error:  finalized.Error,
	}, "Factoring Schedules")
	assert.Equal(t, planWording, notice.Reason,
		"the owner of a scheduled run is told the add-on it needs")
}

func TestRunRetryPolicies_NeverRetryAPlanRefusal(t *testing.T) {
	t.Parallel()

	assert.Contains(t, prepareActivityOptions.RetryPolicy.NonRetryableErrorTypes, ErrTypeReportPlan)
}
