package agenttoolpolicy

import (
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	QueryTools  agenttoolports.AgentQueryToolRegistry
	ActionTools agenttoolports.AgentToolRegistry
	Runtime     agenttoolports.RuntimeToolPolicies
}

type Catalog struct {
	ordered []agenttoolports.ToolPolicy
	byName  map[string]int
}

func NewCatalog(p Params) (*Catalog, error) {
	return Build(p.QueryTools.All(), p.ActionTools.All(), p.Runtime)
}

func Build(
	queries []agenttoolports.AgentQueryTool,
	actions []agenttoolports.AgentTool,
	runtime []agenttoolports.ToolPolicy,
) (*Catalog, error) {
	specs := Specs(queries, actions, runtime)
	if err := Validate(specs); err != nil {
		return nil, fmt.Errorf("agent tool policies are invalid: %w", err)
	}

	catalog := &Catalog{
		ordered: make([]agenttoolports.ToolPolicy, 0, len(specs)),
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
	queries []agenttoolports.AgentQueryTool,
	actions []agenttoolports.AgentTool,
	runtime []agenttoolports.ToolPolicy,
) []Spec {
	specs := make([]Spec, 0, len(queries)+len(actions)+len(runtime))
	for _, tool := range queries {
		specs = append(specs, Spec{ToolName: tool.Name(), Policy: tool.Policy()})
	}
	for _, tool := range actions {
		_, reports := tool.(agenttoolports.ToolResultReporter)
		specs = append(specs, Spec{ToolName: tool.Name(), Policy: tool.Policy(), Reports: reports})
	}
	for idx := range runtime {
		specs = append(specs, Spec{ToolName: runtime[idx].Name, Policy: runtime[idx]})
	}

	return specs
}

func (c *Catalog) Get(name string) (agenttoolports.ToolPolicy, bool) {
	idx, ok := c.byName[name]
	if !ok {
		return agenttoolports.ToolPolicy{}, false
	}

	return c.ordered[idx], true
}

func (c *Catalog) All() []agenttoolports.ToolPolicy {
	out := make([]agenttoolports.ToolPolicy, len(c.ordered))
	copy(out, c.ordered)

	return out
}

func ApplyOverride(
	policy agenttoolports.ToolPolicy,
	override *agent.ToolRuleOverride,
) agenttoolports.ToolPolicy {
	if override.Empty() {
		return policy
	}
	if override.MaxTier.IsValid() && policy.MaxTier.Above(override.MaxTier) {
		policy.MaxTier = override.MaxTier
		policy.DefaultTier = policy.DefaultTier.AtMost(override.MaxTier)
	}
	if override.ReadsExternal.IsValid() &&
		agent.ExternalReadRank(
			override.ReadsExternal,
		) > agent.ExternalReadRank(
			policy.ReadsExternal,
		) {
		policy.ReadsExternal = override.ReadsExternal
	}

	return policy
}

func ApplyOverrides(
	policy agenttoolports.ToolPolicy,
	overrides map[string]*agent.ToolRuleOverride,
) agenttoolports.ToolPolicy {
	return ApplyOverride(policy, overrides[policy.Name])
}
