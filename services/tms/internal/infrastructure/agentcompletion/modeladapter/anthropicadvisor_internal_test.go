package modeladapter

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const advisorBlockingReply = `{"id":"msg_01","model":"claude-haiku-5-5","stop_reason":"tool_use",` +
	`"content":[` +
	`{"type":"thinking","thinking":"plan","signature":"sig_a"},` +
	`{"type":"server_tool_use","id":"srvtoolu_1","name":"advisor","input":{}},` +
	`{"type":"advisor_tool_result","tool_use_id":"srvtoolu_1",` +
	`"content":{"type":"advisor_redacted_result","encrypted_content":"opaque_blob","stop_reason":"end_turn"}},` +
	`{"type":"thinking","thinking":"follow it","signature":"sig_b"},` +
	`{"type":"text","text":"Checking the shipment."},` +
	`{"type":"tool_use","id":"toolu_1","name":"get_shipment","input":{"number":"S1"}}],` +
	`"usage":{"input_tokens":900,"output_tokens":120,"cache_read_input_tokens":4000,` +
	`"cache_creation_input_tokens":300,"iterations":[` +
	`{"type":"message","input_tokens":500,"output_tokens":40,"model":"claude-haiku-5-5"},` +
	`{"type":"advisor_message","input_tokens":2500,"output_tokens":700,` +
	`"cache_read_input_tokens":1000,"cache_creation_input_tokens":200,"model":"claude-opus-5-5"},` +
	`{"type":"message","input_tokens":400,"output_tokens":80,"model":"claude-haiku-5-5"}]}}`

func advisorStreamReply(content string) string {
	return "event: message_start\n" +
		`data: {"type":"message_start","message":{"id":"msg_02","model":"claude-haiku-5-5",` +
		`"usage":{"input_tokens":700,"output_tokens":1}}}` + "\n\n" +
		"event: content_block_start\n" +
		`data: {"type":"content_block_start","index":0,"content_block":` +
		`{"type":"server_tool_use","id":"srvtoolu_9","name":"advisor","input":{}}}` + "\n\n" +
		"event: content_block_stop\n" +
		`data: {"type":"content_block_stop","index":0}` + "\n\n" +
		"event: content_block_start\n" +
		`data: {"type":"content_block_start","index":1,"content_block":` +
		`{"type":"advisor_tool_result","tool_use_id":"srvtoolu_9","content":` + content + `}}` + "\n\n" +
		"event: content_block_stop\n" +
		`data: {"type":"content_block_stop","index":1}` + "\n\n" +
		"event: content_block_start\n" +
		`data: {"type":"content_block_start","index":2,"content_block":{"type":"text","text":""}}` +
		"\n\n" +
		"event: content_block_delta\n" +
		`data: {"type":"content_block_delta","index":2,"delta":{"type":"text_delta","text":"Done."}}` +
		"\n\n" +
		"event: content_block_stop\n" +
		`data: {"type":"content_block_stop","index":2}` + "\n\n" +
		"event: message_delta\n" +
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":30,` +
		`"iterations":[{"type":"message","input_tokens":300,"output_tokens":10},` +
		`{"type":"advisor_message","input_tokens":1200,"output_tokens":400,"model":"claude-opus-5-5"},` +
		`{"type":"message","input_tokens":400,"output_tokens":20}]}}` + "\n\n" +
		"event: message_stop\n" +
		`data: {"type":"message_stop"}` + "\n\n"
}

func advisorCall(t *testing.T, url string, req *Request) *Call {
	t.Helper()

	call := reasoningCall(aiprovider.KindAnthropicMessages, url, aiprovider.ReasoningLow, req)
	call.Provider.Model = "claude-haiku-5-5"
	call.APIKey = "sk-test"

	return call
}

func advisorRequest() Request {
	return Request{
		System:   "Rules.",
		Messages: UserMessage("Where is S1?"),
		Tools:    wireTools(),
		Advisor:  agentports.DefaultAdvisor(""),
	}
}

func bodyTools(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()

	raw, ok := body["tools"].([]any)
	require.True(t, ok)
	tools := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		tool, isTool := entry.(map[string]any)
		require.True(t, isTool)
		tools = append(tools, tool)
	}

	return tools
}

func TestAnthropicAdapter_DeclaresTheAdvisorLastWithItsBeta(t *testing.T) {
	t.Parallel()

	server, captured := wireServer(t, "application/json", advisorBlockingReply)
	request := advisorRequest()
	request.Advisor.MaxTokens = 2048
	_, err := NewAnthropicAdapter().Complete(t.Context(), advisorCall(t, server.URL, &request))
	require.NoError(t, err)

	capture := captured()
	assert.Contains(t, capture.Headers["Anthropic-Beta"], "advisor-tool-2026-03-01")
	tools := bodyTools(t, capture.Body)
	require.Len(t, tools, 3)
	assert.Equal(t, "get_shipment", tools[0]["name"])
	assert.Equal(t, map[string]any{
		"type":          "advisor_20260301",
		"name":          "advisor",
		"model":         "claude-opus-5-5",
		"max_uses":      float64(2),
		"max_tokens":    float64(2048),
		"caching":       map[string]any{"type": "ephemeral", "ttl": "5m"},
		"cache_control": map[string]any{"type": "ephemeral"},
	}, tools[2], "the advisor is a server tool after every client tool, never deferred, "+
		"and carries the tools' cache mark")
	assert.NotContains(t, tools[1], "cache_control")
}

func TestAnthropicAdapter_WithoutAnAdvisorSendsNoneOfIt(t *testing.T) {
	t.Parallel()

	server, captured := wireServer(t, "application/json", wireBlockingReply)
	request := advisorRequest()
	request.Advisor = nil
	_, err := NewAnthropicAdapter().Complete(t.Context(), advisorCall(t, server.URL, &request))
	require.NoError(t, err)

	capture := captured()
	assert.NotContains(t, capture.Headers["Anthropic-Beta"], "advisor")
	for _, tool := range bodyTools(t, capture.Body) {
		assert.NotEqual(t, "advisor", tool["name"])
	}
}

func TestAnthropicAdapter_ReadsAnAdvisedReplyWithoutShowingTheAdvice(t *testing.T) {
	t.Parallel()

	server, _ := wireServer(t, "application/json", advisorBlockingReply)
	request := advisorRequest()
	resp, err := NewAnthropicAdapter().Complete(t.Context(), advisorCall(t, server.URL, &request))
	require.NoError(t, err)

	assert.Equal(t, "Checking the shipment.", resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "get_shipment", resp.ToolCalls[0].Name)
	assert.Equal(t, 900, resp.InputTokens, "the executor's own usage is unchanged")
	assert.Equal(t, 120, resp.OutputTokens)
	assert.Equal(t, &aiprovider.AdvisorUsage{
		Model: "claude-opus-5-5",
		Calls: 1,
		Tokens: aiprovider.TokenUsage{
			Input: 2500, Output: 700, CacheRead: 1000, CacheWrite: 200,
		},
		Executor: []aiprovider.TokenUsage{
			{Input: 500, Output: 40},
			{Input: 400, Output: 80},
		},
	}, resp.Advisor)

	require.NotNil(t, resp.Reasoning)
	assert.Equal(t, "planfollow it", resp.Reasoning.Text)
	assert.NotContains(t, resp.Reasoning.Text, "opaque_blob")
	assert.Equal(t, []ProviderBlock{
		{Type: "thinking", Text: "plan", Signature: "sig_a"},
		{Type: "server_tool_use", ID: "srvtoolu_1", Name: "advisor", Input: map[string]any{}},
		{
			Type:      "advisor_tool_result",
			ToolUseID: "srvtoolu_1",
			Model:     "claude-opus-5-5",
			Result: &conversation.ServerToolResult{
				Type:             "advisor_redacted_result",
				EncryptedContent: "opaque_blob",
				StopReason:       "end_turn",
			},
		},
		{Type: "thinking", Text: "follow it", Signature: "sig_b"},
		{Type: "text", Text: "Checking the shipment."},
	}, resp.Reasoning.Blocks)
	assert.Equal(t, "claude-opus-5-5", resp.Reasoning.AdvisorModel())
}

func TestAnthropicAdapter_StreamsEachAdvisorResultKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    conversation.ServerToolResult
	}{
		{
			name:    "plaintext",
			content: `{"type":"advisor_result","text":"Check the stops first.","stop_reason":"end_turn"}`,
			want: conversation.ServerToolResult{
				Type: "advisor_result", Text: "Check the stops first.", StopReason: "end_turn",
			},
		},
		{
			name: "redacted",
			content: `{"type":"advisor_redacted_result","encrypted_content":"enc_1",` +
				`"stop_reason":"max_tokens"}`,
			want: conversation.ServerToolResult{
				Type: "advisor_redacted_result", EncryptedContent: "enc_1", StopReason: "max_tokens",
			},
		},
		{
			name:    "error",
			content: `{"type":"advisor_tool_result_error","error_code":"max_uses_exceeded"}`,
			want: conversation.ServerToolResult{
				Type: "advisor_tool_result_error", ErrorCode: "max_uses_exceeded",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server, _ := wireServer(t, "text/event-stream", advisorStreamReply(tc.content))
			request := advisorRequest()
			resp, deltas := streamWith(t, NewAnthropicAdapter(), advisorCall(t, server.URL, &request))

			assert.Equal(t, "Done.", resp.Text)
			assert.Equal(t, []string{"Done."}, deltas, "the advice never reaches the reply")
			require.NotNil(t, resp.Reasoning)
			require.Len(t, resp.Reasoning.Blocks, 3)
			assert.Equal(t, ProviderBlock{
				Type: "server_tool_use", ID: "srvtoolu_9", Name: "advisor", Input: map[string]any{},
			}, resp.Reasoning.Blocks[0])
			result := resp.Reasoning.Blocks[1]
			assert.Equal(t, "srvtoolu_9", result.ToolUseID)
			assert.Equal(t, "claude-opus-5-5", result.Model)
			require.NotNil(t, result.Result)
			assert.Equal(t, tc.want, *result.Result)
			assert.Empty(t, resp.Reasoning.Text)
			assert.Equal(t, &aiprovider.AdvisorUsage{
				Model:  "claude-opus-5-5",
				Calls:  1,
				Tokens: aiprovider.TokenUsage{Input: 1200, Output: 400},
				Executor: []aiprovider.TokenUsage{
					{Input: 300, Output: 10},
					{Input: 400, Output: 20},
				},
			}, resp.Advisor)
		})
	}
}

const advisorAroundTextReply = `{"id":"msg_03","model":"claude-haiku-5-5","stop_reason":"end_turn",` +
	`"content":[` +
	`{"type":"text","text":"Let me check. "},` +
	`{"type":"server_tool_use","id":"srvtoolu_2","name":"advisor","input":{}},` +
	`{"type":"advisor_tool_result","tool_use_id":"srvtoolu_2",` +
	`"content":{"type":"advisor_result","text":"Look at the last stop.","stop_reason":"end_turn"}},` +
	`{"type":"text","text":"S1 is in Dallas."}],` +
	`"usage":{"input_tokens":700,"output_tokens":30}}`

func advisorAroundTextStream() string {
	return "event: message_start\n" +
		`data: {"type":"message_start","message":{"id":"msg_04","model":"claude-haiku-5-5",` +
		`"usage":{"input_tokens":700,"output_tokens":1}}}` + "\n\n" +
		"event: content_block_start\n" +
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}` +
		"\n\n" +
		"event: content_block_delta\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta",` +
		`"text":"Let me check. "}}` + "\n\n" +
		"event: content_block_stop\n" +
		`data: {"type":"content_block_stop","index":0}` + "\n\n" +
		"event: content_block_start\n" +
		`data: {"type":"content_block_start","index":1,"content_block":` +
		`{"type":"server_tool_use","id":"srvtoolu_2","name":"advisor","input":{}}}` + "\n\n" +
		"event: content_block_stop\n" +
		`data: {"type":"content_block_stop","index":1}` + "\n\n" +
		"event: content_block_start\n" +
		`data: {"type":"content_block_start","index":2,"content_block":` +
		`{"type":"advisor_tool_result","tool_use_id":"srvtoolu_2","content":` +
		`{"type":"advisor_result","text":"Look at the last stop.","stop_reason":"end_turn"}}}` +
		"\n\n" +
		"event: content_block_stop\n" +
		`data: {"type":"content_block_stop","index":2}` + "\n\n" +
		"event: content_block_start\n" +
		`data: {"type":"content_block_start","index":3,"content_block":{"type":"text","text":""}}` +
		"\n\n" +
		"event: content_block_delta\n" +
		`data: {"type":"content_block_delta","index":3,"delta":{"type":"text_delta",` +
		`"text":"S1 is in Dallas."}}` + "\n\n" +
		"event: content_block_stop\n" +
		`data: {"type":"content_block_stop","index":3}` + "\n\n" +
		"event: message_delta\n" +
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},` +
		`"usage":{"output_tokens":30}}` + "\n\n" +
		"event: message_stop\n" +
		`data: {"type":"message_stop"}` + "\n\n"
}

func assertTextAroundTheAdvice(t *testing.T, resp *Response) {
	t.Helper()

	assert.Equal(t, "Let me check. S1 is in Dallas.", resp.Text,
		"every text block, in order, with the advice between them left out")
	require.NotNil(t, resp.Reasoning)
	replayed := replayProviderBlocks(resp.Reasoning.Blocks, resp.Text, true)
	texts := make([]string, 0, 2)
	for idx := range replayed {
		if replayed[idx].OfText != nil {
			texts = append(texts, replayed[idx].OfText.Text)
		}
	}
	assert.Equal(t, []string{"Let me check. ", "S1 is in Dallas."}, texts,
		"the joined reply is not sent back a second time after its own blocks")
}

func TestAnthropicAdapter_JoinsTheTextOnBothSidesOfTheAdvice(t *testing.T) {
	t.Parallel()

	server, _ := wireServer(t, "application/json", advisorAroundTextReply)
	request := advisorRequest()
	resp, err := NewAnthropicAdapter().Complete(t.Context(), advisorCall(t, server.URL, &request))
	require.NoError(t, err)
	assertTextAroundTheAdvice(t, resp)
}

func TestAnthropicAdapter_StreamsTheTextOnBothSidesOfTheAdvice(t *testing.T) {
	t.Parallel()

	server, _ := wireServer(t, "text/event-stream", advisorAroundTextStream())
	request := advisorRequest()
	resp, deltas := streamWith(t, NewAnthropicAdapter(), advisorCall(t, server.URL, &request))
	assert.Equal(t, []string{"Let me check. ", "S1 is in Dallas."}, deltas)
	assert.Equal(t, strings.Join(deltas, ""), resp.Text, "the reply is what streamed")
	assertTextAroundTheAdvice(t, resp)
}

func advisedHistory(advice *ReasoningTrace) []Message {
	advice.ProviderKind = string(aiprovider.KindAnthropicMessages)

	return []Message{
		{Role: RoleUser, Content: "Where is S1?"},
		{
			Role:      RoleAssistant,
			Content:   "Checking the shipment.",
			Reasoning: advice,
			ToolCalls: []ToolCall{{
				ID: "toolu_1", Name: "get_shipment", Arguments: map[string]any{"number": "S1"},
			}},
		},
		{Role: RoleTool, ToolCallID: "toolu_1", ToolName: "get_shipment", Content: "In Dallas."},
	}
}

func messageBlocks(t *testing.T, body map[string]any, index int) []map[string]any {
	t.Helper()

	messages, ok := body["messages"].([]any)
	require.True(t, ok)
	message, ok := messages[index].(map[string]any)
	require.True(t, ok)
	raw, ok := message["content"].([]any)
	require.True(t, ok)
	blocks := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		block, isBlock := entry.(map[string]any)
		require.True(t, isBlock)
		blocks = append(blocks, block)
	}

	return blocks
}

func replayedAdvice() []map[string]any {
	return []map[string]any{
		{"type": "thinking", "thinking": "plan", "signature": "sig_a"},
		{"type": "server_tool_use", "id": "srvtoolu_1", "name": "advisor", "input": map[string]any{}},
		{
			"type":        "advisor_tool_result",
			"tool_use_id": "srvtoolu_1",
			"content": map[string]any{
				"type":              "advisor_redacted_result",
				"encrypted_content": "opaque_blob",
				"stop_reason":       "end_turn",
			},
		},
		{"type": "thinking", "thinking": "follow it", "signature": "sig_b"},
		{"type": "text", "text": "Checking the shipment."},
		{
			"type": "tool_use", "id": "toolu_1", "name": "get_shipment",
			"input": map[string]any{"number": "S1"},
		},
	}
}

func advisedReply(t *testing.T) *Response {
	t.Helper()

	first, _ := wireServer(t, "application/json", advisorBlockingReply)
	request := advisorRequest()
	resp, err := NewAnthropicAdapter().Complete(t.Context(), advisorCall(t, first.URL, &request))
	require.NoError(t, err)

	return resp
}

func TestAnthropicAdapter_ReplaysTheAdviceUnchangedWhileTheAdvisorIsAsked(t *testing.T) {
	t.Parallel()

	resp := advisedReply(t)
	server, captured := wireServer(t, "application/json", wireBlockingReply)
	next := Request{
		Messages: advisedHistory(resp.Reasoning),
		Tools:    wireTools(),
		Advisor:  agentports.DefaultAdvisor(""),
	}
	_, err := NewAnthropicAdapter().Complete(t.Context(), advisorCall(t, server.URL, &next))
	require.NoError(t, err)

	capture := captured()
	assert.Contains(t, capture.Headers["Anthropic-Beta"], "advisor-tool-2026-03-01")
	tools := bodyTools(t, capture.Body)
	assert.Equal(t, "advisor", tools[len(tools)-1]["name"])
	assert.Equal(t, "claude-opus-5-5", tools[len(tools)-1]["model"])
	assert.Equal(t, replayedAdvice(), messageBlocks(t, capture.Body, 1),
		"the assistant turn goes back in the order it came")
}

func TestAnthropicAdapter_ReplaysTheAdviceWithOnlyTheBetaWhenTheAdvisorIsNotAsked(t *testing.T) {
	t.Parallel()

	resp := advisedReply(t)
	server, captured := wireServer(t, "application/json", wireBlockingReply)
	next := Request{Messages: advisedHistory(resp.Reasoning), Tools: wireTools()}
	_, err := NewAnthropicAdapter().Complete(t.Context(), advisorCall(t, server.URL, &next))
	require.NoError(t, err)

	capture := captured()
	assert.Contains(t, capture.Headers["Anthropic-Beta"], "advisor-tool-2026-03-01",
		"the beta lets the API read the advice the history holds")
	for _, tool := range bodyTools(t, capture.Body) {
		assert.NotEqual(t, "advisor", tool["name"], "a request that does not ask is not advised")
		assert.NotEqual(t, "advisor_20260301", tool["type"])
	}
	assert.Equal(t, replayedAdvice(), messageBlocks(t, capture.Body, 1),
		"the advice still goes back unchanged")
}

func TestAnthropicConversation_KeepsAdviceOfEarlierTurnsButNotTheirThinking(t *testing.T) {
	t.Parallel()

	advice := &ReasoningTrace{
		Text:      "plan",
		Signature: "sig_a",
		Blocks: []ProviderBlock{
			{Type: "thinking", Text: "plan", Signature: "sig_a"},
			{Type: "server_tool_use", ID: "srvtoolu_1", Name: "advisor"},
			{
				Type: "advisor_tool_result", ToolUseID: "srvtoolu_1", Model: "claude-opus-5-5",
				Result: &conversation.ServerToolResult{
					Type: "advisor_result", Text: "Ask for the BOL.",
				},
			},
			{Type: "text", Text: "Checking the shipment."},
		},
	}
	messages := append(advisedHistory(advice),
		Message{Role: RoleAssistant, Content: "It is in Dallas."},
		Message{Role: RoleUser, Content: "And S2?"},
	)

	out := toAnthropicMessages(messages)
	blocks := out[1].Content
	require.Len(t, blocks, 4)
	require.NotNil(t, blocks[0].OfServerToolUse)
	require.NotNil(t, blocks[1].OfAdvisorToolResult)
	result := blocks[1].OfAdvisorToolResult.Content.OfRequestAdvisorResultBlock
	require.NotNil(t, result)
	assert.Equal(t, "Ask for the BOL.", result.Text)
	assert.False(t, result.StopReason.Valid())
	assert.Equal(t, "Checking the shipment.", blocks[2].OfText.Text)
	require.NotNil(t, blocks[3].OfToolUse)
}

func TestAnthropicConversation_ReplaysAnAdvisorErrorAsItCame(t *testing.T) {
	t.Parallel()

	blocks := replayProviderBlocks([]ProviderBlock{
		{Type: "server_tool_use", ID: "srvtoolu_1", Name: "advisor"},
		{
			Type: "advisor_tool_result", ToolUseID: "srvtoolu_1",
			Result: &conversation.ServerToolResult{
				Type: "advisor_tool_result_error", ErrorCode: "overloaded",
			},
		},
	}, "Going on alone.", true)

	require.Len(t, blocks, 3)
	failure := blocks[1].OfAdvisorToolResult.Content.OfRequestAdvisorToolResultError
	require.NotNil(t, failure)
	assert.Equal(t, "overloaded", string(failure.ErrorCode))
	assert.Equal(t, "Going on alone.", blocks[2].OfText.Text,
		"a reply text the blocks do not hold still goes back")
}

func TestAdvisorFor_ComesOnlyFromTheRequestAndTheHistoryOnlyAsksForTheBeta(t *testing.T) {
	t.Parallel()

	direct := &aiprovider.Provider{Kind: aiprovider.KindAnthropicMessages}
	asked := &Call{Provider: direct, Request: &Request{
		Advisor: agentports.DefaultAdvisor("claude-fable-5-1"),
	}}
	assert.Equal(t, "claude-fable-5-1", advisorFor(asked).Model)
	assert.True(t, sendsAdvisorBeta(asked, advisorFor(asked)))

	advice := &ReasoningTrace{Blocks: []ProviderBlock{
		{Type: "advisor_tool_result", Model: "claude-opus-5-5"},
	}}
	held := &Call{Provider: direct, Request: &Request{Messages: advisedHistory(advice)}}
	assert.Nil(t, advisorFor(held), "advice in the history does not declare the advisor")
	assert.True(t, sendsAdvisorBeta(held, nil), "it only sends the beta that lets it be read")

	foreign := &ReasoningTrace{
		ProviderKind: string(aiprovider.KindOpenAIResponses),
		Blocks:       []ProviderBlock{{Type: "advisor_tool_result", Model: "claude-opus-5-5"}},
	}
	other := &Call{Provider: direct, Request: &Request{
		Messages: []Message{{Role: RoleAssistant, Reasoning: foreign}},
	}}
	assert.Nil(t, advisorFor(other))
	assert.False(t, sendsAdvisorBeta(other, nil), "another protocol's blocks are not replayed")

	plain := &Call{Provider: direct, Request: &Request{Messages: UserMessage("hi")}}
	assert.Nil(t, advisorFor(plain))
	assert.False(t, sendsAdvisorBeta(plain, nil))
}

func TestAdvisorFor_OnlyOnPlatformsThatServeIt(t *testing.T) {
	t.Parallel()

	advice := &ReasoningTrace{Blocks: []ProviderBlock{
		{Type: "advisor_tool_result", Model: "claude-opus-5-5"},
	}}
	for _, platform := range aiprovider.AllPlatforms() {
		provider := &aiprovider.Provider{
			Kind:     aiprovider.KindAnthropicMessages,
			Platform: platform,
		}
		asked := advisorFor(&Call{Provider: provider, Request: &Request{
			Advisor: agentports.DefaultAdvisor(""),
		}})
		heldCall := &Call{Provider: provider, Request: &Request{
			Messages: advisedHistory(advice),
		}}
		assert.Nil(t, advisorFor(heldCall), "%s: advice in the history never declares it", platform)
		held := sendsAdvisorBeta(heldCall, nil)
		if platform.ServesAdvisor() {
			assert.NotNil(t, asked, platform)
			assert.True(t, held, "%s reads the replayed advice under the beta", platform)
			continue
		}
		assert.Nil(t, asked, "%s refuses the advisor tool", platform)
		assert.False(t, held, "%s is sent the history without the advice instead", platform)
	}
}

func TestWithoutAdvice_DropsServerToolBlocksAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	advised := &ReasoningTrace{Blocks: []ProviderBlock{
		{Type: anthropicThinkingBlock, Text: "plan", Signature: "sig_a"},
		{Type: anthropicServerToolUseBlock, ID: "srvtoolu_1", Name: "advisor"},
		{Type: anthropicAdvisorResultBlock, ToolUseID: "srvtoolu_1"},
		{Type: anthropicTextBlock, Text: "Checking the shipment."},
	}}
	plain := &ReasoningTrace{Text: "thought"}
	messages := []Message{
		{Role: RoleUser, Content: "where is S1"},
		{Role: RoleAssistant, Content: "Checking the shipment.", Reasoning: advised},
		{Role: RoleAssistant, Content: "Done.", Reasoning: plain},
	}

	out := withoutAdvice(messages)
	require.Len(t, out, 3)
	assert.Equal(t, []ProviderBlock{
		{Type: anthropicThinkingBlock, Text: "plan", Signature: "sig_a"},
		{Type: anthropicTextBlock, Text: "Checking the shipment."},
	}, out[1].Reasoning.Blocks)
	assert.Same(t, plain, out[2].Reasoning)
	assert.Len(t, messages[1].Reasoning.Blocks, 4, "the caller's history is not changed")

	untouched := []Message{{Role: RoleAssistant, Reasoning: plain}}
	assert.Equal(t, &untouched[0], &withoutAdvice(untouched)[0], "nothing to drop, nothing copied")
}

func TestOpenAIAdapters_IgnoreTheAdvisor(t *testing.T) {
	t.Parallel()

	for _, adapter := range []Adapter{NewOpenAIChatAdapter(), NewOpenAIResponsesAdapter()} {
		server, captured := wireServer(t, "application/json", `{}`)
		request := advisorRequest()
		call := callFor(adapter.Kind(), server.URL, &request)
		_, _ = adapter.Complete(t.Context(), call)

		capture := captured()
		require.NotEmpty(t, capture.Body, adapter.Kind())
		sent, err := wireJSON.MarshalToString(capture)
		require.NoError(t, err)
		assert.NotContains(t, strings.ToLower(sent), "advisor", adapter.Kind())
	}
}

func TestAnthropicConversation_ReplacesStoredTextTheMessageNoLongerHolds(t *testing.T) {
	t.Parallel()

	blocks := replayProviderBlocks([]ProviderBlock{
		{Type: "text", Text: "Checking the shipment."},
		{Type: "server_tool_use", ID: "srvtoolu_1", Name: "advisor"},
		{
			Type: "advisor_tool_result", ToolUseID: "srvtoolu_1",
			Result: &conversation.ServerToolResult{Type: "advisor_result", Text: "Ask for the BOL."},
		},
		{Type: "text", Text: "It needs a BOL."},
	}, "The shipment needs a BOL.", true)

	require.Len(t, blocks, 3)
	require.NotNil(t, blocks[0].OfServerToolUse)
	require.NotNil(t, blocks[1].OfAdvisorToolResult)
	assert.Equal(t, "The shipment needs a BOL.", blocks[2].OfText.Text,
		"the stored text is replaced once rather than sent beside the message's")
}

func TestAnthropicConversation_KeepsStoredTextTheMessageMatches(t *testing.T) {
	t.Parallel()

	blocks := replayProviderBlocks([]ProviderBlock{
		{Type: "text", Text: "Checking the shipment. "},
		{Type: "server_tool_use", ID: "srvtoolu_1", Name: "advisor"},
		{Type: "text", Text: "It needs a BOL."},
	}, "Checking the shipment. It needs a BOL.", true)

	require.Len(t, blocks, 3)
	assert.Equal(t, "Checking the shipment. ", blocks[0].OfText.Text)
	require.NotNil(t, blocks[1].OfServerToolUse)
	assert.Equal(t, "It needs a BOL.", blocks[2].OfText.Text)
}
