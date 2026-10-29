package anthropicadapter

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toAnthropicMessages(messages []modeladapter.Message) []anthropic.BetaMessageParam {
	return anthropicConversation(messages, nil, false)
}

func TestToAnthropicMessages_ReplaysAnArgumentFreeCallWithItsInput(t *testing.T) {
	t.Parallel()

	for name, arguments := range map[string]map[string]any{
		"nil":   nil,
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			messages := toAnthropicMessages([]modeladapter.Message{
				{Role: modeladapter.RoleUser, Content: "what is ready to bill"},
				{
					Role: modeladapter.RoleAssistant,
					ToolCalls: []modeladapter.ToolCall{{
						ID:        "toolu_1",
						Name:      "list_billing_transfer_candidates",
						Arguments: arguments,
					}},
				},
				{Role: modeladapter.RoleTool, ToolCallID: "toolu_1", Content: "{}"},
			})

			encoded, err := modeladapter.RequestJSON.Marshal(messages[1])
			require.NoError(t, err)
			assert.Contains(t, string(encoded), `"input":{}`)
		})
	}
}

func TestSplitAnthropicContent_ReadsAMissingInputAsNoArguments(t *testing.T) {
	t.Parallel()

	_, calls := splitAnthropicContent(
		[]anthropicBlock{{Type: "tool_use", ID: "toolu_1", Name: "list_x"}},
	)

	require.Len(t, calls, 1)
	assert.NotNil(t, calls[0].Arguments)
	assert.Empty(t, calls[0].Arguments)
}

// messageShape is each message's role and the type of each of its blocks, with
// a tool result's call id, so a test can read the grouping at a glance.
func messageShape(t *testing.T, messages []modeladapter.Message) []string {
	t.Helper()

	encoded, err := modeladapter.RequestJSON.Marshal(toAnthropicMessages(messages))
	require.NoError(t, err)
	var decoded []struct {
		Role    string `json:"role"`
		Content []struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
		} `json:"content"`
	}
	require.NoError(t, modeladapter.RequestJSON.Unmarshal(encoded, &decoded))

	shape := make([]string, 0, len(decoded))
	for _, message := range decoded {
		line := message.Role + ":"
		for _, block := range message.Content {
			line += " " + block.Type
			if block.ToolUseID != "" {
				line += "(" + block.ToolUseID + ")"
			}
		}
		shape = append(shape, line)
	}

	return shape
}

func parallelCalls(ids ...string) modeladapter.Message {
	calls := make([]modeladapter.ToolCall, 0, len(ids))
	for _, id := range ids {
		calls = append(
			calls,
			modeladapter.ToolCall{
				ID:        id,
				Name:      "get_shipment",
				Arguments: map[string]any{"number": id},
			},
		)
	}

	return modeladapter.Message{Role: modeladapter.RoleAssistant, ToolCalls: calls}
}

/*
The results of the calls one turn made go back in one user message, in the
order they were made. Sent one per message, they teach the model to stop
calling tools in parallel.
*/
func TestToAnthropicMessages_SendsOneTurnsToolResultsTogether(t *testing.T) {
	t.Parallel()

	shape := messageShape(t, []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "where are S1, S2 and S3"},
		parallelCalls("toolu_1", "toolu_2", "toolu_3"),
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_1", Content: `{"status":"InTransit"}`},
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_2", Content: `{"status":"Delivered"}`},
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_3", Content: "not found", IsError: true},
	})

	assert.Equal(t, []string{
		"user: text",
		"assistant: tool_use tool_use tool_use",
		"user: tool_result(toolu_1) tool_result(toolu_2) tool_result(toolu_3)",
	}, shape)
}

func TestToAnthropicMessages_PutsTextThatFollowsResultsAfterThemInTheirMessage(t *testing.T) {
	t.Parallel()

	shape := messageShape(t, []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "where are S1 and S2"},
		parallelCalls("toolu_1", "toolu_2"),
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_1", Content: "{}"},
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_2", Content: "{}"},
		{Role: modeladapter.RoleUser, Content: "and only the late one"},
	})

	assert.Equal(t, []string{
		"user: text",
		"assistant: tool_use tool_use",
		"user: tool_result(toolu_1) tool_result(toolu_2) text",
	}, shape)
}

func TestToAnthropicMessages_KeepsEachTurnsResultsApart(t *testing.T) {
	t.Parallel()

	shape := messageShape(t, []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "check S1, then S2"},
		parallelCalls("toolu_1"),
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_1", Content: "{}"},
		parallelCalls("toolu_2", "toolu_3"),
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_2", Content: "{}"},
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_3", Content: "{}"},
		{Role: modeladapter.RoleAssistant, Content: "Both are moving."},
		{Role: modeladapter.RoleUser, Content: "thanks"},
		{Role: modeladapter.RoleUser, Content: "one more thing"},
	})

	assert.Equal(t, []string{
		"user: text",
		"assistant: tool_use",
		"user: tool_result(toolu_1)",
		"assistant: tool_use tool_use",
		"user: tool_result(toolu_2) tool_result(toolu_3)",
		"assistant: text",
		"user: text",
		"user: text",
	}, shape, "a person's own messages are left as they were sent")
}

func TestToAnthropicMessages_NeverFoldsAResultIntoAPersonsMessage(t *testing.T) {
	t.Parallel()

	shape := messageShape(t, []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "a message the history kept"},
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_orphan", Content: "{}"},
	})

	assert.Equal(t, []string{"user: text", "user: tool_result(toolu_orphan)"}, shape)
}

func TestToAnthropicMessages_KeepsEveryResultAheadOfTextBetweenThem(t *testing.T) {
	t.Parallel()

	shape := messageShape(t, []modeladapter.Message{
		{Role: modeladapter.RoleUser, Content: "where are S1 and S2"},
		parallelCalls("toolu_1", "toolu_2"),
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_1", Content: "{}"},
		{Role: modeladapter.RoleUser, Content: "a note that landed between the results"},
		{Role: modeladapter.RoleTool, ToolCallID: "toolu_2", Content: "{}"},
	})

	assert.Equal(t, []string{
		"user: text",
		"assistant: tool_use tool_use",
		"user: tool_result(toolu_1) tool_result(toolu_2) text",
	}, shape)
}
