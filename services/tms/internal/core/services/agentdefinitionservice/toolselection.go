package agentdefinitionservice

import (
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
}

func buildToolCatalog(
	actions serviceports.AgentToolRegistry,
	queries serviceports.AgentQueryToolRegistry,
) []serviceports.ToolCatalogEntry {
	queryTools := queries.All()
	actionTools := actions.All()
	entries := make([]serviceports.ToolCatalogEntry, 0, len(queryTools)+len(actionTools))

	for _, tool := range queryTools {
		entries = append(entries, serviceports.ToolCatalogEntry{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
			Kind:        serviceports.ToolCatalogKindQuery,
			Resource:    tool.PermissionResource(),
			Operation:   permission.OpRead,
			Reversible:  true,
		})
	}

	for _, tool := range actionTools {
		entries = append(entries, serviceports.ToolCatalogEntry{
			Name:                tool.Name(),
			Description:         tool.Description(),
			Parameters:          tool.ParamSchema(),
			Kind:                serviceports.ToolCatalogKindAction,
			Resource:            tool.PermissionResource(),
			Operation:           tool.PermissionOperation(),
			DefaultAutonomyTier: tool.DefaultAutonomyTier(),
			Reversible:          tool.Reversible(),
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
