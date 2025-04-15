package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/googlemapsconfig"
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

// GoogleMapsConfigRepositoryParams contains the dependencies for the GoogleMapsConfigRepository.
// This includes database connection and logger.
type GoogleMapsConfigRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// googleMapsConfigRepository implements the GoogleMapsConfigRepository interface.
//
// It provides methods to interact with the google maps config table in the database.
type googleMapsConfigRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewGoogleMapsConfigRepository initializes a new instance of googleMapsConfigRepository with its dependencies.
//
// Parameters:
//   - p: GoogleMapsConfigRepositoryParams containing database connection and logger.
//
// Returns:
//   - A new instance of googleMapsConfigRepository.
func NewGoogleMapsConfigRepository(p GoogleMapsConfigRepositoryParams) repositories.GoogleMapsConfigRepository {
	log := p.Logger.With().
		Str("repository", "googlemapsconfig").
		Logger()

	return &googleMapsConfigRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *googleMapsConfigRepository) GetByOrgID(ctx context.Context, orgID pulid.ID) (*googlemapsconfig.GoogleMapsConfig, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByOrgID").
		Str("orgID", orgID.String()).
		Logger()

	entity := new(googlemapsconfig.GoogleMapsConfig)

	q := dba.NewSelect().Model(entity).
		Where("gmc.organization_id = ?", orgID)

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("google maps config not found within your organization")
			return nil, errors.NewNotFoundError("Google maps config not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get google maps config")
		return nil, eris.Wrap(err, "get google maps config")
	}

	return entity, nil
}

func (r *googleMapsConfigRepository) GetAPIKeyByOrgID(ctx context.Context, orgID pulid.ID) (string, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return "", eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetAPIKeyByOrgID").
		Str("orgID", orgID.String()).
		Logger()

	entity := new(googlemapsconfig.GoogleMapsConfig)

	q := dba.NewSelect().Model(entity).
		Where("gmc.organization_id = ?", orgID).
		Column("gmc.api_key")

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("google maps config not found within your organization")
			return "", errors.NewNotFoundError("Google maps config not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get google maps config")
		return "", eris.Wrap(err, "get google maps config")
	}

	return entity.APIKey, nil
}

func (r *googleMapsConfigRepository) Update(ctx context.Context, gmc *googlemapsconfig.GoogleMapsConfig) (*googlemapsconfig.GoogleMapsConfig, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("id", gmc.GetID()).
		Int64("version", gmc.GetVersion()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := gmc.Version

		gmc.Version++

		results, rErr := tx.NewUpdate().
			Model(gmc).
			WherePK().
			Where("gmc.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			if eris.Is(rErr, sql.ErrNoRows) {
				log.Error().Msg("google maps config not found within your organization")
				return errors.NewNotFoundError("Google maps config not found within your organization")
			}

			log.Error().Err(rErr).Msg("failed to update google maps config")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Google Maps Config (%s) has either been updated or deleted since the last request.", gmc.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update google maps config")
		return nil, err
	}

	return gmc, nil
}
