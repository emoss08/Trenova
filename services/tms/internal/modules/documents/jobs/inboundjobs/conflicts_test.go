package inboundjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProcessInboundMessageWorkflow_ChecksConflictsOnceSettled(t *testing.T) {
	t.Parallel()

	env := processEnv(t)
	payload := processPayload()
	var a *Activities

	env.OnActivity(a.FetchInboundContentActivity, mock.Anything, mock.Anything).
		Return(&FetchInboundContentResult{Readable: true}, nil).Once()
	env.OnActivity(a.ListInboundAttachmentsActivity, mock.Anything, mock.Anything).
		Return(&ListInboundAttachmentsResult{}, nil).Once()
	env.OnActivity(a.SettleInboundMessageActivity, mock.Anything, mock.Anything).
		Return(&ProcessInboundMessageResult{
			MessageID: payload.MessageID,
			Status:    inboundmessage.StatusInReview,
		}, nil).Once()

	var checked *ProcessInboundMessagePayload
	env.OnActivity(a.CheckInboundConflictsActivity, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			checked = args.Get(1).(*ProcessInboundMessagePayload)
		}).
		Return(nil).Once()

	env.ExecuteWorkflow(ProcessInboundMessageWorkflow, payload)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.NotNil(t, checked)
	assert.Equal(t, payload.MessageID, checked.MessageID)
	env.AssertExpectations(t)
}

func TestProcessInboundMessageWorkflow_AFailedConflictCheckLeavesTheMessageSettled(t *testing.T) {
	t.Parallel()

	env := processEnv(t)
	payload := processPayload()
	var a *Activities

	env.OnActivity(a.FetchInboundContentActivity, mock.Anything, mock.Anything).
		Return(&FetchInboundContentResult{Readable: true}, nil).Once()
	env.OnActivity(a.ListInboundAttachmentsActivity, mock.Anything, mock.Anything).
		Return(&ListInboundAttachmentsResult{}, nil).Once()
	env.OnActivity(a.SettleInboundMessageActivity, mock.Anything, mock.Anything).
		Return(&ProcessInboundMessageResult{
			MessageID: payload.MessageID,
			Status:    inboundmessage.StatusInReview,
		}, nil).Once()
	env.OnActivity(a.CheckInboundConflictsActivity, mock.Anything, mock.Anything).
		Return(errors.New("database unreachable"))

	env.ExecuteWorkflow(ProcessInboundMessageWorkflow, payload)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ProcessInboundMessageResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, inboundmessage.StatusInReview, result.Status)
}

func TestCheckInboundConflictsActivity_IsANoOpWithoutTheService(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&Activities{}).CheckInboundConflictsActivity(t.Context(), processPayload()))
}
