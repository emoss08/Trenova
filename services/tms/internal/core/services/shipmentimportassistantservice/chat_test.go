package shipmentimportassistantservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func replayService(turns []*shipmentimportchat.Turn) (*Service, pulid.ID) {
	conversationID := pulid.MustNew("sic_")

	return &Service{
		logger: zap.NewNop(),
		chatRepo: &chatRepoStub{
			getConversationByDocumentFn: func(
				_ context.Context,
				_ repoports.GetShipmentImportConversationRequest,
			) (*shipmentimportchat.Conversation, error) {
				return &shipmentimportchat.Conversation{
					ID:         conversationID,
					DocumentID: pulid.MustNew("doc_"),
					Status:     shipmentimportchat.ConversationStatusActive,
				}, nil
			},
			listTurnsFn: func(
				_ context.Context,
				_ repoports.ListShipmentImportTurnsRequest,
			) ([]*shipmentimportchat.Turn, error) {
				return turns, nil
			},
			updateActiveConversationStatusFn: func(
				context.Context, pulid.ID, pagination.TenantInfo,
				shipmentimportchat.ConversationStatus,
				shipmentimportchat.ConversationStatusReason,
			) error {
				return errors.New("unexpected status update")
			},
		},
	}, conversationID
}

func chatRequest() *serviceports.ShipmentImportChatRequest {
	return &serviceports.ShipmentImportChatRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		UserMessage: "Set the customer to Acme",
	}
}

/*
The exchange used to live on OpenAI's side, carried between turns by
PreviousResponseID. That made the conversation unmovable: a handle issued by one
endpoint means nothing to another, so an organization that switched provider
mid-import would have lost the thread entirely. Every turn is already persisted
for the history panel, so the assistant now replays its own record.
*/
func TestBuildMessages_ReplaysThePersistedExchange(t *testing.T) {
	t.Parallel()

	service, conversationID := replayService([]*shipmentimportchat.Turn{{
		ID:               pulid.MustNew("sit_"),
		UserMessage:      "What is the rate?",
		AssistantMessage: "The rate is $1,850.",
		CreatedAt:        1_717_171_750,
	}})

	messages := service.buildMessages(
		t.Context(),
		chatRequest(),
		&shipmentimportchat.Conversation{ID: conversationID, DocumentID: pulid.MustNew("doc_")},
	)

	require.Len(t, messages, 3)
	assert.Equal(t, serviceports.RoleUser, messages[0].Role)
	assert.Equal(t, "What is the rate?", messages[0].Content)
	assert.Equal(t, serviceports.RoleAssistant, messages[1].Role)
	assert.Equal(t, "The rate is $1,850.", messages[1].Content)
	assert.Equal(t, serviceports.RoleUser, messages[2].Role)
	assert.Equal(t, "Set the customer to Acme", messages[2].Content,
		"the new message is last")
}

// Replaying the whole exchange is the cost of holding it ourselves. A long
// import would otherwise spend its entire budget on its own history.
func TestBuildMessages_BoundsHowMuchItReplays(t *testing.T) {
	t.Parallel()

	turns := make([]*shipmentimportchat.Turn, 0, 40)
	for i := range 40 {
		turns = append(turns, &shipmentimportchat.Turn{
			ID:               pulid.MustNew("sit_"),
			UserMessage:      fmt.Sprintf("question %d", i),
			AssistantMessage: fmt.Sprintf("answer %d", i),
		})
	}

	service, conversationID := replayService(turns)

	messages := service.buildMessages(
		t.Context(),
		chatRequest(),
		&shipmentimportchat.Conversation{ID: conversationID, DocumentID: pulid.MustNew("doc_")},
	)

	assert.LessOrEqual(t, len(messages), maxReplayedTurns*2+1)
	assert.Equal(t, "answer 39", messages[len(messages)-2].Content,
		"the most recent turns are the ones kept")
}

// A turn with no text on one side is a failed or in-flight turn. Sending an
// empty message is not neutral: some protocols reject it outright.
func TestBuildMessages_SkipsEmptyTurns(t *testing.T) {
	t.Parallel()

	service, conversationID := replayService([]*shipmentimportchat.Turn{{
		ID:               pulid.MustNew("sit_"),
		UserMessage:      "Create it",
		AssistantMessage: "",
	}})

	messages := service.buildMessages(
		t.Context(),
		chatRequest(),
		&shipmentimportchat.Conversation{ID: conversationID, DocumentID: pulid.MustNew("doc_")},
	)

	for _, message := range messages {
		assert.NotEmpty(t, message.Content)
	}
}

/*
A round has to come back as the assistant turn that asked plus one result per
call. Dropping the assistant turn leaves tool results with nothing to attach to,
which every protocol rejects differently and none of them usefully.
*/
func TestToolRound_PairsEveryCallWithItsResult(t *testing.T) {
	t.Parallel()

	calls := []serviceports.ToolCall{
		{ID: "call_1", Name: "search_customers"},
		{ID: "call_2", Name: "set_field_value"},
	}
	results := map[string]toolResult{
		"call_1": {output: `{"customers":[]}`, status: toolStatusCompleted},
		"call_2": {output: `{"error":"no such field"}`, status: toolStatusError},
	}

	messages := toolRound(calls, results)

	require.Len(t, messages, 3)
	assert.Equal(t, serviceports.RoleAssistant, messages[0].Role)
	assert.Len(t, messages[0].ToolCalls, 2)

	assert.Equal(t, serviceports.RoleTool, messages[1].Role)
	assert.Equal(t, "call_1", messages[1].ToolCallID)
	assert.False(t, messages[1].IsError)

	assert.Equal(t, "call_2", messages[2].ToolCallID)
	assert.True(t, messages[2].IsError, "a failed tool is marked so the model can recover")
}

/*
suggest_quick_actions is not a lookup: it carries the follow-up chips shown
under the reply. It has no handler, so routing it like any other tool would
answer the model "unknown tool" and lose the chips.
*/
func TestRunToolCalls_AnswersSuggestQuickActionsItself(t *testing.T) {
	t.Parallel()

	service := &Service{logger: zap.NewNop()}
	state := &turnState{}

	results := service.runToolCalls(
		t.Context(),
		pagination.TenantInfo{},
		[]serviceports.ToolCall{{
			ID:   "call_9",
			Name: suggestQuickActionsTool,
			Arguments: map[string]any{
				"suggestions": []any{
					map[string]any{
						"label":       "Confirm",
						"prompt":      "Confirm the customer",
						"submitLabel": "Confirm",
						"type":        "Confirm",
					},
				},
			},
		}},
		state,
		nil,
	)

	require.Len(t, state.suggestions, 1)
	assert.Equal(t, "Confirm", state.suggestions[0].Label)
	assert.Equal(t, toolStatusCompleted, results["call_9"].status)
	assert.Empty(t, state.toolCalls, "it is not a tool call the reader needs to see")
}

func TestRunToolCalls_RecordsAnUnknownTool(t *testing.T) {
	t.Parallel()

	service := &Service{logger: zap.NewNop()}
	state := &turnState{}

	results := service.runToolCalls(
		t.Context(),
		pagination.TenantInfo{},
		[]serviceports.ToolCall{{ID: "call_1", Name: "not_a_tool"}},
		state,
		nil,
	)

	assert.Equal(t, toolStatusError, results["call_1"].status)
	require.Len(t, state.toolCalls, 1)
	assert.Equal(t, "error", state.toolCalls[0].Status)
}

// The streamed path emits progress events; the non-streaming one passes nil and
// runs the same body, so the two cannot drift.
func TestRunToolCalls_EmitsProgressOnlyWhenStreaming(t *testing.T) {
	t.Parallel()

	service := &Service{logger: zap.NewNop()}
	var events []string

	service.runToolCalls(
		t.Context(),
		pagination.TenantInfo{},
		[]serviceports.ToolCall{{ID: "call_1", Name: "not_a_tool"}},
		&turnState{},
		func(event serviceports.StreamEvent) { events = append(events, event.Event) },
	)

	assert.Equal(t, []string{"tool_call_start", "tool_call_done"}, events)
}

func TestEncodeArguments_AlwaysProducesAnObject(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "{}", encodeArguments(nil))
	assert.JSONEq(t, `{"query":"Acme"}`, encodeArguments(map[string]any{"query": "Acme"}))
}

/*
The old mapping read HTTP status codes off one vendor's error type. The router
has already tried every provider that serves this task by the time it returns,
so what a person needs to know is whether anything is configured at all.
*/
func TestFriendlyCompletionError_TellsThePersonWhatToDo(t *testing.T) {
	t.Parallel()

	assert.Contains(t,
		friendlyCompletionError(serviceports.ErrNoProviderConfigured),
		"AI Control",
		"an unconfigured install is told where to configure one")

	assert.Contains(t,
		friendlyCompletionError(context.DeadlineExceeded),
		"timed out")

	assert.NotContains(t,
		friendlyCompletionError(errors.New("dial tcp 127.0.0.1:11434: connection refused")),
		"127.0.0.1",
		"an endpoint is not shown to the operator")
}
