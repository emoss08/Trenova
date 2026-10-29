package runtimetest

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
)

func ReadTurn(id, tool, content string) []conversation.Message {
	return []conversation.Message{
		{Role: conversation.RoleUser, Content: "Look it up."},
		{
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: id, Name: tool}},
		},
		{Role: conversation.RoleTool, ToolCallID: id, ToolName: tool, Content: content},
		{Role: conversation.RoleAssistant, Content: "Here it is."},
	}
}

func QuietTurns(n int) []conversation.Message {
	out := make([]conversation.Message, 0, 2*n)
	for i := range n {
		out = append(out,
			conversation.Message{Role: conversation.RoleUser, Content: fmt.Sprintf("Turn %d", i)},
			conversation.Message{Role: conversation.RoleAssistant, Content: "Noted."},
		)
	}

	return out
}
