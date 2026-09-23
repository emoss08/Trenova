package modeladapter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureServer records the request body an adapter sent and replies with a
// canned payload, so the wire translation can be asserted in both directions.
func captureServer(t *testing.T, reply any) (*httptest.Server, *map[string]any) {
	t.Helper()

	captured := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
		for k, v := range body {
			captured[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := sonic.Marshal(reply)
		_, _ = w.Write(encoded)
	}))
	t.Cleanup(server.Close)

	return server, &captured
}

func callFor(kind aiprovider.Kind, baseURL string, req *Request) *Call {
	return &Call{
		Provider: &aiprovider.Provider{
			Kind:                 kind,
			BaseURL:              baseURL,
			Model:                "test-model",
			AllowPrivateNetwork:  true,
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		},
		APIKey: "k",
		Client: &http.Client{Timeout: 5 * time.Second},
		Request: &Request{
			System:       req.System,
			Messages:     req.Messages,
			Tools:        req.Tools,
			OutputSchema: req.OutputSchema,
			SchemaName:   req.SchemaName,
			MaxTokens:    512,
		},
	}
}

func lookupTool() []ToolSpec {
	return []ToolSpec{{
		Name:        "lookup_shipment",
		Description: "Look up a shipment by number",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"number": map[string]any{"type": "string"}},
			"required":   []string{"number"},
		},
	}}
}

func TestAnthropicAdapter_ToolCallRoundTrip(t *testing.T) {
	t.Parallel()

	server, captured := captureServer(t, map[string]any{
		"model":       "test-model",
		"stop_reason": "tool_use",
		"content": []map[string]any{
			{"type": "text", "text": "Looking that up."},
			{
				"type":  "tool_use",
				"id":    "toolu_1",
				"name":  "lookup_shipment",
				"input": map[string]any{"number": "S12345"},
			},
		},
		"usage": map[string]any{"input_tokens": 5, "output_tokens": 3},
	})

	resp, err := NewAnthropicAdapter().Complete(t.Context(), callFor(
		aiprovider.KindAnthropicMessages,
		server.URL,
		&Request{
			System:   "sys",
			Messages: UserMessage("Where is S12345?"),
			Tools:    lookupTool(),
		},
	))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "toolu_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "lookup_shipment", resp.ToolCalls[0].Name)
	assert.Equal(t, "S12345", resp.ToolCalls[0].Arguments["number"])
	assert.Equal(t, "Looking that up.", resp.Text)

	// Anthropic takes the schema under input_schema, not parameters.
	tools, ok := (*captured)["tools"].([]any)
	require.True(t, ok, "tools should be sent")
	require.Len(t, tools, 1)
	assert.Contains(t, tools[0], "input_schema")
}

// A tool result is a user-role message carrying a tool_result block, not a role
// of its own, which is the shape most likely to be got wrong.
func TestAnthropicAdapter_SendsToolResultAsUserBlock(t *testing.T) {
	t.Parallel()

	server, captured := captureServer(t, map[string]any{
		"model":   "test-model",
		"content": []map[string]any{{"type": "text", "text": "It is in Memphis."}},
		"usage":   map[string]any{"input_tokens": 1, "output_tokens": 1},
	})

	_, err := NewAnthropicAdapter().Complete(t.Context(), callFor(
		aiprovider.KindAnthropicMessages,
		server.URL,
		&Request{
			Messages: []Message{
				{Role: RoleUser, Content: "Where is S12345?"},
				{Role: RoleAssistant, ToolCalls: []ToolCall{
					{
						ID:        "toolu_1",
						Name:      "lookup_shipment",
						Arguments: map[string]any{"number": "S12345"},
					},
				}},
				{
					Role:       RoleTool,
					ToolCallID: "toolu_1",
					ToolName:   "lookup_shipment",
					Content:    `{"city":"Memphis"}`,
				},
			},
			Tools: lookupTool(),
		},
	))
	require.NoError(t, err)

	messages, ok := (*captured)["messages"].([]any)
	require.True(t, ok)
	require.Len(t, messages, 3)

	last, _ := messages[2].(map[string]any)
	assert.Equal(t, "user", last["role"], "a tool result rides as a user message")

	content, _ := last["content"].([]any)
	require.Len(t, content, 1)
	block, _ := content[0].(map[string]any)
	assert.Equal(t, "tool_result", block["type"])
	assert.Equal(t, "toolu_1", block["tool_use_id"])
}

func TestOpenAIChatAdapter_ToolCallRoundTrip(t *testing.T) {
	t.Parallel()

	server, captured := captureServer(t, map[string]any{
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

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), callFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&Request{
			System:   "sys",
			Messages: UserMessage("Where is S12345?"),
			Tools:    lookupTool(),
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

	server, _ := captureServer(t, map[string]any{
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

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), callFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&Request{Messages: UserMessage("hi"), Tools: lookupTool()},
	))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.NotNil(t, resp.ToolCalls[0].Arguments, "arguments must never be nil")
	assert.Empty(t, resp.ToolCalls[0].Arguments)
}

func TestOllamaAdapter_ToolCallRoundTrip(t *testing.T) {
	t.Parallel()

	server, captured := captureServer(t, map[string]any{
		"model": "test-model",
		"message": map[string]any{
			"role":    "assistant",
			"content": "",
			"tool_calls": []map[string]any{{
				"function": map[string]any{
					"name": "lookup_shipment",
					// Ollama sends arguments as an object, unlike OpenAI's string.
					"arguments": map[string]any{"number": "S12345"},
				},
			}},
		},
		"prompt_eval_count": 6,
		"eval_count":        2,
	})

	resp, err := NewOllamaAdapter().Complete(t.Context(), callFor(
		aiprovider.KindOllama,
		server.URL,
		&Request{
			System:   "sys",
			Messages: UserMessage("Where is S12345?"),
			Tools:    lookupTool(),
		},
	))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "S12345", resp.ToolCalls[0].Arguments["number"])
	// The protocol supplies no call id, so one is synthesized for pairing.
	assert.NotEmpty(t, resp.ToolCalls[0].ID)

	assert.Contains(t, *captured, "tools")
	assert.NotContains(t, *captured, "format", "a schema must not ride alongside tools")
}

func TestOpenAIResponsesAdapter_ToolCallRoundTrip(t *testing.T) {
	t.Parallel()

	server, captured := captureServer(t, map[string]any{
		"model": "test-model",
		"output": []map[string]any{{
			"type":      "function_call",
			"call_id":   "fc_1",
			"name":      "lookup_shipment",
			"arguments": `{"number":"S12345"}`,
		}},
		"usage": map[string]any{"input_tokens": 3, "output_tokens": 1},
	})

	resp, err := NewOpenAIResponsesAdapter().Complete(t.Context(), callFor(
		aiprovider.KindOpenAIResponses,
		server.URL,
		&Request{
			System:   "sys",
			Messages: UserMessage("Where is S12345?"),
			Tools:    lookupTool(),
		},
	))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "fc_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "S12345", resp.ToolCalls[0].Arguments["number"])
	assert.Contains(t, *captured, "tools")
}

// This protocol carries tool traffic as input items rather than message roles.
func TestOpenAIResponsesAdapter_SendsToolResultAsFunctionCallOutput(t *testing.T) {
	t.Parallel()

	server, captured := captureServer(t, map[string]any{
		"model": "test-model",
		"output": []map[string]any{{
			"type":    "message",
			"content": []map[string]any{{"type": "output_text", "text": "Memphis."}},
		}},
	})

	_, err := NewOpenAIResponsesAdapter().Complete(t.Context(), callFor(
		aiprovider.KindOpenAIResponses,
		server.URL,
		&Request{
			Messages: []Message{
				{Role: RoleUser, Content: "Where is S12345?"},
				{Role: RoleAssistant, ToolCalls: []ToolCall{
					{
						ID:        "fc_1",
						Name:      "lookup_shipment",
						Arguments: map[string]any{"number": "S12345"},
					},
				}},
				{Role: RoleTool, ToolCallID: "fc_1", Content: `{"city":"Memphis"}`},
			},
			Tools: lookupTool(),
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

// Every protocol must survive a turn with no tools, since most turns have none.
func TestAdapters_OmitToolFieldsWhenNoToolsOffered(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    aiprovider.Kind
		adapter Adapter
		reply   any
	}{
		{
			"anthropic", aiprovider.KindAnthropicMessages, NewAnthropicAdapter(),
			map[string]any{
				"model":   "m",
				"content": []map[string]any{{"type": "text", "text": "hi"}},
			},
		},
		{
			"openai chat", aiprovider.KindOpenAIChat, NewOpenAIChatAdapter(),
			map[string]any{
				"model": "m",
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "content": "hi"}},
				},
			},
		},
		{
			"ollama", aiprovider.KindOllama, NewOllamaAdapter(),
			map[string]any{
				"model":   "m",
				"message": map[string]any{"role": "assistant", "content": "hi"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, captured := captureServer(t, tc.reply)

			resp, err := tc.adapter.Complete(t.Context(), callFor(tc.kind, server.URL, &Request{
				System:   "sys",
				Messages: UserMessage("hello"),
			}))
			require.NoError(t, err)

			assert.Equal(t, "hi", resp.Text)
			assert.Empty(t, resp.ToolCalls)
			assert.NotContains(t, *captured, "tools")
		})
	}
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
	server, captured := captureServer(t, map[string]any{
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

	gemini := callFor(aiprovider.KindOpenAIChat, server.URL, &Request{
		Messages: UserMessage("Where is S1?"), Tools: lookupTool(),
	})
	gemini.Provider.ID = pulid.MustNew("aip_")

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), gemini)
	require.NoError(t, err)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, signature, resp.ToolCalls[0].ProviderData)

	call := resp.ToolCalls[0]
	call.ProviderID = gemini.Provider.ID
	followUp := []Message{
		{Role: RoleUser, Content: "Where is S1?"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{call}},
		{
			Role:       RoleTool,
			ToolCallID: "call_1",
			ToolName:   "lookup_shipment",
			Content:    `{"status":"InTransit"}`,
		},
	}

	gemini.Request = &Request{Messages: followUp, Tools: lookupTool()}
	_, err = NewOpenAIChatAdapter().Complete(t.Context(), gemini)
	require.NoError(t, err)
	assert.Equal(t, signature, capturedToolCallExtra(t, *captured),
		"the provider that signed the call gets its signature back")

	other := callFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&Request{Messages: followUp, Tools: lookupTool()},
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

	history := []Message{
		{Role: RoleUser, Content: "Where is S1?"},
		{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{
				{ID: "call_1", Name: "lookup_shipment", Arguments: map[string]any{"number": "S1"}},
			},
		},
		{Role: RoleTool, ToolCallID: "call_1", ToolName: "lookup_shipment", Content: "{}"},
	}

	gemini := toChatMessages("", history, pulid.MustNew("aip_"), "gemini-3.8-flash")
	require.Len(t, gemini[1].ToolCalls, 1)
	assert.Equal(t, geminiUnsignedCall, gemini[1].ToolCalls[0].ExtraContent)

	other := toChatMessages("", history, pulid.MustNew("aip_"), "qwen3:32b")
	assert.Nil(t, other[1].ToolCalls[0].ExtraContent)
}
