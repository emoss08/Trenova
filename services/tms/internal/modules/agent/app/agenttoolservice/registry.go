package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"go.uber.org/fx"
)

type RegistryParams struct {
	fx.In

	Tools []agenttoolports.AgentTool `group:"agent_tools"`
}

type registry struct {
	byName  map[string]agenttoolports.AgentTool
	ordered []agenttoolports.AgentTool
}

func NewRegistry(p RegistryParams) agenttoolports.AgentToolRegistry {
	byName := make(map[string]agenttoolports.AgentTool, len(p.Tools))
	ordered := make([]agenttoolports.AgentTool, 0, len(p.Tools))

	for _, tool := range p.Tools {
		if _, exists := byName[tool.Name()]; exists {
			continue
		}
		byName[tool.Name()] = tool
		ordered = append(ordered, tool)
	}

	return &registry{byName: byName, ordered: ordered}
}

func (r *registry) Get(name string) (agenttoolports.AgentTool, bool) {
	tool, ok := r.byName[name]
	return tool, ok
}

func (r *registry) All() []agenttoolports.AgentTool {
	return r.ordered
}

func (r *registry) Descriptors() []agenttoolports.AgentToolDescriptor {
	descriptors := make([]agenttoolports.AgentToolDescriptor, 0, len(r.ordered))

	for _, tool := range r.ordered {
		descriptors = append(descriptors,
			agenttoolports.DescribeTool(tool, tool.Policy().DefaultTier, false))
	}

	return descriptors
}
