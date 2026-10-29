package agentruntime

import (
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const namePlanName = "name_plan"

const namePlanDescription = "Name the plan this turn's proposals are approved as. Two or " +
	"more proposals filed in one turn reach the person as one plan, approved whole and run " +
	"in order; its name is the line they read above the steps. Give a short phrase that says " +
	"what the changes do together, e.g. \"Recover S-240201 after the Tyson dock delay\", not " +
	"\"4 changes\" and not a list of the tools. Call it once, after filing the proposals. A " +
	"turn that files one proposal or none makes no plan, so the name is not used. Naming " +
	"changes no record."

const (
	paramPlanTitle  = "title"
	unofferedNaming = "name_plan is only for a turn that files proposals, and this agent " +
		"holds no tool that does. Answer without it."
	delegatedNaming = "A task handed to you cannot name the plan: the agent that handed it " +
		"to you names it."
)

func namePlanSpec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        namePlanName,
		Description: namePlanDescription,
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramPlanTitle: map[string]any{
					toolschema.KeyType:      toolschema.TypeString,
					toolschema.KeyMaxLength: agentplan.MaxNamedPlanTitleLength,
					toolschema.KeyDescription: fmt.Sprintf("The plan's name: one line of plain "+
						"text, at most %d characters, no markdown.",
						agentplan.MaxNamedPlanTitleLength),
				},
			},
			toolschema.KeyRequired:             []string{paramPlanTitle},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func (s *Service) declareNamePlan(set *toolSet, delegated bool) {
	if delegated || !slices.ContainsFunc(set.allowed, s.filesProposals) {
		return
	}

	set.add(namePlanSpec())
}

func (t *Turn) namePlan(arguments map[string]any) toolOutcome {
	title, err := agentplan.NamedPlanTitle(stringArg(arguments, paramPlanTitle))
	if err != nil {
		return failedOutcome("Tool %q was not run: %s", namePlanName, err.Error())
	}

	t.result.PlanTitle = title
	content := fmt.Sprintf("The plan this turn files will be named %q.", title)
	if waiting := t.waitingCount(); waiting < 2 {
		content += " It is used only if this turn files two or more proposals."
	}

	return toolOutcome{
		content: content + " Nothing has changed.",
		summary: summaryLine("Named the plan: " + title),
	}
}
