package runtimetools

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const NamePlanName = "name_plan"

const namePlanDescription = "Name the plan this turn's proposals are approved as. Two or " +
	"more proposals filed in one turn reach the person as one plan, approved whole and run " +
	"in order; its name is the line they read above the steps. Give a short phrase that says " +
	"what the changes do together, e.g. \"Recover S-240201 after the Tyson dock delay\", not " +
	"\"4 changes\" and not a list of the tools. Call it once, after filing the proposals. A " +
	"turn that files one proposal or none makes no plan, so the name is not used. Naming " +
	"changes no record."

const (
	paramPlanTitle  = "title"
	UnofferedNaming = "name_plan is only for a turn that files proposals, and this agent " +
		"holds no tool that does. Answer without it."
	DelegatedNaming = "A task handed to you cannot name the plan: the agent that handed it " +
		"to you names it."
)

func NamePlanSpec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        NamePlanName,
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

func namePlan(t Turn, arguments map[string]any) toolcall.Result {
	title, err := agentplan.NamedPlanTitle(argcheck.String(arguments, paramPlanTitle))
	if err != nil {
		return toolcall.Failed("Tool %q was not run: %s", NamePlanName, err.Error())
	}

	t.Result().PlanTitle = title
	content := fmt.Sprintf("The plan this turn files will be named %q.", title)
	if waiting := waitingCount(t); waiting < 2 {
		content += " It is used only if this turn files two or more proposals."
	}

	return toolcall.Result{
		Content: content + " Nothing has changed.",
		Summary: toolsummary.Line("Named the plan: " + title),
	}
}
