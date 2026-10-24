package agentproposalrepository

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
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const proposalEntity = "AgentProposal"

var proposalErrors = dberror.Mapping{Entity: proposalEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AgentProposalRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentproposal-repository"),
	}
}

func keepVersion() keel.Assignment {
	p := agentdb.AgentProposals

	return p.Version.ToExpr(p.Version)
}

func expiry() []keel.Assignment {
	p := agentdb.AgentProposals

	return []keel.Assignment{
		p.Status.To(agent.ProposalStatusExpired),
		p.PendingModifications.ToNull(),
		keepVersion(),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *agentports.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*agent.AgentProposal], error) {
			p := agentdb.AgentProposals
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				p.Query(),
				&listquery.Request[agent.AgentProposal]{
					Filter: req.Filter,
					Then:   []keel.Order{p.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to scan and count agent proposals", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.AgentProposal], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				agentdb.AgentProposals.Query(),
				&listquery.Request[agent.AgentProposal]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to scan agent proposals", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentports.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		entity, err := agentdb.AgentProposals.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			r.l.Error("failed to get agent proposal",
				zap.String("id", req.ID.String()), zap.Error(err))
			return nil, proposalErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		if err := agentdb.AgentProposals.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create agent proposal", zap.Error(err))
			return nil, proposalErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) updateOne(
	ctx context.Context,
	id pulid.ID,
	q *keel.Query[agent.AgentProposal],
	sets ...keel.Assignment,
) (*agent.AgentProposal, error) {
	rows, err := q.UpdateReturning(ctx, r.db.Keel(ctx), append(sets, keepVersion())...)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckAffected(int64(len(rows)), proposalEntity, id.String()); err != nil {
		return nil, err
	}

	return rows[0], nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req agentports.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		sets := []keel.Assignment{p.Status.To(req.Status)}
		if clearsPendingModifications(req.Status) {
			sets = append(sets, p.PendingModifications.ToNull())
		}

		entity, err := r.updateOne(ctx, req.ID, p.Where(
			p.ID.Eq(req.ID),
			keel.When(req.FromStatus != "", p.Status.Eq(req.FromStatus)),
		), sets...)
		if err != nil {
			r.l.Error("failed to update agent proposal status",
				zap.String("id", req.ID.String()), zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ExpirePendingByRun(
	ctx context.Context,
	req agentports.ExpireAgentProposalsByRunRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		p := agentdb.AgentProposals
		n, err := p.Where(p.RunID.Eq(req.RunID), p.Status.Eq(agent.ProposalStatusPending)).
			Update(ctx, r.db.Keel(ctx), expiry()...)
		if err != nil {
			r.l.Error("failed to expire pending agent proposals",
				zap.String("runId", req.RunID.String()), zap.Error(err))
			return 0, err
		}

		return int(n), nil
	})
}

func pendingOfRuns(runs *keel.Query[agent.AgentRun]) *keel.Query[agent.AgentProposal] {
	p := agentdb.AgentProposals

	return p.Where(p.Status.Eq(agent.ProposalStatusPending), p.RunID.InQuery(runs))
}

func (r *repository) expireByRuns(ctx context.Context, runs *keel.Query[agent.AgentRun]) (int, error) {
	n, err := pendingOfRuns(runs).Update(ctx, r.db.Keel(ctx), expiry()...)
	if err != nil {
		return 0, err
	}

	return int(n), nil
}

func (r *repository) ExpirePendingByDefinition(
	ctx context.Context,
	req agentports.ExpireAgentProposalsByDefinitionRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		run := agentdb.AgentRuns
		n, err := r.expireByRuns(ctx, run.Pick(run.ID).Where(
			run.AgentDefinitionID.Eq(req.AgentDefinitionID),
		))
		if err != nil {
			r.l.Error("failed to withdraw the pending proposals of an agent",
				zap.String("agentDefinitionId", req.AgentDefinitionID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return n, nil
	})
}

// ExpirePendingByThread withdraws what a conversation's turns proposed and
// nobody decided, when the conversation is deleted: approving one afterwards
// would follow up into a thread that no longer exists.
func (r *repository) ExpirePendingByThread(
	ctx context.Context,
	req agentports.ExpireAgentProposalsByThreadRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		run := agentdb.AgentRuns
		n, err := r.expireByRuns(ctx, run.Pick(run.ID).Where(
			run.SubjectType.Eq(agent.SubjectAssistantThread),
			run.SubjectID.Eq(req.ThreadID),
		))
		if err != nil {
			r.l.Error("failed to withdraw the pending proposals of a conversation",
				zap.String("threadId", req.ThreadID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return n, nil
	})
}

// ExpirePending closes the decision window on every pending proposal whose
// expiry has passed, across all tenants. It is the sweeper's query and the
// only unscoped write in this repository; the partial index on pending expiry
// is what keeps it cheap at any size.
func (r *repository) ExpirePending(
	ctx context.Context,
	req agentports.ExpireAgentProposalsRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, "expire stale agent proposals across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		p := agentdb.AgentProposals
		plan := agentdb.AgentPlans
		n, err := p.AllTenants().
			Where(
				p.Status.Eq(agent.ProposalStatusPending),
				p.ExpiresAt.IsNotNull(),
				p.ExpiresAt.Lte(req.Before),
				keel.NotExists(plan.AllTenants().Where(
					plan.ID.EqCol(p.PlanID),
					plan.OrganizationID.EqCol(p.OrganizationID),
					plan.BusinessUnitID.EqCol(p.BusinessUnitID),
					plan.Status.Eq(agent.PlanStatusApproving),
				)),
			).
			Update(ctx, r.db.Keel(ctx), expiry()...)
		if err != nil {
			r.l.Error("failed to expire pending agent proposals",
				zap.Int64("before", req.Before), zap.Error(err))
			return 0, err
		}

		return int(n), nil
	})
}

// ListPendingForReminder finds the proposals the sweeper should remind people
// about: pending, from a run a person did not start in a chat, older than the
// cut-off, not yet expired and never reminded. Unscoped like ExpirePending; the
// partial index on pending unreminded rows keeps it cheap.
func (r *repository) ListPendingForReminder(
	ctx context.Context,
	req agentports.ListPendingProposalsForReminderRequest,
) ([]*agent.AgentProposal, error) {
	ctx = dbscope.WithSystem(ctx, "list pending proposals due a reminder across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		run := agentdb.AgentRuns
		q := p.AllTenants().
			Join(run,
				run.ID.EqCol(p.RunID),
				run.OrganizationID.EqCol(p.OrganizationID),
				run.BusinessUnitID.EqCol(p.BusinessUnitID),
			).
			Where(
				p.Status.Eq(agent.ProposalStatusPending),
				p.RemindedAt.IsNull(),
				p.CreatedAt.Lte(req.Before),
				run.Trigger.Ne(agent.RunTriggerChat),
				keel.Or(p.ExpiresAt.IsNull(), p.ExpiresAt.Gt(req.Now)),
			).
			Order(p.CreatedAt.Asc(), p.ID.Asc())

		var (
			proposals []*agent.AgentProposal
			err       error
		)
		if req.Limit > 0 {
			proposals, err = q.Limit(req.Limit).List(ctx, r.db.Keel(ctx))
		} else {
			proposals, err = q.All(ctx, r.db.Keel(ctx))
		}
		if err != nil {
			r.l.Error("failed to list pending agent proposals for reminder",
				zap.Int64("before", req.Before), zap.Error(err))
			return nil, err
		}

		return proposals, nil
	})
}

func (r *repository) MarkReminded(
	ctx context.Context,
	req agentports.MarkProposalsRemindedRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.IDs) == 0 {
			return 0, nil
		}

		p := agentdb.AgentProposals
		n, err := p.Where(p.ID.In(req.IDs...), p.RemindedAt.IsNull()).
			Update(ctx, r.db.Keel(ctx), p.RemindedAt.To(req.At), keepVersion())
		if err != nil {
			r.l.Error("failed to mark agent proposals reminded", zap.Error(err))
			return 0, err
		}

		return int(n), nil
	})
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req agentports.ListAgentProposalsByIDsRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		if len(req.IDs) == 0 {
			return []*agent.AgentProposal{}, nil
		}

		p := agentdb.AgentProposals
		proposals, err := p.Where(p.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent proposals by ids", zap.Error(err))

			return nil, fmt.Errorf("list agent proposals by ids: %w", err)
		}

		return proposals, nil
	})
}

func byPlansQuery(planIDs ...pulid.ID) *keel.Query[agent.AgentProposal] {
	p := agentdb.AgentProposals

	return p.Where(p.PlanID.In(planIDs...)).
		Order(p.PlanStep.Asc(), p.CreatedAt.Asc(), p.ID.Asc())
}

// ListByPlan reads a plan's steps in the order they are meant to run.
func (r *repository) ListByPlan(
	ctx context.Context,
	req agentports.ListAgentProposalsByPlanRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		proposals, err := byPlansQuery(req.PlanID).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent proposals by plan", zap.Error(err))

			return nil, fmt.Errorf("list agent proposals by plan: %w", err)
		}

		return proposals, nil
	})
}

func (r *repository) ListByPlans(
	ctx context.Context,
	req agentports.ListAgentProposalsByPlansRequest,
) ([]*agent.AgentProposal, error) {
	if len(req.PlanIDs) == 0 {
		return []*agent.AgentProposal{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		proposals, err := byPlansQuery(req.PlanIDs...).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent proposals by plans", zap.Error(err))

			return nil, fmt.Errorf("list agent proposals by plans: %w", err)
		}

		return proposals, nil
	})
}

// SkipPendingByPlan closes every step of a plan that is still pending once an
// earlier step has failed: they were never decided against and did not
// expire, and the record should say so.
func (r *repository) SkipPendingByPlan(
	ctx context.Context,
	req agentports.SkipPendingByPlanRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		p := agentdb.AgentProposals
		n, err := p.Where(p.PlanID.Eq(req.PlanID), p.Status.Eq(agent.ProposalStatusPending)).
			Update(
				ctx,
				r.db.Keel(ctx),
				p.Status.To(agent.ProposalStatusSkipped),
				p.PendingModifications.ToNull(),
				keepVersion(),
			)
		if err != nil {
			r.l.Error("failed to skip agent proposals by plan", zap.Error(err))

			return 0, err
		}

		return int(n), nil
	})
}

func (r *repository) ListByRun(
	ctx context.Context,
	req agentports.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		rows, err := p.Where(p.RunID.Eq(req.RunID)).
			Order(p.CreatedAt.Asc(), p.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list proposals by run",
				zap.String("runId", req.RunID.String()), zap.Error(err))

			return nil, fmt.Errorf("list proposals by run: %w", err)
		}

		return rows, nil
	})
}
