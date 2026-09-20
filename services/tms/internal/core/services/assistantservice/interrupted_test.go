package assistantservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lookupTurn(name string) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{
		ToolCalls: []serviceports.ToolCall{
			{ID: "call_1", Name: name, Arguments: map[string]any{"id": "S1"}},
		},
		ModelIdentifier: "test-model",
	}
}

// A model that fails after a tool has run used to take the whole turn with
// it: SendMessage returned before anything was saved, so the question, the
// lookup, and any write a tool had already made all vanished from the thread.
// The person saw an error banner over a conversation that claimed nothing had
// happened. What ran is kept, closed with a note saying it was interrupted.
func TestSendMessageStream_KeepsWhatRanWhenTheModelFailsPartway(t *testing.T) {
	t.Parallel()

	tool := &agentruntimetest.StubQueryTool{
		ToolName: "get_shipment",
		Result:   map[string]any{"proNumber": "S1"},
	}
	completion := &scriptedCompletion{
		Turns:  []*serviceports.ChatCompletionResult{lookupTurn("get_shipment")},
		Errors: map[int]error{1: errors.New("every configured chat provider failed")},
	}
	svc := newService(completion, &stubQueryRegistry{
		Tools: []serviceports.AgentQueryTool{tool},
	}, &stubActionRegistry{})
	conversations := &stubConversations{thread: &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Title:             "Dispatch",
	}}
	svc.conversations = conversations
	svc.definitions = &stubDefinitions{definition: testDefinition("get_shipment")}
	actor := testActor()

	_, err := svc.SendMessageStream(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Where is S1?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)

	require.Error(t, err, "the failure is still reported")
	require.Len(t, conversations.appended, 4, "user, the tool call, its result, and the note")
	assert.Equal(t, conversation.RoleUser, conversations.appended[0].Role)
	assert.Equal(t, "Where is S1?", conversations.appended[0].Content)
	assert.Equal(t, conversation.RoleTool, conversations.appended[2].Role)

	closing := conversations.appended[3]
	assert.Equal(t, conversation.RoleAssistant, closing.Role)
	assert.Contains(t, closing.Content, "interrupted before it finished")
	assert.False(t, closing.Refused, "an interruption is not a refusal")
}

// Stopping a reply, or closing the tab, cancels the request context. The save
// must not ride that context, or the one act of keeping the turn is the act
// the cancellation defeats.
func TestSendMessageStream_SavesTheTurnOnACancelledRequest(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("It is in Los Angeles."),
	}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _ = svc.SendMessageStream(ctx, &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Where is it?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)

	require.Equal(t, 1, conversations.appendCalls, "the turn is saved")
	assert.NoError(t, conversations.appendCtxErr, "on a context the cancellation cannot reach")
}
