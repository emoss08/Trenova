package agentexceptionrepository

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

func New(p Params) repositories.AgentExceptionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentexception-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAgentExceptionRequest,
) *bun.SelectQuery {
	cols := buncolgen.AgentExceptionColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.AgentExceptionTable.Alias,
		req.Filter,
		(*agent.AgentException)(nil),
	)

	return q.Apply(buncolgen.AgentExceptionApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAgentExceptionRequest,
) (*pagination.ListResult[*agent.AgentException], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*agent.AgentException, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count agent exceptions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*agent.AgentException]{Items: entities, Total: total}, nil
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListAgentExceptionConnectionRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.AgentExceptionTable.Alias,
		req.Filter,
		(*agent.AgentException)(nil),
	)

	return q.Apply(buncolgen.AgentExceptionApplyTenant(req.Filter.TenantInfo))
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListAgentExceptionConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.AgentExceptionTable.Alias,
		req.Filter,
		req.Cursor,
		(*agent.AgentException)(nil),
	)
}

func applyAgentExceptionColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.AgentExceptionTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentExceptionConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentException], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*agent.AgentException)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count agent exceptions", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*agent.AgentException]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*agent.AgentException) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyAgentExceptionColumns(sq, req.Columns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan agent exceptions", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentExceptionByIDRequest,
) (*agent.AgentException, error) {
	log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

	entity := new(agent.AgentException)
	cols := buncolgen.AgentExceptionColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentExceptionScopeTenant(sq, *req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get agent exception", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "AgentException")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentException,
) (*agent.AgentException, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create agent exception", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateResolution(
	ctx context.Context,
	req repositories.UpdateAgentExceptionResolutionRequest,
) (*agent.AgentException, error) {
	log := r.l.With(zap.String("operation", "UpdateResolution"), zap.String("id", req.ID.String()))

	entity := new(agent.AgentException)
	cols := buncolgen.AgentExceptionColumns
	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AgentExceptionScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.ResolutionState.Set(), req.ResolutionState).
		Set(cols.ResolutionNotes.Set(), req.ResolutionNotes).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update agent exception resolution", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "AgentException", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
