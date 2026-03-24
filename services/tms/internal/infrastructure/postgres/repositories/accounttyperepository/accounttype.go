package accounttyperepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
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

func New(p Params) repositories.AccountTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounttype-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAccountTypesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"at",
		req.Filter,
		(*accounttype.AccountType)(nil),
	)

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAccountTypesRequest,
) (*pagination.ListResult[*accounttype.AccountType], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*accounttype.AccountType, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count account types", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*accounttype.AccountType]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAccountTypeByIDRequest,
) (*accounttype.AccountType, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(accounttype.AccountType)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("at.id = ?", req.ID).
				Where("at.organization_id = ?", req.TenantInfo.OrgID).
				Where("at.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get account type", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "AccountType")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *accounttype.AccountType,
) (*accounttype.AccountType, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create account type", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *accounttype.AccountType,
) (*accounttype.AccountType, error) {
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
		log.Error("failed to update account type", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "AccountType", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateAccountTypeStatusRequest,
) ([]*accounttype.AccountType, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*accounttype.AccountType, 0, len(req.AccountTypeIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("at.organization_id = ?", req.TenantInfo.OrgID).
				Where("at.business_unit_id = ?", req.TenantInfo.BuID).
				Where("at.id IN (?)", bun.In(req.AccountTypeIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update account type status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "AccountType", req.AccountTypeIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetAccountTypesByIDsRequest,
) ([]*accounttype.AccountType, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*accounttype.AccountType, 0, len(req.AccountTypeIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("at.organization_id = ?", req.TenantInfo.OrgID).
				Where("at.business_unit_id = ?", req.TenantInfo.BuID).
				Where("at.id IN (?)", bun.In(req.AccountTypeIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get account types", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "AccountType")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.AccountTypeSelectOptionsRequest,
) (*pagination.ListResult[*accounttype.AccountType], error) {
	return dbhelper.SelectOptions[*accounttype.AccountType](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"name",
				"category",
			},
			OrgColumn: "at.organization_id",
			BuColumn:  "at.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("at.status = ?", domaintypes.StatusActive)
			},
			EntityName: "AccountType",
			SearchColumns: []string{
				"at.code",
				"at.name",
			},
		},
	)
}
