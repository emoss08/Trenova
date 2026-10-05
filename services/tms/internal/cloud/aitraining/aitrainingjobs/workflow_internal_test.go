package aitrainingjobs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const testExportID = pulid.ID("aitx_1")

func organizations(n int) []ConsentingOrganization {
	out := make([]ConsentingOrganization, 0, n)
	for i := range n {
		out = append(out, ConsentingOrganization{
			OrganizationID: pulid.ID(fmt.Sprintf("org_%03d", i)),
			BusinessUnitID: pulid.ID("bu_1"),
			GrantedAt:      100,
		})
	}
	return out
}

func inactiveError() error {
	return temporal.NewNonRetryableApplicationError("inactive", errorTypeExportInactive, nil)
}

func TestAITrainingExportWorkflow_ExportsEachOrganizationThenFinishes(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	orgs := organizations(2)
	env.OnActivity(a.BeginAITrainingExportActivity, mock.Anything, mock.Anything).Return(nil).Once()
	env.OnActivity(a.ListAITrainingOrganizationsActivity, mock.Anything, mock.MatchedBy(
		func(in *ListOrganizationsInput) bool { return in.AfterOrganizationID.IsNil() },
	)).Return(orgs, nil).Once()
	env.OnActivity(a.ExportAITrainingOrganizationActivity, mock.Anything, mock.MatchedBy(
		func(in *ExportOrganizationInput) bool {
			return in.Ordinal == 1 && in.Organization.OrganizationID == orgs[0].OrganizationID
		},
	)).Return(&ExportOrganizationOutcome{Examples: 3}, nil).Once()
	env.OnActivity(a.ExportAITrainingOrganizationActivity, mock.Anything, mock.MatchedBy(
		func(in *ExportOrganizationInput) bool {
			return in.Ordinal == 2 && in.Organization.OrganizationID == orgs[1].OrganizationID
		},
	)).Return(&ExportOrganizationOutcome{ConsentWithdrawn: true}, nil).Once()
	env.OnActivity(a.FinishAITrainingExportActivity, mock.Anything, mock.Anything).
		Return(&ExportOutcome{Status: "Completed", Examples: 3}, nil).Once()

	env.ExecuteWorkflow(AITrainingExportWorkflow, &ExportPayload{ExportID: testExportID})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var outcome ExportOutcome
	require.NoError(t, env.GetWorkflowResult(&outcome))
	assert.Equal(t, 3, outcome.Examples)
	env.AssertExpectations(t)
}

func TestAITrainingExportWorkflow_ContinuesAsNewWithItsCursor(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	orgs := organizations(organizationsPerExecution + 5)
	env.OnActivity(a.BeginAITrainingExportActivity, mock.Anything, mock.Anything).Return(nil).Once()
	env.OnActivity(a.ListAITrainingOrganizationsActivity, mock.Anything, mock.Anything).Return(orgs, nil).Once()
	env.OnActivity(a.ExportAITrainingOrganizationActivity, mock.Anything, mock.Anything).
		Return(&ExportOrganizationOutcome{Examples: 1}, nil).Times(organizationsPerExecution)

	env.ExecuteWorkflow(AITrainingExportWorkflow, &ExportPayload{ExportID: testExportID})

	require.True(t, env.IsWorkflowCompleted())
	var continued *workflow.ContinueAsNewError
	require.ErrorAs(t, env.GetWorkflowError(), &continued)
	env.AssertNotCalled(t, "FinishAITrainingExportActivity", mock.Anything, mock.Anything)
}

func TestAITrainingExportWorkflow_ResumesFromItsCursorWithoutBeginning(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.ListAITrainingOrganizationsActivity, mock.Anything, mock.MatchedBy(
		func(in *ListOrganizationsInput) bool { return in.AfterOrganizationID == "org_024" },
	)).Return([]ConsentingOrganization{{OrganizationID: "org_025", BusinessUnitID: "bu_1"}}, nil).Once()
	env.OnActivity(a.ExportAITrainingOrganizationActivity, mock.Anything, mock.MatchedBy(
		func(in *ExportOrganizationInput) bool { return in.Ordinal == 26 },
	)).Return(&ExportOrganizationOutcome{Examples: 1}, nil).Once()
	env.OnActivity(a.FinishAITrainingExportActivity, mock.Anything, mock.Anything).
		Return(&ExportOutcome{Status: "Completed", Examples: 26}, nil).Once()

	env.ExecuteWorkflow(AITrainingExportWorkflow, &ExportPayload{
		ExportID:            testExportID,
		AfterOrganizationID: "org_024",
		AfterBusinessUnitID: "bu_1",
		NextOrdinal:         26,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "BeginAITrainingExportActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func TestAITrainingExportWorkflow_FinishesWhenCanceled(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.BeginAITrainingExportActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ListAITrainingOrganizationsActivity, mock.Anything, mock.Anything).
		Return(organizations(3), nil)
	env.OnActivity(a.ExportAITrainingOrganizationActivity, mock.Anything, mock.Anything).
		Return(nil, inactiveError()).Once()
	env.OnActivity(a.FinishAITrainingExportActivity, mock.Anything, mock.Anything).
		Return(&ExportOutcome{Status: "Canceled"}, nil).Once()

	env.ExecuteWorkflow(AITrainingExportWorkflow, &ExportPayload{ExportID: testExportID})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "FailAITrainingExportActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func TestAITrainingExportWorkflow_FailsTheExportOnAnError(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.BeginAITrainingExportActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ListAITrainingOrganizationsActivity, mock.Anything, mock.Anything).
		Return(organizations(1), nil)
	env.OnActivity(a.ExportAITrainingOrganizationActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("storage down", "storage", errors.New("boom")))
	env.OnActivity(a.FailAITrainingExportActivity, mock.Anything, mock.MatchedBy(
		func(in *FailInput) bool { return in.ExportID == testExportID && in.Message != "" },
	)).Return(nil).Once()

	env.ExecuteWorkflow(AITrainingExportWorkflow, &ExportPayload{ExportID: testExportID})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "FinishAITrainingExportActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func TestClassifyMarksInactiveExportsNonRetryable(t *testing.T) {
	t.Parallel()

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, classify(fmt.Errorf("wrap: %w", aitraining.ErrExportInactive)), &appErr)
	assert.Equal(t, errorTypeExportInactive, appErr.Type())
	assert.True(t, appErr.NonRetryable())

	plain := errors.New("plain")
	assert.Equal(t, plain, classify(plain))
}
