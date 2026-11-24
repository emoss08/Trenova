package agenttoolcatalog

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func CapabilityFacts(
	queries agenttoolports.AgentQueryToolRegistry,
	actions agenttoolports.AgentToolRegistry,
	runtime agenttoolports.RuntimeToolPolicies,
) []agentcapability.Facts {
	queryTools := queries.All()
	actionTools := actions.All()
	facts := make([]agentcapability.Facts, 0, len(queryTools)+len(actionTools)+len(runtime))
	for _, tool := range queryTools {
		facts = append(facts, factsOf(agenttoolports.DescribeTool(tool, "", true), tool.Policy()))
	}
	for _, tool := range actionTools {
		facts = append(facts, factsOf(agenttoolports.DescribeTool(tool, "", false), tool.Policy()))
	}
	for idx := range runtime {
		facts = append(facts, factsOf(agenttoolports.AgentToolDescriptor{Name: runtime[idx].Name}, runtime[idx]))
	}

	return facts
}

func BuildCapabilities(
	queries agenttoolports.AgentQueryToolRegistry,
	actions agenttoolports.AgentToolRegistry,
	runtime agenttoolports.RuntimeToolPolicies,
) (*agentcapability.Registry, error) {
	return agentcapability.Build(CapabilityFacts(queries, actions, runtime))
}

func factsOf(
	descriptor agenttoolports.AgentToolDescriptor,
	policy agenttoolports.ToolPolicy,
) agentcapability.Facts {
	effect := policy.EffectiveEffect()

	return agentcapability.Facts{
		Name:        descriptor.Name,
		Description: descriptor.Description,
		Effect:      string(effect),
		Write:       effect == agenttooltrust.ToolEffectChange,
		Turn:        policy.Kind == agenttooltrust.ToolKindRuntime,
		Approval:    string(policy.DefaultTier),
		MaxApproval: string(policy.MaxTier),
		Resource:    string(policy.Resource),
		Operation:   string(policy.Operation),
		SearchTerms: descriptor.SearchTerms,
		Recipe:      descriptor.Recipe,
	}
}
