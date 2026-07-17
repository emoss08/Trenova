package reportjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func testPayload() *RunReportPayload {
	return &RunReportPayload{
		RunID:          pulid.ID("rrun_test"),
		OrganizationID: pulid.ID("org_test"),
		BusinessUnitID: pulid.ID("bu_test"),
	}
}

func testPrepared() *PreparedRun {
	return &PreparedRun{
		RunID:          pulid.ID("rrun_test"),
		OrganizationID: pulid.ID("org_test"),
		BusinessUnitID: pulid.ID("bu_test"),
		RequestedByID:  pulid.ID("usr_test"),
		RevisionID:     pulid.ID("rdr_test"),
		Format:         report.FormatCSV,
		Title:          "Test Report",
		MaxRunSeconds:  60,
	}
}

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(RunReportWorkflow)

	var a *Activities
	env.RegisterActivity(a.PrepareRunActivity)
	env.RegisterActivity(a.ExecuteAndRenderActivity)
	env.RegisterActivity(a.FinalizeRunActivity)
	return env
}

func TestRunReportWorkflowSuccess(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(testPrepared(), nil)
	env.OnActivity("ExecuteAndRenderActivity", mock.Anything, mock.Anything).
		Return(&ExecuteResult{
			ArtifactKey: "reports/org_test/rrun_test/1/report.csv",
			RowCount:    42,
			ByteSize:    1024,
		}, nil)

	var finalized *FinalizePayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizePayload)
		}).
		Return(nil)

	env.ExecuteWorkflow(RunReportWorkflow, testPayload())
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result RunReportResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, report.RunStatusSucceeded, result.Status)
	assert.Equal(t, int64(42), result.RowCount)

	require.NotNil(t, finalized)
	assert.Equal(t, report.RunStatusSucceeded, finalized.Status)
	assert.Equal(t, "reports/org_test/rrun_test/1/report.csv", finalized.ArtifactKey)
}

func TestRunReportWorkflowValidationFailureFinalizesFailed(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"unknown field", ErrTypeReportValidation, errors.New("unknown field"),
		))

	var finalized *FinalizePayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizePayload)
		}).
		Return(nil)

	env.ExecuteWorkflow(RunReportWorkflow, testPayload())
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError(), "the workflow surfaces the failure")

	require.NotNil(t, finalized, "a failed run must still be finalized")
	assert.Equal(t, report.RunStatusFailed, finalized.Status)
	require.NotNil(t, finalized.Error)
	assert.Equal(t, ErrTypeReportValidation, finalized.Error.Code)
}

func TestRunReportWorkflowExecuteFailureFinalizesFailed(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(testPrepared(), nil)
	env.OnActivity("ExecuteAndRenderActivity", mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError(
			"too expensive", ErrTypeReportTooExpensive, errors.New("cost"),
		))

	var finalized *FinalizePayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizePayload)
		}).
		Return(nil)

	env.ExecuteWorkflow(RunReportWorkflow, testPayload())
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	require.NotNil(t, finalized)
	assert.Equal(t, report.RunStatusFailed, finalized.Status)
	assert.Equal(t, ErrTypeReportTooExpensive, finalized.Error.Code)
}

func TestRunReportWorkflowCancellationFinalizesCanceled(t *testing.T) {
	env := newEnv(t)

	env.OnActivity("PrepareRunActivity", mock.Anything, mock.Anything).
		Return(testPrepared(), nil)
	env.OnActivity("ExecuteAndRenderActivity", mock.Anything, mock.Anything).
		Return(nil, temporal.NewCanceledError("canceled by user"))

	var finalized *FinalizePayload
	env.OnActivity("FinalizeRunActivity", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			finalized = args.Get(1).(*FinalizePayload)
		}).
		Return(nil)

	env.ExecuteWorkflow(RunReportWorkflow, testPayload())
	require.True(t, env.IsWorkflowCompleted())

	require.NotNil(t, finalized, "a canceled run must still be finalized")
	assert.Equal(t, report.RunStatusCanceled, finalized.Status)
}
