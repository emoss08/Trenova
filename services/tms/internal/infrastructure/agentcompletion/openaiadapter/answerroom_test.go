package openaiadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func finishedChatStream() string {
	return modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"Done."},"finish_reason":"stop"}]}`,
		},
		[2]string{"", "[DONE]"},
	)
}

func TestOpenAIChatAdapter_GivesAReasoningCallRoomToAnswer(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", finishedChatStream())

	resp, _ := modeladaptertest.StreamWith(
		t,
		NewOpenAIChatAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindOpenAIChat, server.URL, aireasoning.ReasoningLow,
			&modeladapter.Request{Messages: modeladapter.UserMessage("transfer them")},
		),
	)

	assert.InDelta(t, modeladapter.ReasoningAnswerFloor, (*captured)["max_tokens"], 0)
	assert.Equal(t, modeladapter.ReasoningAnswerFloor, resp.OutputLimit)
}

func TestOpenAIChatAdapter_GivesAModelThatHasShownItThinksRoomToAnswer(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", finishedChatStream())

	call := modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: []modeladapter.Message{
			{Role: modeladapter.RoleUser, Content: "which can go?"},
			{
				Role:    modeladapter.RoleAssistant,
				Content: "Twenty-one can go.",
				Reasoning: &modeladapter.ReasoningTrace{
					Text:         "Listing them.",
					ProviderKind: "OpenAIChat",
				},
			},
			{Role: modeladapter.RoleUser, Content: "transfer them"},
		}},
	)
	call.Provider.MaxTokens = 1024
	call.Request.MaxTokens = 1024

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), call)

	assert.InDelta(t, modeladapter.ReasoningAnswerFloor, (*captured)["max_tokens"], 0)
	assert.Equal(t, modeladapter.ReasoningAnswerFloor, resp.OutputLimit)
}

func TestOpenAIChatAdapter_KeepsTheConfiguredCeilingForACallThatDoesNotReason(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", finishedChatStream())

	call := modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: []modeladapter.Message{
			{Role: modeladapter.RoleUser, Content: "hi"},
			{
				Role:    modeladapter.RoleAssistant,
				Content: "Hello.",
				Reasoning: &modeladapter.ReasoningTrace{
					Text:         "Signed.",
					ProviderKind: "AnthropicMessages",
				},
			},
			{Role: modeladapter.RoleUser, Content: "again"},
		}},
	)

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), call)

	assert.InDelta(t, 512, (*captured)["max_tokens"], 0)
	assert.Equal(t, 512, resp.OutputLimit)
}

func TestOpenAIChatAdapter_KeepsAHigherConfiguredCeiling(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", finishedChatStream())

	call := modeladaptertest.ReasoningCall(
		aiprovider.KindOpenAIChat, server.URL, aireasoning.ReasoningHigh,
		&modeladapter.Request{Messages: modeladapter.UserMessage("go")},
	)
	call.Request.MaxTokens = 40000

	_, _ = modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), call)

	assert.InDelta(t, 40000, (*captured)["max_tokens"], 0)
}

func TestOpenAIChatAdapter_CompleteReportsACutWithNothingButThinking(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.CaptureServer(t, map[string]any{
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

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("transfer")},
	))

	require.NoError(t, err)
	assert.True(t, resp.Truncated)
	assert.Empty(t, resp.Text)
	require.NotNil(t, resp.Reasoning)
}

func TestOpenAIResponsesAdapter_GivesAReasoningCallRoomToAnswer(t *testing.T) {
	t.Parallel()

	thinking := modeladaptertest.ReasoningCall(
		aiprovider.KindOpenAIResponses, "http://unused", aireasoning.ReasoningMedium,
		&modeladapter.Request{Messages: modeladapter.UserMessage("go")},
	)
	body := openAIResponsesAdapter{}.requestFor(thinking)
	assert.Equal(t, modeladapter.ReasoningAnswerFloor, body.MaxOutputTokens)

	plain := modeladaptertest.CallFor(aiprovider.KindOpenAIResponses, "http://unused",
		&modeladapter.Request{Messages: modeladapter.UserMessage("go")})
	assert.Equal(t, 512, openAIResponsesAdapter{}.requestFor(plain).MaxOutputTokens)
}
