package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
