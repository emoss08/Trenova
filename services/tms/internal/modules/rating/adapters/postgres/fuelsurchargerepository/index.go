package fuelsurchargerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge/fuelsurchargedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var fuelIndexErrors = dberror.Mapping{Entity: "FuelIndex"}

type IndexParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type indexRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewIndexRepository(p IndexParams) fuelsurchargeports.FuelIndexRepository {
	return &indexRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fuel-index-repository"),
	}
}

func (r *indexRepository) ListConnection(
	ctx context.Context,
	req *fuelsurchargeports.ListFuelIndexConnectionRequest,
) (*pagination.CursorListResult[*fuelsurcharge.FuelIndex], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fuelsurcharge.FuelIndex], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				fuelsurchargedb.FuelIndexes.Query(),
				&listquery.Request[fuelsurcharge.FuelIndex]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.FuelIndexColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list fuel indices", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func activeIndexesQuery(source *fuelsurcharge.IndexSource) *keel.Query[*fuelsurcharge.FuelIndex] {
	fi := fuelsurchargedb.FuelIndexes
	q := fi.Where(fi.IsActive.IsTrue())
	if source != nil {
		q = q.Where(fi.Source.Eq(*source))
	}

	return q.Order(fi.Code.Asc())
}

func (r *indexRepository) listBySource(
	ctx context.Context,
	source *fuelsurcharge.IndexSource,
) ([]*fuelsurcharge.FuelIndex, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelsurcharge.FuelIndex, error) {
		return activeIndexesQuery(source).All(ctx, r.db.Keel(ctx))
	})
}

func (r *indexRepository) ListActiveEIA(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*fuelsurcharge.FuelIndex, error) {
	source := fuelsurcharge.IndexSourceEIA
	entities, err := r.listBySource(ctx, &source)
	if err != nil {
		r.l.Error("failed to list active EIA fuel indices", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *indexRepository) ListActive(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*fuelsurcharge.FuelIndex, error) {
	entities, err := r.listBySource(ctx, nil)
	if err != nil {
		r.l.Error("failed to list active fuel indices", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *indexRepository) GetByID(
	ctx context.Context,
	req *fuelsurchargeports.GetFuelIndexByIDRequest,
) (*fuelsurcharge.FuelIndex, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelsurcharge.FuelIndex, error) {
		entity, err := fuelsurchargedb.FuelIndexes.Get(ctx, r.db.Keel(ctx), req.FuelIndexID)
		if err != nil {
			r.l.Error("failed to get fuel index",
				zap.String("id", req.FuelIndexID.String()),
				zap.Error(err),
			)
			return nil, fuelIndexErrors.Map(err, req.FuelIndexID.String())
		}

		return entity, nil
	})
}

func (r *indexRepository) GetByIDs(
	ctx context.Context,
	req *fuelsurchargeports.GetFuelIndexesByIDsRequest,
) ([]*fuelsurcharge.FuelIndex, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelsurcharge.FuelIndex, error) {
		if len(req.FuelIndexIDs) == 0 {
			return []*fuelsurcharge.FuelIndex{}, nil
		}

		fi := fuelsurchargedb.FuelIndexes
		entities, err := fi.Where(fi.ID.In(req.FuelIndexIDs...)).
			Limit(len(req.FuelIndexIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get fuel indexes", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *indexRepository) Create(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndex,
) (*fuelsurcharge.FuelIndex, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelsurcharge.FuelIndex, error) {
		if err := fuelsurchargedb.FuelIndexes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create fuel index", zap.Error(err))
			return nil, fuelIndexErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *indexRepository) Update(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndex,
) (*fuelsurcharge.FuelIndex, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelsurcharge.FuelIndex, error) {
		if err := fuelsurchargedb.FuelIndexes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update fuel index",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, fuelIndexErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *indexRepository) Delete(
	ctx context.Context,
	req *fuelsurchargeports.GetFuelIndexByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := fuelsurchargedb.FuelIndexes.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&fuelsurcharge.FuelIndex{ID: req.FuelIndexID},
		)
		if err != nil {
			r.l.Error("failed to delete fuel index",
				zap.String("id", req.FuelIndexID.String()),
				zap.Error(err),
			)
			return fuelIndexErrors.Map(err, req.FuelIndexID.String())
		}

		return nil
	})
}

func indexOptionsQuery() *keel.Query[*fuelsurcharge.FuelIndex] {
	fi := fuelsurchargedb.FuelIndexes

	return fi.Pick(fi.ID, fi.Name, fi.Code, fi.Description, fi.Source, fi.EIASeriesID, fi.IsActive).
		Where(fi.IsActive.IsTrue()).
		Order(fi.Code.Asc())
}

func (r *indexRepository) SelectOptions(
	ctx context.Context,
	req *fuelsurchargeports.FuelIndexSelectOptionsRequest,
) (*pagination.ListResult[*fuelsurcharge.FuelIndex], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*fuelsurcharge.FuelIndex], error) {
			fi := fuelsurchargedb.FuelIndexes

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				indexOptionsQuery(),
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						fi.Name.ContainsFold(s),
						fi.Code.ContainsFold(s),
						fi.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}
