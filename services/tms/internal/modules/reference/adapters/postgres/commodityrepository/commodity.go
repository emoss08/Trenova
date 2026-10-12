package commodityrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/commoditycol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
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

func New(p Params) commodityports.CommodityRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.commodity-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *commodityports.ListCommodityRequest,
) *bun.SelectQuery {
	cols := commoditycol.CommodityColumns
	q = querybuilder.ApplyFilters(
		q,
		commoditycol.CommodityTable.Alias,
		req.Filter,
		(*commodity.Commodity)(nil),
	)

	return q.Apply(commoditycol.CommodityApplyTenant(req.Filter.TenantInfo)).
		Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset())).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *commodityports.ListCommodityRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*commodity.Commodity], error) {
			log := r.l.With(
				zap.String("operation", "List"),
				zap.Any("request", req),
			)

			entities := make([]*commodity.Commodity, 0, req.Filter.Pagination.SafeLimit())
			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.filterQuery(sq, req)
				}).ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to scan and count commodities", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*commodity.Commodity]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req commodityports.GetCommodityByIDRequest,
) (*commodity.Commodity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*commodity.Commodity, error) {
		log := r.l.With(
			zap.String("operation", "GetByID"),
			zap.String("id", req.ID.String()),
		)

		entity := new(commodity.Commodity)
		cols := commoditycol.CommodityColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return commoditycol.CommodityScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get commodity", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "Commodity")
		}

		return entity, nil
	})
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *commodityports.ListCommodityConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		commoditycol.CommodityTable.Alias,
		req.Filter,
		req.Cursor,
		(*commodity.Commodity)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *commodityports.ListCommodityConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		commoditycol.CommodityTable.Alias,
		req.Filter,
		(*commodity.Commodity)(nil),
	)
}

func applyCommodityColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(commoditycol.CommodityTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *commodityports.ListCommodityConnectionRequest,
) (*pagination.CursorListResult[*commodity.Commodity], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*commodity.Commodity], error) {
			log := r.l.With(
				zap.String("operation", "ListConnection"),
				zap.Any("request", req),
			)

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*commodity.Commodity)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.applyTotalCountFilters(sq, req)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count commodities", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*commodity.Commodity]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*commodity.Commodity) *bun.SelectQuery {
						return dba.
							NewSelect().
							Model(entities).
							Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
								return applyCommodityColumns(sq, req.CommodityColumns)
							})
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return r.applyCursorPageFilters(sq, req)
					},
				})
			if err != nil {
				log.Error("failed to scan commodities", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *commodity.Commodity,
) (*commodity.Commodity, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*commodity.Commodity, error) {
		log := r.l.With(
			zap.String("operation", "Create"),
			zap.String("name", entity.Name),
		)

		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			log.Error("failed to create commodity", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *commodity.Commodity,
) (*commodity.Commodity, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*commodity.Commodity, error) {
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("id", entity.ID.String()),
		)

		ov := entity.Version
		entity.Version++
		cols := commoditycol.CommodityColumns

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(cols.Version.Eq(), ov).
			OmitZero().
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update commodity", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(results, "Commodity", entity.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *commodityports.BulkUpdateCommodityStatusRequest,
) ([]*commodity.Commodity, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*commodity.Commodity, error) {
		log := r.l.With(
			zap.String("operation", "BulkUpdateStatus"),
			zap.Any("request", req),
		)

		entities := make([]*commodity.Commodity, 0, len(req.CommodityIDs))
		cols := commoditycol.CommodityColumns
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(&entities).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return commoditycol.CommodityScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.In(), bun.List(req.CommodityIDs))
			}).
			Set(cols.Status.Set(), req.Status).
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to bulk update commodity status", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckBulkRowsAffected(results, "Commodity", req.CommodityIDs); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req commodityports.GetCommoditiesByIDsRequest,
) ([]*commodity.Commodity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*commodity.Commodity, error) {
		log := r.l.With(
			zap.String("operation", "GetByIDs"),
			zap.Any("request", req),
		)

		entities := make([]*commodity.Commodity, 0, len(req.CommodityIDs))
		cols := commoditycol.CommodityColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Relation(commoditycol.CommodityRelations.HazardousMaterial).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return commoditycol.CommodityScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.In(), bun.List(req.CommodityIDs))
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get commodities", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "Commodity")
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *commodityports.CommoditySelectOptionsRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*commodity.Commodity], error) {
			cols := commoditycol.CommodityColumns

			return dbhelper.SelectOptions[*commodity.Commodity](
				ctx,
				r.db.DBForContext(ctx),
				req.SelectQueryRequest,
				&dbhelper.SelectOptionsConfig{
					ColumnRefs: []buncolgen.Column{
						cols.ID,
						cols.Name,
						cols.HazardousMaterialID,
						cols.FreightClass,
					},
					OrgColumnRef: &cols.OrganizationID,
					BuColumnRef:  &cols.BusinessUnitID,
					QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
						return q.Where(cols.Status.Eq(), domaintypes.StatusActive)
					},
					EntityName:       "Commodity",
					SearchColumnRefs: []buncolgen.Column{cols.Name, cols.Description},
				},
			)
		},
	)
}
