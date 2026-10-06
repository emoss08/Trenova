package inboundjobs

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
)

func processEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{WorkflowExecutionTimeout: time.Hour})

	return env
}

func processPayload() *ProcessInboundMessagePayload {
	return &ProcessInboundMessagePayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		MessageID: pulid.MustNew("imsg_"),
	}
}

func TestProcessInboundMessageWorkflow_ReadsTheFetchedMessage(t *testing.T) {
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
			MessageID:      payload.MessageID,
			Status:         inboundmessage.StatusClassified,
			Classification: inboundmessage.ClassificationTender,
		}, nil).Once()

	env.ExecuteWorkflow(ProcessInboundMessageWorkflow, payload)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ProcessInboundMessageResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, inboundmessage.ClassificationTender, result.Classification)
	env.AssertExpectations(t)
}

// A message whose content could not be read is already with a person, saying
// why. Classifying its subject line alone and acting on it would be the desk
// guessing, so the workflow stops there.
func TestProcessInboundMessageWorkflow_StopsWhenTheContentCouldNotBeRead(t *testing.T) {
	t.Parallel()

	env := processEnv(t)
	payload := processPayload()
	var a *Activities

	env.OnActivity(a.FetchInboundContentActivity, mock.Anything, mock.Anything).
		Return(&FetchInboundContentResult{Readable: false}, nil).Once()

	env.ExecuteWorkflow(ProcessInboundMessageWorkflow, payload)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ProcessInboundMessageResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, payload.MessageID, result.MessageID)
	assert.Equal(t, inboundmessage.StatusInReview, result.Status)
	env.AssertNotCalled(t, "SettleInboundMessageActivity", mock.Anything, mock.Anything)
	env.AssertNotCalled(t, "ListInboundAttachmentsActivity", mock.Anything, mock.Anything)
}

// A fetch that fails outright, through every retry, is still written onto the
// message, so it never sits at Received looking as though it is being read.
func TestProcessInboundMessageWorkflow_RecordsAFetchThatFailedOutright(t *testing.T) {
	t.Parallel()

	env := processEnv(t)
	payload := processPayload()
	var a *Activities

	env.OnActivity(a.FetchInboundContentActivity, mock.Anything, mock.Anything).
		Return(nil, errors.New("database unreachable")).Times(fetchAttempts)

	var recorded *FailInboundMessagePayload
	env.OnActivity(a.FailInboundMessageActivity, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			recorded = args.Get(1).(*FailInboundMessagePayload)
		}).
		Return(nil).Once()

	env.ExecuteWorkflow(ProcessInboundMessageWorkflow, payload)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	require.NotNil(t, recorded)
	assert.Equal(t, inboundmessageservice.ContentUnavailableCode, recorded.Code)
	assert.Equal(t, payload.MessageID, recorded.MessageID)
	env.AssertNotCalled(t, "SettleInboundMessageActivity", mock.Anything, mock.Anything)
}
