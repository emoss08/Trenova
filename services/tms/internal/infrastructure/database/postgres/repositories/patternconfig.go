/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
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

type PatternConfigRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type patternConfigRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewPatternConfigRepository(
	p PatternConfigRepositoryParams,
) repositories.PatternConfigRepository {
	log := p.Logger.With().
		Str("repository", "pattern_config").
		Logger()

	return &patternConfigRepository{
		db: p.DB,
		l:  &log,
	}
}

func (pcr *patternConfigRepository) GetAll(
	ctx context.Context,
) ([]*dedicatedlane.PatternConfig, error) {
	dba, err := pcr.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.In("pattern_config_repository").
			With("op", "get_all").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := pcr.l.With().Str("operation", "GetAll").Logger()

	entities := make([]*dedicatedlane.PatternConfig, 0)

	query := dba.NewSelect().Model(&entities).Relation("Organization")

	if err = query.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get pattern configs")
		return nil, oops.In("pattern_config_repository").
			With("op", "get_all").
			Time(time.Now()).
			Wrapf(err, "get pattern configs")
	}

	log.Info().Int("patternConfigCount", len(entities)).Msg("fetched pattern configs")

	// Log details about each pattern config
	for i, config := range entities {
		log.Info().
			Int("configIndex", i).
			Str("organizationId", config.OrganizationID.String()).
			Str("organizationName", func() string {
				if config.Organization != nil {
					return config.Organization.Name
				}
				return "unknown"
			}()).
			Int64("minFrequency", config.MinFrequency).
			Str("minConfidenceScore", config.MinConfidenceScore.String()).
			Bool("requireExactMatch", config.RequireExactMatch).
			Bool("weightRecentShipments", config.WeightRecentShipments).
			Int64("suggestionTTLDays", config.SuggestionTTLDays).
			Msg("pattern config details")
	}

	return entities, nil
}

func (pcr *patternConfigRepository) GetByOrgID(
	ctx context.Context,
	req repositories.GetPatternConfigRequest,
) (*dedicatedlane.PatternConfig, error) {
	dba, err := pcr.db.DB(ctx)
	if err != nil {
		return nil, oops.In("pattern_config_repository").
			With("op", "get_by_org_id").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := pcr.l.With().Str("operation", "GetByOrgID").
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	entity := new(dedicatedlane.PatternConfig)

	query := dba.NewSelect().
		Model(entity).
		Relation("Organization").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("pc.organization_id = ?", req.OrgID).
				Where("pc.business_unit_id = ?", req.BuID)
		})

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("pattern config not found within your organization")
			return nil, errors.NewNotFoundError("Pattern config not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get pattern config")
		return nil, oops.In("pattern_config_repository").
			With("op", "get_by_org_id").
			Time(time.Now()).
			Wrapf(err, "get pattern config")
	}

	return entity, nil
}

func (pcr *patternConfigRepository) Update(
	ctx context.Context,
	pc *dedicatedlane.PatternConfig,
) (*dedicatedlane.PatternConfig, error) {
	dba, err := pcr.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.In("pattern_config_repository").
			With("op", "update").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := pcr.l.With().Str("operation", "Update").Str("orgID", pc.OrganizationID.String()).Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := pc.Version

		pc.Version++

		results, rErr := tx.NewUpdate().
			Model(pc).
			WherePK().
			Where("pc.version = ?", ov).
			// OmitZero().
			Returning("*").
			Exec(c)
		if rErr != nil {
			if eris.Is(rErr, sql.ErrNoRows) {
				log.Error().Msg("pattern config not found within your organization")
				return errors.NewNotFoundError(
					"Pattern config not found within your organization",
				)
			}

			log.Error().Err(rErr).Msg("failed to update pattern config")
			return oops.In("pattern_config_repository").
				With("op", "update").
				Time(time.Now()).Wrapf(rErr, "update pattern config")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return oops.In("pattern_config_repository").
				With("op", "update").
				Time(time.Now()).Wrapf(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Pattern Config (%s) has either been updated or deleted since the last request.",
					pc.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update pattern config")
		return nil, oops.In("pattern_config_repository").
			With("op", "update").
			Time(time.Now()).
			Wrapf(err, "update pattern config")
	}

	return pc, nil
}
