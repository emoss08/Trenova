package agent

// RunOwnerKind names the unit of work a step, a usage row or a delegate run
// belongs to: a background agent run or a conversation turn.
type RunOwnerKind string

const (
	RunOwnerAgentRun      = RunOwnerKind("AgentRun")
	RunOwnerAssistantTurn = RunOwnerKind("AssistantTurn")
)

func (k RunOwnerKind) IsValid() bool {
	switch k {
	case RunOwnerAgentRun, RunOwnerAssistantTurn:
		return true
	default:
		return false
	}
}

func AllRunOwnerKinds() []RunOwnerKind {
	return []RunOwnerKind{RunOwnerAgentRun, RunOwnerAssistantTurn}
}
