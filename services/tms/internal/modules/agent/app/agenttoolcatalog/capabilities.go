package agenttoolcatalog

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/capabilitysearch"
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

func CapabilityTexts(
	registry *agentcapability.Registry,
	queries agenttoolports.AgentQueryToolRegistry,
	actions agenttoolports.AgentToolRegistry,
) map[string]capabilitysearch.Text {
	descriptors := make(map[string]agenttoolports.AgentToolDescriptor, 1024)
	for _, descriptor := range allDescriptors(queries, actions) {
		descriptors[descriptor.Name] = descriptor
	}
	textOf := func(name string) capabilitysearch.Text {
		descriptor, ok := descriptors[name]
		if !ok {
			return capabilitysearch.Text{}
		}

		return capabilitysearch.Text{
			Body: descriptor.Description + " " + strings.Join(descriptor.SearchTerms, " ") + " " +
				strings.ReplaceAll(agenttoolports.ReadingOrder(name), "_", " "),
			Params: parameterText(descriptor.Parameters),
		}
	}

	texts := make(map[string]capabilitysearch.Text, len(descriptors))
	for _, capability := range registry.All() {
		if !capability.IsEntityRead() {
			texts[capability.ID] = textOf(capability.Name)
			continue
		}
		entity, ok := agentcapability.EntityNamed(capability.Entity)
		if !ok {
			continue
		}
		backing := entity.List
		if capability.Verb == agentcapability.VerbGet {
			backing = entity.Get
		}
		text := textOf(backing)
		text.Params = ""
		texts[capability.ID] = text
	}

	return texts
}

type CapabilityIndex struct {
	Registry *agentcapability.Registry
	Index    *capabilitysearch.Index
	Keyword  *Catalog
}

func BuildCapabilityIndex(
	queries agenttoolports.AgentQueryToolRegistry,
	actions agenttoolports.AgentToolRegistry,
	runtime agenttoolports.RuntimeToolPolicies,
) (*CapabilityIndex, error) {
	registry, err := BuildCapabilities(queries, actions, runtime)
	if err != nil {
		return nil, err
	}

	return &CapabilityIndex{
		Registry: registry,
		Index:    capabilitysearch.New(registry.All(), CapabilityTexts(registry, queries, actions)),
		Keyword:  New(allDescriptors(queries, actions)),
	}, nil
}

func (c *CapabilityIndex) Search(text string, semantic map[string]float64) capabilitysearch.Answer {
	return c.Index.Search(capabilitysearch.Query{
		Text:     text,
		Limit:    capabilitysearch.DefaultLimit,
		Keyword:  c.Keyword.Scores(text),
		Semantic: semantic,
	})
}

func allDescriptors(
	queries agenttoolports.AgentQueryToolRegistry,
	actions agenttoolports.AgentToolRegistry,
) []agenttoolports.AgentToolDescriptor {
	queryTools := queries.All()
	actionTools := actions.All()
	out := make([]agenttoolports.AgentToolDescriptor, 0, len(queryTools)+len(actionTools))
	for _, tool := range queryTools {
		out = append(out, agenttoolports.DescribeTool(tool, "", true))
	}
	for _, tool := range actionTools {
		out = append(out, agenttoolports.DescribeTool(tool, tool.Policy().DefaultTier, false))
	}

	return out
}
