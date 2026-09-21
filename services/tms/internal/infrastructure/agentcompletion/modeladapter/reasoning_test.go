package modeladapter

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// streamWithReasoning is streamWith with the thinking captured too.
func streamWithReasoning(t *testing.T, adapter Adapter, call *Call) (*Response, []string, []string) {
	t.Helper()

	thoughts := []string{}
	call.Reasoning = func(delta string) { thoughts = append(thoughts, delta) }
	resp, deltas := streamWith(t, adapter, call)

	return resp, deltas, thoughts
}

func reasoningCall(kind aiprovider.Kind, url string, effort aiprovider.ReasoningEffort, req *Request) *Call {
	call := callFor(kind, url, req)
	call.Provider.ReasoningEffort = effort

	return call
}

// Anthropic streams its thinking as its own block. The text reaches the
// reasoning sink, never the text sink, and the block comes back signed so the
// next call can replay it.
func TestAnthropicAdapter_StreamsThinkingSeparatelyAndKeepsTheSignature(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{"message_start", `{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":9}}}`},
		[2]string{"content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`},
		[2]string{"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"The card "}}`},
		[2]string{"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"expires soon."}}`},
		[2]string{"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig_abc"}}`},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":0}`},
		[2]string{"content_block_start", `{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`},
		[2]string{"content_block_delta", `{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"It expires Friday."}}`},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":1}`},
		[2]string{"message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":12}}`},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	resp, deltas, thoughts := streamWithReasoning(t, NewAnthropicAdapter(), reasoningCall(
		aiprovider.KindAnthropicMessages, server.URL, aiprovider.ReasoningMedium,
		&Request{System: "sys", Messages: UserMessage("When?"), MaxTokens: 1000},
	))

	assert.Equal(t, []string{"The card ", "expires soon."}, thoughts)
	assert.Equal(t, []string{"It expires Friday."}, deltas, "thinking never leaks into the reply")
	require.NotNil(t, resp.Reasoning)
	assert.Equal(t, "The card expires soon.", resp.Reasoning.Text)
	assert.Equal(t, "sig_abc", resp.Reasoning.Signature)

	thinking, _ := (*captured)["thinking"].(map[string]any)
	require.NotNil(t, thinking, "medium effort asks for thinking")
	assert.InDelta(t, 4096, thinking["budget_tokens"], 0)
	assert.InDelta(t, 4096+thinkingAnswerRoom, (*captured)["max_tokens"], 0,
		"the reply ceiling is raised to fit the budget with room to answer")
}

func TestAnthropicAdapter_SendsNoThinkingWhenTheProviderIsOff(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{"message_start", `{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":1}}}`},
		[2]string{"content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`},
		[2]string{"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":0}`},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	call := reasoningCall(
		aiprovider.KindAnthropicMessages, server.URL, aiprovider.ReasoningOff,
		&Request{Messages: UserMessage("hi")},
	)
	call.Request.MaxTokens = 1000
	resp, _, thoughts := streamWithReasoning(t, NewAnthropicAdapter(), call)

	_, asked := (*captured)["thinking"]
	assert.False(t, asked)
	assert.InDelta(t, 1000, (*captured)["max_tokens"], 0, "untouched when nothing asks for thinking")
	assert.Empty(t, thoughts)
	assert.Nil(t, resp.Reasoning)
}

// Anthropic refuses a tool result whose preceding thinking is missing, so a
// replayed assistant turn opens with the signed block it came with, and any
// redacted block verbatim.
func TestToAnthropicMessages_ReplaysSignedThinkingAheadOfToolCalls(t *testing.T) {
	t.Parallel()

	messages := toAnthropicMessages([]Message{
		{Role: RoleUser, Content: "hold it"},
		{
			Role:      RoleAssistant,
			ToolCalls: []ToolCall{{ID: "toolu_1", Name: "place_hold", Arguments: map[string]any{}}},
			Reasoning: &ReasoningTrace{Text: "Hold seems right.", Signature: "sig_1", Redacted: []string{"blob"}},
		},
		{Role: RoleTool, ToolCallID: "toolu_1", Content: "{}"},
	})

	require.Len(t, messages, 3)
	blocks := messages[1].Content
	require.Len(t, blocks, 3)
	assert.Equal(t, "thinking", blocks[0].Type)
	assert.Equal(t, "Hold seems right.", blocks[0].Thinking)
	assert.Equal(t, "sig_1", blocks[0].Signature)
	assert.Equal(t, "redacted_thinking", blocks[1].Type)
	assert.Equal(t, "blob", blocks[1].Data)
	assert.Equal(t, "tool_use", blocks[2].Type)
}

// A trace without a signature — one read off a provider that never signed it
// — is not replayed to Anthropic, which would reject an unsigned block.
func TestToAnthropicMessages_DoesNotReplayUnsignedThinking(t *testing.T) {
	t.Parallel()

	messages := toAnthropicMessages([]Message{
		{Role: RoleAssistant, Content: "Friday.", Reasoning: &ReasoningTrace{Text: "thought"}},
	})

	require.Len(t, messages, 1)
	require.Len(t, messages[0].Content, 1)
	assert.Equal(t, "text", messages[0].Content[0].Type)
}

// DeepSeek-shaped servers stream the chain of thought as reasoning_content,
// asked or not. It is read either way, and the effort word is sent only when
// the provider is configured to reason, because models without it reject it.
func TestOpenAIChatAdapter_ReadsReasoningContentAndSendsEffortOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{"", `{"model":"m","choices":[{"index":0,"delta":{"reasoning_content":"Let me check "},"finish_reason":null}]}`},
		[2]string{"", `{"model":"m","choices":[{"index":0,"delta":{"reasoning_content":"the dates."},"finish_reason":null}]}`},
		[2]string{"", `{"model":"m","choices":[{"index":0,"delta":{"content":"Friday."},"finish_reason":"stop"}]}`},
		[2]string{"", "[DONE]"},
	))

	resp, deltas, thoughts := streamWithReasoning(t, NewOpenAIChatAdapter(), reasoningCall(
		aiprovider.KindOpenAIChat, server.URL, aiprovider.ReasoningHigh,
		&Request{Messages: UserMessage("when?")},
	))

	assert.Equal(t, []string{"Let me check ", "the dates."}, thoughts)
	assert.Equal(t, []string{"Friday."}, deltas)
	require.NotNil(t, resp.Reasoning)
	assert.Equal(t, "Let me check the dates.", resp.Reasoning.Text)
	assert.Equal(t, "high", (*captured)["reasoning_effort"])
}

func TestOpenAIChatAdapter_SendsNoEffortWhenOff(t *testing.T) {
	t.Parallel()

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{"", `{"model":"m","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":"stop"}]}`},
		[2]string{"", "[DONE]"},
	))

	_, _, _ = streamWithReasoning(t, NewOpenAIChatAdapter(), reasoningCall(
		aiprovider.KindOpenAIChat, server.URL, aiprovider.ReasoningOff, &Request{Messages: UserMessage("hi")},
	))

	_, sent := (*captured)["reasoning_effort"]
	assert.False(t, sent)
}

// The chain of thought is never echoed back to a chat server: DeepSeek
// rejects a request that carries reasoning_content in its messages.
func TestToChatMessages_NeverReplaysReasoning(t *testing.T) {
	t.Parallel()

	messages := toChatMessages("", []Message{
		{Role: RoleAssistant, Content: "Friday.", Reasoning: &ReasoningTrace{Text: "thought"}},
	})

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

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{"response.created", `{"type":"response.created","response":{"model":"gpt-x"}}`},
		[2]string{"response.reasoning_summary_text.delta", `{"type":"response.reasoning_summary_text.delta","delta":"Checking "}`},
		[2]string{"response.reasoning_summary_text.delta", `{"type":"response.reasoning_summary_text.delta","delta":"the hold."}`},
		[2]string{"response.output_text.delta", `{"type":"response.output_text.delta","delta":"Placing it."}`},
		[2]string{"response.completed", `{"type":"response.completed","response":{"model":"gpt-x","status":"completed","output":[{"type":"reasoning","id":"rs_1","summary":[{"type":"summary_text","text":"Checking the hold."}],"encrypted_content":"enc_1"},{"type":"function_call","call_id":"call_1","name":"place_hold","arguments":"{}"}],"usage":{"input_tokens":3,"output_tokens":8}}}`},
	))

	resp, _, thoughts := streamWithReasoning(t, NewOpenAIResponsesAdapter(), reasoningCall(
		aiprovider.KindOpenAIResponses, server.URL, aiprovider.ReasoningLow,
		&Request{Messages: UserMessage("hold it"), Tools: lookupTool()},
	))

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

	items := toResponsesInput("", []Message{
		{Role: RoleUser, Content: "hold it"},
		{
			Role:      RoleAssistant,
			ToolCalls: []ToolCall{{ID: "call_1", Name: "place_hold", Arguments: map[string]any{}}},
			Reasoning: &ReasoningTrace{Text: "summary", Signature: "rs_1", Encrypted: "enc_1"},
		},
		{Role: RoleTool, ToolCallID: "call_1", Content: "{}"},
	})

	require.Len(t, items, 4)
	assert.Equal(t, "reasoning", items[1].Type)
	assert.Equal(t, "rs_1", items[1].ID)
	assert.Equal(t, "enc_1", items[1].EncryptedContent)
	assert.Equal(t, "function_call", items[2].Type)
}

// Ollama reports a thinking model's reasoning in its own field when asked.
func TestOllamaAdapter_ReadsThinkingAndAsksForItWhenConfigured(t *testing.T) {
	t.Parallel()

	body := strings.Join([]string{
		`{"model":"qwen","message":{"role":"assistant","content":"","thinking":"Weighing "},"done":false}`,
		`{"model":"qwen","message":{"role":"assistant","content":"","thinking":"the options."},"done":false}`,
		`{"model":"qwen","message":{"role":"assistant","content":"Friday."},"done":true,"done_reason":"stop","prompt_eval_count":1,"eval_count":2}`,
	}, "\n") + "\n"
	server, captured := streamServer(t, "application/x-ndjson", body)

	resp, deltas, thoughts := streamWithReasoning(t, NewOllamaAdapter(), reasoningCall(
		aiprovider.KindOllama, server.URL, aiprovider.ReasoningMedium, &Request{Messages: UserMessage("when?")},
	))

	assert.Equal(t, []string{"Weighing ", "the options."}, thoughts)
	assert.Equal(t, []string{"Friday."}, deltas)
	require.NotNil(t, resp.Reasoning)
	assert.Equal(t, "Weighing the options.", resp.Reasoning.Text)
	assert.Equal(t, true, (*captured)["think"])
}
