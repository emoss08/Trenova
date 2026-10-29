package openaiadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatAdapter_ToolCallRoundTrip(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.CaptureServer(t, map[string]any{
		"model": "test-model",
		"choices": []map[string]any{{
			"finish_reason": "tool_calls",
			"message": map[string]any{
				"role":    "assistant",
				"content": "",
				"tool_calls": []map[string]any{{
					"id":   "call_1",
					"type": "function",
					"function": map[string]any{
						"name": "lookup_shipment",
						// This protocol encodes arguments as a JSON string.
						"arguments": `{"number":"S12345"}`,
					},
				}},
			},
		}},
		"usage": map[string]any{"prompt_tokens": 4, "completion_tokens": 2},
	})

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{
			System:   "sys",
			Messages: modeladapter.UserMessage("Where is S12345?"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "S12345", resp.ToolCalls[0].Arguments["number"])

	// Offering tools and demanding a schema at once leaves the model no way to
	// express a call, so response_format must be omitted.
	assert.NotContains(t, *captured, "response_format")
}

// A small model sometimes emits arguments that do not parse. The turn should
// survive so the tool can report a clear validation error the model can recover
// from.
func TestOpenAIChatAdapter_ToleratesUnparseableArguments(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.CaptureServer(t, map[string]any{
		"model": "test-model",
		"choices": []map[string]any{{
			"finish_reason": "tool_calls",
			"message": map[string]any{
				"role": "assistant",
				"tool_calls": []map[string]any{{
					"id":       "call_1",
					"type":     "function",
					"function": map[string]any{"name": "lookup_shipment", "arguments": "{not json"},
				}},
			},
		}},
	})

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{
			Messages: modeladapter.UserMessage("hi"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.NotNil(t, resp.ToolCalls[0].Arguments, "arguments must never be nil")
	assert.Empty(t, resp.ToolCalls[0].Arguments)
}

func TestOpenAIResponsesAdapter_ToolCallRoundTrip(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.CaptureServer(t, map[string]any{
		"model": "test-model",
		"output": []map[string]any{{
			"type":      "function_call",
			"call_id":   "fc_1",
			"name":      "lookup_shipment",
			"arguments": `{"number":"S12345"}`,
		}},
		"usage": map[string]any{"input_tokens": 3, "output_tokens": 1},
	})

	resp, err := NewOpenAIResponsesAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIResponses,
		server.URL,
		&modeladapter.Request{
			System:   "sys",
			Messages: modeladapter.UserMessage("Where is S12345?"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "fc_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "S12345", resp.ToolCalls[0].Arguments["number"])
	require.Contains(t, *captured, "tools")
	tools, ok := (*captured)["tools"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, tools)
	sent, ok := tools[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, sent["strict"],
		"left out, the Responses API's default made every optional parameter required")
}

// This protocol carries tool traffic as input items rather than message roles.
func TestOpenAIResponsesAdapter_SendsToolResultAsFunctionCallOutput(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.CaptureServer(t, map[string]any{
		"model": "test-model",
		"output": []map[string]any{{
			"type":    "message",
			"content": []map[string]any{{"type": "output_text", "text": "Memphis."}},
		}},
	})

	_, err := NewOpenAIResponsesAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIResponses,
		server.URL,
		&modeladapter.Request{
			Messages: []modeladapter.Message{
				{Role: modeladapter.RoleUser, Content: "Where is S12345?"},
				{Role: modeladapter.RoleAssistant, ToolCalls: []modeladapter.ToolCall{
					{
						ID:        "fc_1",
						Name:      "lookup_shipment",
						Arguments: map[string]any{"number": "S12345"},
					},
				}},
				{Role: modeladapter.RoleTool, ToolCallID: "fc_1", Content: `{"city":"Memphis"}`},
			},
			Tools: modeladaptertest.LookupTool(),
		},
	))
	require.NoError(t, err)

	input, ok := (*captured)["input"].([]any)
	require.True(t, ok)

	var sawCall, sawOutput bool
	for _, raw := range input {
		item, _ := raw.(map[string]any)
		switch item["type"] {
		case "function_call":
			sawCall = true
			assert.Equal(t, "fc_1", item["call_id"])
		case "function_call_output":
			sawOutput = true
			assert.Equal(t, "fc_1", item["call_id"])
		}
	}
	assert.True(t, sawCall, "the assistant's call must be replayed")
	assert.True(t, sawOutput, "the tool result must ride as function_call_output")
}

// Gemini 3 signs each function call with a thought signature that the
// OpenAI-compatible protocol carries under extra_content, and it rejects the
// follow-up request outright when the call is replayed without it. The
// signature is opaque and belongs to the provider that produced it: it is
// kept on the call, sent back to that provider, and never sent to another,
// since a strict endpoint refuses a field it does not know.
func TestOpenAIChatAdapter_ReplaysProviderDataOnlyToTheProviderThatMadeTheCall(t *testing.T) {
	t.Parallel()

	signature := map[string]any{"google": map[string]any{"thought_signature": "sig-1"}}
	server, captured := modeladaptertest.CaptureServer(t, map[string]any{
		"model": "gemini-3.8-flash",
		"choices": []map[string]any{{
			"finish_reason": "tool_calls",
			"message": map[string]any{
				"role":    "assistant",
				"content": "",
				"tool_calls": []map[string]any{{
					"id":   "call_1",
					"type": "function",
					"function": map[string]any{
						"name":      "lookup_shipment",
						"arguments": `{"number":"S1"}`,
					},
					"extra_content": signature,
				}},
			},
		}},
	})

	gemini := modeladaptertest.CallFor(aiprovider.KindOpenAIChat, server.URL, &modeladapter.Request{
		Messages: modeladapter.UserMessage("Where is S1?"), Tools: modeladaptertest.LookupTool(),
	})
	gemini.Provider.ID = pulid.MustNew("aip_")

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), gemini)
	require.NoError(t, err)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, signature, resp.ToolCalls[0].ProviderData)

	call := resp.ToolCalls[0]
	call.ProviderID = gemini.Provider.ID
	followUp := []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "Where is S1?"},
		{Role: modeladapter.RoleAssistant, ToolCalls: []modeladapter.ToolCall{call}},
		{
			Role:       modeladapter.RoleTool,
			ToolCallID: "call_1",
			ToolName:   "lookup_shipment",
			Content:    `{"status":"InTransit"}`,
		},
	}

	gemini.Request = &modeladapter.Request{Messages: followUp, Tools: modeladaptertest.LookupTool()}
	_, err = NewOpenAIChatAdapter().Complete(t.Context(), gemini)
	require.NoError(t, err)
	assert.Equal(t, signature, capturedToolCallExtra(t, *captured),
		"the provider that signed the call gets its signature back")

	other := modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: followUp, Tools: modeladaptertest.LookupTool()},
	)
	other.Provider.ID = pulid.MustNew("aip_")
	_, err = NewOpenAIChatAdapter().Complete(t.Context(), other)
	require.NoError(t, err)
	assert.Nil(t, capturedToolCallExtra(t, *captured),
		"another provider is not sent a field only the first understands")
}

func capturedToolCallExtra(t *testing.T, body map[string]any) any {
	t.Helper()

	messages, ok := body["messages"].([]any)
	require.True(t, ok)
	for _, raw := range messages {
		message, isMap := raw.(map[string]any)
		if !isMap || message["role"] != "assistant" {
			continue
		}
		calls, hasCalls := message["tool_calls"].([]any)
		if !hasCalls || len(calls) == 0 {
			continue
		}
		first, isCall := calls[0].(map[string]any)
		require.True(t, isCall)

		return first["extra_content"]
	}

	return nil
}

// A call Gemini did not sign — made by another provider before the thread
// switched model, or stored before signatures were kept — is sent with
// Google's documented bypass value, so an old thread is not refused on its
// first turn after the switch. Any other model gets nothing added.
func TestOpenAIChatAdapter_SendsGeminiTheBypassForAnUnsignedCall(t *testing.T) {
	t.Parallel()

	history := []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "Where is S1?"},
		{
			Role: modeladapter.RoleAssistant,
			ToolCalls: []modeladapter.ToolCall{
				{ID: "call_1", Name: "lookup_shipment", Arguments: map[string]any{"number": "S1"}},
			},
		},
		{
			Role:       modeladapter.RoleTool,
			ToolCallID: "call_1",
			ToolName:   "lookup_shipment",
			Content:    "{}",
		},
	}

	gemini := toChatMessages("", history, pulid.MustNew("aip_"), "gemini-3.8-flash")
	require.Len(t, gemini[1].ToolCalls, 1)
	assert.Equal(t, geminiUnsignedCall, gemini[1].ToolCalls[0].ExtraContent)

	other := toChatMessages("", history, pulid.MustNew("aip_"), "qwen3:32b")
	assert.Nil(t, other[1].ToolCalls[0].ExtraContent)
}
