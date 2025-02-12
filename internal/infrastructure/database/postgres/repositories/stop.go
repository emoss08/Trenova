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

func (sr *stopRepository) Update(ctx context.Context, stp *shipment.Stop) (*shipment.Stop, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Update").
		Str("id", stp.ID.String()).
		Int64("version", stp.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := stp.Version

		stp.Version++

		results, rErr := tx.NewUpdate().
			Model(stp).
			WherePK().
			Where("stp.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("stop", stp).
				Msg("failed to update stop")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("stop", stp).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Stop (%s) has either been updated or deleted since the last requestp.", stp.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("transaction failed to update stop")
		return nil, err
	}

	return stp, nil
}
