package agentproposalrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/agentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// RecordExecution stores the outcome of running an approved proposal.
//
// Every outcome column is always written, including on success where the
// error is cleared and on failure where the result is: a proposal that failed,
// was re-approved, and then succeeded must not keep showing the old failure,
// nor a failed one what an earlier run made.
func (r *repository) RecordExecution(
	ctx context.Context,
	req agentports.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		log := r.l.With(
			zap.String("operation", "RecordExecution"),
			zap.String("id", req.ID.String()),
		)

		entity := new(agent.AgentProposal)
		cols := agentcol.AgentProposalColumns

		query := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return agentcol.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Set(cols.Status.Set(), req.Status).
			Set(cols.ExecutedAt.Set(), req.ExecutedAt).
			Set(cols.ExecutionError.Set(), req.ExecutionError).
			Set(cols.ExecutionResult.Set(), req.ExecutionResult.Bounded()).
			Set(cols.ExecutedTargetVersion.Set(), req.ExecutedTargetVersion).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix())
		if req.EgressClass.IsValid() {
			query = query.Set(cols.EgressClass.Set(), req.EgressClass)
		}
		if req.ExecutedByUserID.IsNotNil() {
			query = query.Set(cols.ExecutedByUserID.Set(), req.ExecutedByUserID)
		}

		results, err := query.Returning("*").Exec(ctx)
		if err != nil {
			log.Error("failed to record proposal execution", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(results, "AgentProposal", req.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

// ListByThread returns every proposal raised during one assistant conversation.
//
// The link runs through the agent run rather than the proposal, because a
// proposal belongs to a run and a chat run's subject is the thread. Filtering on
// the subject type as well as the id keeps a thread id that happens to collide
// with some other subject's id from matching.
func (r *repository) ListByThread(
	ctx context.Context,
	req agentports.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		log := r.l.With(
			zap.String("operation", "ListByThread"),
			zap.String("threadId", req.ThreadID.String()),
		)

		entities := make([]*agent.AgentProposal, 0)
		cols := agentcol.AgentProposalColumns
		runCols := agentcol.AgentRunColumns

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Join("JOIN agent_runs AS ar ON "+
				runCols.ID.EqColumn(cols.RunID)+" AND "+
				runCols.OrganizationID.EqColumn(cols.OrganizationID)+" AND "+
				runCols.BusinessUnitID.EqColumn(cols.BusinessUnitID),
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentcol.AgentProposalScopeTenant(sq, req.TenantInfo).
					Where(runCols.SubjectType.Eq(), agent.SubjectAssistantThread).
					Where(runCols.SubjectID.Eq(), req.ThreadID)
			}).
			Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx)
		if err != nil {
			log.Error("failed to list proposals for thread", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

// RecordSimulation stores a preview in place of an execution. The proposal
// moves to Simulated whatever it was, because the decision that cleared it
// has been made and this is what came of it.
func (r *repository) RecordSimulation(
	ctx context.Context,
	req agentports.RecordAgentProposalSimulationRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		entity := new(agent.AgentProposal)
		cols := agentcol.AgentProposalColumns

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return agentcol.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Set(cols.Status.Set(), agent.ProposalStatusSimulated).
			Set(cols.SimulatedAt.Set(), req.SimulatedAt).
			Set(cols.Simulation.Set(), req.Simulation).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				if req.ExecutedByUserID.IsNil() {
					return uq
				}

				return uq.Set(cols.ExecutedByUserID.Set(), req.ExecutedByUserID)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to record proposal simulation",
				zap.String("id", req.ID.String()), zap.Error(err))

			return nil, fmt.Errorf("record proposal simulation: %w", err)
		}

		if err = dberror.CheckRowsAffected(results, "AgentProposal", req.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

// CountExecutedTool counts one agent's executions of one tool since an
// instant, through the run that made each proposal.
func (r *repository) CountExecutedTool(
	ctx context.Context,
	req agentports.CountExecutedToolRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		proposals := agentcol.AgentProposalColumns
		runs := agentcol.AgentRunColumns

		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*agent.AgentProposal)(nil)).
			Join("JOIN ? AS ? ON ? = ? AND ? = ? AND ? = ?",
				bun.Ident(agentcol.AgentRunTable.Name),
				bun.Ident(agentcol.AgentRunTable.Alias),
				bun.Safe(runs.ID.Qualified()),
				bun.Safe(proposals.RunID.Qualified()),
				bun.Safe(runs.OrganizationID.Qualified()),
				bun.Safe(proposals.OrganizationID.Qualified()),
				bun.Safe(runs.BusinessUnitID.Qualified()),
				bun.Safe(proposals.BusinessUnitID.Qualified()),
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentcol.AgentProposalScopeTenant(sq, req.TenantInfo).
					Where(runs.AgentDefinitionID.Eq(), req.DefinitionID).
					Where(proposals.ToolName.Eq(), req.ToolName).
					Where(proposals.ExecutedAt.Gte(), req.Since).
					Where(proposals.ExecutionError.IsNull())
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count executed tool proposals", zap.Error(err))

			return 0, fmt.Errorf("count executed tool proposals: %w", err)
		}

		return int(count), nil
	})
}

type automaticMoneyRow struct {
	Calls       int   `bun:"calls"`
	AmountMinor int64 `bun:"amount_minor"`
}

var errMoneyCurrencyRequired = errors.New("sum automatic money writes: a currency is required")

func (r *repository) SumAutomaticMoney(
	ctx context.Context,
	req agentports.SumAutomaticMoneyRequest,
) (agent.MoneyUse, error) {
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		return agent.MoneyUse{}, errMoneyCurrencyRequired
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (agent.MoneyUse, error) {
		cols := agentcol.AgentProposalColumns
		status, at := agent.ProposalStatusExecuted, cols.ExecutedAt
		if req.Simulated {
			status, at = agent.ProposalStatusSimulated, cols.SimulatedAt
		}

		row := new(automaticMoneyRow)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*agent.AgentProposal)(nil)).
			ColumnExpr(buncolgen.Count("calls")).
			ColumnExpr(cols.MoneyMinor.Expr("COALESCE(SUM(ABS({})), 0) AS amount_minor")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = agentcol.AgentProposalScopeTenant(sq, req.TenantInfo).
					Where(cols.AutonomyTier.Eq(), agent.TierAutoExecute).
					Where(cols.Status.Eq(), status).
					Where(cols.MoneyMinor.IsNotNull()).
					Where(cols.MoneyCurrency.Eq(), currency).
					Where(at.IsNotNull())
				if req.Since > 0 {
					sq = sq.Where(at.Gte(), req.Since)
				}
				if req.RunID.IsNotNil() {
					sq = sq.Where(cols.RunID.Eq(), req.RunID)
				}

				return sq
			}).
			Scan(ctx, row)
		if err != nil {
			r.l.Error("failed to sum automatic money writes", zap.Error(err))

			return agent.MoneyUse{}, fmt.Errorf("sum automatic money writes: %w", err)
		}

		return agent.MoneyUse{Calls: row.Calls, AmountMinor: row.AmountMinor}, nil
	})
}
