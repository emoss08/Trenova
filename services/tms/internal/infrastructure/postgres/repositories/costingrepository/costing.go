package costingrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) repositories.CostingControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.costing-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	req *repositories.GetCostingControlRequest,
) (*costingcontrol.CostingControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
	)

	entity, err := r.selectControl(ctx, req)
	if err == nil {
		return entity, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		log.Error("failed to get costing control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "CostingControl")
	}

	if err = r.createDefaults(ctx, req); err != nil {
		log.Error("failed to create default costing control", zap.Error(err))
		return nil, err
	}

	entity, err = r.selectControl(ctx, req)
	if err != nil {
		log.Error("failed to get costing control after creation", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "CostingControl")
	}

	return entity, nil
}

func (r *repository) selectControl(
	ctx context.Context,
	req *repositories.GetCostingControlRequest,
) (*costingcontrol.CostingControl, error) {
	entity := new(costingcontrol.CostingControl)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		Relation("FuelIndex").
		Relation("Categories", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("ccat.sort_order ASC", "ccat.created_at ASC")
		}).
		Relation("Categories.GLAccounts").
		Relation("Categories.GLAccounts.GLAccount").
		Where("cstc.organization_id = ?", req.TenantInfo.OrgID).
		Where("cstc.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) createDefaults(
	ctx context.Context,
	req *repositories.GetCostingControlRequest,
) error {
	return r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		control := &costingcontrol.CostingControl{
			BusinessUnitID:       req.TenantInfo.BuID,
			OrganizationID:       req.TenantInfo.OrgID,
			UseLiveFuelPrice:     false,
			MilesPerGallon:       costingcontrol.DefaultMilesPerGallon(),
			IncludeDeadheadMiles: true,
			GLRollingMonths:      3,
		}

		result, err := tx.NewInsert().
			Model(control).
			On("CONFLICT (organization_id) DO NOTHING").
			Exec(txCtx)
		if err != nil {
			return err
		}

		if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows == 0 {
			return nil
		}

		categories := costingcontrol.DefaultCategories()
		for _, category := range categories {
			category.BusinessUnitID = req.TenantInfo.BuID
			category.OrganizationID = req.TenantInfo.OrgID
			category.CostingControlID = control.ID
		}

		_, err = tx.NewInsert().Model(&categories).Exec(txCtx)
		return err
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *costingcontrol.CostingControl,
) (*costingcontrol.CostingControl, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	ov := entity.Version
	entity.Version++

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update costing control", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "CostingControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateCategory(
	ctx context.Context,
	category *costingcontrol.CostCategory,
) (*costingcontrol.CostCategory, error) {
	log := r.l.With(
		zap.String("operation", "UpdateCategory"),
		zap.String("categoryID", category.ID.String()),
	)

	ov := category.Version
	category.Version++

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(category).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update cost category", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "CostCategory", category.ID.String()); err != nil {
		return nil, err
	}

	return category, nil
}

func (r *repository) ReplaceCategoryGLAccounts(
	ctx context.Context,
	req *repositories.ReplaceCategoryGLAccountsRequest,
) error {
	log := r.l.With(
		zap.String("operation", "ReplaceCategoryGLAccounts"),
		zap.String("categoryID", req.CostCategoryID.String()),
	)

	err := r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().
			Model((*costingcontrol.CostCategoryGLAccount)(nil)).
			Where("ccga.cost_category_id = ?", req.CostCategoryID).
			Where("ccga.organization_id = ?", req.TenantInfo.OrgID).
			Where("ccga.business_unit_id = ?", req.TenantInfo.BuID).
			Exec(txCtx); err != nil {
			return err
		}

		if len(req.GLAccountIDs) == 0 {
			return nil
		}

		links := make([]*costingcontrol.CostCategoryGLAccount, 0, len(req.GLAccountIDs))
		for _, glAccountID := range req.GLAccountIDs {
			links = append(links, &costingcontrol.CostCategoryGLAccount{
				BusinessUnitID: req.TenantInfo.BuID,
				OrganizationID: req.TenantInfo.OrgID,
				CostCategoryID: req.CostCategoryID,
				GLAccountID:    glAccountID,
			})
		}

		_, err := tx.NewInsert().Model(&links).Exec(txCtx)
		return err
	})
	if err != nil {
		log.Error("failed to replace cost category GL accounts", zap.Error(err))
		return err
	}

	return nil
}
