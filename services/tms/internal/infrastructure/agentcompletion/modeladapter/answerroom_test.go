package modeladapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func finishedChatStream() string {
	return sse(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"Done."},"finish_reason":"stop"}]}`,
		},
		[2]string{"", "[DONE]"},
	)
}

func TestOpenAIChatAdapter_GivesAReasoningCallRoomToAnswer(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", finishedChatStream())

	resp, _ := streamWith(t, NewOpenAIChatAdapter(), reasoningCall(
		aiprovider.KindOpenAIChat, server.URL, aiprovider.ReasoningLow,
		&Request{Messages: UserMessage("transfer them")},
	))

	assert.InDelta(t, reasoningAnswerFloor, (*captured)["max_tokens"], 0)
	assert.Equal(t, reasoningAnswerFloor, resp.OutputLimit)
}

func TestOpenAIChatAdapter_GivesAModelThatHasShownItThinksRoomToAnswer(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", finishedChatStream())

	call := callFor(aiprovider.KindOpenAIChat, server.URL, &Request{Messages: []Message{
		{Role: RoleUser, Content: "which can go?"},
		{
			Role:      RoleAssistant,
			Content:   "Twenty-one can go.",
			Reasoning: &ReasoningTrace{Text: "Listing them.", ProviderKind: "OpenAIChat"},
		},
		{Role: RoleUser, Content: "transfer them"},
	}})
	call.Provider.MaxTokens = 1024
	call.Request.MaxTokens = 1024

	resp, _ := streamWith(t, NewOpenAIChatAdapter(), call)

	assert.InDelta(t, reasoningAnswerFloor, (*captured)["max_tokens"], 0)
	assert.Equal(t, reasoningAnswerFloor, resp.OutputLimit)
}

func TestOpenAIChatAdapter_KeepsTheConfiguredCeilingForACallThatDoesNotReason(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", finishedChatStream())

	call := callFor(aiprovider.KindOpenAIChat, server.URL, &Request{Messages: []Message{
		{Role: RoleUser, Content: "hi"},
		{
			Role:      RoleAssistant,
			Content:   "Hello.",
			Reasoning: &ReasoningTrace{Text: "Signed.", ProviderKind: "AnthropicMessages"},
		},
		{Role: RoleUser, Content: "again"},
	}})

	resp, _ := streamWith(t, NewOpenAIChatAdapter(), call)

	assert.InDelta(t, 512, (*captured)["max_tokens"], 0)
	assert.Equal(t, 512, resp.OutputLimit)
}

func TestOpenAIChatAdapter_KeepsAHigherConfiguredCeiling(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", finishedChatStream())

	call := reasoningCall(
		aiprovider.KindOpenAIChat, server.URL, aiprovider.ReasoningHigh,
		&Request{Messages: UserMessage("go")},
	)
	call.Request.MaxTokens = 40000

	_, _ = streamWith(t, NewOpenAIChatAdapter(), call)

	assert.InDelta(t, 40000, (*captured)["max_tokens"], 0)
}

func TestOpenAIChatAdapter_CompleteReportsACutWithNothingButThinking(t *testing.T) {
	t.Parallel()

	server, _ := captureServer(t, map[string]any{
		"model": "glm",
		"choices": []any{map[string]any{
			"finish_reason": "length",
			"message": map[string]any{
				"role":              "assistant",
				"content":           "",
				"reasoning_content": "Let me list the 21 ids again: shp_1, shp_2,",
			},
		}},
		"usage": map[string]any{"completion_tokens": 512},
	})

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), callFor(
		aiprovider.KindOpenAIChat, server.URL, &Request{Messages: UserMessage("transfer")},
	))

	require.NoError(t, err)
	assert.True(t, resp.Truncated)
	assert.Empty(t, resp.Text)
	require.NotNil(t, resp.Reasoning)
}

func TestOpenAIResponsesAdapter_GivesAReasoningCallRoomToAnswer(t *testing.T) {
	t.Parallel()

	thinking := reasoningCall(
		aiprovider.KindOpenAIResponses, "http://unused", aiprovider.ReasoningMedium,
		&Request{Messages: UserMessage("go")},
	)
	body := openAIResponsesAdapter{}.requestFor(thinking)
	assert.Equal(t, reasoningAnswerFloor, body.MaxOutputTokens)

	plain := callFor(aiprovider.KindOpenAIResponses, "http://unused",
		&Request{Messages: UserMessage("go")})
	assert.Equal(t, 512, openAIResponsesAdapter{}.requestFor(plain).MaxOutputTokens)
}

func TestAnthropicAdapter_ReportsTheCeilingItSent(t *testing.T) {
	t.Parallel()

	server, captured := captureServer(t, map[string]any{
		"model":       "claude-x",
		"stop_reason": "end_turn",
		"content":     []any{map[string]any{"type": "text", "text": "Done."}},
	})

	resp, err := NewAnthropicAdapter().Complete(t.Context(), reasoningCall(
		aiprovider.KindAnthropicMessages, server.URL, aiprovider.ReasoningMedium,
		&Request{Messages: UserMessage("go")},
	))

	require.NoError(t, err)
	assert.InDelta(t, 4096+thinkingAnswerRoom, (*captured)["max_tokens"], 0)
	assert.Equal(t, 4096+thinkingAnswerRoom, resp.OutputLimit)
}
