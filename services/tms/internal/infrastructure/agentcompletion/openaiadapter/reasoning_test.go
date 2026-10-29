package openaiadapter

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DeepSeek-shaped servers stream the chain of thought as reasoning_content,
// asked or not. It is read either way, and the effort word is sent only when
// the provider is configured to reason, because models without it reject it.
func TestOpenAIChatAdapter_ReadsReasoningContentAndSendsEffortOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"reasoning_content":"Let me check "},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"reasoning_content":"the dates."},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"Friday."},"finish_reason":"stop"}]}`,
		},
		[2]string{"", "[DONE]"},
	))

	resp, deltas, thoughts := modeladaptertest.StreamWithReasoning(
		t,
		NewOpenAIChatAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindOpenAIChat, server.URL, aireasoning.ReasoningHigh,
			&modeladapter.Request{Messages: modeladapter.UserMessage("when?")},
		),
	)

	assert.Equal(t, []string{"Let me check ", "the dates."}, thoughts)
	assert.Equal(t, []string{"Friday."}, deltas)
	require.NotNil(t, resp.Reasoning)
	assert.Equal(t, "Let me check the dates.", resp.Reasoning.Text)
	assert.Equal(t, "high", (*captured)["reasoning_effort"])
}

func TestOpenAIChatAdapter_SendsNoEffortWhenOff(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":"stop"}]}`,
		},
		[2]string{"", "[DONE]"},
	))

	_, _, _ = modeladaptertest.StreamWithReasoning(
		t,
		NewOpenAIChatAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindOpenAIChat,
			server.URL,
			aireasoning.ReasoningOff,
			&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
		),
	)

	_, sent := (*captured)["reasoning_effort"]
	assert.False(t, sent)
}

// GPT-5-family models reason at their default effort when no effort is sent,
// so a provider set to None has to say so on the wire.
func TestOpenAIChatAdapter_SendsNoneExplicitly(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":"stop"}]}`,
		},
		[2]string{"", "[DONE]"},
	))

	_, _, _ = modeladaptertest.StreamWithReasoning(
		t,
		NewOpenAIChatAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindOpenAIChat,
			server.URL,
			aireasoning.ReasoningNone,
			&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
		),
	)

	assert.Equal(t, "none", (*captured)["reasoning_effort"])
	assert.InDelta(t, 512, (*captured)["max_tokens"], 0, "no room is reserved for thinking")
}

// The chain of thought is never echoed back to a chat server: DeepSeek
// rejects a request that carries reasoning_content in its messages.
func TestToChatMessages_NeverReplaysReasoning(t *testing.T) {
	t.Parallel()

	messages := toChatMessages("", []modeladapter.Message{
		{
			Role:      modeladapter.RoleAssistant,
			Content:   "Friday.",
			Reasoning: &modeladapter.ReasoningTrace{Text: "thought"},
		},
	}, pulid.Nil, "")

	encoded, err := sonic.Marshal(messages)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "reasoning_content")
}

// The Responses API streams a summary of its reasoning and returns the chain
// encrypted, when asked. The summary is what a person reads; the id and the
// encrypted content are what the next call replays ahead of the function
// calls, which are refused as orphans without it.
func TestOpenAIResponsesAdapter_ReadsReasoningSummaryAndKeepsTheEncryptedChain(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{"response.created", `{"type":"response.created","response":{"model":"gpt-x"}}`},
		[2]string{
			"response.reasoning_summary_text.delta",
			`{"type":"response.reasoning_summary_text.delta","delta":"Checking "}`,
		},
		[2]string{
			"response.reasoning_summary_text.delta",
			`{"type":"response.reasoning_summary_text.delta","delta":"the hold."}`,
		},
		[2]string{
			"response.output_text.delta",
			`{"type":"response.output_text.delta","delta":"Placing it."}`,
		},
		[2]string{
			"response.completed",
			`{"type":"response.completed","response":{"model":"gpt-x","status":"completed","output":[{"type":"reasoning","id":"rs_1","summary":[{"type":"summary_text","text":"Checking the hold."}],"encrypted_content":"enc_1"},{"type":"function_call","call_id":"call_1","name":"place_hold","arguments":"{}"}],"usage":{"input_tokens":3,"output_tokens":8}}}`,
		},
	))

	resp, _, thoughts := modeladaptertest.StreamWithReasoning(
		t,
		NewOpenAIResponsesAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindOpenAIResponses,
			server.URL,
			aireasoning.ReasoningLow,
			&modeladapter.Request{
				Messages: modeladapter.UserMessage("hold it"),
				Tools:    modeladaptertest.LookupTool(),
			},
		),
	)

	assert.Equal(t, []string{"Checking ", "the hold."}, thoughts)
	require.NotNil(t, resp.Reasoning)
	assert.Equal(t, "Checking the hold.", resp.Reasoning.Text)
	assert.Equal(t, "rs_1", resp.Reasoning.Signature)
	assert.Equal(t, "enc_1", resp.Reasoning.Encrypted)

	reasoning, _ := (*captured)["reasoning"].(map[string]any)
	require.NotNil(t, reasoning)
	assert.Equal(t, "low", reasoning["effort"])
	assert.Equal(t, "auto", reasoning["summary"])
	assert.Equal(t, []any{"reasoning.encrypted_content"}, (*captured)["include"])
}

func TestToResponsesInput_ReplaysTheReasoningItemAheadOfItsCalls(t *testing.T) {
	t.Parallel()

	items := toResponsesInput("", []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "hold it"},
		{
			Role: modeladapter.RoleAssistant,
			ToolCalls: []modeladapter.ToolCall{
				{ID: "call_1", Name: "place_hold", Arguments: map[string]any{}},
			},
			Reasoning: &modeladapter.ReasoningTrace{
				Text:      "summary",
				Signature: "rs_1",
				Encrypted: "enc_1",
			},
		},
		{Role: modeladapter.RoleTool, ToolCallID: "call_1", Content: "{}"},
	}, nil)

	require.Len(t, items, 4)
	assert.Equal(t, "reasoning", items[1].Type)
	assert.Equal(t, "rs_1", items[1].ID)
	assert.Equal(t, "enc_1", items[1].EncryptedContent)
	assert.Equal(t, "function_call", items[2].Type)
}

// The Responses API refuses a replayed reasoning item without a summary
// ("Missing required parameter: 'input[N].summary'"), even an empty one, while
// a message item must not carry the field at all.
func TestToResponsesInput_SendsASummaryOnEveryReplayedReasoningItemOnly(t *testing.T) {
	t.Parallel()

	items := toResponsesInput("be brief", []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "hold it"},
		{
			Role:    modeladapter.RoleAssistant,
			Content: "Placing the hold.",
			ToolCalls: []modeladapter.ToolCall{
				{ID: "call_1", Name: "place_hold", Arguments: map[string]any{}},
			},
			Reasoning: &modeladapter.ReasoningTrace{Signature: "rs_1"},
		},
		{Role: modeladapter.RoleTool, ToolCallID: "call_1", Content: "{}"},
	}, nil)

	encoded, err := sonic.Marshal(items)
	require.NoError(t, err)

	var wire []map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &wire))
	require.Len(t, wire, 6)

	for idx, item := range wire {
		summary, present := item["summary"]
		if item["type"] == "reasoning" {
			require.True(t, present, "reasoning item %d has no summary", idx)
			assert.Equal(t, []any{}, summary)
			continue
		}
		assert.False(t, present, "item %d (%v) carries a summary", idx, item["type"])
	}
	assert.Equal(t, "reasoning", wire[2]["type"])
}

// None asks the Responses API not to reason, and asks for neither a summary
// nor the encrypted chain, since there is nothing to summarise or replay.
func TestOpenAIResponsesAdapter_SendsNoneWithoutSummaryOrChain(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"response.output_text.delta",
			`{"type":"response.output_text.delta","delta":"Hi"}`,
		},
		[2]string{
			"response.completed",
			`{"type":"response.completed","response":{"model":"gpt-x","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Hi"}]}],"usage":{"input_tokens":3,"output_tokens":1}}}`,
		},
	))

	_, _, _ = modeladaptertest.StreamWithReasoning(
		t,
		NewOpenAIResponsesAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindOpenAIResponses, server.URL, aireasoning.ReasoningNone,
			&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
		),
	)

	reasoning, _ := (*captured)["reasoning"].(map[string]any)
	require.NotNil(t, reasoning)
	assert.Equal(t, map[string]any{"effort": "none"}, reasoning)
	_, included := (*captured)["include"]
	assert.False(t, included)
	assert.InDelta(t, 512, (*captured)["max_output_tokens"], 0, "no room is reserved for thinking")
}

func TestOpenAIResponsesAdapter_SendsMinimalWithTheChain(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"response.completed",
			`{"type":"response.completed","response":{"model":"gpt-x","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Hi"}]}],"usage":{"input_tokens":3,"output_tokens":1}}}`,
		},
	))

	_, _, _ = modeladaptertest.StreamWithReasoning(
		t,
		NewOpenAIResponsesAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindOpenAIResponses, server.URL, aireasoning.ReasoningMinimal,
			&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
		),
	)

	reasoning, _ := (*captured)["reasoning"].(map[string]any)
	require.NotNil(t, reasoning)
	assert.Equal(t, "minimal", reasoning["effort"])
	assert.Equal(t, []any{"reasoning.encrypted_content"}, (*captured)["include"])
}

func TestReplayReasoning_SendsATraceOnlyToTheProtocolThatProducedIt(t *testing.T) {
	t.Parallel()

	fromResponses := &modeladapter.ReasoningTrace{
		Text:         "t",
		Signature:    "rs_1",
		Encrypted:    "enc",
		ProviderKind: string(aiprovider.KindOpenAIResponses),
	}
	fromAnthropic := &modeladapter.ReasoningTrace{
		Text:         "t",
		Signature:    "sig_1",
		ProviderKind: string(aiprovider.KindAnthropicMessages),
	}
	untagged := &modeladapter.ReasoningTrace{Text: "t", Signature: "sig_0"}

	assert.Empty(t, replayReasoning(fromAnthropic))
	assert.Len(t, replayReasoning(fromResponses), 1)
	assert.Len(t, replayReasoning(untagged), 1)
}

// A request's reasoning override wins over the provider's setting, so a
// classifier on a provider configured Off still tells the model not to reason.
func TestOpenAIResponsesAdapter_ARequestCanAskForNoReasoning(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{"response.created", `{"type":"response.created","response":{"model":"gpt-x"}}`},
		[2]string{
			"response.completed",
			`{"type":"response.completed","response":{"model":"gpt-x","status":"completed","output":[],"usage":{"input_tokens":3,"output_tokens":1}}}`,
		},
	))
	_, _ = modeladaptertest.StreamWith(
		t,
		NewOpenAIResponsesAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindOpenAIResponses,
			server.URL,
			aireasoning.ReasoningOff,
			&modeladapter.Request{
				Messages:  modeladapter.UserMessage("hi"),
				Reasoning: aireasoning.ReasoningNone,
			},
		),
	)

	reasoning, _ := (*captured)["reasoning"].(map[string]any)
	require.NotNil(t, reasoning)
	assert.Equal(t, "none", reasoning["effort"])
}
