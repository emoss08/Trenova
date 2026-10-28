package agentruntime

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
)

const (
	otherToolsArea       = "Other"
	maxToolAreaResources = 6
)

func (s *Service) toolMap(names []string) []agentturn.ToolNamespace {
	if s.registry == nil {
		s.registry = permissioncatalog.NewRegistry()
	}
	type area struct {
		resources []string
		tools     int
	}
	areas := make(map[string]*area, 16)
	for _, name := range slices.Compact(slices.Sorted(slices.Values(names))) {
		resource, _, ok := s.toolGate(name)
		if !ok {
			continue
		}
		label, covers := otherToolsArea, ""
		if definition, found := s.registry.Get(resource.String()); found {
			label = strings.TrimSpace(definition.Category)
			covers = strings.ToLower(strings.TrimSpace(definition.DisplayName))
		}
		if label == "" {
			label = otherToolsArea
		}
		entry, seen := areas[label]
		if !seen {
			entry = &area{}
			areas[label] = entry
		}
		entry.tools++
		if covers != "" && !slices.Contains(entry.resources, covers) {
			entry.resources = append(entry.resources, covers)
		}
	}

	out := make([]agentturn.ToolNamespace, 0, len(areas))
	for label, entry := range areas {
		slices.Sort(entry.resources)
		namespace := agentturn.ToolNamespace{Name: label, Tools: entry.tools}
		namespace.Covers = entry.resources[:min(len(entry.resources), maxToolAreaResources)]
		namespace.MoreCovered = len(entry.resources) - len(namespace.Covers)
		out = append(out, namespace)
	}
	slices.SortFunc(out, func(a, b agentturn.ToolNamespace) int {
		return strings.Compare(a.Name, b.Name)
	})

	return out
}
