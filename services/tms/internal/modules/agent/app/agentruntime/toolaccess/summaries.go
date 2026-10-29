package toolaccess

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (s *Access) ToolSummaries(
	definition *agentdefinition.Definition,
) []agentturn.ToolSummary {
	return s.Summarize(definition, s.HeldTools(definition))
}

func (s *Access) Summarize(
	definition *agentdefinition.Definition,
	names []string,
) []agentturn.ToolSummary {
	summaries := make([]agentturn.ToolSummary, 0, len(names))

	for _, name := range names {
		if tool, ok := s.queryTools.Get(name); ok {
			summaries = append(summaries, agentturn.ToolSummary{
				Name:        tool.Name(),
				Description: tool.Description(),
				Query:       true,
				Lookup:      agenttoolports.EffectOf(tool) == agenttooltrust.ToolEffectLookup,
				Recipe:      recipeOf(tool),
			})
			continue
		}

		tool, ok := s.actionTools.Get(name)
		if !ok {
			continue
		}

		policy := tool.Policy()
		summaries = append(summaries, agentturn.ToolSummary{
			Name:                tool.Name(),
			Description:         tool.Description(),
			Tier:                agenttoolpolicy.StaticTier(definition, &policy),
			PersonalRunsUnasked: policy.PersonalRunsUnasked && !tierSetFor(definition, name),
			AlwaysProposes:      agenttoolpolicy.AlwaysProposes(definition, &policy),
			Escalates:           agenttoolpolicy.Escalates(definition, &policy),
			Recipe:              recipeOf(tool),
		})
	}

	return summaries
}

// tierSetFor reports a tool whose tier a person set on the agent, which
// takes away the personal exemption a tool like remember otherwise has.
func tierSetFor(definition *agentdefinition.Definition, name string) bool {
	_, set := definition.ToolTiers[name]

	return set
}

func recipeOf(tool any) []string {
	if recipe, ok := tool.(agenttoolports.RecipeTool); ok {
		return recipe.Recipe()
	}

	return nil
}
