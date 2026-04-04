package glaccountrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.GLAccountRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.glaccount-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListGLAccountsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"gla",
		req.Filter,
		(*glaccount.GLAccount)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListGLAccountsRequest,
) (*pagination.ListResult[*glaccount.GLAccount], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*glaccount.GLAccount, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Relation("AccountType").
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count gl accounts", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*glaccount.GLAccount]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetGLAccountByIDRequest,
) (*glaccount.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(glaccount.GLAccount)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Relation("AccountType").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("gla.id = ?", req.ID).
				Where("gla.organization_id = ?", req.TenantInfo.OrgID).
				Where("gla.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get gl account", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "GLAccount")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *glaccount.GLAccount,
) (*glaccount.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("accountCode", entity.AccountCode),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create gl account", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *glaccount.GLAccount,
) (*glaccount.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update gl account", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "GLAccount", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateGLAccountStatusRequest,
) ([]*glaccount.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*glaccount.GLAccount, 0, len(req.GLAccountIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("gla.organization_id = ?", req.TenantInfo.OrgID).
				Where("gla.business_unit_id = ?", req.TenantInfo.BuID).
				Where("gla.id IN (?)", bun.List(req.GLAccountIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update gl account status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "GLAccount", req.GLAccountIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetGLAccountsByIDsRequest,
) ([]*glaccount.GLAccount, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*glaccount.GLAccount, 0, len(req.GLAccountIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("gla.organization_id = ?", req.TenantInfo.OrgID).
				Where("gla.business_unit_id = ?", req.TenantInfo.BuID).
				Where("gla.id IN (?)", bun.List(req.GLAccountIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get gl accounts", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "GLAccount")
	}

	return entities, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteGLAccountRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	result, err := r.db.DB().
		NewDelete().
		Model((*glaccount.GLAccount)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("gla.id = ?", req.ID).
				Where("gla.organization_id = ?", req.TenantInfo.OrgID).
				Where("gla.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete gl account", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "GLAccount", req.ID.String())
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.GLAccountSelectOptionsRequest,
) (*pagination.ListResult[*glaccount.GLAccount], error) {
	return dbhelper.SelectOptions[*glaccount.GLAccount](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"account_code",
				"name",
			},
			OrgColumn: "gla.organization_id",
			BuColumn:  "gla.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("gla.status = ?", domaintypes.StatusActive)
			},
			EntityName: "GLAccount",
			SearchColumns: []string{
				"gla.account_code",
				"gla.name",
			},
		},
	)
}
