package agentdefinitionservice

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func validateToolSelection(
	definition *agentdefinition.Definition,
	actions serviceports.AgentToolRegistry,
	queries serviceports.AgentQueryToolRegistry,
	multiErr *errortypes.MultiError,
) {
	for idx, name := range definition.ToolNames {
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

func buildToolCatalog(
	actions serviceports.AgentToolRegistry,
	queries serviceports.AgentQueryToolRegistry,
) []serviceports.ToolCatalogEntry {
	queryTools := queries.All()
	actionTools := actions.All()
	entries := make([]serviceports.ToolCatalogEntry, 0, len(queryTools)+len(actionTools))

	for _, tool := range queryTools {
		entries = append(entries, catalogEntry(tool, serviceports.ToolCatalogKindQuery))
	}

	for _, tool := range actionTools {
		entries = append(entries, catalogEntry(tool, serviceports.ToolCatalogKindAction))
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Resource != entries[j].Resource {
			return entries[i].Resource < entries[j].Resource
		}

		return entries[i].Name < entries[j].Name
	})

	return entries
}

func catalogEntry(
	tool interface {
		Name() string
		Description() string
		ParamSchema() map[string]any
		Policy() serviceports.ToolPolicy
	},
	kind serviceports.ToolCatalogKind,
) serviceports.ToolCatalogEntry {
	policy := tool.Policy()
	entry := serviceports.ToolCatalogEntry{
		Name:        tool.Name(),
		Description: tool.Description(),
		Parameters:  tool.ParamSchema(),
		Kind:        kind,
		Resource:    policy.Resource,
		Operation:   policy.Operation,
		Reversible:  policy.Reversible,
		Core:        agentdefinition.IsCoreTool(tool.Name()),
		Effect:      serviceports.EffectOf(tool),
	}
	entry.Prerequisites = prerequisitesOf(tool)
	if kind == serviceports.ToolCatalogKindAction {
		entry.DefaultAutonomyTier = policy.DefaultTier
	}

	return entry
}

func prerequisitesOf(tool any) []string {
	if dependent, ok := tool.(serviceports.PrerequisiteTool); ok {
		return dependent.Prerequisites()
	}

	return []string{}
}

func registeredStarterTools(
	template agentdefinition.Template,
	actions serviceports.AgentToolRegistry,
	queries serviceports.AgentQueryToolRegistry,
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
