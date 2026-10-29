package assistanttest

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
)

// LongConversation is a conversation of turns that each read a long listing,
// numbered as the thread stores them.
func LongConversation(turns int) []conversation.Message {
	history := make([]conversation.Message, 0, turns*4)
	for turn := range turns {
		callID := fmt.Sprintf("call_%d", turn)
		history = append(
			history,
			conversation.Message{
				Role:    conversation.RoleUser,
				Content: fmt.Sprintf("Which loads for customer %d?", turn),
			},
			conversation.Message{
				Role:      conversation.RoleAssistant,
				ToolCalls: []conversation.ToolCallRecord{{ID: callID, Name: "shipment_list"}},
			},
			conversation.Message{
				Role:       conversation.RoleTool,
				ToolCallID: callID,
				ToolName:   "shipment_list",
				Content:    strings.Repeat(`{"pro":"S-1001","status":"InTransit"},`, 800),
			},
			conversation.Message{
				Role: conversation.RoleAssistant,
				Content: fmt.Sprintf(
					"Customer %d has 40 loads. ",
					turn,
				) + strings.Repeat(
					"Most are on time. ",
					120,
				),
			},
		)
	}
	for idx := range history {
		history[idx].Sequence = idx
		history[idx].Kind = conversation.MessageKindMessage
	}

	return history
}
