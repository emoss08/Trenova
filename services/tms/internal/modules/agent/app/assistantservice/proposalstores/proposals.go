package proposalstores

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
)

type ProposalStore interface {
	ListByThread(
		ctx context.Context,
		req agentports.ListAgentProposalsByThreadRequest,
	) ([]*agent.AgentProposal, error)
	ListByRun(
		ctx context.Context,
		req agentports.ListAgentProposalsByRunRequest,
	) ([]*agent.AgentProposal, error)
	ExpirePendingByThread(
		ctx context.Context,
		req agentports.ExpireAgentProposalsByThreadRequest,
	) (int, error)
}

func ModificationsOf(decision *agentdecision.AgentDecision) map[string]any {
	if decision == nil || decision.Decision != agentdecision.DecisionModified ||
		len(decision.Modifications) == 0 {
		return nil
	}

	return decision.Modifications
}

type PlanStore interface {
	ListByThread(
		ctx context.Context,
		req agentports.ListAgentPlansByThreadRequest,
	) ([]*agentplan.AgentPlan, error)
	ExpirePendingByThread(
		ctx context.Context,
		req agentports.ExpireAgentPlansByThreadRequest,
	) (int, error)
}

// PlansOrNil keeps a typed nil out of the interface, so the absence of a
// plan store reads as nil rather than as a store that panics.
func PlansOrNil(plans agentports.AgentPlanRepository) PlanStore {
	if plans == nil {
		return nil
	}

	return plans
}

func ToAssistantProposal(
	proposal *agent.AgentProposal,
	hold *assistantproposalports.ProposalHold,
) assistantproposalports.AssistantProposal {
	out := assistantproposalports.AssistantProposal{
		ID:              proposal.ID,
		RunID:           proposal.RunID,
		ToolName:        proposal.ToolName,
		Arguments:       proposal.ToolParams,
		Rationale:       proposal.Rationale,
		AutonomyTier:    proposal.AutonomyTier,
		Status:          proposal.Status,
		SourceMessageID: proposal.SourceMessageID,
		Confidence:      proposal.Confidence.InexactFloat64(),
		ExecutedAt:      proposal.ExecutedAt,
		ExecutionError:  proposal.ExecutionError,
		ExecutionResult: proposal.ExecutionResult,
		ExpiresAt:       proposal.ExpiresAt,
		Hold:            hold,
		PlanStep:        proposal.PlanStep,
		SimulatedAt:     proposal.SimulatedAt,
		Simulation:      proposal.Simulation,
		CreatedAt:       proposal.CreatedAt,
		ReviewRequired:  proposal.ApprovalNeedsReview(),
	}
	if proposal.PlanID != nil {
		out.PlanID = *proposal.PlanID
	}
	if proposal.Status == agent.ProposalStatusPending && len(proposal.PendingModifications) > 0 {
		out.PendingModifications = proposal.PendingModifications
	}

	return out
}
