package agentdefinitionservice

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type extensionState map[agentextension.Type]agentextension.Availability

func (e extensionState) offers(name string) (agentextension.Type, bool) {
	typ, isExtension := agentextension.ExtensionForTool(name)
	if !isExtension {
		return "", true
	}
	_, active := e[typ]

	return typ, active
}

type toolSelection struct {
	definition *agentdefinition.Definition
	previous   *agentdefinition.Definition
	actions    serviceports.AgentToolRegistry
	queries    serviceports.AgentQueryToolRegistry
	extensions extensionState
}

func validateToolSelection(selection toolSelection, multiErr *errortypes.MultiError) {
	definition := selection.definition
	actions := selection.actions
	queries := selection.queries
	for idx, name := range definition.ToolNames {
		if typ, offered := selection.extensions.offers(name); !offered &&
			!selection.previouslyHeld(name) {
			multiErr.Add(
				fmt.Sprintf("toolNames[%d]", idx),
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"%q comes with the %s extension, which is not turned on for this "+
						"organization. Turn it on under AI Control, Extensions first",
					name, typ,
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
		if ceiling := serviceports.CeilingOf(tool); tier.Above(ceiling) {
			multiErr.Add(
				fmt.Sprintf("toolTiers.%s", name),
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"%q sends work outside the organization, so a person approves it: it can be set to %s at most",
					name, ceiling,
				),
			)
		}
	}
}

func (s toolSelection) previouslyHeld(name string) bool {
	return s.previous != nil && slices.Contains(s.previous.ToolNames, name)
}

func buildToolCatalog(
	actions serviceports.AgentToolRegistry,
	queries serviceports.AgentQueryToolRegistry,
	extensions extensionState,
) []serviceports.ToolCatalogEntry {
	queryTools := queries.All()
	actionTools := actions.All()
	entries := make([]serviceports.ToolCatalogEntry, 0, len(queryTools)+len(actionTools))

	for _, tool := range queryTools {
		typ, offered := extensions.offers(tool.Name())
		if !offered {
			continue
		}
		entries = append(entries, serviceports.ToolCatalogEntry{
			Name:          tool.Name(),
			Description:   tool.Description(),
			Parameters:    tool.ParamSchema(),
			Kind:          serviceports.ToolCatalogKindQuery,
			Resource:      tool.PermissionResource(),
			Operation:     permission.OpRead,
			Reversible:    true,
			Core:          agentdefinition.IsCoreTool(tool.Name()),
			Effect:        serviceports.EffectOf(tool),
			Prerequisites: prerequisitesOf(tool),
			Extension:     typ,
			GrantedToEveryAgent: typ != "" &&
				extensions[typ] == agentextension.AvailabilityAllAgents,
		})
	}

	for _, tool := range actionTools {
		typ, offered := extensions.offers(tool.Name())
		if !offered {
			continue
		}
		entries = append(entries, serviceports.ToolCatalogEntry{
			Name:                tool.Name(),
			Description:         tool.Description(),
			Parameters:          tool.ParamSchema(),
			Kind:                serviceports.ToolCatalogKindAction,
			Resource:            tool.PermissionResource(),
			Operation:           tool.PermissionOperation(),
			DefaultAutonomyTier: tool.DefaultAutonomyTier(),
			Reversible:          tool.Reversible(),
			Core:                agentdefinition.IsCoreTool(tool.Name()),
			Effect:              serviceports.EffectOf(tool),
			Prerequisites:       prerequisitesOf(tool),
			Extension:           typ,
			GrantedToEveryAgent: typ != "" &&
				extensions[typ] == agentextension.AvailabilityAllAgents,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Resource != entries[j].Resource {
			return entries[i].Resource < entries[j].Resource
		}

		return entries[i].Name < entries[j].Name
	})

	return entries
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
