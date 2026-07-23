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

type payAdvanceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayAdvance(p Params) repositories.PayAdvanceRepository {
	return &payAdvanceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pay-advance-repository"),
	}
}

func (r *payAdvanceRepository) List(
	ctx context.Context,
	req *repositories.ListPayAdvancesRequest,
) (*pagination.ListResult[*driverpay.PayAdvance], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driverpay.PayAdvance, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("padv.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("padv.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("padv.issued_date DESC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where(
			"(padv.reference ILIKE ? OR padv.notes ILIKE ?)",
			"%"+req.Filter.Query+"%",
			"%"+req.Filter.Query+"%",
		)
	}
	if !req.WorkerID.IsNil() {
		query = query.Where("padv.worker_id = ?", req.WorkerID)
	}
	if req.Status != "" {
		query = query.Where("padv.status = ?", req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pay advances: %w", err)
	}

	return &pagination.ListResult[*driverpay.PayAdvance]{Items: items, Total: total}, nil
}

func (r *payAdvanceRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListPayAdvanceConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayAdvance], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driverpay.PayAdvance)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"padv",
				req.Filter,
				(*driverpay.PayAdvance)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count pay advances", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*driverpay.PayAdvance]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*driverpay.PayAdvance) *bun.SelectQuery {
			return dba.NewSelect().
				Model(entities).
				ColumnExpr(buncolgen.PayAdvanceTable.All()).
				Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				"padv",
				req.Filter,
				req.Cursor,
				(*driverpay.PayAdvance)(nil),
			)
		},
	})
	if err != nil {
		log.Error("failed to scan pay advances", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *payAdvanceRepository) GetByID(
	ctx context.Context,
	req repositories.GetPayAdvanceByIDRequest,
) (*driverpay.PayAdvance, error) {
	entity := new(driverpay.PayAdvance)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("padv.id = ?", req.ID).
		Where("padv.organization_id = ?", req.TenantInfo.OrgID).
		Where("padv.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PayAdvance")
	}
	return entity, nil
}

func (r *payAdvanceRepository) ListOutstandingForWorker(
	ctx context.Context,
	req repositories.ListOutstandingAdvancesForWorkerRequest,
) ([]*driverpay.PayAdvance, error) {
	items := make([]*driverpay.PayAdvance, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("padv.organization_id = ?", req.TenantInfo.OrgID).
		Where("padv.business_unit_id = ?", req.TenantInfo.BuID).
		Where("padv.worker_id = ?", req.WorkerID).
		Where("padv.status IN (?)", bun.List([]driverpay.AdvanceStatus{
			driverpay.AdvanceStatusOutstanding,
			driverpay.AdvanceStatusPartiallyRecovered,
		})).
		Order("padv.issued_date ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list outstanding advances for worker: %w", err)
	}
	return items, nil
}

func (r *payAdvanceRepository) Create(
	ctx context.Context,
	entity *driverpay.PayAdvance,
) (*driverpay.PayAdvance, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("padv_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create pay advance: %w", err)
	}
	return r.GetByID(ctx, repositories.GetPayAdvanceByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *payAdvanceRepository) Update(
	ctx context.Context,
	entity *driverpay.PayAdvance,
) (*driverpay.PayAdvance, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("source = ?", entity.Source).
		Set("reference = ?", entity.Reference).
		Set("issued_date = ?", entity.IssuedDate).
		Set("amount_minor = ?", entity.AmountMinor).
		Set("recovered_minor = ?", entity.RecoveredMinor).
		Set("written_off_minor = ?", entity.WrittenOffMinor).
		Set("write_off_reason = ?", entity.WriteOffReason).
		Set("notes = ?", entity.Notes).
		Set("written_off_by_id = ?", entity.WrittenOffByID).
		Set("written_off_at = ?", entity.WrittenOffAt).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update pay advance: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "PayAdvance", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetPayAdvanceByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}
