package agentproposalrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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
	req repositories.SetPendingModificationsRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		log := r.l.With(
			zap.String("operation", "SetPendingModifications"),
			zap.String("id", req.ID.String()),
		)

		entity := new(agent.AgentProposal)
		cols := buncolgen.AgentProposalColumns

		query := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Status.Eq(), agent.ProposalStatusPending)
			}).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix())
		if len(req.Modifications) == 0 {
			query = query.Set(cols.PendingModifications.SetNull())
		} else {
			query = query.Set(cols.PendingModifications.Set(), req.Modifications)
		}

		results, err := query.Returning("*").Exec(ctx)
		if err != nil {
			log.Error("failed to save pending proposal modifications", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(results, "AgentProposal", req.ID.String()); err != nil {
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
