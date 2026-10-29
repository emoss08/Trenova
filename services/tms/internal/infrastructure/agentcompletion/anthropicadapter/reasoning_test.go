package anthropicadapter

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Anthropic streams its thinking as its own block. The text reaches the
// reasoning sink, never the text sink, and the block comes back signed so the
// next call can replay it.
func TestAnthropicAdapter_StreamsThinkingSeparatelyAndKeepsTheSignature(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":9}}}`,
		},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"The card "}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"expires soon."}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig_abc"}}`,
		},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":0}`},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"It expires Friday."}}`,
		},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":1}`},
		[2]string{
			"message_delta",
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":12}}`,
		},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	resp, deltas, thoughts := modeladaptertest.StreamWithReasoning(
		t,
		NewAnthropicAdapter(),
		modeladaptertest.ReasoningCall(
			aiprovider.KindAnthropicMessages,
			server.URL,
			aireasoning.ReasoningMedium,
			&modeladapter.Request{
				System:    "sys",
				Messages:  modeladapter.UserMessage("When?"),
				MaxTokens: 1000,
			},
		),
	)

	assert.Equal(t, []string{"The card ", "expires soon."}, thoughts)
	assert.Equal(t, []string{"It expires Friday."}, deltas, "thinking never leaks into the reply")
	require.NotNil(t, resp.Reasoning)
	assert.Equal(t, "The card expires soon.", resp.Reasoning.Text)
	assert.Equal(t, "sig_abc", resp.Reasoning.Signature)

	thinking, _ := (*captured)["thinking"].(map[string]any)
	require.NotNil(t, thinking, "medium effort asks for thinking")
	assert.InDelta(t, 4096, thinking["budget_tokens"], 0)
	assert.InDelta(t, 4096+modeladapter.ThinkingAnswerRoom, (*captured)["max_tokens"], 0,
		"the reply ceiling is raised to fit the budget with room to answer")
}

func TestAnthropicAdapter_SendsNoThinkingWhenTheProviderIsOff(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":1}}}`,
		},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`,
		},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":0}`},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	call := modeladaptertest.ReasoningCall(
		aiprovider.KindAnthropicMessages, server.URL, aireasoning.ReasoningOff,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	)
	call.Request.MaxTokens = 1000
	resp, _, thoughts := modeladaptertest.StreamWithReasoning(t, NewAnthropicAdapter(), call)

	_, asked := (*captured)["thinking"]
	assert.False(t, asked)
	assert.InDelta(
		t,
		1000,
		(*captured)["max_tokens"],
		0,
		"untouched when nothing asks for thinking",
	)
	assert.Empty(t, thoughts)
	assert.Nil(t, resp.Reasoning)
}

// Anthropic refuses a tool result whose preceding thinking is missing, so a
// replayed assistant turn opens with the signed block it came with, and any
// redacted block verbatim.
func TestToAnthropicMessages_ReplaysSignedThinkingAheadOfToolCalls(t *testing.T) {
	t.Parallel()

	messages := toAnthropicMessages([]modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "hold it"},
		{
			Role: modeladapter.RoleAssistant,
			ToolCalls: []modeladapter.ToolCall{
				{ID: "toolu_1", Name: "place_hold", Arguments: map[string]any{}},
			},
			Reasoning: &modeladapter.ReasoningTrace{
				Text:      "Hold seems right.",
				Signature: "sig_1",
				Redacted:  []string{"blob"},
			},
		},
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_1", Content: "{}"},
	})

	require.Len(t, messages, 3)
	blocks := messages[1].Content
	require.Len(t, blocks, 3)
	require.NotNil(t, blocks[0].OfThinking)
	assert.Equal(t, "Hold seems right.", blocks[0].OfThinking.Thinking)
	assert.Equal(t, "sig_1", blocks[0].OfThinking.Signature)
	require.NotNil(t, blocks[1].OfRedactedThinking)
	assert.Equal(t, "blob", blocks[1].OfRedactedThinking.Data)
	assert.NotNil(t, blocks[2].OfToolUse)
}

// A signed thinking block whose summary came back empty still carries the
// thinking field: Anthropic refuses the block without it, which failed every
// tool loop where the model thought briefly. Other blocks never carry it.
func TestToAnthropicMessages_ReplaysEmptyThinkingWithItsField(t *testing.T) {
	t.Parallel()

	messages := toAnthropicMessages([]modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "who works here"},
		{
			Role: modeladapter.RoleAssistant,
			ToolCalls: []modeladapter.ToolCall{
				{ID: "toolu_1", Name: "find_in_trenova", Arguments: map[string]any{}},
			},
			Reasoning: &modeladapter.ReasoningTrace{Signature: "sig_1"},
		},
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_1", Content: "{}"},
	})

	require.Len(t, messages, 3)
	encoded, err := sonic.Marshal(messages[1].Content)
	require.NoError(t, err)

	var blocks []map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &blocks))
	require.Len(t, blocks, 2)
	assert.Equal(t, "thinking", blocks[0]["type"])
	thinking, present := blocks[0]["thinking"]
	assert.True(t, present)
	assert.Empty(t, thinking)
	assert.Equal(t, "sig_1", blocks[0]["signature"])
	assert.Equal(t, "tool_use", blocks[1]["type"])
	assert.NotContains(t, blocks[1], "thinking")
}

// A trace without a signature — one read off a provider that never signed it
// — is not replayed to Anthropic, which would reject an unsigned block.
func TestToAnthropicMessages_DoesNotReplayUnsignedThinking(t *testing.T) {
	t.Parallel()

	messages := toAnthropicMessages([]modeladapter.Message{
		{
			Role:      modeladapter.RoleAssistant,
			Content:   "Friday.",
			Reasoning: &modeladapter.ReasoningTrace{Text: "thought"},
		},
	})

	require.Len(t, messages, 1)
	require.Len(t, messages[0].Content, 1)
	assert.NotNil(t, messages[0].Content[0].OfText)
}

// A thread can change model between turns. A signature Anthropic issued
// means nothing to the Responses API and the other way round, and each
// refuses the other's with a 400, so a trace is replayed only by the kind
// that produced it. A trace from before kinds were recorded is replayed as
// it always was.
func TestReplayThinking_SendsATraceOnlyToTheProtocolThatProducedIt(t *testing.T) {
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

	assert.Empty(t, replayThinking(fromResponses))
	assert.Len(t, replayThinking(fromAnthropic), 1)
	assert.Len(t, replayThinking(untagged), 1)
}
