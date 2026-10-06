package carriercapacityrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entityName      = "Carrier capacity posting"
	defaultOpenList = 500
	maxOpenList     = 2000
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

func New(p Params) repositories.CarrierCapacityPostingRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.carrier-capacity-posting-repository")}
}

func withPostingRelations(q *bun.SelectQuery) *bun.SelectQuery {
	rel := buncolgen.PostingRelations

	return q.
		Relation(rel.Carrier).
		Relation(rel.OriginLocation).
		Relation(rel.OriginState).
		Relation(rel.DestinationState).
		Relation(rel.EquipmentType)
}

func applyListOptions(
	q *bun.SelectQuery,
	req *repositories.ListCarrierCapacityPostingsRequest,
) *bun.SelectQuery {
	cols := buncolgen.PostingColumns
	if req.CarrierID.IsNotNil() {
		q = q.Where(cols.CarrierID.Eq(), req.CarrierID)
	}
	if req.OpenAt > 0 {
		q = q.Where(cols.AvailableTo.Gte(), req.OpenAt)
	}
	return q
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCarrierCapacityPostingsRequest,
) (*pagination.ListResult[*carriercapacity.Posting], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*carriercapacity.Posting], error) {
			cols := buncolgen.PostingColumns
			entities := make([]*carriercapacity.Posting, 0, req.Filter.Pagination.SafeLimit())
			total, err := r.db.DBForContext(ctx).NewSelect().
				Model(&entities).
				Apply(withPostingRelations).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = querybuilder.ApplyFilters(
						sq,
						buncolgen.PostingTable.Alias,
						req.Filter,
						(*carriercapacity.Posting)(nil),
					)
					return applyListOptions(sq, req).
						Limit(req.Filter.Pagination.SafeLimit()).
						Offset(req.Filter.Pagination.SafeOffset()).
						Order(cols.AvailableFrom.OrderAsc(), cols.ID.OrderAsc())
				}).
				ScanAndCount(ctx)
			if err != nil {
				r.l.Error("failed to list carrier capacity postings", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*carriercapacity.Posting]{
				Items: entities,
				Total: total,
			}, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierCapacityPostingsRequest,
) (*pagination.CursorListResult[*carriercapacity.Posting], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*carriercapacity.Posting], error) {
			dba := r.db.DBForContext(ctx)

			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.NewSelect().
					Model((*carriercapacity.Posting)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyListOptions(querybuilder.ApplyFiltersWithoutSort(
							sq,
							buncolgen.PostingTable.Alias,
							req.Filter,
							(*carriercapacity.Posting)(nil),
						), req)
					}).
					Count(ctx)
				if err != nil {
					r.l.Error("failed to count carrier capacity postings", zap.Error(err))
					return nil, err
				}
				totalCount = &total
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*carriercapacity.Posting]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*carriercapacity.Posting) *bun.SelectQuery {
						return dba.NewSelect().
							Model(entities).
							ColumnExpr(buncolgen.PostingTable.All()).
							Apply(withPostingRelations)
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						sq, aErr := querybuilder.ApplyCursorFilters(
							sq,
							buncolgen.PostingTable.Alias,
							req.Filter,
							req.Cursor,
							(*carriercapacity.Posting)(nil),
						)
						if aErr != nil {
							return sq, aErr
						}
						return applyListOptions(sq, req), nil
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list carrier capacity postings", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetCarrierCapacityPostingRequest,
) (*carriercapacity.Posting, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carriercapacity.Posting, error) {
		entity := new(carriercapacity.Posting)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			Apply(withPostingRelations).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.PostingScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.PostingColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, entityName)
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *carriercapacity.Posting,
) (*carriercapacity.Posting, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*carriercapacity.Posting, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			r.l.Error("failed to create carrier capacity posting", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *carriercapacity.Posting,
) (*carriercapacity.Posting, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*carriercapacity.Posting, error) {
		cols := buncolgen.PostingColumns
		previous := entity.Version
		entity.Version++

		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model(entity).
			WherePK().
			Where(cols.Version.Eq(), previous).
			ExcludeColumn(cols.CreatedAt.Bare()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to update carrier capacity posting", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(result, entityName, entity.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteCarrierCapacityPostingRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.PostingColumns
		result, err := r.db.DBForContext(ctx).NewDelete().
			Model((*carriercapacity.Posting)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.PostingScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Version.Eq(), req.Version)
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to delete carrier capacity posting", zap.Error(err))
			return err
		}

		return dberror.CheckRowsAffected(result, entityName, req.ID.String())
	})
}

func (r *repository) ListOpen(
	ctx context.Context,
	req *repositories.ListOpenCarrierCapacityRequest,
) ([]*carriercapacity.Posting, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carriercapacity.Posting, error) {
		cols := buncolgen.PostingColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultOpenList
		}
		limit = min(limit, maxOpenList)

		through := req.Through
		if through < req.OpenAt {
			through = req.OpenAt
		}

		entities := make([]*carriercapacity.Posting, 0, 32)
		q := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			Apply(withPostingRelations).
			Apply(buncolgen.PostingApplyTenant(req.TenantInfo)).
			Where(cols.AvailableTo.Gte(), req.OpenAt).
			Where(cols.AvailableFrom.Lte(), through).
			Order(cols.AvailableFrom.OrderAsc(), cols.ID.OrderAsc()).
			Limit(limit)
		if len(req.CarrierIDs) > 0 {
			q = q.Where(cols.CarrierID.In(), bun.List(req.CarrierIDs))
		}

		if err := q.Scan(ctx); err != nil {
			r.l.Error("failed to list open carrier capacity", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
