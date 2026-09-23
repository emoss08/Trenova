package agentdefinition

import "slices"

const (
	CoreToolRecallMemory   = "recall_memory"
	CoreToolRemember       = "remember"
	CoreToolRaiseException = "raise_exception"
	CoreToolFlagForReview  = "flag_for_manual_review"
)

var coreTools = [...]string{
	CoreToolRecallMemory,
	CoreToolRemember,
	CoreToolRaiseException,
	CoreToolFlagForReview,
}

// CoreTools are held by every agent without being selected. Memory, escalation
// and review are how an agent works at all rather than what it works on, so an
// organization building an agent should not have to know to tick them.
// find_tools and ask_user are the other two always-on tools; the runtime
// answers those itself, so they are not registry tools and are not listed here.
func CoreTools() []string {
	return slices.Clone(coreTools[:])
}

func IsCoreTool(name string) bool {
	return slices.Contains(coreTools[:], name)
}

// WithoutCoreTools drops the core tools from a selection, so they are never
// stored per agent.
func WithoutCoreTools(names []string) []string {
	kept := make([]string, 0, len(names))
	for _, name := range names {
		if !IsCoreTool(name) {
			kept = append(kept, name)
		}
	}

	return kept
}

// EffectiveToolNames is every tool the agent holds: the core tools first, then
// the ones the organization selected.
func (d *Definition) EffectiveToolNames() []string {
	names := make([]string, 0, len(coreTools)+len(d.ToolNames))
	names = append(names, coreTools[:]...)
	for _, name := range d.ToolNames {
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	return names
}
