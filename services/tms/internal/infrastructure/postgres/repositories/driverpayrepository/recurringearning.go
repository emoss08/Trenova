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

type recurringEarningRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRecurringEarning(p Params) repositories.RecurringEarningRepository {
	return &recurringEarningRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.recurring-earning-repository"),
	}
}

func (r *recurringEarningRepository) List(
	ctx context.Context,
	req *repositories.ListRecurringEarningsRequest,
) (*pagination.ListResult[*driverpay.RecurringEarning], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driverpay.RecurringEarning, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("rern.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("rern.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Relation("PayCode", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("rern.created_at DESC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where("rern.description ILIKE ?", "%"+req.Filter.Query+"%")
	}
	if !req.WorkerID.IsNil() {
		query = query.Where("rern.worker_id = ?", req.WorkerID)
	}
	if req.Status != "" {
		query = query.Where("rern.status = ?", req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list recurring earnings: %w", err)
	}

	return &pagination.ListResult[*driverpay.RecurringEarning]{Items: items, Total: total}, nil
}

func (r *recurringEarningRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListRecurringEarningConnectionRequest,
) (*pagination.CursorListResult[*driverpay.RecurringEarning], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driverpay.RecurringEarning)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"rern",
				req.Filter,
				(*driverpay.RecurringEarning)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count recurring earnings", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*driverpay.RecurringEarning]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*driverpay.RecurringEarning) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.RecurringEarningTable.All()).
					Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
					Relation("PayCode", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					"rern",
					req.Filter,
					req.Cursor,
					(*driverpay.RecurringEarning)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan recurring earnings", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *recurringEarningRepository) GetByID(
	ctx context.Context,
	req repositories.GetRecurringEarningByIDRequest,
) (*driverpay.RecurringEarning, error) {
	entity := new(driverpay.RecurringEarning)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("rern.id = ?", req.ID).
		Where("rern.organization_id = ?", req.TenantInfo.OrgID).
		Where("rern.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Relation("PayCode", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "RecurringEarning")
	}
	return entity, nil
}

func (r *recurringEarningRepository) ListActiveForWorker(
	ctx context.Context,
	req repositories.ListActiveEarningsForWorkerRequest,
) ([]*driverpay.RecurringEarning, error) {
	items := make([]*driverpay.RecurringEarning, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("rern.organization_id = ?", req.TenantInfo.OrgID).
		Where("rern.business_unit_id = ?", req.TenantInfo.BuID).
		Where("rern.worker_id = ?", req.WorkerID).
		Where("rern.status = ?", driverpay.EarningStatusActive).
		Where("rern.start_date <= ?", req.AsOf).
		Where("rern.end_date IS NULL OR rern.end_date > ?", req.AsOf).
		Relation("PayCode", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("rern.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active earnings for worker: %w", err)
	}
	return items, nil
}

func (r *recurringEarningRepository) Create(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
) (*driverpay.RecurringEarning, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("rern_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create recurring earning: %w", err)
	}
	return r.GetByID(ctx, repositories.GetRecurringEarningByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *recurringEarningRepository) Update(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
) (*driverpay.RecurringEarning, error) {
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
		Set("paid_to_date_minor = ?", entity.PaidToDateMinor).
		Set("start_date = ?", entity.StartDate).
		Set("end_date = ?", entity.EndDate).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update recurring earning: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "RecurringEarning", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetRecurringEarningByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}
