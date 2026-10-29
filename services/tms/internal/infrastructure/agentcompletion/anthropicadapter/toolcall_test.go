package anthropicadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicAdapter_ToolCallRoundTrip(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.CaptureServer(t, map[string]any{
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

	resp, err := NewAnthropicAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages,
		server.URL,
		&modeladapter.Request{
			System:   "sys",
			Messages: modeladapter.UserMessage("Where is S12345?"),
			Tools:    modeladaptertest.LookupTool(),
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

	server, captured := modeladaptertest.CaptureServer(t, map[string]any{
		"model":   "test-model",
		"content": []map[string]any{{"type": "text", "text": "It is in Memphis."}},
		"usage":   map[string]any{"input_tokens": 1, "output_tokens": 1},
	})

	_, err := NewAnthropicAdapter().Complete(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages,
		server.URL,
		&modeladapter.Request{
			Messages: []modeladapter.Message{
				{Role: modeladapter.RoleUser, Content: "Where is S12345?"},
				{Role: modeladapter.RoleAssistant, ToolCalls: []modeladapter.ToolCall{
					{
						ID:        "toolu_1",
						Name:      "lookup_shipment",
						Arguments: map[string]any{"number": "S12345"},
					},
				}},
				{
					Role:       modeladapter.RoleTool,
					ToolCallID: "toolu_1",
					ToolName:   "lookup_shipment",
					Content:    `{"city":"Memphis"}`,
				},
			},
			Tools: modeladaptertest.LookupTool(),
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
