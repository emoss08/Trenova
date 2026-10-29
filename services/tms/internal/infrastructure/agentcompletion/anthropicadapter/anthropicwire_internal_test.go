package anthropicadapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/stretchr/testify/require"
)

const updateWireEnv = "UPDATE_ANTHROPIC_WIRE"

const (
	wireStreamReply = "event: message_start\n" +
		`data: {"type":"message_start","message":{"model":"m","usage":{"input_tokens":1}}}` + "\n\n" +
		"event: message_stop\n" +
		`data: {"type":"message_stop"}` + "\n\n"
	wireBlockingReply = `{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"ok"}],` +
		`"usage":{"input_tokens":1,"output_tokens":1}}`
)

func wireHistory() []modeladapter.Message {
	earlier := &modeladapter.ReasoningTrace{Text: "earlier turn", Signature: "sig_earlier"}
	current := &modeladapter.ReasoningTrace{
		Text:      "this turn",
		Signature: "sig_current",
		Redacted:  []string{"redacted_payload"},
	}

	return []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "Where is S1?"},
		{
			Role:      modeladapter.RoleAssistant,
			Content:   "Looking it up.",
			Reasoning: earlier,
			ToolCalls: []modeladapter.ToolCall{{
				ID:        "call_1",
				Name:      "get_shipment",
				Arguments: map[string]any{"number": "S1", "include": []any{"stops", "moves"}},
			}},
		},
		{
			Role:       modeladapter.RoleTool,
			ToolCallID: "call_1",
			ToolName:   "get_shipment",
			Content:    `{"status":"InTransit"}`,
		},
		{Role: modeladapter.RoleAssistant, Content: "In Dallas.", Reasoning: earlier},
		{Role: modeladapter.RoleUser, Content: "And S2 and S3?"},
		{
			Role:      modeladapter.RoleAssistant,
			Reasoning: current,
			ToolCalls: []modeladapter.ToolCall{
				{ID: "call_2", Name: "get_shipment", Arguments: map[string]any{"number": "S2"}},
				{ID: "call_3", Name: "get_shipment", Arguments: nil},
			},
		},
		{
			Role:       modeladapter.RoleTool,
			ToolCallID: "call_2",
			ToolName:   "get_shipment",
			Content:    `{"status":"Delivered"}`,
		},
		{
			Role:       modeladapter.RoleTool,
			ToolCallID: "call_3",
			ToolName:   "get_shipment",
			Content:    "not found",
			IsError:    true,
		},
	}
}

func advisedWireHistory() []modeladapter.Message {
	earlier := &modeladapter.ReasoningTrace{
		Text:         "earlier turn",
		Signature:    "sig_earlier",
		ProviderKind: string(aiprovider.KindAnthropicMessages),
		Blocks: []modeladapter.ProviderBlock{
			{Type: "thinking", Text: "earlier turn", Signature: "sig_earlier"},
			{Type: "server_tool_use", ID: "srvtoolu_1", Name: "advisor", Input: map[string]any{}},
			{
				Type: "advisor_tool_result", ToolUseID: "srvtoolu_1", Model: "claude-opus-5-5",
				Result: &conversation.ServerToolResult{
					Type: "advisor_redacted_result", EncryptedContent: "enc_earlier",
					StopReason: "end_turn",
				},
			},
			{Type: "text", Text: "Looking it up."},
		},
	}
	current := &modeladapter.ReasoningTrace{
		Text:         "this turn",
		Signature:    "sig_current",
		ProviderKind: string(aiprovider.KindAnthropicMessages),
		Blocks: []modeladapter.ProviderBlock{
			{Type: "thinking", Text: "this turn", Signature: "sig_current"},
			{Type: "server_tool_use", ID: "srvtoolu_2", Name: "advisor", Input: map[string]any{}},
			{
				Type: "advisor_tool_result", ToolUseID: "srvtoolu_2", Model: "claude-opus-5-5",
				Result: &conversation.ServerToolResult{
					Type: "advisor_tool_result_error", ErrorCode: "max_uses_exceeded",
				},
			},
		},
	}
	history := wireHistory()
	history[1].Reasoning = earlier
	history[5].Reasoning = current

	return history
}

func wireSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"stops": map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": 10,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"city":   map[string]any{"type": "string", "minLength": 1},
						"weight": map[string]any{"type": "number", "minimum": 0},
					},
					"required":             []any{"city"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []any{"stops"},
		"additionalProperties": false,
	}
}

type wireCase struct {
	name    string
	model   string
	effort  aireasoning.ReasoningEffort
	style   aireasoning.ThinkingStyle
	stream  bool
	request modeladapter.Request
}

func wireCases() []wireCase {
	agent := modeladapter.Request{
		System:       "Rules that never change.\n\nToday is Friday.",
		SystemStable: len("Rules that never change."),
		Messages:     wireHistory(),
		Tools:        modeladaptertest.WireTools(),
	}
	structured := modeladapter.Request{
		System:       "Extract the stops.",
		Messages:     modeladapter.UserMessage("Pick up in Dallas, deliver in Austin."),
		OutputSchema: wireSchema(),
		SchemaName:   "stops",
	}
	structuredWithTools := structured
	structuredWithTools.Tools = modeladaptertest.WireTools()[:1]
	bare := modeladapter.Request{Messages: modeladapter.UserMessage("hi")}
	finalAnswer := agent
	finalAnswer.Messages = append(
		append([]modeladapter.Message(nil), agent.Messages...),
		modeladapter.Message{
			Role:    modeladapter.RoleUser,
			Content: "Answer now from what you have; do not call a tool.",
		},
	)
	finalAnswer.ToolChoiceNone = true
	deferred := modeladaptertest.DeferredWireRequest()
	advised := agent
	advised.Messages = advisedWireHistory()
	advised.Advisor = &modeladapter.AdvisorSpec{Model: "claude-opus-5-5", MaxUses: 2}

	return []wireCase{
		{
			name:    "advisor_haiku_5_5_low_stream",
			model:   "claude-haiku-5-5",
			effort:  aireasoning.ReasoningLow,
			stream:  true,
			request: advised,
		},
		{
			name:    "agent_opus_5_5_off_stream",
			model:   "claude-opus-5-5",
			effort:  aireasoning.ReasoningOff,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_opus_5_5_high_stream",
			model:   "claude-opus-5-5",
			effort:  aireasoning.ReasoningHigh,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_sonnet_5_5_none_stream",
			model:   "claude-sonnet-5-5",
			effort:  aireasoning.ReasoningNone,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_fable_5_1_medium_stream",
			model:   "claude-fable-5-1",
			effort:  aireasoning.ReasoningMedium,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_opus_5_off_stream",
			model:   "claude-opus-5",
			effort:  aireasoning.ReasoningOff,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_haiku_5_5_none_stream",
			model:   "claude-haiku-5-5",
			effort:  aireasoning.ReasoningNone,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_opus_4_8_low_stream",
			model:   "claude-opus-4-8",
			effort:  aireasoning.ReasoningLow,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_haiku_4_5_medium_stream",
			model:   "claude-haiku-4-5",
			effort:  aireasoning.ReasoningMedium,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_bedrock_sonnet_4_6_off_stream",
			model:   "us.anthropic.claude-sonnet-4-6-v1:0",
			effort:  aireasoning.ReasoningOff,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_declared_effort_none_stream",
			model:   "house-model",
			effort:  aireasoning.ReasoningNone,
			style:   aireasoning.ThinkingStyleEffort,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_budget_style_high_stream",
			model:   "claude-opus-5-5",
			effort:  aireasoning.ReasoningHigh,
			style:   aireasoning.ThinkingStyleBudget,
			stream:  true,
			request: agent,
		},
		{
			name:    "agent_opus_5_5_off_blocking",
			model:   "claude-opus-5-5",
			effort:  aireasoning.ReasoningOff,
			request: agent,
		},
		{
			name:    "structured_opus_4_8_off_blocking",
			model:   "claude-opus-4-8",
			effort:  aireasoning.ReasoningOff,
			request: structured,
		},
		{
			name:    "structured_with_tools_blocking",
			model:   "claude-opus-4-8",
			effort:  aireasoning.ReasoningOff,
			request: structuredWithTools,
		},
		{
			name:    "structured_opus_5_5_none_blocking",
			model:   "claude-opus-5-5",
			effort:  aireasoning.ReasoningNone,
			request: structured,
		},
		{
			name:    "final_answer_haiku_5_5_medium_stream",
			model:   "claude-haiku-5-5",
			effort:  aireasoning.ReasoningMedium,
			stream:  true,
			request: finalAnswer,
		},
		{
			name:    "final_answer_opus_5_5_off_stream",
			model:   "claude-opus-5-5",
			effort:  aireasoning.ReasoningOff,
			stream:  true,
			request: finalAnswer,
		},
		{
			name:    "bare_unknown_model_blocking",
			model:   "test-model",
			effort:  aireasoning.ReasoningOff,
			request: bare,
		},
		{
			name:    "deferred_haiku_5_5_medium_stream",
			model:   "claude-haiku-5-5",
			effort:  aireasoning.ReasoningMedium,
			stream:  true,
			request: deferred,
		},
	}
}

func runWireCase(t *testing.T, tc wireCase) modeladaptertest.WireCapture {
	t.Helper()

	contentType, reply := "application/json", wireBlockingReply
	if tc.stream {
		contentType, reply = "text/event-stream", wireStreamReply
	}
	server, captured := modeladaptertest.WireServer(t, contentType, reply)

	request := tc.request
	call := modeladaptertest.ReasoningCall(
		aiprovider.KindAnthropicMessages,
		server.URL,
		tc.effort,
		&request,
	)
	call.Provider.Model = tc.model
	call.Provider.ThinkingStyle = tc.style
	call.APIKey = "sk-test"

	if tc.stream {
		_, _ = modeladaptertest.StreamWith(t, NewAnthropicAdapter(), call)
	} else {
		_, err := NewAnthropicAdapter().Complete(t.Context(), call)
		require.NoError(t, err)
	}

	return captured()
}

func encodeWire(t *testing.T, capture modeladaptertest.WireCapture) []byte {
	t.Helper()

	encoded, err := modeladaptertest.WireJSON.MarshalIndent(capture, "", "  ")
	require.NoError(t, err)

	return append(encoded, '\n')
}

/*
What goes on the wire to Anthropic, pinned case by case.

Each case is a request this system actually makes: an agent turn mid tool loop
on each family of model and thinking setting, and a structured extraction. The
golden files are the exact method, path, headers and body. A change to how the
adapter builds a request shows up here as a diff to read, rather than as a 400
from the API or a cache that stops hitting in production.

Run with UPDATE_ANTHROPIC_WIRE=1 to rewrite the files after a deliberate change.
*/
func TestAnthropicAdapter_WireMatchesGolden(t *testing.T) {
	t.Parallel()

	update := os.Getenv(updateWireEnv) != ""
	for _, tc := range wireCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := encodeWire(t, runWireCase(t, tc))
			path := filepath.Join("testdata", "anthropic_wire", tc.name+".json")
			if update {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, got, 0o600))
				return
			}

			want, err := os.ReadFile(path)
			require.NoError(t, err, "golden file missing; run with %s=1", updateWireEnv)
			require.JSONEq(t, string(want), string(got))
		})
	}
}

func TestAnthropicAdapter_FinalAnswerKeepsToolsAndForbidsCalls(t *testing.T) {
	t.Parallel()

	var final, turn wireCase
	for _, tc := range wireCases() {
		switch tc.name {
		case "final_answer_opus_5_5_off_stream":
			final = tc
		case "agent_opus_5_5_off_stream":
			turn = tc
		}
	}

	finalBody := runWireCase(t, final).Body
	turnBody := runWireCase(t, turn).Body
	require.Equal(t, map[string]any{"type": "none"}, finalBody["tool_choice"])
	require.NotContains(t, turnBody, "tool_choice")
	require.Equal(t, turnBody["tools"], finalBody["tools"])
	require.Equal(t, turnBody["system"], finalBody["system"])
}

func TestAnthropicAdapter_DefersToolsAndLoadsThemByReference(t *testing.T) {
	t.Parallel()

	var tc wireCase
	for _, candidate := range wireCases() {
		if candidate.name == "deferred_haiku_5_5_medium_stream" {
			tc = candidate
		}
	}
	body := runWireCase(t, tc).Body

	tools, ok := body["tools"].([]any)
	require.True(t, ok)
	byName := make(map[string]map[string]any, len(tools))
	for _, raw := range tools {
		tool, isTool := raw.(map[string]any)
		require.True(t, isTool)
		byName[tool["name"].(string)] = tool
	}
	require.Len(t, byName, 4, "every declared tool is sent, deferred or not")
	require.Equal(t, true, byName["list_trailers"]["defer_loading"])
	require.Equal(t, true, byName["list_workers"]["defer_loading"])
	require.NotContains(t, byName["get_shipment"], "defer_loading")
	require.NotContains(t, byName["list_trailers"], "cache_control",
		"a deferred tool cannot carry the cache mark")
	require.Contains(t, byName["find_tools"], "cache_control",
		"the mark is on the last tool sent in full")
	require.NotContains(t, byName["get_shipment"], "cache_control")

	messages := body["messages"].([]any)
	blocks := messages[len(messages)-1].(map[string]any)["content"].([]any)
	require.Len(t, blocks, 2, "the result, then its text")
	result := blocks[0].(map[string]any)
	require.Equal(t, "tool_result", result["type"])
	require.Equal(
		t,
		[]any{map[string]any{"type": "tool_reference", "tool_name": "list_trailers"}},
		result["content"],
		"only the references: the API refuses one beside text, and only a declared deferred tool is named",
	)
	note := blocks[1].(map[string]any)
	require.Equal(t, "text", note["type"])
	require.Equal(
		t,
		"find_tools (call_find):\nThese tools are now callable:\n- list_trailers: List trailers",
		note["text"],
	)
}

func TestAnthropicConversation_PutsLoadedToolsTextAfterEveryResult(t *testing.T) {
	t.Parallel()

	out := anthropicConversation([]modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "Which trailers and workers are due?"},
		{
			Role: modeladapter.RoleAssistant,
			ToolCalls: []modeladapter.ToolCall{
				{ID: "call_find", Name: "find_tools"},
				{ID: "call_ship", Name: "get_shipment"},
				{ID: "call_quiet", Name: "find_tools"},
			},
		},
		{
			Role: modeladapter.RoleTool, ToolCallID: "call_find", ToolName: "find_tools",
			Content:        "These tools are now callable:\n- list_trailers: List trailers",
			ToolReferences: []string{"list_trailers"},
		},
		{
			Role:       modeladapter.RoleTool,
			ToolCallID: "call_ship",
			ToolName:   "get_shipment",
			Content:    "S-1",
		},
		{
			Role: modeladapter.RoleTool, ToolCallID: "call_quiet", ToolName: "find_tools",
			ToolReferences: []string{"list_workers"},
		},
	}, nil, false)

	blocks := out[len(out)-1].Content
	require.Len(t, blocks, 4)
	require.Equal(t, "call_find", blocks[0].OfToolResult.ToolUseID)
	require.Len(t, blocks[0].OfToolResult.Content, 1)
	require.NotNil(t, blocks[0].OfToolResult.Content[0].OfToolReference)
	require.Equal(t, "call_ship", blocks[1].OfToolResult.ToolUseID)
	require.Equal(t, "S-1", blocks[1].OfToolResult.Content[0].OfText.Text)
	require.Equal(t, "call_quiet", blocks[2].OfToolResult.ToolUseID)
	require.Len(t, blocks[2].OfToolResult.Content, 1)
	require.Equal(t, "list_workers", blocks[2].OfToolResult.Content[0].OfToolReference.ToolName)
	require.Contains(t, blocks[3].OfText.Text, "find_tools (call_find):")
}

func TestAnthropicConversation_KeepsAnErrorResultsTextInside(t *testing.T) {
	t.Parallel()

	out := anthropicConversation([]modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "find trailers"},
		{
			Role:      modeladapter.RoleAssistant,
			ToolCalls: []modeladapter.ToolCall{{ID: "call_find", Name: "find_tools"}},
		},
		{
			Role: modeladapter.RoleTool, ToolCallID: "call_find", ToolName: "find_tools", IsError: true,
			Content: "find_tools failed", ToolReferences: []string{"list_trailers"},
		},
	}, nil, false)

	blocks := out[len(out)-1].Content
	require.Len(t, blocks, 1)
	content := blocks[0].OfToolResult.Content
	require.Len(t, content, 1)
	require.Equal(t, "find_tools failed", content[0].OfText.Text)
}

func TestAnthropicAdapter_OnFoundrySendsWhatIsLoaded(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.WireServer(t, "text/event-stream", wireStreamReply)
	request := modeladaptertest.DeferredWireRequest()
	call := modeladaptertest.ReasoningCall(
		aiprovider.KindAnthropicMessages,
		server.URL,
		aireasoning.ReasoningOff,
		&request,
	)
	call.Provider.Model = "claude-haiku-5-5"
	call.Provider.Platform = aiplatform.PlatformFoundry
	call.APIKey = "sk-test"
	_, _ = modeladaptertest.StreamWith(t, NewAnthropicAdapter(), call)

	body := captured().Body
	names := make([]string, 0, 4)
	for _, raw := range body["tools"].([]any) {
		tool := raw.(map[string]any)
		require.NotContains(t, tool, "defer_loading")
		names = append(names, tool["name"].(string))
	}
	require.ElementsMatch(t, []string{"get_shipment", "find_tools", "list_trailers"}, names,
		"a loaded tool is sent, an unloaded deferred one is not")
	messages := body["messages"].([]any)
	result := messages[len(messages)-1].(map[string]any)["content"].([]any)[0].(map[string]any)
	require.Len(t, result["content"].([]any), 1, "no reference where tools are not deferred")
}
