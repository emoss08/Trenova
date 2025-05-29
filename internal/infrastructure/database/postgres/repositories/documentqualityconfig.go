package repositories

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/core/domain/documentqualityconfig"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type DocumentQualityConfigRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type documentQualityConfigRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewDocumentQualityConfigRepository(
	p DocumentQualityConfigRepositoryParams,
) repositories.DocumentQualityConfigRepository {
	log := p.Logger.With().
		Str("repository", "documentqualityconfig").
		Logger()

	return &documentQualityConfigRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *documentQualityConfigRepository) Get(
	ctx context.Context,
	opts *repositories.GetDocumentQualityConfigOptions,
) (*documentqualityconfig.DocumentQualityConfig, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().Str("operation", "Get").Logger()

	dqc := new(documentqualityconfig.DocumentQualityConfig)

	query := dba.NewSelect().Model(dqc).
		Where("dqc.organization_id = ? AND dqc.business_unit_id = ?", opts.OrgID, opts.BuID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError(
				"Document Quality Config not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get document quality config")
		return nil, eris.Wrap(err, "get document quality config")
	}

	return dqc, nil
}

func (r *documentQualityConfigRepository) Update(
	ctx context.Context,
	dqc *documentqualityconfig.DocumentQualityConfig,
) (*documentqualityconfig.DocumentQualityConfig, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().Str("operation", "Update").
		Int64("version", dqc.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := dqc.Version

		dqc.Version++

		results, rErr := tx.NewUpdate().
			Model(dqc).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("documentqualityconfig", dqc).
				Msg("failed to update document quality config")
			return eris.Wrap(rErr, "update document quality config")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("documentqualityconfig", dqc).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				"Version mismatch. The document quality config has either been updated or deleted since the last request.",
			)
		}

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "update document quality config")
	}

	return dqc, nil
}
