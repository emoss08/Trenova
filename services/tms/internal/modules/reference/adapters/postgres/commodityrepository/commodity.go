package commodityrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var commodityErrors = dberror.Mapping{Entity: "Commodity"}

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

func (r *repository) List(
	ctx context.Context,
	req *commodityports.ListCommodityRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*commodity.Commodity], error) {
			c := commoditydb.Commodities
			result, err := listquery.List(ctx, r.db.Keel(ctx), c.Query(),
				&listquery.Request[commodity.Commodity]{
					Filter: req.Filter,
					Then:   []keel.Order{c.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list commodities", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req commodityports.GetCommodityByIDRequest,
) (*commodity.Commodity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*commodity.Commodity, error) {
		entity, err := commoditydb.Commodities.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, commodityErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *commodityports.ListCommodityConnectionRequest,
) (*pagination.CursorListResult[*commodity.Commodity], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*commodity.Commodity], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				commoditydb.Commodities.Query(),
				&listquery.Request[commodity.Commodity]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.CommodityColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list commodities", zap.Error(err))
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
		if err := commoditydb.Commodities.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create commodity",
				zap.String("name", entity.Name),
				zap.Error(err),
			)
			return nil, commodityErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *commodity.Commodity,
) (*commodity.Commodity, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*commodity.Commodity, error) {
		if err := commoditydb.Commodities.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update commodity",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, commodityErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *commodityports.BulkUpdateCommodityStatusRequest,
) ([]*commodity.Commodity, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*commodity.Commodity, error) {
		c := commoditydb.Commodities
		entities, err := c.Where(c.ID.In(req.CommodityIDs...)).
			UpdateReturning(ctx, r.db.Keel(ctx), c.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update commodity status", zap.Error(err))
			return nil, err
		}

		err = dberror.CheckBulkAffected(int64(len(entities)), "Commodity", req.CommodityIDs)
		if err != nil {
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
		if len(req.CommodityIDs) == 0 {
			return []*commodity.Commodity{}, nil
		}

		entities, err := commoditiesByIDs(req.CommodityIDs).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get commodities", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func commoditiesByIDs(ids []pulid.ID) *keel.Query[*commodity.Commodity] {
	c := commoditydb.Commodities

	return c.Where(c.ID.In(ids...)).Load(c.HazardousMaterial()).Limit(len(ids))
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *commodityports.CommoditySelectOptionsRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*commodity.Commodity], error) {
			c := commoditydb.Commodities
			q := c.Pick(c.ID, c.Name, c.HazardousMaterialID, c.FreightClass).
				Where(c.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(c.Name.ContainsFold(s), c.Description.ContainsFold(s))
				},
			)
		},
	)
}
