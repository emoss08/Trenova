package formulatemplaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRepository(p Params) repositories.FormulaTemplateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formulatemplate-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.FormulaTemplateOptions,
) *bun.SelectQuery {
	if !opts.IncludeInactive {
		q = q.Where("ft.is_active = ?", true)
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	opts *repositories.ListFormulaTemplateRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"ft",
		opts.Filter,
		(*formulatemplate.FormulaTemplate)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, opts.FormulaTemplateOptions)
	})

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListFormulaTemplateRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*formulatemplate.FormulaTemplate, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan formula templates", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*formulatemplate.FormulaTemplate]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(formulatemplate.FormulaTemplate)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("cus.id = ?", req.ID).
				Where("cus.organization_id = ?", req.OrgID).
				Where("cus.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FormulaTemplateOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Formula Template")
	}

	return entity, nil
}

func (r *repository) GetByCategory(
	ctx context.Context,
	category formulatemplate.Category,
	orgID pulid.ID,
	buID pulid.ID,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetByCategory"),
		zap.String("category", category.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*formulatemplate.FormulaTemplate, 0)

	err = db.NewSelect().
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
		log.Error("failed to scan formula templates by category", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetDefault(
	ctx context.Context,
	opts *repositories.GetDefaultFormulaTemplateRequest,
) (*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetDefault"),
		zap.String("category", opts.Category.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(formulatemplate.FormulaTemplate)

	q := db.NewSelect().
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
		log.Error("failed to get default formula template", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Formula Template")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert formula template", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Update updates an existing formula template.
func (r *repository) Update(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := entity.Version
		entity.Version++

		if entity.IsDefault {
			_, err = tx.NewUpdate().
				Model((*formulatemplate.FormulaTemplate)(nil)).
				Set("is_default = ?", false).
				Set("version = version + 1").
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return uq.
						Where("ft.organization_id = ?", entity.OrganizationID).
						Where("ft.business_unit_id = ?", entity.BusinessUnitID).
						Where("ft.category = ?", entity.Category).
						Where("ft.id != ?", entity.ID)
				}).
				Exec(c)
			if err != nil {
				log.Error("failed to unset default templates", zap.Error(err))
				return dberror.HandleNotFoundError(err, "Formula Template")
			}
		}

		result, rErr := tx.NewUpdate().
			Model(entity).
			Where("ft.version = ?", ov).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.
					Where("ft.id = ?", entity.ID).
					Where("ft.organization_id = ?", entity.OrganizationID).
					Where("ft.business_unit_id = ?", entity.BusinessUnitID)
			}).
			OmitZero().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to update formula template", zap.Error(rErr))
			return err
		}

		roErr := dberror.CheckRowsAffected(result, "Formula Template", entity.ID.String())
		if roErr != nil {
			return roErr
		}

		return nil
	})
	if err != nil {
		log.Error("failed to update formula template", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) SetDefault(
	ctx context.Context,
	req *repositories.SetDefaultFormulaTemplateRequest,
) error {
	log := r.l.With(
		zap.String("operation", "SetDefault"),
		zap.String("templateID", req.TemplateID.String()),
		zap.String("category", req.Category.String()),
	)

	dba, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
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
			log.Error("failed to unset default templates", zap.Error(err))
			return dberror.HandleNotFoundError(err, "Formula Template")
		}

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
			log.Error("failed to set default template", zap.Error(rErr))
			return dberror.HandleNotFoundError(rErr, "Formula Template")
		}

		roErr := dberror.CheckRowsAffected(result, "Formula Template", req.TemplateID.String())
		if roErr != nil {
			return roErr
		}

		return nil
	})

	return err
}
