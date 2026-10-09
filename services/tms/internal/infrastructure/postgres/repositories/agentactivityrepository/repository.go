package agentactivityrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) repositories.AgentActivityRepository {
	return &repository{db: p.DB}
}

func workingStatuses() []agent.RunStatus {
	return []agent.RunStatus{
		agent.RunStatusPending,
		agent.RunStatusGatheringContext,
		agent.RunStatusDiagnosing,
	}
}

func openResolutionStates() []agent.ResolutionState {
	return []agent.ResolutionState{agent.ResolutionStateOpen, agent.ResolutionStateInReview}
}

func (r *repository) Totals(
	ctx context.Context,
	req repositories.AgentActivityTotalsRequest,
) (*repositories.AgentActivityTotals, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*repositories.AgentActivityTotals, error) {
		totals := new(repositories.AgentActivityTotals)
		dba := r.db.DBForContext(ctx)

		runs := buncolgen.AgentRunColumns
		if err := dba.NewSelect().
			Model((*agent.AgentRun)(nil)).
			ColumnExpr(buncolgen.CountFilter("runs", runs.CreatedAt.Gte()), req.RunsSince).
			ColumnExpr(
				buncolgen.CountFilter("runs_failed", runs.CreatedAt.Gte(), runs.Status.Eq()),
				req.RunsSince,
				agent.RunStatusFailed,
			).
			ColumnExpr(buncolgen.CountFilter("runs_working", runs.Status.In()), bun.List(workingStatuses())).
			ColumnExpr(
				buncolgen.CountFilter("runs_awaiting", runs.Status.Eq()),
				agent.RunStatusAwaitingDecision,
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentRunScopeTenant(sq, req.TenantInfo)
			}).
			Scan(ctx, totals); err != nil {
			return nil, fmt.Errorf("count agent runs: %w", err)
		}

		proposals := buncolgen.AgentProposalColumns
		if err := dba.NewSelect().
			Model((*agent.AgentProposal)(nil)).
			ColumnExpr("COUNT(*) AS pending_proposals").
			ColumnExpr(buncolgen.Min(proposals.CreatedAt, "oldest_pending_at")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentProposalScopeTenant(sq, req.TenantInfo).
					Where(proposals.Status.Eq(), agent.ProposalStatusPending)
			}).
			Scan(ctx, totals); err != nil {
			return nil, fmt.Errorf("count pending agent proposals: %w", err)
		}

		exceptions := buncolgen.AgentExceptionColumns
		if err := dba.NewSelect().
			Model((*agent.AgentException)(nil)).
			ColumnExpr("COUNT(*) AS open_exceptions").
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentExceptionScopeTenant(sq, req.TenantInfo).
					Where(exceptions.ResolutionState.In(), bun.List(openResolutionStates()))
			}).
			Scan(ctx, totals); err != nil {
			return nil, fmt.Errorf("count open agent exceptions: %w", err)
		}

		decisions := buncolgen.AgentDecisionColumns
		if err := dba.NewSelect().
			Model((*agent.AgentDecision)(nil)).
			ColumnExpr(
				buncolgen.CountFilter("decisions_accepted", decisions.Decision.Eq()),
				agent.DecisionAccepted,
			).
			ColumnExpr(
				buncolgen.CountFilter("decisions_modified", decisions.Decision.Eq()),
				agent.DecisionModified,
			).
			ColumnExpr(
				buncolgen.CountFilter("decisions_rejected", decisions.Decision.Eq()),
				agent.DecisionRejected,
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentDecisionScopeTenant(sq, req.TenantInfo).
					Where(decisions.ProposalID.IsNotNull()).
					Where(decisions.UndoneAt.IsNull()).
					Where(decisions.CreatedAt.Gte(), req.DecisionsSince)
			}).
			Scan(ctx, totals); err != nil {
			return nil, fmt.Errorf("count agent decisions: %w", err)
		}

		return totals, nil
	})
}
