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
