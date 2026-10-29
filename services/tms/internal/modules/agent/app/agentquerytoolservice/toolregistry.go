package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/tableviewquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"go.uber.org/fx"
)

type RegistryParams struct {
	fx.In

	Tools []agenttoolports.AgentQueryTool `group:"agent_query_tools"`
}

type registry struct {
	byName  map[string]agenttoolports.AgentQueryTool
	ordered []agenttoolports.AgentQueryTool
}

// legacyToolNames maps a name an agent may still have saved to the tool that
// now carries it.
//
// A tool name is persisted — in agent_definitions.tool_names and as a key in
// tool_tiers — so renaming one is a data change. The migration rewrites both
// columns, but a definition written by an API client that hard-coded the old
// name would otherwise resolve to nothing, and the failure mode is silent: the
// agent simply stops being able to look drivers up. Resolving the old name
// costs one map entry.
//
// These are deliberately absent from Descriptors, so the model is never offered
// a name we no longer want it to learn.
var legacyToolNames = map[string]string{
	"search_workers": "worker_list",
}

func NewRegistry(p RegistryParams) agenttoolports.AgentQueryToolRegistry {
	byName := make(map[string]agenttoolports.AgentQueryTool, len(p.Tools))
	ordered := make([]agenttoolports.AgentQueryTool, 0, len(p.Tools))

	for _, tool := range p.Tools {
		if _, exists := byName[tool.Name()]; exists {
			continue
		}
		byName[tool.Name()] = tool
		ordered = append(ordered, tool)
	}

	for legacy, current := range legacyToolNames {
		if _, taken := byName[legacy]; taken {
			continue
		}
		if tool, ok := byName[current]; ok {
			byName[legacy] = tool
		}
	}

	linkViewsToLists(ordered)

	return &registry{byName: byName, ordered: ordered}
}

// linkViewsToLists gives the view composer each table's list tool, so a view
// it composes is counted and previewed through the same fetch that listing
// the table uses. It is wired here because both are tools the registry
// collects; neither can be handed the other when it is built.
func linkViewsToLists(tools []agenttoolports.AgentQueryTool) {
	lists := make(map[permission.Resource]querytool.ViewRunner)
	link := func(resource permission.Resource, list querytool.ViewRunner) {
		if _, taken := lists[resource]; !taken {
			lists[resource] = list
		}
	}
	for _, tool := range tools {
		switch list := tool.(type) {
		case *listtool.Tool:
			link(list.Spec.Resource, list)
		case *graphqllisttool.Tool:
			link(list.Spec.Resource, list)
		}
	}
	for _, tool := range tools {
		if composer, ok := tool.(*tableviewquerytools.ComposeTableViewTool); ok {
			composer.Lists = lists
		}
	}
}

func (r *registry) Get(name string) (agenttoolports.AgentQueryTool, bool) {
	tool, ok := r.byName[name]
	return tool, ok
}

func (r *registry) All() []agenttoolports.AgentQueryTool { return r.ordered }

func (r *registry) Descriptors() []agenttoolports.AgentToolDescriptor {
	descriptors := make([]agenttoolports.AgentToolDescriptor, 0, len(r.ordered))

	for _, tool := range r.ordered {
		descriptors = append(descriptors, agenttoolports.DescribeTool(tool, "", true))
	}

	return descriptors
}
