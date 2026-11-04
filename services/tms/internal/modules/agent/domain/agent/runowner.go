package agent

// RunOwnerKind names the unit of work a step, a usage row or a delegate run
// belongs to: a background agent run, a conversation turn, or a question
// asked from the command palette (its model calls and reads, never a step).
type RunOwnerKind string

const (
	RunOwnerAgentRun        = RunOwnerKind("AgentRun")
	RunOwnerAssistantTurn   = RunOwnerKind("AssistantTurn")
	RunOwnerPaletteQuestion = RunOwnerKind("PaletteQuestion")
)

func (k RunOwnerKind) IsValid() bool {
	switch k {
	case RunOwnerAgentRun, RunOwnerAssistantTurn, RunOwnerPaletteQuestion:
		return true
	default:
		return false
	}
}

func AllRunOwnerKinds() []RunOwnerKind {
	return []RunOwnerKind{RunOwnerAgentRun, RunOwnerAssistantTurn, RunOwnerPaletteQuestion}
}
