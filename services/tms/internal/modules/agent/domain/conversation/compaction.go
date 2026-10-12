package conversation

// AutoCompactShare is how full a conversation's context may get before it is
// compacted on its own. It leaves room for one more long turn: a reply that
// reads a few listings can add a tenth of a window, and a turn that does not
// fit is one the model cannot answer at all.
const AutoCompactShare = 0.85

// MinCompactionTokens is the least a compaction must free to be worth doing.
// Below it the summary costs a model call and loses detail for room nobody
// needs yet.
const MinCompactionTokens = 4000

// ContextUsage is how much of the model's context window the next turn of a
// conversation fills, estimated from what that turn would send: the agent's
// instructions and tools, the messages replayed, the tool results among them
// and the files read through them.
//
// It is measured when a turn ends and when a compaction ends, and kept on the
// thread, so reading a conversation never rebuilds a prompt to say how full
// it is.
type ContextUsage struct {
	// Instructions is the system prompt and the tool definitions: the
	// agent's instructions, the pinned facts and memories it carries, and
	// the tools it may call. Compaction never touches it.
	Instructions int `json:"instructions"`
	// Messages is what the person and the agent said, and the summary of
	// any compacted stretch.
	Messages int `json:"messages"`
	// ToolResults is what the agent's tools returned, as replayed.
	ToolResults int `json:"toolResults"`
	// Files is the part of the tool results that is the text of a document
	// the person attached.
	Files int `json:"files"`
	// Compactable is how much of the replay a compaction would summarize:
	// everything before the latest turns, which stay whole.
	Compactable int `json:"compactable"`
	// Window is the model's context window, in tokens.
	Window int `json:"window"`
	// Model is the model the window is for.
	Model      string `json:"model,omitempty"`
	MeasuredAt int64  `json:"measuredAt"`
}

// Total is every token the next turn would send.
func (u *ContextUsage) Total() int {
	if u == nil {
		return 0
	}

	return u.Instructions + u.Messages + u.ToolResults + u.Files
}

// Share is how full the window is, from 0 to 1 and beyond for a conversation
// that already outgrew the model it is on.
func (u *ContextUsage) Share() float64 {
	if u == nil || u.Window <= 0 {
		return 0
	}

	return float64(u.Total()) / float64(u.Window)
}

// Frees is roughly what compacting now would give back: the stretch it
// summarizes, less the summary that replaces it.
func (u *ContextUsage) Frees() int {
	if u == nil {
		return 0
	}

	return u.Compactable - SummaryEstimate(u.Compactable)
}

// WorthCompacting reports a conversation with enough behind its latest turns
// to be worth summarizing.
func (u *ContextUsage) WorthCompacting() bool {
	return u.Frees() > MinCompactionTokens
}

// NeedsCompaction reports a conversation full enough to compact on its own,
// with enough to compact that doing so helps.
func (u *ContextUsage) NeedsCompaction() bool {
	return u.Share() >= AutoCompactShare && u.WorthCompacting()
}

// SummaryTokenBudget is the most a compaction summary may be. It is what the
// model is allowed to write, and so the most a summary can cost every later
// turn.
const SummaryTokenBudget = 1200

// SummaryEstimate is how long a summary of a stretch of conversation is
// expected to come out: a fifth of it, never more than the budget.
func SummaryEstimate(tokens int) int {
	return min(tokens/5, SummaryTokenBudget)
}

// Compaction says what a compaction summary stands in for. It is kept on the
// summary message, which the model is sent in place of every message up to
// and including Through.
type Compaction struct {
	// Auto says the conversation compacted itself on crossing
	// AutoCompactShare, rather than because somebody asked.
	Auto bool `json:"auto"`
	// Summarized is how many messages the summary replaces.
	Summarized int `json:"summarized"`
	// Through is the sequence of the last message it replaces. The messages
	// after it, which include the latest turns kept whole, are replayed
	// after the summary.
	Through int `json:"through"`
	// Before and After are the conversation's context use, in tokens, either
	// side of the compaction.
	Before int `json:"before"`
	After  int `json:"after"`
	// Kept names what stayed in full: KeptRecent for the latest turns,
	// KeptApprovals for the decisions still waiting.
	Kept []string `json:"kept,omitempty"`
}

// What a compaction keeps in full besides the agent's instructions and the
// pinned facts, which never take part in it.
const (
	KeptRecent    = "recent"
	KeptApprovals = "approvals"
)

// Compacted reports a compaction summary.
func (m *Message) Compacted() bool {
	return m.Kind == MessageKindCompaction && m.Compaction != nil
}

// ReplayOrder is a conversation's history as the model reads it once part of
// it has been compacted: the latest summary first, in place of everything it
// stands in for, then every message after that stretch in order. A history
// with no summary in it is returned as it is.
//
// The summary is written after the turns it keeps whole, so in sequence
// order it follows them; the model has to read it before them, or the
// conversation would run backwards.
func ReplayOrder(history []Message) []Message {
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
	ordered := make([]Message, 0, len(history))
	ordered = append(ordered, summary)
	for idx := range history {
		if history[idx].Compacted() || history[idx].Sequence <= summary.Compaction.Through {
			continue
		}
		ordered = append(ordered, history[idx])
	}

	return ordered
}
