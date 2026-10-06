package agentdecisionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AgentDecisionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdecision-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentDecisionByIDRequest,
) (*agent.AgentDecision, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.AgentDecision, error) {
		log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

		entity := new(agent.AgentDecision)
		cols := buncolgen.AgentDecisionColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentDecisionScopeTenant(sq, *req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get agent decision", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "AgentDecision")
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentDecision,
) (*agent.AgentDecision, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentDecision, error) {
		log := r.l.With(zap.String("operation", "Create"))

		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to create agent decision", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ListByProposals(
	ctx context.Context,
	req repositories.ListAgentDecisionsByProposalsRequest,
) ([]*agent.AgentDecision, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentDecision, error) {
		if len(req.ProposalIDs) == 0 {
			return []*agent.AgentDecision{}, nil
		}

		cols := buncolgen.AgentDecisionColumns
		rows := make([]*agent.AgentDecision, 0, len(req.ProposalIDs))

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentDecisionScopeTenant(sq, req.TenantInfo).
					Where(cols.ProposalID.In(), bun.In(req.ProposalIDs)).
					Where(cols.UndoneAt.IsNull())
			}).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			Scan(ctx); err != nil {
			r.l.Error("failed to list decisions by proposals", zap.Error(err))

			return nil, fmt.Errorf("list decisions by proposals: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ListByCommitWorkflow(
	ctx context.Context,
	req repositories.ListAgentDecisionsByCommitWorkflowRequest,
) ([]*agent.AgentDecision, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentDecision, error) {
		if req.WorkflowID == "" {
			return []*agent.AgentDecision{}, nil
		}

		cols := buncolgen.AgentDecisionColumns
		rows := make([]*agent.AgentDecision, 0, 1)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentDecisionScopeTenant(sq, req.TenantInfo).
					Where(cols.CommitWorkflowID.Eq(), req.WorkflowID)
			}).
			OrderExpr(cols.CreatedAt.OrderAsc()).
			OrderExpr(cols.ID.OrderAsc()).
			Scan(ctx); err != nil {
			r.l.Error("failed to list decisions by commit workflow", zap.Error(err))

			return nil, fmt.Errorf("list decisions by commit workflow: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) MarkCommitted(
	ctx context.Context,
	req repositories.SettleAgentDecisionsRequest,
) ([]pulid.ID, error) {
	return r.settle(ctx, req, false)
}

func (r *repository) MarkUndone(
	ctx context.Context,
	req repositories.SettleAgentDecisionsRequest,
) ([]pulid.ID, error) {
	return r.settle(ctx, req, true)
}

// settle closes the window on the decisions still in it, in one statement:
// a commit and an undo racing each other each take all of a batch or none.
func (r *repository) settle(
	ctx context.Context,
	req repositories.SettleAgentDecisionsRequest,
	undo bool,
) ([]pulid.ID, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if len(req.IDs) == 0 {
			return []pulid.ID{}, nil
		}

		cols := buncolgen.AgentDecisionColumns
		query := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.AgentDecision)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.AgentDecisionScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.In(), bun.In(req.IDs)).
					Where(cols.CommitsAt.IsNotNull()).
					Where(cols.CommittedAt.IsNull()).
					Where(cols.UndoneAt.IsNull())
			}).
			Set(cols.UpdatedAt.Set(), req.At)
		if undo {
			query = query.
				Set(cols.UndoneAt.Set(), req.At).
				Set(cols.UndoneByUserID.Set(), req.UndoneByUserID)
		} else {
			query = query.Set(cols.CommittedAt.Set(), req.At)
		}

		settled := make([]pulid.ID, 0, len(req.IDs))
		if _, err := query.Returning(cols.ID.Bare()).Exec(ctx, &settled); err != nil {
			r.l.Error("failed to settle agent decisions", zap.Bool("undo", undo), zap.Error(err))

			return nil, fmt.Errorf("settle agent decisions: %w", err)
		}

		return settled, nil
	})
}
