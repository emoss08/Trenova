package agentplanrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const planEntity = "AgentPlan"

var planErrors = dberror.Mapping{Entity: planEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AgentPlanRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentplan-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentPlan,
) (*agent.AgentPlan, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentPlan, error) {
		if err := agentdb.AgentPlans.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create agent plan", zap.Error(err))

			return nil, fmt.Errorf("create agent plan: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req agentports.ListAgentPlansByIDsRequest,
) ([]*agent.AgentPlan, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentPlan, error) {
		if len(req.IDs) == 0 {
			return []*agent.AgentPlan{}, nil
		}

		p := agentdb.AgentPlans
		plans, err := p.Where(p.ID.In(req.IDs...)).Limit(len(req.IDs)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent plans by ids: %w", err)
		}

		return plans, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentports.GetAgentPlanByIDRequest,
) (*agent.AgentPlan, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.AgentPlan, error) {
		entity, err := agentdb.AgentPlans.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, planErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

// ListByThread finds the plans of every run a conversation started, oldest
// first, so a reopened thread can show them where their turns were.
func (r *repository) ListByThread(
	ctx context.Context,
	req agentports.ListAgentPlansByThreadRequest,
) ([]*agent.AgentPlan, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentPlan, error) {
		p := agentdb.AgentPlans
		run := p.Run()
		plans, err := p.Where(
			run.SubjectType.Eq(agent.SubjectAssistantThread),
			run.SubjectID.Eq(req.ThreadID),
		).
			Order(p.CreatedAt.Asc(), p.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent plans for thread", zap.Error(err))

			return nil, fmt.Errorf("list agent plans by thread: %w", err)
		}

		return plans, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentPlanConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentPlan], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.AgentPlan], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				agentdb.AgentPlans.Query(),
				&listquery.Request[agent.AgentPlan]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list agent plans", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func statusAssignments(req *agentports.UpdateAgentPlanStatusRequest) []keel.Assignment {
	p := agentdb.AgentPlans
	sets := []keel.Assignment{p.Status.To(req.Status)}
	approval := []keel.Assignment{
		p.ApprovalReasonCode.To(req.Approval.ReasonCode),
		p.ApprovalNote.To(req.Approval.Note),
		p.ApprovalPreviewDigest.To(req.Approval.PreviewDigest),
	}
	switch {
	case req.Reopen:
		sets = append(sets,
			p.CommitsAt.ToNull(),
			p.ApprovalReasonCode.ToNull(),
			p.ApprovalNote.ToNull(),
			p.ApprovalPreviewDigest.ToNull(),
			p.DecidedByUserID.ToNull(),
			p.DecidedAt.ToNull(),
		)
	case req.CommitsAt != 0:
		if req.DecidedByUserID.IsNotNil() {
			sets = append(sets,
				p.DecidedByUserID.To(req.DecidedByUserID),
				p.DecidedAt.To(req.DecidedAt),
			)
		}
		sets = append(sets, p.CommitsAt.To(req.CommitsAt))
		sets = append(sets, approval...)
		if req.UndoneByUserID.IsNil() {
			sets = append(sets, p.UndoneAt.ToNull(), p.UndoneByUserID.ToNull())
		}
	case req.DecidedByUserID.IsNotNil():
		sets = append(sets,
			p.DecidedByUserID.To(req.DecidedByUserID),
			p.DecidedAt.To(req.DecidedAt),
		)
		sets = append(sets, approval...)
	}
	if req.UndoneByUserID.IsNotNil() {
		sets = append(sets,
			p.UndoneByUserID.To(req.UndoneByUserID),
			p.UndoneAt.To(req.UndoneAt),
		)
	}

	return sets
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req agentports.UpdateAgentPlanStatusRequest,
) (*agent.AgentPlan, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentPlan, error) {
		p := agentdb.AgentPlans
		rows, err := p.Where(
			p.ID.Eq(req.ID),
			keel.When(req.FromStatus != "", p.Status.Eq(req.FromStatus)),
		).UpdateReturning(ctx, r.db.Keel(ctx), statusAssignments(&req)...)
		if err != nil {
			return nil, fmt.Errorf("update agent plan status: %w", err)
		}
		if err = dberror.CheckAffected(int64(len(rows)), planEntity, req.ID.String()); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) RecordProgress(
	ctx context.Context,
	req agentports.RecordAgentPlanProgressRequest,
) (*agent.AgentPlan, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentPlan, error) {
		p := agentdb.AgentPlans
		failedStep := p.FailedStep.ToNull()
		if req.FailedStep != nil {
			failedStep = p.FailedStep.To(int64(*req.FailedStep))
		}

		rows, err := p.Where(p.ID.Eq(req.ID)).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			p.Status.To(req.Status),
			p.CompletedSteps.To(int64(req.CompletedSteps)),
			failedStep,
			p.FailureError.To(req.FailureError),
		)
		if err != nil {
			return nil, fmt.Errorf("record agent plan progress: %w", err)
		}
		if err = dberror.CheckAffected(int64(len(rows)), planEntity, req.ID.String()); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) ExpirePending(
	ctx context.Context,
	req agentports.ExpireAgentPlansRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, "expire stale agent plans across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		p := agentdb.AgentPlans
		n, err := p.AllTenants().
			Where(
				p.Status.Eq(agent.PlanStatusPending),
				p.ExpiresAt.IsNotNull(),
				p.ExpiresAt.Lte(req.Before),
			).
			Update(
				ctx,
				r.db.Keel(ctx),
				p.Status.To(agent.PlanStatusExpired),
				p.Version.ToExpr(p.Version),
			)
		if err != nil {
			r.l.Error("failed to expire pending agent plans", zap.Error(err))

			return 0, err
		}

		return int(n), nil
	})
}

func (r *repository) ExpirePendingByThread(
	ctx context.Context,
	req agentports.ExpireAgentPlansByThreadRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		run := agentdb.AgentRuns
		n, err := r.expireByRuns(ctx, run.Pick(run.ID).Where(
			run.SubjectType.Eq(agent.SubjectAssistantThread),
			run.SubjectID.Eq(req.ThreadID),
		))
		if err != nil {
			r.l.Error("failed to withdraw the pending plans of a conversation",
				zap.String("threadId", req.ThreadID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return n, nil
	})
}

func (r *repository) ExpirePendingByDefinition(
	ctx context.Context,
	req agentports.ExpireAgentPlansByDefinitionRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		run := agentdb.AgentRuns
		n, err := r.expireByRuns(ctx, run.Pick(run.ID).Where(
			run.AgentDefinitionID.Eq(req.AgentDefinitionID),
		))
		if err != nil {
			r.l.Error("failed to withdraw the pending plans of an agent",
				zap.String("agentDefinitionId", req.AgentDefinitionID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return n, nil
	})
}

func (r *repository) expireByRuns(ctx context.Context, runs *keel.Query[agent.AgentRun]) (int, error) {
	p := agentdb.AgentPlans
	n, err := p.Where(p.Status.Eq(agent.PlanStatusPending), p.RunID.InQuery(runs)).
		Update(ctx, r.db.Keel(ctx), p.Status.To(agent.PlanStatusExpired))
	if err != nil {
		return 0, err
	}

	return int(n), nil
}
