package agentproposalrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*agent.AgentProposal, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
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
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

	entity := new(agent.AgentProposal)
	cols := buncolgen.AgentProposalColumns
	err := r.db.DB().
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
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create agent proposal", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	log := r.l.With(zap.String("operation", "UpdateStatus"), zap.String("id", req.ID.String()))

	entity := new(agent.AgentProposal)
	cols := buncolgen.AgentProposalColumns
	results, err := r.db.DB().
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
}

func (r *repository) ExpirePendingByRun(
	ctx context.Context,
	req repositories.ExpireAgentProposalsByRunRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "ExpirePendingByRun"),
		zap.String("runId", req.RunID.String()),
	)

	cols := buncolgen.AgentProposalColumns
	results, err := r.db.DB().
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
}

// ExpirePending closes the decision window on every pending proposal whose
// expiry has passed, across all tenants. It is the sweeper's query and the
// only unscoped write in this repository; the partial index on pending expiry
// is what keeps it cheap at any size.
func (r *repository) ExpirePending(
	ctx context.Context,
	req repositories.ExpireAgentProposalsRequest,
) (int, error) {
	log := r.l.With(zap.String("operation", "ExpirePending"), zap.Int64("before", req.Before))

	cols := buncolgen.AgentProposalColumns
	results, err := r.db.DB().
		NewUpdate().
		Model((*agent.AgentProposal)(nil)).
		Where(cols.Status.Eq(), agent.ProposalStatusPending).
		Where(cols.ExpiresAt.IsNotNull()).
		Where(cols.ExpiresAt.Lte(), req.Before).
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
}

// ListPendingForReminder finds the proposals the sweeper should remind people
// about: pending, from a run a person did not start in a chat, older than the
// cut-off, not yet expired and never reminded. Unscoped like ExpirePending; the
// partial index on pending unreminded rows keeps it cheap.
func (r *repository) ListPendingForReminder(
	ctx context.Context,
	req repositories.ListPendingProposalsForReminderRequest,
) ([]*agent.AgentProposal, error) {
	log := r.l.With(zap.String("operation", "ListPendingForReminder"), zap.Int64("before", req.Before))

	cols := buncolgen.AgentProposalColumns
	runCols := buncolgen.AgentRunColumns
	proposals := make([]*agent.AgentProposal, 0, req.Limit)

	err := r.db.DB().
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
		OrderExpr(cols.CreatedAt.OrderAsc()).
		Limit(req.Limit).
		Scan(ctx)
	if err != nil {
		log.Error("failed to list pending agent proposals for reminder", zap.Error(err))
		return nil, err
	}

	return proposals, nil
}

func (r *repository) MarkReminded(
	ctx context.Context,
	req repositories.MarkProposalsRemindedRequest,
) (int, error) {
	if len(req.IDs) == 0 {
		return 0, nil
	}

	cols := buncolgen.AgentProposalColumns
	results, err := r.db.DB().
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
}
