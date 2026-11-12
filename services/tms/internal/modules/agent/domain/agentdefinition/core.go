package agentdefinition

import "slices"

var coreTools = [...]string{
	"agent_list_memories",
	"agent_create_memory",
	"agent_edit_memory",
	"agent_guide_profile",
	"agent_raise_exception",
	"agent_find_page",
	"agent_open_page",
	"agent_create_wait",
	"agent_cancel_wait",
	"agent_offer_options",
	"agent_create_schedule",
	"agent_read_attachment",
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
