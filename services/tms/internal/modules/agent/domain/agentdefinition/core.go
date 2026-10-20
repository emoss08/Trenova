package agentdefinition

import "slices"

const (
	CoreToolRecallMemory   = "agent_list_memories"
	CoreToolRemember       = "agent_create_memory"
	CoreToolGuideProfile   = "agent_guide_profile"
	CoreToolRaiseException = "agent_raise_exception"
	CoreToolFindInTrenova  = "agent_find_page"
	CoreToolOpenPage       = "agent_open_page"
	CoreToolWaitUntil      = "agent_create_wait"
	CoreToolCancelWait     = "agent_cancel_wait"
	CoreToolReadAttachment = "agent_read_attachment"
)

var coreTools = [...]string{
	CoreToolRecallMemory,
	CoreToolRemember,
	CoreToolGuideProfile,
	CoreToolRaiseException,
	CoreToolFindInTrenova,
	CoreToolOpenPage,
	CoreToolWaitUntil,
	CoreToolCancelWait,
	CoreToolReadAttachment,
}

// CoreTools are held by every agent without being selected. Memory, escalation,
// waiting on the world and knowing the product it runs in are how an agent works at all
// rather than what it works on, so an organization building an agent should
// not have to know to tick them. agent_find_page, agent_open_page and agent_read_attachment
// answer for the person in the conversation, so a run nobody is watching goes
// without them; agent_read_attachment rides only on a conversation a file was
// attached to. agent_guide_profile writes down what a person said about the
// organization's business or their own work, so it rides only where a person
// is talking to the agent.
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

func (d *Definition) HoldingReads(reads []string) *Definition {
	held := make(map[string]struct{}, len(d.ToolNames)+len(d.DisabledToolNames))
	for _, name := range d.ToolNames {
		held[name] = struct{}{}
	}
	for _, name := range d.DisabledToolNames {
		held[name] = struct{}{}
	}

	names := make([]string, 0, len(d.ToolNames)+len(reads))
	names = append(names, d.ToolNames...)
	for _, name := range reads {
		if _, skip := held[name]; skip || IsCoreTool(name) {
			continue
		}
		held[name] = struct{}{}
		names = append(names, name)
	}

	reading := *d
	reading.ToolNames = names

	return &reading
}
