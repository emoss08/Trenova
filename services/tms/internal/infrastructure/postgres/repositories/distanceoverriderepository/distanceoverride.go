package distanceoverriderepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
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

func NewRepository(p Params) repositories.DistanceOverrideRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.distanceoverride-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDistanceOverrideRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"diso",
		req.Filter,
		(*distanceoverride.Override)(nil),
	)

	if req.ExpandDetails {
		q = q.Relation("OriginLocation").
			Relation("OriginLocation.State").
			Relation("DestinationLocation").
			Relation("DestinationLocation.State").
			Relation("Customer")
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDistanceOverrideRequest,
) (*pagination.ListResult[*distanceoverride.Override], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgID", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buID", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*distanceoverride.Override, 0, req.Filter.Limit)
	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan distance overrides", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*distanceoverride.Override]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByLocationIDs(
	ctx context.Context,
	req *repositories.GetByLocationIDsRequest,
) (*distanceoverride.Override, error) {
	log := r.l.With(
		zap.String("operation", "GetByLocationIDs"),
		zap.String("originLocationID", req.OriginLocationID.String()),
		zap.String("destinationLocationID", req.DestinationLocationID.String()),
		zap.String("orgID", req.OrgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(distanceoverride.Override)
	err = db.NewSelect().
		Model(entity).
		Distinct().
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("diso.origin_location_id = ?", req.OriginLocationID).
				Where("diso.destination_location_id = ?", req.DestinationLocationID).
				Where("diso.organization_id = ?", req.OrgID).
				Where("diso.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get distance override", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DistanceOverride")
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetDistanceOverrideRequest,
) (*distanceoverride.Override, error) {
	log := r.l.With(
		zap.String("operation", "Get"),
		zap.String("id", req.ID.String()),
		zap.String("orgID", req.OrgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(distanceoverride.Override)
	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("diso.id = ?", req.ID).
				Where("diso.organization_id = ?", req.OrgID).
				Where("diso.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get distance override", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DistanceOverride")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *distanceoverride.Override,
) (*distanceoverride.Override, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert distance override", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *distanceoverride.Override,
) (*distanceoverride.Override, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.GetID()),
		zap.String("orgID", entity.GetOrganizationID().String()),
		zap.String("buID", entity.GetBusinessUnitID().String()),
		zap.Int64("version", entity.Version),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("diso.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update distance override", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Distance Override", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteDistanceOverrideRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewDelete().Model((*distanceoverride.Override)(nil)).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.
				Where("diso.id = ?", req.ID).
				Where("diso.organization_id = ?", req.OrgID).
				Where("diso.business_unit_id = ?", req.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete distance override", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Distance Override", req.ID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}
