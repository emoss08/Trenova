package proposaloutcomes

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// DecisionsFor reads the decision behind each decided proposal, keyed by
// proposal: what the approver changed, who decided it, when, and what they
// told the agent. A read that fails degrades to showing none: the
// proposal's status is the record and the card still says how it ended.
func (s *Reader) DecisionsFor(
	ctx context.Context,
	stored []*agent.AgentProposal,
	tenant pagination.TenantInfo,
) map[pulid.ID]*agentdecision.AgentDecision {
	if s.Decisions == nil {
		return nil
	}

	ids := make([]pulid.ID, 0, len(stored))
	for _, proposal := range stored {
		if proposal != nil && proposal.Status != agent.ProposalStatusPending {
			ids = append(ids, proposal.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	decisions, err := s.Decisions.ListByProposals(
		ctx,
		agentports.ListAgentDecisionsByProposalsRequest{
			ProposalIDs: ids,
			TenantInfo:  tenant,
		},
	)
	if err != nil {
		s.Logger.Warn("could not read the decisions behind the thread's proposals", zap.Error(err))

		return nil
	}

	out := make(map[pulid.ID]*agentdecision.AgentDecision, len(decisions))
	for _, decision := range decisions {
		if decision == nil || decision.ProposalID == nil {
			continue
		}
		if current, ok := out[*decision.ProposalID]; ok && current.CreatedAt > decision.CreatedAt {
			continue
		}
		out[*decision.ProposalID] = decision
	}

	return out
}

// Outcomes reads what became of every proposal this conversation
// raised, for the model to see the current state of each. A read that fails
// degrades to the stored tool results rather than failing the turn: a stale
// "awaiting review" is what the model always used to see.
func (s *Reader) Outcomes(
	ctx context.Context,
	thread *conversation.Thread,
	tenant pagination.TenantInfo,
) []agentruntimeports.ProposalOutcome {
	if s.Proposals == nil {
		return nil
	}

	stored, err := s.Proposals.ListByThread(ctx, agentports.ListAgentProposalsByThreadRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.Logger.Warn("could not read the thread's proposals for the model",
			zap.String("thread", thread.ID.String()), zap.Error(err))

		return nil
	}

	decided := s.DecisionsFor(ctx, stored, tenant)

	outcomes := make([]agentruntimeports.ProposalOutcome, 0, len(stored))
	for _, proposal := range stored {
		if proposal == nil {
			continue
		}
		planID := pulid.Nil
		if proposal.PlanID != nil {
			planID = *proposal.PlanID
		}
		outcomes = append(outcomes, agentruntimeports.ProposalOutcome{
			ProposalID:      proposal.ID,
			PlanID:          planID,
			SourceMessageID: proposal.SourceMessageID,
			ToolName:        proposal.ToolName,
			ToolParams:      proposal.ToolParams,
			Rationale:       proposal.Rationale,
			Status:          proposal.Status,
			AutonomyTier:    proposal.AutonomyTier,
			ExecutionError:  proposal.ExecutionError,
			ExecutedAt:      proposal.ExecutedAt,
			ExecutionResult: proposal.ExecutionResult,
			Modifications:   proposalstores.ModificationsOf(decided[proposal.ID]),
		})
	}

	return outcomes
}
