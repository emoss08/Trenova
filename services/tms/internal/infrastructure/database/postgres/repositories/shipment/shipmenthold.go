package shipment

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ShipmentHoldRepositoryParams defines dependencies required for initializing the ShipmentHoldRepository.
// This includes database connection and logger.
type ShipmentHoldRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type shipmentHoldRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewShipmentHoldRepository initializes a new instance of shipmentHoldRepository with its dependencies.
//
// Parameters:
//   - p: ShipmentHoldRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.ShipmentHoldRepository: A ready-to-use shipment hold repository instance.
func NewShipmentHoldRepository(
	p ShipmentHoldRepositoryParams,
) repositories.ShipmentHoldRepository {
	log := p.Logger.With().
		Str("repository", "shipmenthold").
		Logger()
	return &shipmentHoldRepository{
		db: p.DB,
		l:  &log,
	}
}

func (sh *shipmentHoldRepository) GetByID(
	ctx context.Context,
	req *repositories.GetShipmentHoldByIDRequest,
) (*shipment.ShipmentHold, error) {
	dba, err := sh.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.In("shipment_hold_repository").
			With("op", "get_by_id").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sh.l.With().
		Str("operation", "get_by_id").
		Str("shipment_hold_id", req.ID.String()).
		Logger()

	entity := new(shipment.ShipmentHold)

	query := dba.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sh.id = ?", req.ID).
				Where("sh.organization_id = ?", req.OrgID).
				Where("sh.business_unit_id = ?", req.BuID)
		})

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Shipment hold not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get shipment hold")
		return nil, err
	}

	return entity, nil
}

// GetByShipmentID retrieves a shipment hold by its shipment ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: GetShipmentHoldByShipmentIDRequest containing Shipment ID and tentant options.
//
// Returns:
//   - *ports.ListResult[*shipment.ShipmentHold]: The shipment hold entities.
//   - error: An error if the operation fails.
func (sh *shipmentHoldRepository) GetByShipmentID(
	ctx context.Context,
	req *repositories.GetShipmentHoldByShipmentIDRequest,
) (*ports.ListResult[*shipment.ShipmentHold], error) {
	dba, err := sh.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.In("shipment_hold_repository").
			With("op", "get_by_shipment_id").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sh.l.With().
		Str("operation", "get_by_shipment_id").
		Str("shipment_id", req.ShipmentID.String()).
		Logger()

	entities := make([]*shipment.ShipmentHold, 0)

	q := dba.NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sh.shipment_id = ?", req.ShipmentID).
				Where("sh.organization_id = ?", req.OrgID).
				Where("sh.business_unit_id = ?", req.BuID)
		})

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan shipment holds")
		return nil, err
	}

	return &ports.ListResult[*shipment.ShipmentHold]{
		Items: entities,
		Total: total,
	}, nil
}

// Create creates a new shipment hold.
//
// Parameters:
//   - ctx: The context for the operation.
//   - hold: The shipment hold entity to create.
//
// Returns:
//   - *shipment.ShipmentHold: The created shipment hold entity.
//   - error: An error if the operation fails.
func (sh *shipmentHoldRepository) Create(
	ctx context.Context,
	hold *shipment.ShipmentHold,
) (*shipment.ShipmentHold, error) {
	dba, err := sh.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.In("shipment_hold_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sh.l.With().Interface("shipment_hold", hold).Logger()

	if _, err = dba.NewInsert().Model(hold).Returning("*").Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Msg("failed to insert shipment hold")
		return nil, err
	}

	return hold, nil
}

// Update updates a shipment hold.
//
// Parameters:
//   - ctx: The context for the operation.
//   - hold: The shipment hold entity to update.
//
// Returns:
//   - *shipment.ShipmentHold: The updated shipment hold entity.
//   - error: An error if the operation fails.
func (sh *shipmentHoldRepository) Update(
	ctx context.Context,
	hold *shipment.ShipmentHold,
) (*shipment.ShipmentHold, error) {
	dba, err := sh.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.In("shipment_hold_repository").
			With("op", "update").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sh.l.With().
		Str("operation", "update").
		Int64("version", hold.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := hold.Version
		hold.Version++

		results, rErr := tx.NewUpdate().
			Model(hold).
			WherePK().
			OmitZero().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("shipment_hold", hold).
				Msg("failed to update shipment hold")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("shipment_hold", hold).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				"version mismatch",
			)
		}

		return nil
	})
	if err != nil {
		log.Error().
			Err(err).
			Interface("shipment_hold", hold).
			Msg("failed to update shipment hold")
		return nil, err
	}

	return hold, nil
}
