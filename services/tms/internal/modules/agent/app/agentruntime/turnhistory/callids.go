package turnhistory

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
)

// UsedCallIDs is every tool call id the conversation already holds.
func UsedCallIDs(history []conversation.Message) map[string]struct{} {
	used := make(map[string]struct{})
	for idx := range history {
		for _, call := range history[idx].ToolCalls {
			if call.ID != "" {
				used[call.ID] = struct{}{}
			}
		}
	}

	return used
}
