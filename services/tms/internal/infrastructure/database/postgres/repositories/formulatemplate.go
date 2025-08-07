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

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// FormulaTemplateRepositoryParams defines dependencies required for initializing the FormulaTemplateRepository.
type FormulaTemplateRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// formulaTemplateRepository implements the FormulaTemplateRepository interface
type formulaTemplateRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewFormulaTemplateRepository initializes a new instance of formulaTemplateRepository with its dependencies.
func NewFormulaTemplateRepository(
	p FormulaTemplateRepositoryParams,
) repositories.FormulaTemplateRepository {
	log := p.Logger.With().
		Str("repository", "formulatemplate").
		Logger()

	return &formulaTemplateRepository{
		db: p.DB,
		l:  &log,
	}
}

// addOptions expands the query with related entities based on FormulaTemplateOptions.
func (r *formulaTemplateRepository) addOptions(
	q *bun.SelectQuery,
	opts repositories.FormulaTemplateOptions,
) *bun.SelectQuery {
	if !opts.IncludeInactive {
		q = q.Where("ft.is_active = ?", true)
	}

	return q
}

// filterQuery applies filters and pagination to the formula template query.
func (r *formulaTemplateRepository) filterQuery(
	q *bun.SelectQuery,
	opts *repositories.ListFormulaTemplateOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "ft",
		Filter:     opts.Filter,
	})

	q = r.addOptions(q, opts.FormulaTemplateOptions)

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

// List retrieves a list of formula templates based on the provided options.
func (r *formulaTemplateRepository) List(
	ctx context.Context,
	opts *repositories.ListFormulaTemplateOptions,
) (*ports.ListResult[*formulatemplate.FormulaTemplate], error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "list").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*formulatemplate.FormulaTemplate, 0)

	q := dba.NewSelect().Model(&entities).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(q, opts)
		})

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to scan formula templates")
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "list").
			Time(time.Now()).
			Wrapf(err, "scan formula templates")
	}

	return &ports.ListResult[*formulatemplate.FormulaTemplate]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves a formula template by its ID.
func (r *formulaTemplateRepository) GetByID(
	ctx context.Context,
	opts *repositories.GetFormulaTemplateByIDOptions,
) (*formulatemplate.FormulaTemplate, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByID").
		Str("id", opts.ID.String()).
		Logger()

	entity := new(formulatemplate.FormulaTemplate)

	q := dba.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("ft.id = ?", opts.ID).
				Where("ft.organization_id = ?", opts.OrgID).
				Where("ft.business_unit_id = ?", opts.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, opts.FormulaTemplateOptions)
		})

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().
				Err(err).
				Msg("failed to get formula template")
			return nil, errors.NewNotFoundError(
				"Formula Template not found within your organization",
			)
		}

		log.Error().
			Err(err).
			Msg("failed to get formula template")
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "get").
			Time(time.Now()).
			Wrapf(err, "scan formula template")
	}

	return entity, nil
}

// GetByCategory retrieves formula templates by category.
func (r *formulaTemplateRepository) GetByCategory(
	ctx context.Context,
	category formulatemplate.Category,
	orgID pulid.ID,
	buID pulid.ID,
) ([]*formulatemplate.FormulaTemplate, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "get").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByCategory").
		Str("category", category.String()).
		Logger()

	entities := make([]*formulatemplate.FormulaTemplate, 0)

	err = dba.NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ft.category = ?", category).
				Where("ft.organization_id = ?", orgID).
				Where("ft.business_unit_id = ?", buID).
				Where("ft.is_active = ?", true)
		}).
		Order("ft.is_default DESC").
		Order("ft.name ASC").
		Scan(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to scan formula templates by category")
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "get").
			Time(time.Now()).
			Wrapf(err, "scan formula templates by category")
	}

	return entities, nil
}

// GetDefault retrieves the default formula template for a category.
func (r *formulaTemplateRepository) GetDefault(
	ctx context.Context,
	opts *repositories.GetDefaultFormulaTemplateOptions,
) (*formulatemplate.FormulaTemplate, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "get").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetDefault").
		Str("category", opts.Category.String()).
		Logger()

	entity := new(formulatemplate.FormulaTemplate)

	q := dba.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ft.category = ?", opts.Category).
				Where("ft.organization_id = ?", opts.OrgID).
				Where("ft.business_unit_id = ?", opts.BuID).
				Where("ft.is_default = ?", true)
		})

	q = r.addOptions(q, opts.FormulaTemplateOptions)

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().
				Err(err).
				Msg("failed to get default formula template")
			return nil, errors.NewNotFoundError(
				fmt.Sprintf("no default formula template found for category %s", opts.Category),
			)
		}

		log.Error().
			Err(err).
			Msg("failed to get default formula template")
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "get").
			Time(time.Now()).
			Wrapf(err, "scan default formula template")
	}

	return entity, nil
}

// Create creates a new formula template.
func (r *formulaTemplateRepository) Create(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if template.IsDefault {
			_, err = tx.NewUpdate().
				Model((*formulatemplate.FormulaTemplate)(nil)).
				Set("is_default = ?", false).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return uq.
						Where("ft.organization_id = ?", template.OrganizationID).
						Where("ft.business_unit_id = ?", template.BusinessUnitID).
						Where("ft.category = ?", template.Category).
						Where("ft.id != ?", template.ID)
				}).
				Exec(c)
			if err != nil {
				return oops.In("formulatemplate_repository").
					Tags("crud", "create").
					Time(time.Now()).
					Wrapf(err, "unset default templates")
			}
		}

		if _, err = tx.NewInsert().Model(template).Exec(ctx); err != nil {
			return oops.In("formulatemplate_repository").
				Tags("crud", "create").
				Time(time.Now()).
				Wrapf(err, "insert formula template")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetFormulaTemplateByIDOptions{
		ID:    template.ID,
		OrgID: template.OrganizationID,
		BuID:  template.BusinessUnitID,
	})
}

// Update updates an existing formula template.
func (r *formulaTemplateRepository) Update(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.In("formulatemplate_repository").
			Tags("crud", "update").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("id", template.GetID()).
		Int64("version", template.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := template.Version
		template.Version++

		// * If this is set as default, unset other defaults in the same category
		if template.IsDefault {
			_, err = tx.NewUpdate().
				Model((*formulatemplate.FormulaTemplate)(nil)).
				Set("is_default = ?", false).
				Set("version = version + 1").
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return uq.
						Where("ft.organization_id = ?", template.OrganizationID).
						Where("ft.business_unit_id = ?", template.BusinessUnitID).
						Where("ft.category = ?", template.Category).
						Where("ft.id != ?", template.ID)
				}).
				Exec(c)
			if err != nil {
				log.Error().
					Err(err).
					Msg("failed to unset default templates")
				return oops.In("formulatemplate_repository").
					Tags("crud", "update").
					Time(time.Now()).
					Wrapf(err, "unset default templates")
			}
		}

		result, rErr := tx.NewUpdate().
			Model(template).
			Where("ft.version = ?", ov).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.
					Where("ft.id = ?", template.ID).
					Where("ft.organization_id = ?", template.OrganizationID).
					Where("ft.business_unit_id = ?", template.BusinessUnitID)
			}).
			OmitZero().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("template", template).
				Msg("failed to update formula template")
			return err
		}

		rowsAffected, roErr := result.RowsAffected()
		if roErr != nil {
			log.Error().
				Interface("result", result).
				Msg("failed to get rows affected")
			log.Error().
				Err(roErr).
				Msg("failed to get rows affected")
			return err
		}

		if rowsAffected == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Formula Template (%s) has either been updated or deleted since the last request.",
					template.GetID(),
				),
			)
		}

		return nil
	})

	return r.GetByID(ctx, &repositories.GetFormulaTemplateByIDOptions{
		ID:    template.ID,
		OrgID: template.OrganizationID,
		BuID:  template.BusinessUnitID,
	})
}

// SetDefault sets a formula template as the default for its category.
func (r *formulaTemplateRepository) SetDefault(
	ctx context.Context,
	req *repositories.SetDefaultFormulaTemplateRequest,
) error {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return oops.In("formulatemplate_repository").
			Tags("crud", "set default").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "SetDefault").
		Str("templateID", req.TemplateID.String()).
		Str("category", req.Category.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// * Unset all defaults for this category
		_, err = tx.NewUpdate().
			Model((*formulatemplate.FormulaTemplate)(nil)).
			Set("is_default = ?", false).
			Set("version = version + 1").
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.
					Where("ft.organization_id = ?", req.OrgID).
					Where("ft.business_unit_id = ?", req.BuID).
					Where("ft.category = ?", req.Category)
			}).
			Exec(c)
		if err != nil {
			log.Error().
				Err(err).
				Msg("failed to unset default templates")
			return oops.In("formulatemplate_repository").
				Tags("crud", "set default").
				Time(time.Now()).
				Wrapf(err, "unset default templates")
		}

		// * Set the new default
		result, rErr := tx.NewUpdate().
			Model((*formulatemplate.FormulaTemplate)(nil)).
			Set("is_default = ?", true).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.
					Where("ft.id = ?", req.TemplateID).
					Where("ft.organization_id = ?", req.OrgID).
					Where("ft.business_unit_id = ?", req.BuID).
					Where("ft.category = ?", req.Category)
			}).
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Msg("failed to set default template")
			return oops.In("formulatemplate_repository").
				Tags("crud", "set default").
				Time(time.Now()).
				Wrapf(rErr, "set default template")
		}

		rowsAffected, roErr := result.RowsAffected()
		if err != nil {
			log.Error().
				Err(roErr).
				Msg("failed to get rows affected")
			return oops.In("formulatemplate_repository").
				Tags("crud", "set default").
				Time(time.Now()).
				Wrapf(roErr, "get rows affected")
		}

		if rowsAffected == 0 {
			log.Error().
				Msg("formula template not found or category mismatch")
			return errors.NewNotFoundError("Formula Template not found within your organization")
		}

		return nil
	})

	return err
}

// Delete deletes a formula template.
func (r *formulaTemplateRepository) Delete(
	ctx context.Context,
	id pulid.ID,
	orgID pulid.ID,
	buID pulid.ID,
) error {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return oops.In("formulatemplate_repository").
			Tags("crud", "delete").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Delete").
		Str("id", id.String()).
		Logger()

	result, err := dba.NewDelete().
		Model((*formulatemplate.FormulaTemplate)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.
				Where("ft.id = ?", id).
				Where("ft.organization_id = ?", orgID).
				Where("ft.business_unit_id = ?", buID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to delete formula template")
		return oops.In("formulatemplate_repository").
			Tags("crud", "delete").
			Time(time.Now()).
			Wrapf(err, "delete formula template")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get rows affected")
		return oops.In("formulatemplate_repository").
			Tags("crud", "delete").
			Time(time.Now()).
			Wrapf(err, "get rows affected")
	}

	if rowsAffected == 0 {
		log.Error().
			Msg("formula template not found")
		return errors.NewNotFoundError("Formula Template not found within your organization")
	}

	return nil
}
