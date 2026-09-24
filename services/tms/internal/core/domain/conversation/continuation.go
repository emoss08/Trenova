package conversation

// ContinueRefusal says why the person reading a conversation may no longer
// ask its agent anything. Empty while they may.
type ContinueRefusal string

const (
	// ContinueAgentDeleted is a conversation whose agent was removed.
	ContinueAgentDeleted = ContinueRefusal("AgentDeleted")
	// ContinueAgentDisabled is a conversation whose agent an administrator
	// turned off.
	ContinueAgentDisabled = ContinueRefusal("AgentDisabled")
	// ContinueAgentNotConversational is a conversation whose agent now runs on
	// a schedule or an event and takes no conversations.
	ContinueAgentNotConversational = ContinueRefusal("AgentNotConversational")
	// ContinueNoAccess is a conversation whose reader may no longer use its
	// agent, or the assistant at all.
	ContinueNoAccess = ContinueRefusal("NoAccess")
)

func (r ContinueRefusal) IsValid() bool {
	switch r {
	case ContinueAgentDeleted,
		ContinueAgentDisabled,
		ContinueAgentNotConversational,
		ContinueNoAccess:
		return true
	default:
		return false
	}
}

// MarkContinuable says the reader may still ask the conversation's agent.
func (t *Thread) MarkContinuable() {
	t.CanContinue = true
	t.CannotContinueReason = ""
}

// MarkNotContinuable says the reader may no longer ask the conversation's
// agent, and why.
func (t *Thread) MarkNotContinuable(reason ContinueRefusal) {
	t.CanContinue = false
	t.CannotContinueReason = reason
}
