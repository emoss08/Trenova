package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
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

// ConsolidationSettingRepositoryParams contains the dependencies for the ConsolidationSettingRepository.
// This includes database connection and logger.
type ConsolidationSettingRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// consolidationSettingRepository implements the ConsolidationSettingRepository interface.
//
// It provides methods to interact with the consolidation setting table in the database.
type consolidationSettingRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewConsolidationSettingRepository initializes a new instance of consolidationSettingRepository with its dependencies.
//
// Parameters:
//   - p: ConsolidationSettingRepositoryParams containing database connection and logger.
//
// Returns:
//   - A new instance of consolidationSettingRepository.
func NewConsolidationSettingRepository(
	p ConsolidationSettingRepositoryParams,
) repositories.ConsolidationSettingRepository {
	log := p.Logger.With().
		Str("repository", "consolidationsetting").
		Logger()

	return &consolidationSettingRepository{
		db: p.DB,
		l:  &log,
	}
}

// GetByOrgID retrieves a consolidation setting by organization ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - orgID: The organization ID to filter by.
//
// Returns:
//   - *consolidation.ConsolidationSettings: The consolidation setting entity.
//   - error: If any database operation fails.
func (r consolidationSettingRepository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*consolidation.ConsolidationSettings, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByOrgID").
		Str("orgID", orgID.String()).
		Logger()

	entity := new(consolidation.ConsolidationSettings)

	query := dba.NewSelect().Model(entity).
		Where("cs.organization_id = ?", orgID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("consolidation setting not found within your organization")
			return nil, errors.NewNotFoundError(
				"Consolidation setting not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get consolidation setting")
		return nil, eris.Wrap(err, "get consolidation setting")
	}

	return entity, nil
}

// Update updates a singular consolidation setting entity.
//
// Parameters:
//   - ctx: The context for the operation.
//   - cs: The consolidation setting entity to update.
//
// Returns:
//   - *consolidation.ConsolidationSettings: The updated consolidation setting entity.
//   - error: If any database operation fails.
func (r consolidationSettingRepository) Update(
	ctx context.Context,
	cs *consolidation.ConsolidationSettings,
) (*consolidation.ConsolidationSettings, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("id", cs.GetID()).
		Int64("version", cs.GetVersion()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := cs.Version

		cs.Version++

		results, rErr := tx.NewUpdate().
			Model(cs).
			WherePK().
			Where("cs.version = ?", ov).
			OmitZero().
			Returning("*").
			Exec(c)
		if rErr != nil {
			if eris.Is(rErr, sql.ErrNoRows) {
				log.Error().Msg("Consolidation setting not found within your organization")
				return errors.NewNotFoundError(
					"Consolidation setting not found within your organization",
				)
			}

			log.Error().
				Err(rErr).
				Interface("consolidationsetting", cs).
				Msg("failed to update consolidation setting")
			return err
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("consolidationsetting", cs).
				Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			// * If the rows affected is 0, return a version mismatch error
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Consolidation Setting (%s) has either been updated or deleted since the last request.",
					cs.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update consolidation setting")
		return nil, err
	}

	return cs, nil
}
