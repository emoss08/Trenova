package openaiadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func finalAnswerRequest(none bool) *modeladapter.Request {
	return &modeladapter.Request{
		System: "sys",
		Messages: []modeladapter.Message{
			{Role: modeladapter.RoleUser, Content: "Where is S12345?"},
			{Role: modeladapter.RoleAssistant, ToolCalls: []modeladapter.ToolCall{{
				ID:        "call_1",
				Name:      "lookup_shipment",
				Arguments: map[string]any{"number": "S12345"},
			}}},
			{
				Role:       modeladapter.RoleTool,
				ToolCallID: "call_1",
				ToolName:   "lookup_shipment",
				Content:    "Memphis",
			},
			{Role: modeladapter.RoleUser, Content: "Answer now from what you have."},
		},
		Tools:          modeladaptertest.LookupTool(),
		ToolChoiceNone: none,
	}
}

func TestOpenAIResponsesAdapter_FinalAnswerSendsToolChoiceNone(t *testing.T) {
	t.Parallel()

	reply := map[string]any{
		"model":  "test-model",
		"status": "completed",
		"output": []map[string]any{{
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]any{{"type": "output_text", "text": "In Memphis."}},
		}},
		"usage": map[string]any{"input_tokens": 1, "output_tokens": 1},
	}

	server, captured := modeladaptertest.CaptureServer(t, reply)
	_, err := NewOpenAIResponsesAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIResponses, server.URL, finalAnswerRequest(true),
	))
	require.NoError(t, err)
	assert.Equal(t, "none", (*captured)["tool_choice"])
	tools, ok := (*captured)["tools"].([]any)
	require.True(t, ok, "the tools stay declared")
	assert.Len(t, tools, 1)

	server, captured = modeladaptertest.CaptureServer(t, reply)
	_, err = NewOpenAIResponsesAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIResponses, server.URL, finalAnswerRequest(false),
	))
	require.NoError(t, err)
	assert.NotContains(t, *captured, "tool_choice")
}

func TestOpenAIChatAdapter_FinalAnswerSendsToolChoiceNone(t *testing.T) {
	t.Parallel()

	reply := map[string]any{
		"model": "test-model",
		"choices": []map[string]any{{
			"message":       map[string]any{"role": "assistant", "content": "In Memphis."},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1},
	}

	server, captured := modeladaptertest.CaptureServer(t, reply)
	_, err := NewOpenAIChatAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat, server.URL, finalAnswerRequest(true),
	))
	require.NoError(t, err)
	assert.Equal(t, "none", (*captured)["tool_choice"])
	tools, ok := (*captured)["tools"].([]any)
	require.True(t, ok, "the tools stay declared")
	assert.Len(t, tools, 1)

	server, captured = modeladaptertest.CaptureServer(t, reply)
	_, err = NewOpenAIChatAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat, server.URL, finalAnswerRequest(false),
	))
	require.NoError(t, err)
	assert.NotContains(t, *captured, "tool_choice")
}

func TestToolChoiceWire_NeedsToolsToForbid(t *testing.T) {
	t.Parallel()

	assert.Empty(t, toolChoiceWire(&modeladapter.Request{ToolChoiceNone: true}))
	assert.Equal(
		t,
		"none",
		toolChoiceWire(
			&modeladapter.Request{ToolChoiceNone: true, Tools: modeladaptertest.LookupTool()},
		),
	)
	assert.Empty(t, toolChoiceWire(&modeladapter.Request{Tools: modeladaptertest.LookupTool()}))
}
