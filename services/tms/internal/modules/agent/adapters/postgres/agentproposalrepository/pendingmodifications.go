package agentproposalrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/zap"
)

// SetPendingModifications keeps, or clears, the wording a person changed on
// a proposal before approving it.
//
// Conditional on the proposal still being pending: an edit saved as someone
// else decides it would otherwise sit on a decided proposal, where nothing
// reads it, and the person would think it had gone with the approval.
func (r *repository) SetPendingModifications(
	ctx context.Context,
	req agentports.SetPendingModificationsRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		modifications := p.PendingModifications.ToNull()
		if len(req.Modifications) > 0 {
			modifications = p.PendingModifications.To(req.Modifications)
		}

		entity, err := r.updateOne(ctx, req.ID, p.Where(
			p.ID.Eq(req.ID),
			p.Status.Eq(agent.ProposalStatusPending),
		), modifications)
		if err != nil {
			r.l.Error("failed to save pending proposal modifications",
				zap.String("id", req.ID.String()), zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

// clearsPendingModifications says whether moving a proposal to a status
// ends its unapproved wording: every status but waiting on a decision and
// waiting out an approval's undo window.
func clearsPendingModifications(status agent.ProposalStatus) bool {
	return status != agent.ProposalStatusPending && status != agent.ProposalStatusApproving
}
