package agentrunrepository

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

func New(p Params) repositories.AgentRunRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentrun-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAgentRunRequest,
) *bun.SelectQuery {
	cols := buncolgen.AgentRunColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.AgentRunTable.Alias,
		req.Filter,
		(*agent.AgentRun)(nil),
	)

	return q.Apply(buncolgen.AgentRunApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAgentRunRequest,
) (*pagination.ListResult[*agent.AgentRun], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*agent.AgentRun, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count agent runs", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*agent.AgentRun]{Items: entities, Total: total}, nil
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListAgentRunConnectionRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.AgentRunTable.Alias,
		req.Filter,
		(*agent.AgentRun)(nil),
	)

	return q.Apply(buncolgen.AgentRunApplyTenant(req.Filter.TenantInfo))
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListAgentRunConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.AgentRunTable.Alias,
		req.Filter,
		req.Cursor,
		(*agent.AgentRun)(nil),
	)
}

func applyAgentRunColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.AgentRunTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentRunConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentRun], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*agent.AgentRun)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count agent runs", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*agent.AgentRun]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*agent.AgentRun) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyAgentRunColumns(sq, req.Columns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan agent runs", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

	entity := new(agent.AgentRun)
	cols := buncolgen.AgentRunColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentRunScopeTenant(sq, *req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get agent run", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "AgentRun")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create agent run", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++
	cols := buncolgen.AgentRunColumns

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).WherePK().
		Where(cols.Version.Eq(), ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update agent run", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "AgentRun", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
