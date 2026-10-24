package locationcategoryrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var locationCategoryErrors = dberror.Mapping{
	Entity: "LocationCategory",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_location_categories_name",
			Field:   "name",
			Message: "Location category with this name already exists in your organization",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) locationcategoryports.LocationCategoryRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.location-category-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *locationcategoryports.ListLocationCategoriesRequest,
) (*pagination.ListResult[*locationcategory.LocationCategory], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*locationcategory.LocationCategory], error) {
			lc := locationcategorydb.LocationCategories
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				lc.Query(),
				&listquery.Request[locationcategory.LocationCategory]{
					Filter: req.Filter,
					Then:   []keel.Order{lc.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list location categories", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *locationcategoryports.ListLocationCategoryConnectionRequest,
) (*pagination.CursorListResult[*locationcategory.LocationCategory], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*locationcategory.LocationCategory], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				locationcategorydb.LocationCategories.Query(),
				&listquery.Request[locationcategory.LocationCategory]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.LocationCategoryColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list location categories", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *locationcategory.LocationCategory,
) (*locationcategory.LocationCategory, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*locationcategory.LocationCategory, error) {
			if err := locationcategorydb.LocationCategories.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error(
					"failed to create location category",
					zap.String("name", entity.Name),
					zap.Error(err),
				)
				return nil, locationCategoryErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *locationcategory.LocationCategory,
) (*locationcategory.LocationCategory, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*locationcategory.LocationCategory, error) {
			if err := locationcategorydb.LocationCategories.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error(
					"failed to update location category",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, locationCategoryErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req locationcategoryports.GetLocationCategoryByIDRequest,
) (*locationcategory.LocationCategory, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*locationcategory.LocationCategory, error) {
			entity, err := locationcategorydb.LocationCategories.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				r.l.Error(
					"failed to get location category",
					zap.String("id", req.ID.String()),
					zap.Error(err),
				)
				return nil, locationCategoryErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req locationcategoryports.GetLocationCategoriesByIDsRequest,
) ([]*locationcategory.LocationCategory, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*locationcategory.LocationCategory, error) {
			if len(req.LocationCategoryIDs) == 0 {
				return []*locationcategory.LocationCategory{}, nil
			}

			lc := locationcategorydb.LocationCategories
			entities, err := lc.Where(lc.ID.In(req.LocationCategoryIDs...)).
				Limit(len(req.LocationCategoryIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get location categories", zap.Error(err))
				return nil, err
			}
			return entities, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*locationcategory.LocationCategory], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*locationcategory.LocationCategory], error) {
			lc := locationcategorydb.LocationCategories
			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				lc.Pick(lc.ID, lc.Name, lc.Description, lc.Type, lc.Color),
				req,
				func(s string) keel.Pred {
					return keel.Or(lc.Name.ContainsFold(s), lc.Description.ContainsFold(s))
				},
			)
		},
	)
}
