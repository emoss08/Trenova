package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runReplies(
	t *testing.T,
	turns ...*serviceports.ChatCompletionResult,
) (*serviceports.RunResult, *scriptedCompletion, []serviceports.StreamEvent) {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	service := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	var events []serviceports.StreamEvent

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "what is on my dashboard",
		Emit:       func(event serviceports.StreamEvent) { events = append(events, event) },
	})
	require.NoError(t, err)

	return result, completion, events
}

func loopingTurn() *serviceports.ChatCompletionResult {
	return textTurn("Your dashboard has three tiles: on-" + strings.Repeat("time-", 200))
}

func restarts(events []serviceports.StreamEvent) int {
	count := 0
	for _, event := range events {
		retry, ok := event.Data.(serviceports.AssistantRetryingEvent)
		if ok && event.Event == serviceports.AssistantEventRetrying &&
			retry.Kind == serviceports.RetryKindRestart {
			count++
		}
	}

	return count
}

/*
A reply that falls into a loop is thrown away and written again.

The reader is told it is starting over — the same restart a provider failover
sends, so the words that looped are withdrawn from the screen — and the reply
that is kept is the second one.
*/
func TestRun_WritesAgainAReplyThatLooped(t *testing.T) {
	t.Parallel()

	result, completion, events := runReplies(t, loopingTurn(), textTurn("You have three tiles."))

	assert.Equal(t, "You have three tiles.", result.Reply)
	assert.Equal(t, 2, completion.CallCount)
	assert.Equal(t, 1, restarts(events))
}

// Twice running is not bad luck. The loop is never kept; the person gets a
// line they can act on.
func TestRun_EndsOnAPlainLineWhenTheReplyLoopsAgain(t *testing.T) {
	t.Parallel()

	result, completion, _ := runReplies(
		t,
		loopingTurn(),
		loopingTurn(),
		textTurn("never asked for"),
	)

	assert.Equal(t, loopedReply, result.Reply)
	assert.Equal(t, 2, completion.CallCount)
	assert.NotContains(t, result.Messages[len(result.Messages)-1].Content, "time-time")
}

/*
A turn never ends in silence.

"Approved" in the Report Builder thread got no reply at all: the model returned
nothing, and the screen looked hung. An empty reply is asked for once more,
then answered with a plain line.
*/
func TestRun_AsksAgainForAnEmptyReply(t *testing.T) {
	t.Parallel()

	result, completion, _ := runReplies(t, textTurn(""), textTurn("It was created."))

	assert.Equal(t, "It was created.", result.Reply)
	assert.Equal(t, 2, completion.CallCount)
}

func TestRun_EndsOnAPlainLineWhenTheReplyIsEmptyAgain(t *testing.T) {
	t.Parallel()

	result, _, _ := runReplies(t, textTurn(""), textTurn("   "))

	assert.Equal(t, emptyReply, result.Reply)
}

// A turn that asked the person something has said what it needed to; the
// question is on screen, and the model was told to stop there.
func TestRun_AcceptsSilenceAfterAQuestion(t *testing.T) {
	t.Parallel()

	result, completion, _ := runReplies(t,
		toolTurn(askUserName, map[string]any{"question": "Which dashboard?", "allowOther": true}),
		textTurn(""),
	)

	assert.Empty(t, result.Reply)
	assert.Equal(t, 2, completion.CallCount)
}

/*
A question the person has just answered is not asked again.

The Homepage Widget Builder asked which dashboard, got an answer, and asked
the same thing on its next turn. The second ask is refused, and the model is
told to use the answer it has.
*/
func TestRun_RefusesAQuestionThePersonJustAnswered(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn(askUserName, map[string]any{"question": "Which dashboard do you mean?"}),
		textTurn("Using your home page."),
	}}
	service := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "my home page",
		History: []conversation.Message{
			{Role: conversation.RoleUser, Content: "what is on my dashboard"},
			{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
				{ID: "c1", Name: askUserName, Arguments: map[string]any{
					"question": "which dashboard do you mean",
				}},
			}},
			{Role: conversation.RoleTool, Content: "shown"},
		},
	})
	require.NoError(t, err)

	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "already asked")
	assert.Equal(t, "Using your home page.", result.Reply)
}

func TestRun_AsksANewQuestion(t *testing.T) {
	t.Parallel()

	result, _, _ := runReplies(t,
		toolTurn(askUserName, map[string]any{"question": "Which dashboard?", "allowOther": true}),
		textTurn(""),
	)

	assert.False(t, result.Messages[2].ToolFailed)
}

// A model loops in its thinking as readily as in its reply. The loop is cut
// off before it reaches the reader, and the turn is asked again rather than
// ending on thousands of repeated fragments.
func TestRun_AsksAgainWhenTheThinkingLoops(t *testing.T) {
	t.Parallel()

	looping := textTurn("You have three tiles.")
	looping.Reasoning = &conversation.ReasoningTrace{Text: "The user wants time-" + strings.Repeat("time-", 400)}

	result, completion, events := runReplies(t, looping, textTurn("You have three tiles."))

	assert.Equal(t, "You have three tiles.", result.Reply)
	assert.Equal(t, 2, completion.CallCount)
	assert.Equal(t, 1, restarts(events))
	for _, event := range events {
		thought, ok := event.Data.(serviceports.AssistantReasoningEvent)
		if ok {
			assert.NotContains(t, thought.Text, "time-time-time", "the loop never reaches the reader")
		}
	}
}
