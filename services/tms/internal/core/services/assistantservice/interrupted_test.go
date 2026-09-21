package assistantservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
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

// A turn stopped before the model said or did anything used to close with
// "What is shown above is what had happened by then" under a question with
// nothing above it but the question. The note now says what happened.
func TestInterruptedTurn_SaysNothingRanWhenNothingDid(t *testing.T) {
	t.Parallel()

	req := &TurnRequest{Input: "Where is S1?"}
	nothingRan := &serviceports.RunResult{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "Where is S1?"}},
	}

	stopped := interruptedTurn(req, agentguard.Decision{Allowed: true}, nothingRan, context.Canceled)
	require.NotNil(t, stopped, "the question is kept")
	require.Len(t, stopped.Messages, 2)
	assert.NotContains(t, stopped.Reply, "shown above")
	assert.Contains(t, stopped.Reply, "before a reply started")

	failed := interruptedTurn(req, agentguard.Decision{Allowed: true}, nothingRan, errors.New("provider down"))
	require.NotNil(t, failed)
	assert.NotContains(t, failed.Reply, "shown above")
	assert.Contains(t, failed.Reply, "before it started")
	assert.Contains(t, failed.Reply, "Ask again")

	ran := &serviceports.RunResult{Messages: []conversation.Message{
		{Role: conversation.RoleUser, Content: "Where is S1?"},
		{Role: conversation.RoleAssistant, Content: "Looking it up."},
	}}
	partial := interruptedTurn(req, agentguard.Decision{Allowed: true}, ran, context.Canceled)
	require.NotNil(t, partial)
	assert.Contains(t, partial.Reply, "shown above", "with something above, the note points at it")
}

// A run that failed before it produced anything, even the echo of the
// question, still leaves a turn: the person asked something and the thread
// shows the question with the note that nothing ran. Returning nothing
// dropped the question from the thread as if it had never been sent.
func TestInterruptedTurn_KeepsTheQuestionWhenNoRunCameBack(t *testing.T) {
	t.Parallel()

	req := &TurnRequest{Input: "Where is S1?"}

	turn := interruptedTurn(req, agentguard.Decision{Allowed: true}, nil, errors.New("provider down"))
	require.NotNil(t, turn)
	require.Len(t, turn.Messages, 2)
	assert.Equal(t, conversation.RoleUser, turn.Messages[0].Role)
	assert.Equal(t, "Where is S1?", turn.Messages[0].Content)
	assert.Equal(t, conversation.RoleAssistant, turn.Messages[1].Role)
	assert.Contains(t, turn.Reply, "before it started")

	empty := interruptedTurn(req, agentguard.Decision{Allowed: true}, &serviceports.RunResult{}, context.Canceled)
	require.NotNil(t, empty)
	require.Len(t, empty.Messages, 2)
	assert.Contains(t, empty.Reply, "before a reply started")
}

type fakeProviderFailure struct {
	status    int
	retryable bool
}

func (f fakeProviderFailure) Error() string           { return "provider failed" }
func (f fakeProviderFailure) ProviderStatus() int     { return f.status }
func (f fakeProviderFailure) ProviderRetryable() bool { return f.retryable }

// "Ask again" is the wrong advice for a request the provider refused: asking
// again sends the same request. The note says which kind of failure it was,
// so a refusal sends the person to an administrator and an outage tells
// them to wait, and neither has to guess from a note that said only that
// the reply failed.
func TestClosingNotice_SaysWhetherTheProviderRefusedOrWasUnavailable(t *testing.T) {
	t.Parallel()

	refused := closingNotice(
		fmt.Errorf("chat: %w", fakeProviderFailure{status: 400, retryable: false}), true,
	)
	assert.Contains(t, refused, "rejected the request (status 400)")
	assert.Contains(t, refused, "AI Control")
	assert.True(t, strings.HasSuffix(refused, "_"), "the note stays one italic span: %q", refused)

	unavailable := closingNotice(
		fmt.Errorf("chat: %w", fakeProviderFailure{status: 503, retryable: true}), false,
	)
	assert.Contains(t, unavailable, "unavailable (status 503)")
	assert.Contains(t, unavailable, "before it started")

	resting := closingNotice(fmt.Errorf("chat: %w", serviceports.ErrProvidersResting), false)
	assert.Contains(t, resting, "paused after repeated failures")

	plain := closingNotice(errors.New("something else"), false)
	assert.Equal(t, failedBeforeStartNotice, plain, "an unclassified failure adds nothing")
}
