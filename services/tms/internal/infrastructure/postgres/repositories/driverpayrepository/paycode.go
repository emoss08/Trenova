package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type payCodeRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayCode(p Params) repositories.PayCodeRepository {
	return &payCodeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pay-code-repository"),
	}
}

func (r *payCodeRepository) List(
	ctx context.Context,
	req *repositories.ListPayCodesRequest,
) (*pagination.ListResult[*driverpay.PayCode], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driverpay.PayCode, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("payc.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("payc.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Relation("GLAccount", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("payc.direction ASC", "payc.code ASC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		term := "%" + req.Filter.Query + "%"
		query = query.Where("(payc.code ILIKE ? OR payc.name ILIKE ?)", term, term)
	}
	if req.Direction != "" {
		query = query.Where("payc.direction = ?", req.Direction)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pay codes: %w", err)
	}

	return &pagination.ListResult[*driverpay.PayCode]{Items: items, Total: total}, nil
}

func (r *payCodeRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListPayCodeConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayCode], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driverpay.PayCode)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"payc",
				req.Filter,
				(*driverpay.PayCode)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count pay codes", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*driverpay.PayCode]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*driverpay.PayCode) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.PayCodeTable.All()).
					Relation("GLAccount", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					"payc",
					req.Filter,
					req.Cursor,
					(*driverpay.PayCode)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan pay codes", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *payCodeRepository) ListActive(
	ctx context.Context,
	req repositories.ListActivePayCodesRequest,
) ([]*driverpay.PayCode, error) {
	items := make([]*driverpay.PayCode, 0)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("payc.organization_id = ?", req.TenantInfo.OrgID).
		Where("payc.business_unit_id = ?", req.TenantInfo.BuID).
		Where("payc.status = ?", domaintypes.StatusActive).
		Order("payc.code ASC")
	if req.Direction != "" {
		query = query.Where("payc.direction = ?", req.Direction)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list active pay codes: %w", err)
	}
	return items, nil
}

func (r *payCodeRepository) GetByID(
	ctx context.Context,
	req repositories.GetPayCodeByIDRequest,
) (*driverpay.PayCode, error) {
	entity := new(driverpay.PayCode)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("payc.id = ?", req.ID).
		Where("payc.organization_id = ?", req.TenantInfo.OrgID).
		Where("payc.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("GLAccount", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PayCode")
	}
	return entity, nil
}

func (r *payCodeRepository) GetByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*driverpay.PayCode, error) {
	if len(ids) == 0 {
		return []*driverpay.PayCode{}, nil
	}
	items := make([]*driverpay.PayCode, 0, len(ids))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("payc.organization_id = ?", tenantInfo.OrgID).
		Where("payc.business_unit_id = ?", tenantInfo.BuID).
		Where("payc.id IN (?)", bun.List(ids)).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pay codes by ids: %w", err)
	}
	return items, nil
}

func (r *payCodeRepository) Create(
	ctx context.Context,
	entity *driverpay.PayCode,
) (*driverpay.PayCode, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("payc_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create pay code: %w", err)
	}
	return r.GetByID(ctx, repositories.GetPayCodeByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *payCodeRepository) Update(
	ctx context.Context,
	entity *driverpay.PayCode,
) (*driverpay.PayCode, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("code = ?", entity.Code).
		Set("name = ?", entity.Name).
		Set("description = ?", entity.Description).
		Set("taxable = ?", entity.Taxable).
		Set("counts_toward_guarantee = ?", entity.CountsTowardGuarantee).
		Set("gl_account_id = ?", entity.GLAccountID).
		Set("default_amount_minor = ?", entity.DefaultAmountMinor).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update pay code: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "PayCode", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetPayCodeByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *payCodeRepository) EnsureSystemDefaults(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	defs := driverpay.SystemPayCodes()
	rows := make([]*driverpay.PayCode, 0, len(defs))
	for _, def := range defs {
		rows = append(rows, &driverpay.PayCode{
			ID:                    pulid.MustNew("payc_"),
			BusinessUnitID:        tenantInfo.BuID,
			OrganizationID:        tenantInfo.OrgID,
			Status:                domaintypes.StatusActive,
			Direction:             def.Direction,
			Code:                  def.Code,
			Name:                  def.Name,
			Taxable:               def.Taxable,
			CountsTowardGuarantee: true,
			IsSystem:              true,
		})
	}

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&rows).
		On("CONFLICT (organization_id, business_unit_id, direction, code) DO NOTHING").
		Exec(ctx); err != nil {
		return fmt.Errorf("ensure system pay codes: %w", err)
	}
	return nil
}
