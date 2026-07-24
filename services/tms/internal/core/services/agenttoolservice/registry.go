package agenttoolservice

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

type RegistryParams struct {
	fx.In

	Tools []serviceports.AgentTool `group:"agent_tools"`
}

type registry struct {
	byName  map[string]serviceports.AgentTool
	ordered []serviceports.AgentTool
}

func NewRegistry(p RegistryParams) serviceports.AgentToolRegistry {
	byName := make(map[string]serviceports.AgentTool, len(p.Tools))
	ordered := make([]serviceports.AgentTool, 0, len(p.Tools))

	for _, tool := range p.Tools {
		if _, exists := byName[tool.Name()]; exists {
			continue
		}
		byName[tool.Name()] = tool
		ordered = append(ordered, tool)
	}

	return &registry{byName: byName, ordered: ordered}
}

func (r *registry) Get(name string) (serviceports.AgentTool, bool) {
	tool, ok := r.byName[name]
	return tool, ok
}

func (r *registry) All() []serviceports.AgentTool {
	return r.ordered
}

func (r *registry) Descriptors() []serviceports.AgentToolDescriptor {
	descriptors := make([]serviceports.AgentToolDescriptor, 0, len(r.ordered))

	for _, tool := range r.ordered {
		descriptors = append(descriptors, serviceports.AgentToolDescriptor{
			Name:         tool.Name(),
			Description:  tool.Description(),
			Parameters:   tool.ParamSchema(),
			AutonomyTier: tool.DefaultAutonomyTier(),
		})
	}

	return descriptors
}
