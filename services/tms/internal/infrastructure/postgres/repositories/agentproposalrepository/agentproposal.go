package agentproposalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.AgentProposalRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentproposal-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAgentProposalRequest,
) *bun.SelectQuery {
	cols := buncolgen.AgentProposalColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.AgentProposalTable.Alias,
		req.Filter,
		(*agent.AgentProposal)(nil),
	)

	q = q.Apply(buncolgen.AgentProposalApplyTenant(req.Filter.TenantInfo))
	if req.ExcludeShadowDefinitions {
		q = excludeShadowDefinitions(q)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*pagination.ListResult[*agent.AgentProposal], error) {
		log := r.l.With(zap.String("operation", "List"))

		entities := make([]*agent.AgentProposal, 0, req.Filter.Pagination.SafeLimit())
		total, err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return r.filterQuery(sq, req)
			}).ScanAndCount(ctx)
		if err != nil {
			log.Error("failed to scan and count agent proposals", zap.Error(err))
			return nil, err
		}

		return &pagination.ListResult[*agent.AgentProposal]{Items: entities, Total: total}, nil
	})
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListAgentProposalConnectionRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.AgentProposalTable.Alias,
		req.Filter,
		(*agent.AgentProposal)(nil),
	)

	q = q.Apply(buncolgen.AgentProposalApplyTenant(req.Filter.TenantInfo))
	if req.ExcludeShadowDefinitions {
		q = excludeShadowDefinitions(q)
	}

	return q
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListAgentProposalConnectionRequest,
) (*bun.SelectQuery, error) {
	q, err := querybuilder.ApplyCursorFilters(
		q,
		buncolgen.AgentProposalTable.Alias,
		req.Filter,
		req.Cursor,
		(*agent.AgentProposal)(nil),
	)
	if err != nil {
		return nil, err
	}
	if req.ExcludeShadowDefinitions {
		q = excludeShadowDefinitions(q)
	}

	return q, nil
}

func excludeShadowDefinitions(q *bun.SelectQuery) *bun.SelectQuery {
	proposals := buncolgen.AgentProposalColumns
	runs := buncolgen.AgentRunColumns
	definitions := buncolgen.DefinitionColumns

	return q.Where(
		"NOT EXISTS (SELECT 1 FROM ? AS ? JOIN ? AS ? ON ? = ? AND ? = ? AND ? = ? "+
			"WHERE ? = ? AND ? = ? AND ? = ? AND ? = TRUE)",
		bun.Ident(buncolgen.AgentRunTable.Name),
		bun.Ident(buncolgen.AgentRunTable.Alias),
		bun.Ident(buncolgen.DefinitionTable.Name),
		bun.Ident(buncolgen.DefinitionTable.Alias),
		bun.Safe(definitions.ID.Qualified()),
		bun.Safe(runs.AgentDefinitionID.Qualified()),
		bun.Safe(definitions.OrganizationID.Qualified()),
		bun.Safe(runs.OrganizationID.Qualified()),
		bun.Safe(definitions.BusinessUnitID.Qualified()),
		bun.Safe(runs.BusinessUnitID.Qualified()),
		bun.Safe(runs.ID.Qualified()),
		bun.Safe(proposals.RunID.Qualified()),
		bun.Safe(runs.OrganizationID.Qualified()),
		bun.Safe(proposals.OrganizationID.Qualified()),
		bun.Safe(runs.BusinessUnitID.Qualified()),
		bun.Safe(proposals.BusinessUnitID.Qualified()),
		bun.Safe(definitions.ShadowMode.Qualified()),
	)
}

func applyAgentProposalColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.AgentProposalTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagination.CursorListResult[*agent.AgentProposal], error) {
		log := r.l.With(zap.String("operation", "ListConnection"))

		dba := r.db.DBForContext(ctx)
		var totalCount *int
		if req.Cursor.IncludeTotalCount {
			total, err := dba.
				NewSelect().
				Model((*agent.AgentProposal)(nil)).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.applyTotalCountFilters(sq, req)
				}).
				Count(ctx)
			if err != nil {
				log.Error("failed to count agent proposals", zap.Error(err))
				return nil, err
			}
			totalCount = &total
		}

		result, err := dbhelper.CursorList(
			ctx,
			dbhelper.CursorListParams[*agent.AgentProposal]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(entities *[]*agent.AgentProposal) *bun.SelectQuery {
					return dba.
						NewSelect().
						Model(entities).
						Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
							return applyAgentProposalColumns(sq, req.Columns)
						})
				},
				Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
					return r.applyCursorPageFilters(sq, req)
				},
			})
		if err != nil {
			log.Error("failed to scan agent proposals", zap.Error(err))
			return nil, err
		}

		return result, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

		entity := new(agent.AgentProposal)
		cols := buncolgen.AgentProposalColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentProposalScopeTenant(sq, *req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get agent proposal", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "AgentProposal")
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		log := r.l.With(zap.String("operation", "Create"))

		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to create agent proposal", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentProposal, error) {
		log := r.l.With(zap.String("operation", "UpdateStatus"), zap.String("id", req.ID.String()))

		entity := new(agent.AgentProposal)
		cols := buncolgen.AgentProposalColumns
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				scoped := buncolgen.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
				if req.FromStatus != "" {
					scoped = scoped.Where(cols.Status.Eq(), req.FromStatus)
				}

				return scoped
			}).
			Set(cols.Status.Set(), req.Status).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update agent proposal status", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(results, "AgentProposal", req.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ExpirePendingByRun(
	ctx context.Context,
	req repositories.ExpireAgentProposalsByRunRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		log := r.l.With(
			zap.String("operation", "ExpirePendingByRun"),
			zap.String("runId", req.RunID.String()),
		)

		cols := buncolgen.AgentProposalColumns
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.AgentProposal)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.RunID.Eq(), req.RunID).
					Where(cols.Status.Eq(), agent.ProposalStatusPending)
			}).
			Set(cols.Status.Set(), agent.ProposalStatusExpired).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			log.Error("failed to expire pending agent proposals", zap.Error(err))
			return 0, err
		}

		affected, err := results.RowsAffected()
		if err != nil {
			return 0, err
		}

		return int(affected), nil
	})
}

// ExpirePending closes the decision window on every pending proposal whose
// expiry has passed, across all tenants. It is the sweeper's query and the
// only unscoped write in this repository; the partial index on pending expiry
// is what keeps it cheap at any size.
func (r *repository) ExpirePending(
	ctx context.Context,
	req repositories.ExpireAgentProposalsRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, "expire stale agent proposals across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		log := r.l.With(zap.String("operation", "ExpirePending"), zap.Int64("before", req.Before))

		cols := buncolgen.AgentProposalColumns
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.AgentProposal)(nil)).
			Where(cols.Status.Eq(), agent.ProposalStatusPending).
			Where(cols.ExpiresAt.IsNotNull()).
			Where(cols.ExpiresAt.Lte(), req.Before).
			// A step of a plan in its undo window was approved with the plan,
			// inside its window, and runs when the plan commits.
			Where("NOT EXISTS (SELECT 1 FROM agent_plans AS apl WHERE apl.id = "+
				cols.PlanID.Qualified()+" AND apl.organization_id = "+
				cols.OrganizationID.Qualified()+" AND apl.business_unit_id = "+
				cols.BusinessUnitID.Qualified()+" AND apl.status = ?)", agent.PlanStatusApproving).
			Set(cols.Status.Set(), agent.ProposalStatusExpired).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			log.Error("failed to expire pending agent proposals", zap.Error(err))
			return 0, err
		}

		affected, err := results.RowsAffected()
		if err != nil {
			return 0, err
		}

		return int(affected), nil
	})
}

// ListPendingForReminder finds the proposals the sweeper should remind people
// about: pending, from a run a person did not start in a chat, older than the
// cut-off, not yet expired and never reminded. Unscoped like ExpirePending; the
// partial index on pending unreminded rows keeps it cheap.
func (r *repository) ListPendingForReminder(
	ctx context.Context,
	req repositories.ListPendingProposalsForReminderRequest,
) ([]*agent.AgentProposal, error) {
	ctx = dbscope.WithSystem(ctx, "list pending proposals due a reminder across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		log := r.l.With(
			zap.String("operation", "ListPendingForReminder"),
			zap.Int64("before", req.Before),
		)

		cols := buncolgen.AgentProposalColumns
		runCols := buncolgen.AgentRunColumns
		proposals := make([]*agent.AgentProposal, 0, req.Limit)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&proposals).
			Join("JOIN agent_runs AS ar ON "+runCols.ID.Qualified()+" = "+cols.RunID.Qualified()+
				" AND "+runCols.OrganizationID.Qualified()+" = "+cols.OrganizationID.Qualified()+
				" AND "+runCols.BusinessUnitID.Qualified()+" = "+cols.BusinessUnitID.Qualified()).
			Where(cols.Status.Eq(), agent.ProposalStatusPending).
			Where(cols.RemindedAt.IsNull()).
			Where(cols.CreatedAt.Lte(), req.Before).
			Where(runCols.Trigger.Qualified()+" <> ?", agent.RunTriggerChat).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where(cols.ExpiresAt.IsNull()).
					WhereOr(cols.ExpiresAt.Gt(), req.Now)
			}).
			OrderExpr(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Limit(req.Limit).
			Scan(ctx)
		if err != nil {
			log.Error("failed to list pending agent proposals for reminder", zap.Error(err))
			return nil, err
		}

		return proposals, nil
	})
}

func (r *repository) MarkReminded(
	ctx context.Context,
	req repositories.MarkProposalsRemindedRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.IDs) == 0 {
			return 0, nil
		}

		cols := buncolgen.AgentProposalColumns
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.AgentProposal)(nil)).
			Where(cols.ID.In(), bun.In(req.IDs)).
			Where(cols.RemindedAt.IsNull()).
			Set(cols.RemindedAt.Set(), req.At).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to mark agent proposals reminded", zap.Error(err))
			return 0, err
		}

		affected, err := results.RowsAffected()
		if err != nil {
			return 0, err
		}

		return int(affected), nil
	})
}

// ListByPlan reads a plan's steps in the order they are meant to run.
func (r *repository) ListByIDs(
	ctx context.Context,
	req repositories.ListAgentProposalsByIDsRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		if len(req.IDs) == 0 {
			return []*agent.AgentProposal{}, nil
		}

		cols := buncolgen.AgentProposalColumns
		proposals := make([]*agent.AgentProposal, 0, len(req.IDs))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&proposals).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentProposalScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.In(), bun.In(req.IDs))
			}).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list agent proposals by ids", zap.Error(err))

			return nil, fmt.Errorf("list agent proposals by ids: %w", err)
		}

		return proposals, nil
	})
}

func (r *repository) ListByPlan(
	ctx context.Context,
	req repositories.ListAgentProposalsByPlanRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		cols := buncolgen.AgentProposalColumns
		proposals := make([]*agent.AgentProposal, 0)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&proposals).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentProposalScopeTenant(sq, req.TenantInfo).
					Where(cols.PlanID.Eq(), req.PlanID)
			}).
			OrderExpr(cols.PlanStep.OrderAsc()).
			OrderExpr(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list agent proposals by plan", zap.Error(err))

			return nil, fmt.Errorf("list agent proposals by plan: %w", err)
		}

		return proposals, nil
	})
}

// SkipPendingByPlan closes every step of a plan that is still pending once an
// earlier step has failed: they were never decided against and did not
// expire, and the record should say so.
func (r *repository) SkipPendingByPlan(
	ctx context.Context,
	req repositories.SkipPendingByPlanRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := buncolgen.AgentProposalColumns
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.AgentProposal)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.PlanID.Eq(), req.PlanID).
					Where(cols.Status.Eq(), agent.ProposalStatusPending)
			}).
			Set(cols.Status.Set(), agent.ProposalStatusSkipped).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to skip agent proposals by plan", zap.Error(err))

			return 0, err
		}

		affected, err := results.RowsAffected()
		if err != nil {
			return 0, err
		}

		return int(affected), nil
	})
}

func (r *repository) ListByRun(
	ctx context.Context,
	req repositories.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentProposal, error) {
		cols := buncolgen.AgentProposalColumns
		rows := make([]*agent.AgentProposal, 0, 8)

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentProposalScopeTenant(sq, req.TenantInfo).
					Where(cols.RunID.Eq(), req.RunID)
			}).
			OrderExpr(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			OrderExpr(cols.ID.OrderAsc()).
			Scan(ctx); err != nil {
			r.l.Error("failed to list proposals by run",
				zap.String("runId", req.RunID.String()), zap.Error(err))

			return nil, fmt.Errorf("list proposals by run: %w", err)
		}

		return rows, nil
	})
}
