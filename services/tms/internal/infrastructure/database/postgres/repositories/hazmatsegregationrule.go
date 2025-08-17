/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// HazmatSegregationRuleRepositoryParams defines the dependencies required for initializing the HazmatSegregationRuleRepository.
// This includes the database connection and the logger.
type HazmatSegregationRuleRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// hazmatSegregationRuleRepository implements the HazmatSegregationRuleRepository interface
// and provides methods to interact with hazmat segregation rules, including CRUD operations.
type hazmatSegregationRuleRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewHazmatSegregationRuleRepository(
	p HazmatSegregationRuleRepositoryParams,
) repositories.HazmatSegregationRuleRepository {
	log := p.Logger.With().Str("repository", "hazmatsegregationrule").Logger()

	return &hazmatSegregationRuleRepository{db: p.DB, l: &log}
}

// filterQuery applies filters and pagination to the hazmat segregation rule query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - req: ListHazmatSegregationRuleRequest containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (r *hazmatSegregationRuleRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListHazmatSegregationRuleRequest,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "hsr",
		Filter:     req.Filter,
	})

	// * If the request includes hazmat materials, join the hazmat materials
	if req.IncludeHazmatMaterials {
		q = q.Relation("HazmatAMaterial").Relation("HazmatBMaterial")
	}

	if req.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			req.Filter.Query,
			(*hazmatsegregationrule.HazmatSegregationRule)(nil),
		)
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

// List retrieves hazmat segregation rules based on filtering and pagination options.
// It returns a list of hazmat segregation rules along with the total count.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: ListHazmatSegregationRuleRequest for filtering and pagination.
//
// Returns:
//   - *ports.ListResult[*hazmatsegregationrule.HazmatSegregationRule]: List of hazmat segregation rules and total count.
//   - error: If any database operation fails.
func (r *hazmatSegregationRuleRepository) List(
	ctx context.Context,
	req *repositories.ListHazmatSegregationRuleRequest,
) (*ports.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("userID", req.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*hazmatsegregationrule.HazmatSegregationRule, 0, req.Filter.Limit)

	q := dba.NewSelect().Model(&entities)
	q = r.filterQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan hazmat segregation rules")
		return nil, eris.Wrap(err, "scan hazmat segregation rules")
	}

	return &ports.ListResult[*hazmatsegregationrule.HazmatSegregationRule]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves a hazmat segregation rule by its unique ID.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: GetHazmatSegregationRuleByIDRequest containing ID and tenant details.
//
// Returns:
//   - *hazmatsegregationrule.HazmatSegregationRule: The retrieved hazmat segregation rule.
//   - error: If the hazmat segregation rule is not found or query fails.
func (r *hazmatSegregationRuleRepository) GetByID(
	ctx context.Context,
	req *repositories.GetHazmatSegregationRuleByIDRequest,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByID").
		Str("hazmatSegregationRuleID", req.ID.String()).
		Logger()

	entity := new(hazmatsegregationrule.HazmatSegregationRule)

	q := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("hsr.id = ?", req.ID).
				Where("hsr.organization_id = ?", req.OrgID).
				Where("hsr.business_unit_id = ?", req.BuID)
		})

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError(
				"Hazmat Segregation Rule not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get hazmat segregation rule")
		return nil, eris.Wrap(err, "get hazmat segregation rule")
	}

	return entity, nil
}

// Create inserts a new hazmat segregation rule into the database.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - hsr: The hazmat segregation rule to create.
//
// Returns:
//   - *hazmatsegregationrule.HazmatSegregationRule: The created hazmat segregation rule.
//   - error: If any database operation fails.
func (r *hazmatSegregationRuleRepository) Create(
	ctx context.Context,
	hsr *hazmatsegregationrule.HazmatSegregationRule,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Create").
		Str("orgID", hsr.OrganizationID.String()).
		Str("buID", hsr.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(hsr).Returning("*").Exec(c); iErr != nil {
			log.Error().Err(iErr).Msg("failed to insert hazmat segregation rule")
			return iErr
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create hazmat segregation rule")
		return nil, eris.Wrap(err, "create hazmat segregation rule")
	}

	return hsr, nil
}

// Update updates an existing hazmat segregation rule in the database.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - hsr: The hazmat segregation rule to update.
//
// Returns:
//   - *hazmatsegregationrule.HazmatSegregationRule: The updated hazmat segregation rule.
//   - error: If any database operation fails.
func (r *hazmatSegregationRuleRepository) Update(
	ctx context.Context,
	hsr *hazmatsegregationrule.HazmatSegregationRule,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("hazmatSegregationRuleID", hsr.ID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := hsr.Version

		hsr.Version++

		results, rErr := tx.NewUpdate().
			Model(hsr).
			WherePK().
			OmitZero().
			Where("hsr.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update hazmat segregation rule")
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
					"Version mismatch. The Hazmat Segregation Rule (%s) has either been updated or deleted since the last request.",
					hsr.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update hazmat segregation rule")
		return nil, eris.Wrap(err, "update hazmat segregation rule")
	}

	return hsr, nil
}
