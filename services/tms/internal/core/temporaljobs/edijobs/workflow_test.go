package edijobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func newDeliverWorkflowTestEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(DeliverEDIMessageWorkflow)
	return env
}

func deliverWorkflowPayload() *DeliverEDIMessageWorkflowPayload {
	return &DeliverEDIMessageWorkflowPayload{
		MessageID: pulid.MustNew("edimsg_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}
}

func TestDeliverEDIMessageWorkflow_Success(t *testing.T) {
	env := newDeliverWorkflowTestEnv(t)
	payload := deliverWorkflowPayload()
	var a *Activities

	env.OnActivity(a.DeliverEDIMessageActivity, mock.Anything, mock.Anything).
		Return(&DeliverEDIMessageWorkflowResult{
			MessageID:      payload.MessageID,
			DeliveryStatus: edi.MessageDeliveryStatusSent,
			RemotePath:     "/outbound/file.x12",
		}, nil).
		Once()

	env.ExecuteWorkflow(DeliverEDIMessageWorkflow, payload)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	result := new(DeliverEDIMessageWorkflowResult)
	require.NoError(t, env.GetWorkflowResult(result))
	require.Equal(t, edi.MessageDeliveryStatusSent, result.DeliveryStatus)
	env.AssertNotCalled(t, "MarkEDIMessageDeadLetteredActivity")
}

func TestDeliverEDIMessageWorkflow_DeadLettersAfterExhaustedRetries(t *testing.T) {
	env := newDeliverWorkflowTestEnv(t)
	payload := deliverWorkflowPayload()
	var a *Activities
	deliveryErr := errors.New("connect SFTP server: connection refused")

	env.OnActivity(a.DeliverEDIMessageActivity, mock.Anything, mock.Anything).
		Return(nil, deliveryErr).
		Times(6)
	env.OnActivity(a.MarkEDIMessageDeadLetteredActivity, mock.Anything, mock.MatchedBy(
		func(deadLetter *MarkEDIMessageDeadLetteredPayload) bool {
			return deadLetter.MessageID == payload.MessageID && deadLetter.Reason != ""
		},
	)).
		Return(nil).
		Once()

	env.ExecuteWorkflow(DeliverEDIMessageWorkflow, payload)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}
