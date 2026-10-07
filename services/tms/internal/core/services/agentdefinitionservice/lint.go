package agentdefinitionservice

import (
	"context"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentlint"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

var resourceNames = permission.NewRegistry()

// LintInstructions reads instructions against the tools a draft holds and
// returns each thing they ask for that none of those tools can do.
func (s *Service) LintInstructions(
	ctx context.Context,
	req *services.LintAgentInstructionsRequest,
) ([]agentlint.Finding, error) {
	if utf8.RuneCountInString(req.Instructions) > agentdefinition.MaxInstructionsRunes {
		return nil, errortypes.NewValidationError(
			"instructions",
			errortypes.ErrInvalidLength,
			"Instructions cannot be longer than 20000 characters",
		)
	}

	catalog, err := s.ToolCatalog(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return agentlint.Lint(req.Instructions, lintRegistry(catalog, req)), nil
}

// lintRegistry is the catalog as the lint reads it: each tool with the name
// its record goes by, and whether the draft holds it. A held tool brings its
// prerequisites with it, and core tools and those an extension gives every
// agent are held by all.
func lintRegistry(
	catalog []services.ToolCatalogEntry,
	req *services.LintAgentInstructionsRequest,
) []agentlint.Tool {
	held := make(map[string]struct{}, len(req.ToolNames))
	for _, name := range req.ToolNames {
		held[name] = struct{}{}
	}
	for _, name := range req.DisabledToolNames {
		delete(held, name)
	}
	for idx := range catalog {
		entry := &catalog[idx]
		if entry.Core || entry.GrantedToEveryAgent {
			held[entry.Name] = struct{}{}
		}
	}
	for idx := range catalog {
		entry := &catalog[idx]
		if _, ok := held[entry.Name]; !ok {
			continue
		}
		for _, prerequisite := range entry.Prerequisites {
			held[prerequisite] = struct{}{}
		}
	}

	tools := make([]agentlint.Tool, 0, len(catalog))
	for idx := range catalog {
		entry := &catalog[idx]
		if entry.Resource == "" {
			continue
		}
		label := entry.Resource.String()
		if def, ok := resourceNames.Get(entry.Resource.String()); ok && def.DisplayName != "" {
			label = def.DisplayName
		}
		_, isHeld := held[entry.Name]
		tools = append(tools, agentlint.Tool{
			Name:          entry.Name,
			Resource:      entry.Resource,
			ResourceLabel: label,
			Operation:     entry.Operation,
			Held:          isHeld,
		})
	}
	return tools
}
