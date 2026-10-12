package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/fx"
)

type RegistryParams struct {
	fx.In

	Tools []agentports.AgentTool `group:"agent_tools"`
}

type registry struct {
	byName  map[string]agentports.AgentTool
	ordered []agentports.AgentTool
}

func NewRegistry(p RegistryParams) agentports.AgentToolRegistry {
	byName := make(map[string]agentports.AgentTool, len(p.Tools))
	ordered := make([]agentports.AgentTool, 0, len(p.Tools))

	for _, tool := range p.Tools {
		if _, exists := byName[tool.Name()]; exists {
			continue
		}
		byName[tool.Name()] = tool
		ordered = append(ordered, tool)
	}

	return &registry{byName: byName, ordered: ordered}
}

func (r *registry) Get(name string) (agentports.AgentTool, bool) {
	tool, ok := r.byName[name]
	return tool, ok
}

func (r *registry) All() []agentports.AgentTool {
	return r.ordered
}

func (r *registry) Descriptors() []agentports.AgentToolDescriptor {
	descriptors := make([]agentports.AgentToolDescriptor, 0, len(r.ordered))

	for _, tool := range r.ordered {
		descriptors = append(descriptors,
			agentports.DescribeTool(tool, tool.Policy().DefaultTier, false))
	}

	return descriptors
}
