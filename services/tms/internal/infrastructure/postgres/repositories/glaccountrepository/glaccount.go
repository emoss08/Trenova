package glaccountrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func NewRepository(p Params) repositories.GLAccountRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.glaccount-repository"),
	}
}

func (r *repository) GetOption(
	ctx context.Context,
	req repositories.GetGLAccountByIDRequest,
) (*accounting.GLAccount, error) {
	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	entity := new(accounting.GLAccount)
	if err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.id = ?", req.GLAccountID).
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req repositories.GLAccountSelectOptionsRequest,
) ([]*repositories.GLAccountSelectOptionResponse, error) {
	log := r.l.With(
		zap.String("operation", "SelectOptions"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	entities := make([]*repositories.GLAccountSelectOptionResponse, 0)
	q := db.NewSelect().Model((*accounting.GLAccount)(nil)).
		Column("id", "account_code", "name").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		})

	if req.Query != "" {
		q = q.Where(
			"gla.account_code ILIKE ? OR gla.name ILIKE ?",
			"%"+req.Query+"%",
			"%"+req.Query+"%",
		)
	}

	if err = q.Scan(ctx, &entities); err != nil {
		log.Error("failed to scan gl accounts", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts *repositories.GLAccountFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeAccountType {
		q = q.Relation("AccountType")
	}

	if opts.IncludeParent {
		q = q.Relation("Parent")
	}

	if opts.IncludeChildren {
		q = q.Relation("Children")
	}

	if opts.Status != "" {
		q = q.Where("gla.status = ?", opts.Status)
	}

	if opts.AccountTypeID != "" {
		q = q.Where("gla.account_type_id = ?", opts.AccountTypeID)
	}

	if opts.ParentID != "" {
		q = q.Where("gla.parent_id = ?", opts.ParentID)
	}

	if opts.IsActive != nil {
		q = q.Where("gla.is_active = ?", *opts.IsActive)
	}

	if opts.IsSystem != nil {
		q = q.Where("gla.is_system = ?", *opts.IsSystem)
	}

	if opts.AllowManualJE != nil {
		q = q.Where("gla.allow_manual_je = ?", *opts.AllowManualJE)
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListGLAccountRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"gla",
		req.Filter,
		(*accounting.GLAccount)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.FilterOptions)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListGLAccountRequest,
) (*pagination.ListResult[*accounting.GLAccount], error) {
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

	entities := make([]*accounting.GLAccount, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan gl accounts", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*accounting.GLAccount]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetGLAccountByIDRequest,
) (*accounting.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.GLAccountID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.GLAccount)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.id = ?", req.GLAccountID).
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "GLAccount")
	}

	return entity, nil
}

func (r *repository) GetByCode(
	ctx context.Context,
	req *repositories.GetGLAccountByCodeRequest,
) (*accounting.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "GetByCode"),
		zap.String("accountCode", req.AccountCode),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.GLAccount)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.account_code = ?", req.AccountCode).
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "GLAccount")
	}

	return entity, nil
}

func (r *repository) GetByType(
	ctx context.Context,
	req *repositories.GetGLAccountsByTypeRequest,
) ([]*accounting.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "GetByType"),
		zap.String("accountTypeId", req.AccountTypeID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*accounting.GLAccount, 0)
	err = db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.account_type_id = ?", req.AccountTypeID).
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Order("gla.account_code ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get gl accounts by type", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByParent(
	ctx context.Context,
	req *repositories.GetGLAccountsByParentRequest,
) ([]*accounting.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "GetByParent"),
		zap.String("parentId", req.ParentID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*accounting.GLAccount, 0)
	err = db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.parent_id = ?", req.ParentID).
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Order("gla.account_code ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get gl accounts by parent", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetHierarchy(
	ctx context.Context,
	req *repositories.GetGLAccountHierarchyRequest,
) ([]*accounting.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "GetHierarchy"),
		zap.String("orgId", req.OrgID.String()),
		zap.String("buId", req.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*accounting.GLAccount, 0)
	err = db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.parent_id IS NULL").
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Relation("Children", func(sq *bun.SelectQuery) *bun.SelectQuery {
			// Recursively load children
			return sq.Relation("Children")
		}).
		Order("gla.account_code ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get gl account hierarchy", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *accounting.GLAccount,
) (*accounting.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("accountCode", entity.AccountCode),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	_, err = db.NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		log.Error("failed to create gl account", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) BulkCreate(
	ctx context.Context,
	req *repositories.BulkCreateGLAccountsRequest,
) error {
	log := r.l.With(
		zap.String("operation", "BulkCreate"),
		zap.Int("count", len(req.Accounts)),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewInsert().Model(&req.Accounts).Exec(ctx)
	if err != nil {
		log.Error("failed to bulk create gl accounts", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *accounting.GLAccount,
) (*accounting.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	_, err = db.NewUpdate().
		Model(entity).
		WherePK().
		Exec(ctx)
	if err != nil {
		log.Error("failed to update gl account", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateBalance(
	ctx context.Context,
	req *repositories.UpdateGLAccountBalanceRequest,
) (*accounting.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "UpdateBalance"),
		zap.String("glAccountId", req.GLAccountID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.GLAccount)
	_, err = db.NewUpdate().
		Model(entity).
		Set("debit_balance = debit_balance + ?", req.DebitAmount).
		Set("credit_balance = credit_balance + ?", req.CreditAmount).
		Set("current_balance = ?", req.CurrentBalance).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("gla.id = ?", req.GLAccountID).
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update gl account balance", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteGLAccountRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("entityID", req.GLAccountID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewDelete().
		Model((*accounting.GLAccount)(nil)).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.
				Where("gla.id = ?", req.GLAccountID).
				Where("gla.organization_id = ?", req.OrgID).
				Where("gla.business_unit_id = ?", req.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete gl account", zap.Error(err))
		return err
	}

	return nil
}
