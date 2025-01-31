package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ShipmentRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type shipmentRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewShipmentRepository(p ShipmentRepositoryParams) repositories.ShipmentRepository {
	log := p.Logger.With().
		Str("repository", "shipment").
		Logger()

	return &shipmentRepository{
		db: p.DB,
		l:  &log,
	}
}

func (sr *shipmentRepository) filterQuery(q *bun.SelectQuery, opts *repositories.ListShipmentOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "sp",
		Filter:     opts.Filter,
	})

	if opts.Filter.Query != "" {
		q = q.Where("sp.pro_number ILIKE ?", "%"+opts.Filter.Query+"%")
	}

	if opts.IncludeMoveDetails {
		q = q.Relation("Moves").Relation("Moves.Stops")
	}

	// ! IncludeMoveDetails must be true to include StopDetails
	if opts.IncludeStopDetails {
		q = q.Relation("Moves.Stops.Location")
	}

	if opts.IncludeCommodityDetails {
		q = q.Relation("Commodities")
	}

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (sr *shipmentRepository) List(ctx context.Context, opts *repositories.ListShipmentOptions) (*ports.ListResult[*shipment.Shipment], error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*shipment.Shipment, 0)

	q := dba.NewSelect().Model(&entities)
	q = sr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan shipments")
		return nil, err
	}

	return &ports.ListResult[*shipment.Shipment]{
		Items: entities,
		Total: total,
	}, nil
}

func (sr *shipmentRepository) GetByID(ctx context.Context, opts repositories.GetShipmentByIDOptions) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetByID").
		Str("shipmentID", opts.ID.String()).
		Logger()

	entity := new(shipment.Shipment)

	query := dba.NewSelect().Model(entity).
		Where("sp.id = ? AND sp.organization_id = ? AND sp.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if opts.IncludeMoveDetails {
		query = query.Relation("Moves").Relation("Moves.Stops")
	}

	// ! IncludeMoveDetails must be true to include StopDetails
	if opts.IncludeStopDetails {
		query = query.Relation("Moves.Stops.Location")
	}

	if opts.IncludeCommodityDetails {
		query = query.Relation("Commodities")
	}

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Shipment not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get shipment")
		return nil, err
	}

	return entity, nil
}

func (sr *shipmentRepository) Create(ctx context.Context, shp *shipment.Shipment) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Create").
		Str("orgID", shp.OrganizationID.String()).
		Str("buID", shp.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(shp).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("shipment", shp).
				Msg("failed to insert shipment")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create shipment")
		return nil, err
	}

	return shp, nil
}

func (sr *shipmentRepository) Update(ctx context.Context, shp *shipment.Shipment) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Update").
		Str("id", shp.GetID()).
		Int64("version", shp.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := shp.Version

		shp.Version++

		results, rErr := tx.NewUpdate().
			Model(shp).
			WherePK().
			Where("sp.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("shipment", shp).
				Msg("failed to update shipment")
			return err
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("shipment", shp).
				Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Shipment (%s) has either been updated or deleted since the last request.", shp.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment")
		return nil, err
	}

	return shp, nil
}
