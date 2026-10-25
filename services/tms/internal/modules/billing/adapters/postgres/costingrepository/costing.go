package costingrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol/costingcontroldb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var costingControlErrors = dberror.Mapping{Entity: "CostingControl"}

var costCategoryErrors = dberror.Mapping{Entity: "CostCategory"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) costingcontrolports.CostingControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.costing-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	req *costingcontrolports.GetCostingControlRequest,
) (*costingcontrol.CostingControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
	)

	entity, err := r.selectControl(ctx)
	if err == nil {
		return entity, nil
	}

	if !dberror.IsNotFoundError(err) {
		log.Error("failed to get costing control", zap.Error(err))
		return nil, costingControlErrors.Map(err, "")
	}

	if err = r.createDefaults(ctx, req); err != nil {
		log.Error("failed to create default costing control", zap.Error(err))
		return nil, err
	}

	entity, err = r.selectControl(ctx)
	if err != nil {
		log.Error("failed to get costing control after creation", zap.Error(err))
		return nil, costingControlErrors.Map(err, "")
	}

	return entity, nil
}

func (r *repository) selectControl(ctx context.Context) (*costingcontrol.CostingControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*costingcontrol.CostingControl, error) {
		cc := costingcontroldb.CostingControls
		cats := cc.Categories()
		links := cats.GLAccounts()

		return cc.Query().
			Load(
				cc.FuelIndex(),
				cats.Order(cats.SortOrder.Asc(), cats.CreatedAt.Asc()).
					Load(links.Load(links.GLAccount())),
			).
			One(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) createDefaults(
	ctx context.Context,
	req *costingcontrolports.GetCostingControlRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cc := costingcontroldb.CostingControls
		db := r.db.Keel(ctx)
		control := &costingcontrol.CostingControl{
			BusinessUnitID:       req.TenantInfo.BuID,
			OrganizationID:       req.TenantInfo.OrgID,
			UseLiveFuelPrice:     false,
			MilesPerGallon:       costingcontrol.DefaultMilesPerGallon(),
			IncludeDeadheadMiles: true,
			GLRollingMonths:      3,
		}

		inserted, err := cc.CreateOrSkip(
			ctx,
			db,
			[]*costingcontrol.CostingControl{control},
			cc.OrganizationID,
		)
		if err != nil {
			return err
		}
		if inserted == 0 {
			return nil
		}

		categories := costingcontrol.DefaultCategories()
		for _, category := range categories {
			category.BusinessUnitID = req.TenantInfo.BuID
			category.OrganizationID = req.TenantInfo.OrgID
			category.CostingControlID = control.ID
		}

		return costingcontroldb.CostCategories.CreateMany(ctx, db, categories)
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *costingcontrol.CostingControl,
) (*costingcontrol.CostingControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*costingcontrol.CostingControl, error) {
		if err := costingcontroldb.CostingControls.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update costing control",
				zap.String("orgID", entity.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, costingControlErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateCategory(
	ctx context.Context,
	category *costingcontrol.CostCategory,
) (*costingcontrol.CostCategory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*costingcontrol.CostCategory, error) {
		if err := costingcontroldb.CostCategories.Save(ctx, r.db.Keel(ctx), category); err != nil {
			r.l.Error(
				"failed to update cost category",
				zap.String("categoryID", category.ID.String()),
				zap.Error(err),
			)
			return nil, costCategoryErrors.Map(err, category.ID.String())
		}

		return category, nil
	})
}

func (r *repository) ReplaceCategoryGLAccounts(
	ctx context.Context,
	req *costingcontrolports.ReplaceCategoryGLAccountsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ccga := costingcontroldb.CostCategoryGLAccounts
		db := r.db.Keel(ctx)
		if _, err := ccga.Where(ccga.CostCategoryID.Eq(req.CostCategoryID)).
			Delete(ctx, db); err != nil {
			r.l.Error("failed to clear cost category GL accounts", zap.Error(err))
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

		if err := ccga.CreateMany(ctx, db, links); err != nil {
			r.l.Error("failed to replace cost category GL accounts", zap.Error(err))
			return err
		}

		return nil
	})
}
