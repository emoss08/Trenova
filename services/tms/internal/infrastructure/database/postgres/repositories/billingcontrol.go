/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billing"
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

// BillingControlRepositoryParams contains the dependencies for the BillingControlRepository.
// This includes database connection and logger.
type BillingControlRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// billingControlRepository implements the BillingControlRepository interface.
//
// It provides methods to interact with the billing control table in the database.
type billingControlRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewBillingControlRepository initializes a new instance of billingControlRepository with its dependencies.
//
// Parameters:
//   - p: BillingControlRepositoryParams containing database connection and logger.
//
// Returns:
//   - A new instance of billingControlRepository.
func NewBillingControlRepository(
	p BillingControlRepositoryParams,
) repositories.BillingControlRepository {
	log := p.Logger.With().
		Str("repository", "billingcontrol").
		Logger()

	return &billingControlRepository{
		db: p.DB,
		l:  &log,
	}
}

// GetByOrgID retrieves a billing control by organization ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - orgID: The organization ID to filter by.
//
// Returns:
//   - *billing.BillingControl: The billing control entity.
//   - error: If any database operation fails.
func (r *billingControlRepository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*billing.BillingControl, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := r.l.With().
		Str("operation", "GetByOrgID").
		Str("orgID", orgID.String()).
		Logger()

	entity, err := billing.NewBillingControlQuery(dba).
		WhereOrganizationIDEQ(orgID).
		First(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("billing control not found within your organization")

			return nil, errors.NewNotFoundError(
				"Billing control not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get billing control")
		return nil, err
	}

	return entity, nil
}

// Update updates a singular billing control entity.
//
// Parameters:
//   - ctx: The context for the operation.
//   - bc: The billing control entity to update.
//
// Returns:
//   - *billing.BillingControl: The updated billing control entity.
//   - error: If any database operation fails.
func (r *billingControlRepository) Update(
	ctx context.Context,
	bc *billing.BillingControl,
) (*billing.BillingControl, error) {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return nil, err
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("id", bc.GetID()).
		Int64("version", bc.GetVersion()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := bc.Version

		bc.Version++

		results, rErr := tx.NewUpdate().
			Model(bc).
			WherePK().
			Where("bc.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update billing control")
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
				fmt.Sprintf(
					"Version mismatch. The Billing Control (%s) has either been updated or deleted since the last request.",
					bc.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update billing control")
		return nil, err
	}

	return bc, nil
}
