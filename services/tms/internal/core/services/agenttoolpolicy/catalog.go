package agenttoolpolicy

import (
	"fmt"
	"sort"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	QueryTools  serviceports.AgentQueryToolRegistry
	ActionTools serviceports.AgentToolRegistry
	Runtime     serviceports.RuntimeToolPolicies
}

type Catalog struct {
	ordered []serviceports.ToolPolicy
	byName  map[string]int
}

func NewCatalog(p Params) (*Catalog, error) {
	return Build(p.QueryTools.All(), p.ActionTools.All(), p.Runtime)
}

func Build(
	queries []serviceports.AgentQueryTool,
	actions []serviceports.AgentTool,
	runtime []serviceports.ToolPolicy,
) (*Catalog, error) {
	specs := Specs(queries, actions, runtime)
	if err := Validate(specs); err != nil {
		return nil, fmt.Errorf("agent tool policies are invalid: %w", err)
	}

	catalog := &Catalog{
		ordered: make([]serviceports.ToolPolicy, 0, len(specs)),
		byName:  make(map[string]int, len(specs)),
	}
	for idx := range specs {
		catalog.ordered = append(catalog.ordered, specs[idx].Policy)
	}
	sort.SliceStable(catalog.ordered, func(i, j int) bool {
		return catalog.ordered[i].Name < catalog.ordered[j].Name
	})
	for idx := range catalog.ordered {
		catalog.byName[catalog.ordered[idx].Name] = idx
	}

	return catalog, nil
}

func Specs(
	queries []serviceports.AgentQueryTool,
	actions []serviceports.AgentTool,
	runtime []serviceports.ToolPolicy,
) []Spec {
	specs := make([]Spec, 0, len(queries)+len(actions)+len(runtime))
	for _, tool := range queries {
		specs = append(specs, Spec{ToolName: tool.Name(), Policy: tool.Policy()})
	}
	for _, tool := range actions {
		_, reports := tool.(serviceports.ToolResultReporter)
		specs = append(specs, Spec{ToolName: tool.Name(), Policy: tool.Policy(), Reports: reports})
	}
	for idx := range runtime {
		specs = append(specs, Spec{ToolName: runtime[idx].Name, Policy: runtime[idx]})
	}

	return specs
}

func (c *Catalog) Get(name string) (serviceports.ToolPolicy, bool) {
	idx, ok := c.byName[name]
	if !ok {
		return serviceports.ToolPolicy{}, false
	}

	return c.ordered[idx], true
}

func (c *Catalog) All() []serviceports.ToolPolicy {
	out := make([]serviceports.ToolPolicy, len(c.ordered))
	copy(out, c.ordered)

	return out
}
