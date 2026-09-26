package extractionevaljobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func runPayload() *RunPayload {
	return &RunPayload{
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		RunID:          pulid.ID("eer_1"),
	}
}

func TestExtractionEvalRunWorkflow_EvaluatesEveryCaseThenFinishes(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.BeginExtractionEvalRunActivity, mock.Anything, mock.Anything).Return(nil).Once()
	env.OnActivity(a.ListPendingExtractionEvalCasesActivity, mock.Anything, mock.MatchedBy(
		func(in *ListPendingInput) bool { return in.AfterOrdinal == 0 },
	)).Return([]PendingCase{{ResultID: "eeres_1", Ordinal: 1}, {ResultID: "eeres_2", Ordinal: 2}}, nil).Once()
	env.OnActivity(a.ListPendingExtractionEvalCasesActivity, mock.Anything, mock.MatchedBy(
		func(in *ListPendingInput) bool { return in.AfterOrdinal == 2 },
	)).Return([]PendingCase{}, nil).Once()
	env.OnActivity(a.CheckExtractionEvalBudgetActivity, mock.Anything, mock.Anything).
		Return(&ContinueDecision{}, nil).Times(2)
	env.OnActivity(a.EvaluateExtractionEvalCaseActivity, mock.Anything, mock.Anything).Return(nil).Times(2)
	env.OnActivity(a.FinishExtractionEvalRunActivity, mock.Anything, mock.MatchedBy(
		func(in *FinishInput) bool { return in.Status == extractioneval.RunStatusCompleted.String() },
	)).Return(&RunOutcome{RunID: "eer_1", Status: "Completed"}, nil).Once()

	env.ExecuteWorkflow(ExtractionEvalRunWorkflow, runPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var outcome RunOutcome
	require.NoError(t, env.GetWorkflowResult(&outcome))
	assert.Equal(t, "Completed", outcome.Status)
	env.AssertExpectations(t)
}

func TestExtractionEvalRunWorkflow_StopsAtTheBudget(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.BeginExtractionEvalRunActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ListPendingExtractionEvalCasesActivity, mock.Anything, mock.Anything).
		Return([]PendingCase{{ResultID: "eeres_1", Ordinal: 1}}, nil)
	env.OnActivity(a.CheckExtractionEvalBudgetActivity, mock.Anything, mock.Anything).
		Return(&ContinueDecision{Stop: true, Status: "BudgetStopped", Reason: "spent"}, nil)
	env.OnActivity(a.FinishExtractionEvalRunActivity, mock.Anything, mock.MatchedBy(
		func(in *FinishInput) bool { return in.Status == "BudgetStopped" && in.Reason == "spent" },
	)).Return(&RunOutcome{RunID: "eer_1", Status: "BudgetStopped"}, nil).Once()

	env.ExecuteWorkflow(ExtractionEvalRunWorkflow, runPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "EvaluateExtractionEvalCaseActivity", mock.Anything, mock.Anything)
}

func TestExtractionEvalRunWorkflow_FailsTheRunWhenACaseCannotBeSaved(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.BeginExtractionEvalRunActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ListPendingExtractionEvalCasesActivity, mock.Anything, mock.Anything).
		Return([]PendingCase{{ResultID: "eeres_1", Ordinal: 1}}, nil)
	env.OnActivity(a.CheckExtractionEvalBudgetActivity, mock.Anything, mock.Anything).
		Return(&ContinueDecision{}, nil)
	env.OnActivity(a.EvaluateExtractionEvalCaseActivity, mock.Anything, mock.Anything).
		Return(assert.AnError)
	env.OnActivity(a.FailExtractionEvalRunActivity, mock.Anything, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(ExtractionEvalRunWorkflow, runPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestExtractionEvalRunWorkflow_ContinuesAsNewOnLongRuns(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	pages := make([][]PendingCase, 0, 3)
	for page := range 3 {
		cases := make([]PendingCase, 0, pendingPageSize)
		for i := range pendingPageSize {
			ordinal := page*pendingPageSize + i + 1
			cases = append(cases, PendingCase{ResultID: pulid.MustNew("eeres_"), Ordinal: ordinal})
		}
		pages = append(pages, cases)
	}

	var a *Activities
	env.OnActivity(a.BeginExtractionEvalRunActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.ListPendingExtractionEvalCasesActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *ListPendingInput) ([]PendingCase, error) {
			return pages[in.AfterOrdinal/pendingPageSize], nil
		})
	env.OnActivity(a.CheckExtractionEvalBudgetActivity, mock.Anything, mock.Anything).
		Return(&ContinueDecision{}, nil)
	env.OnActivity(a.EvaluateExtractionEvalCaseActivity, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(ExtractionEvalRunWorkflow, runPayload())

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)
	var continued *workflow.ContinueAsNewError
	require.ErrorAs(t, err, &continued)
	assert.Equal(t, ExtractionEvalRunWorkflowName, continued.WorkflowType.Name)
}
