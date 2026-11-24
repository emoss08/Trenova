package agentdefinitionservice

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
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
	known := func(name string) bool {
		if _, _, ok := agentcapability.ParseCapabilityID(name); ok {
			entity, _, _ := agentcapability.ParseCapabilityID(name)
			_, exists := agentcapability.EntityNamed(entity)

			return exists
		}
		if _, ok := queries.Get(name); ok {
			return true
		}
		_, ok := actions.Get(name)

		return ok
	}
	areas := knownAreas(actions, queries)
	checkAreas := func(field string, values []string) {
		for idx, area := range values {
			if area == agentdefinition.AllAreas || slices.Contains(areas, area) {
				continue
			}
			multiErr.Add(fmt.Sprintf("%s[%d]", field, idx), errortypes.ErrInvalid,
				fmt.Sprintf("%q is not an area of Trenova", area))
		}
	}
	checkAreas("scope.areas", definition.Scope.Areas)
	checkAreas("scope.denyAreas", definition.Scope.DenyAreas)
	checkCapabilities := func(field string, values []string, offered bool) {
		for idx, name := range values {
			switch {
			case !known(name):
				multiErr.Add(fmt.Sprintf("%s[%d]", field, idx), errortypes.ErrInvalid,
					fmt.Sprintf("%q is not a capability this system provides", name))
			case offered && !selection.extensions.offers(name) && !selection.previouslyHeld(name):
				multiErr.Add(fmt.Sprintf("%s[%d]", field, idx), errortypes.ErrInvalid,
					fmt.Sprintf("%q comes with an extension that is not turned on for this "+
						"organization. Turn it on under AI Control, Extensions first", name))
			}
		}
	}
	checkCapabilities("scope.capabilities", definition.Scope.Capabilities, true)
	checkCapabilities("scope.denyCapabilities", definition.Scope.DenyCapabilities, false)
	checkCapabilities("pinnedTools", definition.PinnedTools, true)

	for _, name := range slices.Sorted(maps.Keys(definition.ToolTiers)) {
		tier := definition.ToolTiers[name]
		tool, ok := actions.Get(name)
		if !ok {
			continue
		}
		policy := tool.Policy()
		if limit := agenttoolpolicy.Promotable(&policy); tier.Above(limit) {
			multiErr.Add(
				fmt.Sprintf("toolTiers.%s", name),
				errortypes.ErrInvalid,
				fmt.Sprintf("%q can be set to %s at most: %s", name, limit, policy.Rationale),
			)
			continue
		}
		if tier.Above(policy.DefaultTier) && !selection.earned(name, tier) {
			multiErr.Add(
				fmt.Sprintf("toolTiers.%s", name),
				errortypes.ErrInvalid,
				fmt.Sprintf("%q runs at %s by default; an override can only be stricter. "+
					"More freedom is earned from approvals, not set here.", name, policy.DefaultTier),
			)
		}
	}
}

func (s toolSelection) previouslyHeld(name string) bool {
	return s.previous != nil && s.previous.AllowsTool(name)
}

func (s toolSelection) earned(name string, tier agent.AutonomyTier) bool {
	return s.previous != nil && s.previous.ToolTiers[name] == tier
}

func knownAreas(
	actions agenttoolports.AgentToolRegistry,
	queries agenttoolports.AgentQueryToolRegistry,
) []string {
	areas := make([]string, 0, 40)
	add := func(name string) {
		area := agentcapability.AreaOf(name)
		if entity, _, ok := agentcapability.EntityBacking(name); ok {
			area = entity.Area
		}
		if !slices.Contains(areas, area) {
			areas = append(areas, area)
		}
	}
	for _, tool := range queries.All() {
		add(tool.Name())
	}
	for _, tool := range actions.All() {
		add(tool.Name())
	}

	return areas
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
		Description: agenttoolports.ModelDescription(tool),
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
		entry.MaxAutonomyTier = agenttoolpolicy.Promotable(&policy)
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
	template agenttemplate.Template,
	actions agenttoolports.AgentToolRegistry,
	queries agenttoolports.AgentQueryToolRegistry,
) []string {
	starters := agentpreset.StarterTools(template)
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
