package accesstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

type withEntityReaders struct {
	agenttoolports.AgentQueryToolRegistry
}

// WithEntityReaders completes a rig's query registry with the entity readers
// it leaves out. Production's registry always holds every reader behind
// records_query and record_get, and the capability registry search_tools
// answers from refuses to build without them. Descriptors are the rig's own,
// so the catalog an agent holds from is unchanged, and a core tool the rig
// left out stays out.
func WithEntityReaders(
	registry agenttoolports.AgentQueryToolRegistry,
) agenttoolports.AgentQueryToolRegistry {
	if _, done := registry.(withEntityReaders); done {
		return registry
	}

	return withEntityReaders{AgentQueryToolRegistry: registry}
}

func (r withEntityReaders) Get(name string) (agenttoolports.AgentQueryTool, bool) {
	if tool, ok := r.AgentQueryToolRegistry.Get(name); ok {
		return tool, true
	}
	if agentcapability.IsBacking(name) && !agentdefinition.IsCoreTool(name) {
		return &agenttest.StubQueryTool{ToolName: name}, true
	}

	return nil, false
}

func (r withEntityReaders) All() []agenttoolports.AgentQueryTool {
	own := r.AgentQueryToolRegistry.All()
	held := make(map[string]struct{}, len(own))
	for _, tool := range own {
		held[tool.Name()] = struct{}{}
	}
	backings := agentcapability.Backings()
	all := make([]agenttoolports.AgentQueryTool, 0, len(own)+len(backings))
	all = append(all, own...)
	for _, name := range backings {
		if _, ok := held[name]; !ok {
			all = append(all, &agenttest.StubQueryTool{ToolName: name})
		}
	}

	return all
}
