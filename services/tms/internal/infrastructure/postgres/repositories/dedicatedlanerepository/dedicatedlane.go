package dedicatedlanerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
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

func NewRepository(p Params) repositories.DedicatedLaneRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dedicatedlane-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.DedicatedLaneFilterOptions,
) *bun.SelectQuery {
	relations := []string{}

	if opts.ExpandDetails {
		relations = append(relations,
			"ShipmentType",
			"ServiceType",
			"Customer",
			"TractorType",
			"TrailerType",
			"OriginLocation",
			"OriginLocation.State",
			"DestinationLocation",
			"DestinationLocation.State",
			"PrimaryWorker",
			"SecondaryWorker",
		)
	}

	for _, relation := range relations {
		q = q.Relation(relation)
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDedicatedLaneRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"dl",
		req.Filter,
		(*dedicatedlane.DedicatedLane)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.FilterOptions)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneRequest,
) (*pagination.ListResult[*dedicatedlane.DedicatedLane], error) {
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

	entities := make([]*dedicatedlane.DedicatedLane, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan dedicated lanes", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*dedicatedlane.DedicatedLane]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetDedicatedLaneByIDRequest,
) (*dedicatedlane.DedicatedLane, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(dedicatedlane.DedicatedLane)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("dl.id = ?", req.ID).
				Where("dl.organization_id = ?", req.OrgID).
				Where("dl.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Dedicated Lane")
	}

	return entity, nil
}

func (r *repository) FindByShipment(
	ctx context.Context,
	req *repositories.FindDedicatedLaneByShipmentRequest,
) (*dedicatedlane.DedicatedLane, error) {
	log := r.l.With(
		zap.String("operation", "FindByShipment"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(dedicatedlane.DedicatedLane)
	err = db.NewSelect().
		Model(entity).
		Distinct().
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = sq.Where("dl.status = ?", domain.StatusActive).
				Where("dl.organization_id = ?", req.OrganizationID).
				Where("dl.business_unit_id = ?", req.BusinessUnitID).
				Where("dl.customer_id = ?", req.CustomerID).
				Where("dl.origin_location_id = ?", req.OriginLocationID).
				Where("dl.destination_location_id = ?", req.DestinationLocationID)

			if req.ServiceTypeID.IsNotNil() {
				sq = sq.Where("dl.service_type_id = ?", req.ServiceTypeID.String())
			}

			if req.ShipmentTypeID.IsNotNil() {
				sq = sq.Where("dl.shipment_type_id = ?", req.ShipmentTypeID.String())
			}

			if req.TrailerTypeID.IsNotNil() {
				sq = sq.Where("dl.trailer_type_id = ?", req.TrailerTypeID.String())
			} else {
				sq = sq.Where("dl.trailer_type_id IS NULL")
			}

			if req.TractorTypeID.IsNotNil() {
				sq = sq.Where("dl.tractor_type_id = ?", req.TractorTypeID.String())
			} else {
				sq = sq.Where("dl.tractor_type_id IS NULL")
			}

			return sq
		}).
		Order("dl.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Dedicated Lane")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *dedicatedlane.DedicatedLane,
) (*dedicatedlane.DedicatedLane, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert dedicated lane", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *dedicatedlane.DedicatedLane,
) (*dedicatedlane.DedicatedLane, error) {
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

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("dl.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update dedicated lane", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Dedicated Lane", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}
