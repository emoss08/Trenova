package agentdecisionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision/agentdecisiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var decisionErrors = dberror.Mapping{Entity: "AgentDecision"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentdecisionports.AgentDecisionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdecision-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentdecisionports.GetAgentDecisionByIDRequest,
) (*agentdecision.AgentDecision, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentdecision.AgentDecision, error) {
		entity, err := agentdecisiondb.AgentDecisions.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			r.l.Error(
				"failed to get agent decision",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return nil, decisionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentdecision.AgentDecision,
) (*agentdecision.AgentDecision, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentdecision.AgentDecision, error) {
		if err := agentdecisiondb.AgentDecisions.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create agent decision", zap.Error(err))
			return nil, decisionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByProposals(
	ctx context.Context,
	req agentdecisionports.ListAgentDecisionsByProposalsRequest,
) ([]*agentdecision.AgentDecision, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdecision.AgentDecision, error) {
		if len(req.ProposalIDs) == 0 {
			return []*agentdecision.AgentDecision{}, nil
		}

		d := agentdecisiondb.AgentDecisions
		rows, err := d.Where(d.ProposalID.In(req.ProposalIDs...), d.UndoneAt.IsNull()).
			Order(d.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list decisions by proposals", zap.Error(err))

			return nil, fmt.Errorf("list decisions by proposals: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ListByCommitWorkflow(
	ctx context.Context,
	req agentdecisionports.ListAgentDecisionsByCommitWorkflowRequest,
) ([]*agentdecision.AgentDecision, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdecision.AgentDecision, error) {
		if req.WorkflowID == "" {
			return []*agentdecision.AgentDecision{}, nil
		}

		d := agentdecisiondb.AgentDecisions
		rows, err := d.Where(d.CommitWorkflowID.Eq(req.WorkflowID)).
			Order(d.CreatedAt.Asc(), d.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list decisions by commit workflow", zap.Error(err))

			return nil, fmt.Errorf("list decisions by commit workflow: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) MarkCommitted(
	ctx context.Context,
	req agentdecisionports.SettleAgentDecisionsRequest,
) ([]pulid.ID, error) {
	return r.settle(ctx, req, false)
}

func (r *repository) MarkUndone(
	ctx context.Context,
	req agentdecisionports.SettleAgentDecisionsRequest,
) ([]pulid.ID, error) {
	return r.settle(ctx, req, true)
}

// settle closes the window on the decisions still in it, in one statement:
// a commit and an undo racing each other each take all of a batch or none.
func (r *repository) settle(
	ctx context.Context,
	req agentdecisionports.SettleAgentDecisionsRequest,
	undo bool,
) ([]pulid.ID, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if len(req.IDs) == 0 {
			return []pulid.ID{}, nil
		}

		d := agentdecisiondb.AgentDecisions
		sets := []keel.Assignment{d.UpdatedAt.To(req.At), d.Version.ToExpr(d.Version)}
		switch {
		case !undo:
			sets = append(sets, d.CommittedAt.To(req.At))
		case req.UndoneByUserID.IsNil():
			sets = append(sets, d.UndoneAt.To(req.At), d.UndoneByUserID.ToNull())
		default:
			sets = append(sets, d.UndoneAt.To(req.At), d.UndoneByUserID.To(req.UndoneByUserID))
		}

		rows, err := d.Pick(d.ID).
			Where(
				d.ID.In(req.IDs...),
				d.CommitsAt.IsNotNull(),
				d.CommittedAt.IsNull(),
				d.UndoneAt.IsNull(),
			).
			UpdateReturning(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			r.l.Error("failed to settle agent decisions", zap.Bool("undo", undo), zap.Error(err))

			return nil, fmt.Errorf("settle agent decisions: %w", err)
		}

		settled := make([]pulid.ID, 0, len(rows))
		for _, row := range rows {
			settled = append(settled, row.ID)
		}

		return settled, nil
	})
}
