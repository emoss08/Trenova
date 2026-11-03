package agentdefinition

import "slices"

var coreTools = [...]string{
	"agent_list_memories",
	"agent_create_memory",
	"agent_guide_profile",
	"agent_raise_exception",
	"agent_find_page",
	"agent_open_page",
	"agent_create_wait",
	"agent_cancel_wait",
	"agent_create_schedule",
	"agent_read_attachment",
}

// CoreTools are held by every agent without being selected. Memory, escalation,
// waiting on the world and knowing the product it runs in are how an agent works at all
// rather than what it works on, so an organization building an agent should
// not have to know to tick them. agent_find_page, agent_open_page and agent_read_attachment
// answer for the person in the conversation, so a run nobody is watching goes
// without them; agent_read_attachment rides only on a conversation a file was
// attached to. agent_guide_profile writes down what a person said about the
// organization's business or their own work, and agent_create_schedule
// repeats a request in the person's own conversation, so both ride only where
// a person is talking to the agent.
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
