package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// DedicatedLaneRepositoryParams defines dependencies required for initializing the DedicatedLaneRepository.
// This includes database connection and logger.
type DedicatedLaneRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// dedicatedLaneRepository implements the DedicatedLaneRepository interface
// and provides methods to manage dedicated lane data, including CRUD operations.
type dedicatedLaneRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewDedicatedLaneRepository initializes a new instance of dedicatedLaneRepository with its dependencies.
//
// Parameters:
//   - p: DedicatedLaneRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.DedicatedLaneRepository: A ready-to-use dedicated lane repository instance.
func NewDedicatedLaneRepository(
	p DedicatedLaneRepositoryParams,
) repositories.DedicatedLaneRepository {
	log := p.Logger.With().
		Str("repository", "dedicated_lane").
		Logger()

	return &dedicatedLaneRepository{
		db: p.DB,
		l:  &log,
	}
}

func (dlr *dedicatedLaneRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDedicatedLaneRequest,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "dl",
		Filter:     req.Filter,
	})

	relations := make([]string, 0)

	if req.FilterOptions.ExpandDetails {
		relations = append(
			relations,
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

	for _, rel := range relations {
		q = q.Relation(rel)
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (dlr *dedicatedLaneRepository) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneRequest,
) (*ports.ListResult[*dedicatedlane.DedicatedLane], error) {
	dba, err := dlr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_repository").
			With("op", "list").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlr.l.With().
		Str("op", "list").
		Interface("tenantOps", req.Filter.TenantOpts).
		Logger()

	entities := make([]*dedicatedlane.DedicatedLane, 0)

	q := dba.NewSelect().Model(&entities)
	q = dlr.filterQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan and count dedicated lanes")
		return nil, err
	}

	return &ports.ListResult[*dedicatedlane.DedicatedLane]{
		Total: total,
		Items: entities,
	}, nil
}

func (dlr *dedicatedLaneRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDedicatedLaneByIDRequest,
) (*dedicatedlane.DedicatedLane, error) {
	dba, err := dlr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_repository").
			With("op", "get_by_id").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlr.l.With().
		Str("op", "get_by_id").
		Interface("req", req).
		Logger()

	entity := &dedicatedlane.DedicatedLane{}

	q := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("dl.organization_id = ?", req.OrgID).
				Where("dl.business_unit_id = ?", req.BuID).
				Where("dl.id = ?", req.ID)
		})

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, oops.In("dedicated_lane_repository").
				With("op", "get_by_id").
				Time(time.Now()).
				Wrapf(err, "no dedicated lane found")
		}

		log.Error().Err(err).Msg("failed to get dedicated lane")
		return nil, oops.In("dedicated_lane_repository").
			With("op", "get_by_id").
			Time(time.Now()).
			Wrapf(err, "get dedicated lane")
	}

	return entity, nil
}

func (dlr *dedicatedLaneRepository) Create(
	ctx context.Context,
	dl *dedicatedlane.DedicatedLane,
) (*dedicatedlane.DedicatedLane, error) {
	dba, err := dlr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlr.l.With().
		Str("op", "create").
		Interface("req", dl).
		Logger()

	if _, err = dba.NewInsert().Model(dl).Returning("*").Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to create dedicated lane")
		return nil, oops.In("dedicated_lane_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "create dedicated lane")
	}

	return dl, nil
}

func (dlr *dedicatedLaneRepository) FindByShipment(
	ctx context.Context,
	req *repositories.FindDedicatedLaneByShipmentRequest,
) (*dedicatedlane.DedicatedLane, error) {
	dba, err := dlr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_repository").
			With("op", "find_by_shipment").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlr.l.With().
		Str("op", "find_by_shipment").
		Interface("req", req).
		Logger()

	dl := new(dedicatedlane.DedicatedLane)

	query := dba.NewSelect().Model(dl).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			q := sq.
				Where("dl.status = ?", domain.StatusActive).
				Where("dl.organization_id = ?", req.OrganizationID).
				Where("dl.business_unit_id = ?", req.BusinessUnitID).
				Where("dl.customer_id = ?", req.CustomerID).
				Where("dl.origin_location_id = ?", req.OriginLocationID).
				Where("dl.destination_location_id = ?", req.DestinationLocationID)

			// ServiceTypeID and ShipmentTypeID are required fields
			if lo.IsNotNil(req.ServiceTypeID) {
				q = q.Where("dl.service_type_id = ?", *req.ServiceTypeID)
			}
			if lo.IsNotNil(req.ShipmentTypeID) {
				q = q.Where("dl.shipment_type_id = ?", *req.ShipmentTypeID)
			}

			// Handle optional trailer type - match if both are specified and equal, or both are null
			if lo.IsNotNil(req.TrailerTypeID) {
				q = q.Where("dl.trailer_type_id = ?", *req.TrailerTypeID)
			} else {
				q = q.Where("dl.trailer_type_id IS NULL")
			}

			// Handle optional tractor type - match if both are specified and equal, or both are null
			if lo.IsNotNil(req.TractorTypeID) {
				q = q.Where("dl.tractor_type_id = ?", *req.TractorTypeID)
			} else {
				q = q.Where("dl.tractor_type_id IS NULL")
			}

			return q
		}).
		Order("dl.created_at DESC").
		Limit(1)

	err = query.Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Warn().Msg("no dedicated lane found for shipment")
			return nil, errors.NewNotFoundError("no dedicated lane found for shipment")
		}

		return nil, oops.In("dedicated_lane_repository").
			With("op", "find_by_shipment").
			Time(time.Now()).
			Wrapf(err, "query dedicated lane")
	}

	return dl, nil
}

func (dlr *dedicatedLaneRepository) Update(
	ctx context.Context,
	dl *dedicatedlane.DedicatedLane,
) (*dedicatedlane.DedicatedLane, error) {
	dba, err := dlr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("dedicated_lane_repository").
			With("op", "update").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := dlr.l.With().
		Str("op", "update").
		Interface("req", dl).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := dl.Version

		dl.Version++

		results, rErr := tx.NewUpdate().
			Model(dl).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("dl.id = ?", dl.ID).
					Where("dl.organization_id = ?", dl.OrganizationID).
					Where("dl.business_unit_id = ?", dl.BusinessUnitID).
					Where("dl.version = ?", ov)
			}).
			OmitZero().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update dedicated lane")
			return eris.Wrap(rErr, "update dedicated lane")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Dedicated Lane (%s) has either been updated or deleted since the last request.",
					dl.ID,
				),
			)
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to update dedicated lane")
		return nil, oops.In("dedicated_lane_repository").
			With("op", "update").
			Time(time.Now()).
			Wrapf(err, "update dedicated lane")
	}

	return dl, nil
}
