package conversationreplay

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
)

// What a compaction keeps in full besides the agent's instructions and the
// pinned facts, which never take part in it.
const (
	KeptRecent    = "recent"
	KeptApprovals = "approvals"
)

// ReplayOrder is a conversation's history as the model reads it once part of
// it has been compacted: the latest summary first, in place of everything it
// stands in for, then every message after that stretch in order. A history
// with no summary in it is returned as it is.
//
// The summary is written after the turns it keeps whole, so in sequence
// order it follows them; the model has to read it before them, or the
// conversation would run backwards.
func ReplayOrder(history []conversation.Message) []conversation.Message {
	latest := -1
	for idx := len(history) - 1; idx >= 0; idx-- {
		if history[idx].Compacted() {
			latest = idx
			break
		}
	}
	if latest < 0 {
		return history
	}

	// An earlier summary is folded into the latest one, which was written
	// from it, so it is never read again even where it is numbered after
	// the stretch the latest one replaces.
	summary := history[latest]
	ordered := make([]conversation.Message, 0, len(history))
	ordered = append(ordered, summary)
	for idx := range history {
		if history[idx].Compacted() || history[idx].Sequence <= summary.Compaction.Through {
			continue
		}
		ordered = append(ordered, history[idx])
	}

	return ordered
}
