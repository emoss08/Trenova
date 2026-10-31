package paletteanswer

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/inflect"
)

// catalog is the Desk's own record reads, indexed by the kind of record they
// read: the one get and the one list each kind has. The palette calls them
// through three tools rather than offering the Desk's whole catalog.
type catalog struct {
	gets  map[string]agenttoolports.AgentQueryTool
	lists map[string]agenttoolports.AgentQueryTool
	kinds []string
}

func newCatalog(registry agenttoolports.AgentQueryToolRegistry) *catalog {
	c := &catalog{
		gets:  make(map[string]agenttoolports.AgentQueryTool),
		lists: make(map[string]agenttoolports.AgentQueryTool),
	}
	if registry == nil {
		return c
	}

	for _, tool := range registry.All() {
		resource := tool.Policy().Resource
		if resource == "" {
			continue
		}
		kind := resource.String()
		switch agenttoolports.ReadingOrder(tool.Name()) {
		case "get_" + kind:
			c.gets[kind] = tool
		case "list_" + inflect.PluralNoun(kind):
			c.lists[kind] = tool
		}
	}
	for kind := range c.lists {
		c.kinds = append(c.kinds, kind)
	}
	slices.Sort(c.kinds)
	if len(c.kinds) > maxListKinds {
		c.kinds = c.kinds[:maxListKinds]
	}

	return c
}

func (c *catalog) get(resource permission.Resource) (agenttoolports.AgentQueryTool, bool) {
	tool, ok := c.gets[resource.String()]

	return tool, ok
}

func (c *catalog) list(kind string) (agenttoolports.AgentQueryTool, bool) {
	tool, ok := c.lists[kind]

	return tool, ok
}

// listGuide is one line per kind naming the filters its list takes, so the
// model can fill them without a second round to ask.
func (c *catalog) listGuide() string {
	var b strings.Builder
	for _, kind := range c.kinds {
		b.WriteString("- ")
		b.WriteString(kind)
		if filters := filterNames(c.lists[kind].ParamSchema()); len(filters) > 0 {
			b.WriteString(": ")
			b.WriteString(strings.Join(filters, ", "))
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

// pagingParams page through a list rather than filter it; the guide leaves
// them out, since one answer reads one page.
var pagingParams = map[string]struct{}{
	"after": {}, "cursor": {}, "limit": {}, "offset": {}, "fields": {}, "includeTotal": {},
}

// filterNames is a kind's filters as the guide names them: required ones
// first, then those with fixed values (written out when there are few), then
// the rest, paging left out.
func filterNames(schema map[string]any) []string {
	properties, _ := schema[toolschema.KeyProperties].(map[string]any)
	required := map[string]struct{}{}
	if names, ok := schema[toolschema.KeyRequired].([]string); ok {
		for _, name := range names {
			required[name] = struct{}{}
		}
	}

	type filter struct {
		rank int
		name string
		text string
	}
	filters := make([]filter, 0, len(properties))
	for name, raw := range properties {
		if _, paging := pagingParams[name]; paging {
			continue
		}
		property, _ := raw.(map[string]any)
		values := enumValues(property)
		entry := filter{rank: 2, name: name, text: name}
		if len(values) > 0 {
			entry.rank = 1
			if len(values) <= maxEnumValues {
				entry.text = name + " (" + strings.Join(values, "|") + ")"
			}
		}
		if _, ok := required[name]; ok {
			entry.rank = 0
		}
		filters = append(filters, entry)
	}
	slices.SortFunc(filters, func(a, b filter) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		return strings.Compare(a.name, b.name)
	})

	names := make([]string, 0, min(len(filters), maxFilters))
	for _, entry := range filters {
		if len(names) == maxFilters {
			break
		}
		names = append(names, entry.text)
	}

	return names
}

func enumValues(property map[string]any) []string {
	switch values := property[toolschema.KeyEnum].(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
