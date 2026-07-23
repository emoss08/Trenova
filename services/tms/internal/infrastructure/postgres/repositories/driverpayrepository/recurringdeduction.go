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
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type recurringDeductionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRecurringDeduction(p Params) repositories.RecurringDeductionRepository {
	return &recurringDeductionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.recurring-deduction-repository"),
	}
}

func (r *recurringDeductionRepository) List(
	ctx context.Context,
	req *repositories.ListRecurringDeductionsRequest,
) (*pagination.ListResult[*driverpay.RecurringDeduction], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driverpay.RecurringDeduction, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("rded.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("rded.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Relation("PayCode", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("rded.created_at DESC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where("rded.description ILIKE ?", "%"+req.Filter.Query+"%")
	}
	if !req.WorkerID.IsNil() {
		query = query.Where("rded.worker_id = ?", req.WorkerID)
	}
	if req.Status != "" {
		query = query.Where("rded.status = ?", req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list recurring deductions: %w", err)
	}

	return &pagination.ListResult[*driverpay.RecurringDeduction]{Items: items, Total: total}, nil
}

func (r *recurringDeductionRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListRecurringDeductionConnectionRequest,
) (*pagination.CursorListResult[*driverpay.RecurringDeduction], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driverpay.RecurringDeduction)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"rded",
				req.Filter,
				(*driverpay.RecurringDeduction)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count recurring deductions", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*driverpay.RecurringDeduction]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*driverpay.RecurringDeduction) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.RecurringDeductionTable.All()).
					Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
					Relation("PayCode", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					"rded",
					req.Filter,
					req.Cursor,
					(*driverpay.RecurringDeduction)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan recurring deductions", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *recurringDeductionRepository) GetByID(
	ctx context.Context,
	req repositories.GetRecurringDeductionByIDRequest,
) (*driverpay.RecurringDeduction, error) {
	entity := new(driverpay.RecurringDeduction)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("rded.id = ?", req.ID).
		Where("rded.organization_id = ?", req.TenantInfo.OrgID).
		Where("rded.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Relation("PayCode", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "RecurringDeduction")
	}
	return entity, nil
}

func (r *recurringDeductionRepository) ListActiveForWorker(
	ctx context.Context,
	req repositories.ListActiveDeductionsForWorkerRequest,
) ([]*driverpay.RecurringDeduction, error) {
	items := make([]*driverpay.RecurringDeduction, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("rded.organization_id = ?", req.TenantInfo.OrgID).
		Where("rded.business_unit_id = ?", req.TenantInfo.BuID).
		Where("rded.worker_id = ?", req.WorkerID).
		Where("rded.status = ?", driverpay.DeductionStatusActive).
		Where("rded.start_date <= ?", req.AsOf).
		Where("rded.end_date IS NULL OR rded.end_date > ?", req.AsOf).
		Order("rded.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active deductions for worker: %w", err)
	}
	return items, nil
}

func (r *recurringDeductionRepository) Create(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
) (*driverpay.RecurringDeduction, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("rded_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create recurring deduction: %w", err)
	}
	return r.GetByID(ctx, repositories.GetRecurringDeductionByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *recurringDeductionRepository) Update(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
) (*driverpay.RecurringDeduction, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("pay_code_id = ?", entity.PayCodeID).
		Set("status = ?", entity.Status).
		Set("frequency = ?", entity.Frequency).
		Set("description = ?", entity.Description).
		Set("amount_minor = ?", entity.AmountMinor).
		Set("total_cap_minor = ?", entity.TotalCapMinor).
		Set("deducted_to_date_minor = ?", entity.DeductedToDateMinor).
		Set("start_date = ?", entity.StartDate).
		Set("end_date = ?", entity.EndDate).
		Set("escrow_account_id = ?", entity.EscrowAccountID).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update recurring deduction: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "RecurringDeduction", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetRecurringDeductionByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}
