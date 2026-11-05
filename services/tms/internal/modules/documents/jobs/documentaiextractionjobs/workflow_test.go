package documentaiextractionjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func extractionPayload() *documentanalysis.ProcessDocumentAIExtractionPayload {
	return &documentanalysis.ProcessDocumentAIExtractionPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: 1_700_000_000,
	}
}

func extractionEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	return env
}

// The workflow submits once, waits on its own timer between polls, and applies
// what the model answered.
func TestProcessDocumentAIExtractionWorkflow_PollsOnItsOwnTimerUntilAnswered(t *testing.T) {
	t.Parallel()

	env := extractionEnv(t)
	var a *Activities
	env.OnActivity(a.SubmitDocumentAIExtractionActivity, mock.Anything, mock.Anything).
		Return(&documentanalysis.AIExtractionProgress{}, nil).
		Once()

	polls := 0
	answer := &documentanalysis.AsyncAIExtractionCompletion{
		ResponseID: "resp_1",
		Status:     documentaiextractionports.AIBackgroundExtractionStatusCompleted,
	}
	env.OnActivity(a.PollDocumentAIExtractionActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *documentanalysis.PollDocumentAIExtractionInput) (*documentanalysis.AIExtractionProgress, error) {
			polls++
			assert.False(t, input.GiveUp, "the answer came long before the deadline")
			if polls < 3 {
				return &documentanalysis.AIExtractionProgress{}, nil
			}

			return &documentanalysis.AIExtractionProgress{Completion: answer}, nil
		})

	var applied *documentanalysis.AsyncAIExtractionCompletion
	env.OnActivity(a.ApplyDocumentAIExtractionResultActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, payload *ApplyDocumentAIExtractionPayload) (*documentanalysis.ProcessDocumentAIExtractionResult, error) {
			applied = payload.Completion

			return &documentanalysis.ProcessDocumentAIExtractionResult{
				DocumentID: payload.DocumentID,
			}, nil
		}).
		Once()

	env.ExecuteWorkflow(ProcessDocumentAIExtractionWorkflow, extractionPayload())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, 3, polls)
	require.NotNil(t, applied)
	assert.Equal(t, "resp_1", applied.ResponseID)
	env.AssertExpectations(t)
}

// An answer that came back with the submit is applied without a single poll.
func TestProcessDocumentAIExtractionWorkflow_AppliesAnInlineAnswerAtOnce(t *testing.T) {
	t.Parallel()

	env := extractionEnv(t)
	var a *Activities
	env.OnActivity(a.SubmitDocumentAIExtractionActivity, mock.Anything, mock.Anything).
		Return(&documentanalysis.AIExtractionProgress{Completion: &documentanalysis.AsyncAIExtractionCompletion{
			RawStatus: "inline",
			Status:    documentaiextractionports.AIBackgroundExtractionStatusCompleted,
		}}, nil).
		Once()
	env.OnActivity(a.ApplyDocumentAIExtractionResultActivity, mock.Anything, mock.Anything).
		Return(&documentanalysis.ProcessDocumentAIExtractionResult{}, nil).
		Once()

	env.ExecuteWorkflow(ProcessDocumentAIExtractionWorkflow, extractionPayload())

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

// A model that never answers is waited on for the longest wait, and then the
// last poll gives up, so the document is not left pending forever.
func TestProcessDocumentAIExtractionWorkflow_GivesUpAtTheLongestWait(t *testing.T) {
	t.Parallel()

	env := extractionEnv(t)
	var a *Activities
	env.OnActivity(a.SubmitDocumentAIExtractionActivity, mock.Anything, mock.Anything).
		Return(&documentanalysis.AIExtractionProgress{}, nil).
		Once()

	gaveUp := false
	env.OnActivity(a.PollDocumentAIExtractionActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, input *documentanalysis.PollDocumentAIExtractionInput) (*documentanalysis.AIExtractionProgress, error) {
			if !input.GiveUp {
				return &documentanalysis.AIExtractionProgress{}, nil
			}
			gaveUp = true

			return &documentanalysis.AIExtractionProgress{
				Completion: &documentanalysis.AsyncAIExtractionCompletion{
					Status:      documentaiextractionports.AIBackgroundExtractionStatusFailed,
					FailureCode: "ai_extract_timeout",
				},
			}, nil
		})
	env.OnActivity(a.ApplyDocumentAIExtractionResultActivity, mock.Anything, mock.Anything).
		Return(&documentanalysis.ProcessDocumentAIExtractionResult{}, nil).
		Once()

	env.ExecuteWorkflow(ProcessDocumentAIExtractionWorkflow, extractionPayload())

	require.NoError(t, env.GetWorkflowError())
	assert.True(t, gaveUp)
	env.AssertExpectations(t)
}
