package extractionshadowjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func shadowPayload() *ShadowPayload {
	return &ShadowPayload{
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		ResultID:       pulid.ID("exsr_1"),
	}
}

func TestExtractionShadowWorkflow_RunsTheShadowOnce(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.RunExtractionShadowActivity, mock.Anything, mock.MatchedBy(
		func(in *ShadowPayload) bool { return in.ResultID == "exsr_1" },
	)).Return(nil).Once()

	env.ExecuteWorkflow(ExtractionShadowWorkflow, shadowPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestExtractionShadowWorkflow_RecordsAShadowThatNeverFinished(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.RunExtractionShadowActivity, mock.Anything, mock.Anything).
		Return(errors.New("database unavailable")).Times(runAttempts)

	var failed *FailInput
	env.OnActivity(a.FailExtractionShadowActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *FailInput) error {
			failed = in
			return nil
		}).Once()

	env.ExecuteWorkflow(ExtractionShadowWorkflow, shadowPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	require.NotNil(t, failed)
	assert.Equal(t, pulid.ID("exsr_1"), failed.ResultID)
	assert.Equal(t, unfinishedMessage, failed.Message)
	env.AssertExpectations(t)
}
