package driversettlementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type settlementRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSettlement(p Params) repositories.DriverSettlementRepository {
	return &settlementRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.driver-settlement-repository"),
	}
}

func (r *settlementRepository) List(
	ctx context.Context,
	req *repositories.ListDriverSettlementsRequest,
) (*pagination.ListResult[*driversettlement.Settlement], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driversettlement.Settlement, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dstl.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("dstl.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("dstl.period_end DESC").
		Order("dstl.created_at DESC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where(
			"(dstl.settlement_number ILIKE ? OR dstl.pay_profile_name ILIKE ?)",
			"%"+req.Filter.Query+"%",
			"%"+req.Filter.Query+"%",
		)
	}
	if !req.WorkerID.IsNil() {
		query = query.Where("dstl.worker_id = ?", req.WorkerID)
	}
	if !req.BatchID.IsNil() {
		query = query.Where("dstl.batch_id = ?", req.BatchID)
	}
	if req.Status != "" {
		query = query.Where("dstl.status = ?", req.Status)
	}
	if len(req.Statuses) > 0 {
		query = query.Where("dstl.status IN (?)", bun.List(req.Statuses))
	}
	if req.HasExceptions != nil {
		query = query.Where("dstl.has_exceptions = ?", *req.HasExceptions)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list driver settlements: %w", err)
	}

	return &pagination.ListResult[*driversettlement.Settlement]{Items: items, Total: total}, nil
}

func (r *settlementRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListDriverSettlementConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.Settlement], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driversettlement.Settlement)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"dstl",
				req.Filter,
				(*driversettlement.Settlement)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count driver settlements", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*driversettlement.Settlement]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*driversettlement.Settlement) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.SettlementTable.All()).
					Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					"dstl",
					req.Filter,
					req.Cursor,
					(*driversettlement.Settlement)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan driver settlements", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *settlementRepository) GetByID(
	ctx context.Context,
	req repositories.GetDriverSettlementByIDRequest,
) (*driversettlement.Settlement, error) {
	entity := new(driversettlement.Settlement)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dstl.id = ?", req.ID).
		Where("dstl.organization_id = ?", req.TenantInfo.OrgID).
		Where("dstl.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
	if req.IncludeLines {
		query = query.Relation("Lines", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("dstll.line_number ASC")
		})
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "DriverSettlement")
	}
	return entity, nil
}

func (r *settlementRepository) GetLatestForWorker(
	ctx context.Context,
	req repositories.GetLatestSettlementForWorkerRequest,
) (*driversettlement.Settlement, error) {
	entity := new(driversettlement.Settlement)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dstl.organization_id = ?", req.TenantInfo.OrgID).
		Where("dstl.business_unit_id = ?", req.TenantInfo.BuID).
		Where("dstl.worker_id = ?", req.WorkerID).
		Where("dstl.status != ?", driversettlement.StatusVoided).
		Order("dstl.period_end DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DriverSettlement")
	}
	return entity, nil
}

func (r *settlementRepository) GetOpenDraftForWorker(
	ctx context.Context,
	req repositories.GetOpenDraftForWorkerRequest,
) (*driversettlement.Settlement, error) {
	entity := new(driversettlement.Settlement)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dstl.organization_id = ?", req.TenantInfo.OrgID).
		Where("dstl.business_unit_id = ?", req.TenantInfo.BuID).
		Where("dstl.worker_id = ?", req.WorkerID).
		Where("dstl.status = ?", driversettlement.StatusDraft).
		Order("dstl.period_end DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DriverSettlement")
	}
	return entity, nil
}

func (r *settlementRepository) GetWorkspaceCounts(
	ctx context.Context,
	req *repositories.GetWorkspaceCountsRequest,
) (*repositories.SettlementWorkspaceCounts, error) {
	counts := new(repositories.SettlementWorkspaceCounts)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driversettlement.Settlement)(nil)).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS draft_count",
			driversettlement.StatusDraft).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS pending_approval_count",
			driversettlement.StatusPendingApproval).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS approved_count",
			driversettlement.StatusApproved).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS posted_count",
			driversettlement.StatusPosted).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS paid_count",
			driversettlement.StatusPaid).
		ColumnExpr("COUNT(*) FILTER (WHERE has_exceptions = true AND status NOT IN (?, ?)) "+
			"AS exception_count",
			driversettlement.StatusVoided, driversettlement.StatusPaid).
		ColumnExpr("COALESCE(SUM(net_pay_minor) FILTER (WHERE status != ?), 0) "+
			"AS total_net_minor", driversettlement.StatusVoided).
		ColumnExpr("COALESCE(SUM(gross_earnings_minor) FILTER (WHERE status != ?), 0) "+
			"AS total_gross_minor", driversettlement.StatusVoided).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Where("period_start = ?", req.PeriodStart).
		Where("period_end = ?", req.PeriodEnd).
		Scan(ctx, counts)
	if err != nil {
		return nil, fmt.Errorf("get settlement workspace counts: %w", err)
	}
	return counts, nil
}

func (r *settlementRepository) ListTrailingNetPay(
	ctx context.Context,
	req *repositories.ListTrailingNetPayRequest,
) ([]int64, error) {
	nets := make([]int64, 0, req.Limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driversettlement.Settlement)(nil)).
		Column("net_pay_minor").
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Where("worker_id = ?", req.WorkerID).
		Where("status != ?", driversettlement.StatusVoided).
		Where("period_end < ?", req.BeforeDate).
		Order("period_end DESC").
		Limit(req.Limit).
		Scan(ctx, &nets)
	if err != nil {
		return nil, fmt.Errorf("list trailing net pay: %w", err)
	}
	return nets, nil
}

func (r *settlementRepository) ExistsForWorkerPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	periodStart, periodEnd int64,
) (bool, error) {
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driversettlement.Settlement)(nil)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("worker_id = ?", workerID).
		Where("period_start = ?", periodStart).
		Where("period_end = ?", periodEnd).
		Where("status != ?", driversettlement.StatusVoided).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check settlement exists for period: %w", err)
	}
	return exists, nil
}

func (r *settlementRepository) Create(
	ctx context.Context,
	entity *driversettlement.Settlement,
) (*driversettlement.Settlement, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("dstl_")
	}
	assignLineFields(entity)
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create driver settlement: %w", err)
	}
	if len(entity.Lines) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Lines).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("create driver settlement lines: %w", err)
		}
	}
	return r.GetByID(ctx, repositories.GetDriverSettlementByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeLines: true,
	})
}

func (r *settlementRepository) Update(
	ctx context.Context,
	entity *driversettlement.Settlement,
) (*driversettlement.Settlement, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("batch_id = ?", entity.BatchID).
		Set("pay_profile_id = ?", entity.PayProfileID).
		Set("pay_profile_name = ?", entity.PayProfileName).
		Set("classification = ?", entity.Classification).
		Set("pay_date = ?", entity.PayDate).
		Set("gross_earnings_minor = ?", entity.GrossEarningsMinor).
		Set("reimbursements_minor = ?", entity.ReimbursementsMinor).
		Set("deductions_minor = ?", entity.DeductionsMinor).
		Set("carry_forward_in_minor = ?", entity.CarryForwardInMinor).
		Set("carry_forward_out_minor = ?", entity.CarryForwardOutMinor).
		Set("net_pay_minor = ?", entity.NetPayMinor).
		Set("total_miles = ?", entity.TotalMiles).
		Set("shipment_count = ?", entity.ShipmentCount).
		Set("has_exceptions = ?", entity.HasExceptions).
		Set("exceptions = ?", entity.Exceptions).
		Set("notes = ?", entity.Notes).
		Set("submitted_by_id = ?", entity.SubmittedByID).
		Set("submitted_at = ?", entity.SubmittedAt).
		Set("approved_by_id = ?", entity.ApprovedByID).
		Set("approved_at = ?", entity.ApprovedAt).
		Set("posted_by_id = ?", entity.PostedByID).
		Set("posted_at = ?", entity.PostedAt).
		Set("posted_journal_batch_id = ?", entity.PostedJournalBatchID).
		Set("paid_at = ?", entity.PaidAt).
		Set("paid_by_id = ?", entity.PaidByID).
		Set("payment_method = ?", entity.PaymentMethod).
		Set("payment_reference = ?", entity.PaymentReference).
		Set("voided_by_id = ?", entity.VoidedByID).
		Set("voided_at = ?", entity.VoidedAt).
		Set("void_reason = ?", entity.VoidReason).
		Set("void_journal_batch_id = ?", entity.VoidJournalBatchID).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update driver settlement: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "DriverSettlement", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetDriverSettlementByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeLines: true,
	})
}

func (r *settlementRepository) ReplaceLines(
	ctx context.Context,
	entity *driversettlement.Settlement,
) error {
	assignLineFields(entity)
	if _, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*driversettlement.SettlementLine)(nil)).
		Where("settlement_id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete driver settlement lines: %w", err)
	}
	if len(entity.Lines) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Lines).
			Exec(ctx); err != nil {
			return fmt.Errorf("insert driver settlement lines: %w", err)
		}
	}
	return nil
}

func assignLineFields(entity *driversettlement.Settlement) {
	for idx, line := range entity.Lines {
		if line == nil {
			continue
		}
		line.OrganizationID = entity.OrganizationID
		line.BusinessUnitID = entity.BusinessUnitID
		line.SettlementID = entity.ID
		line.LineNumber = idx + 1
	}
}
