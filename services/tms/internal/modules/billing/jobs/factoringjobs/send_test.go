package factoringjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func sendPayload() *factoringcontract.SendPayload {
	return &factoringcontract.SendPayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		SubmissionID:   pulid.MustNew("facsub_"),
		Group:          1,
	}
}

func TestSendWorkflowPreparesThenDelivers(t *testing.T) {
	t.Parallel()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	var a *Activities
	env.OnActivity(a.PrepareFactoringSubmissionActivity, mock.Anything, mock.Anything).
		Return(nil).
		Once()
	env.OnActivity(a.DeliverFactoringSubmissionActivity, mock.Anything, mock.Anything).
		Return(&factoringsubmissionports.DeliverSubmissionResult{
			Status:    factoringsubmission.SubmissionStatusSent,
			Reference: "/in/FS1",
			Parts:     1,
		}, nil).Once()

	env.ExecuteWorkflow(FactoringSubmissionSendWorkflow, sendPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result factoringsubmissionports.DeliverSubmissionResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, factoringsubmission.SubmissionStatusSent, result.Status)
	env.AssertExpectations(t)
}

func TestSendWorkflowMarksFailedAfterEveryRetry(t *testing.T) {
	t.Parallel()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	var a *Activities
	payload := sendPayload()
	env.OnActivity(a.PrepareFactoringSubmissionActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(a.DeliverFactoringSubmissionActivity, mock.Anything, mock.Anything).
		Return(nil, errors.New("dial tcp: connection refused")).Times(sendMaxAttempts)
	var failed *FailPayload
	env.OnActivity(a.MarkFactoringSubmissionFailedActivity, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { failed = args.Get(1).(*FailPayload) }).
		Return(nil).Once()

	env.ExecuteWorkflow(FactoringSubmissionSendWorkflow, payload)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	require.NotNil(t, failed)
	assert.Equal(t, payload.SubmissionID, failed.Send.SubmissionID)
	assert.Equal(t, sendMaxAttempts, failed.Attempts)
	assert.Contains(t, failed.Cause, "connection refused")
	env.AssertExpectations(t)
}

func TestSendWorkflowStopsAtARefusal(t *testing.T) {
	t.Parallel()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	var a *Activities
	env.OnActivity(a.PrepareFactoringSubmissionActivity, mock.Anything, mock.Anything).
		Return(temporal.NewNonRetryableApplicationError(
			"INV-1 is too large to email",
			ErrTypeSendRefused,
			nil,
		)).Once()
	var failed *FailPayload
	env.OnActivity(a.MarkFactoringSubmissionFailedActivity, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { failed = args.Get(1).(*FailPayload) }).
		Return(nil).Once()

	env.ExecuteWorkflow(FactoringSubmissionSendWorkflow, sendPayload())

	require.Error(t, env.GetWorkflowError())
	require.NotNil(t, failed)
	assert.Equal(t, 1, failed.Attempts)
	assert.Equal(t, "INV-1 is too large to email", failed.Cause)
	env.AssertExpectations(t)
}

func TestRefusalIsNonRetryableOnlyForRefusals(t *testing.T) {
	t.Parallel()

	refused := refusal(errortypes.NewBusinessError("no transport"))
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, refused, &appErr)
	assert.True(t, appErr.NonRetryable())
	assert.Equal(t, ErrTypeSendRefused, appErr.Type())

	transient := errors.New("timeout")
	assert.Equal(t, transient, refusal(transient))
	assert.NoError(t, refusal(nil))
}
