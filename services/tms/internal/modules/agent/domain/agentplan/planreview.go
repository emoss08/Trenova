package agentplan

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
)

func (p *AgentPlan) FromScript() bool {
	return p.ScriptStepKey != ""
}

func PlanNeedsReview(steps []*agent.AgentProposal) bool {
	return slices.ContainsFunc(steps, func(step *agent.AgentProposal) bool {
		return step != nil && step.Status == agent.ProposalStatusPending &&
			step.ApprovalNeedsReview()
	})
}
