package assistantproposals

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/shared/pulid"
)

// runsOf is the run behind each proposal.
func runsOf(stored []*agent.AgentProposal) []pulid.ID {
	ids := make([]pulid.ID, 0, len(stored))
	for _, proposal := range stored {
		if proposal != nil {
			ids = append(ids, proposal.RunID)
		}
	}

	return ids
}

func stepRuns(steps []*agent.AgentProposal) []pulid.ID {
	runs := make([]pulid.ID, 0, len(steps))
	for _, step := range steps {
		if !slices.Contains(runs, step.RunID) {
			runs = append(runs, step.RunID)
		}
	}

	return runs
}

func planVerdict(
	verdicts map[pulid.ID]agentshadow.Verdict,
	plan *agentplan.AgentPlan,
	stepRuns []pulid.ID,
) agentshadow.Verdict {
	if !plan.Status.Decidable() {
		return agentshadow.Verdict{}
	}
	return agentshadow.PlanVerdict(verdicts, plan.RunID, stepRuns)
}

func toAssistantPlan(
	plan *agentplan.AgentPlan,
	hold *assistantproposalports.ProposalHold,
) assistantproposalports.AssistantPlan {
	decidedBy := pulid.Nil
	if plan.DecidedByUserID != nil {
		decidedBy = *plan.DecidedByUserID
	}

	return assistantproposalports.AssistantPlan{
		ID:              plan.ID,
		RunID:           plan.RunID,
		Title:           plan.Title,
		Summary:         plan.Summary,
		Status:          plan.Status,
		StepCount:       plan.StepCount,
		CompletedSteps:  plan.CompletedSteps,
		FailedStep:      plan.FailedStep,
		FailureError:    plan.FailureError,
		DecidedAt:       plan.DecidedAt,
		DecidedByUserID: decidedBy,
		DecisionNote:    plan.ApprovalNote,
		PreviewDigest:   plan.ApprovalPreviewDigest,
		ExpiresAt:       plan.ExpiresAt,
		CommitsAt:       plan.CommitsAt,
		UndoneAt:        plan.UndoneAt,
		RunMode:         agentplan.PlanRunModeInOrder,
		Hold:            hold,
		CreatedAt:       plan.CreatedAt,
		Script:          plan.Script,
	}
}
