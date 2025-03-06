package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ShipmentControlRepositoryParams contains the dependencies for the ShipmentControlRepository.
// This includes database connection and logger.
type ShipmentControlRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// shipmentControlRepository implements the ShipmentControlRepository interface.
//
// It provides methods to interact with the shipment control table in the database.
type shipmentControlRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewShipmentControlRepository initializes a new instance of shipmentControlRepository with its dependencies.
//
// Parameters:
//   - p: ShipmentControlRepositoryParams containing database connection and logger.
//
// Returns:
//   - A new instance of shipmentControlRepository.
func NewShipmentControlRepository(p ShipmentControlRepositoryParams) repositories.ShipmentControlRepository {
	log := p.Logger.With().
		Str("repository", "shipmentcontrol").
		Logger()

	return &shipmentControlRepository{
		db: p.DB,
		l:  &log,
	}
}

// GetByOrgID retrieves a shipment control by organization ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - orgID: The organization ID to filter by.
//
// Returns:
//   - *shipment.ShipmentControl: The shipment control entity.
//   - error: If any database operation fails.
func (r shipmentControlRepository) GetByOrgID(ctx context.Context, orgID pulid.ID) (*shipment.ShipmentControl, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByOrgID").
		Str("orgID", orgID.String()).
		Logger()

	entity := new(shipment.ShipmentControl)

	query := dba.NewSelect().Model(entity).Where("sc.organization_id = ?", orgID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("shipment control not found within your organization")
			return nil, errors.NewNotFoundError("Shipment control not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get shipment control")
		return nil, eris.Wrap(err, "get shipment control")
	}

	return entity, nil
}

// Update updates a singular shipment control entity.
//
// Parameters:
//   - ctx: The context for the operation.
//   - sc: The shipment control entity to update.
//
// Returns:
//   - *shipment.ShipmentControl: The updated shipment control entity.
//   - error: If any database operation fails.
func (r shipmentControlRepository) Update(ctx context.Context, sc *shipment.ShipmentControl) (*shipment.ShipmentControl, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("id", sc.GetID()).
		Int64("version", sc.GetVersion()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := sc.Version

		sc.Version++

		results, rErr := tx.NewUpdate().
			Model(sc).
			WherePK().
			Where("sc.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			// * If the query is [sql.ErrNoRows], return a not found error
			if eris.Is(rErr, sql.ErrNoRows) {
				log.Error().Msg("shipment control not found within your organization")
				return errors.NewNotFoundError("Shipment control not found within your organization")
			}

			log.Error().
				Err(rErr).
				Interface("shipmentcontrol", sc).
				Msg("failed to update shipment control")
			return err
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("shipmentcontrol", sc).
				Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			// * If the rows affected is 0, return a version mismatch error
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Shipment Control (%s) has either been updated or deleted since the last request.", sc.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment control")
		return nil, err
	}

	return sc, nil
}
