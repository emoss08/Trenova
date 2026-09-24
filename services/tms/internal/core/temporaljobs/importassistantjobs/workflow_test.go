package importassistantjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	assistant "github.com/emoss08/trenova/internal/core/services/shipmentimportassistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func turnPayload(stream bool) *TurnPayload {
	return &TurnPayload{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		Request: &serviceports.ShipmentImportChatRequest{
			UserMessage: "Set the customer to Acme",
			DocumentID:  pulid.MustNew("doc_").String(),
		},
		Stream: stream,
	}
}

func turnEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.PrepareImportTurnActivity, mock.Anything, mock.Anything).
		Return(&assistant.PreparedTurn{
			ConversationID: pulid.MustNew("sic_"),
			System:         "You are the import assistant.",
			Messages: []serviceports.Message{{
				Role:    serviceports.RoleUser,
				Content: "Set the customer to Acme",
			}},
		}, nil).
		Once()

	return env
}

// A model that calls a tool, reads its result and answers: the tool runs as
// its own activity, the model is asked again with the result, and what the
// turn did is saved and returned.
func TestImportAssistantTurnWorkflow_RunsTheToolLoopAndSaves(t *testing.T) {
	t.Parallel()

	env := turnEnv(t)
	var a *Activities

	calls := 0
	env.OnActivity(a.ImportModelCallActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *ModelInput) (*serviceports.ChatCompletionResult, error) {
			calls++
			if calls == 1 {
				return &serviceports.ChatCompletionResult{
					ToolCalls: []serviceports.ToolCall{{
						ID:        "call_1",
						Name:      "search_customers",
						Arguments: map[string]any{"query": "Acme"},
					}},
				}, nil
			}

			if assert.Len(t, in.Request.Messages, 3, "the second call carries the tool's result") {
				assert.Equal(t, "call_1", in.Request.Messages[2].ToolCallID)
			}

			return &serviceports.ChatCompletionResult{
				Text:            "Found Acme Freight.",
				ModelIdentifier: "test-model",
			}, nil
		})

	env.OnActivity(a.ImportToolActivity, mock.Anything, mock.Anything).
		Return(&assistant.ToolOutcome{
			Output:  `{"customers":[{"id":"cus_1"}]}`,
			Status:  "completed",
			Actions: []serviceports.ShipmentImportAction{{Type: "set_field", FieldKey: "customer"}},
		}, nil).
		Once()

	var saved *FinishInput
	env.OnActivity(a.FinishImportTurnActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *FinishInput) (*serviceports.ShipmentImportChatResponse, error) {
			saved = in

			return &serviceports.ShipmentImportChatResponse{
				Message:        in.Record.Message,
				ConversationID: "sic_1",
				Actions:        in.Record.Actions,
			}, nil
		}).
		Once()

	env.ExecuteWorkflow(ImportAssistantTurnWorkflow, turnPayload(false))

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var response serviceports.ShipmentImportChatResponse
	require.NoError(t, env.GetWorkflowResult(&response))
	assert.Equal(t, "Found Acme Freight.", response.Message)

	require.NotNil(t, saved)
	assert.Equal(t, "test-model", saved.Record.Model)
	require.Len(t, saved.Record.ToolCalls, 1)
	assert.Equal(t, "search_customers", saved.Record.ToolCalls[0].Name)
	require.Len(t, saved.Record.Actions, 1)
	assert.Equal(t, 2, calls)
	env.AssertExpectations(t)
}

func TestImportAssistantTurnWorkflow_AttributesEveryModelCallAndTotalsItsUsage(t *testing.T) {
	t.Parallel()

	env := turnEnv(t)
	var a *Activities
	payload := turnPayload(false)
	providerID := pulid.MustNew("aip_")

	attributed := make([]serviceports.AIUsageAttribution, 0, 2)
	env.OnActivity(a.ImportModelCallActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *ModelInput) (*serviceports.ChatCompletionResult, error) {
			attributed = append(attributed, in.Request.Attribution)
			if len(attributed) == 1 {
				return &serviceports.ChatCompletionResult{
					ToolCalls: []serviceports.ToolCall{{
						ID:        "call_1",
						Name:      "search_customers",
						Arguments: map[string]any{"query": "Acme"},
					}},
					ModelIdentifier: "first-model",
					InputTokens:     400,
					OutputTokens:    20,
					ReasoningTokens: 5,
				}, nil
			}

			return &serviceports.ChatCompletionResult{
				Text:            "Found Acme Freight.",
				ModelIdentifier: "second-model",
				ProviderID:      providerID,
				ProviderKind:    aiprovider.KindOllama,
				InputTokens:     500,
				OutputTokens:    30,
				ReasoningTokens: 7,
			}, nil
		})

	env.OnActivity(a.ImportToolActivity, mock.Anything, mock.Anything).
		Return(&assistant.ToolOutcome{Output: `{"customers":[]}`, Status: "completed"}, nil).
		Once()

	var saved *FinishInput
	env.OnActivity(a.FinishImportTurnActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *FinishInput) (*serviceports.ShipmentImportChatResponse, error) {
			saved = in

			return &serviceports.ShipmentImportChatResponse{}, nil
		}).
		Once()

	env.ExecuteWorkflow(ImportAssistantTurnWorkflow, payload)

	require.NoError(t, env.GetWorkflowError())
	require.Len(t, attributed, 2)
	for _, attribution := range attributed {
		assert.Equal(t, payload.TenantInfo.UserID, attribution.UserID)
		assert.Equal(t, aiusage.FeatureShipmentImportChat, attribution.Feature)
		assert.Equal(t, aiusage.Subject{
			Type: aiusage.SubjectTypeDocument,
			ID:   payload.Request.DocumentID,
		}, attribution.Subject)
	}
	require.NotNil(t, saved)
	assert.Equal(t, "second-model", saved.Record.Model)
	assert.Equal(t, providerID, saved.Record.ProviderID)
	assert.Equal(t, aiprovider.KindOllama, saved.Record.ProviderKind)
	assert.Equal(t, 900, saved.Record.InputTokens)
	assert.Equal(t, 50, saved.Record.OutputTokens)
	assert.Equal(t, 12, saved.Record.ReasoningTokens)
}

// suggest_quick_actions carries the reply's chips. The loop reads them itself,
// so it never becomes a tool activity or a recorded call.
func TestImportAssistantTurnWorkflow_TakesSuggestionsWithoutATool(t *testing.T) {
	t.Parallel()

	env := turnEnv(t)
	var a *Activities

	calls := 0
	env.OnActivity(a.ImportModelCallActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *ModelInput) (*serviceports.ChatCompletionResult, error) {
			calls++
			if calls == 1 {
				return &serviceports.ChatCompletionResult{
					Text: "Is Acme right?",
					ToolCalls: []serviceports.ToolCall{{
						ID:   "call_9",
						Name: assistant.SuggestQuickActionsTool,
						Arguments: map[string]any{"suggestions": []any{
							map[string]any{"label": "Yes", "prompt": "Yes, Acme"},
						}},
					}},
				}, nil
			}

			return &serviceports.ChatCompletionResult{}, nil
		})

	var saved *FinishInput
	env.OnActivity(a.FinishImportTurnActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *FinishInput) (*serviceports.ShipmentImportChatResponse, error) {
			saved = in

			return &serviceports.ShipmentImportChatResponse{}, nil
		})

	env.ExecuteWorkflow(ImportAssistantTurnWorkflow, turnPayload(false))

	require.NoError(t, env.GetWorkflowError())
	require.NotNil(t, saved)
	require.Len(t, saved.Record.Suggestions, 1)
	assert.Equal(t, "Yes", saved.Record.Suggestions[0].Label)
	assert.Empty(t, saved.Record.ToolCalls)
}

// Creating a shipment twice is worse than failing once: a write that fails is
// not asked again, and the model is told it did not happen.
func TestImportAssistantTurnWorkflow_NeverRetriesAWrite(t *testing.T) {
	t.Parallel()

	env := turnEnv(t)
	var a *Activities

	calls := 0
	env.OnActivity(a.ImportModelCallActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *ModelInput) (*serviceports.ChatCompletionResult, error) {
			calls++
			if calls == 1 {
				return &serviceports.ChatCompletionResult{ToolCalls: []serviceports.ToolCall{{
					ID:   "call_1",
					Name: "create_shipment",
				}}}, nil
			}

			return &serviceports.ChatCompletionResult{Text: "It did not go through."}, nil
		})

	writes := 0
	env.OnActivity(a.ImportToolActivity, mock.Anything, mock.Anything).
		Return(func(context.Context, *ToolInput) (*assistant.ToolOutcome, error) {
			writes++

			return nil, errors.New("worker lost mid-call")
		})

	var saved *FinishInput
	env.OnActivity(a.FinishImportTurnActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *FinishInput) (*serviceports.ShipmentImportChatResponse, error) {
			saved = in

			return &serviceports.ShipmentImportChatResponse{}, nil
		})

	env.ExecuteWorkflow(ImportAssistantTurnWorkflow, turnPayload(false))

	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, 1, writes)
	require.NotNil(t, saved)
	require.Len(t, saved.Record.ToolCalls, 1)
	assert.Equal(t, "error", saved.Record.ToolCalls[0].Status)
}

// A model that fails for good ends the turn. What it did is recorded as a
// failed turn, and the person is told in words written for them.
func TestImportAssistantTurnWorkflow_RecordsAFailedTurn(t *testing.T) {
	t.Parallel()

	env := turnEnv(t)
	var a *Activities

	env.OnActivity(a.ImportModelCallActivity, mock.Anything, mock.Anything).
		Return(nil, modelcall.Classify(serviceports.ErrNoProviderConfigured)).
		Once()

	var failed *FinishInput
	env.OnActivity(a.FailImportTurnActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *FinishInput) (string, error) {
			failed = in

			return "No AI provider is configured for this organization.", nil
		}).
		Once()

	env.ExecuteWorkflow(ImportAssistantTurnWorkflow, turnPayload(true))

	err := env.GetWorkflowError()
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, errTypeTurnFailed, appErr.Type())
	assert.Contains(t, appErr.Message(), "No AI provider")

	require.NotNil(t, failed)
	require.NotNil(t, failed.Failure)
	assert.ErrorIs(t, failed.Failure.Err(), serviceports.ErrNoProviderConfigured)
	env.AssertExpectations(t)
}
