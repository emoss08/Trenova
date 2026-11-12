package agentproposalrepository

import (
	"context"
	"fmt"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/caseownerquery"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan/agentplandb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const proposalEntity = "AgentProposal"

const failedPerTargetLimit = 3

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

func New(p Params) agentproposalports.AgentProposalRepository {
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
	req *agentproposalports.ListAgentProposalRequest,
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
	req *agentproposalports.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.AgentProposal], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				agentdb.AgentProposals.Where(
					caseownerquery.InSight(agentdb.AgentProposals.TargetID, req.Sight),
				),
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
	req agentproposalports.GetAgentProposalByIDRequest,
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
	q keel.Query[*agent.AgentProposal],
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
	req *agentproposalports.UpdateAgentProposalStatusRequest,
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
	req agentproposalports.ExpireAgentProposalsByRunRequest,
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

func pendingOfRuns(runs keel.Query[*agent.AgentRun]) keel.Query[*agent.AgentProposal] {
	p := agentdb.AgentProposals

	return p.Where(p.Status.Eq(agent.ProposalStatusPending), p.RunID.InQuery(runs))
}

func (r *repository) expireByRuns(
	ctx context.Context,
	runs keel.Query[*agent.AgentRun],
) (int, error) {
	n, err := pendingOfRuns(runs).Update(ctx, r.db.Keel(ctx), expiry()...)
	if err != nil {
		return 0, err
	}

	return int(n), nil
}

func (r *repository) ExpirePendingByDefinition(
	ctx context.Context,
	req agentproposalports.ExpireAgentProposalsByDefinitionRequest,
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
	req agentproposalports.ExpireAgentProposalsByThreadRequest,
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
	req agentproposalports.ExpireAgentProposalsRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, "expire stale agent proposals across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		p := agentdb.AgentProposals
		plan := agentplandb.AgentPlans
		n, err := p.AllTenants().
			Where(
				p.Status.Eq(agent.ProposalStatusPending),
				p.ExpiresAt.IsNotNull(),
				p.ExpiresAt.Lte(req.Before),
				keel.NotExists(plan.AllTenants().Where(
					plan.ID.EqCol(p.PlanID),
					plan.OrganizationID.EqCol(p.OrganizationID),
					plan.BusinessUnitID.EqCol(p.BusinessUnitID),
					plan.Status.Eq(agentplan.PlanStatusApproving),
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
	req agentproposalports.ListPendingProposalsForReminderRequest,
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
	req agentproposalports.MarkProposalsRemindedRequest,
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
	req agentproposalports.ListAgentProposalsByIDsRequest,
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

func byPlansQuery(planIDs ...pulid.ID) keel.Query[*agent.AgentProposal] {
	p := agentdb.AgentProposals

	return p.Where(p.PlanID.In(planIDs...)).
		Order(p.PlanStep.Asc(), p.CreatedAt.Asc(), p.ID.Asc())
}

// ListByPlan reads a plan's steps in the order they are meant to run.
func (r *repository) ListByPlan(
	ctx context.Context,
	req agentproposalports.ListAgentProposalsByPlanRequest,
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
	req agentproposalports.ListAgentProposalsByPlansRequest,
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
	req agentproposalports.SkipPendingByPlanRequest,
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
	req agentproposalports.ListAgentProposalsByRunRequest,
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

func (r *repository) ReplacePendingForTargets(
	ctx context.Context,
	req *agentproposalports.ReplaceTargetedProposalsRequest,
) ([]*agent.AgentProposal, error) {
	if len(req.Proposals) == 0 {
		return []*agent.AgentProposal{}, nil
	}

	runID := req.Proposals[0].RunID
	targets := make([]pulid.ID, 0, len(req.Proposals))
	for _, proposal := range req.Proposals {
		if proposal.RunID != runID || proposal.ToolName != req.ToolName ||
			proposal.TargetResource != req.TargetResource || proposal.TargetID.IsNil() {
			return nil, fmt.Errorf(
				"replace targeted proposals: every proposal must be of run %s, tool %s and name a %s",
				runID, req.ToolName, req.TargetResource,
			)
		}
		targets = append(targets, proposal.TargetID)
	}

	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		if _, err := p.Where(
			p.Status.Eq(agent.ProposalStatusPending),
			p.ToolName.Eq(req.ToolName),
			p.TargetResource.Eq(req.TargetResource),
			p.TargetID.In(targets...),
			p.RunID.Ne(runID),
		).Update(ctx, r.db.Keel(ctx),
			p.Status.To(agent.ProposalStatusSuperseded),
			p.PendingModifications.ToNull(),
			keepVersion(),
		); err != nil {
			r.l.Error("failed to supersede earlier pending proposals",
				zap.String("tool", req.ToolName), zap.Error(err))

			return nil, fmt.Errorf("supersede pending proposals for targets: %w", err)
		}

		if err := p.CreateMany(ctx, r.db.Keel(ctx), req.Proposals); err != nil {
			r.l.Error("failed to create targeted proposals",
				zap.String("tool", req.ToolName), zap.Error(err))

			return nil, proposalErrors.Map(err, runID.String())
		}

		return req.Proposals, nil
	})
}

func settledAt() keel.Int64Expr {
	p := agentdb.AgentProposals

	return p.ExecutedAt.Coalesce(p.SimulatedAt)
}

func (r *repository) ListSettled(
	ctx context.Context,
	req *agentproposalports.ListSettledProposalsRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		p := agentdb.AgentProposals
		settled := settledAt()
		proposals, err := p.Where(
			keel.Or(p.ExecutedAt.IsNotNull(), p.SimulatedAt.IsNotNull()),
			settled.Gt(req.Since),
		).
			Order(keel.Desc(settled), p.ID.Desc()).
			Limit(max(req.Limit, 1)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list settled proposals", zap.Error(err))

			return nil, fmt.Errorf("list settled proposals: %w", err)
		}

		return proposals, nil
	})
}

func (r *repository) ListFailedByTargets(
	ctx context.Context,
	req *agentproposalports.ListFailedByTargetsRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		if len(req.TargetIDs) == 0 {
			return []*agent.AgentProposal{}, nil
		}

		p := agentdb.AgentProposals
		proposals, err := p.Where(
			p.Status.Eq(agent.ProposalStatusExecutionFailed),
			p.ToolName.Eq(req.ToolName),
			p.TargetResource.Eq(req.TargetResource),
			p.TargetID.In(req.TargetIDs...),
		).
			Order(p.CreatedAt.Desc(), p.ID.Desc()).
			Limit(len(req.TargetIDs)*failedPerTargetLimit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list failed proposals by target", zap.Error(err))

			return nil, fmt.Errorf("list failed proposals by target: %w", err)
		}

		return proposals, nil
	})
}

func (r *repository) ListStandingByDefinition(
	ctx context.Context,
	req *agentproposalports.ListStandingByDefinitionRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		if len(req.ToolNames) == 0 || req.DefinitionID.IsNil() {
			return []*agent.AgentProposal{}, nil
		}

		p := agentdb.AgentProposals
		proposals, err := p.Where(
			p.Status.Eq(agent.ProposalStatusPending),
			p.ToolName.In(req.ToolNames...),
			p.Run().AgentDefinitionID.Eq(req.DefinitionID),
			keel.Or(p.ExpiresAt.IsNull(), p.ExpiresAt.Gt(req.Now)),
		).
			Order(p.CreatedAt.Desc(), p.ID.Desc()).
			Limit(max(req.Limit, 1)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list an agent's standing proposals",
				zap.String("agentDefinitionId", req.DefinitionID.String()), zap.Error(err))

			return nil, fmt.Errorf("list standing proposals by definition: %w", err)
		}

		return proposals, nil
	})
}

func (r *repository) ListPendingByTargets(
	ctx context.Context,
	req *agentproposalports.ListPendingByTargetsRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		if len(req.TargetIDs) == 0 {
			return []*agent.AgentProposal{}, nil
		}

		p := agentdb.AgentProposals
		proposals, err := p.Where(
			p.Status.Eq(agent.ProposalStatusPending),
			p.ToolName.Eq(req.ToolName),
			p.TargetResource.Eq(req.TargetResource),
			p.TargetID.In(req.TargetIDs...),
			keel.Or(p.ExpiresAt.IsNull(), p.ExpiresAt.Gt(req.Now)),
		).
			Order(p.CreatedAt.Desc(), p.ID.Desc()).
			Limit(len(req.TargetIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list pending proposals by target", zap.Error(err))

			return nil, fmt.Errorf("list pending proposals by target: %w", err)
		}

		return proposals, nil
	})
}
