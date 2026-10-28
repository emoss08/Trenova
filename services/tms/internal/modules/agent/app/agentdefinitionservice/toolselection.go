package agentdefinitionservice

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type extensionState agentextension.Active

func (e extensionState) offers(name string) bool {
	return agentextension.Active(e).Offers(name)
}

type toolSelection struct {
	definition *agentdefinition.Definition
	previous   *agentdefinition.Definition
	actions    agenttoolports.AgentToolRegistry
	queries    agenttoolports.AgentQueryToolRegistry
	extensions extensionState
}

func validateToolSelection(selection toolSelection, multiErr *errortypes.MultiError) {
	definition := selection.definition
	actions := selection.actions
	queries := selection.queries
	for idx, name := range definition.ToolNames {
		if !selection.extensions.offers(name) && !selection.previouslyHeld(name) {
			multiErr.Add(
				fmt.Sprintf("toolNames[%d]", idx),
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"%q comes with an extension that is not turned on for this "+
						"organization. Turn it on under AI Control, Extensions first",
					name,
				),
			)
			continue
		}
		if _, ok := queries.Get(name); ok {
			continue
		}
		if _, ok := actions.Get(name); ok {
			continue
		}

		multiErr.Add(
			fmt.Sprintf("toolNames[%d]", idx),
			errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a tool this system provides", name),
		)
	}

	for _, name := range slices.Sorted(maps.Keys(definition.ToolTiers)) {
		tier := definition.ToolTiers[name]
		tool, ok := actions.Get(name)
		if !ok {
			continue
		}
		policy := tool.Policy()
		if limit := agenttoolpolicy.Promotable(policy); tier.Above(limit) {
			multiErr.Add(
				fmt.Sprintf("toolTiers.%s", name),
				errortypes.ErrInvalid,
				fmt.Sprintf("%q can be set to %s at most: %s", name, limit, policy.Rationale),
			)
		}
	}
}

func (s toolSelection) previouslyHeld(name string) bool {
	return s.previous != nil && slices.Contains(s.previous.ToolNames, name)
}

func buildToolCatalog(
	actions agenttoolports.AgentToolRegistry,
	queries agenttoolports.AgentQueryToolRegistry,
	extensions extensionState,
) []agentdefinitionports.ToolCatalogEntry {
	queryTools := queries.All()
	actionTools := actions.All()
	entries := make([]agentdefinitionports.ToolCatalogEntry, 0, len(queryTools)+len(actionTools))

	for _, tool := range queryTools {
		if entry, offered := extensions.entry(
			tool,
			agentdefinitionports.ToolCatalogKindQuery,
		); offered {
			entries = append(entries, entry)
		}
	}

	for _, tool := range actionTools {
		if entry, offered := extensions.entry(
			tool,
			agentdefinitionports.ToolCatalogKindAction,
		); offered {
			entries = append(entries, entry)
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Resource != entries[j].Resource {
			return entries[i].Resource < entries[j].Resource
		}

		return entries[i].Name < entries[j].Name
	})

	return entries
}

type catalogTool interface {
	Name() string
	Description() string
	ParamSchema() map[string]any
	Policy() agenttoolports.ToolPolicy
}

func (e extensionState) entry(
	tool catalogTool,
	kind agentdefinitionports.ToolCatalogKind,
) (agentdefinitionports.ToolCatalogEntry, bool) {
	if !e.offers(tool.Name()) {
		return agentdefinitionports.ToolCatalogEntry{}, false
	}

	entry := catalogEntry(tool, kind)
	if served, ok := agentextension.Active(e).Serving(tool.Name()); ok {
		entry.Extension = served.Type
		entry.GrantedToEveryAgent = served.Availability == agentextension.AvailabilityAllAgents
	}

	return entry, true
}

func catalogEntry(
	tool catalogTool,
	kind agentdefinitionports.ToolCatalogKind,
) agentdefinitionports.ToolCatalogEntry {
	policy := tool.Policy()
	entry := agentdefinitionports.ToolCatalogEntry{
		Name:        tool.Name(),
		Description: tool.Description(),
		Parameters:  tool.ParamSchema(),
		Kind:        kind,
		Resource:    policy.Resource,
		Operation:   policy.Operation,
		Reversible:  policy.Reversible,
		Core:        agentdefinition.IsCoreTool(tool.Name()),
		Effect:      agenttoolports.EffectOf(tool),
	}
	entry.Prerequisites = prerequisitesOf(tool)
	if kind == agentdefinitionports.ToolCatalogKindAction {
		entry.DefaultAutonomyTier = policy.DefaultTier
		entry.MaxAutonomyTier = agenttoolpolicy.Promotable(policy)
	}

	return entry
}

// prerequisitesOf is always a list, empty when the tool names none: a tool
// that can name prerequisites and names none returns nil, which the client,
// reading a list, refused.
func prerequisitesOf(tool any) []string {
	if dependent, ok := tool.(agenttoolports.PrerequisiteTool); ok {
		if prerequisites := dependent.Prerequisites(); prerequisites != nil {
			return prerequisites
		}
	}

	return []string{}
}

func registeredStarterTools(
	template agentdefinition.Template,
	actions agenttoolports.AgentToolRegistry,
	queries agenttoolports.AgentQueryToolRegistry,
) []string {
	starters := template.StarterTools()
	tools := make([]string, 0, len(starters))
	for _, name := range starters {
		if _, ok := queries.Get(name); ok {
			tools = append(tools, name)
			continue
		}
		if _, ok := actions.Get(name); ok {
			tools = append(tools, name)
		}
	}

	return tools
}
