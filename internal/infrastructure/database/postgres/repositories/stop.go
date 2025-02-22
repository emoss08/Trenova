package repositories

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type StopRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type stopRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewStopRepository(p StopRepositoryParams) repositories.StopRepository {
	log := p.Logger.With().
		Str("repository", "stop").
		Logger()

	return &stopRepository{
		db: p.DB,
		l:  &log,
	}
}

func (sr *stopRepository) GetByID(ctx context.Context, req repositories.GetStopByIDRequest) (*shipment.Stop, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetByID").
		Str("stopID", req.StopID.String()).
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	stop := new(shipment.Stop)

	q := dba.NewSelect().Model(stop).
		Where("stp.id = ?", req.StopID).
		Where("stp.organization_id = ?", req.OrgID).
		Where("stp.business_unit_id = ?", req.BuID)

	if req.ExpandStopDetails {
		q.Relation("Location")
	}

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get stop by id")
		return nil, err
	}

	return stop, nil
}

func (sr *stopRepository) BulkInsert(ctx context.Context, stops []*shipment.Stop) ([]*shipment.Stop, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "BulkInsert").
		Interface("stops", stops).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().Model(&stops).Exec(c); err != nil {
			log.Error().Err(err).Msg("failed to bulk insert stops")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk insert stops")
		return nil, err
	}

	return stops, nil
}

func (sr *stopRepository) Update(ctx context.Context, stop *shipment.Stop, moveIdx, stopIdx int) (*shipment.Stop, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Update").
		Str("stopID", stop.ID.String()).
		Str("orgID", stop.OrganizationID.String()).
		Str("buID", stop.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// ov := stop.Version

		stop.Version++

		results, rErr := tx.NewUpdate().
			Model(stop).
			Where("stp.id = ?", stop.ID).
			// Where("stp.version = ?", ov).
			Where("stp.shipment_move_id = ?", stop.ShipmentMoveID).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update stop")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		log.Debug().Int("rows", int(rows)).Msg("stop rows affected")

		if rows == 0 {
			return errors.NewValidationError(
				fmt.Sprintf("move[%d].stop[%d].version", moveIdx, stopIdx),
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Stop (%s) has either been updated or deleted since the last request.", stop.ID),
			)
		}

		log.Debug().Int("rows", int(rows)).Msg("updated stop")

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update stop")
		return nil, err
	}

	return stop, nil
}
