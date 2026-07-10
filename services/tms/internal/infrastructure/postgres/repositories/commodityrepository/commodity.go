package commodityrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.CommodityRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.commodity-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListCommodityRequest,
) *bun.SelectQuery {
	cols := buncolgen.CommodityColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.CommodityTable.Alias,
		req.Filter,
		(*commodity.Commodity)(nil),
	)

	return q.Apply(buncolgen.CommodityApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCommodityRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*commodity.Commodity, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
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
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetCommodityByIDRequest,
) (*commodity.Commodity, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(commodity.Commodity)
	cols := buncolgen.CommodityColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CommodityScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get commodity", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Commodity")
	}

	return entity, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListCommodityConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.CommodityTable.Alias,
		req.Filter,
		req.Cursor,
		(*commodity.Commodity)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListCommodityConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.CommodityTable.Alias,
		req.Filter,
		(*commodity.Commodity)(nil),
	)
}

func applyCommodityColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.CommodityTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListCommodityConnectionRequest,
) (*pagination.CursorListResult[*commodity.Commodity], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
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

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*commodity.Commodity]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
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
}

func (r *repository) Create(
	ctx context.Context,
	entity *commodity.Commodity,
) (*commodity.Commodity, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create commodity", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *commodity.Commodity,
) (*commodity.Commodity, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++
	cols := buncolgen.CommodityColumns

	results, err := r.db.DB().
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
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateCommodityStatusRequest,
) ([]*commodity.Commodity, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*commodity.Commodity, 0, len(req.CommodityIDs))
	cols := buncolgen.CommodityColumns
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CommodityScopeTenantUpdate(uq, req.TenantInfo).
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
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetCommoditiesByIDsRequest,
) ([]*commodity.Commodity, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*commodity.Commodity, 0, len(req.CommodityIDs))
	cols := buncolgen.CommodityColumns
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Relation(buncolgen.CommodityRelations.HazardousMaterial).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CommodityScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.CommodityIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get commodities", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Commodity")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.CommoditySelectOptionsRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	cols := buncolgen.CommodityColumns

	return dbhelper.SelectOptions[*commodity.Commodity](
		ctx,
		r.db.DB(),
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
}
