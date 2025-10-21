package locationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(p Params) repositories.LocationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.location-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListLocationRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"loc",
		req.Filter,
		(*location.Location)(nil),
	)

	if req.IncludeCategory {
		q = q.Relation("LocationCategory")
	}

	if req.IncludeState {
		q = q.Relation("State")
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListLocationRequest,
) (*pagination.ListResult[*location.Location], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*location.Location, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan locations", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*location.Location]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetLocationByIDRequest,
) (*location.Location, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(location.Location)
	query := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("loc.id = ?", req.ID).
				Where("loc.organization_id = ?", req.OrgID).
				Where("loc.business_unit_id = ?", req.BuID)
		})

	if req.IncludeCategory {
		query.Relation("LocationCategory")
	}

	if req.IncludeState {
		query.Relation("State")
	}

	if err = query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Location")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *location.Location,
) (*location.Location, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity = r.geocodeIfApplicable(entity)

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert location", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *location.Location,
) (*location.Location, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	entity = r.geocodeIfApplicable(entity)

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("loc.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update location", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Location", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) geocodeIfApplicable(entity *location.Location) *location.Location {
	if entity.PlaceID == "" || entity.Latitude == nil || entity.Longitude == nil {
		entity.IsGeocoded = false
		return entity
	}

	entity.IsGeocoded = true
	return entity
}
