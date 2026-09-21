package conversation

// Role identifies who produced a message.
type Role string

const (
	RoleUser      = Role("User")
	RoleAssistant = Role("Assistant")
	// RoleTool carries a tool's result. It is persisted so a conversation can be
	// replayed and audited, not just re-read.
	RoleTool = Role("Tool")
)

func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

// ThreadStatus is a conversation's lifecycle.
type ThreadStatus string

const (
	ThreadStatusActive   = ThreadStatus("Active")
	ThreadStatusArchived = ThreadStatus("Archived")
)

func (s ThreadStatus) IsValid() bool {
	switch s {
	case ThreadStatusActive, ThreadStatusArchived:
		return true
	default:
		return false
	}
}

// ThreadOrigin is where a conversation was started. The Desk shows a
// person's own conversations; a quick question asked from the command
// palette stays out of that list until the person keeps it, and one opened
// from the watchtower or a briefing says so, since it began about something.
type ThreadOrigin string

const (
	ThreadOriginPanel      = ThreadOrigin("Panel")
	ThreadOriginDesk       = ThreadOrigin("Desk")
	ThreadOriginAsk        = ThreadOrigin("Ask")
	ThreadOriginWatchtower = ThreadOrigin("Watchtower")
	ThreadOriginBriefing   = ThreadOrigin("Briefing")
)

func (o ThreadOrigin) IsValid() bool {
	switch o {
	case ThreadOriginPanel, ThreadOriginDesk, ThreadOriginAsk,
		ThreadOriginWatchtower, ThreadOriginBriefing:
		return true
	default:
		return false
	}
}

// Listed reports whether the Desk's thread rail shows a conversation of
// this origin. A quick question is not listed until it is kept.
func (o ThreadOrigin) Listed() bool {
	return o != ThreadOriginAsk
}
