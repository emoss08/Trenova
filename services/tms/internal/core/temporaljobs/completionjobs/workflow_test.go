package completionjobs

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
)

type rejected struct{}

func (rejected) Error() string           { return "provider answered 400" }
func (rejected) ProviderStatus() int     { return http.StatusBadRequest }
func (rejected) ProviderRetryable() bool { return false }

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{WorkflowExecutionTimeout: time.Minute})

	return env
}

func request() *agentports.StructuredCompletionRequest {
	return &agentports.StructuredCompletionRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		System:     "Answer in JSON.",
	}
}

func TestStructuredCompletionWorkflow_ReturnsTheModelsAnswer(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	env.OnActivity(a.CompleteStructuredActivity, mock.Anything, mock.Anything).
		Return(&agentports.StructuredCompletionResult{Text: `{"ok":true}`, ModelIdentifier: "m"}, nil).
		Once()

	env.ExecuteWorkflow(
		StructuredCompletionWorkflow,
		&StructuredCompletionPayload{Request: request()},
	)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result agentports.StructuredCompletionResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.JSONEq(t, `{"ok":true}`, result.Text)
	env.AssertExpectations(t)
}

// A transient failure is retried inside the time the caller waits.
func TestStructuredCompletionWorkflow_RetriesATransientFailure(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	calls := 0
	env.OnActivity(a.CompleteStructuredActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *agentports.StructuredCompletionRequest) (*agentports.StructuredCompletionResult, error) {
			calls++
			if calls == 1 {
				return nil, modelcall.Classify(context.DeadlineExceeded)
			}

			return &agentports.StructuredCompletionResult{Text: "{}"}, nil
		})

	env.ExecuteWorkflow(
		StructuredCompletionWorkflow,
		&StructuredCompletionPayload{Request: request()},
	)

	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, 2, calls)
}

// A request the provider rejected is rejected however often it is sent, and
// the caller still learns it was a rejection.
func TestStructuredCompletionWorkflow_DoesNotRetryARejection(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	calls := 0
	env.OnActivity(a.CompleteStructuredActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *agentports.StructuredCompletionRequest) (*agentports.StructuredCompletionResult, error) {
			calls++

			return nil, modelcall.Classify(rejected{})
		})

	env.ExecuteWorkflow(
		StructuredCompletionWorkflow,
		&StructuredCompletionPayload{Request: request()},
	)

	err := env.GetWorkflowError()
	require.Error(t, err)
	assert.Equal(t, 1, calls)

	var failure agentports.ProviderFailure
	require.ErrorAs(t, modelcall.Err(err), &failure)
	assert.Equal(t, http.StatusBadRequest, failure.ProviderStatus())
}

// A refusal reaches the person in its own words.
func TestStructuredCompletionWorkflow_KeepsARefusalWhole(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	env.OnActivity(a.CompleteStructuredActivity, mock.Anything, mock.Anything).
		Return(nil, modelcall.Classify(errortypes.NewBusinessError("AI is turned off")))

	env.ExecuteWorkflow(
		StructuredCompletionWorkflow,
		&StructuredCompletionPayload{Request: request()},
	)

	var business *errortypes.BusinessError
	require.ErrorAs(t, modelcall.Err(env.GetWorkflowError()), &business)
	assert.Equal(t, "AI is turned off", business.Error())
}

// A test asks whether the connection works now; a retry would answer a
// different question.
func TestTestAIProviderWorkflow_ProbesOnce(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	calls := 0
	env.OnActivity(a.TestAIProviderActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *TestAIProviderPayload) (*agentports.TestAIProviderResult, error) {
			calls++

			return nil, modelcall.Classify(context.DeadlineExceeded)
		})

	env.ExecuteWorkflow(TestAIProviderWorkflow, &TestAIProviderPayload{})

	require.Error(t, env.GetWorkflowError())
	assert.Equal(t, 1, calls)
}

func TestWriteBriefingWorkflow_ReturnsWhatWasWritten(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	env.OnActivity(a.WriteBriefingActivity, mock.Anything, mock.Anything).
		Return(&briefingports.WriteBriefingResult{Written: 1, Narrated: 1}, nil).
		Once()

	env.ExecuteWorkflow(WriteBriefingWorkflow, &WriteBriefingPayload{})

	require.NoError(t, env.GetWorkflowError())
	var result briefingports.WriteBriefingResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Written)
}
