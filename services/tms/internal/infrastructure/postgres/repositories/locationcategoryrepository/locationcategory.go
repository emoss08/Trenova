package locationcategoryrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.LocationCategoryRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.location-category-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListLocationCategoriesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"lc",
		req.Filter,
		(*locationcategory.LocationCategory)(nil),
	)

	q = q.Order("lc.created_at DESC")

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListLocationCategoriesRequest,
) (*pagination.ListResult[*locationcategory.LocationCategory], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*locationcategory.LocationCategory, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count location categories", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*locationcategory.LocationCategory]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *locationcategory.LocationCategory,
) (*locationcategory.LocationCategory, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create location category", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *locationcategory.LocationCategory,
) (*locationcategory.LocationCategory, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update location category", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "LocationCategory", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetLocationCategoryByIDRequest,
) (*locationcategory.LocationCategory, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(locationcategory.LocationCategory)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("lc.id = ?", req.ID).
				Where("lc.organization_id = ?", req.TenantInfo.OrgID).
				Where("lc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get location category", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "LocationCategory")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*locationcategory.LocationCategory], error) {
	return dbhelper.SelectOptions[*locationcategory.LocationCategory](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			Columns:       []string{"id", "name", "description", "type", "color"},
			OrgColumn:     "lc.organization_id",
			BuColumn:      "lc.business_unit_id",
			EntityName:    "LocationCategory",
			SearchColumns: []string{"lc.name", "lc.description"},
		},
	)
}
