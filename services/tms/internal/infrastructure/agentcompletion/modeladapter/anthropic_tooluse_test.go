package modeladapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToAnthropicMessages_ReplaysAnArgumentFreeCallWithItsInput(t *testing.T) {
	t.Parallel()

	for name, arguments := range map[string]map[string]any{
		"nil":   nil,
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			messages := toAnthropicMessages([]Message{
				{Role: RoleUser, Content: "what is ready to bill"},
				{
					Role: RoleAssistant,
					ToolCalls: []ToolCall{{
						ID:        "toolu_1",
						Name:      "list_billing_transfer_candidates",
						Arguments: arguments,
					}},
				},
				{Role: RoleTool, ToolCallID: "toolu_1", Content: "{}"},
			})

			encoded, err := requestJSON.Marshal(messages[1])
			require.NoError(t, err)
			assert.Contains(t, string(encoded), `"input":{}`)
		})
	}
}

func TestSplitAnthropicContent_ReadsAMissingInputAsNoArguments(t *testing.T) {
	t.Parallel()

	_, calls := splitAnthropicContent([]anthropicBlock{{Type: "tool_use", ID: "toolu_1", Name: "list_x"}})

	require.Len(t, calls, 1)
	assert.NotNil(t, calls[0].Arguments)
	assert.Empty(t, calls[0].Arguments)
}
