package turnhistory

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
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

// DistinctCallIDs gives every call in a completion an id no other call in the
// conversation has.
//
// Providers that return no id get one synthesized from the call's position,
// call_0 in every completion, so the same id named a different call on every
// turn. The artifacts a call produces are keyed on it, the transcript pairs
// results with calls by it, and a provider handed a history with two calls
// under one id pairs the wrong result with the wrong call. Checking the id
// against the conversation is not enough: the conversation the turn replays
// is only its most recent messages, and an artifact keyed on a call_0 from
// before them was overwritten by the next call_0. So a synthesized id is
// always replaced. A provider's own unique ids are kept as they are. mint
// makes a new id: the turn's effects supply it, because a fresh id is random
// and workflow code may not be.
func DistinctCallIDs(
	completion *llmports.ChatCompletionResult,
	used map[string]struct{},
	mint func() string,
) {
	for idx := range completion.ToolCalls {
		call := &completion.ToolCalls[idx]
		_, taken := used[call.ID]
		if call.ID == "" || taken || call.SynthesizedID {
			call.ID = mint()
			call.SynthesizedID = false
		}
		used[call.ID] = struct{}{}
	}
}
